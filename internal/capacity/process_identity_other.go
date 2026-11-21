//go:build !linux

package capacity

import "fmt"

func processStartIdentity(pid int) (string, error) {
	return "", fmt.Errorf("process-bound disk admission is not qualified for pid %d on this platform", pid)
}

func processIdentityMatches(pid int, expectedStart string) bool { return false }
