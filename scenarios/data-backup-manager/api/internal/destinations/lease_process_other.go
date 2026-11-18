//go:build !unix

package destinations

import "os/exec"

func configureLeaseProcess(*exec.Cmd) {}

func killLeaseProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}
