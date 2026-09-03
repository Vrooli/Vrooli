//nolint:goconst // test data deliberately reuses stable filesystem fixtures.
package maintenance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/supervision"
	"github.com/vrooli/vrooli/internal/testenv"
)

// mustProcDir resolves runtime-home paths for tests; the process
// helpers now return errors (they resolve via the runtime_home contract).
func mustProcDir(t *testing.T, home, name string) string {
	t.Helper()
	dir, err := process.ScenarioProcessDir(home, name)
	if err != nil {
		t.Fatalf("ScenarioProcessDir(%q, %q): %v", home, name, err)
	}
	return dir
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
				Command:    "postgres: vrooli vrooli_swarm_manager 172.18.0.1(39256) idle",
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

	if err := os.MkdirAll(mustProcDir(t, home, "alpha"), 0o755); err != nil {
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

func TestListOrphansGivenOwnedDaemonAtInitThenItIsNotAnOrphan(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	started := time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC)

	originalListProcessTable := listProcessTableFn
	originalOwnershipIndex := ownershipIndexFn
	t.Cleanup(func() {
		listProcessTableFn = originalListProcessTable
		ownershipIndexFn = originalOwnershipIndex
	})
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{
			4242: {PID: 4242, PPID: 1, PGID: 4242, SID: 4242, Command: filepath.Join(home, ".vrooli", "artifacts", "ollama", "bin", "ollama")},
			4243: {PID: 4243, PPID: 1, PGID: 4243, SID: 4243, Command: filepath.Join(home, ".vrooli", "artifacts", "abandoned", "server")},
		}, nil
	}
	ownershipIndexFn = func(string) (*supervision.Index, error) {
		return supervision.BuildIndex(
			maintenanceProcessSource{4242: {PID: 4242, StartedAt: started}},
			maintenanceOwnerSource{{Kind: supervision.OwnerKindResource, Name: "ollama", PID: 4242, StartedAt: started}},
		)
	}

	orphans, err := NewController(root, home).ListOrphans()
	if err != nil {
		t.Fatalf("ListOrphans: %v", err)
	}
	if len(orphans) != 1 || orphans[0].PID != 4243 {
		t.Fatalf("orphans = %#v, want only unclaimed pid 4243", orphans)
	}
}

type maintenanceProcessSource map[int]supervision.ProcessInfo

func (s maintenanceProcessSource) Processes() (map[int]supervision.ProcessInfo, error) { return s, nil }

type maintenanceOwnerSource []supervision.Owner

func (s maintenanceOwnerSource) Owners() ([]supervision.Owner, error) { return s, nil }

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

	if err := os.MkdirAll(mustProcDir(t, home, "alpha"), 0o755); err != nil {
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

	if err := os.MkdirAll(mustProcDir(t, home, "alpha"), 0o755); err != nil {
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

	if err := os.MkdirAll(mustProcDir(t, home, "alpha"), 0o755); err != nil {
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

	if err := os.MkdirAll(mustProcDir(t, home, "alpha"), 0o755); err != nil {
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

	recordPath := filepath.Join(mustProcDir(t, home, "alpha"), "start-api.json")
	if _, err := os.Stat(recordPath); !os.IsNotExist(err) {
		t.Fatalf("stale record should have been pruned; stat err=%v", err)
	}
}

// TestCleanStaleRecordsIsBestEffort exercises the dedicated entrypoint that
// surfaces record pruning as its own StopReport.
func TestCleanStaleRecordsIsBestEffort(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	if err := os.MkdirAll(mustProcDir(t, home, "alpha"), 0o755); err != nil {
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
	originalPIDRunning := pidIsRunningFn
	t.Cleanup(func() {
		inspectPortListenersFn = originalInspect
		listProcessTableFn = originalListProcessTable
		pidIsRunningFn = originalPIDRunning
	})
	inspectPortListenersFn = func(port int) network.PortInspection {
		return network.PortInspection{Inspection: network.ListenerInspection{Available: true}}
	}
	stubListenerSnapshot(t, true, nil)
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{}, nil
	}
	pidIsRunningFn = func(pid int) bool { return pid == 4100 }

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

func TestCleanStaleLocksExpiresAbandonedRegistryReservations(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	ctx := context.Background()
	clk := testenv.NewClock(time.Now().UTC().Add(-2 * time.Hour))

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

	stubListenerSnapshot(t, true, nil)
	report, err := NewController(root, home).CleanStaleLocks()
	if err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}
	if len(report.Stopped) != 2 {
		t.Fatalf("stopped = %#v, want stale starting lease and reserved claim", report.Stopped)
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

// Regression: a scenario whose setup phase outruns the 30s lease TTL is still a
// live start. Every `--clean-stale` start runs this sweep, so if the sweep
// condemns it on elapsed time alone, concurrent starts reap each other and the
// victim's next lease write aborts and rolls back a healthy start.
func TestCleanStaleLocksSparesLiveStartingLeaseWithElapsedDeadline(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	ctx := context.Background()

	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ownerPID := os.Getpid() // this test process: unambiguously alive
	instance := scenarioruntime.Instance{
		InstanceID: "inst-slow-setup",
		Scenario:   "web-console",
		Phase:      "setup",
		OwnerPID:   &ownerPID,
	}
	host, hostErr := hostsession.DefaultProvider{}.Current(ctx, "")
	if hostErr == nil {
		instance.HostBootID = host.BootID
	}
	// A TTL already elapsed by the time the sweep runs, exactly like a 92s UI
	// build against the 30s default.
	starting, err := store.CreateLease(ctx, instance, time.Nanosecond)
	if err != nil {
		t.Fatalf("CreateLease(starting): %v", err)
	}

	stubListenerSnapshot(t, true, nil)
	if _, err := NewController(root, home).CleanStaleLocks(); err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}

	after, err := store.GetInstance(ctx, starting.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if after.Status != scenarioruntime.StatusStarting {
		t.Fatalf("status = %q (stop_reason %q), want the live start left in %q",
			after.Status, after.StopReason, scenarioruntime.StatusStarting)
	}
	// The owner must still be able to complete its start after the sweep.
	if _, err := store.HeartbeatLease(ctx, after.InstanceID, after.Generation, 30*time.Second); err != nil {
		t.Fatalf("HeartbeatLease after sweep = %v, want the live start to keep its lease", err)
	}
}

// The sweep must still reap a starting lease whose owner is gone, or genuinely
// abandoned starts would leak their rows and port claims forever.
func TestCleanStaleLocksExpiresStartingLeaseWithDeadOwner(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	ctx := context.Background()

	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	deadPID := 4000000 // above pid_max: cannot be live
	instance := scenarioruntime.Instance{
		InstanceID: "inst-abandoned",
		Scenario:   "alpha",
		Phase:      "setup",
		OwnerPID:   &deadPID,
	}
	host, hostErr := hostsession.DefaultProvider{}.Current(ctx, "")
	if hostErr == nil {
		instance.HostBootID = host.BootID
	}
	starting, err := store.CreateLease(ctx, instance, time.Nanosecond)
	if err != nil {
		t.Fatalf("CreateLease(starting): %v", err)
	}

	stubListenerSnapshot(t, true, nil)
	if _, err := NewController(root, home).CleanStaleLocks(); err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}

	after, err := store.GetInstance(ctx, starting.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if after.Status != scenarioruntime.StatusExpired {
		t.Fatalf("status = %q, want %q for an abandoned starter", after.Status, scenarioruntime.StatusExpired)
	}
	if after.StopReason == "" {
		t.Fatal("stop_reason is empty, want the reap trigger recorded for forensics")
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

	stubListenerSnapshot(t, true, nil)
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

	stubListenerSnapshot(t, true, nil)

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

// TestCleanStaleLocksFinalizesStuckStoppingInstanceWhenOwnerPidDead seeds an
// instance that the previous lifecycle runner left in status=stopping when it
// died mid-stop (owner_pid points at a dead process). The reaper must release
// the bound port claim, mark the running process_refs exited, and stop the
// instance with stop_reason=reaper-finalize — otherwise a subsequent restart
// on the same fixed port will fail with "active registry claim already owns
// port".
func TestCleanStaleLocksFinalizesStuckStoppingInstanceWhenOwnerPidDead(t *testing.T) {
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

	deadPID := 99999
	originalPIDRunning := pidIsRunningFn
	t.Cleanup(func() { pidIsRunningFn = originalPIDRunning })
	pidIsRunningFn = func(pid int) bool { return false }

	heartbeatDeadline := time.Now().UTC().Add(time.Hour)
	instance, err := store.CreateInstance(ctx, scenarioruntime.Instance{
		InstanceID:          "inst-stuck",
		Scenario:            "web-console",
		Status:              scenarioruntime.StatusStopping,
		HostBootID:          host.BootID,
		OwnerPID:            &deadPID,
		HeartbeatDeadlineAt: &heartbeatDeadline,
	})
	if err != nil {
		t.Fatalf("CreateInstance: %v", err)
	}
	claim, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-stuck-ui",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "ui",
		Port:       21233,
		Status:     scenarioruntime.ClaimStatusBound,
	})
	if err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}
	runningPID := 11111
	ref, err := store.AddProcessRef(ctx, scenarioruntime.ProcessRef{
		RefID:      "proc-stuck-ui",
		InstanceID: instance.InstanceID,
		PID:        &runningPID,
		Step:       "ui",
		Status:     "running",
		HostBootID: host.BootID,
	})
	if err != nil {
		t.Fatalf("AddProcessRef: %v", err)
	}

	stubListenerSnapshot(t, true, nil)

	if _, err := NewController(root, home).CleanStaleLocks(); err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}

	afterInstance, err := store.GetInstance(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if afterInstance.Status != scenarioruntime.StatusStopped {
		t.Fatalf("instance status = %q, want stopped", afterInstance.Status)
	}
	if afterInstance.StopReason != scenarioruntime.StopReasonReaperFinalize {
		t.Fatalf("stop reason = %q, want %q", afterInstance.StopReason, scenarioruntime.StopReasonReaperFinalize)
	}
	afterClaims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(afterClaims) != 1 || afterClaims[0].ClaimID != claim.ClaimID || afterClaims[0].Status != scenarioruntime.ClaimStatusReleased {
		t.Fatalf("after claims = %#v, want single released claim", afterClaims)
	}
	afterRefs, err := store.ListProcessRefs(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("ListProcessRefs: %v", err)
	}
	if len(afterRefs) != 1 || afterRefs[0].RefID != ref.RefID || afterRefs[0].Status != "exited" {
		t.Fatalf("after refs = %#v, want single exited ref", afterRefs)
	}
}

// TestCleanStaleLocksFinalizesStuckStoppingInstanceOnBootIDMismatch covers the
// post-reboot recovery case: the lifecycle runner died in the OOM-or-kernel
// sense, the box rebooted, and the registry was left with a stopping
// instance whose owner_pid is unknowable (its meaning changed when the pid
// space was reset). Boot-id mismatch is the unambiguous "this row is from
// the old boot" signal.
func TestCleanStaleLocksFinalizesStuckStoppingInstanceOnBootIDMismatch(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	ctx := context.Background()

	host, err := hostsession.DefaultProvider{}.Current(ctx, home)
	if err != nil {
		t.Fatalf("Current host session: %v", err)
	}
	if host.BootID == "" {
		t.Skip("current host has no boot id; cannot test boot mismatch")
	}
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	livePID := os.Getpid()
	originalPIDRunning := pidIsRunningFn
	t.Cleanup(func() { pidIsRunningFn = originalPIDRunning })
	pidIsRunningFn = func(pid int) bool { return pid == livePID }

	heartbeatDeadline := time.Now().UTC().Add(time.Hour)
	instance, err := store.CreateInstance(ctx, scenarioruntime.Instance{
		InstanceID:          "inst-pre-reboot",
		Scenario:            "web-console",
		Status:              scenarioruntime.StatusStopping,
		HostBootID:          "previous-boot",
		OwnerPID:            &livePID,
		HeartbeatDeadlineAt: &heartbeatDeadline,
	})
	if err != nil {
		t.Fatalf("CreateInstance: %v", err)
	}
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-prev-ui",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "ui",
		Port:       21777,
		Status:     scenarioruntime.ClaimStatusBound,
	}); err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}

	stubListenerSnapshot(t, true, nil)

	if _, err := NewController(root, home).CleanStaleLocks(); err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}

	afterInstance, err := store.GetInstance(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	// Boot-id mismatch is caught earlier by expireNonAuthoritativeRegistryState
	// (which transitions to expired) for active statuses; the stopping-instance
	// reaper finalizes it here. Either outcome leaves the port claim released
	// and the instance non-active. Tests assert the safety invariant: the
	// claim must not be left in a bound/reserved state.
	if afterInstance.Status == scenarioruntime.StatusStopping {
		t.Fatalf("instance still stopping; reaper did not run: %#v", afterInstance)
	}
	afterClaims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(afterClaims) == 0 {
		t.Fatalf("expected at least one claim row")
	}
	for _, c := range afterClaims {
		if c.Status == scenarioruntime.ClaimStatusBound || c.Status == scenarioruntime.ClaimStatusReserved {
			t.Fatalf("claim left in active status after reap: %#v", c)
		}
	}
}

// TestCleanStaleLocksDoesNotTouchHealthyStoppingInstance guards the inverse
// invariant: a normal in-flight stop (live owner_pid, fresh heartbeat,
// matching boot) must not be preempted by the reaper. Otherwise a slow but
// healthy lifecycle stop could race the reaper into finalizing it twice and
// double-counting the cleanup.
func TestCleanStaleLocksDoesNotTouchHealthyStoppingInstance(t *testing.T) {
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

	livePID := os.Getpid()
	originalPIDRunning := pidIsRunningFn
	t.Cleanup(func() { pidIsRunningFn = originalPIDRunning })
	pidIsRunningFn = func(pid int) bool { return pid == livePID }

	heartbeatDeadline := time.Now().UTC().Add(time.Hour)
	instance, err := store.CreateInstance(ctx, scenarioruntime.Instance{
		InstanceID:          "inst-graceful",
		Scenario:            "web-console",
		Status:              scenarioruntime.StatusStopping,
		HostBootID:          host.BootID,
		OwnerPID:            &livePID,
		HeartbeatDeadlineAt: &heartbeatDeadline,
	})
	if err != nil {
		t.Fatalf("CreateInstance: %v", err)
	}
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-graceful-ui",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "ui",
		Port:       21999,
		Status:     scenarioruntime.ClaimStatusBound,
	}); err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}

	stubListenerSnapshot(t, true, nil)

	if _, err := NewController(root, home).CleanStaleLocks(); err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}

	afterInstance, err := store.GetInstance(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if afterInstance.Status != scenarioruntime.StatusStopping {
		t.Fatalf("healthy stopping instance was touched by reaper: %#v", afterInstance)
	}
	afterClaims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(afterClaims) != 1 || afterClaims[0].Status != scenarioruntime.ClaimStatusBound {
		t.Fatalf("healthy stopping claim was touched by reaper: %#v", afterClaims)
	}
}

// TestDiagnosePortRecommendationMatchesCleanupAction locks in the contract
// that diagnose-port and cleanup locks stay aligned: every reconcile reason
// for which diagnose-port recommends `vrooli cleanup locks` must actually be
// fixable by CleanStaleLocks. This was the third bug in the original RCA —
// the recommendation pointed at a no-op for stuck-stopping rows.
func TestDiagnosePortRecommendationMatchesCleanupAction(t *testing.T) {
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

	deadPID := 99999
	originalPIDRunning := pidIsRunningFn
	t.Cleanup(func() { pidIsRunningFn = originalPIDRunning })
	pidIsRunningFn = func(pid int) bool { return false }

	heartbeatDeadline := time.Now().UTC().Add(time.Hour)
	instance, err := store.CreateInstance(ctx, scenarioruntime.Instance{
		InstanceID:          "inst-recommend",
		Scenario:            "web-console",
		Status:              scenarioruntime.StatusStopping,
		HostBootID:          host.BootID,
		OwnerPID:            &deadPID,
		HeartbeatDeadlineAt: &heartbeatDeadline,
	})
	if err != nil {
		t.Fatalf("CreateInstance: %v", err)
	}
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-recommend-ui",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "ui",
		Port:       22333,
		Status:     scenarioruntime.ClaimStatusBound,
	}); err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}

	stubListenerSnapshot(t, true, nil)
	originalInspect := inspectPortListenersFn
	t.Cleanup(func() { inspectPortListenersFn = originalInspect })
	inspectPortListenersFn = func(port int) network.PortInspection {
		return network.PortInspection{Inspection: network.ListenerInspection{Available: true, Tool: "test"}}
	}

	controller := NewController(root, home)
	diagnostic, err := controller.DiagnosePort(22333, "web-console")
	if err != nil {
		t.Fatalf("DiagnosePort: %v", err)
	}
	recommendsCleanup := false
	for _, rec := range diagnostic.Recommendations {
		if reflect.DeepEqual(true, true) && containsCleanupLocks(rec) {
			recommendsCleanup = true
			break
		}
	}
	if !recommendsCleanup {
		t.Fatalf("diagnose-port recommendations = %#v, want one referencing `vrooli cleanup locks`", diagnostic.Recommendations)
	}

	if _, err := controller.CleanStaleLocks(); err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}
	afterInstance, err := store.GetInstance(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if afterInstance.Status == scenarioruntime.StatusStopping {
		t.Fatalf("CleanStaleLocks did not act on recommended state: %#v", afterInstance)
	}
}

func containsCleanupLocks(rec string) bool {
	return rec != "" && (indexContains(rec, "vrooli cleanup locks") || indexContains(rec, "cleanup locks"))
}

func indexContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestListRuntimeClaimsCapturesSnapshotAfterClaimReads pins the freshness
// ordering on the listing path: the listener snapshot must be captured AFTER
// the claim reads, so evidence is at least as fresh as the claim set. The
// capture stub binds a NEW claim; with correct ordering it cannot appear in
// the result. A refactor hoisting the capture above the reads would surface
// it — classified against evidence that predates its bind.
func TestListRuntimeClaimsCapturesSnapshotAfterClaimReads(t *testing.T) {
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
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID: "claim-alpha-api", InstanceID: instance.InstanceID, Scenario: "alpha",
		PortName: "api", Port: 18080, Status: scenarioruntime.ClaimStatusBound,
	}); err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}

	original := captureListenerSnapshotFn
	t.Cleanup(func() { captureListenerSnapshotFn = original })
	captureListenerSnapshotFn = func() network.TCPListenerSnapshot {
		if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
			ClaimID: "claim-alpha-late", InstanceID: instance.InstanceID, Scenario: "alpha",
			PortName: "late", Port: 19090, Status: scenarioruntime.ClaimStatusBound,
		}); err != nil {
			t.Errorf("AcquirePortClaim(late): %v", err)
		}
		return network.TCPListenerSnapshot{Known: true, Tool: "test", Ports: map[int][]network.SnapshotListener{18080: nil}}
	}

	claims, err := NewController(root, home).ListRuntimeClaims()
	if err != nil {
		t.Fatalf("ListRuntimeClaims: %v", err)
	}
	for _, claim := range claims {
		if claim.ClaimID == "claim-alpha-late" {
			t.Fatalf("claim bound during capture appeared in the listing — snapshot captured before the claim reads: %#v", claim)
		}
	}
	if len(claims) != 1 || claims[0].ClaimID != "claim-alpha-api" {
		t.Fatalf("claims = %#v, want only claim-alpha-api", claims)
	}
}

// TestCleanStaleLocksCapturesSnapshotAfterStoreReads pins the same freshness
// ordering on the CLEANUP path, where it is safety-critical: this path
// EXPIRES claims on known-absent listeners, so a snapshot captured before the
// store reads could wrongly expire a port bound in between. The capture stub
// binds a NEW claim; with correct ordering it is not part of this run's read
// set and must survive untouched.
func TestCleanStaleLocksCapturesSnapshotAfterStoreReads(t *testing.T) {
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
	pid := os.Getpid()
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
	if _, err := store.AddProcessRef(ctx, scenarioruntime.ProcessRef{
		RefID: "proc-alpha-api", InstanceID: instance.InstanceID, PID: &pid,
		Step: "api", Status: "running", HostBootID: host.BootID,
	}); err != nil {
		t.Fatalf("AddProcessRef: %v", err)
	}
	// Same shape as the non-authoritative-claim test above: a bound claim with
	// known-not-listening evidence gets expired. The late claim must NOT meet
	// that fate, because it is bound only once the snapshot is captured.
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID: "claim-alpha-ws", InstanceID: instance.InstanceID, Scenario: "alpha",
		PortName: "websocket", Port: 28888, Status: scenarioruntime.ClaimStatusBound,
	}); err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}

	original := captureListenerSnapshotFn
	t.Cleanup(func() { captureListenerSnapshotFn = original })
	captureListenerSnapshotFn = func() network.TCPListenerSnapshot {
		_, acquireErr := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
			ClaimID: "claim-alpha-late", InstanceID: instance.InstanceID, Scenario: "alpha",
			PortName: "late", Port: 29999, Status: scenarioruntime.ClaimStatusBound,
		})
		if acquireErr != nil && !errors.Is(acquireErr, scenarioruntime.ErrActiveClaimConflict) {
			t.Errorf("AcquirePortClaim(late): %v", acquireErr)
		}
		return network.TCPListenerSnapshot{Known: true, Tool: "test", Ports: map[int][]network.SnapshotListener{}}
	}

	if _, err := NewController(root, home).CleanStaleLocks(); err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}

	afterClaims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	statusByID := map[string]string{}
	for _, c := range afterClaims {
		statusByID[c.ClaimID] = c.Status
	}
	if statusByID["claim-alpha-ws"] != scenarioruntime.ClaimStatusExpired {
		t.Fatalf("pre-existing not-listening claim = %q, want expired (cleanup semantics changed under this test)", statusByID["claim-alpha-ws"])
	}
	if statusByID["claim-alpha-late"] != scenarioruntime.ClaimStatusBound {
		t.Fatalf("claim bound during capture = %q, want still bound — expiring it means the snapshot was captured before the store reads", statusByID["claim-alpha-late"])
	}
}

// stubListenerSnapshot pins the listener-evidence snapshot seam so reconcile
// reads controlled evidence instead of the live host's TCP table. Without
// this, tests using ports inside the real Vrooli allocation bands flake
// whenever a running scenario occupies one. Ports absent from the map read as
// known-not-listening; pass known=false to model an unavailable evidence
// source (the snapshot degrades to unknown and reconcile must not expire on
// it).
func stubListenerSnapshot(t *testing.T, known bool, ports map[int][]network.SnapshotListener) {
	t.Helper()
	original := captureListenerSnapshotFn
	t.Cleanup(func() { captureListenerSnapshotFn = original })
	captureListenerSnapshotFn = func() network.TCPListenerSnapshot {
		if !known {
			return network.TCPListenerSnapshot{Tool: "test", Reason: "stubbed unavailable"}
		}
		return network.TCPListenerSnapshot{Known: true, Tool: "test", Ports: ports}
	}
}

// TestListOrphansExcludesInstalledCLIInvocations guards every command surface
// under ~/.vrooli/bin, not only the root `vrooli` executable. Scenario CLIs
// run bounded jobs that may outlive the shell which launched them; they are not
// abandoned supervised workloads. Artifact binaries remain in scope.
func TestListOrphansExcludesInstalledCLIInvocations(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	if err := os.MkdirAll(mustProcDir(t, home, "alpha"), 0o755); err != nil {
		t.Fatalf("mkdir process dir: %v", err)
	}

	originalListProcessTable := listProcessTableFn
	t.Cleanup(func() { listProcessTableFn = originalListProcessTable })
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{
			// Installed CLI invocations from a user shell (not tracked).
			6000: {PID: 6000, PPID: 5999, PGID: 6000, SID: 6000, Executable: filepath.Join(home, ".vrooli", "bin", "vrooli"), Command: "vrooli scenario restart beta"},
			6001: {PID: 6001, PPID: 1, PGID: 6001, SID: 6001, Executable: filepath.Join(home, ".vrooli", "bin", "git-control-tower"), Command: "git-control-tower baseline collection diff wait --name proof"},
			6002: {PID: 6002, PPID: 1, PGID: 6002, SID: 6002, Executable: filepath.Join(home, ".vrooli", "bin", "tidiness-manager"), Command: "tidiness-manager scan internal --budget-audit"},
			// Genuine unrelated orphan must still be surfaced.
			5200: {PID: 5200, PPID: 1, PGID: 5200, SID: 5200, Executable: filepath.Join(home, ".vrooli", "artifacts", "beta", "1.0", "server")},
		}, nil
	}

	controller := NewController(root, home)
	orphans, err := controller.ListOrphans()
	if err != nil {
		t.Fatalf("ListOrphans: %v", err)
	}
	if len(orphans) != 1 || orphans[0].PID != 5200 {
		t.Fatalf("installed CLI invocations must not be listed as orphans; got %#v", orphans)
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

func TestLooksLikeVrooliProcessExcludesManagedBridgeAgent(t *testing.T) {
	root := "/Users/alice/vrooli"
	home := "/Users/alice"
	entry := processTableEntry{
		PID:     6200,
		PPID:    1,
		PGID:    6200,
		Command: "/Users/alice/.local/lib/vrooli-bridge/vrooli-bridge-agent --control-plane-url http://127.0.0.1:18767",
	}
	if looksLikeVrooliProcess(root, home, entry) {
		t.Fatal("managed Bridge agent must not be classified as a Vrooli orphan")
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

func TestDiagnosePortBuildsRecommendations(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	originalInspection := inspectPortListenersFn
	originalListProcessTable := listProcessTableFn
	t.Cleanup(func() {
		inspectPortListenersFn = originalInspection
		listProcessTableFn = originalListProcessTable
	})
	inspectPortListenersFn = func(port int) network.PortInspection {
		if port != 21234 {
			t.Fatalf("port = %d", port)
		}
		return network.PortInspection{
			Listeners:  []PortListener{{PID: 4321, Command: "listener command"}},
			Inspection: network.ListenerInspection{Available: true, Tool: "lsof"},
		}
	}
	stubListenerSnapshot(t, true, map[int][]network.SnapshotListener{
		21234: {{PID: 4321, Label: "listener command"}},
	})
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
	if !diagnostic.InUse {
		t.Fatalf("diagnostic = %#v", diagnostic)
	}
	if len(diagnostic.Recommendations) < 2 {
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
	inspectPortListenersFn = func(port int) network.PortInspection {
		return network.PortInspection{
			Inspection: network.ListenerInspection{Available: true, Tool: "stub"},
		}
	}
	stubListenerSnapshot(t, true, nil)
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
	inspectPortListenersFn = func(port int) network.PortInspection {
		return network.PortInspection{
			Inspection: network.ListenerInspection{Available: true, Tool: "stub"},
		}
	}
	stubListenerSnapshot(t, true, nil)
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
	inspectPortListenersFn = func(port int) network.PortInspection {
		return network.PortInspection{
			Inspection: network.ListenerInspection{
				Available: false,
				Reason:    "lsof is not installed",
			},
		}
	}
	stubListenerSnapshot(t, false, nil)
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

	if err := os.MkdirAll(mustProcDir(t, home, "alpha"), 0o755); err != nil {
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

func TestNewPIDLivenessMemoProbesEachPIDOnce(t *testing.T) {
	calls := map[int]int{}
	memo := newPIDLivenessMemo(func(pid int) bool {
		calls[pid]++
		return pid%2 == 0
	})
	for i := 0; i < 3; i++ {
		if !memo(2) || memo(3) {
			t.Fatal("memo changed answers across repeated calls")
		}
	}
	if calls[2] != 1 || calls[3] != 1 {
		t.Fatalf("expected one probe per distinct PID, got %v", calls)
	}
}

// TestListRuntimeClaimsCapturesListenerSnapshotOnce is the anti-regression
// guard for the 200x speedup: listener evidence must be captured ONCE per
// listing, never once per claim. Before the overhaul each claim forked its own
// lsof+ps (the dominant cost — 86 lsof + 422 ps on a 189-claim host).
// Reintroducing a per-claim capture would push this counter above 1 no matter
// how the forks are spelled.
func TestListRuntimeClaimsCapturesListenerSnapshotOnce(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	const claimCount = 5
	for i := 0; i < claimCount; i++ {
		suffix := strconv.Itoa(i)
		instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
			InstanceID: "inst-" + suffix,
			Scenario:   "scenario-" + suffix,
			Status:     scenarioruntime.StatusRunning,
		}, time.Minute)
		if err != nil {
			t.Fatalf("CreateLease %d: %v", i, err)
		}
		if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
			ClaimID:    "claim-" + suffix,
			InstanceID: instance.InstanceID,
			Scenario:   instance.Scenario,
			PortName:   "api",
			EnvVar:     "API_PORT",
			Port:       15080 + i,
			Status:     scenarioruntime.ClaimStatusBound,
		}); err != nil {
			t.Fatalf("AcquirePortClaim %d: %v", i, err)
		}
	}

	original := captureListenerSnapshotFn
	t.Cleanup(func() { captureListenerSnapshotFn = original })
	captures := 0
	captureListenerSnapshotFn = func() network.TCPListenerSnapshot {
		captures++
		return network.TCPListenerSnapshot{Known: true, Tool: "test"}
	}

	claims, err := listRuntimeClaims(ctx, store, 0, "", false)
	if err != nil {
		t.Fatalf("listRuntimeClaims: %v", err)
	}
	if len(claims) != claimCount {
		t.Fatalf("listed %d claims, want %d", len(claims), claimCount)
	}
	if captures != 1 {
		t.Fatalf("listener snapshot captured %d times for %d claims; must be exactly 1 (per-claim fork regression)", captures, claimCount)
	}
}

// TestReclaimSquattedPortEvictsOnlyVrooliOrphans proves the port-reclamation
// guard: a leaked Vrooli process holding a just-released port is evicted, while
// a foreign process (or a recycled/absent PID) that happens to hold the port is
// never killed.
func TestReclaimSquattedPortEvictsOnlyVrooliOrphans(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	vrooliOrphan := processTableEntry{PID: 5200, PPID: 1, PGID: 5200, Executable: filepath.Join(root, "scenarios", "beta", "api", "server"), Command: filepath.Join(root, "scenarios", "beta", "api", "server")}
	foreign := processTableEntry{PID: 8000, PPID: 1, PGID: 8000, Executable: "/usr/bin/postgres", Command: "/usr/bin/postgres"}

	originalKill := killProcessFn
	originalRead := readProcessEntryFn
	originalRunning := pidIsRunningFn
	t.Cleanup(func() {
		killProcessFn = originalKill
		readProcessEntryFn = originalRead
		pidIsRunningFn = originalRunning
	})

	readProcessEntryFn = func(pid int) (processTableEntry, bool) {
		switch pid {
		case 5200:
			return vrooliOrphan, true
		case 8000:
			return foreign, true
		default:
			return processTableEntry{}, false
		}
	}
	var killed []int
	killProcessFn = func(pid int, _ bool) error {
		killed = append(killed, pid)
		return nil
	}
	pidIsRunningFn = func(int) bool { return false } // dies on SIGTERM; no SIGKILL escalation

	c := NewController(root, home)

	// A confirmed Vrooli orphan holding the released port is evicted.
	item, evicted := c.reclaimSquattedPort(PortReclaimCandidate{Scenario: "beta", Port: 7100, PID: 5200})
	if !evicted {
		t.Fatalf("expected a Vrooli orphan to be evicted, got item=%#v", item)
	}
	if len(killed) != 1 || killed[0] != 5200 {
		t.Fatalf("expected exactly pid 5200 signaled, got %v", killed)
	}

	// A foreign process that reused the port is never killed.
	killed = nil
	if _, evicted := c.reclaimSquattedPort(PortReclaimCandidate{Scenario: "beta", Port: 7100, PID: 8000}); evicted {
		t.Fatal("expected a foreign process NOT to be evicted")
	}
	if len(killed) != 0 {
		t.Fatalf("foreign process must not be signaled, got %v", killed)
	}

	// A recycled/absent PID is a silent no-op.
	killed = nil
	if _, evicted := c.reclaimSquattedPort(PortReclaimCandidate{Scenario: "beta", Port: 7100, PID: 9999}); evicted {
		t.Fatal("expected an absent PID NOT to be evicted")
	}
	if len(killed) != 0 {
		t.Fatalf("absent PID must not be signaled, got %v", killed)
	}
}

func TestCleanStaleLocksExpiresClaimsUnderTerminalInstances(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	ctx := context.Background()

	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	// An unclean stop leaves the claim bound while the instance row is already
	// terminal — the active-instance walk cannot see it, so only the orphaned-
	// claims pass can expire it.
	instance, err := store.CreateInstance(ctx, scenarioruntime.Instance{
		InstanceID: "inst-crashed",
		Scenario:   "web-console",
		Status:     scenarioruntime.StatusStopped,
	})
	if err != nil {
		t.Fatalf("CreateInstance: %v", err)
	}
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-crashed-ui",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "ui",
		Port:       21998,
		Status:     scenarioruntime.ClaimStatusBound,
	}); err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}

	stubListenerSnapshot(t, true, nil)

	if _, err := NewController(root, home).CleanStaleLocks(); err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}

	afterClaims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(afterClaims) != 1 || afterClaims[0].Status != scenarioruntime.ClaimStatusExpired {
		t.Fatalf("orphaned bound claim should be expired by cleanup; got %#v", afterClaims)
	}
}

// The stale-lock sweep retires editor leases only on proof of death: a dead
// pid is expired, a live session with an elapsed deadline is left alone.
func TestCleanStaleLocksSweepsEditorLeases(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	ctx := context.Background()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	bootID := ""
	if host, hostErr := (hostsession.DefaultProvider{}).Current(ctx, ""); hostErr == nil {
		bootID = host.BootID
	}
	if _, err := store.CreateEditorLease(ctx, scenarioruntime.EditorLease{SessionID: "dead-session", Agent: "codex", PID: 4000000, HostBootID: bootID, WorkingDir: root}, time.Nanosecond); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateEditorLease(ctx, scenarioruntime.EditorLease{SessionID: "live-session", Agent: "claude", PID: os.Getpid(), HostBootID: bootID, WorkingDir: root}, time.Nanosecond); err != nil {
		t.Fatal(err)
	}
	stubListenerSnapshot(t, true, nil)
	report, err := NewController(root, home).CleanStaleLocks()
	if err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}
	var sawDead bool
	for _, item := range report.Stopped {
		if item.Name == "agent-session/dead-session" {
			sawDead = true
		}
		if item.Name == "agent-session/live-session" {
			t.Fatal("a live session with an elapsed deadline was expired")
		}
	}
	if !sawDead {
		t.Fatalf("dead session not expired: %+v", report.Stopped)
	}
	active, err := store.ListEditorLeases(ctx, false)
	if err != nil || len(active) != 1 || active[0].SessionID != "live-session" {
		t.Fatalf("active leases = %+v, %v", active, err)
	}
}

// A process inside a vrooli-agents.slice scope is a coding-agent session the
// launcher recorded, whatever its cwd or argv look like; the orphan sweep
// never lists or kills it.
func TestListOrphansExemptsAgentSessionScopes(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	originalListProcessTable := listProcessTableFn
	t.Cleanup(func() { listProcessTableFn = originalListProcessTable })
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		return map[int]processTableEntry{
			5200: {PID: 5200, PPID: 1, PGID: 5200, Command: filepath.Join(root, "scenarios", "beta", "api", "server")},
			6100: {PID: 6100, PPID: 1, PGID: 6100, Command: "sh -c build", Cwd: root, Cgroup: "/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/vrooli-agent-abc.scope"},
		}, nil
	}
	orphans, err := NewController(root, home).ListOrphans()
	if err != nil {
		t.Fatalf("ListOrphans: %v", err)
	}
	if len(orphans) != 1 || orphans[0].PID != 5200 {
		t.Fatalf("orphans = %#v, want only the untracked scenario server", orphans)
	}
	controller := NewController(root, home)
	originalRead := readProcessEntryFn
	t.Cleanup(func() { readProcessEntryFn = originalRead })
	readProcessEntryFn = func(int) (processTableEntry, bool) {
		return processTableEntry{PID: 6100, Command: "sh -c build", Cwd: root, Cgroup: "/vrooli.slice/vrooli-agents.slice/vrooli-agent-abc.scope"}, true
	}
	if controller.stillVrooliOrphan(6100) {
		t.Fatal("the kill-time guard must refuse an agent session scope")
	}
}
