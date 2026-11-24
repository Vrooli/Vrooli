package resources

import (
	"os"
	"path/filepath"
)

// writeFileAtomically stages bytes in a unique sibling before replacing path.
// A fixed temporary name can be a stale symlink, causing WriteFile to mutate
// its target instead of staging the new value.
func writeFileAtomically(path string, data []byte, mode os.FileMode) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return replacePath(tmpPath, path)
}
