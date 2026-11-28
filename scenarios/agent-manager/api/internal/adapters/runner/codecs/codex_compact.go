// This file compacts an idle Codex session through the Codex app-server.
package codecs

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/vrooli/cli-core/agenttranscript"
)

// Codex compacts a session only through `codex app-server`: `codex exec
// resume <id> "/compact"` sends the text as an ordinary user message, because
// slash commands exist only in the interactive TUI. The app-server's
// thread/compact/start runs the same compaction the TUI's /compact does and
// keeps the thread ID, so later `codex exec resume` continuations pick up the
// compacted history.
const (
	codexMethodInitialize    = "initialize"
	codexMethodInitialized   = "initialized"
	codexMethodThreadResume  = "thread/resume"
	codexMethodCompactStart  = "thread/compact/start"
	codexNotifyItemCompleted = "item/completed"
	codexNotifyTurnCompleted = "turn/completed"
	codexNotifyError         = "error"
	codexItemCompaction      = "contextCompaction"
	// codexRolloutTailBytes bounds how much of a rollout is read to find the
	// latest context size; rollouts grow to tens of megabytes.
	codexRolloutTailBytes = 4 << 20
)

// CompactSession satisfies [runner.SessionCompactor].
func (c *Codex) CompactSession(ctx context.Context, req runner.CompactSessionRequest) (*runner.CompactSessionResult, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, domain.NewValidationError("session_id", "a session ID is required to compact a Codex session")
	}
	bin := c.BinaryPath()
	if bin == "" {
		return nil, fmt.Errorf("codex binary is not available")
	}

	result := &runner.CompactSessionResult{}
	reading, err := readCodexSessionRollout(strings.TrimSpace(req.Env["CODEX_HOME"]), sessionID)
	if err != nil {
		result.Reason = fmt.Sprintf("the session's rollout could not be read, so its size and goal are unknown: %v", err)
		return result, nil
	}
	result.ContextTokens = reading.contextTokens
	if reading.goalActive {
		result.Reason = "the session holds an active goal, and resuming it elsewhere could clear it"
		return result, nil
	}
	if req.MinContextTokens > 0 && result.ContextTokens == 0 {
		result.Reason = "the session's context size is unknown: its rollout has no token count"
		return result, nil
	}
	if req.MinContextTokens > 0 && result.ContextTokens < req.MinContextTokens {
		result.Reason = fmt.Sprintf("context %d tokens is below the %d-token compaction threshold", result.ContextTokens, req.MinContextTokens)
		return result, nil
	}

	if err := runCodexAppServerCompaction(ctx, bin, sessionID, req); err != nil {
		return nil, err
	}
	result.Compacted = true
	return result, nil
}

// runCodexAppServerCompaction starts `codex app-server`, resumes the thread and
// waits until its compaction turn completes.
func runCodexAppServerCompaction(ctx context.Context, bin, sessionID string, req runner.CompactSessionRequest) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "app-server")
	cmd.Dir = req.WorkingDir
	cmd.Env = mergedEnv(os.Environ(), req.Env)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("codex app-server stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("codex app-server stdout: %w", err)
	}
	var stderr boundedBuffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start codex app-server: %w", err)
	}
	defer func() {
		_ = stdin.Close()
		cancel()
		_ = cmd.Wait()
	}()

	session := newCodexAppServerSession(stdin, stdout)
	defer session.close()
	fail := func(step string, err error) error {
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			return fmt.Errorf("codex app-server %s: %w (stderr: %s)", step, err, detail)
		}
		return fmt.Errorf("codex app-server %s: %w", step, err)
	}

	if _, err := session.call(ctx, codexMethodInitialize, map[string]any{
		"clientInfo":   map[string]any{"name": "agent-manager", "version": "1"},
		"capabilities": map[string]any{},
	}); err != nil {
		return fail("initialize", err)
	}
	if err := session.notify(codexMethodInitialized); err != nil {
		return fail("initialized", err)
	}
	resume := map[string]any{"threadId": sessionID, "excludeTurns": true}
	if model := strings.TrimSpace(req.Model); model != "" {
		resume["model"] = model
	}
	if _, err := session.call(ctx, codexMethodThreadResume, resume); err != nil {
		return fail("thread/resume", err)
	}
	if _, err := session.call(ctx, codexMethodCompactStart, map[string]any{"threadId": sessionID}); err != nil {
		return fail("thread/compact/start", err)
	}
	if err := session.awaitCompaction(ctx, sessionID); err != nil {
		return fail("compaction", err)
	}
	return nil
}

// codexAppServerSession speaks newline-delimited JSON-RPC to one app-server.
type codexAppServerSession struct {
	in       io.Writer
	messages chan codexAppServerMessage
	readErr  chan error
	done     chan struct{}
	nextID   int
	// compacted records a contextCompaction item seen before its turn ended,
	// including one that arrived while a call was still awaiting its response.
	compacted map[string]bool
}

type codexAppServerMessage struct {
	ID     *int            `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func newCodexAppServerSession(in io.Writer, out io.Reader) *codexAppServerSession {
	s := &codexAppServerSession{
		in:        in,
		messages:  make(chan codexAppServerMessage, 64),
		readErr:   make(chan error, 1),
		done:      make(chan struct{}),
		compacted: map[string]bool{},
	}
	go func() {
		scanner := bufio.NewScanner(out)
		scanner.Buffer(make([]byte, 0, 64*1024), 64<<20)
		for scanner.Scan() {
			var msg codexAppServerMessage
			if json.Unmarshal(scanner.Bytes(), &msg) != nil {
				continue
			}
			select {
			case s.messages <- msg:
			case <-s.done:
				return
			}
		}
		err := scanner.Err()
		if err == nil {
			err = io.EOF
		}
		s.readErr <- err
		close(s.messages)
	}()
	return s
}

// close releases the output reader once the caller stops reading.
func (s *codexAppServerSession) close() { close(s.done) }

func (s *codexAppServerSession) send(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.in.Write(append(data, '\n'))
	return err
}

func (s *codexAppServerSession) notify(method string) error {
	return s.send(map[string]any{"method": method})
}

// call sends a request and returns its result, observing notifications that
// arrive before the response.
func (s *codexAppServerSession) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	s.nextID++
	id := s.nextID
	if err := s.send(map[string]any{"id": id, "method": method, "params": params}); err != nil {
		return nil, err
	}
	for {
		msg, err := s.next(ctx)
		if err != nil {
			return nil, err
		}
		if msg.ID != nil && *msg.ID == id && msg.Method == "" {
			if msg.Error != nil {
				return nil, fmt.Errorf("%s: %s", method, msg.Error.Message)
			}
			return msg.Result, nil
		}
		s.observe(msg)
	}
}

// awaitCompaction waits for the thread's compaction turn to complete.
func (s *codexAppServerSession) awaitCompaction(ctx context.Context, threadID string) error {
	for {
		msg, err := s.next(ctx)
		if err != nil {
			return err
		}
		if msg.Method == codexNotifyError {
			return fmt.Errorf("error notification: %s", strings.TrimSpace(string(msg.Params)))
		}
		s.observe(msg)
		if msg.Method != codexNotifyTurnCompleted {
			continue
		}
		var done struct {
			ThreadID string `json:"threadId"`
			Turn     struct {
				Status string          `json:"status"`
				Error  json.RawMessage `json:"error"`
			} `json:"turn"`
		}
		if json.Unmarshal(msg.Params, &done) != nil || done.ThreadID != threadID {
			continue
		}
		if done.Turn.Status != "completed" {
			return fmt.Errorf("compaction turn ended %s: %s", done.Turn.Status, strings.TrimSpace(string(done.Turn.Error)))
		}
		if !s.compacted[threadID] {
			return errors.New("compaction turn completed without a context compaction")
		}
		return nil
	}
}

func (s *codexAppServerSession) observe(msg codexAppServerMessage) {
	if msg.Method != codexNotifyItemCompleted {
		return
	}
	var item struct {
		ThreadID string `json:"threadId"`
		Item     struct {
			Type string `json:"type"`
		} `json:"item"`
	}
	if json.Unmarshal(msg.Params, &item) == nil && item.Item.Type == codexItemCompaction {
		s.compacted[item.ThreadID] = true
	}
}

func (s *codexAppServerSession) next(ctx context.Context) (codexAppServerMessage, error) {
	select {
	case <-ctx.Done():
		return codexAppServerMessage{}, ctx.Err()
	case msg, ok := <-s.messages:
		if !ok {
			return codexAppServerMessage{}, fmt.Errorf("app-server closed its output: %w", <-s.readErr)
		}
		return msg, nil
	}
}

type codexSessionRollout struct {
	contextTokens int64
	goalActive    bool
}

// readCodexSessionRollout reads the session's latest request size, from the
// rollout's tail, and whether a goal is active, from the whole rollout: a goal
// record can be far older than the tail. A rollout with no token count reads as
// a size of zero; one that cannot be found or read is an error.
func readCodexSessionRollout(home, sessionID string) (codexSessionRollout, error) {
	var out codexSessionRollout
	if home == "" {
		return out, errors.New("no CODEX_HOME for this run")
	}
	matches, err := filepath.Glob(filepath.Join(home, "sessions", "*", "*", "*", "*"+sessionID+"*.jsonl"))
	if err != nil {
		return out, err
	}
	if len(matches) == 0 {
		return out, fmt.Errorf("no rollout for session %s", sessionID)
	}
	sort.Strings(matches)
	f, err := os.Open(matches[len(matches)-1])
	if err != nil {
		return out, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return out, err
	}
	offset := info.Size() - codexRolloutTailBytes
	if offset < 0 {
		offset = 0
	}
	tail := make([]byte, info.Size()-offset)
	if _, err := f.ReadAt(tail, offset); err != nil && !errors.Is(err, io.EOF) {
		return out, err
	}
	out.contextTokens, _ = agenttranscript.LastContextTokens(tail, agenttranscript.HarnessCodex)
	var goal agenttranscript.GoalTracker
	if _, err := goal.Read(f, agenttranscript.HarnessCodex); err != nil {
		return out, err
	}
	out.goalActive = goal.Active()
	return out, nil
}

// mergedEnv overlays overrides on base, replacing keys that already exist.
func mergedEnv(base []string, overrides map[string]string) []string {
	out := make([]string, 0, len(base)+len(overrides))
	for _, kv := range base {
		key, _, _ := strings.Cut(kv, "=")
		if _, replaced := overrides[key]; !replaced {
			out = append(out, kv)
		}
	}
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out = append(out, key+"="+overrides[key])
	}
	return out
}

// boundedBuffer keeps the last 8 KiB written, for error reports.
type boundedBuffer struct{ buf []byte }

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	if over := len(b.buf) - 8<<10; over > 0 {
		b.buf = b.buf[over:]
	}
	return len(p), nil
}

func (b *boundedBuffer) String() string { return string(b.buf) }
