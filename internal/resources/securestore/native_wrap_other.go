//go:build !darwin && !windows

package securestore

import "fmt"

func nativeWrapAvailable() (string, error) {
	return "", fmt.Errorf("%w: this platform has no native unattended key wrap", errKeyProviderUnavailable)
}

func nativeWrapProtect([]byte) ([]byte, error) {
	return nil, fmt.Errorf("%w: native key wrap is unavailable", errKeyProviderUnavailable)
}
func nativeWrapUnprotect([]byte) ([]byte, error) {
	return nil, fmt.Errorf("%w: native key wrap is unavailable", errKeyProviderUnavailable)
}
