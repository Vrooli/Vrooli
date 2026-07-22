package goals

import (
	"reflect"
	"testing"
)

func TestComputeScope_ClosureAndProgress(t *testing.T) {
	in := ScopeInput{
		Targets: []string{"execute/a"},
		ItemDeps: map[string][]string{
			"execute/a": {"execute/b"},
			"execute/b": nil,
			"execute/c": nil, // unrelated, must not enter closure
		},
		ItemStatus: map[string]string{
			"execute/a": "ready",
			"execute/b": "completed",
			"execute/c": "backlog",
		},
	}
	got := ComputeScope(in)
	if !reflect.DeepEqual(got.Closure, []string{"execute/a", "execute/b"}) {
		t.Fatalf("closure = %v, want [execute/a execute/b]", got.Closure)
	}
	if got.Total != 2 || got.CompletedCount != 1 {
		t.Fatalf("Total/Completed = %d/%d, want 2/1", got.Total, got.CompletedCount)
	}
	if got.ProgressPct != 50 {
		t.Fatalf("ProgressPct = %v, want 50", got.ProgressPct)
	}
	// b done => a is ready; b is completed.
	if !reflect.DeepEqual(got.Ready, []string{"execute/a"}) {
		t.Fatalf("Ready = %v, want [execute/a]", got.Ready)
	}
	if len(got.Blocked) != 0 {
		t.Fatalf("Blocked = %v, want none", got.Blocked)
	}
}

func TestComputeScope_MilestonePartitionLeavesClosureItemsUnassigned(t *testing.T) {
	in := ScopeInput{
		Targets: []string{"execute/a", "execute/d"},
		ItemDeps: map[string][]string{
			"execute/a": {"execute/b"},
			"execute/b": nil,
			"execute/c": nil,
			"execute/d": nil,
		},
		ItemStatus: map[string]string{
			"execute/a": "ready",
			"execute/b": "completed",
			"execute/c": "backlog",
			"execute/d": "backlog",
		},
		Milestones: []Milestone{
			{Name: "build", Title: "Build", Items: []string{"execute/a", "execute/c"}},
			{Name: "verify", Title: "Verify", Items: []string{"execute/b"}, DependsOn: []string{"build"}},
		},
	}
	got := ComputeScope(in)
	if len(got.Milestones) != 2 || got.Milestones[0].ReadyCount != 1 || got.Milestones[1].CompletedCount != 1 {
		t.Fatalf("milestone rollups = %+v", got.Milestones)
	}
	if !reflect.DeepEqual(got.Milestones[0].Items, []string{"execute/a"}) {
		t.Fatalf("first milestone scope items = %v, want only execute/a", got.Milestones[0].Items)
	}
	if !reflect.DeepEqual(got.Unassigned, []string{"execute/d"}) {
		t.Fatalf("unassigned = %v, want [execute/d]", got.Unassigned)
	}
}

func TestComputeScope_MilestoneDependencyGatesReadiness(t *testing.T) {
	in := ScopeInput{
		Targets:    []string{"execute/release", "execute/build"},
		ItemDeps:   map[string][]string{"execute/build": nil, "execute/release": nil},
		ItemStatus: map[string]string{"execute/build": "completed", "execute/release": "ready"},
		Milestones: []Milestone{
			{Name: "build", Title: "Build", Items: []string{"execute/build"}},
			{Name: "release", Title: "Release", Items: []string{"execute/release"}, DependsOn: []string{"build"}},
		},
	}
	if got := ComputeScope(in); !reflect.DeepEqual(got.Ready, []string{"execute/release"}) {
		t.Fatalf("ready after completed predecessor = %v, want [execute/release]", got.Ready)
	}
	in.ItemStatus["execute/build"] = "ready"
	got := ComputeScope(in)
	if !reflect.DeepEqual(got.Blocked, []string{"execute/release"}) || !reflect.DeepEqual(got.Ready, []string{"execute/build"}) {
		t.Fatalf("incomplete predecessor should block only release, got blocked=%v ready=%v", got.Blocked, got.Ready)
	}
}

func TestComputeScope_CycleTerminatesAndBlocks(t *testing.T) {
	in := ScopeInput{
		Targets: []string{"execute/a"},
		ItemDeps: map[string][]string{
			"execute/a": {"execute/b"},
			"execute/b": {"execute/a"}, // cycle
		},
		ItemStatus: map[string]string{
			"execute/a": "ready",
			"execute/b": "ready",
		},
	}
	got := ComputeScope(in) // must terminate
	// Both are cycle-trapped => neither ready.
	if len(got.Ready) != 0 {
		t.Fatalf("cycle items should not be ready, got Ready=%v", got.Ready)
	}
	if got.BlockedCount != 2 {
		t.Fatalf("both cycle items should be blocked, got %d", got.BlockedCount)
	}
}
