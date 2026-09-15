package hostpressure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestCaptureForkStormFixture records a real fork storm on this host into
// testdata/fixtures/fork-storm. It runs only when HOSTPRESSURE_CAPTURE_FORK_STORM=1:
// it starts a bounded burst (200 sleeps inside a vrooli-test.slice scope with
// TasksMax=512) and captures two snapshots around it. The fixture it writes
// is what TestForkRateFindingCarriesTopParents in cmd/vrooli-watchdog loads.
func TestCaptureForkStormFixture(t *testing.T) {
	if os.Getenv("HOSTPRESSURE_CAPTURE_FORK_STORM") != "1" {
		t.Skip("set HOSTPRESSURE_CAPTURE_FORK_STORM=1 to recapture the fixture")
	}
	if runtime.GOOS != "linux" {
		t.Skip("linux only")
	}
	dir := filepath.Join("testdata", "fixtures", "fork-storm")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	copyProc := func(name, dst string) {
		data, err := os.ReadFile("/proc/" + name)
		if err != nil {
			t.Fatalf("read /proc/%s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, dst), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeProcs := func(list []Process, name string) {
		var b strings.Builder
		for _, p := range list {
			fmt.Fprintf(&b, "%d\t%s\t%d\t%d\t%d\n", p.PID, p.Name, p.PPID, p.Resident, p.Swapped)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	unit := fmt.Sprintf("vrooli-test-burst-%d", os.Getpid())
	t0 := Collect(ctx, Options{})
	stat0, _ := os.ReadFile("/proc/stat")
	started := time.Now()
	cmd := exec.Command("systemd-run", "--user", "--scope", "--quiet", "--slice=vrooli-test.slice", "--unit="+unit, "-p", "TasksMax=512",
		"sh", "-c", "for i in $(seq 1 200); do sleep 20 & done; wait")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start burst: %v", err)
	}
	defer func() {
		_ = exec.Command("systemctl", "--user", "stop", unit+".scope").Run()
		_ = cmd.Wait()
	}()
	// Wait until the burst's sleeps are visible, then capture t1 at once so
	// the measured interval is the burst, not the wait.
	var t1 PressureSnapshot
	deadline := time.Now().Add(10 * time.Second)
	for {
		t1 = Collect(ctx, Options{Previous: &t0})
		if countChildrenNamed(t1.Processes, "sleep") >= 200 || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	stat1, _ := os.ReadFile("/proc/stat")
	elapsed := time.Since(started)
	if err := os.WriteFile(filepath.Join(dir, "proc-stat-t0"), stat0, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "proc-stat-t1"), stat1, 0o644); err != nil {
		t.Fatal(err)
	}
	copyProc("loadavg", "proc-loadavg")
	copyProc("meminfo", "proc-meminfo")
	copyProc("pressure/cpu", "proc-pressure-cpu")
	copyProc("pressure/memory", "proc-pressure-memory")
	copyProc("pressure/io", "proc-pressure-io")
	writeProcs(t0.Processes, "procs-t0.tsv")
	writeProcs(t1.Processes, "procs.tsv")
	f0, _ := statCounter(string(stat0), "processes")
	f1, _ := statCounter(string(stat1), "processes")
	host, _ := os.Hostname()
	manifest := map[string]any{
		"capture_timestamp": started.UTC().Format(time.RFC3339),
		"host_name":         host,
		"core_count":        runtime.NumCPU(),
		"intervals_seconds": map[string]float64{"proc_stat": elapsed.Seconds()},
		"fork_counters":     map[string]uint64{"t0": f0, "t1": f1},
		"burst":             "systemd-run --user --scope --slice=vrooli-test.slice -p TasksMax=512 sh -c 'for i in $(seq 1 200); do sleep 20 & done; wait'",
	}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	readme := "# fork-storm fixture\n\nCaptured on " + host + " (" + strconv.Itoa(runtime.NumCPU()) + " cores) at " + started.UTC().Format(time.RFC3339) +
		" by `HOSTPRESSURE_CAPTURE_FORK_STORM=1 go test ./internal/hostpressure/ -run TestCaptureForkStormFixture` during a bounded burst:\n\n" +
		"    systemd-run --user --scope --slice=vrooli-test.slice -p TasksMax=512 sh -c 'for i in $(seq 1 200); do sleep 20 & done; wait'\n\n" +
		"proc-stat-t0/t1 bracket the burst (" + elapsed.String() + " apart, recorded in manifest.json); procs-t0.tsv is the process tree before it and procs.tsv the tree during it, so `hostpressure.Attribution` ranks the burst's `sh` by child count and by delta.\n"
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("captured %d processes at t0 and %d at t1 in %s; fork delta %d", len(t0.Processes), len(t1.Processes), elapsed, f1-f0)
}

func countChildrenNamed(list []Process, name string) int {
	n := 0
	for _, p := range list {
		if p.Name == name {
			n++
		}
	}
	return n
}
