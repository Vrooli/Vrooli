package runner

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	platform "github.com/vrooli/platform-go"
)

// HostLauncher launches processes directly on the host via os/exec.
//
// It is the legacy default launcher used when SandboxConfig.Mode is not
// Protected. It preserves the exact behavior of the pre-Launcher code
// paths (process group, idle timeout, grandchild cleanup via the
// existing managedProcess machinery in idle_scanner.go).
type HostLauncher struct{}

// NewHostLauncher returns a HostLauncher.
func NewHostLauncher() *HostLauncher { return &HostLauncher{} }

// Launch starts a process via exec.CommandContext.
//
// Lifecycle:
//   - cmd is started with Setpgid so the runner can kill the entire
//     process tree.
//   - Stdin (if provided) is copied in a background goroutine and the
//     pipe is closed when the source is exhausted; otherwise the pipe is
//     closed immediately so the process doesn't block on input.
//   - When ctx is cancelled, exec.CommandContext kills the leader; the
//     managed-process goroutine then kills the rest of the group.
func (l *HostLauncher) Launch(ctx context.Context, req LaunchRequest) (LaunchedProcess, error) {
	if req.Command == "" {
		return nil, errors.New("launcher: command is required")
	}
	cmd := exec.CommandContext(ctx, req.Command, req.Args...)
	cmd.Dir = req.WorkingDir
	cmd.Env = mergeRequestedEnv(scrubInheritedIdentity(os.Environ()), req.Env)
	if err := platform.ConfigureCommand(cmd, platform.ProcessOptions{Detached: true}); err != nil {
		return nil, err
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	mp, err := startManagedProcess(cmd, req.IdleTimeout)
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}

	if req.Stdin != nil {
		go func() {
			defer stdin.Close()
			_, _ = io.Copy(stdin, req.Stdin)
		}()
	} else {
		_ = stdin.Close()
	}

	return &hostLaunchedProcess{cmd: cmd, mp: mp, stderr: mp.Stderr()}, nil
}

func scrubInheritedIdentity(env []string) []string {
	if env == nil {
		env = os.Environ()
	}
	result := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, "VROOLI_AGENT_IDENTITY_TOKEN=") {
			continue
		}
		result = append(result, entry)
	}
	return result
}

// mergeRequestedEnv removes inherited entries that would otherwise leak a
// parent run token, then overlays the explicitly assembled child environment.
// This is the critical distinction: a freshly minted child token supplied in
// req.Env must survive, while the process-wide inherited token must not.
func mergeRequestedEnv(base, requested []string) []string {
	result := make([]string, 0, len(base)+len(requested))
	index := make(map[string]int, len(base)+len(requested))
	for _, entry := range base {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			continue
		}
		index[key] = len(result)
		result = append(result, entry)
	}
	for _, entry := range requested {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			continue
		}
		if i, exists := index[key]; exists {
			result[i] = entry
			continue
		}
		index[key] = len(result)
		result = append(result, entry)
	}
	return result
}

// hostLaunchedProcess implements LaunchedProcess for HostLauncher.
type hostLaunchedProcess struct {
	cmd          *exec.Cmd
	mp           *managedProcess
	stderr       io.ReadCloser
	signalOnce   sync.Once
	signalActive atomic.Bool
}

func (h *hostLaunchedProcess) Stdout() io.Reader { return h.mp.Stdout() }
func (h *hostLaunchedProcess) Stderr() io.Reader { return h.stderr }
func (h *hostLaunchedProcess) ResetIdleTimer()   { h.mp.ResetTimer() }
func (h *hostLaunchedProcess) TimedOut() bool    { return h.mp.TimedOut() }
func (h *hostLaunchedProcess) Kill()             { h.mp.Kill() }
func (h *hostLaunchedProcess) Wait() error       { return h.mp.Wait() }

func (h *hostLaunchedProcess) PID() int {
	if h.cmd.Process == nil {
		return 0
	}
	return h.cmd.Process.Pid
}

// Signal sends SIGTERM to the process group; if the process hasn't exited
// after the grace period, escalates to SIGKILL. Idempotent.
func (h *hostLaunchedProcess) Signal(grace time.Duration) {
	h.signalOnce.Do(func() {
		if h.cmd.Process == nil {
			return
		}
		h.signalActive.Store(true)
		signalProcessTree(h.cmd, grace)
	})
}
