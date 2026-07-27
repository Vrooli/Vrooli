package orchestration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

func TestWorkflowWaitHasDeadlineUsesLatestWaitIntent(t *testing.T) {
	id := uuid.New()
	deadline, _ := json.Marshal(map[string]any{"deadline": time.Now().Add(time.Hour)})
	noDeadline, _ := json.Marshal(map[string]any{"deadline": time.Time{}})
	if !workflowWaitHasDeadline([]*domain.WorkflowJournalEntry{{ExecutionID: id, Kind: domain.WorkflowJournalWait, Payload: deadline}}) {
		t.Fatal("expected durable wait deadline to be detected")
	}
	if workflowWaitHasDeadline([]*domain.WorkflowJournalEntry{{ExecutionID: id, Kind: domain.WorkflowJournalWait, Payload: deadline}, {ExecutionID: id, Kind: domain.WorkflowJournalWait, Payload: noDeadline}}) {
		t.Fatal("latest unarmed wait intent must win")
	}
	if workflowWaitHasDeadline(nil) {
		t.Fatal("missing wait intent must be unarmed")
	}
	if workflowWaitHasDeadline([]*domain.WorkflowJournalEntry{{ExecutionID: id, Kind: domain.WorkflowJournalWait, Payload: json.RawMessage(`not-json`)}}) {
		t.Fatal("malformed wait intent must be treated as unarmed")
	}
	if workflowWaitHasDeadline([]*domain.WorkflowJournalEntry{{ExecutionID: id, Kind: domain.WorkflowJournalInput, Payload: deadline}}) {
		t.Fatal("non-wait journal entries must not arm a wait")
	}
}

func TestReconcileUnarmedWorkflowWaitsWarnsThenReapsWithInjectedClock(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	revision := &domain.WorkflowRevision{
		ID: uuid.New(), Owner: "test", Key: "test/unarmed", SemanticVersion: "1.0.0", Digest: "sha256:unarmed", Active: true, SourcePath: "test", SourceHash: "test", SourceUpdatedAt: now, CreatedAt: now,
		Definition: domain.WorkflowDefinition{
			SchemaVersion: domain.WorkflowSchemaVersionV1, Owner: "test", Key: "test/unarmed", Version: "1.0.0", EntryNode: "end", InputSchema: json.RawMessage(`{}`), OutputSchema: json.RawMessage(`{}`),
			Nodes:   []domain.WorkflowNode{{ID: "end", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}},
			Budgets: domain.WorkflowBudgets{WallTimeSeconds: 600, MaxTurns: 1, MaxTokens: 1, MaxCostUSD: 1, MaxNodeAttempts: 1, MaxChildren: 1, MaxConcurrency: 1, MaxRecursion: 1, MaxRetries: 1, MaxWaitSeconds: 60},
		},
	}
	if err := repos.Workflows.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatalf("activate workflow revision: %v", err)
	}
	execution := &domain.WorkflowExecution{
		ID: uuid.New(), Owner: "test", WorkflowKey: "test/unarmed", DefinitionDigest: "sha256:unarmed", Status: domain.WorkflowExecutionWaiting,
		CurrentNodeID: "approval", Input: json.RawMessage(`{}`), EdgeTraversals: map[string]int{}, Version: 1, IdempotencyKey: "unarmed-wait", CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now.Add(-2 * time.Minute),
	}
	if err := repos.WorkflowExecutions.Create(ctx, execution, &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: execution.ID, Sequence: 1, Kind: domain.WorkflowJournalInput, Payload: json.RawMessage(`{}`), CreatedAt: execution.CreatedAt}); err != nil {
		t.Fatalf("create waiting execution: %v", err)
	}
	persisted, err := repos.WorkflowExecutions.Get(ctx, execution.ID)
	if err != nil {
		t.Fatalf("get waiting execution: %v", err)
	}
	// Repository creation normalizes timestamps, so derive the fake clock from
	// its durable row rather than assuming the caller-supplied timestamps won.
	now = persisted.UpdatedAt.Add(2 * time.Minute)
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithWorkflowRepository(repos.Workflows), WithWorkflowExecutionRepository(repos.WorkflowExecutions), WithClock(func() time.Time { return now }))
	t.Cleanup(o.dispatcher.Close)
	if got := o.now(); !got.Equal(now) {
		t.Fatalf("orchestrator clock = %s, want %s", got, now)
	}
	waiting, err := repos.WorkflowExecutions.List(ctx, repository.WorkflowExecutionListFilter{Status: domain.WorkflowExecutionWaiting, ListFilter: repository.ListFilter{Limit: 200}})
	if err != nil || len(waiting) != 1 {
		t.Fatalf("list waiting executions = %d, err=%v", len(waiting), err)
	}
	initialJournal, err := repos.WorkflowExecutions.ListJournal(ctx, execution.ID, 0, 0)
	if err != nil || workflowWaitHasDeadline(initialJournal) {
		t.Fatalf("initial unarmed journal deadline=%v err=%v", workflowWaitHasDeadline(initialJournal), err)
	}
	if err := o.ReconcileUnarmedWorkflowWaits(ctx, time.Minute, 3*time.Minute); err != nil {
		t.Fatalf("warn unarmed wait: %v", err)
	}
	journal, err := repos.WorkflowExecutions.ListJournal(ctx, execution.ID, 0, 0)
	if err != nil {
		t.Fatalf("list warning journal: %v", err)
	}
	if len(journal) != 2 || journal[1].Kind != domain.WorkflowJournalDiagnostic {
		t.Fatalf("journal after warning count=%d now=%s persistedUpdated=%s listedUpdated=%s entries=%+v, want durable diagnostic", len(journal), now, persisted.UpdatedAt, waiting[0].UpdatedAt, journal)
	}

	now = now.Add(2 * time.Minute)
	if err := o.ReconcileUnarmedWorkflowWaits(ctx, time.Minute, 3*time.Minute); err != nil {
		t.Fatalf("reap unarmed wait: %v", err)
	}
	got, err := repos.WorkflowExecutions.Get(ctx, execution.ID)
	if err != nil {
		t.Fatalf("get reaped execution: %v", err)
	}
	if got.Status != domain.WorkflowExecutionFailed || got.TerminalReason == nil || got.TerminalReason.Code != "unarmed_wait_reaped" {
		t.Fatalf("reaped execution = %+v", got)
	}
}
