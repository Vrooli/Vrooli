package buildinfo

import (
	"fmt"
	"os"

	platform "github.com/vrooli/platform-go"
)

func acquireRebuildLock(executable string) (func(), error) {
	lockPath := executable + ".lock"
	f, err := openFileFn(lockPath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open lock %s: %w", lockPath, err)
	}
	release, err := platform.LockFile(f, false)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("lock %s: %w", lockPath, err)
	}
	return func() {
		release()
		_ = f.Close()
	}, nil
}
