package runner

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
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
	cmd.Env = req.Env
	cmd.SysProcAttr = newProcessAttributes()

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
