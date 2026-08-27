package lifecycle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	platform "github.com/vrooli/platform-go"
)

var sharedPackageLockPollInterval = tuning.FastHealthPollInterval()

var inProcessSharedPackageLocks sync.Map // map[home\x00canonical-root]*sync.Mutex

// acquireSharedPackageLock serializes all lifecycle mutations of one governed
// package, regardless of which scenario requested them. Scenario locks alone
// cannot provide this guarantee because multiple UIs may consume the same
// file: dependency concurrently.
//
// The lock waits rather than failing, because package installation/building is
// an expected multi-minute operation. The first wait emits the lock owner and
// package root so the wait is diagnosable instead of looking like a hang.
func acquireSharedPackageLock(home, packageName, packageRoot string, logWriter io.Writer) (func(), error) {
	// Legacy callers predate lifecycle cancellation. Their bounded polling uses
	// the process lifetime as the compatibility context.
	return acquireSharedPackageLockContext(context.Background(), home, packageName, packageRoot, logWriter)
}

func acquireSharedPackageLockContext(ctx context.Context, home, packageName, packageRoot string, logWriter io.Writer) (func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(home) == "" {
		return nil, fmt.Errorf("acquire shared package lock for %q: home is required", packageName)
	}
	canonicalRoot, err := filepath.Abs(filepath.Clean(packageRoot))
	if err != nil {
		return nil, fmt.Errorf("canonicalize shared package %q root: %w", packageName, err)
	}
	if resolvedRoot, resolveErr := filepath.EvalSymlinks(canonicalRoot); resolveErr == nil {
		canonicalRoot = resolvedRoot
	}
	key := home + "\x00" + canonicalRoot
	mu := sharedPackageMutex(key)
	waitStarted := time.Now()
	waitLogged := false
	muLocked := false
	lockErr := error(nil)
	if err := AwaitContext(ctx, AwaitClock{Now: time.Now, Sleep: time.Sleep}, AwaitPolicy{Timeout: tuning.DailyRetentionWindow(), Interval: sharedPackageLockPollInterval}, func() (bool, error) {
		if mu.TryLock() {
			muLocked = true
			return true, nil
		}
		if !waitLogged {
			writeSharedPackageLockEvent(logWriter, "waiting", packageName, canonicalRoot, 0, time.Since(waitStarted))
			waitLogged = true
		}
		return false, nil
	}); err != nil {
		return nil, err
	}
	if !muLocked {
		return nil, fmt.Errorf("shared package lock wait ended without ownership")
	}

	lockDir := filepath.Join(home, scenarioLockDirName)
	if err := os.MkdirAll(lockDir, tuning.PermDir); err != nil {
		mu.Unlock()
		return nil, fmt.Errorf("create shared package lock dir: %w", err)
	}
	lockPath := sharedPackageLockPath(lockDir, canonicalRoot)
	file, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, tuning.PermFile)
	if err != nil {
		mu.Unlock()
		return nil, fmt.Errorf("open shared package lock %s: %w", lockPath, err)
	}

	var release func()
	if err := AwaitContext(ctx, AwaitClock{Now: time.Now, Sleep: time.Sleep}, AwaitPolicy{Timeout: tuning.DailyRetentionWindow(), Interval: sharedPackageLockPollInterval}, func() (bool, error) {
		releaseFile, err := lockFileFn(file, true)
		if err == nil {
			if _, err := file.Seek(0, 0); err == nil {
				_ = file.Truncate(0)
				_, _ = file.WriteString(fmt.Sprintf("%d\n", os.Getpid()))
				_ = file.Sync()
			}
			if waitLogged {
				writeSharedPackageLockEvent(logWriter, "acquired", packageName, canonicalRoot, 0, time.Since(waitStarted))
			}
			var released bool
			release = func() {
				if released {
					return
				}
				released = true
				releaseFile()
				_ = file.Close()
				mu.Unlock()
			}
			return true, nil
		}

		if !errors.Is(err, platform.ErrLockUnavailable) {
			lockErr = err
			return false, err
		}

		if !waitLogged {
			holder := readLockHolderPID(file)
			writeSharedPackageLockEvent(logWriter, "waiting", packageName, canonicalRoot, holder, time.Since(waitStarted))
			waitLogged = true
		}
		return false, nil
	}); err != nil {
		_ = file.Close()
		mu.Unlock()
		if lockErr != nil {
			return nil, fmt.Errorf("acquire shared package lock %s: %w", lockPath, lockErr)
		}
		return nil, err
	}
	return release, nil
}

func sharedPackageMutex(key string) *sync.Mutex {
	if existing, ok := inProcessSharedPackageLocks.Load(key); ok {
		return existing.(*sync.Mutex)
	}
	created := &sync.Mutex{}
	actual, _ := inProcessSharedPackageLocks.LoadOrStore(key, created)
	return actual.(*sync.Mutex)
}

func sharedPackageLockPath(lockDir, packageRoot string) string {
	digest := sha256.Sum256([]byte(packageRoot))
	return filepath.Join(lockDir, "shared-package-"+hex.EncodeToString(digest[:8])+".lock")
}

func writeSharedPackageLockEvent(writer io.Writer, event, packageName, packageRoot string, holderPID int, waited time.Duration) {
	if writer == nil {
		return
	}
	_, _ = fmt.Fprintf(writer,
		"shared-package-lock event=%s package=%q root=%q holder_pid=%d wait_ms=%d pid=%d\n",
		event, packageName, packageRoot, holderPID, waited.Milliseconds(), os.Getpid())
}
