package sessioncontext

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/operations"
)

type fakeBriefingBuilder struct {
	briefing *operations.OperationsBriefing
	err      error
}

func (f fakeBriefingBuilder) Build(context.Context, operations.Filters) (*operations.OperationsBriefing, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.briefing, nil
}

type fakeSnapshotProvider struct {
	snapshot *operations.OperationsSnapshot
	err      error
}

func (f fakeSnapshotProvider) GetSnapshot(context.Context) (*operations.OperationsSnapshot, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.snapshot, nil
}

func minimalBriefing() *operations.OperationsBriefing {
	return &operations.OperationsBriefing{
		GeneratedAt:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		FreshnessSeconds: 120,
		WindowSeconds:    10800,
		Summary:          operations.OperationsBriefingSummary{ActiveInitiatives: 2, BlockedItems: 1},
	}
}

func TestOperationsStartupBriefWeavesRankedSnapshot(t *testing.T) {
	r := NewResolver("/tmp/scenario", "/tmp/scenarios", nil, fakeBriefingBuilder{briefing: minimalBriefing()})
	r.SetSnapshotBuilder(fakeSnapshotProvider{snapshot: &operations.OperationsSnapshot{
		GeneratedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Initiatives: []operations.RankedInitiative{
			{Name: "ship-cockpit", Title: "Ship the cockpit", Priority: 1, Readiness: operations.ReadinessReady, DownstreamUnblocks: 3},
			{Name: "later-thing", Title: "Later thing", Priority: 0, Readiness: operations.ReadinessBlocked, DownstreamUnblocks: 0},
		},
		Summary: operations.SnapshotSummary{TotalInitiatives: 2, ReadyInitiatives: 1, BlockedInitiatives: 1},
	}})

	item, err := r.operationsStartupBrief(context.Background(), agentsessions.ContextLimits{MaxSummaryRunes: 4000})
	if err != nil {
		t.Fatalf("operationsStartupBrief: %v", err)
	}

	// Summary carries the typed reference span the UI linkifies.
	if !strings.Contains(item.Summary, "`initiative:ship-cockpit`") {
		t.Errorf("summary missing typed initiative reference; got:\n%s", item.Summary)
	}
	if !strings.Contains(item.Summary, "Ranked initiatives") {
		t.Errorf("summary missing ranked initiatives section; got:\n%s", item.Summary)
	}

	// Metadata carries the structured ranked rows with refs.
	var meta startupBriefMetadata
	if err := json.Unmarshal([]byte(item.MetadataJSON), &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if len(meta.RankedInitiatives) != 2 {
		t.Fatalf("expected 2 ranked initiatives in metadata, got %d", len(meta.RankedInitiatives))
	}
	if meta.RankedInitiatives[0].Ref != "initiative:ship-cockpit" {
		t.Errorf("first ranked ref = %q, want initiative:ship-cockpit", meta.RankedInitiatives[0].Ref)
	}
	if meta.SourceCounts["ranked_initiatives"] != 2 {
		t.Errorf("source_counts.ranked_initiatives = %d, want 2", meta.SourceCounts["ranked_initiatives"])
	}
}

func TestWorkflowAuthoringStartupBriefStatesBoundaryAndReferences(t *testing.T) {
	r := NewResolver("/tmp/scenario", "/tmp/scenarios", nil)
	item, err := r.ResolveSessionStartupBrief(context.Background(), agentsessions.KindWorkflowAuthoring, agentsessions.ContextLimits{MaxSummaryRunes: 4000})
	if err != nil {
		t.Fatalf("ResolveSessionStartupBrief: %v", err)
	}
	if item.Ref != agentsessions.StartupBriefWorkflowAuthoringRef {
		t.Fatalf("ref = %q, want %q", item.Ref, agentsessions.StartupBriefWorkflowAuthoringRef)
	}
	if !strings.Contains(item.Summary, "human-led design conversation") || !strings.Contains(item.Summary, "Do not treat this session as permission") {
		t.Fatalf("summary did not state authoring boundary: %s", item.Summary)
	}
}

func TestOperationsStartupBriefDegradesWhenSnapshotErrors(t *testing.T) {
	r := NewResolver("/tmp/scenario", "/tmp/scenarios", nil, fakeBriefingBuilder{briefing: minimalBriefing()})
	r.SetSnapshotBuilder(fakeSnapshotProvider{err: context.DeadlineExceeded})

	item, err := r.operationsStartupBrief(context.Background(), agentsessions.ContextLimits{MaxSummaryRunes: 4000})
	if err != nil {
		t.Fatalf("operationsStartupBrief should not fail when snapshot errors: %v", err)
	}

	var meta startupBriefMetadata
	if err := json.Unmarshal([]byte(item.MetadataJSON), &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if len(meta.RankedInitiatives) != 0 {
		t.Errorf("expected no ranked initiatives when snapshot errors, got %d", len(meta.RankedInitiatives))
	}
	foundWarning := false
	for _, w := range meta.Warnings {
		if strings.Contains(w, "ranked initiatives unavailable") {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Errorf("expected a 'ranked initiatives unavailable' warning, got %v", meta.Warnings)
	}
}

func TestOperationsStartupBriefWithoutSnapshotProvider(t *testing.T) {
	r := NewResolver("/tmp/scenario", "/tmp/scenarios", nil, fakeBriefingBuilder{briefing: minimalBriefing()})

	item, err := r.operationsStartupBrief(context.Background(), agentsessions.ContextLimits{MaxSummaryRunes: 4000})
	if err != nil {
		t.Fatalf("operationsStartupBrief: %v", err)
	}
	if strings.Contains(item.Summary, "Ranked initiatives") {
		t.Errorf("summary should omit ranked section when no provider wired; got:\n%s", item.Summary)
	}
}
