// Package sandbox — SandboxLauncher.
//
// SandboxLauncher implements runner.Launcher by routing process launches
// through workspace-sandbox /processes APIs. This is the protected-mode
// launch path: the agent process tree itself runs inside the sandbox
// container (bwrap isolation, network mode, git allowlist enforcement),
// not just on the host with a tracked overlay.
//
// Wire surface (workspace-sandbox /processes):
//
//   - POST   /sandboxes/{id}/processes
//     Starts a process; body controls argv, env, working dir, stdin.
//   - POST   /sandboxes/{id}/processes/{pid}/stdin?close=true
//     Streams stdin into the running process; close=true sends EOF.
//   - GET    /sandboxes/{id}/processes/{pid}/logs/stream?stream=stdout|stderr
//     Server-Sent Events stream of bytes. The server emits one
//     `event: exit` carrying ExitInfo JSON when the process exits, then
//     `event: end` and closes the connection.
//   - DELETE /sandboxes/{id}/processes/{pid}
//     Terminates the process (best-effort kill).
//
// The launcher opens two SSE streams (stdout + stderr), pipes their
// contents into the runner-visible io.Readers, and watches for the
// terminal `event: exit` to learn the precise exit code, signal, and
// OOM-killed flag. There is no client-side polling.
//
// Stdin flow: req.Stdin is read upfront and POSTed to the stdin endpoint
// with close=true, so the agent receives the prompt as if it had been
// piped from the host. This matches the prompt-via-stdin pattern claude_
// code, codex, and opencode all use.
//
// See execute/protected-sandbox-agent-launch and the four ws-sb follow-on
// items (ws-sb-streaming-process-logs, ws-sb-stdout-stderr-split,
// ws-sb-structured-exit-codes, ws-sb-native-stdin-pipe).
package sandbox

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// SandboxNamespacePath is the path at which the sandbox's merged
// (overlayfs) workspace is bind-mounted inside the bwrap mount namespace.
// It is the *only* path the agent process should see for its workspace —
// host paths to the merged dir do not exist inside the namespace and will
// fail bwrap's `--chdir` step before the runner ever launches.
//
// This constant must stay aligned with the `--bind <merged> /workspace`
// arg emitted by workspace-sandbox/api/internal/driver/bwrap.go (the
// `args = append(args, "--bind", s.MergedDir, "/workspace")` line). If
// either side changes, both must change.
//
// DOC: see scenarios/agent-manager/docs/internal/SEAMS.md #SandboxLauncher
// DOC: see scenarios/workspace-sandbox/docs/internal/SEAMS.md #BwrapMount
const SandboxNamespacePath = "/workspace"

// SandboxLauncher launches processes through workspace-sandbox /processes.
type SandboxLauncher struct {
	provider  *WorkspaceSandboxProvider
	sandboxID uuid.UUID

	// hostMergedOnce caches the host-side merged path lookup for this
	// sandbox. SandboxLauncher is constructed per sandbox so once the
	// path is resolved it can stay for the lifetime of the launcher.
	hostMergedOnce sync.Once
	hostMergedDir  string
	hostMergedErr  error
}

// NewSandboxLauncher builds a Launcher for the given sandbox using the
// workspace-sandbox provider. The provider's configured baseURL and HTTP
// client are reused.
func NewSandboxLauncher(provider *WorkspaceSandboxProvider, sandboxID uuid.UUID) *SandboxLauncher {
	return &SandboxLauncher{
		provider:  provider,
		sandboxID: sandboxID,
	}
}

// resolveHostMergedDir returns the host-side merged path for this sandbox,
// caching the lookup. Used to recognize callers that pass the host path
// where they should be passing the namespace path.
func (l *SandboxLauncher) resolveHostMergedDir(ctx context.Context) (string, error) {
	l.hostMergedOnce.Do(func() {
		l.hostMergedDir, l.hostMergedErr = l.provider.GetWorkspacePath(ctx, l.sandboxID)
	})
	return l.hostMergedDir, l.hostMergedErr
}

// translateCommandToNamespace rewrites a host-absolute binary path so it
// resolves inside the bwrap mount namespace. The contract reflects the
// post-2026-04-28 home-overlay layout in workspace-sandbox:
//
//   - The per-sandbox HOME overlay (driver.Mount → bwrap --bind ...
//     /home/<user>) makes the entire host $HOME visible inside the
//     namespace at the same host path. Anything under $HOME/* is
//     therefore reachable without rewrite — including symlink targets
//     deep under ~/.local/share.
//   - /usr, /bin, /usr/local/bin are bound at the same path by the
//     system / vrooli-aware profile.
//   - Anything else absolute (e.g. /opt/homebrew/bin/X on macOS) has no
//     known sandbox mapping; falling back to the basename + sandbox
//     PATH lookup is the safest behavior.
//
// Rules:
//
//   - $HOME/X → unchanged (HOME overlay)
//   - /usr/bin/X, /bin/X, /usr/local/bin/X → unchanged (system bind)
//   - any other host-absolute path → path.Base(X)
//   - relative path / bare basename → unchanged
//   - empty → unchanged
//
// Returns the (possibly-rewritten) command and a boolean indicating
// whether a rewrite happened (for diagnostic logging).
//
// hostHome is the agent-manager process's $HOME. On the supported
// single-machine deployment this matches workspace-sandbox's $HOME, so
// the host home overlay's lower layer is the same directory tree the
// agent-manager sees. Empty hostHome disables the $HOME rule but the
// basename fallback still applies.
//
// DOC: keep in sync with the bwrap home-overlay layout in
// workspace-sandbox/api/internal/driver/bwrap.go; if a future change
// reroutes the home overlay to a different namespace path, update this
// function and its tests.
func translateCommandToNamespace(command, hostHome string) (string, bool) {
	if command == "" {
		return command, false
	}
	if !strings.HasPrefix(command, "/") {
		// Already a basename or relative path — let sandbox PATH find it.
		return command, false
	}
	if hostHome != "" {
		homeLocal := strings.TrimRight(hostHome, "/") + "/.local/"
		if strings.HasPrefix(command, homeLocal) {
			// $HOME/.local/{bin,share}/X is bound at the same host path
			// inside the namespace by the vrooli-aware profile.
			return command, false
		}
	}
	switch {
	case strings.HasPrefix(command, "/usr/local/bin/"),
		strings.HasPrefix(command, "/usr/bin/"),
		strings.HasPrefix(command, "/bin/"):
		// Already a path the sandbox namespace mounts at the same location.
		return command, false
	}
	// Host-absolute path with no known sandbox mapping. Strip to basename
	// and rely on the sandbox PATH (set to /usr/local/bin:/usr/bin:/bin).
	// If the binary isn't discoverable that way, the run will fail with a
	// real exec error visible in the captured stderr — much better than
	// the silent host-path-doesn't-exist failure mode.
	return path.Base(command), true
}

// translateHostPathToNamespace rewrites a value that may contain the
// host-side merged path so that it points at the in-namespace mount.
//
// Rules:
//   - empty in → empty out (caller decides default).
//   - exact match against hostMerged → SandboxNamespacePath.
//   - path with hostMerged as a clean directory prefix (e.g. a subpath of
//     the merged dir) → SandboxNamespacePath + remainder.
//   - any other value → returned unchanged. Caller is responsible for
//     deciding whether that value is a contract violation.
//
// hostMerged is the resolved host path of the sandbox merged dir
// (e.g. /home/.../workspace-sandbox/<id>/merged). When it is empty the
// helper acts as identity — the launcher will still validate workingDir
// before POSTing.
//
// DOC: see scenarios/agent-manager/docs/internal/SEAMS.md #SandboxLauncher
func translateHostPathToNamespace(value, hostMerged string) string {
	if value == "" {
		return ""
	}
	if hostMerged == "" {
		return value
	}
	if value == hostMerged {
		return SandboxNamespacePath
	}
	prefix := hostMerged
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	if strings.HasPrefix(value, prefix) {
		return SandboxNamespacePath + "/" + strings.TrimPrefix(value, prefix)
	}
	return value
}

// Launch starts the process inside the sandbox.
func (l *SandboxLauncher) Launch(ctx context.Context, req runner.LaunchRequest) (runner.LaunchedProcess, error) {
	if l == nil || l.provider == nil {
		return nil, errors.New("SandboxLauncher: not configured")
	}
	if req.Command == "" {
		return nil, errors.New("SandboxLauncher: command is required")
	}

	// Read stdin upfront (when present). The runner pattern is
	// prompt-via-stdin: a single buffered prompt, not interactive bytes.
	var stdinBytes []byte
	if req.Stdin != nil {
		var err error
		stdinBytes, err = io.ReadAll(req.Stdin)
		if err != nil {
			return nil, fmt.Errorf("SandboxLauncher: read stdin: %w", err)
		}
	}
	withStdin := len(stdinBytes) > 0

	envMap := envSliceToMap(req.Env)

	// Translate host paths to in-namespace paths. Inside the bwrap mount
	// namespace the sandbox's merged dir is bind-mounted at
	// SandboxNamespacePath, so any host path supplied by the caller must
	// be rewritten before it crosses the API boundary. A host workdir
	// outside the sandbox is a contract violation, not a fallback case.
	hostMerged, hostErr := l.resolveHostMergedDir(ctx)
	if hostErr != nil {
		return nil, fmt.Errorf("SandboxLauncher: resolve host merged dir: %w", hostErr)
	}
	workingDir, wdErr := resolveWorkingDir(req.WorkingDir, hostMerged)
	if wdErr != nil {
		return nil, wdErr
	}
	envMap = translateEnvHostPaths(envMap, hostMerged)

	// Command + args translation: agent-manager's runners resolve binary
	// paths via host-side exec.LookPath and pass the absolute host path.
	// Inside the bwrap namespace those host paths usually don't exist
	// (only the vrooli-aware bind layout is mounted), so the host path
	// needs to be rewritten before crossing the API boundary.
	//
	// All three coding-agent runners use buildEnvWrappedLaunchRequest,
	// which sets Command="env" and stuffs the binary path into Args[1+].
	// We must therefore translate Args entries too — Command="env" alone
	// would resolve via PATH but bwrap's env shim then fails to exec the
	// host-absolute binary path embedded in args.
	hostHome := os.Getenv("HOME")
	command, _ := translateCommandToNamespace(req.Command, hostHome)
	translatedArgs := make([]string, len(req.Args))
	for i, a := range req.Args {
		translatedArgs[i], _ = translateCommandToNamespace(a, hostHome)
	}

	pid, err := l.startProcess(ctx, startProcessBody{
		Command:        command,
		Args:           translatedArgs,
		Env:            envMap,
		WorkingDir:     workingDir,
		IsolationLevel: "vrooli-aware",
		WithStdin:      withStdin,
	})
	if err != nil {
		return nil, err
	}

	// Stream stdin to the running process and signal EOF.
	if withStdin {
		if err := l.writeStdin(ctx, pid, stdinBytes, true); err != nil {
			// Best-effort kill so we don't leak an idle process.
			_ = l.killProcess(context.Background(), pid)
			return nil, fmt.Errorf("SandboxLauncher: write stdin: %w", err)
		}
	}

	proc := newSandboxLaunchedProcess(ctx, l, pid, req.IdleTimeout)
	return proc, nil
}

// startProcessBody is the JSON body shape for POST /processes.
type startProcessBody struct {
	Command        string            `json:"command"`
	Args           []string          `json:"args,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	WorkingDir     string            `json:"workingDir,omitempty"`
	IsolationLevel string            `json:"isolationLevel,omitempty"`
	WithStdin      bool              `json:"withStdin,omitempty"`
}

// startProcess POSTs /processes and returns the PID.
func (l *SandboxLauncher) startProcess(ctx context.Context, body startProcessBody) (int, error) {
	endpoint := fmt.Sprintf("/api/v1/sandboxes/%s/processes", l.sandboxID)
	resp, err := l.provider.doRequest(ctx, "POST", endpoint, body)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// 403 → structured guardrail denial (git allowlist, etc.).
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

// writeStdin POSTs the stdin bytes to /processes/{pid}/stdin. When close
// is true, the request also signals EOF to the process.
func (l *SandboxLauncher) writeStdin(ctx context.Context, pid int, content []byte, closeAfter bool) error {
	endpoint := fmt.Sprintf("/api/v1/sandboxes/%s/processes/%d/stdin", l.sandboxID, pid)
	if closeAfter {
		endpoint += "?close=true"
	}
	resp, err := l.provider.doRawRequest(ctx, "POST", endpoint, "application/octet-stream", bytes.NewReader(content))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		buf, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(buf))
	}
	return nil
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
	stderrR *io.PipeReader
	stderrW *io.PipeWriter

	waitCh  chan struct{}
	waitErr error

	// exitInfo holds the structured exit info reported by the server's
	// `event: exit` SSE frame. Set by whichever stream goroutine sees
	// the exit frame first; subsequent frames no-op (sync.Once).
	exitInfoMu sync.Mutex
	exitInfo   *remoteExitInfo
	exitOnce   sync.Once

	killOnce sync.Once
	killed   atomic.Bool

	timedOut atomic.Bool

	idleResetCh chan struct{}
}

// remoteExitInfo is the structured exit payload from the server's exit
// event. Mirrors workspace-sandbox process.ExitInfo.
type remoteExitInfo struct {
	ExitCode  int  `json:"exitCode"`
	Signal    int  `json:"signal,omitempty"`
	OOMKilled bool `json:"oomKilled,omitempty"`
}

func newSandboxLaunchedProcess(ctx context.Context, l *SandboxLauncher, pid int, idleTimeout time.Duration) *sandboxLaunchedProcess {
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	p := &sandboxLaunchedProcess{
		launcher:    l,
		pid:         pid,
		idleTimeout: idleTimeout,
		stdoutR:     stdoutR,
		stdoutW:     stdoutW,
		stderrR:     stderrR,
		stderrW:     stderrW,
		waitCh:      make(chan struct{}),
		idleResetCh: make(chan struct{}, 8),
	}

	// Two SSE streams (stdout + stderr) concurrently. Whichever finishes
	// first joins on the waitWg; when both are done the wait coordinator
	// closes waitCh.
	var streamsWg sync.WaitGroup
	streamsWg.Add(2)
	go p.runStream(ctx, "stdout", p.stdoutW, &streamsWg)
	go p.runStream(ctx, "stderr", p.stderrW, &streamsWg)

	// Coordinator: closes pipes + waitCh once both SSE streams have
	// terminated.
	go func() {
		streamsWg.Wait()
		_ = p.stdoutW.Close()
		_ = p.stderrW.Close()
		p.finalizeWaitErr()
		close(p.waitCh)
	}()

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
	select {
	case p.idleResetCh <- struct{}{}:
	default:
	}
}

func (p *sandboxLaunchedProcess) TimedOut() bool { return p.timedOut.Load() }

func (p *sandboxLaunchedProcess) Kill() {
	p.killOnce.Do(func() {
		p.killed.Store(true)
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

// Wait blocks until both streams have closed (process exited or kill).
func (p *sandboxLaunchedProcess) Wait() error {
	<-p.waitCh
	return p.waitErr
}

// runStream consumes the SSE stream for one log channel (stdout or stderr).
// Each `data:` event is forwarded into the corresponding pipe writer.
// The first `event: exit` frame seen by either runStream sets the
// process's exit info under sync.Once.
func (p *sandboxLaunchedProcess) runStream(ctx context.Context, stream string, w *io.PipeWriter, wg *sync.WaitGroup) {
	defer wg.Done()

	endpoint := fmt.Sprintf("/api/v1/sandboxes/%s/processes/%d/logs/stream?stream=%s", p.launcher.sandboxID, p.pid, stream)
	// Use the streaming client (no total deadline) so long-running agent
	// runs aren't cut off by the 30s default-client timeout. The stream
	// stays open until the process exits, the server emits event:end, or
	// ctx is cancelled by the caller.
	resp, err := p.launcher.provider.doStreamRequest(ctx, "GET", endpoint)
	if err != nil {
		// Context cancellation surfaces as ctx.Err; not unique to this stream.
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 404 right after start is rare but possible if the server hasn't
		// finished registering the writer; the runner can survive an
		// empty stream because parseStreamEvents tolerates it.
		return
	}

	parser := newSSEParser(resp.Body)
	for {
		event, ok := parser.next()
		if !ok {
			return
		}
		switch event.eventType {
		case "exit":
			p.recordExitInfo(event.data)
		case "end":
			return
		case "error":
			// Server-side error mid-stream; surface as wait error.
			p.exitInfoMu.Lock()
			if p.exitInfo == nil {
				p.exitInfo = &remoteExitInfo{ExitCode: 1}
			}
			p.exitInfoMu.Unlock()
			return
		default:
			// Default event type "" or "message" → forward bytes.
			if len(event.data) > 0 {
				_, _ = w.Write(event.data)
			}
		}
	}
}

// recordExitInfo parses the JSON payload of an `event: exit` frame and
// stores it. First call wins.
func (p *sandboxLaunchedProcess) recordExitInfo(payload []byte) {
	p.exitOnce.Do(func() {
		var info remoteExitInfo
		if err := json.Unmarshal(bytes.TrimSpace(payload), &info); err != nil {
			info = remoteExitInfo{ExitCode: 1}
		}
		p.exitInfoMu.Lock()
		p.exitInfo = &info
		p.exitInfoMu.Unlock()
	})
}

// ErrSandboxNoExitInfo signals that both SSE log streams closed without
// the server emitting `event: exit`. Under the current contract (after
// the workspace-sandbox WaitForExit fix in process.StreamProcessLogs),
// this should not happen for a real exit — every process the tracker
// knows about gets its ExitInfo recorded by spawnExitReaper. If the
// client sees this error, either the process was untracked (a bug) or
// the connection dropped between exit and notify. In either case the
// run is NOT a clean success; callers must surface it as failure.
var ErrSandboxNoExitInfo = errors.New("sandbox process ended without exit info")

// finalizeWaitErr sets p.waitErr from the recorded exit info (or kill state).
func (p *sandboxLaunchedProcess) finalizeWaitErr() {
	if p.killed.Load() {
		p.waitErr = errors.New("process killed")
		return
	}
	p.exitInfoMu.Lock()
	info := p.exitInfo
	p.exitInfoMu.Unlock()
	if info == nil {
		// Both SSE streams ended without `event: exit`. Under the new
		// contract (see ErrSandboxNoExitInfo) this is a failure — the
		// previous "treat as success" policy let bwrap launch errors
		// masquerade as exit-0 successes. Callers must surface it.
		//
		// We wrap the sentinel in a typed domain error so the
		// orchestration categorizer routes it to ErrCodeSandboxNoExitInfo
		// (SANDBOX_NO_EXIT_INFO) instead of falling through to generic
		// INTERNAL. errors.Is(p.waitErr, ErrSandboxNoExitInfo) keeps
		// working because SandboxError.Unwrap returns the cause.
		p.waitErr = domain.NewSandboxNoExitInfoError(ErrSandboxNoExitInfo)
		return
	}
	if info.ExitCode == 0 && info.Signal == 0 {
		return
	}
	p.waitErr = &remoteExitError{code: info.ExitCode, signal: info.Signal, oomKilled: info.OOMKilled}
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
//
// Runners share an exit-code interface (ExitCode() int, satisfied by both
// *exec.ExitError and *remoteExitError) so the wait-error type-switch is
// uniform across host and sandbox launches. See runner.extractExitCode.
type remoteExitError struct {
	code      int
	signal    int
	oomKilled bool
}

func (e *remoteExitError) Error() string {
	switch {
	case e.oomKilled:
		return fmt.Sprintf("remote process killed by OOM (exit %d signal %d)", e.code, e.signal)
	case e.signal != 0:
		return fmt.Sprintf("remote process killed by signal %d (exit %d)", e.signal, e.code)
	default:
		return fmt.Sprintf("remote process exited with code %d", e.code)
	}
}

// ExitCode satisfies the runner-side exitCoder interface.
func (e *remoteExitError) ExitCode() int { return e.code }

// =============================================================================
// SSE parser
// =============================================================================

// sseEvent represents one parsed Server-Sent Events block.
type sseEvent struct {
	eventType string
	data      []byte
}

// sseParser scans an io.Reader for newline-delimited SSE events.
type sseParser struct {
	scanner *bufio.Scanner
}

func newSSEParser(r io.Reader) *sseParser {
	s := bufio.NewScanner(r)
	// Larger buffer than default 64KB so single long lines don't trip the
	// scanner. 4MB is plenty for a JSON exit frame or a chunky log line.
	s.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	return &sseParser{scanner: s}
}

// next returns the next SSE event; ok=false when the stream ended.
func (p *sseParser) next() (sseEvent, bool) {
	var ev sseEvent
	var dataBuf bytes.Buffer
	hasData := false
	for p.scanner.Scan() {
		line := p.scanner.Text()
		if line == "" {
			// Empty line terminates the event.
			if !hasData && ev.eventType == "" {
				continue
			}
			ev.data = dataBuf.Bytes()
			return ev, true
		}
		if strings.HasPrefix(line, ":") {
			// Comment line; ignore.
			continue
		}
		idx := strings.Index(line, ":")
		var field, value string
		if idx >= 0 {
			field = line[:idx]
			value = strings.TrimPrefix(line[idx+1:], " ")
		} else {
			field = line
			value = ""
		}
		switch field {
		case "event":
			ev.eventType = value
		case "data":
			if hasData {
				dataBuf.WriteByte('\n')
			}
			dataBuf.WriteString(value)
			hasData = true
		}
	}
	// EOF or scanner error.
	if hasData || ev.eventType != "" {
		ev.data = dataBuf.Bytes()
		return ev, true
	}
	return sseEvent{}, false
}

// =============================================================================
// helpers
// =============================================================================

// resolveWorkingDir picks the in-namespace workingDir for a launch request.
//
//   - empty → SandboxNamespacePath (the merged-dir mount inside bwrap).
//   - matches the host merged dir (or a subpath of it) → translated to
//     the corresponding in-namespace path.
//   - already SandboxNamespacePath (or a subpath) → returned unchanged.
//   - any other absolute host path → contract violation; returned as
//     *LaunchBlocked{Code: "workdir_outside_sandbox"} so the runner can
//     surface it on the run timeline rather than fail opaquely later.
func resolveWorkingDir(requested, hostMerged string) (string, error) {
	if requested == "" {
		return SandboxNamespacePath, nil
	}
	// Already in-namespace (constant or any path under it).
	if requested == SandboxNamespacePath ||
		strings.HasPrefix(requested, SandboxNamespacePath+"/") {
		return requested, nil
	}
	if hostMerged != "" {
		if requested == hostMerged ||
			strings.HasPrefix(requested, strings.TrimSuffix(hostMerged, "/")+"/") {
			return translateHostPathToNamespace(requested, hostMerged), nil
		}
	}
	return "", &LaunchBlocked{
		Code:    "workdir_outside_sandbox",
		Message: fmt.Sprintf("workingDir %q is outside the sandbox merged dir; sandbox-routed launches must run inside %s", requested, SandboxNamespacePath),
	}
}

// translateEnvHostPaths rewrites any env values that exactly match the
// sandbox's host merged path so that the agent process inside the
// namespace sees the in-namespace path. Other values pass through. The
// VROOLI_SANDBOX_MERGED contract specifically expects this translation
// (the env var name documents semantics; the value must be visible to
// the agent).
func translateEnvHostPaths(env map[string]string, hostMerged string) map[string]string {
	if len(env) == 0 || hostMerged == "" {
		return env
	}
	out := make(map[string]string, len(env))
	for k, v := range env {
		out[k] = translateHostPathToNamespace(v, hostMerged)
	}
	return out
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
