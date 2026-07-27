package orchestration_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/storage"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

// TestCanResumeFromFailureRun_AllowsTerminalFailures asserts the predicate
// allows resume only for failed/cancelled runs and rejects everything else.
func TestCanResumeFromFailureRun_AllowsTerminalFailures(t *testing.T) {
	cases := []struct {
		name   string
		status domain.RunStatus
		want   bool
	}{
		{"failed", domain.RunStatusFailed, true},
		{"cancelled", domain.RunStatusCancelled, true},
		{"complete", domain.RunStatusComplete, false},
		{"running", domain.RunStatusRunning, false},
		{"starting", domain.RunStatusStarting, false},
		{"pending", domain.RunStatusPending, false},
		{"needs_review", domain.RunStatusNeedsReview, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run := &domain.Run{Status: tc.status}
			got, reason := domain.CanResumeFromFailureRun(run)
			if got != tc.want {
				t.Fatalf("status=%s: want allowed=%v, got %v (reason=%q)", tc.status, tc.want, got, reason)
			}
			if !got && reason == "" {
				t.Fatalf("status=%s: rejection must include a non-empty reason", tc.status)
			}
		})
	}
}

func TestCanResumeFromFailureRun_NilRun(t *testing.T) {
	allowed, reason := domain.CanResumeFromFailureRun(nil)
	if allowed {
		t.Fatal("nil run must not be resumable")
	}
	if reason == "" {
		t.Fatal("nil rejection must include a reason")
	}
}

// newResumeTestOrchestrator wires the orchestrator with the runners +
// storage that ResumeFromFailedRun exercises. Mirrors the investigation
// attachments test setup.
func newResumeTestOrchestrator(t *testing.T) (*orchestration.Orchestrator, *database.Repositories) {
	t.Helper()

	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	registry := runner.NewRegistry()
	claudeRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	claudeRunner.SetAvailable(true, "mock claude available")
	if err := registry.Register(claudeRunner); err != nil {
		t.Fatalf("register claude runner: %v", err)
	}

	mockStorage := storage.NewMockService()

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
		newTestRolePolicyOption(t),
		orchestration.WithInvestigationSettings(repos.InvestigationSettings),
		orchestration.WithAttachmentStorage(mockStorage),
	)

	return svc, repos
}

// seedFailedRun inserts a profile, task, and a failed run that points to
// them. Returns the created records for assertions.
func seedFailedRun(
	t *testing.T,
	svc *orchestration.Orchestrator,
	repos *database.Repositories,
	originalDescription string,
	originalAttachments []domain.ContextAttachment,
) (*domain.AgentProfile, *domain.Task, *domain.Run) {
	t.Helper()
	ctx := context.Background()

	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:       "resume-source-profile",
		ProfileKey: "resume-source-" + uuid.New().String()[:8],

		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}, RoleRef: "code.default",
	})

	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:              "resume-source-task",
		Description:        originalDescription,
		ScopePath:          "src/",
		ProjectRoot:        "/tmp/resume-test-root",
		ContextAttachments: originalAttachments,
	})

	now := time.Now()
	failedRunID := uuid.New()
	failedRun := &domain.Run{
		ID:             failedRunID,
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Tag:            "resume-test-" + failedRunID.String()[:6],
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusFailed,
		Phase:          domain.RunPhaseCompleted,
		StartedAt:      &now,
		EndedAt:        &now,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
		ApprovalState:  domain.ApprovalStateNone,
		ErrorMsg:       "simulated failure for test",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repos.Runs.Create(ctx, failedRun); err != nil {
		t.Fatalf("insert failed run: %v", err)
	}
	return profile, task, failedRun
}

// TestResumeFromFailedRun_BuildsAttachments asserts that the resumed task
// carries forward the original task's attachments AND adds previous-attempt
// context (overview + timeline) plus the user's custom guidance.
func TestResumeFromFailedRun_BuildsAttachments(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	ctx := context.Background()

	originalAttachments := []domain.ContextAttachment{
		{
			Type:    "note",
			Key:     "original-spec",
			Label:   "Original Spec",
			Content: "build the widget",
			Format:  "markdown",
		},
	}

	_, _, failedRun := seedFailedRun(t, svc, repos, "Build the widget end to end.", originalAttachments)

	resumedRun, err := svc.ResumeFromFailedRun(ctx, orchestration.ResumeFromFailedRunRequest{
		RunID:         failedRun.ID,
		CustomContext: "skip the part that already wrote the README",
	})
	if err != nil {
		t.Fatalf("ResumeFromFailedRun: %v", err)
	}
	if resumedRun == nil {
		t.Fatal("expected a resumed run, got nil")
	}

	resumedTask, err := svc.GetTask(ctx, resumedRun.TaskID)
	if err != nil {
		t.Fatalf("GetTask for resumed run: %v", err)
	}

	if resumedTask.ID == failedRun.TaskID {
		t.Fatal("resumed task must be a NEW task, not the original (mutating the original would break run history)")
	}
	if resumedTask.ProjectRoot != "/tmp/resume-test-root" {
		t.Errorf("resumed task project root = %q, want inherited /tmp/resume-test-root", resumedTask.ProjectRoot)
	}

	keys := map[string]bool{}
	for _, att := range resumedTask.ContextAttachments {
		keys[att.Key] = true
	}

	if !keys["original-spec"] {
		t.Error("resumed task missing original task's ContextAttachment (key=original-spec)")
	}

	hasPrevOverview := false
	for k := range keys {
		if strings.HasPrefix(k, "run-overview-prev-") {
			hasPrevOverview = true
			break
		}
	}
	if !hasPrevOverview {
		t.Errorf("resumed task missing previous-attempt run overview attachment; keys=%v", keys)
	}

	if !keys["user-resume-context"] {
		t.Error("resumed task missing user-resume-context attachment")
	}

	if !strings.Contains(resumedTask.Description, "Prior Attempt") {
		t.Error("resumed task description must include 'Prior Attempt' framing")
	}
	if !strings.Contains(resumedTask.Description, failedRun.ID.String()) {
		t.Error("resumed task description must reference the failed run ID for diagnostics")
	}
	if !strings.Contains(resumedTask.Description, "Build the widget end to end.") {
		t.Error("resumed task description must lead with the original task prompt")
	}
}

// TestResumeFromFailedRun_LinksLineage asserts the resumed run records the
// original failed run in SourceRunIDs so downstream consumers (UI, list
// filters) can trace ancestry.
func TestResumeFromFailedRun_LinksLineage(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	ctx := context.Background()

	_, _, failedRun := seedFailedRun(t, svc, repos, "do the thing", nil)

	resumedRun, err := svc.ResumeFromFailedRun(ctx, orchestration.ResumeFromFailedRunRequest{RunID: failedRun.ID})
	if err != nil {
		t.Fatalf("ResumeFromFailedRun: %v", err)
	}

	got, err := svc.ListRuns(ctx, orchestration.RunListOptions{
		TagPrefix: failedRun.Tag,
	})
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}

	var foundResumed bool
	for _, run := range got {
		if run.ID == resumedRun.ID {
			foundResumed = true
			if !strings.HasSuffix(run.Tag, "-resume") {
				t.Errorf("resumed run tag = %q, want a -resume suffix for traceability", run.Tag)
			}
		}
	}
	if !foundResumed {
		t.Errorf("ListRuns by tag prefix %q did not return the resumed run; got %d runs", failedRun.Tag, len(got))
	}
}

// TestResumeFromFailedRun_InheritsProfile asserts the resumed run uses the
// same agent profile as the failed run (so it has the same tools and
// permissions the original attempt had).
func TestResumeFromFailedRun_InheritsProfile(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	ctx := context.Background()

	profile, _, failedRun := seedFailedRun(t, svc, repos, "do the thing", nil)

	resumedRun, err := svc.ResumeFromFailedRun(ctx, orchestration.ResumeFromFailedRunRequest{RunID: failedRun.ID})
	if err != nil {
		t.Fatalf("ResumeFromFailedRun: %v", err)
	}

	if resumedRun.AgentProfileID == nil {
		t.Fatal("resumed run must inherit the failed run's AgentProfileID, got nil")
	}
	if *resumedRun.AgentProfileID != profile.ID {
		t.Errorf("resumed run profile = %s, want inherited %s", resumedRun.AgentProfileID, profile.ID)
	}
}

// TestResumeFromFailedRun_RejectsRunning asserts the orchestrator refuses to
// resume a run that's still in flight, mirroring the predicate's contract.
func TestResumeFromFailedRun_RejectsRunning(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	ctx := context.Background()

	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:       "running-profile",
		ProfileKey: "running-" + uuid.New().String()[:8],

		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}, RoleRef: "code.default",
	})
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:       "running-task",
		Description: "in flight",
		ScopePath:   "src/",
	})

	now := time.Now()
	runID := uuid.New()
	if err := repos.Runs.Create(ctx, &domain.Run{
		ID:             runID,
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Tag:            runID.String(),
		Status:         domain.RunStatusRunning,
		Phase:          domain.RunPhaseExecuting,
		StartedAt:      &now,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("insert running run: %v", err)
	}

	_, err := svc.ResumeFromFailedRun(ctx, orchestration.ResumeFromFailedRunRequest{RunID: runID})
	if err == nil {
		t.Fatal("expected ResumeFromFailedRun to reject a still-running run, got nil error")
	}
}

// TestResumeTag_AppendsSuffix is a small regression for the tag derivation
// helper to make sure we never double-append -resume.
func TestResumeTag_AppendsSuffixOnce(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	ctx := context.Background()

	_, _, failedRun := seedFailedRun(t, svc, repos, "x", nil)
	failedRun.Tag += "-resume"
	if err := repos.Runs.Update(ctx, failedRun); err != nil {
		t.Fatalf("mark failed run tag as already resumed: %v", err)
	}

	resumed, err := svc.ResumeFromFailedRun(ctx, orchestration.ResumeFromFailedRunRequest{RunID: failedRun.ID})
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if strings.Count(resumed.Tag, "-resume") != 1 {
		t.Errorf("resumed tag = %q, want exactly one -resume suffix (no double-append)", resumed.Tag)
	}
}
