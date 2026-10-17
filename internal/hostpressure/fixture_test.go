package hostpressure

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestCapturedFixtureIsReadableAndRanksStrandedMemory(t *testing.T) {
	root := filepath.Join("testdata", "host-2026-08-22")
	var manifest struct {
		Intervals map[string]float64 `json:"intervals_seconds"`
		Fork      struct {
			T0 uint64 `json:"t0"`
			T1 uint64 `json:"t1"`
		} `json:"fork_counters"`
	}
	b, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Intervals["proc_stat"] <= 0 || manifest.Intervals["storage_manager_io"] <= 0 {
		t.Fatal("fixture delta intervals must be positive")
	}
	stat0 := fixtureForkCounter(t, filepath.Join(root, "proc-stat-t0"))
	stat1 := fixtureForkCounter(t, filepath.Join(root, "proc-stat-t1"))
	if stat0 != manifest.Fork.T0 || stat1 != manifest.Fork.T1 {
		t.Fatal("manifest fork counters do not match captured files")
	}
	rate := float64(stat1-stat0) / manifest.Intervals["proc_stat"]
	if rate < 323 || rate > 325 {
		t.Fatalf("expected captured fork rate within one per second of 324, got %f", rate)
	}
	snapshot := Collect(context.Background(), Options{ProcRoot: root})
	for name, reading := range map[string]Reading{
		"cpu pressure":     snapshot.CPUPressure,
		"memory total":     snapshot.MemoryTotal,
		"memory available": snapshot.MemoryAvail,
		"swap total":       snapshot.SwapTotal,
		"swap used":        snapshot.SwapUsed,
	} {
		if _, ok := reading.Number(); !ok {
			t.Fatalf("captured %s must be readable: %+v", name, reading)
		}
	}

	data, err := os.ReadFile(filepath.Join(root, "procs.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	var processes []Process
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 5 {
			continue
		}
		pid, _ := strconv.ParseInt(f[0], 10, 64)
		ppid, _ := strconv.ParseInt(f[2], 10, 64)
		rss, _ := strconv.ParseUint(f[3], 10, 64)
		swap, _ := strconv.ParseUint(f[4], 10, 64)
		processes = append(processes, Process{PID: pid, Name: f[1], PPID: ppid, Resident: rss, Swapped: swap})
	}
	stranded := Stranded(processes, 2)
	if len(stranded) == 0 || !strings.HasPrefix(stranded[0].Name, "reranker") {
		t.Fatalf("expected reranker to rank first, got %#v", stranded[:min(3, len(stranded))])
	}
	if stranded[0].Swapped < 7_000_000_000 {
		t.Fatalf("expected roughly 7GB swapped by reranker, got %d", stranded[0].Swapped)
	}
}

func fixtureForkCounter(t *testing.T, path string) uint64 {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	n, ok := statCounter(string(b), "processes")
	if !ok {
		t.Fatalf("%s has no process counter", path)
	}
	return n
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
