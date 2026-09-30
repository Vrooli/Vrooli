//go:build darwin && !cgo

package securestore

import "fmt"

func nativeWrapAvailable() (string, error) {
	return "", fmt.Errorf("%w: macOS Keychain requires a cgo-enabled build", errKeyProviderUnavailable)
}

func nativeWrapDiagnosis() string { return "unavailable: macOS Keychain requires a cgo-enabled build" }

func nativeWrapProtect([]byte) ([]byte, error) {
	return nil, fmt.Errorf("%w: macOS Keychain is unavailable", errKeyProviderUnavailable)
}

func nativeWrapUnprotect([]byte) ([]byte, error) {
	return nil, fmt.Errorf("%w: macOS Keychain is unavailable", errKeyProviderUnavailable)
}
