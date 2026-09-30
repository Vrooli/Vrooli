//go:build unix && !linux && !darwin

package platform

import (
	"errors"
	"fmt"
	"syscall"
)

func processWorkingDir(int) (string, error) { return "", ErrUnsupported }

func processHasChildren(int) (bool, error) { return false, ErrUnsupported }

func processName(int) (string, error) { return "", ErrUnsupported }

func processExecutablePath(int) (string, error) { return "", ErrUnsupported }

func pidIsAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if err := syscall.Kill(pid, 0); err != nil {
		return errors.Is(err, syscall.EPERM)
	}
	return true
}

func readProcessEnvironment(int) (map[string]string, error) {
	return nil, fmt.Errorf("platform: process environment inspection is not supported on this platform: %w", ErrUnsupported)
}
