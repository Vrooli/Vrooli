package maintenance

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/process"
)

func TestListOrphansFiltersTrackedAncestors(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	if err := os.MkdirAll(process.ScenarioProcessDir(home, "alpha"), 0o755); err != nil {
		t.Fatalf("mkdir process dir: %v", err)
	}
	if err := process.WriteScenarioRecord(home, "alpha", "start-api", process.Record{PID: 4100, PGID: 4100}); err != nil {
		t.Fatalf("write scenario record: %v", err)
	}

	originalListProcesses := listProcessesFn
	t.Cleanup(func() {
		listProcessesFn = originalListProcesses
	})
	listProcessesFn = func() ([]SystemProcess, error) {
		return []SystemProcess{
			{PID: 4100, PPID: 1, Command: filepath.Join(root, "scenarios", "alpha", "api", "server")},
			{PID: 4101, PPID: 4100, Command: filepath.Join(root, "scenarios", "alpha", "ui", "vite")},
			{PID: 5200, PPID: 1, Command: filepath.Join(root, "scenarios", "beta", "api", "server")},
		}, nil
	}

	controller := NewController(root, home)
	orphans, err := controller.ListOrphans()
	if err != nil {
		t.Fatalf("ListOrphans: %v", err)
	}
	if len(orphans) != 1 || orphans[0].PID != 5200 {
		t.Fatalf("orphans = %#v", orphans)
	}
}

func TestKillOrphansUsesGracefulThenForce(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	originalListProcesses := listProcessesFn
	originalKillProcess := killProcessFn
	t.Cleanup(func() {
		listProcessesFn = originalListProcesses
		killProcessFn = originalKillProcess
	})

	listProcessesFn = func() ([]SystemProcess, error) {
		return []SystemProcess{{PID: 5200, PPID: 1, Command: filepath.Join(root, "scenarios", "beta", "api", "server")}}, nil
	}

	calls := make([]string, 0, 2)
	killProcessFn = func(pid int, force bool) error {
		calls = append(calls, "pid="+strconv.Itoa(pid)+",force="+boolString(force))
		return nil
	}

	controller := NewController(root, home)
	report, err := controller.KillOrphans()
	if err != nil {
		t.Fatalf("KillOrphans: %v", err)
	}
	if len(report.Stopped) != 1 {
		t.Fatalf("stopped = %#v", report.Stopped)
	}
	if !reflect.DeepEqual(calls, []string{"pid=5200,force=false"}) {
		t.Fatalf("kill calls = %#v", calls)
	}
}

func TestCleanStaleLocksRemovesOnlyDeadOwners(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	stateDir := process.ScenarioStateDir(home)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}

	staleLock := filepath.Join(stateDir, ".port_21234.lock")
	liveLock := filepath.Join(stateDir, ".port_21235.lock")
	if err := os.WriteFile(staleLock, []byte("ghost:999999:1\n"), 0o644); err != nil {
		t.Fatalf("write stale lock: %v", err)
	}
	if err := os.WriteFile(liveLock, []byte("alive:"+strconv.Itoa(os.Getpid())+":1\n"), 0o644); err != nil {
		t.Fatalf("write live lock: %v", err)
	}

	controller := NewController(root, home)
	report, err := controller.CleanStaleLocks()
	if err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}
	if len(report.Stopped) != 1 || report.Stopped[0].Name != "21234" {
		t.Fatalf("report = %#v", report)
	}
	if _, err := os.Stat(staleLock); !os.IsNotExist(err) {
		t.Fatalf("expected stale lock removal, stat err=%v", err)
	}
	if _, err := os.Stat(liveLock); err != nil {
		t.Fatalf("expected live lock to remain: %v", err)
	}
}

func TestDiagnosePortBuildsRecommendations(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	stateDir := process.ScenarioStateDir(home)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	lockPath := filepath.Join(stateDir, ".port_21234.lock")
	if err := os.WriteFile(lockPath, []byte("ghost:999999:1\n"), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}

	originalListLocks := listLocksFn
	originalReadLock := readLockFileFn
	originalListeners := listPortListenersFn
	originalListProcesses := listProcessesFn
	t.Cleanup(func() {
		listLocksFn = originalListLocks
		readLockFileFn = originalReadLock
		listPortListenersFn = originalListeners
		listProcessesFn = originalListProcesses
	})
	listLocksFn = func(home string) ([]LockInfo, error) {
		return network.ListLocks(home)
	}
	readLockFileFn = func(path string) (LockInfo, error) {
		return network.ReadLockFile(path)
	}
	listPortListenersFn = func(port int) ([]PortListener, error) {
		if port != 21234 {
			t.Fatalf("port = %d", port)
		}
		return []PortListener{{PID: 4321, Command: "listener command"}}, nil
	}
	listProcessesFn = func() ([]SystemProcess, error) {
		return []SystemProcess{{PID: 5200, PPID: 1, Command: filepath.Join(root, "scenarios", "beta", "api", "server")}}, nil
	}

	controller := NewController(root, home)
	diagnostic, err := controller.DiagnosePort(21234, "alpha")
	if err != nil {
		t.Fatalf("DiagnosePort: %v", err)
	}
	if !diagnostic.InUse || diagnostic.Lock == nil || !diagnostic.Lock.Stale {
		t.Fatalf("diagnostic = %#v", diagnostic)
	}
	if len(diagnostic.Recommendations) < 3 {
		t.Fatalf("recommendations = %#v", diagnostic.Recommendations)
	}
}

func TestListProcessesErrorPropagates(t *testing.T) {
	originalListProcesses := listProcessesFn
	t.Cleanup(func() { listProcessesFn = originalListProcesses })
	listProcessesFn = func() ([]SystemProcess, error) {
		return nil, errors.New("boom")
	}

	controller := NewController(t.TempDir(), t.TempDir())
	if _, err := controller.ListOrphans(); err == nil || err.Error() != "boom" {
		t.Fatalf("err = %v", err)
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
