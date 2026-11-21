//go:build windows

package hostinventory

import "errors"

func platformStorageMounts() ([]storageMount, error) {
	// A Windows implementation must use the volume manager rather than guess
	// drive letters. Keep the unsupported result explicit until that adapter is
	// available; callers surface remediation instead of selecting C:\\ or a
	// temporary directory.
	return nil, errors.New("Windows volume enumeration is unavailable on this build")
}

func physicalDeviceIdentity(string) (string, bool)      { return "", false }
func resolvePathForStorage(path string) (string, error) { return path, nil }
func probeWritableDirectory(string) error {
	return errors.New("Windows writable probe is unavailable on this build")
}
