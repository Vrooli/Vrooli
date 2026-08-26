package buildinfo

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	platform "github.com/vrooli/platform-go"
)

// PreserveRootBinaryFallback keeps the last known-good root control-plane
// executable in a stable location outside ~/.vrooli/bin. The normal install
// path is atomic, but a foreign remover, a repair tool, or an interrupted
// first install can still make the canonical path unavailable. The POSIX
// launcher uses this copy only while the canonical executable is absent.
//
// This is best-effort by design: failure to refresh a recovery copy must not
// turn an otherwise successful binary install into a failed install.
func PreserveRootBinaryFallback(executable string) error {
	base := filepath.Base(filepath.Clean(executable))
	switch base {
	case "vrooli", "vrooli.exe", "vrooli-api", "vrooli-api.exe":
	default:
		return nil
	}

	info, err := os.Stat(executable)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("root executable %s is not a regular file", executable)
	}

	fallbackDir := filepath.Join(filepath.Dir(filepath.Dir(executable)), "libexec")
	if err := os.MkdirAll(fallbackDir, 0o755); err != nil {
		return err
	}
	fallback := filepath.Join(fallbackDir, base+".previous")
	tmp, err := os.CreateTemp(fallbackDir, "."+base+".previous-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	_ = tmp.Chmod(info.Mode().Perm())

	// Copy the known-good bytes before replacing the fallback name. Keeping
	// this as a copy instead of a hard link also works across filesystems and
	// avoids retaining an inode whose ownership may be surprising to cleanup
	// tooling.
	in, openErr := os.Open(executable)
	if openErr != nil {
		_ = tmp.Close()
		return openErr
	}
	_, copyErr := io.Copy(tmp, in)
	closeInErr := in.Close()
	if copyErr == nil {
		copyErr = closeInErr
	}
	if copyErr != nil {
		_ = tmp.Close()
		return copyErr
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, fallback); err != nil {
		return err
	}
	return nil
}

// AcquireBinaryInstallLock serializes every replacement of one installed
// executable. The project installer and the stale self-rebuilder must share
// this lock or they can replace the same path concurrently.
func AcquireBinaryInstallLock(executable string) (func(), error) {
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

func acquireRebuildLock(executable string) (func(), error) {
	return AcquireBinaryInstallLock(executable)
}
