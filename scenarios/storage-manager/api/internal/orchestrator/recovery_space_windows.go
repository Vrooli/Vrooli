//go:build windows

package orchestrator

import "fmt"

func measureRecoveryFreeBytes(string) (int64, error) {
	return 0, fmt.Errorf("statfs recovery measurement is not implemented on windows")
}

func measureRecoverySpace(string) (float64, int64, error) {
	return 0, 0, fmt.Errorf("statfs recovery measurement is not implemented on windows")
}
