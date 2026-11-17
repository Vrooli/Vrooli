//go:build !linux

package assets

import "fmt"

func freeBytes(string) (uint64, error) {
	return 0, fmt.Errorf("bounded image setup is supported on Linux only")
}
func filesystemDevice(string) (uint64, error) {
	return 0, fmt.Errorf("bounded image setup is supported on Linux only")
}
func nearestExistingDirectory(string) (string, error) {
	return "", fmt.Errorf("bounded image setup is supported on Linux only")
}
func rejectSymlinkComponents(string) error {
	return fmt.Errorf("bounded image setup is supported on Linux only")
}
