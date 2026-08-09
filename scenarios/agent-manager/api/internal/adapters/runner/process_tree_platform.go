package runner

import (
	"os/exec"
	"time"

	platform "github.com/vrooli/platform-go"
)

func killProcessTree(cmd *exec.Cmd) {
	_ = signalProcessGroup(cmd, true)
}

func signalProcessTree(cmd *exec.Cmd, grace time.Duration) {
	if err := signalProcessGroup(cmd, false); err != nil {
		return
	}
	go func() {
		time.Sleep(grace)
		_ = signalProcessGroup(cmd, true)
	}()
}

// signalProcessGroup preserves the runner's containment contract after the
// leader exits: descendants can still hold inherited pipes open, so killing
// only the leader PID is insufficient. Detached runner commands use the
// leader PID as their process-group ID on supported Unix hosts; unsupported
// hosts fall back to their native single-process termination primitive.
func signalProcessGroup(cmd *exec.Cmd, force bool) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if err := platform.SignalProcessGroup(cmd.Process.Pid, force); err == nil {
		return nil
	}
	return platform.KillProcess(cmd.Process.Pid, force)
}
