package database

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func TestWorkflowRepositoryActivatesImmutableRevisionsAndRollsBackBatch(t *testing.T) {
	db := setupTestDB(t)
	repo := &workflowRepository{db: db, log: logrus.New()}
	ctx := context.Background()
	first := workflowRevision("owner/flow", "sha256:first", "1.0.0")
	if err := repo.ActivateBatch(ctx, []*domain.WorkflowRevision{first}); err != nil {
		t.Fatal(err)
	}
	active, err := repo.GetActive(ctx, "owner", "owner/flow")
	if err != nil {
		t.Fatal(err)
	}
	if active == nil || active.Digest != first.Digest {
		t.Fatalf("unexpected active revision: %#v", active)
	}
	second := workflowRevision("owner/flow", "sha256:second", "1.1.0")
	bad := workflowRevision("owner/other", "sha256:second", "1.0.0") // digest conflict means second activation cannot find its owner/key row.
	if err := repo.ActivateBatch(ctx, []*domain.WorkflowRevision{second, bad}); err == nil {
		t.Fatal("expected atomic batch failure")
	}
	active, err = repo.GetActive(ctx, "owner", "owner/flow")
	if err != nil {
		t.Fatal(err)
	}
	if active == nil || active.Digest != first.Digest {
		t.Fatalf("failed reload replaced prior revision: %#v", active)
	}
	if found, err := repo.GetByDigest(ctx, second.Digest); err != nil || found != nil {
		t.Fatalf("failed batch leaked revision: %#v %v", found, err)
	}
}

func TestWorkflowRepositoryReactivationIsIdempotent(t *testing.T) {
	db := setupTestDB(t)
	repo := &workflowRepository{db: db, log: logrus.New()}
	ctx := context.Background()
	revision := workflowRevision("owner/flow", "sha256:same", "1.0.0")
	if err := repo.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatal(err)
	}
	replay := workflowRevision("owner/flow", "sha256:same", "1.0.0")
	if err := repo.ActivateBatch(ctx, []*domain.WorkflowRevision{replay}); err != nil {
		t.Fatal(err)
	}
	list, err := repo.List(ctx, "owner", "owner/flow", repository.ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("revision list length = %d, want 1", len(list))
	}
	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM workflow_revisions WHERE digest = ?`, revision.Digest); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("revision count = %d, want 1", count)
	}
}

func TestWorkflowExecutionRevisionPinSurvivesCutoverRestartAndRollback(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	catalog := &workflowRepository{db: db, log: logrus.New()}
	oldRevision := workflowRevision("owner/investigate", "sha256:old", "1.0.0")
	newRevision := workflowRevision("owner/investigate", "sha256:new", "2.0.0")
	if err := catalog.ActivateBatch(ctx, []*domain.WorkflowRevision{oldRevision}); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	execution := &domain.WorkflowExecution{
		ID: uuid.New(), Owner: "owner", WorkflowKey: oldRevision.Key, DefinitionDigest: oldRevision.Digest,
		Status: domain.WorkflowExecutionRunning, CurrentNodeID: "start", Input: json.RawMessage(`{"question":"old"}`),
		EdgeTraversals: map[string]int{}, Version: 1, IdempotencyKey: "pinned-old-investigation", CreatedAt: now, UpdatedAt: now,
	}
	initial := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: execution.ID, Sequence: 1, Kind: domain.WorkflowJournalInput, Payload: execution.Input, CreatedAt: now}
	executions := &workflowExecutionRepository{db: db, log: logrus.New()}
	if err := executions.Create(ctx, execution, initial); err != nil {
		t.Fatal(err)
	}

	if err := catalog.ActivateBatch(ctx, []*domain.WorkflowRevision{newRevision}); err != nil {
		t.Fatal(err)
	}
	reloaded := &workflowExecutionRepository{db: db, log: logrus.New()}
	got, err := reloaded.GetByIdempotencyKey(ctx, execution.IdempotencyKey)
	if err != nil || got.DefinitionDigest != oldRevision.Digest || got.Status != domain.WorkflowExecutionRunning {
		t.Fatalf("cutover changed pinned execution=%+v err=%v", got, err)
	}

	got.Version++
	got.Status = domain.WorkflowExecutionSucceeded
	got.CurrentNodeID = "done"
	got.EndedAt = ptrTime(now.Add(time.Second))
	got.UpdatedAt = now.Add(time.Second)
	completion := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: got.ID, Sequence: 2, Kind: domain.WorkflowJournalHandoff, Payload: json.RawMessage(`{"status":"succeeded"}`), CreatedAt: got.UpdatedAt}
	if ok, err := reloaded.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: 1, Execution: got, Journal: []*domain.WorkflowJournalEntry{completion}}); err != nil || !ok {
		t.Fatalf("pinned completion ok=%t err=%v", ok, err)
	}

	if err := catalog.ActivateBatch(ctx, []*domain.WorkflowRevision{oldRevision}); err != nil {
		t.Fatal(err)
	}
	active, err := catalog.GetActive(ctx, oldRevision.Owner, oldRevision.Key)
	if err != nil || active.Digest != oldRevision.Digest {
		t.Fatalf("rollback active=%+v err=%v", active, err)
	}
	all, err := catalog.List(ctx, oldRevision.Owner, oldRevision.Key, repository.ListFilter{})
	if err != nil || len(all) != 2 {
		t.Fatalf("rollback revision history=%+v err=%v", all, err)
	}
	completed, err := reloaded.GetByIdempotencyKey(ctx, execution.IdempotencyKey)
	if err != nil || completed.DefinitionDigest != oldRevision.Digest || completed.Status != domain.WorkflowExecutionSucceeded {
		t.Fatalf("completed pinned execution=%+v err=%v", completed, err)
	}
}

func TestWorkflowExecutionRepositoryCASAndJournalSurviveReload(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	catalog := &workflowRepository{db: db, log: logrus.New()}
	revision := workflowRevision("owner/flow", "sha256:execution", "1.0.0")
	if err := catalog.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatal(err)
	}
	repo := &workflowExecutionRepository{db: db, log: logrus.New()}
	now := time.Now().UTC()
	parentExecutionID, parentAttemptID := uuid.New(), uuid.New()
	execution := &domain.WorkflowExecution{ID: uuid.New(), Owner: "owner", WorkflowKey: "owner/flow", DefinitionDigest: revision.Digest, Status: domain.WorkflowExecutionRunning, CurrentNodeID: "start", Input: json.RawMessage(`{}`), EdgeTraversals: map[string]int{}, Version: 1, IdempotencyKey: "exec-once", ParentExecutionID: &parentExecutionID, ParentAttemptID: &parentAttemptID, Depth: 1, CreatedAt: now, UpdatedAt: now}
	initial := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: execution.ID, Sequence: 1, Kind: domain.WorkflowJournalInput, Payload: json.RawMessage(`{}`), CreatedAt: now}
	if err := repo.Create(ctx, execution, initial); err != nil {
		t.Fatal(err)
	}
	childExecutionID := uuid.New()
	attempt := &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: execution.ID, NodeID: "start", Ordinal: 1, Strategy: domain.WorkflowAttemptChild, Status: domain.WorkflowAttemptDispatchPending, IdempotencyKey: "attempt-once", InputSnapshot: json.RawMessage(`{}`), PromptSnapshot: "secret prompt persisted, never logged", ChildExecutionID: &childExecutionID, RawOutput: `{"answer":false}`, ValidationError: "schema_mismatch: expected string", Version: 1, CreatedAt: now, UpdatedAt: now}
	execution.Version = 2
	execution.UpdatedAt = now.Add(time.Second)
	entry := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: execution.ID, Sequence: 2, Kind: domain.WorkflowJournalAttempt, NodeID: "start", AttemptID: &attempt.ID, Payload: json.RawMessage(`{"strategy":"fresh_run"}`), CreatedAt: execution.UpdatedAt}
	ok, err := repo.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: 1, Execution: execution, Attempt: attempt, Journal: []*domain.WorkflowJournalEntry{entry}})
	if err != nil || !ok {
		t.Fatalf("commit ok=%t err=%v", ok, err)
	}
	stale := *execution
	stale.Version = 2
	if ok, err := repo.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: 1, Execution: &stale}); err != nil || ok {
		t.Fatalf("stale CAS ok=%t err=%v", ok, err)
	}
	reloaded := &workflowExecutionRepository{db: db, log: logrus.New()}
	got, err := reloaded.GetByIdempotencyKey(ctx, "exec-once")
	if err != nil || got == nil || got.Version != 2 || got.Depth != 1 || got.ParentAttemptID == nil || *got.ParentAttemptID != parentAttemptID {
		t.Fatalf("reloaded=%+v err=%v", got, err)
	}
	attempts, err := reloaded.ListAttempts(ctx, execution.ID)
	if err != nil || len(attempts) != 1 || attempts[0].PromptSnapshot != attempt.PromptSnapshot || attempts[0].RawOutput != attempt.RawOutput || attempts[0].ValidationError != attempt.ValidationError || attempts[0].ChildExecutionID == nil || *attempts[0].ChildExecutionID != childExecutionID {
		t.Fatalf("attempts=%+v err=%v", attempts, err)
	}
	journal, err := reloaded.ListJournal(ctx, execution.ID, 0, 0)
	if err != nil || len(journal) != 2 || journal[1].Sequence != 2 {
		t.Fatalf("journal=%+v err=%v", journal, err)
	}
	recoverable, err := reloaded.ListRecoverable(ctx, 10)
	if err != nil || len(recoverable) != 1 || recoverable[0].ID != execution.ID {
		t.Fatalf("recoverable=%+v err=%v", recoverable, err)
	}
	history, err := reloaded.List(ctx, repository.WorkflowExecutionListFilter{Owner: "owner", WorkflowKey: "owner/flow", Status: domain.WorkflowExecutionRunning, ListFilter: repository.ListFilter{Limit: 1}})
	if err != nil || len(history) != 1 || history[0].ID != execution.ID || string(history[0].Input) != `{}` {
		t.Fatalf("history=%+v err=%v", history, err)
	}
	missing, err := reloaded.List(ctx, repository.WorkflowExecutionListFilter{Status: domain.WorkflowExecutionFailed})
	if err != nil || len(missing) != 0 {
		t.Fatalf("failed history=%+v err=%v", missing, err)
	}
}

func TestWorkflowAttemptCommitRejectsInvalidTransitionAndRecoversWaitingAttempt(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	catalog := &workflowRepository{db: db, log: logrus.New()}
	revision := workflowRevision("owner/attempt-lifecycle", "sha256:attempt-lifecycle", "1.0.0")
	if err := catalog.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatal(err)
	}
	repo := &workflowExecutionRepository{db: db, log: logrus.New()}
	now := time.Now().UTC()
	execution := &domain.WorkflowExecution{ID: uuid.New(), Owner: "owner", WorkflowKey: revision.Key, DefinitionDigest: revision.Digest, Status: domain.WorkflowExecutionRunning, CurrentNodeID: "start", Input: json.RawMessage(`{}`), EdgeTraversals: map[string]int{}, Version: 1, IdempotencyKey: "attempt-lifecycle", CreatedAt: now, UpdatedAt: now}
	initial := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: execution.ID, Sequence: 1, Kind: domain.WorkflowJournalInput, Payload: execution.Input, CreatedAt: now}
	if err := repo.Create(ctx, execution, initial); err != nil {
		t.Fatal(err)
	}
	attempt := &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: execution.ID, NodeID: "start", Ordinal: 1, Strategy: domain.WorkflowAttemptFreshRun, Status: domain.WorkflowAttemptDispatchPending, IdempotencyKey: "attempt-lifecycle-node", InputSnapshot: json.RawMessage(`{}`), Version: 1, CreatedAt: now, UpdatedAt: now}
	commit := func(status domain.WorkflowAttemptStatus) (bool, error) {
		execution.Version++
		execution.UpdatedAt = now.Add(time.Duration(execution.Version) * time.Second)
		attempt.Status = status
		attempt.Version++
		attempt.UpdatedAt = execution.UpdatedAt
		return repo.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: execution.Version - 1, Execution: execution, Attempt: attempt})
	}
	if ok, err := commit(domain.WorkflowAttemptDispatchPending); err != nil || !ok {
		t.Fatalf("persist pending attempt ok=%t err=%v", ok, err)
	}
	if ok, err := commit(domain.WorkflowAttemptCompleted); ok || !errors.Is(err, domain.ErrInvalidWorkflowAttemptStatusTransition) {
		t.Fatalf("pending -> completed committed=%t err=%v", ok, err)
	}
	// The repository transaction rolled back; restore the caller's speculative CAS versions too.
	execution.Version--
	attempt.Version--
	attempt.Status = domain.WorkflowAttemptDispatchPending
	got, err := repo.ListAttempts(ctx, execution.ID)
	if err != nil || len(got) != 1 || got[0].Status != domain.WorkflowAttemptDispatchPending {
		t.Fatalf("rejected transition changed durable attempt: attempts=%+v err=%v", got, err)
	}
	for _, status := range []domain.WorkflowAttemptStatus{domain.WorkflowAttemptDispatched, domain.WorkflowAttemptWaiting, domain.WorkflowAttemptDispatched, domain.WorkflowAttemptCompleted} {
		if ok, err := commit(status); err != nil || !ok {
			t.Fatalf("recovery transition to %s ok=%t err=%v", status, ok, err)
		}
	}
	got, err = repo.ListAttempts(ctx, execution.ID)
	if err != nil || len(got) != 1 || got[0].Status != domain.WorkflowAttemptCompleted {
		t.Fatalf("recovered attempt=%+v err=%v", got, err)
	}
}

func TestWorkflowExecutionCancellationFenceSerializesStartAdmission(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	catalog := &workflowRepository{db: db, log: logrus.New()}
	revision := workflowRevision("owner/staged-message", "sha256:cancel-fence", "1.0.0")
	if err := catalog.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatal(err)
	}
	repo := &workflowExecutionRepository{db: db, log: logrus.New()}
	makeExecution := func(key string) (*domain.WorkflowExecution, *domain.WorkflowJournalEntry) {
		now := time.Now().UTC()
		execution := &domain.WorkflowExecution{ID: uuid.New(), Owner: revision.Owner, WorkflowKey: revision.Key, DefinitionDigest: revision.Digest, Status: domain.WorkflowExecutionRunning, CurrentNodeID: "start", Input: json.RawMessage(`{}`), EdgeTraversals: map[string]int{}, Version: 1, IdempotencyKey: key, CreatedAt: now, UpdatedAt: now}
		initial := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: execution.ID, Sequence: 1, Kind: domain.WorkflowJournalInput, Payload: execution.Input, CreatedAt: now}
		return execution, initial
	}

	// Cancellation wins before the workflow row is committed. A late Start must
	// observe the fence inside the same transaction that would insert the row.
	fencedMiss, err := repo.CancelByIdempotencyKey(ctx, "cancel-wins", revision.Owner, revision.Key)
	if err != nil || fencedMiss != nil {
		t.Fatalf("cancel-first result=%+v err=%v", fencedMiss, err)
	}
	late, lateJournal := makeExecution("cancel-wins")
	if err := repo.Create(ctx, late, lateJournal); err == nil {
		t.Fatal("late Start created a workflow after the idempotency key was fenced")
	}
	if got, err := repo.GetByIdempotencyKey(ctx, "cancel-wins"); err != nil || got != nil {
		t.Fatalf("fenced miss unexpectedly has execution: %+v err=%v", got, err)
	}

	// Start wins first. Cancellation fences subsequent admission and returns
	// the exact execution so the owner can cancel it by handle.
	started, initial := makeExecution("start-wins")
	if err := repo.Create(ctx, started, initial); err != nil {
		t.Fatal(err)
	}
	fencedExecution, err := repo.CancelByIdempotencyKey(ctx, "start-wins", revision.Owner, revision.Key)
	if err != nil || fencedExecution == nil || fencedExecution.ID != started.ID || fencedExecution.IdempotencyKey != started.IdempotencyKey {
		t.Fatalf("start-first cancellation did not recover exact execution: %+v err=%v", fencedExecution, err)
	}
}

func TestWorkflowExecutionRepositoryRejectsInvalidLifecycleTransition(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	catalog := &workflowRepository{db: db, log: logrus.New()}
	revision := workflowRevision("owner/lifecycle", "sha256:lifecycle", "1.0.0")
	if err := catalog.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatal(err)
	}
	store := &workflowExecutionRepository{db: db, log: logrus.New()}
	now := time.Now().UTC()
	execution := &domain.WorkflowExecution{
		ID: uuid.New(), Owner: "owner", WorkflowKey: revision.Key, DefinitionDigest: revision.Digest,
		Status: domain.WorkflowExecutionSucceeded, CurrentNodeID: "done", Input: json.RawMessage(`{}`),
		EdgeTraversals: map[string]int{}, Version: 1, IdempotencyKey: "lifecycle-invalid-edge", CreatedAt: now, UpdatedAt: now,
	}
	initial := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: execution.ID, Sequence: 1, Kind: domain.WorkflowJournalInput, Payload: execution.Input, CreatedAt: now}
	if err := store.Create(ctx, execution, initial); err != nil {
		t.Fatal(err)
	}

	resurrected := *execution
	resurrected.Status = domain.WorkflowExecutionRunning
	resurrected.Version++
	resurrected.UpdatedAt = now.Add(time.Second)
	if ok, err := store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: 1, Execution: &resurrected}); !errors.Is(err, domain.ErrInvalidWorkflowExecutionStatusTransition) || ok {
		t.Fatalf("terminal resurrection committed=%t err=%v", ok, err)
	}
	got, err := store.Get(ctx, execution.ID)
	if err != nil || got == nil || got.Status != domain.WorkflowExecutionSucceeded || got.Version != 1 {
		t.Fatalf("rejected transition changed durable execution: got=%+v err=%v", got, err)
	}
}

func TestWorkflowExecutionRepositoryPersistsEngagementGrantAcrossReload(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	repo := &workflowExecutionRepository{db: db, log: logrus.New()}
	now := time.Now().UTC()
	catalog := &workflowRepository{db: db, log: logrus.New()}
	if err := catalog.ActivateBatch(ctx, []*domain.WorkflowRevision{workflowRevision("owner/flow", "sha256:grant", "1.0.0")}); err != nil {
		t.Fatal(err)
	}
	grant := &domain.WorkflowEngagementGrant{MaxTurns: 4, MaxTokens: 1200, MaxChargeMicroUSD: 75, MaxWallTimeSeconds: 90, MaxNodeAttempts: 5, MaxChildren: 2, MaxConcurrency: 1, MaxRecursion: 1, MaxRetries: 1, MaxWaitSeconds: 30}
	execution := &domain.WorkflowExecution{ID: uuid.New(), Owner: "owner", WorkflowKey: "owner/flow", DefinitionDigest: "sha256:grant", ApprovalDigest: "sha256:approval", GrantDigest: "sha256:grant-binding", Status: domain.WorkflowExecutionRunning, CurrentNodeID: "start", Input: json.RawMessage(`{}`), EdgeTraversals: map[string]int{}, Version: 1, IdempotencyKey: "grant-reload", EngagementGrant: grant, CreatedAt: now, UpdatedAt: now}
	initial := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: execution.ID, Sequence: 1, Kind: domain.WorkflowJournalInput, Payload: execution.Input, CreatedAt: now}
	if err := repo.Create(ctx, execution, initial); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.EngagementGrant == nil || !reflect.DeepEqual(*got.EngagementGrant, *grant) {
		t.Fatalf("grant after reload=%+v, want %+v", got.EngagementGrant, grant)
	}
	if got.ApprovalDigest != execution.ApprovalDigest || got.GrantDigest != execution.GrantDigest {
		t.Fatalf("binding after reload=%+v, want approval=%q grant=%q", got, execution.ApprovalDigest, execution.GrantDigest)
	}
}

func TestWorkflowExecutionRepositoryPersistsExecutionPreferencesAcrossReload(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	repo := &workflowExecutionRepository{db: db, log: logrus.New()}
	catalog := &workflowRepository{db: db, log: logrus.New()}
	if err := catalog.ActivateBatch(ctx, []*domain.WorkflowRevision{workflowRevision("owner/flow", "sha256:preferences", "1.0.0")}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	preferences := &domain.ExecutionPreferences{PreferredRunner: "claude-code", Model: "claude-sonnet", Effort: "high"}
	execution := &domain.WorkflowExecution{ID: uuid.New(), Owner: "owner", WorkflowKey: "owner/flow", DefinitionDigest: "sha256:preferences", Status: domain.WorkflowExecutionRunning, CurrentNodeID: "start", Input: json.RawMessage(`{}`), EdgeTraversals: map[string]int{}, Version: 1, IdempotencyKey: "preferences-reload", ExecutionPreferences: preferences, CreatedAt: now, UpdatedAt: now}
	initial := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: execution.ID, Sequence: 1, Kind: domain.WorkflowJournalInput, Payload: execution.Input, CreatedAt: now}
	if err := repo.Create(ctx, execution, initial); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ExecutionPreferences == nil || !reflect.DeepEqual(*got.ExecutionPreferences, *preferences) {
		t.Fatalf("preferences after reload=%+v, want %+v", got.ExecutionPreferences, preferences)
	}
}

func TestWorkflowExecutionRepositoryListsLegacyAttemptWithNullOptionalFields(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	repo := &workflowExecutionRepository{db: db, log: logrus.New()}
	catalog := &workflowRepository{db: db, log: logrus.New()}
	revision := workflowRevision("owner/flow", "sha256:legacy", "1.0.0")
	if err := catalog.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	executionID, attemptID := uuid.New(), uuid.New()
	if _, err := db.Exec(`INSERT INTO workflow_executions (id,owner,workflow_key,definition_digest,status,current_node_id,input_json,budget_usage_json,edge_traversals_json,version,idempotency_key,depth,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, executionID, "owner", "owner/flow", "sha256:legacy", domain.WorkflowExecutionRunning, "start", `{}`, `{}`, `{}`, 1, "legacy-execution", 0, SQLiteTime(now), SQLiteTime(now)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workflow_node_attempts (id,execution_id,node_id,ordinal,strategy,status,idempotency_key,input_snapshot_json,prompt_snapshot,conversation_id,version,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`, attemptID, executionID, "start", 1, domain.WorkflowAttemptFreshRun, domain.WorkflowAttemptDispatchPending, "legacy-attempt", `{}`, "prompt", "", 1, SQLiteTime(now), SQLiteTime(now)); err != nil {
		t.Fatal(err)
	}

	attempts, err := repo.ListAttempts(ctx, executionID)
	if err != nil || len(attempts) != 1 {
		t.Fatalf("ListAttempts() = %+v, %v", attempts, err)
	}
	if got := attempts[0]; got.ErrorCode != "" || got.RawOutput != "" || got.ValidationError != "" {
		t.Fatalf("legacy nullable fields were not normalized: %+v", got)
	}
}

func TestWorkflowExecutionRepositoryExecutionIDForRun(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	catalog := &workflowRepository{db: db, log: logrus.New()}
	revision := workflowRevision("owner/flow", "sha256:reverse", "1.0.0")
	if err := catalog.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatal(err)
	}
	repo := &workflowExecutionRepository{db: db, log: logrus.New()}
	now := time.Now().UTC()
	execution := &domain.WorkflowExecution{ID: uuid.New(), Owner: "owner", WorkflowKey: "owner/flow", DefinitionDigest: revision.Digest, Status: domain.WorkflowExecutionRunning, CurrentNodeID: "start", Input: json.RawMessage(`{}`), EdgeTraversals: map[string]int{}, Version: 1, IdempotencyKey: "exec-reverse", CreatedAt: now, UpdatedAt: now}
	initial := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: execution.ID, Sequence: 1, Kind: domain.WorkflowJournalInput, Payload: json.RawMessage(`{}`), CreatedAt: now}
	if err := repo.Create(ctx, execution, initial); err != nil {
		t.Fatal(err)
	}
	runID := uuid.New()
	attempt := &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: execution.ID, NodeID: "start", Ordinal: 1, Strategy: domain.WorkflowAttemptFreshRun, Status: domain.WorkflowAttemptDispatchPending, IdempotencyKey: "attempt-reverse", InputSnapshot: json.RawMessage(`{}`), Version: 1, CreatedAt: now, UpdatedAt: now}
	execution.Version = 2
	execution.UpdatedAt = now.Add(time.Second)
	entry := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: execution.ID, Sequence: 2, Kind: domain.WorkflowJournalAttempt, NodeID: "start", AttemptID: &attempt.ID, Payload: json.RawMessage(`{"strategy":"fresh_run"}`), CreatedAt: execution.UpdatedAt}
	if ok, err := repo.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: 1, Execution: execution, Attempt: attempt, Journal: []*domain.WorkflowJournalEntry{entry}}); err != nil || !ok {
		t.Fatalf("commit ok=%t err=%v", ok, err)
	}
	attempt.Status, attempt.RunID = domain.WorkflowAttemptDispatched, &runID
	attempt.Version++
	execution.Version++
	execution.UpdatedAt = now.Add(2 * time.Second)
	if ok, err := repo.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: 2, Execution: execution, Attempt: attempt}); err != nil || !ok {
		t.Fatalf("dispatch acknowledgement commit ok=%t err=%v", ok, err)
	}

	got, err := repo.ExecutionIDForRun(ctx, runID)
	if err != nil || got != execution.ID {
		t.Fatalf("ExecutionIDForRun(dispatched run) = %v err=%v, want %v", got, err, execution.ID)
	}

	// A run that belongs to no workflow attempt resolves to uuid.Nil, nil — the
	// common non-workflow run the completion nudge must ignore.
	orphan, err := repo.ExecutionIDForRun(ctx, uuid.New())
	if err != nil || orphan != uuid.Nil {
		t.Fatalf("ExecutionIDForRun(non-workflow run) = %v err=%v, want Nil,nil", orphan, err)
	}
}

func TestWorkflowExecutionRepositoryRecoveryUsesCurrentCleanupGeneration(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	catalog := &workflowRepository{db: db, log: logrus.New()}
	revision := workflowRevision("owner/flow", "sha256:cleanup-generation", "1.0.0")
	if err := catalog.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatal(err)
	}
	repo := &workflowExecutionRepository{db: db, log: logrus.New()}
	now := time.Now().UTC()
	execution := &domain.WorkflowExecution{
		ID: uuid.New(), Owner: "owner", WorkflowKey: "owner/flow", DefinitionDigest: revision.Digest,
		Status: domain.WorkflowExecutionFailed, CurrentNodeID: "start", Input: json.RawMessage(`{}`),
		BudgetUsage: domain.WorkflowBudgetUsage{Retries: 2, AccountingComplete: true}, EdgeTraversals: map[string]int{},
		Version: 1, IdempotencyKey: "cleanup-generation", CreatedAt: now, UpdatedAt: now, EndedAt: &now,
	}
	initial := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: execution.ID, Sequence: 1, Kind: domain.WorkflowJournalInput, Payload: json.RawMessage(`{}`), CreatedAt: now}
	if err := repo.Create(ctx, execution, initial); err != nil {
		t.Fatal(err)
	}
	recoverable, err := repo.ListRecoverable(ctx, 10)
	if err != nil || len(recoverable) != 1 {
		t.Fatalf("terminal execution without cleanup should be recoverable: %+v err=%v", recoverable, err)
	}
	execution.Version++
	execution.UpdatedAt = now.Add(time.Second)
	disposition := &domain.WorkflowJournalEntry{
		ID: uuid.New(), ExecutionID: execution.ID, Sequence: 2, Kind: domain.WorkflowJournalCleanup,
		Payload: json.RawMessage(`{"retry":2,"stoppedRuns":1}`), CreatedAt: execution.UpdatedAt,
	}
	ok, err := repo.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: 1, Execution: execution, Journal: []*domain.WorkflowJournalEntry{disposition}})
	if err != nil || !ok {
		t.Fatalf("cleanup commit ok=%t err=%v", ok, err)
	}
	recoverable, err = repo.ListRecoverable(ctx, 10)
	if err != nil || len(recoverable) != 0 {
		t.Fatalf("current cleanup generation remained recoverable: %+v err=%v", recoverable, err)
	}
}

func TestWorkflowExecutionRepositoryExcludesFinalizedUnknownAccounting(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	catalog := &workflowRepository{db: db, log: logrus.New()}
	revision := workflowRevision("owner/flow", "sha256:finalized-unknown", "1.0.0")
	if err := catalog.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatal(err)
	}
	repo := &workflowExecutionRepository{db: db, log: logrus.New()}
	now := time.Now().UTC()
	execution := &domain.WorkflowExecution{
		ID: uuid.New(), Owner: "owner", WorkflowKey: "owner/flow", DefinitionDigest: revision.Digest,
		Status: domain.WorkflowExecutionFailed, CurrentNodeID: "start", Input: json.RawMessage(`{}`),
		BudgetUsage:    domain.WorkflowBudgetUsage{AccountingFinalizedUnknown: true, AccountingComplete: false},
		EdgeTraversals: map[string]int{}, Version: 1, IdempotencyKey: "finalized-unknown", CreatedAt: now, UpdatedAt: now, EndedAt: &now,
	}
	initial := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: execution.ID, Sequence: 1, Kind: domain.WorkflowJournalInput, Payload: json.RawMessage(`{}`), CreatedAt: now}
	if err := repo.Create(ctx, execution, initial); err != nil {
		t.Fatal(err)
	}
	recoverable, err := repo.ListRecoverable(ctx, 10)
	if err != nil || len(recoverable) != 0 {
		t.Fatalf("finalized unknown execution remained recoverable: %+v err=%v", recoverable, err)
	}
}

func workflowRevision(key, digest, version string) *domain.WorkflowRevision {
	return &domain.WorkflowRevision{ID: uuid.New(), Owner: "owner", Key: key, SemanticVersion: version, Digest: digest, Definition: domain.WorkflowDefinition{SchemaVersion: domain.WorkflowSchemaVersionV1, Owner: "owner", Key: key, Version: version, InputSchema: json.RawMessage(`{}`), OutputSchema: json.RawMessage(`{}`)}, SourcePath: ".vrooli/agent-workflows/flow.json", SourceHash: digest, SourceUpdatedAt: time.Now().UTC(), CreatedAt: time.Now().UTC()}
}
