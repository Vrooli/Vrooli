package backlogrank

import (
	"testing"
	"time"
)

func TestComputeDepthMap_DependencyDepthBeatsPriority(t *testing.T) {
	items := []Item{
		{Kind: "idea", Name: "base", Status: "backlog", Priority: 5},
		{Kind: "idea", Name: "child", Status: "backlog", Priority: 1, DependsOn: []string{"idea/base"}},
	}

	depths := ComputeDepthMap(items)
	if got := depths["idea/base"]; got != 0 {
		t.Fatalf("base depth = %d, want 0", got)
	}
	if got := depths["idea/child"]; got != 1 {
		t.Fatalf("child depth = %d, want 1", got)
	}
}

func TestComputeDepthMap_ResolvedDependenciesIgnored(t *testing.T) {
	items := []Item{
		{Kind: "idea", Name: "done", Status: "completed"},
		{Kind: "idea", Name: "child", Status: "backlog", DependsOn: []string{"idea/done"}},
	}

	depths := ComputeDepthMap(items)
	if got := depths["idea/child"]; got != 0 {
		t.Fatalf("child depth = %d, want 0", got)
	}
}

func TestComputeDepthMap_CycleStabilizes(t *testing.T) {
	items := []Item{
		{Kind: "idea", Name: "a", Status: "backlog", DependsOn: []string{"idea/b"}},
		{Kind: "idea", Name: "b", Status: "backlog", DependsOn: []string{"idea/a"}},
	}

	depths := ComputeDepthMap(items)
	if depths["idea/a"] == 0 || depths["idea/b"] == 0 {
		t.Fatalf("cycle depths should be non-zero: %+v", depths)
	}
}

func TestComputeUnblockingMap_CountsTransitiveDependents(t *testing.T) {
	items := []Item{
		{Kind: "idea", Name: "root", Status: "backlog"},
		{Kind: "idea", Name: "mid", Status: "backlog", DependsOn: []string{"idea/root"}},
		{Kind: "idea", Name: "leaf", Status: "backlog", DependsOn: []string{"idea/mid"}},
	}

	unblocking := ComputeUnblockingMap(items)
	if got := unblocking["idea/root"]; got != 2 {
		t.Fatalf("root unblocking = %d, want 2", got)
	}
	if got := unblocking["idea/mid"]; got != 1 {
		t.Fatalf("mid unblocking = %d, want 1", got)
	}
}

func TestEffectivePriority_CapsBoost(t *testing.T) {
	got := EffectivePriority(7, 99)
	want := 4.0
	if got != want {
		t.Fatalf("EffectivePriority = %v, want %v", got, want)
	}
}

func TestLess_UsesRecencyThenKindName(t *testing.T) {
	now := time.Now().UTC()
	older := now.Add(-time.Hour)
	depths := map[string]int{"idea/a": 0, "fix/a": 0}
	unblocking := map[string]int{"idea/a": 0, "fix/a": 0}

	if !Less(
		Item{Kind: "idea", Name: "a", Priority: 3, UpdatedAt: now},
		Item{Kind: "fix", Name: "a", Priority: 3, UpdatedAt: older},
		depths, unblocking,
	) {
		t.Fatal("newer item should sort first")
	}

	if !Less(
		Item{Kind: "fix", Name: "a", Priority: 3, UpdatedAt: now},
		Item{Kind: "idea", Name: "a", Priority: 3, UpdatedAt: now},
		depths, unblocking,
	) {
		t.Fatal("kind should break ties when recency matches")
	}
}
