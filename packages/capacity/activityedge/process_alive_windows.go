//go:build windows

package activityedge

import platform "github.com/vrooli/platform-go"

func processAlive(pid int) bool {
	alive, err := platform.ProcessAliveObserved(pid)
	return err == nil && alive
}
