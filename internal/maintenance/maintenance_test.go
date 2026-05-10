package maintenance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

type maintenanceTestClock struct {
	mu  sync.Mutex
	now time.Time
}

func newMaintenanceTestClock(now time.Time) *maintenanceTestClock {
	return &maintenanceTestClock{now: now.UTC()}
}

func (c *maintenanceTestClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func TestLooksLikeVrooliProcessSkipsSystemDaemons(t *testing.T) {
	root := "/home/alice/Vrooli"
	home := "/home/alice"

	cases := []struct {
		name  string
		entry processTableEntry
	}{
		{
			name: "postgres worker with vrooli in command",
			entry: processTableEntry{
				PID:        900,
				Command:    "postgres: vrooli vrooli_ecosystem_manager 172.18.0.1(39256) idle",
				Executable: "/usr/lib/postgresql/15/bin/postgres",
				Cwd:        "/var/lib/postgresql/15/main",
			},
		},
		{
			name: "fuse-overlayfs with vrooli paths in argv",
			entry: processTableEntry{
				PID:        901,
				Command:    "fuse-overlayfs -o lowerdir=/home/alice/Vrooli,upperdir=/home/alice/.local/...",
				Executable: "/usr/bin/fuse-overlayfs",
				Cwd:        "/",
			},
		},
		{
			name: "user shell with Vrooli cwd",
			entry: processTableEntry{
				PID:        902,
				Command:    "/bin/bash -c ...",
				Executable: "/bin/bash",
				Cwd:        root,
			},
		},
		{
			name: "opencode in data dir",
			entry: processTableEntry{
				PID:        903,
				Command:    "/home/alice/Vrooli/data/opencode/bin/opencode",
				Executable: "/home/alice/Vrooli/data/opencode/bin/opencode",
				Cwd:        "/home/alice",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if looksLikeVrooliProcess(root, home, tc.entry) {
				t.Fatalf("entry should not be classified as Vrooli: %+v", tc.entry)
			}
		})
	}
}

func TestLooksLikeVrooliProcessMatchesInstalledCLIs(t *testing.T) {
	root := "/home/alice/Vrooli"
	home := "/home/alice"

	entry := processTableEntry{
		PID:        4100,
		Executable: "/home/alice/.vrooli/bin/test-genie",
		Command:    "test-genie status",
		Cwd:        "/home/alice",
	}
	if !looksLikeVrooliProcess(root, home, entry) {
		t.Fatalf("expected installed Vrooli CLI to match")
	}
}

func TestLooksLikeVrooliProcessExcludesAutohealLoop(t *testing.T) {
	root := "/home/alice/Vrooli"
	home := "/home/alice"

	entry := processTableEntry{
		PID:        4100,
		Executable: "/home/alice/Vrooli/scenarios/vrooli-autoheal/cli/vrooli-autoheal-loop",
		Command:    "/home/alice/Vrooli/scenarios/vrooli-autoheal/cli/vrooli-autoheal-loop",
		Cwd:        "/home/alice/Vrooli/scenarios/vrooli-autoheal/cli",
	}
	if looksLikeVrooliProcess(root, home, entry) {
		t.Fatalf("expected watchdog loop to be excluded from orphan classification")
	}
}

func TestLooksLikeVrooliProcessExcludesDeletedAutohealLoopExecutable(t *testing.T) {
	root := "/home/alice/Vrooli"
	home := "/home/alice"

	entry := processTableEntry{
		PID:        4101,
		Executable: "/home/alice/Vrooli/scenarios/vrooli-autoheal/cli/vrooli-autoheal-loop (deleted)",
		Command:    "/home/alice/Vrooli/scenarios/vrooli-autoheal/cli/vrooli-autoheal-loop",
		Cwd:        "/home/alice/Vrooli/scenarios/vrooli-autoheal/cli",
	}
	if looksLikeVrooliProcess(root, home, entry) {
		t.Fatalf("expected deleted watchdog executable to be excluded from orphan classification")
	}
}

func TestLooksLikeVrooliProcessMatchesScenarioBinary(t *testing.T) {
	root := "/home/alice/Vrooli"
	home := "/home/alice"

	entry := processTableEntry{
		PID:        4101,
		Executable: "/home/alice/Vrooli/scenarios/beta/api/beta-api",
		Command:    "beta-api",
		Cwd:        "/home/alice/Vrooli/scenarios/beta/api",
	}
	if !looksLikeVrooliProcess(root, home, entry) {
		t.Fatalf("expected compiled scenario binary to match")
	}
}

func TestLooksLikeVrooliProcessMatchesVrooliBinary(t *testing.T) {
	root := "/home/alice/Vrooli"
	home := "/home/alice"

	entry := processTableEntry{
		PID:        4102,
		Executable: "/home/alice/Vrooli/vrooli",
		Command:    "/home/alice/Vrooli/vrooli status",
	}
	if !looksLikeVrooliProcess(root, home, entry) {
		t.Fatalf("expected root-level vrooli binary to match")
	}
}

func TestLooksLikeVrooliProcessMatchesInterpreterInScenarioCwd(t *testing.T) {
	root := "/home/alice/Vrooli"
	home := "/home/alice"

	entry := processTableEntry{
		PID:        4103,
		Executable: "/usr/bin/node",
		Command:    "node node_modules/.bin/vite",
		Cwd:        "/home/alice/Vrooli/scenarios/test-genie/ui",
	}
	if !looksLikeVrooliProcess(root, home, entry) {
		t.Fatalf("expected node in scenario cwd to match")
	}
}

func TestLooksLikeVrooliProcessIgnoresInterpreterOutsideScenarios(t *testing.T) {
	root := "/home/alice/Vrooli"
	home := "/home/alice"

	entry := processTableEntry{
		PID:        4104,
		Executable: "/usr/bin/node",
		Command:    "node some-unrelated-script",
		Cwd:        "/home/alice",
	}
	if looksLikeVrooliProcess(root, home, entry) {
		t.Fatalf("expected node outside scenarios/resources to be excluded")
	}
}

func TestLooksLikeVrooliProcessLegacyFallbackWhenProcUnavailable(t *testing.T) {
	// When /proc is unavailable, Executable is empty. Fall back to checking
	// whether the command line contains a Vrooli-owned prefix.
	root := "/home/alice/Vrooli"
	home := "/home/alice"

	match := processTableEntry{
		PID:     5200,
		Command: "/home/alice/Vrooli/scenarios/beta/api/server --port 18700",
	}
	if !looksLikeVrooliProcess(root, home, match) {
		t.Fatalf("legacy fallback should match scenario path in command")
	}

	noMatch := processTableEntry{
		PID:     5201,
		Command: "postgres: vrooli vrooli_beta 172.18.0.1(12345) idle",
	}
	if looksLikeVrooliProcess(root, home, noMatch) {
		t.Fatalf("legacy fallback must not match postgres worker by substring alone")
	}
}

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

func TestSnapshotHonorsRegistryProcessRefs(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	ctx := context.Background()

	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{InstanceID: "inst-alpha", Scenario: "alpha", Status: scenarioruntime.StatusRunning}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	pid := 4100
	pgid := 4100
	if _, err := store.AddProcessRef(ctx, scenarioruntime.ProcessRef{
		RefID:      "proc-alpha-api",
		InstanceID: instance.InstanceID,
		PID:        &pid,
		PGID:       &pgid,
		Step:       "start-api",
		Status:     "running",
	}); err != nil {
		t.Fatalf("AddProcessRef: %v", err)
	}

	originalListProcessTable := listProcessTableFn
	t.Cleanup(func() { listProcessTableFn = originalListProcessTable })
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{
			4100: {PID: 4100, PPID: 1, PGID: 4100, SID: 4100, Executable: filepath.Join(root, "scenarios", "alpha", "api", "server")},
			4101: {PID: 4101, PPID: 4100, PGID: 4100, SID: 4100, Executable: filepath.Join(root, "scenarios", "alpha", "api", "worker")},
			5200: {PID: 5200, PPID: 1, PGID: 5200, SID: 5200, Executable: filepath.Join(root, "scenarios", "beta", "api", "server")},
		}, nil
	}

	snapshot, err := NewController(root, home).Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snapshot.TrackedProcesses != 1 || snapshot.RunningTracked != 1 {
		t.Fatalf("tracked counts = %+v", snapshot)
	}
	if len(snapshot.Orphans) != 1 || snapshot.Orphans[0].PID != 5200 {
		t.Fatalf("orphans = %#v", snapshot.Orphans)
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

	orphan := processTableEntry{PID: 5200, PPID: 1, PGID: 5200, Executable: filepath.Join(root, "scenarios", "beta", "api", "server"), Command: filepath.Join(root, "scenarios", "beta", "api", "server")}

	originalListProcessTable := listProcessTableFn
	originalKillProcess := killProcessFn
	originalReadEntry := readProcessEntryFn
	t.Cleanup(func() {
		listProcessTableFn = originalListProcessTable
		killProcessFn = originalKillProcess
		readProcessEntryFn = originalReadEntry
	})

	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{5200: orphan}, nil
	}
	readProcessEntryFn = func(pid int) (processTableEntry, bool) {
		if pid == 5200 {
			return orphan, true
		}
		return processTableEntry{}, false
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

// TestKillOrphansSkipsRecycledPID verifies the check-and-act race fix: if the
// orphan PID has been recycled to an unrelated process by the time we iterate,
// we do not send a signal to it. The orphan is still reported as "stopped"
// since it is effectively gone.
func TestKillOrphansSkipsRecycledPID(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	vrooliOrphan := processTableEntry{PID: 5200, PPID: 1, PGID: 5200, Executable: filepath.Join(root, "scenarios", "beta", "api", "server"), Command: filepath.Join(root, "scenarios", "beta", "api", "server")}
	recycled := processTableEntry{PID: 5200, PPID: 1, PGID: 5200, Executable: "/usr/bin/sshd", Command: "/usr/bin/sshd"}

	originalListProcessTable := listProcessTableFn
	originalKillProcess := killProcessFn
	originalReadEntry := readProcessEntryFn
	t.Cleanup(func() {
		listProcessTableFn = originalListProcessTable
		killProcessFn = originalKillProcess
		readProcessEntryFn = originalReadEntry
	})

	// Snapshot sees the orphan as Vrooli.
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{5200: vrooliOrphan}, nil
	}
	// By the time KillOrphans re-reads /proc, the PID has been recycled.
	readProcessEntryFn = func(pid int) (processTableEntry, bool) {
		if pid == 5200 {
			return recycled, true
		}
		return processTableEntry{}, false
	}

	calls := make([]string, 0)
	killProcessFn = func(pid int, force bool) error {
		calls = append(calls, "pid="+strconv.Itoa(pid)+",force="+boolString(force))
		return nil
	}

	controller := NewController(root, home)
	report, err := controller.KillOrphans()
	if err != nil {
		t.Fatalf("KillOrphans: %v", err)
	}
	if len(calls) != 0 {
		t.Fatalf("expected no signals sent to a recycled PID; got %v", calls)
	}
	if len(report.Stopped) != 1 || report.Stopped[0].Name != "5200" {
		t.Fatalf("expected recycled orphan to be reported stopped; got %#v", report.Stopped)
	}
}

// TestKillOrphansSkipsExitedPID: if the PID has already exited between the
// snapshot and KillOrphans iteration, no signal should fire.
func TestKillOrphansSkipsExitedPID(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	vrooliOrphan := processTableEntry{PID: 5200, PPID: 1, PGID: 5200, Executable: filepath.Join(root, "scenarios", "beta", "api", "server"), Command: filepath.Join(root, "scenarios", "beta", "api", "server")}

	originalListProcessTable := listProcessTableFn
	originalKillProcess := killProcessFn
	originalReadEntry := readProcessEntryFn
	t.Cleanup(func() {
		listProcessTableFn = originalListProcessTable
		killProcessFn = originalKillProcess
		readProcessEntryFn = originalReadEntry
	})

	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{5200: vrooliOrphan}, nil
	}
	readProcessEntryFn = func(pid int) (processTableEntry, bool) {
		return processTableEntry{}, false // process gone
	}

	calls := make([]string, 0)
	killProcessFn = func(pid int, force bool) error {
		calls = append(calls, "pid="+strconv.Itoa(pid)+",force="+boolString(force))
		return nil
	}

	controller := NewController(root, home)
	report, err := controller.KillOrphans()
	if err != nil {
		t.Fatalf("KillOrphans: %v", err)
	}
	if len(calls) != 0 {
		t.Fatalf("expected no signals for an exited PID; got %v", calls)
	}
	if len(report.Stopped) != 1 {
		t.Fatalf("expected exited orphan to still be reported stopped; got %#v", report.Stopped)
	}
}

// TestListOrphansHonorsTrackedSession covers the session-id fallback: a worker
// process spawned by a tracked dev server may have its own PGID (via setsid in
// node's tinypool/esbuild) but still shares the session with the tracked
// parent. Such workers must NOT be classified as orphans.
func TestListOrphansHonorsTrackedSession(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	if err := os.MkdirAll(process.ScenarioProcessDir(home, "alpha"), 0o755); err != nil {
		t.Fatalf("mkdir process dir: %v", err)
	}
	if err := process.WriteScenarioRecord(home, "alpha", "start-ui", process.Record{PID: 4100, PGID: 4100}); err != nil {
		t.Fatalf("write scenario record: %v", err)
	}

	originalListProcessTable := listProcessTableFn
	t.Cleanup(func() { listProcessTableFn = originalListProcessTable })
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		// 4100 is the tracked node dev server (session leader).
		// 4200 is an esbuild worker spawned by node with its own PGID (setsid)
		// but still in the same session. 4200's ppid is 1 here to simulate a
		// reparented intermediate (the chain walk alone could not catch it).
		return map[int]processTableEntry{
			4100: {PID: 4100, PPID: 1, PGID: 4100, SID: 4100, Executable: filepath.Join(root, "scenarios", "alpha", "ui", "node")},
			4200: {PID: 4200, PPID: 1, PGID: 4200, SID: 4100, Executable: filepath.Join(root, "scenarios", "alpha", "ui", "node_modules", ".pnpm", "@esbuild+linux-x64@0.21.5", "node_modules", "@esbuild", "linux-x64", "bin", "esbuild")},
			// 5200 is a genuine orphan from a dead scenario.
			5200: {PID: 5200, PPID: 1, PGID: 5200, SID: 5200, Executable: filepath.Join(root, "scenarios", "beta", "api", "server")},
		}, nil
	}

	controller := NewController(root, home)
	orphans, err := controller.ListOrphans()
	if err != nil {
		t.Fatalf("ListOrphans: %v", err)
	}
	if len(orphans) != 1 || orphans[0].PID != 5200 {
		t.Fatalf("expected only 5200 orphan (esbuild worker sharing SID should be filtered); got %#v", orphans)
	}
}

// TestListOrphansIgnoresSessionOneMatch guards against the degenerate case
// where a tracked process reports SID=1 (session leader is init). We must not
// then claim every process in the system is tracked.
func TestListOrphansIgnoresSessionOneMatch(t *testing.T) {
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
		// Tracked pid 4100 has SID=1 (pathological). Unrelated orphan 5200 also
		// has SID=1. It must still be classified as orphan.
		return map[int]processTableEntry{
			4100: {PID: 4100, PPID: 1, PGID: 4100, SID: 1, Executable: filepath.Join(root, "scenarios", "alpha", "api", "server")},
			5200: {PID: 5200, PPID: 1, PGID: 5200, SID: 1, Executable: filepath.Join(root, "scenarios", "beta", "api", "server")},
		}, nil
	}

	controller := NewController(root, home)
	orphans, err := controller.ListOrphans()
	if err != nil {
		t.Fatalf("ListOrphans: %v", err)
	}
	if len(orphans) != 1 || orphans[0].PID != 5200 {
		t.Fatalf("SID=1 must not bridge unrelated processes; orphans=%#v", orphans)
	}
}

// TestKillOrphansPrunesStaleRecords verifies that KillOrphans also sweeps
// scenario process records whose PID no longer resolves to a live process.
// Stale records accumulate whenever a scenario exits outside the normal stop
// path (crash, external kill, host reboot) and should be cleaned up alongside
// the orphan sweep.
func TestKillOrphansPrunesStaleRecords(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	if err := os.MkdirAll(process.ScenarioProcessDir(home, "alpha"), 0o755); err != nil {
		t.Fatalf("mkdir process dir: %v", err)
	}
	// Record points at a PID that's definitely not running.
	if err := process.WriteScenarioRecord(home, "alpha", "start-api", process.Record{PID: 123456789}); err != nil {
		t.Fatalf("write scenario record: %v", err)
	}

	originalListProcessTable := listProcessTableFn
	t.Cleanup(func() { listProcessTableFn = originalListProcessTable })
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{}, nil
	}

	controller := NewController(root, home)
	if _, err := controller.KillOrphans(); err != nil {
		t.Fatalf("KillOrphans: %v", err)
	}

	recordPath := filepath.Join(process.ScenarioProcessDir(home, "alpha"), "start-api.json")
	if _, err := os.Stat(recordPath); !os.IsNotExist(err) {
		t.Fatalf("stale record should have been pruned; stat err=%v", err)
	}
}

// TestCleanStaleRecordsIsBestEffort exercises the dedicated entrypoint that
// surfaces record pruning as its own StopReport.
func TestCleanStaleRecordsIsBestEffort(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	if err := os.MkdirAll(process.ScenarioProcessDir(home, "alpha"), 0o755); err != nil {
		t.Fatalf("mkdir process dir: %v", err)
	}
	if err := process.WriteScenarioRecord(home, "alpha", "start-api", process.Record{PID: 123456789}); err != nil {
		t.Fatalf("write scenario record: %v", err)
	}

	controller := NewController(root, home)
	report, err := controller.CleanStaleRecords()
	if err != nil {
		t.Fatalf("CleanStaleRecords: %v", err)
	}
	if len(report.Stopped) != 1 {
		t.Fatalf("expected 1 pruned record, got %d: %#v", len(report.Stopped), report.Stopped)
	}

	// A second pass should prune nothing.
	report, err = controller.CleanStaleRecords()
	if err != nil {
		t.Fatalf("CleanStaleRecords (second pass): %v", err)
	}
	if len(report.Stopped) != 0 {
		t.Fatalf("second pass should be a no-op, got %#v", report.Stopped)
	}
}

func TestDiagnosePortShowsRegistryClaimHealthAndProcessRefs(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	ctx := context.Background()

	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{InstanceID: "inst-alpha", Scenario: "alpha", Status: scenarioruntime.StatusRunning}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	if _, err := store.CreateSupervisorSession(ctx, scenarioruntime.SupervisorSession{
		SupervisorID:    "sup-alpha",
		HostBootID:      "boot-current",
		HostSessionID:   "session-current",
		Status:          scenarioruntime.SupervisorStatusRunning,
		LastHeartbeatAt: time.Now().UTC(),
	}, time.Minute); err != nil {
		t.Fatalf("CreateSupervisorSession: %v", err)
	}
	instance, err = store.ClaimSupervision(ctx, scenarioruntime.SupervisionClaim{
		InstanceID:   instance.InstanceID,
		Generation:   instance.Generation,
		SupervisorID: "sup-alpha",
	})
	if err != nil {
		t.Fatalf("ClaimSupervision: %v", err)
	}
	claim, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-alpha-api",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "api",
		EnvVar:     "ALPHA_API_PORT",
		Port:       15080,
		Status:     scenarioruntime.ClaimStatusBound,
	})
	if err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}
	ready := true
	if _, err := store.UpsertHealthSnapshot(ctx, scenarioruntime.HealthSnapshot{
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		Status:     scenarioruntime.HealthStatusHealthy,
		Readiness:  &ready,
	}); err != nil {
		t.Fatalf("UpsertHealthSnapshot: %v", err)
	}
	pid := 4100
	if _, err := store.AddProcessRef(ctx, scenarioruntime.ProcessRef{RefID: "proc-alpha-api", InstanceID: instance.InstanceID, PID: &pid, Step: "start-api"}); err != nil {
		t.Fatalf("AddProcessRef: %v", err)
	}

	originalInspect := inspectPortListenersFn
	originalListProcessTable := listProcessTableFn
	originalReadEntry := readProcessEntryFn
	t.Cleanup(func() {
		inspectPortListenersFn = originalInspect
		listProcessTableFn = originalListProcessTable
		readProcessEntryFn = originalReadEntry
	})
	inspectPortListenersFn = func(port int) (network.PortInspection, error) {
		return network.PortInspection{Inspection: network.ListenerInspection{Available: true}}, nil
	}
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{}, nil
	}
	readProcessEntryFn = func(pid int) (processTableEntry, bool) {
		if pid == 4100 {
			return processTableEntry{PID: 4100}, true
		}
		return processTableEntry{}, false
	}

	diagnostic, err := NewController(root, home).DiagnosePort(claim.Port, "alpha")
	if err != nil {
		t.Fatalf("DiagnosePort: %v", err)
	}
	if len(diagnostic.RegistryClaims) != 1 || diagnostic.RegistryClaims[0].ClaimID != claim.ClaimID {
		t.Fatalf("registry claims = %#v", diagnostic.RegistryClaims)
	}
	if diagnostic.RegistryClaims[0].HealthStatus != scenarioruntime.HealthStatusHealthy {
		t.Fatalf("health status = %q", diagnostic.RegistryClaims[0].HealthStatus)
	}
	if diagnostic.RegistryClaims[0].SupervisorID != "sup-alpha" || diagnostic.RegistryClaims[0].SupervisorFresh == nil || !*diagnostic.RegistryClaims[0].SupervisorFresh {
		t.Fatalf("supervisor diagnostics = %#v, want fresh sup-alpha", diagnostic.RegistryClaims[0])
	}
	if len(diagnostic.RegistryProcesses) != 1 || diagnostic.RegistryProcesses[0].PIDRunning == nil || !*diagnostic.RegistryProcesses[0].PIDRunning {
		t.Fatalf("registry processes = %#v", diagnostic.RegistryProcesses)
	}
}

func TestCleanStaleLocksExpiresAbandonedRegistryReservationsAndLegacyLocks(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	ctx := context.Background()
	clk := newMaintenanceTestClock(time.Now().UTC().Add(-2 * time.Hour))

	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	starting, err := store.CreateLease(ctx, scenarioruntime.Instance{InstanceID: "inst-starting", Scenario: "alpha"}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease(starting): %v", err)
	}
	running, err := store.CreateLease(ctx, scenarioruntime.Instance{InstanceID: "inst-running", Scenario: "beta", Status: scenarioruntime.StatusRunning}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease(running): %v", err)
	}
	expiresAt := clk.Now().Add(-time.Minute)
	claim, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-alpha-api",
		InstanceID: starting.InstanceID,
		Scenario:   starting.Scenario,
		PortName:   "api",
		Port:       15080,
		Status:     scenarioruntime.ClaimStatusReserved,
		ExpiresAt:  &expiresAt,
	})
	if err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-beta-api",
		InstanceID: running.InstanceID,
		Scenario:   running.Scenario,
		PortName:   "api",
		Port:       15081,
		Status:     scenarioruntime.ClaimStatusBound,
	}); err != nil {
		t.Fatalf("AcquirePortClaim(bound): %v", err)
	}

	originalPrune := pruneStaleLocksFn
	t.Cleanup(func() { pruneStaleLocksFn = originalPrune })
	pruneStaleLocksFn = func(home string) ([]network.LockInfo, error) {
		return []network.LockInfo{{Port: 15080, Scenario: "alpha", Stale: true}}, nil
	}

	report, err := NewController(root, home).CleanStaleLocks()
	if err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}
	if len(report.Stopped) != 3 {
		t.Fatalf("stopped = %#v, want stale starting lease, reserved claim, legacy lock", report.Stopped)
	}
	afterClaim, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: starting.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(afterClaim) != 1 || afterClaim[0].ClaimID != claim.ClaimID || afterClaim[0].Status != scenarioruntime.ClaimStatusExpired {
		t.Fatalf("after claim = %#v", afterClaim)
	}
	afterRunning, err := store.GetInstance(ctx, running.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance(running): %v", err)
	}
	if afterRunning.Status != scenarioruntime.StatusRunning {
		t.Fatalf("running lease was expired: %#v", afterRunning)
	}
}

func TestCleanStaleLocksExpiresPreviousBootRegistryInstanceAndClaims(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	ctx := context.Background()

	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID: "inst-alpha",
		Scenario:   "alpha",
		Status:     scenarioruntime.StatusRunning,
		HostBootID: "previous-boot",
		WorkingDir: filepath.Join(root, "scenarios", "alpha"),
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	claim, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-alpha-api",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "api",
		Port:       15082,
		Status:     scenarioruntime.ClaimStatusBound,
	})
	if err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}

	originalPrune := pruneStaleLocksFn
	t.Cleanup(func() { pruneStaleLocksFn = originalPrune })
	pruneStaleLocksFn = func(home string) ([]network.LockInfo, error) { return nil, nil }

	report, err := NewController(root, home).CleanStaleLocks()
	if err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}
	if len(report.Stopped) != 2 {
		t.Fatalf("stopped = %#v, want expired instance and claim", report.Stopped)
	}
	afterInstance, err := store.GetInstance(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if afterInstance.Status != scenarioruntime.StatusExpired {
		t.Fatalf("instance status = %q, want expired", afterInstance.Status)
	}
	afterClaims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(afterClaims) != 1 || afterClaims[0].ClaimID != claim.ClaimID || afterClaims[0].Status != scenarioruntime.ClaimStatusExpired {
		t.Fatalf("after claims = %#v", afterClaims)
	}
}

func TestCleanStaleLocksExpiresNonAuthoritativeClaimOnAuthoritativeInstance(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	ctx := context.Background()

	host, err := hostsession.DefaultProvider{}.Current(ctx, home)
	if err != nil {
		t.Fatalf("Current host session: %v", err)
	}
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID: "inst-alpha",
		Scenario:   "alpha",
		Status:     scenarioruntime.StatusRunning,
		HostBootID: host.BootID,
		WorkingDir: filepath.Join(root, "scenarios", "alpha"),
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	pid := os.Getpid()
	if _, err := store.AddProcessRef(ctx, scenarioruntime.ProcessRef{
		RefID:      "proc-alpha-api",
		InstanceID: instance.InstanceID,
		PID:        &pid,
		Step:       "api",
		Status:     "running",
		HostBootID: host.BootID,
	}); err != nil {
		t.Fatalf("AddProcessRef: %v", err)
	}
	claim, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-alpha-ws",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "websocket",
		EnvVar:     "WS_PORT",
		Port:       28888,
		Status:     scenarioruntime.ClaimStatusBound,
	})
	if err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}

	originalPrune := pruneStaleLocksFn
	originalInspect := inspectPortListenersFn
	t.Cleanup(func() {
		pruneStaleLocksFn = originalPrune
		inspectPortListenersFn = originalInspect
	})
	pruneStaleLocksFn = func(home string) ([]network.LockInfo, error) { return nil, nil }
	inspectPortListenersFn = func(port int) (network.PortInspection, error) {
		return network.PortInspection{Inspection: network.ListenerInspection{Available: true, Tool: "test"}}, nil
	}

	report, err := NewController(root, home).CleanStaleLocks()
	if err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}
	if len(report.Stopped) != 1 {
		t.Fatalf("stopped = %#v, want expired claim only", report.Stopped)
	}
	afterInstance, err := store.GetInstance(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if afterInstance.Status != scenarioruntime.StatusRunning {
		t.Fatalf("instance status = %q, want running", afterInstance.Status)
	}
	afterClaims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(afterClaims) != 1 || afterClaims[0].ClaimID != claim.ClaimID || afterClaims[0].Status != scenarioruntime.ClaimStatusExpired {
		t.Fatalf("after claims = %#v", afterClaims)
	}
}

// TestListOrphansExcludesVrooliCLIInvocation guards against classifying a
// transient user-initiated `vrooli` CLI command (and its build subtree) as
// orphans. Without this, `vrooli cleanup orphans` would SIGTERM a concurrent
// `vrooli scenario restart <name>` invocation in progress.
func TestListOrphansExcludesVrooliCLIInvocation(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	if err := os.MkdirAll(process.ScenarioProcessDir(home, "alpha"), 0o755); err != nil {
		t.Fatalf("mkdir process dir: %v", err)
	}

	originalListProcessTable := listProcessTableFn
	t.Cleanup(func() { listProcessTableFn = originalListProcessTable })
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{
			// Sibling vrooli CLI invocation from a user shell (not tracked).
			6000: {PID: 6000, PPID: 5999, PGID: 6000, SID: 6000, Executable: filepath.Join(home, ".vrooli", "bin", "vrooli"), Command: "vrooli scenario restart beta"},
			// Genuine unrelated orphan must still be surfaced.
			5200: {PID: 5200, PPID: 1, PGID: 5200, SID: 5200, Executable: filepath.Join(root, "scenarios", "beta", "api", "server")},
		}, nil
	}

	controller := NewController(root, home)
	orphans, err := controller.ListOrphans()
	if err != nil {
		t.Fatalf("ListOrphans: %v", err)
	}
	if len(orphans) != 1 || orphans[0].PID != 5200 {
		t.Fatalf("vrooli CLI invocation must not be listed as orphan; got %#v", orphans)
	}
}

func TestListOrphansExcludesControlPlaneAPIs(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	originalListProcessTable := listProcessTableFn
	t.Cleanup(func() { listProcessTableFn = originalListProcessTable })
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{
			6000: {PID: 6000, PPID: 1, PGID: 6000, SID: 6000, Executable: filepath.Join(root, "scenarios", "agent-manager", "api", "agent-manager-api"), Command: "./agent-manager-api"},
			6001: {PID: 6001, PPID: 1, PGID: 6001, SID: 6001, Executable: filepath.Join(root, "scenarios", "workspace-sandbox", "api", "workspace-sandbox-api"), Command: "./workspace-sandbox-api"},
			6002: {PID: 6002, PPID: 1, PGID: 6002, SID: 6002, Executable: filepath.Join(root, "scenarios", "swarm-manager", "api", "swarm-manager-api"), Command: "./swarm-manager-api"},
			6100: {PID: 6100, PPID: 1, PGID: 6100, SID: 6100, Executable: filepath.Join(root, "scenarios", "beta", "api", "beta-api"), Command: "./beta-api"},
		}, nil
	}

	controller := NewController(root, home)
	orphans, err := controller.ListOrphans()
	if err != nil {
		t.Fatalf("ListOrphans: %v", err)
	}
	if len(orphans) != 1 || orphans[0].PID != 6100 {
		t.Fatalf("control-plane APIs must not be listed as orphans; got %#v", orphans)
	}
}

// TestParseProcessTableLineParsesSIDColumn covers the new ps -o sid= parse.
func TestParseProcessTableLineParsesSIDColumn(t *testing.T) {
	entry, ok := parseProcessTableLine("  4100  1  4100  4100  S  /usr/bin/node server.js")
	if !ok {
		t.Fatal("parse failed")
	}
	if entry.PID != 4100 || entry.PPID != 1 || entry.PGID != 4100 || entry.SID != 4100 || entry.State != "S" {
		t.Fatalf("fields = %+v", entry)
	}
	if entry.Command != "/usr/bin/node server.js" {
		t.Fatalf("command = %q", entry.Command)
	}
}

func TestParseProcessTableLineRejectsTooFewFields(t *testing.T) {
	if _, ok := parseProcessTableLine("4100 1 4100 S cmd"); ok {
		t.Fatal("expected rejection for missing sid column")
	}
	if _, ok := parseProcessTableLine(""); ok {
		t.Fatal("expected rejection for empty line")
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

func TestDiagnosePortReportsEphemeralOverlap(t *testing.T) {
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
			Inspection: network.ListenerInspection{Available: true, Tool: "stub"},
		}, nil
	}
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{}, nil
	}

	controller := NewController(root, home)
	// 36234 is inside the Linux default ephemeral range. Whatever OS the
	// test runs on, describePortPolicy reports the live range — so we
	// assert that either the InsideEphemeralRange flag fires or the port
	// is at least classified as outside canonical bands.
	diagnostic, err := controller.DiagnosePort(36234, "")
	if err != nil {
		t.Fatalf("DiagnosePort: %v", err)
	}
	if diagnostic.PortPolicy.CanonicalBand != "" {
		t.Errorf("36234 should be outside canonical bands, got %q", diagnostic.PortPolicy.CanonicalBand)
	}
	// On Linux the probe returns 32768-60999 by default and the port falls
	// inside, so the ephemeral recommendation should be the first one. On
	// other hosts (macOS/Windows fallback = 49152-65535) the port is below
	// ephemeral but above canonical max; either way the policy report is
	// populated and one of the two findings shows up in recommendations.
	if diagnostic.PortPolicy.EphemeralMin <= 0 {
		t.Errorf("port policy ephemeral min should be populated, got %+v", diagnostic.PortPolicy)
	}
	haveFlag := diagnostic.PortPolicy.InsideEphemeralRange || diagnostic.PortPolicy.AboveCanonicalMax
	if !haveFlag {
		t.Errorf("expected port 36234 to be flagged as ephemeral or above canonical max, got %+v", diagnostic.PortPolicy)
	}
}

func TestDiagnosePortCanonicalBandForSafePort(t *testing.T) {
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
			Inspection: network.ListenerInspection{Available: true, Tool: "stub"},
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
	if diagnostic.PortPolicy.CanonicalBand != "ui" {
		t.Errorf("canonical band = %q, want ui", diagnostic.PortPolicy.CanonicalBand)
	}
	if diagnostic.PortPolicy.InsideEphemeralRange {
		t.Errorf("21234 should never be inside ephemeral range, got %+v", diagnostic.PortPolicy)
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
