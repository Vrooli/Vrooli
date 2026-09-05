//go:build windows

package setup

import "os"

func processIdentityAlive(pid int, host string) bool {
	localHost, err := os.Hostname()
	if err != nil || host == "" || host != localHost || pid <= 0 {
		return false
	}
	_, err = os.FindProcess(pid)
	return err == nil
}
