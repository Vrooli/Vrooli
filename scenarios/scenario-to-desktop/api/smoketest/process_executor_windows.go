//go:build windows

package smoketest

import (
	"os/exec"
	"strconv"
	"time"
)

func configureProcessGroup(_ *exec.Cmd) {}

func terminateProcessTree(cmd *exec.Cmd) {
	killCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid))
	if err := killCmd.Run(); err != nil {
		_ = cmd.Process.Kill()
	}
}

func cleanupProcessGroupAfterLeaderExit(_ int, _ time.Duration) {}
