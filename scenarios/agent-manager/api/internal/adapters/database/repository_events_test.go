package database

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/repository"
	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite" // SQLite driver for tests
)

// projectStatsRun seeds the authoritative read model used by stats queries.
// Run persistence is intentionally not an implicit analytics write path.
func projectStatsRun(t *testing.T, repos *Repositories, ctx context.Context, run *domain.Run, at time.Time) {
	t.Helper()
	config := run.ResolvedConfig
	if config == nil {
		config = &domain.RunConfig{}
	}
	if err := repos.InvocationReadModel.(invocationreadmodel.RunStore).ReplaceRun(ctx, invocationreadmodel.RunFact{
		RunID: run.ID.String(), OccurredAt: at, CreatedAt: at, Status: string(run.Status), Tag: run.Tag,
		RunnerType: string(config.RunnerType), Model: config.Model, ProjectedAt: at,
	}); err != nil {
		t.Fatalf("project run fact: %v", err)
	}
}

func TestCheckpointRepository(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create task and run first
	task := &domain.Task{ID: uuid.New(), Title: "Checkpoint Task", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	checkpoint := &domain.RunCheckpoint{
		RunID:             run.ID,
		Phase:             domain.RunPhaseExecuting,
		StepWithinPhase:   5,
		WorkDir:           "/tmp/work",
		LastEventSequence: 10,
		LastHeartbeat:     time.Now(),
		RetryCount:        0,
		Metadata:          map[string]string{"key": "value"},
	}

	// Save (insert)
	if err := repos.Checkpoints.Save(ctx, checkpoint); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Get
	got, err := repos.Checkpoints.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Phase != domain.RunPhaseExecuting {
		t.Errorf("expected phase executing, got %q", got.Phase)
	}
	if got.StepWithinPhase != 5 {
		t.Errorf("expected step 5, got %d", got.StepWithinPhase)
	}
	if got.Metadata["key"] != "value" {
		t.Errorf("expected metadata key=value, got %v", got.Metadata)
	}

	// Save (upsert)
	checkpoint.StepWithinPhase = 10
	checkpoint.LastEventSequence = 20
	if err := repos.Checkpoints.Save(ctx, checkpoint); err != nil {
		t.Fatalf("Save update: %v", err)
	}

	got, err = repos.Checkpoints.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.StepWithinPhase != 10 {
		t.Errorf("expected step 10, got %d", got.StepWithinPhase)
	}

	// Heartbeat
	if err := repos.Checkpoints.Heartbeat(ctx, run.ID); err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}

	// ListStale - checkpoint is fresh, should not appear
	stale, err := repos.Checkpoints.ListStale(ctx, 1*time.Hour)
	if err != nil {
		t.Fatalf("ListStale: %v", err)
	}
	if len(stale) != 0 {
		t.Errorf("expected 0 stale checkpoints, got %d", len(stale))
	}

	// Delete
	if err := repos.Checkpoints.Delete(ctx, run.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, err = repos.Checkpoints.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

// ============================================================================
// Idempotency Repository Tests
// ============================================================================

func TestIdempotencyRepository(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	key := "test-idempotency-key"
	ttl := 1 * time.Hour

	// Check non-existent key
	record, err := repos.Idempotency.Check(ctx, key)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if record != nil {
		t.Error("expected nil for non-existent key")
	}

	// Reserve
	reserved, err := repos.Idempotency.Reserve(ctx, key, ttl)
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if reserved == nil {
		t.Fatal("Reserve returned nil")
	}
	if reserved.Status != domain.IdempotencyStatusPending {
		t.Errorf("expected pending status, got %q", reserved.Status)
	}

	// Check reserved key
	record, err = repos.Idempotency.Check(ctx, key)
	if err != nil {
		t.Fatalf("Check reserved: %v", err)
	}
	if record == nil {
		t.Fatal("expected record for reserved key")
	}
	if record.Status != domain.IdempotencyStatusPending {
		t.Errorf("expected pending status, got %q", record.Status)
	}

	// Complete
	entityID := uuid.New()
	if err := repos.Idempotency.Complete(ctx, key, entityID, "run", []byte(`{"result": "ok"}`)); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	record, err = repos.Idempotency.Check(ctx, key)
	if err != nil {
		t.Fatalf("Check completed: %v", err)
	}
	if record.Status != domain.IdempotencyStatusComplete {
		t.Errorf("expected complete status, got %q", record.Status)
	}
	if record.EntityID == nil || *record.EntityID != entityID {
		t.Errorf("expected entity ID %s, got %v", entityID, record.EntityID)
	}

	// Fail a different key
	failKey := "fail-key"
	if _, err := repos.Idempotency.Reserve(ctx, failKey, ttl); err != nil {
		t.Fatalf("Reserve fail key: %v", err)
	}
	if err := repos.Idempotency.Fail(ctx, failKey); err != nil {
		t.Fatalf("Fail: %v", err)
	}

	record, err = repos.Idempotency.Check(ctx, failKey)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if record.Status != domain.IdempotencyStatusFailed {
		t.Errorf("expected failed status, got %q", record.Status)
	}
}

func TestIdempotencyExpiration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Reserve with very short TTL (already expired)
	key := "expired-key"
	ttl := -1 * time.Hour // Negative TTL = already expired
	if _, err := repos.Idempotency.Reserve(ctx, key, ttl); err != nil {
		t.Fatalf("Reserve: %v", err)
	}

	// Check should return nil for expired key
	record, err := repos.Idempotency.Check(ctx, key)
	if err != nil {
		t.Fatalf("Check expired: %v", err)
	}
	if record != nil {
		t.Error("expected nil for expired key")
	}

	// Cleanup expired
	count, err := repos.Idempotency.CleanupExpired(ctx)
	if err != nil {
		t.Fatalf("CleanupExpired: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 cleaned up, got %d", count)
	}
}

func TestContinuationIdempotencySurvivesExpiryAndRejectsCollisions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	runID := uuid.New()
	requestHash := []byte("exact-continuation-request-hash")

	// The pending receipt is already expired by wall clock, but timeout alone
	// must not make an effect-possible continuation reusable.
	key := "continuation-expired-pending"
	if _, acquired, err := repos.Idempotency.(repository.ContinuationIdempotencyRepository).ReserveContinuation(ctx, key, runID, requestHash, -time.Hour); err != nil || !acquired {
		t.Fatalf("ReserveContinuation = acquired %v, err %v", acquired, err)
	}
	if got, err := repos.Idempotency.Check(ctx, key); err != nil || got != nil {
		t.Fatalf("generic Check = %+v, %v; expired records should remain hidden to unrelated callers", got, err)
	}
	if _, err := repos.Idempotency.Reserve(ctx, key, time.Hour); err == nil {
		t.Fatal("generic Reserve reclaimed a continuation-owned pending key")
	}
	if count, err := repos.Idempotency.CleanupExpired(ctx); err != nil || count != 0 {
		t.Fatalf("CleanupExpired = %d, %v; effect-possible continuation should be retained", count, err)
	}

	continuations := repos.Idempotency.(repository.ContinuationIdempotencyRepository)
	if got, err := continuations.CheckContinuation(ctx, key); err != nil || got == nil || got.Status != domain.IdempotencyStatusPending {
		t.Fatalf("typed pending check = %+v, %v", got, err)
	}
	if _, acquired, err := continuations.ReserveContinuation(ctx, key, uuid.New(), requestHash, time.Hour); err == nil || acquired {
		t.Fatal("same key was accepted for a different run")
	}
	if _, acquired, err := continuations.ReserveContinuation(ctx, key, runID, []byte("changed-request"), time.Hour); err == nil || acquired {
		t.Fatal("same key was accepted for a different request hash")
	}
	if err := continuations.CompleteContinuation(ctx, key, runID, requestHash); err != nil {
		t.Fatalf("CompleteContinuation after expiry: %v", err)
	}
	if count, err := repos.Idempotency.CleanupExpired(ctx); err != nil || count != 0 {
		t.Fatalf("CleanupExpired removed completed continuation: %d, %v", count, err)
	}
	if _, err := repos.Idempotency.Reserve(ctx, key, time.Hour); err == nil {
		t.Fatal("generic Reserve reused a completed continuation key")
	}
	if got, err := continuations.CheckContinuation(ctx, key); err != nil || got == nil || got.Status != domain.IdempotencyStatusComplete {
		t.Fatalf("typed completed check = %+v, %v", got, err)
	}
	if err := continuations.CompleteContinuation(ctx, key, runID, requestHash); err != nil {
		t.Fatalf("idempotent duplicate completion: %v", err)
	}
	if err := continuations.CompleteContinuation(ctx, key, uuid.New(), requestHash); err == nil {
		t.Fatal("completion acknowledged a changed run identity")
	}
	if err := repos.Idempotency.Complete(ctx, key, runID, "other-operation", nil); err == nil {
		t.Fatal("generic completion acknowledged a continuation-owned key")
	}

	failedKey := "continuation-proven-pre-effect-failure"
	failedRun := uuid.New()
	failedHash := []byte("pre-effect-request")
	if _, acquired, err := continuations.ReserveContinuation(ctx, failedKey, failedRun, failedHash, time.Hour); err != nil || !acquired {
		t.Fatalf("reserve failure control: acquired %v err %v", acquired, err)
	}
	if err := continuations.FailContinuation(ctx, failedKey, failedRun, []byte("wrong-hash")); err == nil {
		t.Fatal("wrong request hash marked a continuation retryable")
	}
	if got, err := continuations.CheckContinuation(ctx, failedKey); err != nil || got == nil || got.Status != domain.IdempotencyStatusPending {
		t.Fatalf("wrong-hash failure changed receipt: %+v, %v", got, err)
	}
	if err := continuations.FailContinuation(ctx, failedKey, failedRun, failedHash); err != nil {
		t.Fatalf("mark proven pre-effect failure: %v", err)
	}
	if _, acquired, err := continuations.ReserveContinuation(ctx, failedKey, failedRun, failedHash, time.Hour); err != nil || !acquired {
		t.Fatalf("exact pre-effect retry was not acquired: acquired %v err %v", acquired, err)
	}

	// Generic operation expiry remains reusable and cleanup still removes its
	// expired records.
	ordinary := "ordinary-expired-operation"
	if _, err := repos.Idempotency.Reserve(ctx, ordinary, -time.Hour); err != nil {
		t.Fatalf("Reserve ordinary expired: %v", err)
	}
	if _, err := repos.Idempotency.Reserve(ctx, ordinary, time.Hour); err != nil {
		t.Fatalf("generic reserve lost ordinary expiry behavior: %v", err)
	}
}

func TestContinuationIdempotencyFailsClosedOnLegacyPendingAndMissingComplete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	continuations := repos.Idempotency.(repository.ContinuationIdempotencyRepository)

	legacyKey := "legacy-untyped-pending"
	if _, err := repos.Idempotency.Reserve(ctx, legacyKey, -time.Hour); err != nil {
		t.Fatal(err)
	}
	if got, err := continuations.CheckContinuation(ctx, legacyKey); err != nil || got == nil || got.EntityType != "" {
		t.Fatalf("legacy pending receipt = %+v, %v", got, err)
	}
	if _, acquired, err := continuations.ReserveContinuation(ctx, legacyKey, uuid.New(), []byte("hash"), time.Hour); err == nil || acquired {
		t.Fatal("ambiguous legacy pending row was reclaimed")
	}
	legacyCompleteKey := "legacy-expired-complete"
	legacyRun := uuid.New()
	legacyHash := []byte("legacy-request-hash")
	if _, err := repos.Idempotency.Reserve(ctx, legacyCompleteKey, -time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := repos.Idempotency.Complete(ctx, legacyCompleteKey, legacyRun, "Continuation", legacyHash); err != nil {
		t.Fatalf("create old-format completed continuation receipt: %v", err)
	}
	if count, err := repos.Idempotency.CleanupExpired(ctx); err != nil || count != 1 {
		t.Fatalf("cleanup should remove only ambiguous legacy pending row, preserving typed completion: count %d err %v", count, err)
	}
	if got, err := continuations.CheckContinuation(ctx, legacyCompleteKey); err != nil || got == nil || got.Status != domain.IdempotencyStatusComplete || got.EntityID == nil || *got.EntityID != legacyRun || string(got.Response) != string(legacyHash) {
		t.Fatalf("legacy expired completion could not recover identity: %+v err %v", got, err)
	}

	missingKey := "deleted-before-completion"
	runID := uuid.New()
	hash := []byte("hash")
	if _, acquired, err := continuations.ReserveContinuation(ctx, missingKey, runID, hash, time.Hour); err != nil || !acquired {
		t.Fatalf("reserve missing-row control: acquired %v err %v", acquired, err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM idempotency_records WHERE key = ?`, missingKey); err != nil {
		t.Fatal(err)
	}
	if err := continuations.CompleteContinuation(ctx, missingKey, runID, hash); err == nil {
		t.Fatal("completion silently acknowledged a missing receipt")
	}
}

// ============================================================================
// Policy Repository Tests
// ============================================================================

func TestPolicyCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	policy := &domain.Policy{
		ID:           uuid.New(),
		Name:         "test-policy",
		Description:  "A test policy",
		Priority:     100,
		ScopePattern: "/home/*",
		Rules: domain.PolicyRules{
			RequireSandbox:    func() *bool { b := true; return &b }(),
			RequireApproval:   func() *bool { b := false; return &b }(),
			MaxConcurrentRuns: func() *int { n := 5; return &n }(),
			AllowedRunners:    []domain.RunnerType{domain.RunnerTypeClaudeCode},
		},
		Enabled:   true,
		CreatedBy: "test-user",
	}

	// Create
	if err := repos.Policies.Create(ctx, policy); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get
	got, err := repos.Policies.Get(ctx, policy.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Name != policy.Name {
		t.Errorf("expected name %q, got %q", policy.Name, got.Name)
	}
	if got.Priority != 100 {
		t.Errorf("expected priority 100, got %d", got.Priority)
	}
	if len(got.Rules.AllowedRunners) != 1 {
		t.Errorf("expected 1 allowed runner, got %d", len(got.Rules.AllowedRunners))
	}
	if got.Rules.RequireSandbox == nil || !*got.Rules.RequireSandbox {
		t.Error("expected RequireSandbox to be true")
	}

	// List
	policies, err := repos.Policies.List(ctx, repository.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(policies))
	}

	// ListEnabled
	enabledPolicies, err := repos.Policies.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("ListEnabled: %v", err)
	}
	if len(enabledPolicies) != 1 {
		t.Errorf("expected 1 enabled policy, got %d", len(enabledPolicies))
	}

	// Update - disable
	policy.Enabled = false
	if err := repos.Policies.Update(ctx, policy); err != nil {
		t.Fatalf("Update: %v", err)
	}

	enabledPolicies, err = repos.Policies.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("ListEnabled after disable: %v", err)
	}
	if len(enabledPolicies) != 0 {
		t.Errorf("expected 0 enabled policies, got %d", len(enabledPolicies))
	}

	// Delete
	if err := repos.Policies.Delete(ctx, policy.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, err = repos.Policies.Get(ctx, policy.ID)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

// ============================================================================
// JSONB Type Tests
// ============================================================================

func TestJSONBTypes(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Test complex task with context attachments
	attachment1 := domain.ContextAttachment{
		Type:    "file",
		Path:    "/path/to/file.txt",
		Content: "file content",
	}
	attachment2 := domain.ContextAttachment{
		Type:    "url",
		Path:    "https://example.com",
		Content: "",
	}

	task := &domain.Task{
		ID:                 uuid.New(),
		Title:              "JSONB Test Task",
		ScopePath:          "/test",
		Status:             domain.TaskStatusQueued,
		PhasePromptIDs:     []uuid.UUID{uuid.New(), uuid.New()},
		ContextAttachments: []domain.ContextAttachment{attachment1, attachment2},
	}

	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task with attachments: %v", err)
	}

	got, err := repos.Tasks.Get(ctx, task.ID)
	if err != nil {
		t.Fatalf("Get task: %v", err)
	}

	if len(got.PhasePromptIDs) != 2 {
		t.Errorf("expected 2 phase prompt IDs, got %d", len(got.PhasePromptIDs))
	}

	if len(got.ContextAttachments) != 2 {
		t.Errorf("expected 2 context attachments, got %d", len(got.ContextAttachments))
	}

	if got.ContextAttachments[0].Type != "file" {
		t.Errorf("expected attachment type 'file', got %q", got.ContextAttachments[0].Type)
	}
}

func TestRunWithComplexFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create task
	task := &domain.Task{ID: uuid.New(), Title: "Complex Run Task", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	// Create profile
	profile := &domain.AgentProfile{ID: uuid.New(), Name: "complex-profile", ProfileKey: "complex-profile", RoleRef: "code.default"}
	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatalf("Create profile: %v", err)
	}

	sandboxID := uuid.New()
	startedAt := time.Now()
	exitCode := 0

	run := &domain.Run{
		ID:              uuid.New(),
		TaskID:          task.ID,
		AgentProfileID:  &profile.ID,
		SandboxID:       &sandboxID,
		Tag:             "complex-run",
		RunMode:         domain.RunModeSandboxed,
		Status:          domain.RunStatusComplete,
		Phase:           domain.RunPhaseCompleted,
		StartedAt:       &startedAt,
		ProgressPercent: 100,
		ApprovalState:   domain.ApprovalStateApproved,
		ExitCode:        &exitCode,
		Summary: &domain.RunSummary{
			Description:   "Completed successfully",
			FilesModified: []string{"file1.go", "file2.go", "file3.go"},
			FilesCreated:  []string{"new_file.go"},
			TokensUsed:    1000,
			TurnsUsed:     10,
			CostEstimate:  0.05,
		},
		Result: &domain.RunResult{
			FinalOutput: "Completed successfully",
			Selection: domain.FinalOutputSelection{
				Status:              domain.FinalOutputSelectionSelected,
				SelectedCandidateID: "candidate-1",
				Rule:                "unique_terminal_main_assistant",
				AlgorithmVersion:    domain.FinalOutputResolverVersion,
			},
			Candidates: []domain.FinalOutputCandidate{{ID: "candidate-1", Content: "Completed successfully", Terminal: true, EvidenceTier: 3}},
			Success:    true,
			Structured: &domain.StructuredResult{
				Status: domain.StructuredResultSuccess, SpecKind: domain.ResultSpecKindClassification,
				SchemaDigest: "sha256:result-schema", Value: json.RawMessage(`"complete"`),
				Method: "whole_document", SourceCandidateID: "candidate-1",
			},
		},
		ResolvedConfig: &domain.RunConfig{
			MaxTurns:     100,
			AllowedTools: []string{"read", "write", "bash"},
			DeniedTools:  []string{},
			ResultSpec: &domain.ResultSpec{
				Version: "result-spec/v1", Kind: domain.ResultSpecKindClassification,
				Schema: json.RawMessage(`{"enum":["complete"],"type":"string"}`), SchemaDigest: "sha256:result-schema",
				ExtractionMode: domain.StructuredExtractionDeterministic,
			},
			PolicySnapshot: &domain.ExecutionPolicySnapshot{
				CatalogDigest: "sha256:test-revision",
				RoleRef:       "code.smart",
				Candidates: []domain.ExecutionCandidate{{
					RunnerType:    domain.RunnerTypeCodex,
					SelectionType: domain.ModelSelectionTypeModel,
					Model:         "gpt-test",
				}},
				SelectedIndex: 0,
				SelectedCandidate: domain.ExecutionCandidate{
					RunnerType:    domain.RunnerTypeCodex,
					SelectionType: domain.ModelSelectionTypeModel,
					Model:         "gpt-test",
				},
				Explanation: domain.PolicyResolutionExplanation{
					Source:  "named_policy",
					Summary: "repository round trip",
				},
			},
		},
		DiffPath:       "/path/to/diff",
		LogPath:        "/path/to/log",
		ChangedFiles:   3,
		TotalSizeBytes: 1024,
	}

	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get run: %v", err)
	}

	// Verify nullable UUID fields
	if got.AgentProfileID == nil || *got.AgentProfileID != profile.ID {
		t.Errorf("expected agent profile ID %s, got %v", profile.ID, got.AgentProfileID)
	}
	if got.SandboxID == nil || *got.SandboxID != sandboxID {
		t.Errorf("expected sandbox ID %s, got %v", sandboxID, got.SandboxID)
	}

	// Verify nullable time fields
	if got.StartedAt == nil {
		t.Error("expected started_at to be set")
	}

	// Verify exit code
	if got.ExitCode == nil || *got.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %v", got.ExitCode)
	}

	// Verify summary JSONB
	if got.Summary == nil {
		t.Fatal("expected summary to be set")
	}
	if len(got.Summary.FilesModified) != 3 {
		t.Errorf("expected 3 files modified, got %d", len(got.Summary.FilesModified))
	}
	if got.Summary.TokensUsed != 1000 {
		t.Errorf("expected 1000 tokens used, got %d", got.Summary.TokensUsed)
	}
	if got.Result == nil || got.Result.Selection.Status != domain.FinalOutputSelectionSelected || got.Result.FinalOutput != "Completed successfully" {
		t.Fatalf("run result round trip = %#v", got.Result)
	}
	if got.Result.Structured == nil || got.Result.Structured.Status != domain.StructuredResultSuccess || string(got.Result.Structured.Value) != `"complete"` {
		t.Fatalf("structured result round trip = %#v", got.Result.Structured)
	}

	// Verify resolved config JSONB
	if got.ResolvedConfig == nil {
		t.Fatal("expected resolved config to be set")
	}
	if got.ResolvedConfig.MaxTurns != 100 {
		t.Errorf("expected max turns 100, got %d", got.ResolvedConfig.MaxTurns)
	}
	if len(got.ResolvedConfig.AllowedTools) != 3 {
		t.Errorf("expected 3 allowed tools, got %d", len(got.ResolvedConfig.AllowedTools))
	}
	if got.ResolvedConfig.PolicySnapshot == nil {
		t.Fatal("expected policy snapshot to be persisted")
	}
	if got.ResolvedConfig.ResultSpec == nil || got.ResolvedConfig.ResultSpec.SchemaDigest != "sha256:result-schema" {
		t.Fatalf("result spec round trip = %+v", got.ResolvedConfig.ResultSpec)
	}
	if got.ResolvedConfig.PolicySnapshot.CatalogDigest != "sha256:test-revision" ||
		got.ResolvedConfig.PolicySnapshot.SelectedCandidate.Model != "gpt-test" {
		t.Fatalf("policy snapshot round trip = %+v", got.ResolvedConfig.PolicySnapshot)
	}
}

// ============================================================================
// Stats Repository Time Scanning Tests
// ============================================================================

func TestStatsGetTimeSeries_DailyBucketParsesTimestamp(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Stats Task",
		ScopePath: "/stats",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	run := &domain.Run{
		ID:            uuid.New(),
		TaskID:        task.ID,
		Tag:           "stats-time-series",
		Status:        domain.RunStatusComplete,
		Phase:         domain.RunPhaseCompleted,
		ApprovalState: domain.ApprovalStateApproved,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeClaudeCode,
			Model:      "claude-sonnet-4-20250514",
		},
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	now := time.Now()
	projectStatsRun(t, repos, ctx, run, now)
	filter := repository.StatsFilter{
		Window: repository.StatsTimeWindow{
			Start: now.Add(-24 * time.Hour),
			End:   now.Add(24 * time.Hour),
		},
	}

	buckets, err := repos.Stats.GetTimeSeries(ctx, filter, 24*time.Hour)
	if err != nil {
		t.Fatalf("GetTimeSeries: %v", err)
	}
	if len(buckets) == 0 {
		t.Fatal("expected at least one time-series bucket")
	}
	if buckets[0].Timestamp.IsZero() {
		t.Fatal("expected non-zero timestamp in first bucket")
	}
}

func TestStatsUsageQueries_ParseCreatedAtTimestamps(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Usage Stats Task",
		ScopePath: "/stats/usage",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	run := &domain.Run{
		ID:            uuid.New(),
		TaskID:        task.ID,
		Tag:           "stats-usage",
		Status:        domain.RunStatusComplete,
		Phase:         domain.RunPhaseCompleted,
		ApprovalState: domain.ApprovalStateApproved,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeClaudeCode,
			Model:      "claude-sonnet-4-20250514",
		},
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	toolCall := domain.NewToolCallEvent(run.ID, "read", "call-1", map[string]interface{}{"path": "README.md"})
	toolResult := domain.NewToolResultEvent(run.ID, "read", "call-1", "ok", nil)
	if err := event.NewSQLiteStore(db, logrus.New()).Append(ctx, run.ID, toolCall, toolResult); err != nil {
		t.Fatalf("Append events: %v", err)
	}

	now := time.Now()
	projectStatsRun(t, repos, ctx, run, now)
	if err := repos.InvocationReadModel.Replace(ctx, []invocationreadmodel.Fact{{InvocationFact: runsignal.InvocationFact{Version: runsignal.InvocationFactVersion, CallEventID: "call-1", ToolName: "read", Ownership: "project", Outcome: "success", Fingerprint: "read", Availability: "available"}, RunID: run.ID.String(), OccurredAt: now, TimeBasis: "call_event", RunnerType: string(run.ResolvedConfig.RunnerType), Model: run.ResolvedConfig.Model, RunStatus: string(run.Status)}}, invocationreadmodel.Watermark{RunID: run.ID.String(), LastEventID: "call-1", LastEventAt: now, ClassifierVersion: runsignal.InvocationFactVersion, ProjectedAt: now}); err != nil {
		t.Fatalf("project invocation fact: %v", err)
	}
	filter := repository.StatsFilter{
		Window: repository.StatsTimeWindow{
			Start: now.Add(-24 * time.Hour),
			End:   now.Add(24 * time.Hour),
		},
	}

	modelRuns, err := repos.Stats.GetModelRunUsage(ctx, filter, "claude-sonnet-4-20250514", 10)
	if err != nil {
		t.Fatalf("GetModelRunUsage: %v", err)
	}
	if len(modelRuns) == 0 {
		t.Fatal("expected model usage rows")
	}
	if modelRuns[0].CreatedAt.IsZero() {
		t.Fatal("expected model usage created_at to be parsed")
	}

	toolRuns, err := repos.Stats.GetToolRunUsage(ctx, filter, "read", 10)
	if err != nil {
		t.Fatalf("GetToolRunUsage: %v", err)
	}
	if len(toolRuns) == 0 {
		t.Fatal("expected tool usage rows")
	}
	if toolRuns[0].CreatedAt.IsZero() {
		t.Fatal("expected tool usage created_at to be parsed")
	}
}

func TestStatsGetErrorPatterns_ParsesLastSeenTimestamp(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Error Stats Task",
		ScopePath: "/stats/errors",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	run := &domain.Run{
		ID:            uuid.New(),
		TaskID:        task.ID,
		Tag:           "stats-errors",
		Status:        domain.RunStatusFailed,
		Phase:         domain.RunPhaseCompleted,
		ApprovalState: domain.ApprovalStateNone,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	errEvent := domain.NewErrorEvent(run.ID, "TOOL_FAILED", "tool execution failed", true)
	if err := event.NewSQLiteStore(db, logrus.New()).Append(ctx, run.ID, errEvent); err != nil {
		t.Fatalf("Append error event: %v", err)
	}

	now := time.Now()
	projectStatsRun(t, repos, ctx, run, now)
	if _, err := db.ExecContext(ctx, `INSERT INTO invocation_read_model_errors (run_id,event_id,occurred_at,time_basis,error_code,profile_id,runner_type,model,tag) VALUES (?,?,?,?,?,?,?,?,?)`, run.ID.String(), "error-1", SQLiteTime(now), "error_event", "TOOL_FAILED", "", "", "", run.Tag); err != nil {
		t.Fatalf("project error fact: %v", err)
	}
	filter := repository.StatsFilter{
		Window: repository.StatsTimeWindow{
			Start: now.Add(-24 * time.Hour),
			End:   now.Add(24 * time.Hour),
		},
	}

	patterns, err := repos.Stats.GetErrorPatterns(ctx, filter, 10)
	if err != nil {
		t.Fatalf("GetErrorPatterns: %v", err)
	}
	if len(patterns) == 0 {
		t.Fatal("expected error patterns rows")
	}
	if patterns[0].LastSeen.IsZero() {
		t.Fatal("expected non-zero last_seen timestamp")
	}
}

func TestIdempotencyReserveReclaimsExpiredAndFailed(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// A live pending reservation blocks a second reserve.
	live := "live-key"
	if _, err := repos.Idempotency.Reserve(ctx, live, time.Hour); err != nil {
		t.Fatalf("Reserve live: %v", err)
	}
	if _, err := repos.Idempotency.Reserve(ctx, live, time.Hour); err == nil {
		t.Fatal("second reserve on a live key must fail")
	}

	// A completed reservation stays held even after a reserve attempt.
	done := "done-key"
	if _, err := repos.Idempotency.Reserve(ctx, done, time.Hour); err != nil {
		t.Fatalf("Reserve done: %v", err)
	}
	if err := repos.Idempotency.Complete(ctx, done, uuid.New(), "run", nil); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if _, err := repos.Idempotency.Reserve(ctx, done, time.Hour); err == nil {
		t.Fatal("reserve on a completed key must fail")
	}

	// An expired pending reservation (crash between Reserve and Complete) is
	// reclaimed by the next reserve instead of wedging the key forever.
	expired := "expired-key"
	if _, err := repos.Idempotency.Reserve(ctx, expired, -time.Minute); err != nil {
		t.Fatalf("Reserve expired: %v", err)
	}
	if _, err := repos.Idempotency.Reserve(ctx, expired, time.Hour); err != nil {
		t.Fatalf("reserve must reclaim an expired reservation: %v", err)
	}
	record, err := repos.Idempotency.Check(ctx, expired)
	if err != nil || record == nil || record.Status != domain.IdempotencyStatusPending {
		t.Fatalf("reclaimed key should be live pending, got %+v err=%v", record, err)
	}

	// A failed reservation is reclaimed (the documented allow-retry path).
	failed := "failed-key"
	if _, err := repos.Idempotency.Reserve(ctx, failed, time.Hour); err != nil {
		t.Fatalf("Reserve failed-key: %v", err)
	}
	if err := repos.Idempotency.Fail(ctx, failed); err != nil {
		t.Fatalf("Fail: %v", err)
	}
	if _, err := repos.Idempotency.Reserve(ctx, failed, time.Hour); err != nil {
		t.Fatalf("reserve must reclaim a failed reservation: %v", err)
	}
}
