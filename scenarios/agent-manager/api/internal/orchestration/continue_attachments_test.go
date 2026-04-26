package orchestration_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/storage"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
)

// setupContinueTest creates an orchestrator wired with the given options and
// inserts a run directly into the repository in a continuable state (status
// complete, session ID set, resolved config with runner type). This avoids
// calling CreateRun, which triggers async execution.
func setupContinueTest(t *testing.T, opts ...orchestration.Option) (
	svc orchestration.Service,
	run *domain.Run,
	cleanup func(),
) {
	t.Helper()
	ctx := context.Background()

	repos, eventStore, dbCleanup := testutil.SetupTestRepos(t)

	// Build orchestrator options: base config + caller-supplied options.
	baseOpts := []orchestration.Option{
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          5 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: false,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
	}
	allOpts := append(baseOpts, opts...)

	svc = orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		allOpts...,
	)

	// Create a profile via the service (so it exists in DB for lookups).
	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:            "continue-attach-profile",
		ProfileKey:      "continue-attach-" + uuid.New().String()[:8],
		RunnerType:      domain.RunnerTypeClaudeCode,
		RequiresSandbox: false,
	})

	// Create a task via the service.
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:       "continue-attach-task",
		Description: "task for attachment continuation tests",
		ScopePath:   "src/",
	})

	// Insert a run directly into the repository in a continuable state,
	// bypassing CreateRun (which triggers async execution).
	now := time.Now()
	runID := uuid.New()
	directRun := &domain.Run{
		ID:             runID,
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Tag:            runID.String(),
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusComplete,
		Phase:          domain.RunPhaseCompleted,
		SessionID:      "test-session-" + uuid.New().String()[:8],
		StartedAt:      &now,
		EndedAt:        &now,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeClaudeCode,
		},
		ApprovalState: domain.ApprovalStateNone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repos.Runs.Create(ctx, directRun); err != nil {
		dbCleanup()
		t.Fatalf("Create run directly: %v", err)
	}

	// Re-fetch to get the persisted version.
	run, err := svc.GetRun(ctx, runID)
	if err != nil {
		dbCleanup()
		t.Fatalf("GetRun after direct create: %v", err)
	}

	return svc, run, dbCleanup
}

// newContinuationRunner creates a mock runner with SupportsContinuation=true
// and a ContinueFunc that captures the request and signals via done channel.
func newContinuationRunner(t *testing.T) (
	mockRunner *runner.MockRunner,
	captured *runner.ContinueRequest,
	captureMu *sync.Mutex,
	done chan struct{},
) {
	t.Helper()
	captured = &runner.ContinueRequest{}
	captureMu = &sync.Mutex{}
	done = make(chan struct{})

	mockRunner = runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "available")
	mockRunner.SetCapabilities(runner.Capabilities{
		SupportsMessages:         true,
		SupportsStreaming:        true,
		SupportsCancellation:     true,
		SupportsContinuation:     true,
		SupportsImageAttachments: true,
		MaxTurns:                 100,
		SupportedModels:          []string{"mock-model"},
	})
	mockRunner.ContinueFunc = func(ctx context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
		captureMu.Lock()
		*captured = req
		captureMu.Unlock()
		close(done)
		return &runner.ExecuteResult{
			Success:   true,
			ExitCode:  0,
			SessionID: req.SessionID,
			Summary: &domain.RunSummary{
				Description: "Mock continuation completed",
			},
		}, nil
	}
	return
}

// TestContinueRun_WithAttachmentIDs verifies that when AttachmentIDs are
// provided, the orchestrator resolves them via storage and passes them to the
// runner's ContinueRequest.
func TestContinueRun_WithAttachmentIDs(t *testing.T) {
	// Set up mock storage with pre-set attachments.
	mockStorage := storage.NewMockService()
	mockStorage.SetFile("att-1", &storage.AttachmentMeta{
		ID:          "att-1",
		FileName:    "screenshot.png",
		ContentType: "image/png",
		FileSize:    1024,
		StoragePath: "uploads/screenshot.png",
	})
	mockStorage.SetFile("att-2", &storage.AttachmentMeta{
		ID:          "att-2",
		FileName:    "log.txt",
		ContentType: "text/plain",
		FileSize:    256,
		StoragePath: "uploads/log.txt",
	})

	mockRunner, captured, captureMu, done := newContinuationRunner(t)

	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	svc, run, cleanup := setupContinueTest(t,
		orchestration.WithRunners(registry),
		orchestration.WithAttachmentStorage(mockStorage),
	)
	t.Cleanup(cleanup)

	ctx := context.Background()
	_, err := svc.ContinueRun(ctx, orchestration.ContinueRunRequest{
		RunID:         run.ID,
		Message:       "Follow up with attachments",
		AttachmentIDs: []string{"att-1", "att-2"},
	})
	if err != nil {
		t.Fatalf("ContinueRun: %v", err)
	}

	// Wait for the background goroutine to call Continue.
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for runner.Continue to be called")
	}

	captureMu.Lock()
	defer captureMu.Unlock()

	if len(captured.Attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(captured.Attachments))
	}

	// Verify first attachment.
	att1 := captured.Attachments[0]
	if att1.ID != "att-1" {
		t.Errorf("attachment[0].ID = %q, want %q", att1.ID, "att-1")
	}
	if att1.FileName != "screenshot.png" {
		t.Errorf("attachment[0].FileName = %q, want %q", att1.FileName, "screenshot.png")
	}
	if att1.ContentType != "image/png" {
		t.Errorf("attachment[0].ContentType = %q, want %q", att1.ContentType, "image/png")
	}
	// FilePath should come from storage.GetFilePath.
	if att1.FilePath != "/mock-storage/uploads/screenshot.png" {
		t.Errorf("attachment[0].FilePath = %q, want %q", att1.FilePath, "/mock-storage/uploads/screenshot.png")
	}

	// Verify second attachment.
	att2 := captured.Attachments[1]
	if att2.ID != "att-2" {
		t.Errorf("attachment[1].ID = %q, want %q", att2.ID, "att-2")
	}
	if att2.FileName != "log.txt" {
		t.Errorf("attachment[1].FileName = %q, want %q", att2.FileName, "log.txt")
	}

	// Verify storage was called.
	if mockStorage.GetMultipleCalls != 1 {
		t.Errorf("GetMultipleCalls = %d, want 1", mockStorage.GetMultipleCalls)
	}
	if mockStorage.GetFilePathCalls != 2 {
		t.Errorf("GetFilePathCalls = %d, want 2", mockStorage.GetFilePathCalls)
	}
}

// TestContinueRun_WithInvalidAttachmentIDs verifies that when some attachment
// IDs are not found in storage, the continuation still proceeds with whatever
// attachments could be resolved (best-effort).
func TestContinueRun_WithInvalidAttachmentIDs(t *testing.T) {
	// Only one of two IDs exists in storage.
	mockStorage := storage.NewMockService()
	mockStorage.SetFile("att-valid", &storage.AttachmentMeta{
		ID:          "att-valid",
		FileName:    "valid.png",
		ContentType: "image/png",
		FileSize:    512,
		StoragePath: "uploads/valid.png",
	})

	mockRunner, captured, captureMu, done := newContinuationRunner(t)

	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	svc, run, cleanup := setupContinueTest(t,
		orchestration.WithRunners(registry),
		orchestration.WithAttachmentStorage(mockStorage),
	)
	t.Cleanup(cleanup)

	ctx := context.Background()
	_, err := svc.ContinueRun(ctx, orchestration.ContinueRunRequest{
		RunID:         run.ID,
		Message:       "Follow up with mixed IDs",
		AttachmentIDs: []string{"att-valid", "att-nonexistent"},
	})
	if err != nil {
		t.Fatalf("ContinueRun: %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for runner.Continue to be called")
	}

	captureMu.Lock()
	defer captureMu.Unlock()

	// GetMultiple returns only found items, so we should get 1 attachment.
	if len(captured.Attachments) != 1 {
		t.Fatalf("expected 1 attachment (best-effort), got %d", len(captured.Attachments))
	}
	if captured.Attachments[0].ID != "att-valid" {
		t.Errorf("attachment[0].ID = %q, want %q", captured.Attachments[0].ID, "att-valid")
	}
}

// TestContinueRun_WithoutStorageConfigured verifies that when no storage
// service is set on the orchestrator, attachment IDs in the request are
// silently ignored and the continuation proceeds with no attachments.
func TestContinueRun_WithoutStorageConfigured(t *testing.T) {
	mockRunner, captured, captureMu, done := newContinuationRunner(t)

	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	// Deliberately do NOT pass WithAttachmentStorage.
	svc, run, cleanup := setupContinueTest(t,
		orchestration.WithRunners(registry),
	)
	t.Cleanup(cleanup)

	ctx := context.Background()
	_, err := svc.ContinueRun(ctx, orchestration.ContinueRunRequest{
		RunID:         run.ID,
		Message:       "Follow up with no storage configured",
		AttachmentIDs: []string{"att-1", "att-2"},
	})
	if err != nil {
		t.Fatalf("ContinueRun: %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for runner.Continue to be called")
	}

	captureMu.Lock()
	defer captureMu.Unlock()

	// No storage configured, so attachments should be empty.
	if len(captured.Attachments) != 0 {
		t.Fatalf("expected 0 attachments when storage not configured, got %d", len(captured.Attachments))
	}

	// Verify the message was still passed through.
	if captured.Prompt != "Follow up with no storage configured" {
		t.Errorf("Prompt = %q, want %q", captured.Prompt, "Follow up with no storage configured")
	}
}
