package sessioncontext

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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

	item, err := r.operationsStartupBrief(context.Background(), "", nil, agentsessions.ContextLimits{MaxSummaryRunes: 4000})
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
	item, err := r.ResolveSessionStartupBrief(context.Background(), agentsessions.KindWorkflowAuthoring, "", nil, agentsessions.ContextLimits{MaxSummaryRunes: 4000})
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
	// The server resolves precedent into the related-work section. The brief
	// must explain that contract without sending the agent to a search tool.
	if !strings.Contains(item.Summary, "attached related-work section") ||
		strings.Contains(item.Summary, "search-hub") {
		t.Fatalf("summary did not use server-resolved precedent: %s", item.Summary)
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
	item, err := r.ResolveSessionStartupBrief(context.Background(), agentsessions.KindWorkflowAuthoring, "", nil, agentsessions.ContextLimits{MaxSummaryRunes: 4000})
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

	item, err := r.operationsStartupBrief(context.Background(), "", nil, agentsessions.ContextLimits{MaxSummaryRunes: 4000})
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

	item, err := r.operationsStartupBrief(context.Background(), "", nil, agentsessions.ContextLimits{MaxSummaryRunes: 4000})
	if err != nil {
		t.Fatalf("operationsStartupBrief: %v", err)
	}
	if strings.Contains(item.Summary, "Ranked goals") {
		t.Errorf("summary should omit ranked section when no provider wired; got:\n%s", item.Summary)
	}
}

func TestOperationsStartupBriefAddsStalenessSweepJobSlice(t *testing.T) {
	root := t.TempDir()
	writeBacklogSpec(t, root, "fix", "stale-item", `{"name":"stale-item","title":"Stale item","kind":"fix","status":"backlog","priority":2,"created":"2025-01-01T00:00:00Z","updated":"2025-01-01T00:00:00Z"}`)
	writeBacklogSpec(t, root, "fix", "fresh-item", `{"name":"fresh-item","title":"Fresh item","kind":"fix","status":"backlog","priority":2,"created":"2026-01-19T00:00:00Z","updated":"2026-01-19T00:00:00Z"}`)
	r := NewResolver(root, filepath.Join(root, "scenarios"), nil, fakeBriefingBuilder{briefing: minimalBriefing()})
	r.now = func() time.Time { return time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC) }

	item, err := r.ResolveSessionStartupBrief(context.Background(), agentsessions.KindSwarmOperations, "operations-sweep-staleness", nil, agentsessions.ContextLimits{MaxSummaryRunes: 4000})
	if err != nil {
		t.Fatalf("ResolveSessionStartupBrief: %v", err)
	}
	for _, want := range []string{"stale-item", "keep", "refresh", "supersede"} {
		if !strings.Contains(item.Summary, want) {
			t.Errorf("summary missing %q:\n%s", want, item.Summary)
		}
	}
	if strings.Contains(item.Summary, "fresh-item") {
		t.Fatalf("fresh item appeared in stale verdict set: %s", item.Summary)
	}
	var meta startupBriefMetadata
	if err := json.Unmarshal([]byte(item.MetadataJSON), &meta); err != nil {
		t.Fatal(err)
	}
	if meta.JobSlice == nil || meta.JobSlice.JobID != "operations-sweep-staleness" || len(meta.JobSlice.StaleItems) != 1 {
		t.Fatalf("job slice metadata = %+v", meta.JobSlice)
	}
}

func TestOperationsStartupBriefAddsRunTerminalReasonSlice(t *testing.T) {
	r := NewResolver(t.TempDir(), "/tmp/scenarios", nil, fakeBriefingBuilder{briefing: minimalBriefing()})
	attached := []agentsessions.ContextItem{{
		Type: agentsessions.ContextExecution, Ref: "exec-1", Title: "Failed run",
		MetadataJSON: `{"status":"failed","terminal_code":"budget_exhausted","budget_name":"tokens","failure_reason":"token ceiling reached"}`,
	}}
	item, err := r.ResolveSessionStartupBrief(context.Background(), agentsessions.KindSwarmOperations, "operations-run", attached, agentsessions.ContextLimits{MaxSummaryRunes: 4000})
	if err != nil {
		t.Fatalf("ResolveSessionStartupBrief: %v", err)
	}
	for _, want := range []string{"exec-1", "budget_exhausted", "tokens", "token ceiling reached"} {
		if !strings.Contains(item.Summary, want) {
			t.Errorf("summary missing %q:\n%s", want, item.Summary)
		}
	}
	var meta startupBriefMetadata
	if err := json.Unmarshal([]byte(item.MetadataJSON), &meta); err != nil {
		t.Fatal(err)
	}
	if meta.JobSlice == nil || meta.JobSlice.TerminalReason == nil || meta.JobSlice.TerminalReason.Code != "budget_exhausted" {
		t.Fatalf("terminal reason metadata = %+v", meta.JobSlice)
	}
}

func TestExecutionContextPreservesTerminalReasonMetadata(t *testing.T) {
	r := NewResolver(t.TempDir(), "/tmp/scenarios", nil)
	r.executions = filepath.Join(t.TempDir(), "execution-runs.json")
	if err := os.WriteFile(r.executions, []byte(`[{"execution_id":"exec-1","status":"failed","terminal_code":"budget_exhausted","budget_name":"tokens","failure_reason":"token ceiling reached"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	items, err := r.ResolveSessionMessageContext(context.Background(), []agentsessions.ContextRef{{Type: agentsessions.ContextExecution, Ref: "exec-1"}}, agentsessions.ContextLimits{MaxSummaryRunes: 4000})
	if err != nil {
		t.Fatalf("ResolveSessionMessageContext: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %+v", items)
	}
	for _, want := range []string{`"terminal_code":"budget_exhausted"`, `"budget_name":"tokens"`, `"failure_reason":"token ceiling reached"`} {
		if !strings.Contains(items[0].MetadataJSON, want) {
			t.Errorf("execution metadata missing %s: %s", want, items[0].MetadataJSON)
		}
	}
}

func TestOperationsStartupBriefUnregisteredJobIsByteIdenticalToNoJob(t *testing.T) {
	r := NewResolver(t.TempDir(), "/tmp/scenarios", nil, fakeBriefingBuilder{briefing: minimalBriefing()})
	limits := agentsessions.ContextLimits{MaxSummaryRunes: 4000}
	withoutJob, err := r.ResolveSessionStartupBrief(context.Background(), agentsessions.KindSwarmOperations, "", nil, limits)
	if err != nil {
		t.Fatal(err)
	}
	unknownJob, err := r.ResolveSessionStartupBrief(context.Background(), agentsessions.KindSwarmOperations, "not-registered", nil, limits)
	if err != nil {
		t.Fatal(err)
	}
	if withoutJob != unknownJob {
		t.Fatalf("unregistered job changed brief:\nno job: %+v\nunknown: %+v", withoutJob, unknownJob)
	}
}

func TestOperationsStartupBriefJobSliceFailureKeepsSummaryAndWarns(t *testing.T) {
	r := NewResolver(t.TempDir(), "/tmp/scenarios", nil, fakeBriefingBuilder{briefing: minimalBriefing()})
	r.jobSlices["failing-job"] = func(context.Context, *Resolver, []agentsessions.ContextItem) (startupBriefJobSlice, error) {
		return startupBriefJobSlice{}, errors.New("source offline")
	}
	limits := agentsessions.ContextLimits{MaxSummaryRunes: 4000}
	base, err := r.ResolveSessionStartupBrief(context.Background(), agentsessions.KindSwarmOperations, "", nil, limits)
	if err != nil {
		t.Fatal(err)
	}
	degraded, err := r.ResolveSessionStartupBrief(context.Background(), agentsessions.KindSwarmOperations, "failing-job", nil, limits)
	if err != nil {
		t.Fatalf("failing job slice must not fail the brief: %v", err)
	}
	if degraded.Summary != base.Summary {
		t.Fatalf("slice failure changed base summary:\nbase: %s\ndegraded: %s", base.Summary, degraded.Summary)
	}
	var meta startupBriefMetadata
	if err := json.Unmarshal([]byte(degraded.MetadataJSON), &meta); err != nil {
		t.Fatal(err)
	}
	if len(meta.Warnings) == 0 || !strings.Contains(strings.Join(meta.Warnings, " "), "source offline") {
		t.Fatalf("slice failure warning missing: %+v", meta.Warnings)
	}
}

func writeBacklogSpec(t *testing.T, root, kind, name, payload string) {
	t.Helper()
	dir := filepath.Join(root, kind, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.json"), []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
}
