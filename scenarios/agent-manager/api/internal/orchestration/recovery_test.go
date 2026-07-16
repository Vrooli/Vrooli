package orchestration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/runstate"
	"agent-manager/internal/testutil"
	"agent-manager/internal/testutil/mocks"

	"github.com/google/uuid"
)

func TestRecoverRun_DeadProcessWithTerminalSuccess(t *testing.T) {
	reconciler, repos, eventStore := newRecoveryTestReconciler(t, domain.RunnerTypeCodex)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
	transcriptPath := writeRecoveryTranscript(t, "session:thread-123\nmessage:Recovered final answer\ndone:Recovered final answer\n")
	run.TranscriptPath = transcriptPath
	if err := repos.Runs.Update(context.Background(), run); err != nil {
		t.Fatalf("update run: %v", err)
	}

	result, err := reconciler.RecoverRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("RecoverRun: %v", err)
	}
	if !result.Recovered {
		t.Fatalf("expected recovered result, got %+v", result)
	}

	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusComplete {
		t.Fatalf("status = %s, want %s", got.Status, domain.RunStatusComplete)
	}
	if got.SessionID != "thread-123" {
		t.Fatalf("session_id = %q, want %q", got.SessionID, "thread-123")
	}
	if got.Summary == nil || got.Summary.Description != "Recovered final answer" {
		t.Fatalf("summary = %+v, want recovered final answer", got.Summary)
	}
	if got.Result == nil || got.Result.Selection.Status != domain.FinalOutputSelectionSelected || got.Result.FinalOutput != "Recovered final answer" {
		t.Fatalf("result = %+v, want selected recovered final answer", got.Result)
	}
	if got.ExitCode == nil || *got.ExitCode != 0 {
		t.Fatalf("exit_code = %+v, want 0", got.ExitCode)
	}
	if got.TranscriptCursor <= 0 {
		t.Fatalf("transcript cursor = %d, want > 0", got.TranscriptCursor)
	}

	_, _ = eventStore.Get(context.Background(), run.ID, event.GetOptions{})
}

func TestRecoverRun_DeadProcessWithTerminalFailure(t *testing.T) {
	reconciler, repos, _ := newRecoveryTestReconciler(t, domain.RunnerTypeOpenCode)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeOpenCode)
	run.TranscriptPath = writeRecoveryTranscript(t, "session:session-9\nfail:runner crashed after restart\n")
	if err := repos.Runs.Update(context.Background(), run); err != nil {
		t.Fatalf("update run: %v", err)
	}

	if _, err := reconciler.RecoverRun(context.Background(), run.ID); err != nil {
		t.Fatalf("RecoverRun: %v", err)
	}

	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusFailed {
		t.Fatalf("status = %s, want %s", got.Status, domain.RunStatusFailed)
	}
	if got.ErrorMsg != "runner crashed after restart" {
		t.Fatalf("error = %q, want %q", got.ErrorMsg, "runner crashed after restart")
	}
	if got.ExitCode == nil || *got.ExitCode != 1 {
		t.Fatalf("exit_code = %+v, want 1", got.ExitCode)
	}
}

func TestRecoverRun_DeadProcessWithoutTerminalEventMarksFailed(t *testing.T) {
	reconciler, repos, _ := newRecoveryTestReconciler(t, domain.RunnerTypeClaudeCode)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeClaudeCode)
	run.TranscriptPath = writeRecoveryTranscript(t, "message:partial output only\n")
	if err := repos.Runs.Update(context.Background(), run); err != nil {
		t.Fatalf("update run: %v", err)
	}

	if _, err := reconciler.RecoverRun(context.Background(), run.ID); err != nil {
		t.Fatalf("RecoverRun: %v", err)
	}

	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusFailed {
		t.Fatalf("status = %s, want %s", got.Status, domain.RunStatusFailed)
	}
	if got.ErrorMsg != "runner exited before terminal event" {
		t.Fatalf("error = %q, want %q", got.ErrorMsg, "runner exited before terminal event")
	}
}

func TestHandleStaleRun_DrainsTranscriptBeforeFailing(t *testing.T) {
	reconciler, repos, _ := newRecoveryTestReconciler(t, domain.RunnerTypeCodex)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
	stale := time.Now().Add(-15 * time.Minute)
	run.LastHeartbeat = &stale
	run.TranscriptPath = writeRecoveryTranscript(t, "message:stale recovery summary\ndone:stale recovery summary\n")
	if err := repos.Runs.Update(context.Background(), run); err != nil {
		t.Fatalf("update run: %v", err)
	}

	stats := &ReconcileStats{}
	reconciler.handleStaleRun(context.Background(), run, stats)

	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusComplete {
		t.Fatalf("status = %s, want %s", got.Status, domain.RunStatusComplete)
	}
	if stats.RunsRecovered != 1 {
		t.Fatalf("runs recovered = %d, want 1", stats.RunsRecovered)
	}
}

func TestRecoverRun_LiveProcessTailsTranscriptUntilTerminal(t *testing.T) {
	reconciler, repos, _ := newRecoveryTestReconciler(t, domain.RunnerTypeCodex)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
	transcriptPath := filepath.Join(t.TempDir(), "transcript.ndjson")
	if err := os.WriteFile(transcriptPath, []byte("message:initial live output\n"), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	run.TranscriptPath = transcriptPath
	run.Tag = uuid.NewString()
	if err := repos.Runs.Update(context.Background(), run); err != nil {
		t.Fatalf("update run: %v", err)
	}

	cmd := startTaggedRunnerProcess(t, "codex", run.Tag)
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	result, err := reconciler.recoverRun(context.Background(), run, true)
	if err != nil {
		t.Fatalf("recoverRun: %v", err)
	}
	if !result.Recovered {
		t.Fatalf("expected recovered result, got %+v", result)
	}

	appendRecoveryTranscript(t, transcriptPath, "message:tail completion\ndone:tail completion\n")
	_ = cmd.Process.Kill()
	_ = cmd.Wait()

	waitForRecoveryStatus(t, repos, run.ID, domain.RunStatusComplete)
	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Summary == nil || got.Summary.Description != "tail completion" {
		t.Fatalf("summary = %+v, want tail completion", got.Summary)
	}
}

func TestCleanupRunStateDirs_RemovesOldTerminalRunDirectories(t *testing.T) {
	reconciler, repos, _ := newRecoveryTestReconciler(t, domain.RunnerTypeClaudeCode)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeClaudeCode)
	oldTime := time.Now().Add(-8 * 24 * time.Hour)
	run.Status = domain.RunStatusComplete
	run.UpdatedAt = oldTime
	run.EndedAt = &oldTime
	if err := repos.Runs.Update(context.Background(), run); err != nil {
		t.Fatalf("update run: %v", err)
	}

	dir := runstate.RunDir("", run.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	reconciler.cleanupRunStateDirs(context.Background())

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed, stat err=%v", dir, err)
	}
}

func newRecoveryTestReconciler(t *testing.T, rt domain.RunnerType) (*Reconciler, *database.Repositories, event.Store) {
	t.Helper()

	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	registry := runner.NewRegistry()
	if err := registry.Register(mocks.NewTranscriptReplayRunner(rt)); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	reconciler := NewReconciler(
		repos.Runs,
		registry,
		WithReconcilerEvents(eventStore),
		WithReconcilerConfig(ReconcilerConfig{
			Interval:          time.Hour,
			StaleThreshold:    time.Minute,
			MaxRecoveryAge:    10 * time.Minute,
			OrphanGracePeriod: time.Minute,
			MaxStaleRuns:      10,
			KillOrphans:       true,
			AutoRecover:       true,
		}),
	)

	return reconciler, repos, eventStore
}

func createRecoveryTestRun(t *testing.T, repos *database.Repositories, rt domain.RunnerType) *domain.Run {
	t.Helper()

	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "Recovery test task",
		Description: "Validate transcript recovery",
		ScopePath:   ".",
		ProjectRoot: ".",
		Status:      domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	now := time.Now().UTC()
	cfg := domain.DefaultRunConfig()
	cfg.RunnerType = rt
	run := &domain.Run{
		ID:             uuid.New(),
		TaskID:         task.ID,
		Tag:            uuid.NewString(),
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusRunning,
		StartedAt:      &now,
		LastHeartbeat:  &now,
		Phase:          domain.RunPhaseExecuting,
		ApprovalState:  domain.ApprovalStateNone,
		ResolvedConfig: cfg,
	}
	if err := repos.Runs.Create(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	return run
}

func writeRecoveryTranscript(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "transcript.ndjson")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	return path
}

func appendRecoveryTranscript(t *testing.T, path, contents string) {
	t.Helper()

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open transcript append: %v", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := f.WriteString(contents); err != nil {
		t.Fatalf("append transcript: %v", err)
	}
}

func startTaggedRunnerProcess(t *testing.T, runnerName, tag string) *exec.Cmd {
	t.Helper()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, runnerName)
	script := "#!/bin/sh\nwhile true; do sleep 1; done\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write runner script: %v", err)
	}

	cmd := exec.Command(scriptPath, "--tag", tag)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start runner script: %v", err)
	}

	time.Sleep(150 * time.Millisecond)
	return cmd
}

func waitForRecoveryStatus(t *testing.T, repos *database.Repositories, runID uuid.UUID, want domain.RunStatus) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		run, err := repos.Runs.Get(context.Background(), runID)
		if err != nil {
			t.Fatalf("get run: %v", err)
		}
		if run.Status == want {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	run, err := repos.Runs.Get(context.Background(), runID)
	if err != nil {
		t.Fatalf("get run after timeout: %v", err)
	}
	t.Fatalf("timed out waiting for status %s, got %s", want, run.Status)
}
