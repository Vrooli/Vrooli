package platform

import (
	"errors"
	"os/exec"
	"time"
)

// HeldProcess is a started process that has not yet executed its target
// program. Placement that must cover the whole process tree (a services-slice
// scope, a ceiling) happens while the process is held: it cannot fork, so
// every descendant the target later creates is born inside that placement.
//
// Moving a pid after cmd.Start is not equivalent. The kernel migrates one
// thread group, never its children, and launchers fork or re-exec at once:
// rootlesskit re-executes /proc/self/exe before its parent can be adopted, so
// on 2026-11-22 a restart run from a coding-agent shell left `ollama serve`
// and its runners in the agent's scope while only the launcher moved.
type HeldProcess struct {
	Cmd *exec.Cmd
	// hold is the native gate; nil where the platform has no hold primitive
	// and the target is already running.
	hold processHold
}

type processHold interface {
	release(cmd *exec.Cmd, budget time.Duration) error
	abort()
}

// ErrHoldUnavailable means the command cannot be held before its target
// executes on this host. No process was started.
var ErrHoldUnavailable = errors.New("platform: cannot hold the process before its target executes")

// HeldReleaseBudget bounds how long Release waits for the target to replace
// the hold after the gate opens.
var HeldReleaseBudget = 5 * time.Second

// StartHeld starts cmd with its process held before the target program
// executes. start performs the spawn (nil means cmd.Start); it receives the
// rewritten command. The caller places the process (cmd.Process.Pid), then
// calls Release so the target executes, or Abort so it never does.
//
// On Linux the hold is a POSIX shell gate that blocks on a private pipe and
// then execs the target in the same process: the pid, environment, working
// directory, process attributes, resource limits and open files are the
// target's own. argv[0] becomes cmd.Path (the resolved executable, which keeps
// argv[0]'s base name). Platforms without a placement primitive start the
// target directly; Release and Abort are then no-ops.
func StartHeld(cmd *exec.Cmd, start func(*exec.Cmd) error) (*HeldProcess, error) {
	if cmd == nil {
		return nil, errors.New("platform: nil command")
	}
	if cmd.Process != nil {
		return nil, errors.New("platform: command already started")
	}
	if cmd.Err != nil {
		return nil, cmd.Err
	}
	if start == nil {
		start = func(c *exec.Cmd) error { return c.Start() }
	}
	hold, err := startHeld(cmd, start)
	if err != nil {
		return nil, err
	}
	return &HeldProcess{Cmd: cmd, hold: hold}, nil
}

// Pid is the held process id; it stays the target's pid after Release.
func (h *HeldProcess) Pid() int { return h.Cmd.Process.Pid }

// Release lets the target execute and waits until it has replaced the hold,
// or the process has exited, so identity reads after Release observe the
// target rather than the gate. A timeout leaves the process to the caller.
func (h *HeldProcess) Release() error {
	if h == nil || h.hold == nil {
		return nil
	}
	hold := h.hold
	h.hold = nil
	return hold.release(h.Cmd, HeldReleaseBudget)
}

// Abort closes the hold without executing the target: the process exits on
// its own having run nothing but the gate. The caller still owns Wait (and
// may Kill first to reap without delay).
func (h *HeldProcess) Abort() {
	if h == nil || h.hold == nil {
		return
	}
	h.hold.abort()
	h.hold = nil
}
