//go:build linux

package platform

import "os/exec"

func serviceManagerCommand() string { return "systemctl" }

func serviceManagerCommandPath() string {
	if path, err := exec.LookPath("systemctl"); err == nil {
		return path
	}
	return "/usr/bin/systemctl"
}
