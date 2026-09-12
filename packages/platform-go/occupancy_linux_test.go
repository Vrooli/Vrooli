//go:build linux

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

// fixtureCgroup writes a cgroup v2 directory under a temporary mount and
// points the readers at it. The files are the kernel's own formats.
func fixtureCgroup(t *testing.T, rel string, files map[string]string) ScopeRef {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("fixture dir: %v", err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("fixture %s: %v", name, err)
		}
	}
	previous := cgroupMount
	cgroupMount = root
	t.Cleanup(func() { cgroupMount = previous })
	return ScopeRef{Kind: ScopeKindCgroup, Path: "/" + rel}
}

func TestScopeOccupancyReadsCeilingAndUse(t *testing.T) {
	ref := fixtureCgroup(t, "user.slice/vrooli-agents.slice", map[string]string{
		"pids.current":   "4059\n",
		"pids.max":       "4096\n",
		"memory.current": "23077347328\n",
		"memory.high":    "32716079104\n",
		"memory.max":     "39259295744\n",
		"memory.events":  "low 0\nhigh 4747008\nmax 0\noom 0\noom_kill 0\n",
	})
	got, err := ScopeOccupancy(ref)
	if err != nil {
		t.Fatalf("occupancy: %v", err)
	}
	if got.Tasks != 4059 || got.TasksMax != 4096 {
		t.Fatalf("tasks: got %d/%d, want 4059/4096", got.Tasks, got.TasksMax)
	}
	if got.MemoryHighEvents != 4747008 {
		t.Fatalf("memory high events: got %d, want 4747008", got.MemoryHighEvents)
	}
	if saturation := got.TaskSaturation(); saturation < 0.99 {
		t.Fatalf("task saturation: got %f, want the near-full reading the 2026-09-04 outage had", saturation)
	}
}

// A ceiling systemd reports as "max" is unlimited, and an unlimited ceiling
// has no saturation: a caller must not read it as full.
func TestOccupancyUnlimitedCeilingIsNotSaturated(t *testing.T) {
	ref := fixtureCgroup(t, "user.slice/free.slice", map[string]string{
		"pids.current": "12\n",
		"pids.max":     "max\n",
	})
	got, err := ScopeOccupancy(ref)
	if err != nil {
		t.Fatalf("occupancy: %v", err)
	}
	if got.TasksMax != Unlimited {
		t.Fatalf("tasks max: got %d, want Unlimited", got.TasksMax)
	}
	if got.TaskSaturation() != SaturationUnknown {
		t.Fatalf("saturation of an unlimited ceiling: got %f, want SaturationUnknown", got.TaskSaturation())
	}
}

// An unreadable file is Unknown, never zero: zero use against a real ceiling
// is the reading that would make a wedged slice look healthy.
func TestOccupancyMissingFileIsUnknownNotZero(t *testing.T) {
	ref := fixtureCgroup(t, "user.slice/partial.slice", map[string]string{
		"pids.max": "4096\n",
	})
	got, err := ScopeOccupancy(ref)
	if err != nil {
		t.Fatalf("occupancy: %v", err)
	}
	if got.Tasks != Unknown {
		t.Fatalf("tasks: got %d, want Unknown", got.Tasks)
	}
	if got.TaskSaturation() != SaturationUnknown {
		t.Fatalf("saturation: got %f, want SaturationUnknown", got.TaskSaturation())
	}
}

func TestScopeProcessesReadsCgroupProcs(t *testing.T) {
	ref := fixtureCgroup(t, "user.slice/vrooli-agents.slice/vrooli-agent-codex-abc.scope", map[string]string{
		"cgroup.procs": "101\n202\n\n303\n",
	})
	pids, err := ScopeProcesses(ref)
	if err != nil {
		t.Fatalf("processes: %v", err)
	}
	if len(pids) != 3 || pids[0] != 101 || pids[2] != 303 {
		t.Fatalf("pids: got %v, want [101 202 303]", pids)
	}
}

// Reading is not freezing: occupancy must work for the slice itself and for
// a supervisor scope, which freezableCgroup refuses.
func TestOccupancyReadsOutsideTheFreezableSlices(t *testing.T) {
	ref := fixtureCgroup(t, "system.slice/vrooli-supervisor.service", map[string]string{
		"pids.current": "9\n",
		"pids.max":     "512\n",
	})
	if _, err := ScopeOccupancy(ref); err != nil {
		t.Fatalf("occupancy of a non-agent cgroup must be readable: %v", err)
	}
	if err := FreezeScope(ref); err == nil {
		t.Fatal("freezing a supervisor scope must still be refused")
	}
}

// A traversal path is clamped inside the mount rather than followed out of
// it: the reader can be pointed at a cgroup, never at the filesystem.
func TestOccupancyClampsTraversalInsideTheMount(t *testing.T) {
	ref := fixtureCgroup(t, "user.slice/clamped.slice", map[string]string{"pids.max": "7\n"})
	root := cgroupMount
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "pids.max"), []byte("3\n"), 0o644); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	ref.Path = "/user.slice/../../etc"
	got, err := ScopeOccupancy(ref)
	if err != nil {
		t.Fatalf("occupancy: %v", err)
	}
	if got.Path != "/etc" || got.TasksMax != 3 {
		t.Fatalf("traversal must resolve inside the mount: got path %q max %d", got.Path, got.TasksMax)
	}
}
