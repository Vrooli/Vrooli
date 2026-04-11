//go:build windows

package maintenance

import "os"

func killProcess(pid int, force bool) error {
	if pid <= 0 {
		return nil
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}
