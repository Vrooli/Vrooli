package cliinstall

import (
	"github.com/vrooli/vrooli/internal/config"
	"os"
)

// syncPreparedPath persists file contents or directory entries before a
// prepared bundle can be referenced by durable recovery progress.
func syncPreparedPath(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	if info.IsDir() {
		if err := f.Close(); err != nil {
			return err
		}
		return config.SyncOwnedDirectoryChain(path)
	}
	err = f.Sync()
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
