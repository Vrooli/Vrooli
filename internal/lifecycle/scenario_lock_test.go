package lifecycle

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestRunSetupIfStoppedAdmission(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    string
		variant   string
		deferred  bool
		openError bool
		readError bool
	}{
		{name: "no_instance"},
		{name: "stopped_instance", status: scenarioruntime.StatusStopped},
		{name: "start_before_lock", status: scenarioruntime.StatusRunning, deferred: true},
		{name: "starting_instance", status: scenarioruntime.StatusStarting, deferred: true},
		{name: "stopping_instance", status: scenarioruntime.StatusStopping, deferred: true},
		{name: "other_variant", status: scenarioruntime.StatusRunning, variant: "preview", deferred: true},
		{name: "registry_unavailable", openError: true},
		{name: "registry_read_failure", readError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, home := t.TempDir(), t.TempDir()
			writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
				Service: scenario.ServiceMetadata{Name: "alpha"},
				Lifecycle: scenario.Lifecycle{Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
					Name: "setup-marker", Exec: []string{"bash", "-c", "printf setup >> setup.txt"},
				}}}},
			})
			dbPath := filepath.Join(home, "registry.db")
			openStore := func() (*scenarioruntime.SQLiteStore, error) {
				return scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{DBPath: dbPath})
			}
			registryErr := errors.New("fixture registry failure")
			lockAcquired := false
			seeded := false
			registryReads := 0
			runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
				deps.runtimeRegistry = func(context.Context, string) (scenarioRuntimeStore, error) {
					registryReads++
					if !lockAcquired {
						t.Error("registry admission read before acquiring scenario lock")
					}
					if tc.openError {
						return nil, registryErr
					}
					store, err := openStore()
					if err != nil {
						return nil, err
					}
					if tc.readError {
						return &setupAdmissionReadErrorStore{scenarioRuntimeStore: store, err: registryErr}, nil
					}
					return store, nil
				}
			})
			// Inject the competing start's registry commit at lock acquisition.
			// Any pre-lock observation sees no instance. No sleep or live start
			// is needed to reproduce the stale-state interleaving.
			originalLock := lockFileFn
			t.Cleanup(func() { lockFileFn = originalLock })
			lockFileFn = func(file *os.File, nonBlocking bool) (func(), error) {
				if tc.status != "" && !seeded {
					store, err := openStore()
					if err != nil {
						return nil, err
					}
					_, err = store.CreateInstance(context.Background(), scenarioruntime.Instance{
						InstanceID: "competing-start", Scenario: "alpha", Variant: tc.variant, Status: tc.status,
					})
					_ = store.Close()
					if err != nil {
						return nil, err
					}
					seeded = true
				}
				release, err := originalLock(file, nonBlocking)
				if err != nil {
					return nil, err
				}
				lockAcquired = true
				return func() { lockAcquired = false; release() }, nil
			}
			output := &setupLockCheckingWriter{t: t, runner: runner}
			runner.Out = output
			runner.WithVerbosity(VerbosityVerbose)
			result, err := runner.RunSetupIfStopped("alpha", PhaseOptions{ProjectMode: true})
			if tc.openError || tc.readError {
				if !errors.Is(err, registryErr) {
					t.Fatalf("error = %v, want registry failure", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if registryReads != 1 {
				t.Fatalf("registry reads = %d, want 1", registryReads)
			}
			marker, markerErr := os.ReadFile(filepath.Join(root, "scenarios", "alpha", "setup.txt"))
			if tc.deferred || tc.openError || tc.readError {
				if !os.IsNotExist(markerErr) || result.ExecutedSteps != 0 || output.writes != 0 {
					t.Fatalf("setup effects on refused admission: marker=%q err=%v result=%#v writes=%d", marker, markerErr, result, output.writes)
				}
				if tc.deferred && (result.Status != PhaseExecutionRunningDeferred || result.RunID != "") {
					t.Fatalf("deferred result = %#v", result)
				}
			} else if markerErr != nil || string(marker) != "setup" || result.ExecutedSteps != 1 || result.Status != PhaseExecutionCompleted || output.writes == 0 {
				t.Fatalf("stopped setup: marker=%q err=%v result=%#v writes=%d", marker, markerErr, result, output.writes)
			}
			release, err := runner.tryAcquireScenarioLock("alpha")
			if err != nil {
				t.Fatalf("admission leaked scenario lock: %v", err)
			}
			release()
		})
	}
}

type setupAdmissionReadErrorStore struct {
	scenarioRuntimeStore
	err error
}

func (s *setupAdmissionReadErrorStore) ListInstances(context.Context, scenarioruntime.InstanceFilter) ([]scenarioruntime.Instance, error) {
	return nil, s.err
}

type setupLockCheckingWriter struct {
	t      *testing.T
	runner *Runner
	writes int
}

func (w *setupLockCheckingWriter) Write(data []byte) (int, error) {
	w.writes++
	if release, err := w.runner.tryAcquireScenarioLock("alpha"); err == nil {
		release()
		w.t.Error("setup released the scenario lock before phase execution finished")
	} else if !errors.Is(err, ErrScenarioBusy) {
		w.t.Errorf("check setup lock: %v", err)
	}
	return io.Discard.Write(data)
}

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

	origLock := lockFileFn
	defer func() { lockFileFn = origLock }()
	lockFileFn = func(_ *os.File, nonBlocking bool) (func(), error) {
		if !nonBlocking {
			return func() {}, nil
		}
		return nil, platform.ErrLockUnavailable
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

func TestAcquireDependencyScenarioLockReusesReadyDependencyWhileBusy(t *testing.T) {
	home := t.TempDir()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	r := &Runner{
		Home: home,
		deps: lifecycleDeps{
			now: func() time.Time { return now },
			sleep: func(d time.Duration) {
				now = now.Add(d)
			},
		},
	}
	seedScenarioLockFile(t, home, "test-genie", "99999\n")

	origLock := lockFileFn
	defer func() { lockFileFn = origLock }()
	lockFileFn = func(_ *os.File, nonBlocking bool) (func(), error) {
		if !nonBlocking {
			return func() {}, nil
		}
		return nil, platform.ErrLockUnavailable
	}

	checks := 0
	release, reused, err := r.acquireDependencyScenarioLock("test-genie", func() (bool, error) {
		checks++
		return checks >= 2, nil
	})
	if err != nil {
		t.Fatalf("acquireDependencyScenarioLock() error = %v", err)
	}
	if !reused {
		t.Fatalf("reused = false, want true after dependency became ready")
	}
	if release != nil {
		t.Fatalf("release is non-nil, want nil when no lock was acquired")
	}
}

func TestAcquireDependencyScenarioLockRetriesUntilLockAvailable(t *testing.T) {
	home := t.TempDir()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	r := &Runner{
		Home: home,
		deps: lifecycleDeps{
			now: func() time.Time { return now },
			sleep: func(d time.Duration) {
				now = now.Add(d)
			},
		},
	}
	seedScenarioLockFile(t, home, "test-genie", "99999\n")

	origLock := lockFileFn
	defer func() { lockFileFn = origLock }()
	attempts := 0
	lockFileFn = func(_ *os.File, nonBlocking bool) (func(), error) {
		if !nonBlocking {
			return func() {}, nil
		}
		attempts++
		if attempts < 3 {
			return nil, platform.ErrLockUnavailable
		}
		return func() {}, nil
	}

	release, reused, err := r.acquireDependencyScenarioLock("test-genie", func() (bool, error) {
		return false, nil
	})
	if err != nil {
		t.Fatalf("acquireDependencyScenarioLock() error = %v", err)
	}
	if reused {
		t.Fatalf("reused = true, want false when lock was acquired")
	}
	if release == nil {
		t.Fatalf("release = nil, want acquired lock release")
	}
	release()
	if attempts != 3 {
		t.Fatalf("lock attempts = %d, want 3", attempts)
	}
}

func TestAcquireDependencyScenarioLockTimesOutWhenBusy(t *testing.T) {
	home := t.TempDir()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	r := &Runner{
		Home: home,
		deps: lifecycleDeps{
			now: func() time.Time { return now },
			sleep: func(d time.Duration) {
				now = now.Add(3 * time.Minute)
			},
		},
	}
	seedScenarioLockFile(t, home, "test-genie", "99999\n")

	origLock := lockFileFn
	defer func() { lockFileFn = origLock }()
	lockFileFn = func(_ *os.File, nonBlocking bool) (func(), error) {
		if !nonBlocking {
			return func() {}, nil
		}
		return nil, platform.ErrLockUnavailable
	}

	_, _, err := r.acquireDependencyScenarioLock("test-genie", func() (bool, error) {
		return false, nil
	})
	if !errors.Is(err, ErrScenarioBusy) {
		t.Fatalf("error = %v, want ErrScenarioBusy", err)
	}
	if msg := err.Error(); !contains(msg, "remained busy") {
		t.Fatalf("error message should mention timeout, got %q", msg)
	}
}

func TestAcquireDependencyScenarioLockBestEffortUsesShortBound(t *testing.T) {
	home := t.TempDir()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	r := &Runner{
		Home: home,
		deps: lifecycleDeps{
			now: func() time.Time { return now },
			sleep: func(d time.Duration) {
				now = now.Add(d)
			},
		},
	}
	// Hold the in-process mutex directly. A blocking mu.Lock() here would
	// prevent the dependency policy from ever observing its 10-second bound.
	releaseHeld, err := r.acquireScenarioLock("optional-dependency")
	if err != nil {
		t.Fatalf("seed in-process lock: %v", err)
	}
	defer releaseHeld()

	_, _, err = r.acquireDependencyScenarioLockContextWithPolicy(
		context.Background(),
		"optional-dependency",
		func() (bool, error) { return false, nil },
		dependencyBestEffortLockPolicy,
	)
	if !errors.Is(err, ErrScenarioBusy) {
		t.Fatalf("error = %v, want ErrScenarioBusy", err)
	}
	if msg := err.Error(); !contains(msg, "10s") {
		t.Fatalf("error should report the best-effort bound, got %q", msg)
	}
	if elapsed := now.Sub(time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)); elapsed > dependencyBestEffortLockPolicy.Timeout {
		t.Fatalf("virtual wait = %s, want no more than %s", elapsed, dependencyBestEffortLockPolicy.Timeout)
	}
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

func seedScenarioLockFile(t *testing.T, home, name, contents string) {
	t.Helper()
	lockDir := filepath.Join(home, scenarioLockDirName)
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatalf("mkdir lock dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(lockDir, "scenario-"+sanitizeScenarioName(name)+".lock"), []byte(contents), 0o644); err != nil {
		t.Fatalf("seed lock file: %v", err)
	}
}
