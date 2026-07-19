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
	"agent-manager/internal/config"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// SandboxLauncher launches processes through workspace-sandbox /processes.
type SandboxLauncher struct {
	provider  *WorkspaceSandboxProvider
	sandboxID uuid.UUID

	// layoutOnce caches the NamespaceLayout (host merged dir, reported
	// agent-visible workspace path, path-illusion flag, home-overlay state)
	// fetched from workspace-sandbox. SandboxLauncher is constructed per
	// sandbox so a single lookup serves the launcher's lifetime.
	layoutOnce sync.Once
	layout     NamespaceLayout
	layoutErr  error
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

// resolveNamespaceLayout fetches the workspace-sandbox-side metadata
// (home overlay state) and caches it. The launcher refuses host-$HOME
// commands when the overlay is absent, so this lookup is on the hot
// path of every Launch.
func (l *SandboxLauncher) resolveNamespaceLayout(ctx context.Context) (NamespaceLayout, error) {
	l.layoutOnce.Do(func() {
		sb, err := l.provider.Get(ctx, l.sandboxID)
		if err != nil {
			l.layoutErr = err
			return
		}
		l.layout = NamespaceLayout{
			HostHome:         os.Getenv("HOME"),
			HomeOverlayState: sb.HomeOverlayState,
			HostMerged:       sb.WorkDir,
			WorkspacePath:    sb.WorkspacePath,
			PathIllusion:     sb.PathIllusion,
		}
	})
	return l.layout, l.layoutErr
}

// NamespaceLayout describes what's visible inside the sandbox's bwrap
// mount namespace from the agent-manager's perspective. Built from the
// sandbox metadata returned by workspace-sandbox; consulted by every
// command/path translation so the rules live in one place.
//
// DOC: namespace contract seam. See scenarios/agent-manager/docs/internal/SEAMS.md.
type NamespaceLayout struct {
	// HostHome is the host-side $HOME path. Empty disables $HOME-prefix
	// reasoning.
	HostHome string

	// HomeOverlayState mirrors workspace-sandbox's per-sandbox state. When
	// HomeOverlayPresent the host $HOME tree is reachable inside the
	// namespace at the same host path; otherwise $HOME-prefixed paths
	// MUST NOT be passed through unchanged.
	HomeOverlayState HomeOverlayState

	// HostMerged is the host-side overlay merged dir (Sandbox.WorkDir). It
	// is the prefix host-absolute caller paths are recognized and rewritten
	// from.
	HostMerged string

	// WorkspacePath is the agent-visible workspace path reported by
	// workspace-sandbox — the target of host→namespace translation and the
	// default working dir. "/workspace" under a path illusion, else the host
	// merged dir (identity layout).
	WorkspacePath string

	// PathIllusion is true when the containment backend rewrites host paths
	// onto WorkspacePath (bwrap). false means WorkspacePath == HostMerged and
	// translation is identity; command basename fallback is then a no-op
	// because host-absolute binaries resolve at their real paths.
	PathIllusion bool
}

// PathEntries returns the PATH entries the sandbox profile exposes
// inside the namespace. Single source of truth for command-resolution;
// used by both translateCommandToNamespace (for fallbacks) and any
// future env composition that needs to mirror the profile's PATH.
//
// Currently returns the static vrooli-aware PATH layout. When the home
// overlay is absent, $HOME-relative CLI and toolchain paths are excluded.
//
// DOC: namespace contract seam.
func (l NamespaceLayout) PathEntries() []string {
	base := []string{"/usr/local/bin", "/usr/bin", "/bin"}
	if IsHomeOverlayPresent(l.HomeOverlayState) && l.HostHome != "" {
		home := strings.TrimRight(l.HostHome, "/")
		return append([]string{
			home + "/.vrooli/bin",
			home + "/go/bin",
			home + "/.local/bin",
		}, base...)
	}
	return base
}

// ErrCommandHomeOverlayUnavailable is returned by translateCommandToNamespace
// when the command lives under $HOME but the sandbox's home overlay is
// not Present. Surfaces upstream as SANDBOX_HOME_OVERLAY_UNAVAILABLE in
// the run timeline rather than the silent-and-confusing
// `env: …/claude: No such file or directory` at exec time.
//
// DOC: home-overlay seam — agent-manager-side enforcement.
type ErrCommandHomeOverlayUnavailable struct {
	Command string
	State   HomeOverlayState
}

func (e *ErrCommandHomeOverlayUnavailable) Error() string {
	return fmt.Sprintf(
		"command %q lives under $HOME but the sandbox home overlay is %q (need %q); the agent CLI is not reachable inside the namespace",
		e.Command, e.State, HomeOverlayPresent,
	)
}

// Code returns the stable error code surfaced to the run timeline.
func (e *ErrCommandHomeOverlayUnavailable) Code() string { return "SANDBOX_HOME_OVERLAY_UNAVAILABLE" }

// translateCommandToNamespace rewrites a host-absolute binary path so it
// resolves inside the bwrap mount namespace, OR returns
// ErrCommandHomeOverlayUnavailable when the command requires the host
// $HOME overlay and the sandbox didn't get one.
//
// The contract reflects the post-2026-04-28 home-overlay layout in
// workspace-sandbox:
//
//   - The per-sandbox HOME overlay (driver.Mount → bwrap --bind ...
//     /home/<user>) makes the entire host $HOME visible inside the
//     namespace at the same host path — IFF state == Present.
//   - /usr, /bin, /usr/local/bin are bound at the same path by the
//     system / vrooli-aware profile.
//   - Anything else absolute (e.g. /opt/homebrew/bin/X on macOS) has no
//     known sandbox mapping; falling back to the basename + sandbox
//     PATH lookup is the safest behavior.
//
// Rules:
//
//   - $HOME/X with state==Present → unchanged
//   - $HOME/X with state!=Present → ErrCommandHomeOverlayUnavailable
//   - /usr/bin/X, /bin/X, /usr/local/bin/X → unchanged (system bind)
//   - any other host-absolute path, pathIllusion → path.Base(X)
//   - any other host-absolute path, identity layout → unchanged
//   - relative path / bare basename → unchanged
//   - empty → unchanged
//
// The basename fallback exists only for the path-illusion layout, where
// arbitrary host paths do not exist inside the namespace. Under an identity
// layout (pathIllusion=false — copy driver, macOS seatbelt) the process
// runs against real host paths, so stripping to a basename would break a
// binary that legitimately lives outside the system bind dirs.
//
// DOC: home-overlay seam.
func translateCommandToNamespace(command string, layout NamespaceLayout) (string, error) {
	if command == "" {
		return command, nil
	}
	if !strings.HasPrefix(command, "/") {
		return command, nil
	}
	if layout.HostHome != "" {
		homeAbs := strings.TrimRight(layout.HostHome, "/") + "/"
		if strings.HasPrefix(command, homeAbs) {
			if !IsHomeOverlayPresent(layout.HomeOverlayState) {
				return "", &ErrCommandHomeOverlayUnavailable{
					Command: command,
					State:   layout.HomeOverlayState,
				}
			}
			// $HOME/X is bound at the same host path inside the namespace
			// by the vrooli-aware profile (via the home overlay).
			return command, nil
		}
	}
	switch {
	case strings.HasPrefix(command, "/usr/local/bin/"),
		strings.HasPrefix(command, "/usr/bin/"),
		strings.HasPrefix(command, "/bin/"):
		return command, nil
	}
	if !layout.PathIllusion {
		// Identity layout: the process sees real host paths, so a
		// host-absolute binary path resolves as-is.
		return command, nil
	}
	// Path-illusion layout: a host-absolute path with no known sandbox
	// mapping does not exist inside the namespace. Strip to basename and
	// rely on the sandbox PATH declared by the vrooli-aware profile.
	return path.Base(command), nil
}

// translateHostPathToNamespace rewrites a value that may contain the
// host-side merged path so that it points at the agent-visible workspace
// path reported by workspace-sandbox.
//
// Rules:
//   - empty in → empty out (caller decides default).
//   - exact match against hostMerged → workspacePath.
//   - path with hostMerged as a clean directory prefix (e.g. a subpath of
//     the merged dir) → workspacePath + remainder.
//   - any other value → returned unchanged. Caller is responsible for
//     deciding whether that value is a contract violation.
//
// hostMerged is the resolved host path of the sandbox merged dir
// (e.g. /home/.../workspace-sandbox/<id>/merged). workspacePath is the
// server-reported agent-visible path. Under an identity layout
// (pathIllusion=false) workspacePath == hostMerged, so the mapping is a
// no-op rewrite. When hostMerged is empty the helper acts as identity.
//
// DOC: see scenarios/agent-manager/docs/internal/SEAMS.md #SandboxLauncher
func translateHostPathToNamespace(value, hostMerged, workspacePath string) string {
	if value == "" {
		return ""
	}
	if hostMerged == "" {
		return value
	}
	if value == hostMerged {
		return workspacePath
	}
	prefix := hostMerged
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	if strings.HasPrefix(value, prefix) {
		return strings.TrimRight(workspacePath, "/") + "/" + strings.TrimPrefix(value, prefix)
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
	if envMap == nil {
		envMap = make(map[string]string, 1)
	}
	// The Vrooli CLI inside a protected sandbox uses this one bootstrap
	// transport to reach workspace-sandbox's narrow host-lifecycle proxy.
	// It is deliberately not a target-scenario endpoint: the proxy executes
	// the host Vrooli CLI, which resolves target ports from the authoritative
	// runtime registry. Supplying it here prevents recursive in-sandbox
	// attempts to discover workspace-sandbox's own dynamic port.
	envMap["VROOLI_HOST_LIFECYCLE_BASE"] = strings.TrimRight(l.provider.baseURL, "/")
	// workspace-sandbox's vrooli-aware profile owns PATH construction.
	// Dropping inherited PATH here prevents an interactive shell's tool
	// order from becoming part of the audited sandbox launch contract.
	delete(envMap, "PATH")
	// The same profile owns HOME (it sets HOME=$HOME against the sandbox's
	// own absolute home). Dropping inherited HOME here stops a relative
	// HOME (e.g. HOME=.home from a sandboxed parent process) from crossing
	// the API boundary; with `--chdir <workspace>` a relative HOME
	// otherwise materializes `<workspace>/.home` from $HOME-relative writes
	// (Go build cache, vrooli CLI metrics, tool config). See
	// exec/config.go for the matching profile-side guard.
	delete(envMap, "HOME")

	// Resolve the negotiated layout once (host merged dir, reported
	// agent-visible workspace path, path-illusion flag, home-overlay state).
	// All host→namespace translation is driven off these server-reported
	// fields — never inferred from GOOS or driver id. A host workdir outside
	// the sandbox is a contract violation, not a fallback case.
	layout, layoutErr := l.resolveNamespaceLayout(ctx)
	if layoutErr != nil {
		return nil, fmt.Errorf("SandboxLauncher: resolve namespace layout: %w", layoutErr)
	}
	hostMerged := layout.HostMerged
	workingDir, wdErr := resolveWorkingDir(req.WorkingDir, hostMerged, layout.WorkspacePath)
	if wdErr != nil {
		return nil, wdErr
	}
	envMap = translateEnvHostPaths(envMap, hostMerged, layout.WorkspacePath)

	// Command + args translation: agent-manager's runners resolve binary
	// paths via host-side exec.LookPath and pass the absolute host path.
	// Under a path-illusion layout those host paths usually don't exist
	// inside the namespace (only the vrooli-aware bind layout is mounted),
	// so the host path needs to be rewritten before crossing the API
	// boundary. Under an identity layout the paths are left intact.
	//
	// All three coding-agent runners use BuildEnvWrappedLaunchRequest,
	// which sets Command="env" and stuffs the binary path into Args[1+].
	// We must therefore translate Args entries too — Command="env" alone
	// would resolve via PATH but bwrap's env shim then fails to exec the
	// host-absolute binary path embedded in args.
	command, err := translateCommandToNamespace(req.Command, layout)
	if err != nil {
		return nil, err
	}
	translatedArgs := make([]string, len(req.Args))
	for i, a := range req.Args {
		translatedArgs[i], err = translateCommandToNamespace(a, layout)
		if err != nil {
			return nil, err
		}
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
		ctx, cancel := context.WithTimeout(context.Background(), config.DefaultLevers().Runners.ProbeTimeout)
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
// uniform across host and sandbox launches. See runner.ExtractExitCode.
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
	// scanner. Sized by Scanner.StdoutMaxLineBytes — a JSON exit frame or
	// chunky log line fits comfortably under the runner-stream ceiling.
	s.Buffer(make([]byte, 0, 64*1024), config.DefaultLevers().Scanner.StdoutMaxLineBytes)
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

// resolveWorkingDir picks the in-namespace workingDir for a launch request,
// keyed off the server-reported agent-visible workspacePath.
//
//   - empty → workspacePath (the reported agent-visible workspace root).
//   - already workspacePath (or a subpath) → returned unchanged.
//   - matches the host merged dir (or a subpath of it) → translated to
//     the corresponding agent-visible path.
//   - any other absolute host path → contract violation; returned as
//     *LaunchBlocked{Code: "workdir_outside_sandbox"} so the runner can
//     surface it on the run timeline rather than fail opaquely later.
//
// Under an identity layout (pathIllusion=false) workspacePath == hostMerged,
// so the two recognized branches collapse to the same host root.
func resolveWorkingDir(requested, hostMerged, workspacePath string) (string, error) {
	if requested == "" {
		return workspacePath, nil
	}
	// Already the reported workspace root or a path under it.
	if workspacePath != "" &&
		(requested == workspacePath ||
			strings.HasPrefix(requested, strings.TrimSuffix(workspacePath, "/")+"/")) {
		return requested, nil
	}
	if hostMerged != "" {
		if requested == hostMerged ||
			strings.HasPrefix(requested, strings.TrimSuffix(hostMerged, "/")+"/") {
			return translateHostPathToNamespace(requested, hostMerged, workspacePath), nil
		}
	}
	return "", &LaunchBlocked{
		Code:    "workdir_outside_sandbox",
		Message: fmt.Sprintf("workingDir %q is outside the sandbox workspace; sandbox-routed launches must run inside %s", requested, workspacePath),
	}
}

// translateEnvHostPaths rewrites any env values that exactly match the
// sandbox's host merged path so that the agent process inside the
// namespace sees the in-namespace path. Other values pass through. The
// VROOLI_SANDBOX_MERGED contract specifically expects this translation
// (the env var name documents semantics; the value must be visible to
// the agent).
func translateEnvHostPaths(env map[string]string, hostMerged, workspacePath string) map[string]string {
	if len(env) == 0 || hostMerged == "" {
		return env
	}
	out := make(map[string]string, len(env))
	for k, v := range env {
		out[k] = translateHostPathToNamespace(v, hostMerged, workspacePath)
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
	_ runner.Launcher                   = (*SandboxLauncher)(nil)
	_ runner.LaunchedProcess            = (*sandboxLaunchedProcess)(nil)
	_ runner.SandboxLauncherFactory     = (*WorkspaceSandboxProvider)(nil)
	_ runner.SandboxContainmentReporter = (*WorkspaceSandboxProvider)(nil)
)
