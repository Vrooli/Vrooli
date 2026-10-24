//go:build windows

package maintenance

import "github.com/vrooli/platform-go"

func killProcess(pid int, force bool) error {
	if pid <= 0 {
		return nil
	}
	return platform.KillProcess(pid, force)
}
