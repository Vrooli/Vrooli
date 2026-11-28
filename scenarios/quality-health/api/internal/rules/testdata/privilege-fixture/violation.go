package fixture

import "os/exec"

func restart() error {
	return exec.Command("sudo", "systemctl", "restart", "foo").Run()
}
