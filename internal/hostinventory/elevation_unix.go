//go:build !windows

package hostinventory

import (
	"os"
	"os/exec"
)

func currentElevation() ElevationCapability {
	if os.Geteuid() == 0 {
		return ElevationCapability{Elevated: true, CanElevate: true, Mechanism: "root"}
	}
	if _, err := exec.LookPath("sudo"); err == nil {
		return ElevationCapability{CanElevate: true, Mechanism: "sudo"}
	}
	return ElevationCapability{Mechanism: "none"}
}
