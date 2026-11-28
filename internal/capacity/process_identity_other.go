//go:build !linux

package capacity

import "fmt"

func processStartIdentity(pid int) (string, error) {
	return "", fmt.Errorf("process-bound disk admission is not qualified for pid %d on this platform", pid)
}

func processIdentityMatches(pid int, expectedStart string) bool { return false }

// ProcessStartIdentity is unavailable off Linux; owners fall back to PID-only
// identities, which the liveness sweep treats as unverifiable.
func ProcessStartIdentity(pid int) (string, error) { return processStartIdentity(pid) }

// processGone is never true off Linux: without a qualified identity source a
// sweep must not release anything on the strength of a process probe.
func processGone(pid int, expectedStart string) bool { return false }
