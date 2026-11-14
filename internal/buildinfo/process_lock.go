package buildinfo

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/vrooli/vrooli/internal/tuning"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
)

// PreserveRootBinaryFallback keeps the prior canonical root executable
// in a stable location outside ~/.vrooli/bin. This is a byte snapshot, not
// evidence that the prior image activated or passed health checks. The install
// path is atomic, but a foreign remover, a repair tool, or an interrupted
// first install can still make the canonical path unavailable. The POSIX
// launcher uses this copy only while the canonical executable is absent.
//
// Callers decide whether failed retention prevents their installation.
func PreserveRootBinaryFallback(executable string) error {
	base := filepath.Base(filepath.Clean(executable))
	switch base {
	case "vrooli", "vrooli.exe", "vrooli-api", "vrooli-api.exe":
	default:
		return nil
	}

	info, err := os.Lstat(executable)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 512<<20 {
		return fmt.Errorf("root executable %s is not a bounded nonempty regular file", executable)
	}

	fallbackDir := filepath.Join(filepath.Dir(filepath.Dir(executable)), "libexec")
	if _, err := config.EnsureOwnedDir(fallbackDir); err != nil {
		return err
	}
	fallback := filepath.Join(fallbackDir, base+".previous")
	in, err := os.Open(executable)
	if err != nil {
		return err
	}
	defer in.Close()
	opened, err := in.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return fmt.Errorf("prior root executable changed while opening")
	}
	return config.WriteOwnedFileDurableStream(fallback, info.Mode().Perm(), func(w io.Writer) error {
		n, err := io.Copy(w, io.LimitReader(in, (512<<20)+1))
		if err != nil {
			return err
		}
		if n != info.Size() {
			return fmt.Errorf("prior root executable changed during retention")
		}
		return nil
	})
}

// AcquireBinaryInstallLock serializes every replacement of one installed
// executable. The project installer and the stale self-rebuilder must share
// this lock or they can replace the same path concurrently.
func AcquireBinaryInstallLock(executable string) (func(), error) {
	return AcquireBinaryInstallLockContext(context.Background(), executable)
}

// AcquireBinaryInstallLockContext lets a bounded convergence caller leave a
// competing install wait without a lock goroutine or reopened descriptor.
func AcquireBinaryInstallLockContext(ctx context.Context, executable string) (func(), error) {
	if ctx == nil {
		return nil, fmt.Errorf("binary install context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	lockPath := executable + ".lock"
	f, err := platform.OpenLockFile(lockPath, tuning.PermFile)
	if err != nil {
		return nil, fmt.Errorf("open lock %s: %w", lockPath, err)
	}
	if err := f.Chmod(tuning.PermFile); err != nil {
		_ = f.Close()
		return nil, err
	}
	if err := config.ChownFileToInvokingUser(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	release, err := platform.LockFileContext(ctx, f)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("lock %s: %w", lockPath, err)
	}
	if err := platform.VerifyLockFile(f, lockPath); err != nil {
		release()
		_ = f.Close()
		return nil, err
	}
	var once sync.Once
	return func() {
		once.Do(func() { release(); _ = f.Close() })
	}, nil
}

func acquireRebuildLock(executable string) (func(), error) {
	return AcquireBinaryInstallLock(executable)
}
