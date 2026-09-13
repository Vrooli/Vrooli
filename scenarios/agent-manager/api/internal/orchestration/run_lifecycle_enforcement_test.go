package orchestration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

func TestContinuationLifecycleClearsTerminalFieldsPreservesHistory(t *testing.T) {
	for _, status := range []domain.RunStatus{domain.RunStatusComplete, domain.RunStatusFailed, domain.RunStatusCancelled, domain.RunStatusNeedsReview, domain.RunStatusParked} {
		t.Run(string(status), func(t *testing.T) {
			ctx := t.Context()
			repos, events, cleanup := testutil.SetupTestRepos(t)
			t.Cleanup(cleanup)
			svc := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events))
			task, err := svc.CreateTask(ctx, &domain.Task{Title: "continuation lifecycle", ScopePath: "src"})
			if err != nil {
				t.Fatal(err)
			}
			started := time.Now().Add(-time.Hour).UTC()
			ended := started.Add(time.Minute)
			exit := 1
			run := &domain.Run{
				ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), RunMode: domain.RunModeInPlace,
				Status: status, Phase: domain.RunPhaseCompleted, StartedAt: &started, EndedAt: &ended,
				ErrorMsg: "old attempt failed", ExitCode: &exit, TerminalClass: domain.RunTerminalClassInterruption,
				StopReason: domain.RunStopReasonTimeout, SessionID: "retained-session", LastHandoff: "prior handoff",
				Summary: &domain.RunSummary{TokensUsed: 123, CostEstimate: 0.4},
				Result:  &domain.RunResult{FinalOutput: "prior output"}, CreatedAt: started, UpdatedAt: ended,
			}
			if err := repos.Runs.Create(ctx, run); err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC()
			returned, err := svc.applyRunStatusTransition(ctx, RunStatusTransitionInput{Run: run, NewStatus: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, LastHeartbeat: &now})
			if err != nil {
				t.Fatal(err)
			}
			persisted, err := repos.Runs.Get(ctx, run.ID)
			if err != nil {
				t.Fatal(err)
			}
			for _, got := range []*domain.Run{returned, persisted} {
				if got.Status != domain.RunStatusRunning || got.EndedAt != nil || got.ErrorMsg != "" || got.ExitCode != nil || got.TerminalClass != "" || got.StopReason != "" {
					t.Errorf("running continuation exposes prior terminal fields: status=%s ended=%v error=%q exit=%v class=%s reason=%s", got.Status, got.EndedAt, got.ErrorMsg, got.ExitCode, got.TerminalClass, got.StopReason)
				}
				if got.LastHeartbeat == nil || !got.LastHeartbeat.Equal(now) || got.StartedAt == nil || !got.StartedAt.Equal(started) || got.SessionID != "retained-session" || got.LastHandoff != "prior handoff" || got.Summary == nil || got.Summary.TokensUsed != 123 || got.Result == nil || got.Result.FinalOutput != "prior output" {
					t.Error("continuation lost prior accounting/output or current heartbeat")
				}
			}
		})
	}
}

func TestContinueOpenCodeMissingSessionRejectedBeforeEffects(t *testing.T) {
	ctx := t.Context()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	root := t.TempDir()
	registry := runner.NewRegistry()
	codec := codecs.NewOpenCodeForTestWithBinary("/bin/true")
	if err := registry.Register(core.NewRunner(codec, nil, nil)); err != nil {
		t.Fatal(err)
	}
	svc := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events), WithRunners(registry), WithRunStateRoot(root))
	task, err := svc.CreateTask(ctx, &domain.Task{Title: "missing native session", ScopePath: "src"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	run := &domain.Run{
		ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), RunMode: domain.RunModeSandboxed,
		Status: domain.RunStatusFailed, Phase: domain.RunPhaseCompleted, EndedAt: &now, ErrorMsg: "prior failure",
		SessionID: "ses_missing", ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeOpenCode}, CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	before, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.ContinueRun(ctx, ContinueRunRequest{RunID: run.ID, Message: "resume"})
	var runnerErr *domain.RunnerError
	if !errors.As(err, &runnerErr) || runnerErr.Code() != domain.ErrCodeRunnerSessionExpired {
		t.Errorf("missing session must fail at runner admission, before sandbox preparation: %v", err)
	}
	if !domain.IsPreEffectRefusal(err) {
		t.Fatalf("owner did not distinguish pre-effect refusal: %v", err)
	}
	after, getErr := repos.Runs.Get(ctx, run.ID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if !reflect.DeepEqual(before, after) {
		t.Error("rejected continuation mutated the run")
	}
	observed, err := events.Get(ctx, run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatal(err)
	}
	if len(observed) != 0 {
		t.Errorf("rejected continuation emitted %d events", len(observed))
	}
	if _, err := os.Stat(filepath.Join(root, run.ID.String())); !os.IsNotExist(err) {
		t.Errorf("rejected continuation created runtime state: %v", err)
	}
}

func TestContinuationCallbacksCannotClobberNewLifecycle(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	svc := New(repos.Profiles, repos.Tasks, repos.Runs, WithRunStateRoot(t.TempDir()))
	task, err := svc.CreateTask(t.Context(), &domain.Task{Title: "stream fencing", ScopePath: "src"})
	if err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeOpenCode}}
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	stream, closeStream, err := svc.prepareRunTranscript(t.Context(), run, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer closeStream()
	run.Status = domain.RunStatusFailed
	if err := repos.Runs.Update(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	run.Status = domain.RunStatusRunning
	run.RunnerPID = 22222
	run.SessionID = "new-session"
	if err := repos.Runs.Update(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	before, _ := repos.Runs.Get(t.Context(), run.ID)
	if err := stream.OnProcessStart(11111, 11111); err == nil {
		t.Error("stale process callback accepted")
	}
	if err := stream.OnSessionID("old-session"); err == nil {
		t.Error("stale session callback accepted")
	}
	if err := stream.OnAdvance(100, 10); err == nil {
		t.Error("stale cursor callback accepted")
	}
	after, _ := repos.Runs.Get(t.Context(), run.ID)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("old callback changed current lifecycle state")
	}
}

type recordingCredentialUseReleaser struct {
	runID  uuid.UUID
	reason string
}

func (r *recordingCredentialUseReleaser) RevokeRunCredentialUse(_ context.Context, runID uuid.UUID, reason string) error {
	r.runID, r.reason = runID, reason
	return nil
}

// TestApplyRunStatusTransition_RejectsIllegalTransition verifies that the single
// status-mutation helper enforces the run state machine: an illegal transition
// (here pending → complete) is rejected and the persisted run is left untouched.
// This is the guard that stops future statuses from being set ad-hoc.
func TestApplyRunStatusTransition_RejectsIllegalTransition(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	svc := New(repos.Profiles, repos.Tasks, repos.Runs)

	task, terr := svc.CreateTask(ctx, &domain.Task{Title: "enforce task", ScopePath: "src/"})
	if terr != nil {
		t.Fatalf("create task: %v", terr)
	}

	now := time.Now()
	runID := uuid.New()
	run := &domain.Run{
		ID:            runID,
		TaskID:        task.ID,
		Tag:           runID.String(),
		RunMode:       domain.RunModeInPlace,
		Status:        domain.RunStatusPending,
		Phase:         domain.RunPhaseQueued,
		ApprovalState: domain.ApprovalStateNone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	_, err := svc.applyRunStatusTransition(ctx, RunStatusTransitionInput{
		Run:       run,
		NewStatus: domain.RunStatusComplete, // pending → complete is not allowed
	})
	if err == nil {
		t.Fatal("expected illegal transition to be rejected, got nil error")
	}

	persisted, gerr := repos.Runs.Get(ctx, runID)
	if gerr != nil {
		t.Fatalf("get run: %v", gerr)
	}
	if persisted.Status != domain.RunStatusPending {
		t.Fatalf("run status = %s, want pending (illegal transition must not persist)", persisted.Status)
	}
}

// TestApplyRunStatusTransition_AllowsSameStatusNoop verifies that a same-status
// write (e.g. a heartbeat/progress refresh) is treated as a no-op update rather
// than a transition, and is therefore always permitted even though running →
// running is not an edge in the transition table.
func TestApplyRunStatusTransition_AllowsSameStatusNoop(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	svc := New(repos.Profiles, repos.Tasks, repos.Runs)

	task, terr := svc.CreateTask(ctx, &domain.Task{Title: "noop task", ScopePath: "src/"})
	if terr != nil {
		t.Fatalf("create task: %v", terr)
	}

	now := time.Now()
	runID := uuid.New()
	run := &domain.Run{
		ID:            runID,
		TaskID:        task.ID,
		Tag:           runID.String(),
		RunMode:       domain.RunModeInPlace,
		Status:        domain.RunStatusRunning,
		Phase:         domain.RunPhaseExecuting,
		StartedAt:     &now,
		ApprovalState: domain.ApprovalStateNone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	progress := 42
	if _, err := svc.applyRunStatusTransition(ctx, RunStatusTransitionInput{
		Run:             run,
		NewStatus:       domain.RunStatusRunning,
		ProgressPercent: &progress,
	}); err != nil {
		t.Fatalf("same-status update should be permitted, got: %v", err)
	}

	persisted, gerr := repos.Runs.Get(ctx, runID)
	if gerr != nil {
		t.Fatalf("get run: %v", gerr)
	}
	if persisted.Status != domain.RunStatusRunning {
		t.Fatalf("run status = %s, want running", persisted.Status)
	}
	if persisted.ProgressPercent != progress {
		t.Fatalf("progress = %d, want %d", persisted.ProgressPercent, progress)
	}
}

func TestApplyRunStatusTransitionRevokesRunBoundCredentialUseOnTerminalState(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	releaser := &recordingCredentialUseReleaser{}
	svc := New(repos.Profiles, repos.Tasks, repos.Runs, WithCredentialUseReleaser(releaser))
	now := time.Now()
	task := &domain.Task{ID: uuid.New(), Title: "credential cleanup", ScopePath: "src", Status: domain.TaskStatusQueued, CreatedAt: now, UpdatedAt: now}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: "credential-cleanup", RunMode: domain.RunModeInPlace, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, CreatedAt: now, UpdatedAt: now}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.applyRunStatusTransition(ctx, RunStatusTransitionInput{Run: run, NewStatus: domain.RunStatusCancelled, Phase: domain.RunPhaseCompleted}); err != nil {
		t.Fatal(err)
	}
	if releaser.runID != run.ID || releaser.reason != string(domain.RunStatusCancelled) {
		t.Fatalf("credential cleanup = run %s reason %q, want run %s reason %q", releaser.runID, releaser.reason, run.ID, domain.RunStatusCancelled)
	}
}
