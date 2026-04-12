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

	originalListProcessTable := listProcessTableFn
	t.Cleanup(func() {
		listProcessTableFn = originalListProcessTable
	})
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{
			4100: {PID: 4100, PPID: 1, PGID: 4100, Command: filepath.Join(root, "scenarios", "alpha", "api", "server")},
			4101: {PID: 4101, PPID: 4100, PGID: 4100, Command: filepath.Join(root, "scenarios", "alpha", "ui", "vite")},
			5200: {PID: 5200, PPID: 1, PGID: 5200, Command: filepath.Join(root, "scenarios", "beta", "api", "server")},
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

func TestListOrphansHonorsTrackedProcessGroup(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	if err := os.MkdirAll(process.ScenarioProcessDir(home, "alpha"), 0o755); err != nil {
		t.Fatalf("mkdir process dir: %v", err)
	}
	if err := process.WriteScenarioRecord(home, "alpha", "start-ui", process.Record{PID: 4100, PGID: 9000}); err != nil {
		t.Fatalf("write scenario record: %v", err)
	}

	originalListProcessTable := listProcessTableFn
	t.Cleanup(func() {
		listProcessTableFn = originalListProcessTable
	})
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{
			4101: {PID: 4101, PPID: 1, PGID: 9000, Command: filepath.Join(root, "scenarios", "alpha", "ui", "vite")},
			5200: {PID: 5200, PPID: 1, PGID: 5200, Command: filepath.Join(root, "scenarios", "beta", "api", "server")},
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

	originalListProcessTable := listProcessTableFn
	originalKillProcess := killProcessFn
	t.Cleanup(func() {
		listProcessTableFn = originalListProcessTable
		killProcessFn = originalKillProcess
	})

	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{
			5200: {PID: 5200, PPID: 1, PGID: 5200, Command: filepath.Join(root, "scenarios", "beta", "api", "server")},
		}, nil
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
	originalInspection := inspectPortListenersFn
	originalListProcessTable := listProcessTableFn
	t.Cleanup(func() {
		listLocksFn = originalListLocks
		readLockFileFn = originalReadLock
		inspectPortListenersFn = originalInspection
		listProcessTableFn = originalListProcessTable
	})
	listLocksFn = func(home string) ([]LockInfo, error) {
		return network.ListLocks(home)
	}
	readLockFileFn = func(path string) (LockInfo, error) {
		return network.ReadLockFile(path)
	}
	inspectPortListenersFn = func(port int) (network.PortInspection, error) {
		if port != 21234 {
			t.Fatalf("port = %d", port)
		}
		return network.PortInspection{
			Listeners:  []PortListener{{PID: 4321, Command: "listener command"}},
			Inspection: network.ListenerInspection{Available: true, Tool: "lsof"},
		}, nil
	}
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{
			5200: {PID: 5200, PPID: 1, PGID: 5200, Command: filepath.Join(root, "scenarios", "beta", "api", "server")},
		}, nil
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

func TestDiagnosePortReportsUnavailableListenerInspection(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	originalInspection := inspectPortListenersFn
	originalListProcessTable := listProcessTableFn
	t.Cleanup(func() {
		inspectPortListenersFn = originalInspection
		listProcessTableFn = originalListProcessTable
	})
	inspectPortListenersFn = func(port int) (network.PortInspection, error) {
		return network.PortInspection{
			Inspection: network.ListenerInspection{
				Available: false,
				Reason:    "lsof is not installed",
			},
		}, nil
	}
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{}, nil
	}

	controller := NewController(root, home)
	diagnostic, err := controller.DiagnosePort(21234, "")
	if err != nil {
		t.Fatalf("DiagnosePort: %v", err)
	}
	if diagnostic.ListenerInspection.Available {
		t.Fatalf("listener inspection = %#v", diagnostic.ListenerInspection)
	}
	if len(diagnostic.Recommendations) == 0 || diagnostic.Recommendations[0] != "Listener inspection unavailable: lsof is not installed" {
		t.Fatalf("recommendations = %#v", diagnostic.Recommendations)
	}
}

func TestListProcessesErrorPropagates(t *testing.T) {
	originalListProcessTable := listProcessTableFn
	t.Cleanup(func() { listProcessTableFn = originalListProcessTable })
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return nil, errors.New("boom")
	}

	controller := NewController(t.TempDir(), t.TempDir())
	if _, err := controller.ListOrphans(); err == nil || err.Error() != "boom" {
		t.Fatalf("err = %v", err)
	}
}

func TestSnapshotReturnsSharedHealthMetrics(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	if err := os.MkdirAll(process.ScenarioProcessDir(home, "alpha"), 0o755); err != nil {
		t.Fatalf("mkdir process dir: %v", err)
	}
	if err := process.WriteScenarioRecord(home, "alpha", "start-api", process.Record{PID: 4100, PGID: 4100}); err != nil {
		t.Fatalf("write scenario record: %v", err)
	}

	originalListProcessTable := listProcessTableFn
	t.Cleanup(func() { listProcessTableFn = originalListProcessTable })
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{
			4100: {PID: 4100, PPID: 1, PGID: 4100, State: "S", Command: filepath.Join(root, "scenarios", "alpha", "api", "server")},
			5200: {PID: 5200, PPID: 1, PGID: 5200, State: "Z", Command: filepath.Join(root, "scenarios", "beta", "api", "server")},
		}, nil
	}

	controller := NewController(root, home)
	snapshot, err := controller.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snapshot.TrackedProcesses != 1 || snapshot.RunningTracked != 1 {
		t.Fatalf("snapshot tracking = %#v", snapshot)
	}
	if snapshot.ZombieProcesses != 1 || snapshot.OrphanProcesses != 1 {
		t.Fatalf("snapshot health counts = %#v", snapshot)
	}

	health := snapshot.HealthSnapshot()
	if health.ZombieStatus != "normal" || health.OrphanStatus != "normal" || health.OverallStatus != "normal" {
		t.Fatalf("health = %#v", health)
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
