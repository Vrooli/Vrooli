//go:build !unix

package storagehealth

import "errors"

func freeBytes(string) (uint64, error) {
	return 0, errors.New("free disk space is not measurable on this platform")
}

// Unknown devices are treated as shared so the space check stays conservative.
func sameDevice(string, string) bool { return true }
