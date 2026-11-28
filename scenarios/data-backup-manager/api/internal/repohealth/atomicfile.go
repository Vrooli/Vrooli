package repohealth

import (
	"os"
	"path/filepath"
)

// writePrivateFileAtomically publishes sensitive JSON state with one replace:
// the temporary stays beside the destination so rename is same-filesystem.
func writePrivateFileAtomically(destination, pattern string, data []byte) error {
	dir := filepath.Dir(destination)
	tmp, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, destination)
}
