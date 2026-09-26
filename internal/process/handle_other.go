//go:build !linux

package process

import "fmt"

func openHandle(pid int) (Handle, error) {
	return nil, fmt.Errorf("stable process handles are unavailable on this platform (pid %d)", pid)
}
