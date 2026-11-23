package orchestration_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

// This independent regression case checks that a checkpoint for an earlier
// phase cannot authorize resuming a run that has already entered execution.
func TestResumeRunRejectsPhaseCheckpointMismatchWithoutAdmissionOrMutation(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	svc := resumeTestOrchestrator(t, repos, events)
	now := time.Now()
	task := &domain.Task{
		ID: uuid.New(), Title: "resume checkpoint consistency", ScopePath: "src/",
		ProjectRoot: t.TempDir(), Status: domain.TaskStatusQueued, CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{
		ID: uuid.New(), TaskID: task.ID, Tag: "resume-checkpoint-" + uuid.NewString()[:8],
		RunMode: domain.RunModeInPlace, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting,
		ApprovalState:  domain.ApprovalStateNone,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
		CreatedAt:      now, UpdatedAt: now,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := repos.Checkpoints.Save(ctx, domain.NewCheckpoint(run.ID, domain.RunPhaseQueued)); err != nil {
		t.Fatal(err)
	}

	before, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	queueBefore := svc.SpawnStats().QueueDepth
	accepted, resumeErr := svc.ResumeRun(ctx, run.ID)
	if accepted != nil || resumeErr == nil || !strings.Contains(strings.ToLower(resumeErr.Error()), "checkpoint") {
		t.Fatalf("ResumeRun accepted a phase/checkpoint mismatch: run=%#v err=%v", accepted, resumeErr)
	}
	after, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != before.Status || after.Phase != before.Phase || after.LifecycleVersion != before.LifecycleVersion {
		t.Fatalf("rejected resume changed durable run: before=%+v after=%+v", before, after)
	}
	if got := svc.SpawnStats().QueueDepth; got != queueBefore {
		t.Fatalf("resume queue depth = %d, want unchanged %d", got, queueBefore)
	}
}
