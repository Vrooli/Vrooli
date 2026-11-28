//go:build !linux

package platform

import "os/exec"

// startHeld starts the target directly where no scope placement primitive
// exists (adoption is unsupported off Linux), so there is nothing to hold for.
func startHeld(cmd *exec.Cmd, start func(*exec.Cmd) error) (processHold, error) {
	return nil, start(cmd)
}
