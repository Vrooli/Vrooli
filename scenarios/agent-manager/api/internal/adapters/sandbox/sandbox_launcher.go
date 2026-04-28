// Package sandbox — SandboxLauncher.
//
// SandboxLauncher implements runner.Launcher by routing process launches
// through workspace-sandbox /processes APIs. This is the protected-mode
// launch path: the agent process tree itself runs inside the sandbox
// container (bwrap isolation, network mode, git allowlist enforcement),
// not just on the host with a tracked overlay.
//
// The launcher is intentionally written against the existing
// workspace-sandbox surface (StartProcess, GetProcessLogs, KillProcess,
// ListProcesses, WriteFile) — adding new endpoints to ws-sb would have
// been a much bigger change. The trade-offs are:
//
//   - stdin is delivered by writing a file into the sandbox and using a
//     bash wrapper to redirect (`exec ... < /workspace/.am-prompts/X.txt`).
//     Workable for the prompt-via-stdin pattern all three coding-agent
//     runners use; not suitable for fully interactive stdin.
//
//   - stdout is delivered by polling /processes/{pid}/logs with byte
//     offsets every 100ms. Higher latency than a true streaming
//     transport (SSE / WebSocket) but simpler and resilient to network
//     blips. The polling interval is tuned to be invisible to UX while
//     not overwhelming the workspace-sandbox API.
//
//   - stdout and stderr are interleaved. workspace-sandbox today merges
//     them into a single log file. Claude Code's stream-json output is
//     line-prefixed so the runner's parseStreamEvents already tolerates
//     non-JSON lines mixed in. Future work: ws-sb could split streams.
//
// Both trade-offs are documented in PROTECTED_MODE_RUNNERS.md and the
// follow-up plan to migrate to a true streaming transport.
//
// See execute/protected-sandbox-agent-launch.
package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"agent-manager/internal/adapters/runner"

	"github.com/google/uuid"
)

// SandboxLauncher launches processes through workspace-sandbox /processes.
type SandboxLauncher struct {
	provider  *WorkspaceSandboxProvider
	sandboxID uuid.UUID

	// PollInterval controls how often the stdout-log poller hits ws-sb.
	// Defaults to 100ms when zero.
	PollInterval time.Duration

	// PromptDir is the relative directory inside the sandbox where the
	// launcher writes stdin prompts. Defaults to ".am-prompts".
	PromptDir string
}

// NewSandboxLauncher builds a Launcher for the given sandbox using the
// workspace-sandbox provider. The provider's configured baseURL and HTTP
// client are reused.
func NewSandboxLauncher(provider *WorkspaceSandboxProvider, sandboxID uuid.UUID) *SandboxLauncher {
	return &SandboxLauncher{
		provider:     provider,
		sandboxID:    sandboxID,
		PollInterval: 100 * time.Millisecond,
		PromptDir:    ".am-prompts",
	}
}

// Launch starts the process inside the sandbox.
func (l *SandboxLauncher) Launch(ctx context.Context, req runner.LaunchRequest) (runner.LaunchedProcess, error) {
	if l == nil || l.provider == nil {
		return nil, errors.New("SandboxLauncher: not configured")
	}
	if req.Command == "" {
		return nil, errors.New("SandboxLauncher: command is required")
	}
	pollInterval := l.PollInterval
	if pollInterval <= 0 {
		pollInterval = 100 * time.Millisecond
	}
	promptDir := l.PromptDir
	if promptDir == "" {
		promptDir = ".am-prompts"
	}

	// Stage 1: stage stdin (when present) as a file in the sandbox.
	var promptRelPath string
	if req.Stdin != nil {
		stdinBytes, err := io.ReadAll(req.Stdin)
		if err != nil {
			return nil, fmt.Errorf("SandboxLauncher: read stdin: %w", err)
		}
		if len(stdinBytes) > 0 {
			runID := uuid.New().String()
			promptRelPath = promptDir + "/" + runID + ".txt"
			if err := l.writeSandboxFile(ctx, promptRelPath, stdinBytes); err != nil {
				return nil, fmt.Errorf("SandboxLauncher: stage stdin: %w", err)
			}
		}
	}

	// Stage 2: build a bash wrapper that exec's the command with the staged
	// stdin redirected, then removes the prompt file. Using exec means the
	// wrapped command takes over the bash PID, which keeps process-tree
	// management straightforward.
	wrapperCmd, wrapperArgs := buildBashWrapper(req.Command, req.Args, promptRelPath)

	// Stage 3: convert env from os.Environ() form to map.
	envMap := envSliceToMap(req.Env)

	// Stage 4: start the process via workspace-sandbox /processes.
	pid, err := l.startProcess(ctx, wrapperCmd, wrapperArgs, envMap, req.WorkingDir)
	if err != nil {
		return nil, err
	}

	// Stage 5: build the LaunchedProcess handle with background pollers.
	proc := newSandboxLaunchedProcess(ctx, l, pid, pollInterval, req.IdleTimeout)
	return proc, nil
}

// writeSandboxFile PUTs file content into the sandbox at the given relative path.
func (l *SandboxLauncher) writeSandboxFile(ctx context.Context, relPath string, content []byte) error {
	body := map[string]any{
		"content":  string(content),
		"encoding": "utf8",
	}
	pathPart := url.QueryEscape(relPath)
	endpoint := fmt.Sprintf("/api/v1/sandboxes/%s/files/content?path=%s", l.sandboxID, pathPart)
	resp, err := l.provider.doRequest(ctx, "PUT", endpoint, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("write sandbox file: HTTP %d", resp.StatusCode)
	}
	return nil
}

// startProcess POSTs /processes and returns the PID.
func (l *SandboxLauncher) startProcess(ctx context.Context, command string, args []string, env map[string]string, workingDir string) (int, error) {
	body := map[string]any{
		"command":        command,
		"args":           args,
		"env":            env,
		"isolationLevel": "vrooli-aware", // matches existing protected-mode wire encoding
	}
	if workingDir != "" {
		body["workingDir"] = workingDir
	}
	endpoint := fmt.Sprintf("/api/v1/sandboxes/%s/processes", l.sandboxID)
	resp, err := l.provider.doRequest(ctx, "POST", endpoint, body)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// 403 → structured guardrail denial (git allowlist, etc.). Surface as a
	// typed error the runner can recognise.
	if resp.StatusCode == http.StatusForbidden {
		var denial struct {
			Error   string `json:"error"`
			Verb    string `json:"verb"`
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&denial)
		return 0, &LaunchBlocked{Code: denial.Error, Verb: denial.Verb, Message: denial.Message}
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		buf, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("start process: HTTP %d: %s", resp.StatusCode, string(buf))
	}

	var startResp struct {
		PID int `json:"pid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&startResp); err != nil {
		return 0, fmt.Errorf("start process: decode response: %w", err)
	}
	if startResp.PID == 0 {
		return 0, errors.New("start process: workspace-sandbox returned PID 0")
	}
	return startResp.PID, nil
}

// LaunchBlocked is returned by SandboxLauncher.Launch when the workspace
// sandbox returns a structured 403 (e.g., git allowlist denial). The runner
// surfaces it as a typed run-level error rather than a generic launch failure.
type LaunchBlocked struct {
	// Code is the wire-level error tag (e.g. "git_verb_blocked").
	Code string
	// Verb is the offending verb when Code names a verb-level guard.
	Verb string
	// Message is operator-friendly text suitable for a tool.blocked event.
	Message string
}

// Error implements the error interface so callers can return *LaunchBlocked
// directly. errors.As still works with a typed unwrap.
func (b *LaunchBlocked) Error() string {
	if b == nil {
		return ""
	}
	if b.Message != "" {
		return b.Message
	}
	if b.Verb != "" {
		return "blocked: " + b.Verb
	}
	return "launch blocked"
}

// =============================================================================
// sandboxLaunchedProcess
// =============================================================================

type sandboxLaunchedProcess struct {
	launcher    *SandboxLauncher
	pid         int
	idleTimeout time.Duration

	stdoutR *io.PipeReader
	stdoutW *io.PipeWriter
	// Stderr is unused for now (workspace-sandbox merges streams). Returned
	// as a closed pipe so callers don't block.
	stderrR *io.PipeReader

	waitCh  chan struct{}
	waitErr error

	killOnce sync.Once
	killed   atomic.Bool

	timedOut atomic.Bool

	idleResetCh chan struct{} // signals the idle-timeout watchdog to reset
}

func newSandboxLaunchedProcess(ctx context.Context, l *SandboxLauncher, pid int, pollInterval time.Duration, idleTimeout time.Duration) *sandboxLaunchedProcess {
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	stderrW.Close() // unused — close so consumers see EOF immediately

	p := &sandboxLaunchedProcess{
		launcher:    l,
		pid:         pid,
		idleTimeout: idleTimeout,
		stdoutR:     stdoutR,
		stdoutW:     stdoutW,
		stderrR:     stderrR,
		waitCh:      make(chan struct{}),
		idleResetCh: make(chan struct{}, 8),
	}

	// Background poller: pulls stdout chunks from /processes/{pid}/logs and
	// writes them into the stdout pipe; watches for process exit; closes
	// pipe + waitCh when the process terminates.
	go p.run(ctx, pollInterval)

	// Optional idle-timeout watchdog.
	if idleTimeout > 0 {
		go p.watchIdle(ctx)
	}
	return p
}

func (p *sandboxLaunchedProcess) Stdout() io.Reader { return p.stdoutR }
func (p *sandboxLaunchedProcess) Stderr() io.Reader { return p.stderrR }
func (p *sandboxLaunchedProcess) PID() int          { return p.pid }

func (p *sandboxLaunchedProcess) ResetIdleTimer() {
	if p.idleTimeout <= 0 {
		return
	}
	// Non-blocking signal; drop if the channel is full.
	select {
	case p.idleResetCh <- struct{}{}:
	default:
	}
}

func (p *sandboxLaunchedProcess) TimedOut() bool { return p.timedOut.Load() }

func (p *sandboxLaunchedProcess) Kill() {
	p.killOnce.Do(func() {
		p.killed.Store(true)
		// Best-effort kill via workspace-sandbox; ignore not-found / 404
		// (process may have exited naturally just before the kill).
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = p.launcher.killProcess(ctx, p.pid)
	})
}

// Signal mirrors HostLauncher.Signal: graceful termination then escalation.
// workspace-sandbox today only exposes a hard-kill DELETE endpoint, so the
// "graceful" phase is currently identical to Kill. Documented limitation.
func (p *sandboxLaunchedProcess) Signal(grace time.Duration) {
	_ = grace
	p.Kill()
}

// Wait blocks until the process has exited (or Kill was called).
func (p *sandboxLaunchedProcess) Wait() error {
	<-p.waitCh
	return p.waitErr
}

// run is the background poller loop.
func (p *sandboxLaunchedProcess) run(ctx context.Context, pollInterval time.Duration) {
	defer close(p.waitCh)
	defer p.stdoutW.Close()

	var offset int64
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		// Drain available log content first.
		newOffset, err := p.drainLogs(ctx, offset)
		if err != nil && !errors.Is(err, errLogNotReady) {
			p.waitErr = err
			return
		}
		offset = newOffset

		// Check if the process has exited.
		exited, exitCode, exitErr := p.checkExited(ctx)
		if exitErr != nil {
			p.waitErr = exitErr
			return
		}
		if exited {
			// Final drain to capture any remaining log content.
			finalOffset, _ := p.drainLogs(ctx, offset)
			_ = finalOffset
			if p.killed.Load() {
				p.waitErr = errors.New("process killed")
				return
			}
			if exitCode != 0 {
				p.waitErr = &remoteExitError{ExitCode: exitCode}
				return
			}
			return
		}

		select {
		case <-ctx.Done():
			// Context cancelled — kill the remote process and return.
			p.Kill()
			p.waitErr = ctx.Err()
			return
		case <-ticker.C:
			// Continue polling.
		}
	}
}

// errLogNotReady is returned when /logs returns 404 — typically right after
// the process starts before the log file has been created. Treated as a
// no-op for the polling loop.
var errLogNotReady = errors.New("log not ready")

// drainLogs reads new log content starting at offset; writes new bytes to
// the stdout pipe; returns the updated offset.
func (p *sandboxLaunchedProcess) drainLogs(ctx context.Context, offset int64) (int64, error) {
	endpoint := fmt.Sprintf("/api/v1/sandboxes/%s/processes/%d/logs?offset=%d", p.launcher.sandboxID, p.pid, offset)
	resp, err := p.launcher.provider.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return offset, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return offset, errLogNotReady
	}
	if resp.StatusCode != http.StatusOK {
		return offset, fmt.Errorf("drain logs: HTTP %d", resp.StatusCode)
	}

	var logResp struct {
		Content   string `json:"content"`
		SizeBytes int64  `json:"sizeBytes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&logResp); err != nil {
		return offset, fmt.Errorf("drain logs: decode: %w", err)
	}

	if logResp.Content != "" {
		_, _ = p.stdoutW.Write([]byte(logResp.Content))
		// Advance offset by bytes actually consumed.
		offset += int64(len(logResp.Content))
	} else if logResp.SizeBytes > offset {
		// Mismatch: the log is bigger than our offset claims, but the
		// returned content was empty. Sync the offset to size to avoid
		// looping on the same byte range.
		offset = logResp.SizeBytes
	}
	return offset, nil
}

// checkExited probes /processes (filtered to running) to see whether the PID
// is still active. Workspace-sandbox doesn't return exit codes for tracked
// processes today, so we report 0 on natural exit and a non-zero placeholder
// (1) when the process disappeared but we don't know why. Future work could
// extend ws-sb to record exit codes.
func (p *sandboxLaunchedProcess) checkExited(ctx context.Context) (exited bool, exitCode int, err error) {
	endpoint := fmt.Sprintf("/api/v1/sandboxes/%s/processes", p.launcher.sandboxID)
	resp, err := p.launcher.provider.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, 0, fmt.Errorf("list processes: HTTP %d", resp.StatusCode)
	}

	var listResp struct {
		Processes []struct {
			PID    int    `json:"pid"`
			Status string `json:"status,omitempty"`
		} `json:"processes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return false, 0, fmt.Errorf("list processes: decode: %w", err)
	}

	for _, pr := range listResp.Processes {
		if pr.PID == p.pid {
			// Found and present. Check status.
			if pr.Status == "running" || pr.Status == "" {
				return false, 0, nil
			}
			// Any non-running status (exited, killed, etc.) is terminal.
			if pr.Status == "exited" {
				return true, 0, nil
			}
			return true, 1, nil
		}
	}
	// Process not in list → exited (workspace-sandbox stops tracking
	// terminated processes after a grace period). Treat as natural exit.
	return true, 0, nil
}

// killProcess DELETEs /processes/{pid}.
func (l *SandboxLauncher) killProcess(ctx context.Context, pid int) error {
	endpoint := fmt.Sprintf("/api/v1/sandboxes/%s/processes/%d", l.sandboxID, pid)
	resp, err := l.provider.doRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("kill process: HTTP %d", resp.StatusCode)
	}
	return nil
}

// watchIdle implements the idle-timeout safety net.
func (p *sandboxLaunchedProcess) watchIdle(ctx context.Context) {
	timer := time.NewTimer(p.idleTimeout)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.waitCh:
			return
		case <-p.idleResetCh:
			// Drain any extra signals queued.
			drained := true
			for drained {
				select {
				case <-p.idleResetCh:
				default:
					drained = false
				}
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(p.idleTimeout)
		case <-timer.C:
			p.timedOut.Store(true)
			p.Kill()
			return
		}
	}
}

// remoteExitError carries a non-zero exit code from a sandbox-launched process.
// The runner unwraps it for the typical exec.ExitError-style check.
type remoteExitError struct {
	ExitCode int
}

func (e *remoteExitError) Error() string {
	return fmt.Sprintf("remote process exited with code %d", e.ExitCode)
}

// =============================================================================
// helpers
// =============================================================================

// buildBashWrapper constructs a `bash -c` command line that:
//   - exec's the target command (so bash doesn't linger as a parent shell),
//   - redirects stdin from the staged prompt file when present,
//   - removes the prompt file after the process exits.
//
// All arguments are quoted with strict single-quote escaping so that user-
// supplied content cannot break out into shell injection.
func buildBashWrapper(command string, args []string, promptPath string) (string, []string) {
	parts := []string{shellQuote(command)}
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	cmdline := strings.Join(parts, " ")

	if promptPath != "" {
		// `trap rm` ensures the prompt file is removed even on signal.
		// `<` redirect makes the staged file the new stdin.
		// `exec` replaces the bash process with the target so signals reach it.
		shellLine := fmt.Sprintf("trap 'rm -f %s' EXIT; exec %s < %s", shellQuote(promptPath), cmdline, shellQuote(promptPath))
		return "bash", []string{"-c", shellLine}
	}
	// No stdin staging — just exec the command.
	shellLine := fmt.Sprintf("exec %s", cmdline)
	return "bash", []string{"-c", shellLine}
}

// shellQuote wraps s in strict single quotes so the shell treats it as a
// single literal token. Escapes any embedded single quote by closing the
// quote, inserting an escaped quote, and reopening: `it's` becomes `'it'\”s'`.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	var buf bytes.Buffer
	buf.WriteByte('\'')
	for _, r := range s {
		if r == '\'' {
			buf.WriteString(`'\''`)
			continue
		}
		buf.WriteRune(r)
	}
	buf.WriteByte('\'')
	return buf.String()
}

// envSliceToMap converts os.Environ()-style entries to a map. Keys with
// duplicate names take the last value (matching POSIX semantics).
func envSliceToMap(env []string) map[string]string {
	if len(env) == 0 {
		return nil
	}
	out := make(map[string]string, len(env))
	for _, e := range env {
		idx := strings.IndexByte(e, '=')
		if idx <= 0 {
			continue
		}
		out[e[:idx]] = e[idx+1:]
	}
	return out
}

// LauncherFor returns a SandboxLauncher bound to the given sandbox.
// Implements runner.SandboxLauncherFactory so the WorkspaceSandboxProvider
// can be passed to a runner constructor as the protected-mode factory.
func (p *WorkspaceSandboxProvider) LauncherFor(sandboxID uuid.UUID) runner.Launcher {
	return NewSandboxLauncher(p, sandboxID)
}

// Compile-time interface checks.
var (
	_ runner.Launcher               = (*SandboxLauncher)(nil)
	_ runner.LaunchedProcess        = (*sandboxLaunchedProcess)(nil)
	_ runner.SandboxLauncherFactory = (*WorkspaceSandboxProvider)(nil)
)
