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
		Summary:          operations.OperationsBriefingSummary{ActiveGoals: 2, BlockedItems: 1},
	}
}

func TestOperationsStartupBriefWeavesRankedSnapshot(t *testing.T) {
	r := NewResolver("/tmp/scenario", "/tmp/scenarios", nil, fakeBriefingBuilder{briefing: minimalBriefing()})
	r.SetSnapshotBuilder(fakeSnapshotProvider{snapshot: &operations.OperationsSnapshot{
		GeneratedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Goals: []operations.RankedGoal{
			{Name: "ship-cockpit", Title: "Ship the cockpit", Priority: 1, Readiness: operations.ReadinessReady},
			{Name: "later-thing", Title: "Later thing", Priority: 0, Readiness: operations.ReadinessBlocked},
		},
		Summary: operations.SnapshotSummary{TotalGoals: 2, ReadyGoals: 1, BlockedGoals: 1},
	}})

	item, err := r.operationsStartupBrief(context.Background(), agentsessions.ContextLimits{MaxSummaryRunes: 4000})
	if err != nil {
		t.Fatalf("operationsStartupBrief: %v", err)
	}

	// Summary carries the typed reference span the UI linkifies.
	if !strings.Contains(item.Summary, "`goal:ship-cockpit`") {
		t.Errorf("summary missing typed goal reference; got:\n%s", item.Summary)
	}
	if !strings.Contains(item.Summary, "Ranked goals") {
		t.Errorf("summary missing ranked goals section; got:\n%s", item.Summary)
	}

	// Metadata carries the structured ranked rows with refs.
	var meta startupBriefMetadata
	if err := json.Unmarshal([]byte(item.MetadataJSON), &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if len(meta.RankedGoals) != 2 {
		t.Fatalf("expected 2 ranked goals in metadata, got %d", len(meta.RankedGoals))
	}
	if meta.RankedGoals[0].Ref != "goal:ship-cockpit" {
		t.Errorf("first ranked ref = %q, want goal:ship-cockpit", meta.RankedGoals[0].Ref)
	}
	if meta.SourceCounts["ranked_goals"] != 2 {
		t.Errorf("source_counts.ranked_goals = %d, want 2", meta.SourceCounts["ranked_goals"])
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
	// The brief must state the object/meta boundary, because choosing the wrong
	// session kind is the failure this brief exists to prevent.
	if !strings.Contains(item.Summary, "the machine, not the product") ||
		!strings.Contains(item.Summary, "Plan Work session") {
		t.Fatalf("summary did not state the object/meta boundary: %s", item.Summary)
	}
	// Precedent search comes before design: the scarce resource in a meta
	// conversation is knowing the problem was already solved elsewhere.
	if !strings.Contains(item.Summary, "search-hub query") {
		t.Fatalf("summary did not direct the agent to search for precedent: %s", item.Summary)
	}
	// The live architecture is derived from runtime constants so the brief
	// cannot drift from the code it describes.
	if !strings.Contains(item.Summary, agentsessions.SkillWorkflowAuthoring) {
		t.Fatalf("summary omitted the live session architecture: %s", item.Summary)
	}
	if !strings.Contains(item.Summary, "SESSION-ARCHITECTURE-DESIGN-RECORD.md") {
		t.Fatalf("summary omitted the authoritative references: %s", item.Summary)
	}
}

// The resolver is pointed at paths that do not exist. A missing design-record
// directory or goal store must degrade to a usable brief with a recorded
// warning, never to a failed session start.
func TestWorkflowAuthoringStartupBriefDegradesWhenSourcesAreMissing(t *testing.T) {
	r := NewResolver("/tmp/does-not-exist-scenario", "/tmp/does-not-exist-scenarios", nil)
	item, err := r.ResolveSessionStartupBrief(context.Background(), agentsessions.KindWorkflowAuthoring, agentsessions.ContextLimits{MaxSummaryRunes: 4000})
	if err != nil {
		t.Fatalf("brief must not fail when its optional sources are unreadable: %v", err)
	}
	if strings.TrimSpace(item.Summary) == "" {
		t.Fatal("brief degraded to an empty summary")
	}

	var meta startupBriefMetadata
	if err := json.Unmarshal([]byte(item.MetadataJSON), &meta); err != nil {
		t.Fatalf("metadata: %v", err)
	}
	if len(meta.Warnings) == 0 {
		t.Error("unreadable design-record directory produced no warning; the gap would be invisible")
	}
	if len(meta.DrillDownCommands) == 0 {
		t.Error("brief lost its drill-down commands when optional sources were missing")
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
	if len(meta.RankedGoals) != 0 {
		t.Errorf("expected no ranked goals when snapshot errors, got %d", len(meta.RankedGoals))
	}
	foundWarning := false
	for _, w := range meta.Warnings {
		if strings.Contains(w, "ranked goals unavailable") {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Errorf("expected a 'ranked goals unavailable' warning, got %v", meta.Warnings)
	}
}

func TestOperationsStartupBriefWithoutSnapshotProvider(t *testing.T) {
	r := NewResolver("/tmp/scenario", "/tmp/scenarios", nil, fakeBriefingBuilder{briefing: minimalBriefing()})

	item, err := r.operationsStartupBrief(context.Background(), agentsessions.ContextLimits{MaxSummaryRunes: 4000})
	if err != nil {
		t.Fatalf("operationsStartupBrief: %v", err)
	}
	if strings.Contains(item.Summary, "Ranked goals") {
		t.Errorf("summary should omit ranked section when no provider wired; got:\n%s", item.Summary)
	}
}
