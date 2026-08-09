package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	platform "github.com/vrooli/platform-go"
)

// scenarioLockDirName is the subdirectory under Home where per-scenario
// advisory lock files live. Runner.Home points at the user's home
// directory ($HOME), and the canonical vrooli state tree lives under
// "$HOME/.vrooli/state" — the lock files live alongside runtime.db so
// every vrooli invocation, regardless of cwd or scenario context,
// resolves the same path. Lock files are not deleted: flock(2) releases
// the lock when the holding fd closes, and leaving inodes in place keeps
// identity stable across acquisitions.
const scenarioLockDirName = ".vrooli/state/locks"

// ErrScenarioBusy is returned when another vrooli process already holds
// the lifecycle lock for the requested scenario. Surfaces to users as
// "another invocation is already running for this scenario."
var ErrScenarioBusy = errors.New("scenario is already being started, stopped, or restarted")

// lockFileFn is the platform seam used to take the advisory lock. Tests
// override it to simulate contention without taking real kernel locks.
var lockFileFn = func(file *os.File, nonBlocking bool) (func(), error) {
	return platform.LockFile(file, nonBlocking)
}

// lockFileOpenFn opens the lock file; tests override to inject failures.
var lockFileOpenFn = func(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

// inProcessScenarioLocks holds one sync.Mutex per (home, scenario) tuple.
// flock(2) is per-fd, so two goroutines in the same process opening
// separate fds against the same lock file would both succeed at the
// kernel level — the in-process mutex provides the single-flight
// guarantee inside one process. The kernel flock provides it across
// processes.
var inProcessScenarioLocks sync.Map // map[string]*sync.Mutex

func lockKey(home, name string) string {
	return home + "\x00" + name
}

func scenarioMutex(key string) *sync.Mutex {
	if existing, ok := inProcessScenarioLocks.Load(key); ok {
		return existing.(*sync.Mutex)
	}
	created := &sync.Mutex{}
	actual, _ := inProcessScenarioLocks.LoadOrStore(key, created)
	return actual.(*sync.Mutex)
}

// acquireScenarioLock takes the single-flight lock for the named scenario.
// It blocks at the in-process layer (so two goroutines in the same process
// serialize) and uses flock(2) LOCK_EX|LOCK_NB at the file layer (so two
// different OS processes get a fast-fail with ErrScenarioBusy rather than
// blocking).
//
// The returned release closure must be invoked exactly once when the
// caller is finished. Deferring it is the expected pattern.
func (r *Runner) acquireScenarioLock(name string) (func(), error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("acquire scenario lock: name is required")
	}
	home := r.Home
	if home == "" {
		return nil, fmt.Errorf("acquire scenario lock for %q: runner Home is empty", name)
	}
	key := lockKey(home, name)
	mu := scenarioMutex(key)
	mu.Lock()

	lockDir := filepath.Join(home, scenarioLockDirName)
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		mu.Unlock()
		return nil, fmt.Errorf("create scenario lock dir: %w", err)
	}
	lockPath := filepath.Join(lockDir, "scenario-"+sanitizeScenarioName(name)+".lock")

	f, err := lockFileOpenFn(lockPath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		mu.Unlock()
		return nil, fmt.Errorf("open scenario lock %s: %w", lockPath, err)
	}
	releaseFile, err := lockFileFn(f, true)
	if err != nil {
		holder := readLockHolderPID(f)
		_ = f.Close()
		mu.Unlock()
		if errors.Is(err, platform.ErrLockUnavailable) {
			if holder > 0 {
				return nil, fmt.Errorf("%w: %q (held by pid %d)", ErrScenarioBusy, name, holder)
			}
			return nil, fmt.Errorf("%w: %q", ErrScenarioBusy, name)
		}
		return nil, fmt.Errorf("acquire scenario lock %s: %w", lockPath, err)
	}

	// Record our pid in the file so contending callers can report it.
	if _, err := f.Seek(0, 0); err == nil {
		_ = f.Truncate(0)
		_, _ = f.WriteString(strconv.Itoa(os.Getpid()) + "\n")
		_ = f.Sync()
	}

	var released bool
	return func() {
		if released {
			return
		}
		released = true
		releaseFile()
		_ = f.Close()
		mu.Unlock()
	}, nil
}

func readLockHolderPID(f *os.File) int {
	if _, err := f.Seek(0, 0); err != nil {
		return 0
	}
	buf := make([]byte, 64)
	n, _ := f.Read(buf)
	if n <= 0 {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(buf[:n])))
	if err != nil {
		return 0
	}
	return pid
}

// sanitizeScenarioName replaces filesystem-unsafe characters in scenario
// slugs so the lock filename is well-formed regardless of the input.
func sanitizeScenarioName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}
