//go:build linux

package platform

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStopScopeContextBoundsManagerAndUsesGuardedFallback(t *testing.T) {
	ref := fixtureCgroup(t, "user.slice/vrooli-test.slice/vrooli-test-render.scope", map[string]string{"cgroup.procs": "42\n"})
	old := stopScopeManager
	t.Cleanup(func() { stopScopeManager = old })
	stopScopeManager = func(ctx context.Context, unit string) ([]byte, error) { <-ctx.Done(); return nil, ctx.Err() }
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	started := time.Now()
	if err := StopScopeContext(ctx, ref); err != nil {
		t.Fatal(err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("manager wait was not bounded")
	}
	b, err := os.ReadFile(filepath.Join(cgroupMount, ref.Path, "cgroup.kill"))
	if err != nil || string(b) != "1" {
		t.Fatalf("guarded fallback missing %q %v", b, err)
	}
}

func TestStopScopeContextRefusesUnownedScopeBeforeManager(t *testing.T) {
	ref := fixtureCgroup(t, "user.slice/app.slice/desktop.scope", map[string]string{"cgroup.procs": "42\n"})
	old := stopScopeManager
	t.Cleanup(func() { stopScopeManager = old })
	called := false
	stopScopeManager = func(context.Context, string) ([]byte, error) { called = true; return nil, nil }
	if err := StopScopeContext(context.Background(), ref); err == nil || called {
		t.Fatal("unowned scope reached manager")
	}
}

// A scope outside the agent and test slices is a supervisor, a resource or
// the operator's desktop: StopScope must refuse it before touching anything.
func TestStopScopeRefusesACgroupOutsideTheAgentSlice(t *testing.T) {
	ref := fixtureCgroup(t, "user.slice/app.slice/editor", map[string]string{"cgroup.procs": "42\n"})
	if err := StopScope(ref); err == nil {
		t.Fatal("StopScope stopped a cgroup outside vrooli-agents.slice")
	}
	if _, err := os.Stat(filepath.Join(cgroupMount, ref.Path, "cgroup.kill")); !os.IsNotExist(err) {
		t.Fatalf("a refused stop must not write cgroup.kill (stat err = %v)", err)
	}
}

// A tree placed by hand (not a manager unit) is ended through cgroup.kill.
func TestStopScopeKillsAHandMadeAgentCgroup(t *testing.T) {
	ref := fixtureCgroup(t, "user.slice/vrooli-agents.slice/vrooli-agent-handmade", map[string]string{"cgroup.procs": "42\n"})
	if err := StopScope(ref); err != nil {
		t.Fatalf("StopScope: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(cgroupMount, ref.Path, "cgroup.kill"))
	if err != nil || string(data) != "1" {
		t.Fatalf("cgroup.kill = %q, %v; want \"1\"", data, err)
	}
}

// A scope that no longer exists is already in the goal state.
func TestStopScopeOfAVanishedScopeSucceeds(t *testing.T) {
	ref := fixtureCgroup(t, "user.slice/vrooli-agents.slice", nil)
	ref.Path += "/vrooli-agent-gone.scope"
	if err := StopScope(ref); err != nil {
		t.Fatalf("StopScope of a vanished scope = %v, want nil", err)
	}
}

func TestScopeOccupancyReadsSwap(t *testing.T) {
	ref := fixtureCgroup(t, "user.slice/vrooli-agents.slice/vrooli-agent-a.scope", map[string]string{
		"memory.current":      "1048576\n",
		"memory.swap.current": "2097152\n",
	})
	got, err := ScopeOccupancy(ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.SwapBytes != 2097152 {
		t.Fatalf("SwapBytes = %d, want 2097152", got.SwapBytes)
	}
}
