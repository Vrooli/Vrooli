package backlog

import (
	"path/filepath"
	"testing"

	"swarm-manager/internal/testutil"
)

// ---------------------------------------------------------------------------
// EvaluateDependencyBlocking unit tests
// ---------------------------------------------------------------------------

func setupKindDirs(t *testing.T, rootDir string) {
	t.Helper()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
}

func TestEvaluateDependencyBlocking_NoDeps(t *testing.T) {
	rootDir := t.TempDir()
	store := NewFileStore(rootDir)

	item := BacklogItem{Name: "no-deps", Kind: KindIdea, Status: StatusBacklog}
	reasons, err := EvaluateDependencyBlocking(item, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reasons) != 0 {
		t.Fatalf("expected no blocking reasons, got %d", len(reasons))
	}
}

func TestEvaluateDependencyBlocking_BacklogDepBlocks(t *testing.T) {
	rootDir := t.TempDir()
	store := NewFileStore(rootDir)
	setupKindDirs(t, rootDir)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "dep-item", Title: "Dep", Status: StatusBacklog, Priority: 5,
		Tags: []string{}, Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	item := BacklogItem{
		Name: "child", Kind: KindIdea, Status: StatusBacklog,
		DependsOn: []string{"idea/dep-item"},
	}
	reasons, err := EvaluateDependencyBlocking(item, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reasons) != 1 {
		t.Fatalf("expected 1 blocking reason, got %d", len(reasons))
	}
	if !reasons[0].Forceable {
		t.Error("expected dependency blocking reason to be forceable")
	}
}

func TestEvaluateDependencyBlocking_ResearchingDepBlocks(t *testing.T) {
	rootDir := t.TempDir()
	store := NewFileStore(rootDir)
	setupKindDirs(t, rootDir)

	createTestItem(t, rootDir, KindFix, BacklogItem{
		Name: "researching-dep", Title: "Researching Dep", Status: StatusResearching, Priority: 5,
		Tags: []string{}, Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	item := BacklogItem{
		Name: "child", Kind: KindIdea, Status: StatusBacklog,
		DependsOn: []string{"fix/researching-dep"},
	}
	reasons, err := EvaluateDependencyBlocking(item, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reasons) != 1 {
		t.Fatalf("expected 1 blocking reason, got %d", len(reasons))
	}
	if !reasons[0].Forceable {
		t.Error("expected dependency blocking reason to be forceable")
	}
}

func TestEvaluateDependencyBlocking_ReadyDepPasses(t *testing.T) {
	rootDir := t.TempDir()
	store := NewFileStore(rootDir)
	setupKindDirs(t, rootDir)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ready-dep", Title: "Ready Dep", Status: StatusReady, Priority: 5,
		Tags: []string{}, Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	item := BacklogItem{
		Name: "child", Kind: KindIdea, Status: StatusBacklog,
		DependsOn: []string{"idea/ready-dep"},
	}
	reasons, err := EvaluateDependencyBlocking(item, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reasons) != 0 {
		t.Fatalf("expected no blocking reasons, got %d", len(reasons))
	}
}

func TestEvaluateDependencyBlocking_CompletedDepPasses(t *testing.T) {
	rootDir := t.TempDir()
	store := NewFileStore(rootDir)
	setupKindDirs(t, rootDir)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "completed-dep", Title: "Completed Dep", Status: StatusCompleted, Priority: 5,
		Tags: []string{}, Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	item := BacklogItem{
		Name: "child", Kind: KindIdea, Status: StatusBacklog,
		DependsOn: []string{"idea/completed-dep"},
	}
	reasons, err := EvaluateDependencyBlocking(item, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reasons) != 0 {
		t.Fatalf("expected no blocking reasons, got %d", len(reasons))
	}
}

func TestEvaluateDependencyBlocking_MissingDepFailOpen(t *testing.T) {
	rootDir := t.TempDir()
	store := NewFileStore(rootDir)
	setupKindDirs(t, rootDir)

	item := BacklogItem{
		Name: "child", Kind: KindIdea, Status: StatusBacklog,
		DependsOn: []string{"idea/deleted-dep"},
	}
	reasons, err := EvaluateDependencyBlocking(item, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reasons) != 0 {
		t.Fatalf("expected no blocking reasons for missing dep (fail-open), got %d", len(reasons))
	}
}

func TestEvaluateDependencyBlocking_AllReasonsForceable(t *testing.T) {
	rootDir := t.TempDir()
	store := NewFileStore(rootDir)
	setupKindDirs(t, rootDir)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "dep-a", Title: "Dep A", Status: StatusBacklog, Priority: 5,
		Tags: []string{}, Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})
	createTestItem(t, rootDir, KindFix, BacklogItem{
		Name: "dep-b", Title: "Dep B", Status: StatusResearching, Priority: 5,
		Tags: []string{}, Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	item := BacklogItem{
		Name: "child", Kind: KindIdea, Status: StatusBacklog,
		DependsOn: []string{"idea/dep-a", "fix/dep-b"},
	}
	reasons, err := EvaluateDependencyBlocking(item, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reasons) == 0 {
		t.Fatal("expected at least one blocking reason")
	}
	if !AllForceable(reasons) {
		t.Error("expected all dependency blocking reasons to be forceable")
	}
}

// ---------------------------------------------------------------------------
// HasNonForceableReasons / AllForceable
// ---------------------------------------------------------------------------

func TestHasNonForceableReasons(t *testing.T) {
	tests := []struct {
		name    string
		reasons []BlockingReason
		want    bool
	}{
		{"empty", nil, false},
		{"all forceable", []BlockingReason{{Message: "a", Forceable: true}}, false},
		{"one non-forceable", []BlockingReason{
			{Message: "a", Forceable: true},
			{Message: "b", Forceable: false},
		}, true},
		{"all non-forceable", []BlockingReason{{Message: "a", Forceable: false}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := HasNonForceableReasons(tc.reasons)
			if got != tc.want {
				t.Errorf("HasNonForceableReasons = %v, want %v", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DedupeReasons
// ---------------------------------------------------------------------------

func TestDedupeReasons(t *testing.T) {
	reasons := []BlockingReason{
		{Message: "dup", Forceable: true},
		{Message: "dup", Forceable: true},
		{Message: "unique", Forceable: false},
		{Message: "  ", Forceable: true},
		{Message: "", Forceable: false},
		{Message: "dup", Forceable: false}, // same message, different forceable — still deduped
	}
	result := DedupeReasons(reasons)

	if len(result) != 2 {
		t.Fatalf("expected 2 deduped reasons, got %d: %+v", len(result), result)
	}
	if result[0].Message != "dup" {
		t.Errorf("expected first reason 'dup', got %q", result[0].Message)
	}
	if !result[0].Forceable {
		t.Error("expected first occurrence's Forceable=true to be preserved")
	}
	if result[1].Message != "unique" {
		t.Errorf("expected second reason 'unique', got %q", result[1].Message)
	}
}

func TestDedupeReasons_NilInput(t *testing.T) {
	result := DedupeReasons(nil)
	if result != nil {
		t.Fatalf("expected nil result for nil input, got %+v", result)
	}
}

// ---------------------------------------------------------------------------
// ComputeListBlockingInfo
// ---------------------------------------------------------------------------

func TestComputeListBlockingInfo_BasicBlocking(t *testing.T) {
	items := []BacklogItem{
		{Name: "dep-item", Kind: KindIdea, Status: StatusBacklog},
		{Name: "child-item", Kind: KindFix, Status: StatusBacklog, DependsOn: []string{"idea/dep-item"}},
	}

	result := ComputeListBlockingInfo(items)

	info, found := result["fix/child-item"]
	if !found {
		t.Fatal("expected blocking info for fix/child-item")
	}
	if !info.Blocked {
		t.Error("expected Blocked=true")
	}
	if len(info.BlockingDepKeys) != 1 || info.BlockingDepKeys[0] != "idea/dep-item" {
		t.Errorf("expected BlockingDepKeys=[idea/dep-item], got %v", info.BlockingDepKeys)
	}
	if !info.AllForceable {
		t.Error("expected AllForceable=true")
	}
}

func TestComputeListBlockingInfo_NoDepsOmitted(t *testing.T) {
	items := []BacklogItem{
		{Name: "no-deps", Kind: KindIdea, Status: StatusBacklog},
	}

	result := ComputeListBlockingInfo(items)

	if _, found := result["idea/no-deps"]; found {
		t.Error("items without deps should not appear in the blocking map")
	}
}

func TestComputeListBlockingInfo_DepNotBlocking(t *testing.T) {
	items := []BacklogItem{
		{Name: "ready-dep", Kind: KindIdea, Status: StatusReady},
		{Name: "child-item", Kind: KindFix, Status: StatusBacklog, DependsOn: []string{"idea/ready-dep"}},
	}

	result := ComputeListBlockingInfo(items)

	if _, found := result["fix/child-item"]; found {
		t.Error("child with ready dep should not be in blocking map")
	}
}

func TestComputeListBlockingInfo_MissingDepNotBlocking(t *testing.T) {
	items := []BacklogItem{
		{Name: "child-item", Kind: KindFix, Status: StatusBacklog, DependsOn: []string{"idea/nonexistent"}},
	}

	result := ComputeListBlockingInfo(items)

	if _, found := result["fix/child-item"]; found {
		t.Error("child with missing dep reference should not be blocked (fail-open)")
	}
}

func TestComputeListBlockingInfo_ArchivedDepNotBlocking(t *testing.T) {
	archivedAt := "2026-01-02T00:00:00Z"
	items := []BacklogItem{
		{Name: "archived-dep", Kind: KindIdea, Status: StatusBacklog, ArchivedAt: &archivedAt},
		{Name: "child-item", Kind: KindFix, Status: StatusBacklog, DependsOn: []string{"idea/archived-dep"}},
	}

	result := ComputeListBlockingInfo(items)

	if _, found := result["fix/child-item"]; found {
		t.Error("child with archived dependency should not be blocked")
	}
}

func TestComputeListBlockingInfo_ArchivedChildOmitted(t *testing.T) {
	archivedAt := "2026-01-02T00:00:00Z"
	items := []BacklogItem{
		{Name: "dep-item", Kind: KindIdea, Status: StatusBacklog},
		{Name: "archived-child", Kind: KindFix, Status: StatusBacklog, ArchivedAt: &archivedAt, DependsOn: []string{"idea/dep-item"}},
	}

	result := ComputeListBlockingInfo(items)

	if _, found := result["fix/archived-child"]; found {
		t.Error("archived child should not appear in blocking map")
	}
}
