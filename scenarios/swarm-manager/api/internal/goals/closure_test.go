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

func TestComputeScope_InitiativeTargetExpands(t *testing.T) {
	in := ScopeInput{
		Targets: []string{"initiative/alpha"},
		ItemDeps: map[string][]string{
			"execute/x":   {"execute/dep"},
			"execute/y":   nil,
			"execute/dep": nil,
		},
		ItemStatus: map[string]string{
			"execute/x":   "backlog",
			"execute/y":   "completed",
			"execute/dep": "completed",
		},
		InitiativeItems: map[string][]string{
			"alpha": {"execute/x", "execute/y"},
		},
	}
	got := ComputeScope(in)
	want := []string{"execute/dep", "execute/x", "execute/y"}
	if !reflect.DeepEqual(got.Closure, want) {
		t.Fatalf("closure = %v, want %v", got.Closure, want)
	}
	if got.CompletedCount != 2 { // y + dep
		t.Fatalf("CompletedCount = %d, want 2", got.CompletedCount)
	}
}

func TestComputeScope_InitiativeGate(t *testing.T) {
	// Initiative A (item a1) depends on initiative B (item b1).
	base := func(b1Status string) ScopeInput {
		return ScopeInput{
			Targets: []string{"initiative/A"},
			ItemDeps: map[string][]string{
				"execute/a1": nil,
				"execute/b1": nil,
			},
			ItemStatus: map[string]string{
				"execute/a1": "ready",
				"execute/b1": b1Status,
			},
			InitiativeItems: map[string][]string{
				"A": {"execute/a1"},
				"B": {"execute/b1"},
			},
			InitiativeDeps: map[string][]string{
				"A": {"B"},
			},
		}
	}

	// B incomplete => a1 is gate-blocked.
	blocked := ComputeScope(base("in_progress"))
	if !reflect.DeepEqual(blocked.Blocked, []string{"execute/a1"}) {
		t.Fatalf("expected a1 gate-blocked, got Blocked=%v Ready=%v", blocked.Blocked, blocked.Ready)
	}

	// B complete => a1 becomes ready.
	ready := ComputeScope(base("completed"))
	if !reflect.DeepEqual(ready.Ready, []string{"execute/a1"}) {
		t.Fatalf("expected a1 ready once B complete, got Ready=%v Blocked=%v", ready.Ready, ready.Blocked)
	}
}

func TestComputeScope_MixedItemDependsOnInitiative(t *testing.T) {
	// An item depends directly on an initiative (mixed edge, item→initiative).
	in := ScopeInput{
		Targets: []string{"execute/consumer"},
		ItemDeps: map[string][]string{
			"execute/consumer": {"initiative/prov"},
			"execute/p1":       nil,
		},
		ItemStatus: map[string]string{
			"execute/consumer": "ready",
			"execute/p1":       "in_progress",
		},
		InitiativeItems: map[string][]string{
			"prov": {"execute/p1"},
		},
	}
	blocked := ComputeScope(in)
	if !reflect.DeepEqual(blocked.Blocked, []string{"execute/consumer"}) {
		t.Fatalf("consumer should be blocked on initiative prov, got Blocked=%v Ready=%v", blocked.Blocked, blocked.Ready)
	}

	in.ItemStatus["execute/p1"] = "completed"
	ready := ComputeScope(in)
	if !reflect.DeepEqual(ready.Ready, []string{"execute/consumer"}) {
		t.Fatalf("consumer should be ready once prov complete, got Ready=%v", ready.Ready)
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
