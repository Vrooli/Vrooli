package orchestration_test

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/storage"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
)

// TestCreateInvestigationRun_WithAttachmentIDs verifies that user-uploaded
// image attachment IDs submitted to CreateInvestigationRun are persisted onto
// the investigation task's ContextAttachments as image-typed entries. That
// plumbing is what lets the existing CreateRun → runner pipeline resolve the
// uploads to file paths and hand them to the agent.
func TestCreateInvestigationRun_WithAttachmentIDs(t *testing.T) {
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	ctx := context.Background()

	registry := runner.NewRegistry()
	// Investigation profile uses the Codex runner type.
	codexRunner := runner.NewMockRunner(domain.RunnerTypeCodex)
	codexRunner.SetAvailable(true, "mock codex available")
	if err := registry.Register(codexRunner); err != nil {
		t.Fatalf("register codex runner: %v", err)
	}

	mockStorage := storage.NewMockService()
	// The attachments referenced here don't have to actually exist in storage
	// for this test — we're asserting the task-level plumbing. The run
	// execution path resolves storage on its own.

	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          5 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: false,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
		orchestration.WithInvestigationSettings(repos.InvestigationSettings),
		orchestration.WithAttachmentStorage(mockStorage),
	)

	// Seed a source task + run so CreateInvestigationRun has something to
	// investigate. We insert directly to bypass async execution.
	sourceTask := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:       "source-task",
		Description: "task that the investigation targets",
		ScopePath:   "src/",
	})

	sourceProfile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:            "source-profile",
		ProfileKey:      "source-" + uuid.New().String()[:8],
		RunnerType:      domain.RunnerTypeClaudeCode,
		RequiresSandbox: false,
	})

	// Pre-create the investigation profile with a concrete Model so the CreateRun
	// config validation doesn't hit the ModelPreset → model-registry path (the
	// built-in default profile uses ModelPreset=Smart, which requires a
	// configured registry we don't wire up for this unit test).
	mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:            "investigation-profile",
		ProfileKey:      domain.InvestigationTag,
		RunnerType:      domain.RunnerTypeCodex,
		Model:           "mock-model",
		RequiresSandbox: false,
	})

	now := time.Now()
	sourceRunID := uuid.New()
	sourceRun := &domain.Run{
		ID:             sourceRunID,
		TaskID:         sourceTask.ID,
		AgentProfileID: &sourceProfile.ID,
		Tag:            sourceRunID.String(),
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusComplete,
		Phase:          domain.RunPhaseCompleted,
		StartedAt:      &now,
		EndedAt:        &now,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeClaudeCode,
		},
		ApprovalState: domain.ApprovalStateNone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repos.Runs.Create(ctx, sourceRun); err != nil {
		t.Fatalf("insert source run: %v", err)
	}

	attachmentIDs := []string{"att-img-1", "att-img-2"}

	investigationRun, err := svc.CreateInvestigationRun(ctx, orchestration.CreateInvestigationRequest{
		RunIDs:        []uuid.UUID{sourceRunID},
		CustomContext: "please investigate",
		Depth:         domain.InvestigationDepthQuick,
		AttachmentIDs: attachmentIDs,
	})
	if err != nil {
		t.Fatalf("CreateInvestigationRun: %v", err)
	}

	investigationTask, err := svc.GetTask(ctx, investigationRun.TaskID)
	if err != nil {
		t.Fatalf("GetTask for investigation run: %v", err)
	}

	// Collect image-typed ContextAttachments by their AttachmentID so order
	// doesn't matter — other text attachments (metadata, run overview, etc.)
	// may appear before or after ours.
	got := map[string]bool{}
	for _, att := range investigationTask.ContextAttachments {
		if att.Type == "image" && att.AttachmentID != "" {
			got[att.AttachmentID] = true
		}
	}

	for _, id := range attachmentIDs {
		if !got[id] {
			t.Errorf("investigation task missing image ContextAttachment with AttachmentID=%q; got %d image attachments: %v",
				id, len(got), got)
		}
	}
}

// TestCreateInvestigationRun_SkipsBlankAttachmentIDs verifies that empty/
// whitespace entries in AttachmentIDs are silently filtered out rather than
// producing image-typed attachments with no AttachmentID (which the runner
// bridge would later discard anyway, but cleaner to drop up front).
func TestCreateInvestigationRun_SkipsBlankAttachmentIDs(t *testing.T) {
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	ctx := context.Background()

	registry := runner.NewRegistry()
	codexRunner := runner.NewMockRunner(domain.RunnerTypeCodex)
	codexRunner.SetAvailable(true, "mock codex available")
	if err := registry.Register(codexRunner); err != nil {
		t.Fatalf("register codex runner: %v", err)
	}

	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          5 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: false,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
		orchestration.WithInvestigationSettings(repos.InvestigationSettings),
	)

	sourceTask := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:     "source-task-blank-atts",
		ScopePath: "src/",
	})
	sourceProfile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:            "source-profile-blank",
		ProfileKey:      "source-blank-" + uuid.New().String()[:8],
		RunnerType:      domain.RunnerTypeClaudeCode,
		RequiresSandbox: false,
	})
	mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:            "investigation-profile-blank",
		ProfileKey:      domain.InvestigationTag,
		RunnerType:      domain.RunnerTypeCodex,
		Model:           "mock-model",
		RequiresSandbox: false,
	})
	now := time.Now()
	sourceRunID := uuid.New()
	sourceRun := &domain.Run{
		ID:             sourceRunID,
		TaskID:         sourceTask.ID,
		AgentProfileID: &sourceProfile.ID,
		Tag:            sourceRunID.String(),
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusComplete,
		Phase:          domain.RunPhaseCompleted,
		StartedAt:      &now,
		EndedAt:        &now,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
		ApprovalState:  domain.ApprovalStateNone,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repos.Runs.Create(ctx, sourceRun); err != nil {
		t.Fatalf("insert source run: %v", err)
	}

	investigationRun, err := svc.CreateInvestigationRun(ctx, orchestration.CreateInvestigationRequest{
		RunIDs:        []uuid.UUID{sourceRunID},
		Depth:         domain.InvestigationDepthQuick,
		AttachmentIDs: []string{"", "  ", "att-real"},
	})
	if err != nil {
		t.Fatalf("CreateInvestigationRun: %v", err)
	}

	investigationTask, err := svc.GetTask(ctx, investigationRun.TaskID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	imageIDs := []string{}
	for _, att := range investigationTask.ContextAttachments {
		if att.Type == "image" {
			imageIDs = append(imageIDs, att.AttachmentID)
		}
	}
	if len(imageIDs) != 1 || imageIDs[0] != "att-real" {
		t.Errorf("expected exactly one image attachment with AttachmentID=att-real; got %v", imageIDs)
	}
}

// TestCreateInvestigationApplyRun_WithAttachmentIDs verifies that user-uploaded
// image attachment IDs submitted to CreateInvestigationApplyRun are persisted
// onto the apply run's task as image-typed ContextAttachments, on top of any
// attachments carried over from the source investigation task.
func TestCreateInvestigationApplyRun_WithAttachmentIDs(t *testing.T) {
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	ctx := context.Background()

	registry := runner.NewRegistry()
	codexRunner := runner.NewMockRunner(domain.RunnerTypeCodex)
	codexRunner.SetAvailable(true, "mock codex available")
	if err := registry.Register(codexRunner); err != nil {
		t.Fatalf("register codex runner: %v", err)
	}

	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          5 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: false,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
		orchestration.WithInvestigationSettings(repos.InvestigationSettings),
	)

	// Both built-in profiles need concrete Model values so the CreateRun
	// validation doesn't require a model registry.
	mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:            "investigation-profile-apply-test",
		ProfileKey:      domain.InvestigationTag,
		RunnerType:      domain.RunnerTypeCodex,
		Model:           "mock-model",
		RequiresSandbox: false,
	})
	applyProfile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:            "apply-investigation-profile-test",
		ProfileKey:      domain.InvestigationApplyTag,
		RunnerType:      domain.RunnerTypeCodex,
		Model:           "mock-model",
		RequiresSandbox: false,
	})

	// Seed an investigation task (which the apply flow will copy attachments
	// from) and a completed investigation run referencing it.
	investigationTask := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:       "investigation-task",
		Description: "investigation that apply will build on",
		ScopePath:   "src/",
	})

	now := time.Now()
	investigationRunID := uuid.New()
	investigationRun := &domain.Run{
		ID:             investigationRunID,
		TaskID:         investigationTask.ID,
		AgentProfileID: &applyProfile.ID, // any complete run w/ matching tag works
		Tag:            domain.InvestigationTag,
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusComplete,
		Phase:          domain.RunPhaseCompleted,
		StartedAt:      &now,
		EndedAt:        &now,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex},
		ApprovalState:  domain.ApprovalStateNone,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repos.Runs.Create(ctx, investigationRun); err != nil {
		t.Fatalf("insert investigation run: %v", err)
	}

	attachmentIDs := []string{"apply-img-1", "apply-img-2", "", "  "}

	applyRun, err := svc.CreateInvestigationApplyRun(ctx, orchestration.CreateInvestigationApplyRequest{
		InvestigationRunID: investigationRunID,
		CustomContext:      "apply the fixes",
		AttachmentIDs:      attachmentIDs,
	})
	if err != nil {
		t.Fatalf("CreateInvestigationApplyRun: %v", err)
	}

	applyTask, err := svc.GetTask(ctx, applyRun.TaskID)
	if err != nil {
		t.Fatalf("GetTask for apply run: %v", err)
	}

	got := map[string]bool{}
	for _, att := range applyTask.ContextAttachments {
		if att.Type == "image" && att.AttachmentID != "" {
			got[att.AttachmentID] = true
		}
	}

	for _, id := range []string{"apply-img-1", "apply-img-2"} {
		if !got[id] {
			t.Errorf("apply task missing image ContextAttachment with AttachmentID=%q; got image IDs %v",
				id, got)
		}
	}

	// Blanks must be filtered out.
	if got[""] || got["  "] {
		t.Errorf("blank/whitespace AttachmentIDs leaked into apply task attachments: %v", got)
	}
}
