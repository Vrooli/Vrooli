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

// A dropped prerequisite must release its dependents. Before `dropped` existed,
// abandoned work sat at `backlog` forever and every item behind it stayed in
// `blocked` permanently — invisible to the goal-priority drain, which only ever
// picks up ready items.
func TestComputeScope_DroppedPrerequisiteUnblocksDependents(t *testing.T) {
	in := ScopeInput{
		Targets: []string{"execute/a"},
		ItemDeps: map[string][]string{
			"execute/a": {"execute/b"},
			"execute/b": nil,
		},
		ItemStatus: map[string]string{
			"execute/a": "backlog",
			"execute/b": "dropped",
		},
	}
	got := ComputeScope(in)
	if !reflect.DeepEqual(got.Ready, []string{"execute/a"}) {
		t.Fatalf("Ready = %v, want [execute/a] — a dropped prerequisite must not block", got.Ready)
	}
	if len(got.Blocked) != 0 {
		t.Fatalf("Blocked = %v, want none", got.Blocked)
	}
	if !reflect.DeepEqual(got.Dropped, []string{"execute/b"}) {
		t.Fatalf("Dropped = %v, want [execute/b]", got.Dropped)
	}
	// b is out of scope entirely: 0 of the 1 remaining item is done.
	if got.CompletedCount != 0 || got.DroppedCount != 1 || got.ProgressPct != 0 {
		t.Fatalf("completed/dropped/pct = %d/%d/%v, want 0/1/0",
			got.CompletedCount, got.DroppedCount, got.ProgressPct)
	}
}

// Dropping the remainder of a goal must let it reach 100%, without counting the
// abandoned work as an achievement.
func TestComputeScope_DroppedLeavesProgressDenominator(t *testing.T) {
	in := ScopeInput{
		Targets: []string{"execute/a", "execute/b", "execute/c"},
		ItemDeps: map[string][]string{
			"execute/a": nil,
			"execute/b": nil,
			"execute/c": nil,
		},
		ItemStatus: map[string]string{
			"execute/a": "completed",
			"execute/b": "completed",
			"execute/c": "dropped",
		},
	}
	got := ComputeScope(in)
	if got.Total != 3 {
		t.Fatalf("Total = %d, want 3 — the closure still describes 3 items", got.Total)
	}
	if got.ProgressPct != 100 {
		t.Fatalf("ProgressPct = %v, want 100 — 2 of the 2 items still in scope are done", got.ProgressPct)
	}
	if got.CompletedCount != 2 {
		t.Fatalf("CompletedCount = %d, want 2 — dropped work is not an achievement", got.CompletedCount)
	}
}

// A milestone whose remaining items were all dropped is settled, and must stop
// gating the milestones sequenced behind it.
func TestComputeScope_AllDroppedMilestoneStopsGatingSuccessors(t *testing.T) {
	in := ScopeInput{
		Targets: []string{"execute/a", "execute/b"},
		ItemDeps: map[string][]string{
			"execute/a": nil,
			"execute/b": nil,
		},
		ItemStatus: map[string]string{
			"execute/a": "dropped",
			"execute/b": "backlog",
		},
		Milestones: []Milestone{
			{Name: "m1", Items: []string{"execute/a"}},
			{Name: "m2", Items: []string{"execute/b"}, DependsOn: []string{"m1"}},
		},
	}
	got := ComputeScope(in)
	if !reflect.DeepEqual(got.Ready, []string{"execute/b"}) {
		t.Fatalf("Ready = %v, want [execute/b] — m1 is settled, so m2 is open", got.Ready)
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
