//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package main

import (
	"os"
	"os/exec"
	"syscall"
)

// applyProcessGroupContainment puts the command in its own process group and
// SIGKILLs that whole group when ctx is cancelled.
//
// Setpgid makes bash the leader of a new process group whose pgid equals its
// pid, so every descendant shares that group unless it deliberately escapes.
// Signalling the negative pid signals the group rather than the leader alone.
func applyProcessGroupContainment(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
			return err
		}
		return os.ErrProcessDone
	}
}
