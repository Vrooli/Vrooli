package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	precommitStreamTailLines       = 10
	precommitStreamHeartbeatPeriod = 2 * time.Second
)

// PrecommitStreamEvent is one logical event in the precommit SSE stream.
// Type is one of: "started", "progress", "finished", "error".
type PrecommitStreamEvent struct {
	Type      string              `json:"type"`
	ElapsedMs int64               `json:"elapsed_ms"`
	Command   string              `json:"command,omitempty"`
	Tail      []string            `json:"tail,omitempty"`
	Result    *PrecommitRunResult `json:"result,omitempty"`
	Error     string              `json:"error,omitempty"`
}

// PrecommitEventEmitter receives one event at a time. The implementation
// decides what to do with it (write SSE, append to a slice in tests, etc.).
type PrecommitEventEmitter interface {
	Emit(event PrecommitStreamEvent)
}

// StreamingCommandRunner extends CommandRunner with a streaming variant that
// surfaces stdout/stderr lines as they are produced.
type StreamingCommandRunner interface {
	CommandRunner
	RunStream(ctx context.Context, req CommandRunRequest, onLine func(stream, line string)) (CommandRunResult, error)
}

// RunStream on ShellCommandRunner runs the command and forwards each line of
// stdout/stderr to onLine while still buffering the full output for the final
// result. The buffered output is capped indirectly by the caller via capOutput.
func (ShellCommandRunner) RunStream(ctx context.Context, req CommandRunRequest, onLine func(stream, line string)) (CommandRunResult, error) {
	cmd := exec.CommandContext(ctx, "bash", "-lc", req.Command)
	cmd.Dir = req.WorkingDirectory
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return CommandRunResult{}, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return CommandRunResult{}, err
	}
	if err := cmd.Start(); err != nil {
		return CommandRunResult{}, err
	}
	var (
		stdoutBuf, stderrBuf bytes.Buffer
		wg                   sync.WaitGroup
	)
	wg.Add(2)
	go pumpLines(&wg, stdoutPipe, &stdoutBuf, "stdout", onLine)
	go pumpLines(&wg, stderrPipe, &stderrBuf, "stderr", onLine)
	wg.Wait()
	waitErr := cmd.Wait()
	return CommandRunResult{Stdout: stdoutBuf.String(), Stderr: stderrBuf.String()}, waitErr
}

func pumpLines(wg *sync.WaitGroup, r io.Reader, buf *bytes.Buffer, stream string, onLine func(string, string)) {
	defer wg.Done()
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadString('\n')
		if line != "" {
			buf.WriteString(line)
			if onLine != nil {
				onLine(stream, strings.TrimRight(line, "\n"))
			}
		}
		if err != nil {
			return
		}
	}
}

// RunStream executes the precommit configured for repoDir and emits SSE-shaped
// events as the command runs. It returns the same PrecommitRunResult as Run
// and persists last-result the same way.
func (s *PrecommitService) RunStream(ctx context.Context, repoDir string, req PrecommitRunRequest, emitter PrecommitEventEmitter) (PrecommitRunResult, error) {
	cfg, err := s.Get(ctx, repoDir)
	if err != nil {
		return PrecommitRunResult{}, err
	}
	if strings.TrimSpace(req.Command) != "" {
		cfg.Command = req.Command
	}
	if strings.TrimSpace(req.WorkingDirectory) != "" {
		cfg.WorkingDirectory = req.WorkingDirectory
	}
	if req.TimeoutSeconds > 0 {
		cfg.TimeoutSeconds = req.TimeoutSeconds
	}
	cfg = normalizePrecommitConfig(repoDir, cfg)
	if strings.TrimSpace(cfg.Command) == "" {
		return PrecommitRunResult{}, fmt.Errorf("precommit command is required")
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	started := time.Now()

	tail := newTailBuffer(precommitStreamTailLines)
	emitSafe(emitter, PrecommitStreamEvent{
		Type:      "started",
		ElapsedMs: 0,
		Command:   cfg.Command,
	})

	var (
		stopHeartbeat = make(chan struct{})
		heartbeatWG   sync.WaitGroup
	)
	heartbeatWG.Add(1)
	go func() {
		defer heartbeatWG.Done()
		ticker := time.NewTicker(precommitStreamHeartbeatPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-stopHeartbeat:
				return
			case <-ticker.C:
				emitSafe(emitter, PrecommitStreamEvent{
					Type:      "progress",
					ElapsedMs: time.Since(started).Milliseconds(),
					Tail:      tail.snapshot(),
				})
			}
		}
	}()

	runner := s.runner
	if runner == nil {
		runner = ShellCommandRunner{}
	}
	streamingRunner, ok := runner.(StreamingCommandRunner)
	var (
		runResult CommandRunResult
		runErr    error
	)
	if ok {
		runResult, runErr = streamingRunner.RunStream(runCtx, CommandRunRequest{
			Command:          cfg.Command,
			WorkingDirectory: cfg.WorkingDirectory,
		}, func(_, line string) {
			tail.add(line)
			emitSafe(emitter, PrecommitStreamEvent{
				Type:      "progress",
				ElapsedMs: time.Since(started).Milliseconds(),
				Tail:      tail.snapshot(),
			})
		})
	} else {
		runResult, runErr = runner.Run(runCtx, CommandRunRequest{
			Command:          cfg.Command,
			WorkingDirectory: cfg.WorkingDirectory,
		})
		for _, line := range strings.Split(runResult.Stdout+runResult.Stderr, "\n") {
			if line == "" {
				continue
			}
			tail.add(line)
		}
	}
	close(stopHeartbeat)
	heartbeatWG.Wait()

	result := PrecommitRunResult{
		Status:          "passed",
		Command:         cfg.Command,
		ExitCode:        0,
		Summary:         "Precommit checks passed",
		Stdout:          capOutput(runResult.Stdout),
		Stderr:          capOutput(runResult.Stderr),
		DurationMs:      time.Since(started).Milliseconds(),
		OverrideAllowed: cfg.AllowOverride,
		Timestamp:       time.Now().UTC(),
	}
	if runErr != nil {
		result.Status = "failed"
		result.Summary = "Precommit checks failed"
		result.ExitCode = 1
		if exitErr, ok := runErr.(interface{ ExitCode() int }); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		if runCtx.Err() == context.DeadlineExceeded {
			result.Status = "timeout"
			result.Summary = "Precommit checks timed out"
			result.ExitCode = 124
		}
	}
	_ = s.saveLastResult(context.Background(), repoDir, result)
	emitSafe(emitter, PrecommitStreamEvent{
		Type:      "finished",
		ElapsedMs: result.DurationMs,
		Result:    &result,
	})
	return result, nil
}

func emitSafe(emitter PrecommitEventEmitter, event PrecommitStreamEvent) {
	if emitter == nil {
		return
	}
	emitter.Emit(event)
}

type tailBuffer struct {
	mu    sync.Mutex
	lines []string
	cap   int
}

func newTailBuffer(capacity int) *tailBuffer {
	if capacity <= 0 {
		capacity = precommitStreamTailLines
	}
	return &tailBuffer{cap: capacity, lines: make([]string, 0, capacity)}
}

func (t *tailBuffer) add(line string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lines = append(t.lines, line)
	if len(t.lines) > t.cap {
		t.lines = t.lines[len(t.lines)-t.cap:]
	}
}

func (t *tailBuffer) snapshot() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, len(t.lines))
	copy(out, t.lines)
	return out
}

// sseEmitter writes PrecommitStreamEvent values to an HTTP ResponseWriter as
// Server-Sent Events. It flushes after each event so the client sees progress
// in real time.
type sseEmitter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	mu      sync.Mutex
}

func newSSEEmitter(w http.ResponseWriter) (*sseEmitter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming unsupported by response writer")
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	return &sseEmitter{w: w, flusher: flusher}, nil
}

func (e *sseEmitter) Emit(event PrecommitStreamEvent) {
	if e == nil {
		return
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	fmt.Fprintf(e.w, "event: %s\ndata: %s\n\n", event.Type, payload)
	e.flusher.Flush()
}
