//go:build !linux

package capacity

// diskJobFilesystemIdentity uses the same native physical-filesystem identity
// as admission. The existing adapter refuses platforms without qualified disk
// admission rather than manufacturing an identity or reading Linux mount data.
func diskJobFilesystemIdentity(path string) (string, string, error) {
	_, filesystem, _, probe, err := diskPathIdentity(path)
	return filesystem, probe, err
}
