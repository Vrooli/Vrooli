package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
)

func TestAcquireScenarioLockBlocksSecondCallerSameProcess(t *testing.T) {
	home := t.TempDir()
	r := &Runner{Home: home}

	release1, err := r.acquireScenarioLock("web-console")
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	acquired := make(chan struct{})
	var release2 func()
	go func() {
		var err error
		release2, err = r.acquireScenarioLock("web-console")
		if err != nil {
			t.Errorf("second acquire returned error: %v", err)
		}
		close(acquired)
	}()

	select {
	case <-acquired:
		t.Fatal("second acquireScenarioLock returned before first release; expected to block")
	default:
	}

	release1()
	<-acquired
	if release2 != nil {
		release2()
	}
}

func TestAcquireScenarioLockReturnsErrBusyAcrossSimulatedProcesses(t *testing.T) {
	// Simulate the cross-process case: the lock file already exists and
	// is held by another process (pid 99999). When this process tries to
	// acquire, the kernel would return EWOULDBLOCK — we inject that
	// directly so the test does not need a real second OS process.
	home := t.TempDir()
	r := &Runner{Home: home}

	// Pre-seed the lock file with a foreign pid so the contention error
	// can quote it back to the user.
	lockDir := filepath.Join(home, scenarioLockDirName)
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(lockDir, "scenario-web-console.lock"), []byte("99999\n"), 0o644); err != nil {
		t.Fatalf("seed lock file: %v", err)
	}

	origFlock := lockFileFlockFn
	defer func() { lockFileFlockFn = origFlock }()
	lockFileFlockFn = func(fd int, how int) error {
		if how == syscall.LOCK_UN {
			return nil
		}
		return syscall.EWOULDBLOCK
	}

	_, err := r.acquireScenarioLock("web-console")
	if err == nil {
		t.Fatalf("acquire: expected error, got nil")
	}
	if !errors.Is(err, ErrScenarioBusy) {
		t.Fatalf("expected ErrScenarioBusy, got %v", err)
	}
	if msg := err.Error(); !contains(msg, "pid 99999") {
		t.Fatalf("error message should include holder pid 99999, got %q", msg)
	}
}

func TestAcquireScenarioLockDifferentScenariosDoNotBlock(t *testing.T) {
	home := t.TempDir()
	r := &Runner{Home: home}

	releaseA, err := r.acquireScenarioLock("scenario-a")
	if err != nil {
		t.Fatalf("acquire a: %v", err)
	}
	defer releaseA()

	releaseB, err := r.acquireScenarioLock("scenario-b")
	if err != nil {
		t.Fatalf("acquire b should not block on different scenario: %v", err)
	}
	releaseB()
}

func TestAcquireScenarioLockRejectsEmptyName(t *testing.T) {
	r := &Runner{Home: t.TempDir()}
	if _, err := r.acquireScenarioLock(""); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := r.acquireScenarioLock("   "); err == nil {
		t.Fatal("expected error for whitespace-only name")
	}
}

func TestAcquireScenarioLockRejectsEmptyHome(t *testing.T) {
	r := &Runner{Home: ""}
	if _, err := r.acquireScenarioLock("web-console"); err == nil {
		t.Fatal("expected error for empty Home")
	}
}

func TestAcquireScenarioLockWritesAndReadsHolderPID(t *testing.T) {
	home := t.TempDir()
	r := &Runner{Home: home}

	release, err := r.acquireScenarioLock("ports-foo")
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer release()

	contents, err := os.ReadFile(filepath.Join(home, scenarioLockDirName, "scenario-ports-foo.lock"))
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	got := string(contents)
	want := pidString(os.Getpid()) + "\n"
	if got != want {
		t.Fatalf("lock file contents = %q, want %q", got, want)
	}
}

func TestSanitizeScenarioNameStripsPathTraversal(t *testing.T) {
	got := sanitizeScenarioName("../../etc/passwd")
	for _, r := range got {
		if r == '/' || r == '.' {
			// dot is allowed, slash is not
			if r == '/' {
				t.Fatalf("sanitized name contains slash: %q", got)
			}
		}
	}
}

func TestReleaseIsIdempotent(t *testing.T) {
	home := t.TempDir()
	r := &Runner{Home: home}

	release, err := r.acquireScenarioLock("idem")
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	release()
	release() // second call must not panic or unlock-an-unlocked-mutex
	// And we can re-acquire after release.
	release2, err := r.acquireScenarioLock("idem")
	if err != nil {
		t.Fatalf("re-acquire: %v", err)
	}
	release2()
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func pidString(pid int) string {
	// Avoid importing strconv here just for symmetry — keep the test
	// dependency surface minimal. fmt would also work; either way the
	// implementation is trivial.
	if pid == 0 {
		return "0"
	}
	neg := pid < 0
	if neg {
		pid = -pid
	}
	var buf [20]byte
	i := len(buf)
	for pid > 0 {
		i--
		buf[i] = byte('0' + pid%10)
		pid /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// Sanity check that we link to syscall and sync so unused-import warnings
// don't trip on the conditional logic above when test files are restructured.
var _ = sync.Mutex{}
