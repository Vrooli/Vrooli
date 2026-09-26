package orchestration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

type pendingRunRecoveryStub struct{ ids []uuid.UUID }

func TestReconcilerRecognizesRecordedLegacyContinuationProcess(t *testing.T) {
	rec, repos, _ := newRecoveryTestReconciler(t, domain.RunnerTypeOpenCode)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeOpenCode)
	run.Tag = "canonical-run-tag"
	native := core.NewRunner(codecs.NewOpenCodeForTest(), nil, nil)
	registry := runner.NewRegistry()
	if err := registry.Register(native); err != nil {
		t.Fatal(err)
	}
	rec.runners = registry
	cmd := exec.CommandContext(t.Context(), "sleep", "10")
	cmd.Env = []string{"OPENCODE_AGENT_TAG=" + native.ContinuationTag(runner.ContinueRequest{RunID: run.ID})}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	run.RunnerPID = cmd.Process.Pid
	// /proc can briefly expose the pre-exec image on heavily loaded hosts.
	deadline := time.Now().Add(time.Second)
	for extractTagFromEnv(run.RunnerPID) == "" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !rec.isProcessAlive(t.Context(), run) {
		t.Fatal("recorded continuation declared dead because its tag differs from canonical tag")
	}
	run.ID = uuid.New()
	if rec.isProcessAlive(t.Context(), run) {
		t.Fatal("unrelated PID accepted without matching continuation identity")
	}
}

func TestRestartRecoveryDoesNotTimeoutReattachedLegacyExecutor(t *testing.T) {
	rec, repos, _ := newRecoveryTestReconciler(t, domain.RunnerTypeOpenCode)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeOpenCode)

	native := core.NewRunner(codecs.NewOpenCodeForTest(), nil, nil)
	registry := runner.NewRegistry()
	if err := registry.Register(native); err != nil {
		t.Fatal(err)
	}
	rec.runners = registry
	// The fixture starts stale; normal repository updates cannot erase a
	// newer heartbeat just to manufacture a timeout.
	run.ID = uuid.New()
	old := time.Now().Add(-time.Hour)
	run.LastHeartbeat = &old
	run.LifecycleVersion = 0
	run.EndedAt = &old
	run.ExitCode = intPtr(0)
	run.ErrorMsg = "old attempt metadata"
	run.TerminalClass = domain.RunTerminalClassInterruption
	run.StopReason = domain.RunStopReasonTimeout
	run.ProgressPercent = 42
	run.Result = &domain.RunResult{FinalOutput: "prior accepted output"}
	run.Summary = &domain.RunSummary{TokensUsed: 123, CostEstimate: 0.4}
	attachedAt := time.Now().UTC()
	rec.clock = func() time.Time { return attachedAt }
	run.TranscriptPath = writeRecoveryTranscript(t, "")
	cmd := exec.CommandContext(t.Context(), "sleep", "30")
	cmd.Env = []string{"OPENCODE_AGENT_TAG=" + native.ContinuationTag(runner.ContinueRequest{RunID: run.ID})}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	run.RunnerPID = cmd.Process.Pid
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	if err := excludeLiveExecutor(t.Context(), run); err == nil {
		t.Fatal("live recorded executor was not conservatively excluded")
	}
	// A liveness-only recovery read has not attached an owner and must not
	// invent an attachment heartbeat or normalize the row as if it had.
	if _, err := rec.recoverRun(t.Context(), run, false); err != nil {
		t.Fatal(err)
	}
	unattached, err := repos.Runs.Get(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unattached.EndedAt == nil || !unattached.LastHeartbeat.Equal(old) {
		t.Fatal("non-attaching recovery manufactured attachment state")
	}
	rec.config.MaxRecoveryAge = time.Minute
	rec.handleStaleRun(t.Context(), run, &ReconcileStats{})
	rec.CancelInteractiveTail(run.ID)
	got, err := repos.Runs.Get(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RunStatusRunning || got.EndedAt != nil || got.ErrorMsg != "" {
		t.Fatalf("healthy reattached process timed out: %+v", got)
	}
	if got.OwnerIdentity == "" || got.OwnerEpoch != 1 {
		t.Fatalf("recovery did not durably claim the live executor: identity=%q epoch=%d", got.OwnerIdentity, got.OwnerEpoch)
	}
	if got.ExitCode != nil || got.TerminalClass != "" || got.StopReason != "" {
		t.Fatal("reattached current turn retains stale terminal fields")
	}
	if got.LastHeartbeat == nil || !got.LastHeartbeat.Equal(attachedAt) {
		t.Fatalf("attachment did not establish an owned heartbeat: %v", got.LastHeartbeat)
	}
	if got.ProgressPercent != 42 || got.Result == nil || got.Result.FinalOutput != "prior accepted output" || got.Summary == nil || got.Summary.TokensUsed != 123 || got.Summary.CostEstimate != 0.4 {
		t.Fatal("reattachment fabricated progress or changed prior evidence")
	}
	if got.StartedAt == nil || !got.StartedAt.Equal(*run.StartedAt) || got.SessionID != run.SessionID || got.RunnerPID != cmd.Process.Pid {
		t.Fatal("reattachment replaced original run/process identity")
	}
	if !rec.isProcessAlive(t.Context(), got) {
		t.Fatal("healthy reattached executor was killed")
	}
}

func TestRecoveryArtifactFailurePreservesLiveLegacyExecutor(t *testing.T) {
	for _, artifact := range []string{"unreadable", "malformed", "missing", "old_attempt_deadline"} {
		t.Run(artifact, func(t *testing.T) {
			rec, repos, _ := newRecoveryTestReconciler(t, domain.RunnerTypeOpenCode)
			run := createRecoveryTestRun(t, repos, domain.RunnerTypeOpenCode)
			run.ID = uuid.New()
			old := time.Now().Add(-2 * time.Hour)
			run.LastHeartbeat, run.StartedAt = &old, &old
			run.ResolvedConfig.Timeout = time.Minute
			switch artifact {
			case "unreadable":
				run.TranscriptPath = t.TempDir() // EISDIR even when tests run as root.
			case "malformed":
				run.TranscriptPath = writeRecoveryTranscript(t, "{\"type\":invalid-json}\n")
			case "missing":
				run.TranscriptPath = filepath.Join(t.TempDir(), "absent.ndjson")
			case "old_attempt_deadline":
				run.TranscriptPath = writeRecoveryTranscript(t, "{\"type\":\"step_finish\",\"part\":{\"type\":\"step-finish\",\"reason\":\"error\",\"output\":\"old attempt deadline exceeded\"}}\n")
			}
			native := core.NewRunner(codecs.NewOpenCodeForTest(), nil, nil)
			registry := runner.NewRegistry()
			if err := registry.Register(native); err != nil {
				t.Fatal(err)
			}
			rec.runners = registry
			cmd := exec.CommandContext(t.Context(), "sleep", "30")
			// The negative baseline must not invoke host resource cleanup CLIs.
			// sleep was resolved above; liveness uses this exact recorded PID.
			t.Setenv("PATH", t.TempDir())
			legacyTag := native.ContinuationTag(runner.ContinueRequest{RunID: run.ID})
			cmd.Env = []string{"OPENCODE_AGENT_TAG=" + legacyTag}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
			run.RunnerPID = cmd.Process.Pid
			if err := repos.Runs.Create(t.Context(), run); err != nil {
				t.Fatal(err)
			}
			if !rec.isProcessAlive(t.Context(), run) {
				t.Fatal("fixture's exact legacy identity is not recognized")
			}
			rec.config.MaxRecoveryAge = time.Minute
			stats := &ReconcileStats{}
			rec.handleStaleRun(t.Context(), run, stats)
			rec.CancelInteractiveTail(run.ID)
			got, err := repos.Runs.Get(t.Context(), run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != domain.RunStatusRunning || got.EndedAt != nil || got.ErrorMsg != "" || got.RunnerPID != cmd.Process.Pid {
				t.Errorf("artifact failure changed the live lifecycle: status=%s ended=%v error=%q pid=%d", got.Status, got.EndedAt, got.ErrorMsg, got.RunnerPID)
			}
			if extractTagFromEnv(cmd.Process.Pid) != legacyTag || !rec.isProcessAlive(t.Context(), got) {
				t.Error("exact legacy executor was killed or replaced")
			}
			if err := excludeLiveExecutor(t.Context(), got); err == nil {
				t.Error("artifact failure admitted replacement of the live executor")
			}
			owner := New(repos.Profiles, repos.Tasks, repos.Runs, WithRunners(registry), WithRunStateRoot(t.TempDir()))
			if _, err := owner.RecoverMissingSessionRun(t.Context(), ResumeFromFailedRunRequest{RunID: got.ID}); err == nil || !domain.IsPreEffectRefusal(err) {
				t.Errorf("fresh recovery was not refused before effects: %v", err)
			}
			if replacement, err := repos.Runs.GetByIdempotencyKey(t.Context(), "resume-from-failed:"+got.ID.String()); err != nil || replacement != nil {
				t.Errorf("artifact failure created a replacement: %v %v", replacement, err)
			}
			if got.TranscriptCursor != 0 {
				t.Error("unverified artifact/terminal was consumed; later owner diagnosis cannot replay it")
			}
			if stats.RunsRecovered != 0 {
				t.Error("artifact failure claimed verified recovery")
			}
			diagnosis := strings.Join(stats.Errors, " ")
			if !strings.Contains(diagnosis, "recovery outcome unknown") || !strings.Contains(diagnosis, "agent-manager owner diagnosis") {
				t.Errorf("missing named owner diagnosis: %q", diagnosis)
			}
		})
	}
}

func TestRecoveredCodexTranscriptUsesSavedBillingReceipt(t *testing.T) {
	reconciler, repos, eventStore := newRecoveryTestReconciler(t, domain.RunnerTypeCodex)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
	run.Billing = domain.BillingSnapshot{Basis: domain.ChargeBasisSubscription, Source: "original-owner-snapshot"}
	run.ResolvedConfig.Model = "original-model"
	// The row's immutable billing stamp is authoritative during recovery,
	// even if a different config object would suggest another basis.
	run.ResolvedConfig.Billing = domain.BillingSnapshot{Basis: domain.ChargeBasisMetered}
	run.ID = uuid.New()
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	saved, err := repos.Runs.Get(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "transcript.ndjson")
	if err := os.WriteFile(path, []byte("{\"type\":\"turn.completed\",\"usage\":{\"input_tokens\":19,\"output_tokens\":4}}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	terminal, err := reconciler.drainTranscript(t.Context(), saved, path, nil, codecs.NewCodexForTest())
	if err != nil || terminal == nil || !terminal.Success {
		t.Fatalf("recovered terminal=%+v err=%v", terminal, err)
	}
	events, err := eventStore.Get(t.Context(), run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatal(err)
	}
	saved.Status = domain.RunStatusComplete
	state, err := meteredWorkflowChildState(saved, events, time.Now())
	if err != nil || !state.TokensKnown || !state.ChargeMeasured || state.Tokens != 23 || state.ChargeMicroUSD != 0 {
		t.Fatalf("saved receipt lost: %+v %v", state, err)
	}
	for _, evt := range events {
		if usage, ok := evt.Data.(*domain.UsageEventData); ok && usage.Model != "original-model" {
			t.Fatalf("recovered model changed: %+v", usage)
		}
	}
}

func TestContinuationTranscriptBoundarySurvivesOwnerRestart(t *testing.T) {
	reconciler, repos, eventStore := newRecoveryTestReconciler(t, domain.RunnerTypeCodex)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
	root := t.TempDir()
	o := New(nil, repos.Tasks, repos.Runs, WithRunStateRoot(root))
	t.Cleanup(o.dispatcher.Close)
	first, closeFirst, err := o.prepareRunTranscript(t.Context(), run, ".")
	if err != nil {
		t.Fatal(err)
	}
	old := "{\"type\":\"turn.completed\",\"usage\":{\"input_tokens\":10}}\n"
	if _, err := first.StdoutFile.WriteString(old); err != nil {
		t.Fatal(err)
	}
	if err := first.OnAdvance(int64(len(old)), 7); err != nil {
		t.Fatal(err)
	}
	closeFirst()
	saved, err := repos.Runs.Get(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, closeSecond, err := o.prepareRunTranscript(t.Context(), saved, ".")
	if err != nil {
		t.Fatal(err)
	}
	defer closeSecond()
	snapshot, err := runstate.Load(run.ID, root)
	if err != nil || snapshot.Cursor.TranscriptCursor != int64(len(old)) || snapshot.Cursor.TranscriptLastSeq != 7 {
		t.Fatalf("continuation reset retained cursor: %+v %v", snapshot, err)
	}
	reloaded, err := repos.Runs.Get(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.TranscriptCursor != int64(len(old)) || reloaded.TranscriptLastSeq != 7 {
		t.Fatalf("durable owner cursor lost: %+v", reloaded)
	}
	// Recovery before the new provider emits anything must not see the
	// original turn's success marker as completion of this continuation.
	terminal, err := reconciler.drainTranscript(t.Context(), reloaded, second.TranscriptPath, nil, codecs.NewCodexForTest())
	if err != nil || terminal != nil {
		t.Fatalf("old completion replayed into correction: %+v %v", terminal, err)
	}
	if _, err := second.StdoutFile.WriteString("{\"type\":\"turn.completed\",\"usage\":{\"input_tokens\":5}}\n"); err != nil {
		t.Fatal(err)
	}
	terminal, err = reconciler.drainTranscript(t.Context(), reloaded, second.TranscriptPath, nil, codecs.NewCodexForTest())
	if err != nil || terminal == nil || !terminal.Success {
		t.Fatalf("new terminal lost: %+v %v", terminal, err)
	}
	events, err := eventStore.Get(t.Context(), run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatal(err)
	}
	tokens := 0
	for _, event := range events {
		if usage, ok := event.Data.(*domain.UsageEventData); ok {
			tokens += usage.InputTokens
		}
	}
	if tokens != 5 {
		t.Fatalf("recovery duplicated prior invocation: tokens=%d", tokens)
	}
}

func (s *pendingRunRecoveryStub) ResumeRun(_ context.Context, id uuid.UUID) (*domain.Run, error) {
	s.ids = append(s.ids, id)
	return &domain.Run{ID: id}, nil
}

func TestRecoverInFlightRunsReenqueuesPendingRuns(t *testing.T) {
	reconciler, repos, _ := newRecoveryTestReconciler(t, domain.RunnerTypeCodex)
	stub := &pendingRunRecoveryStub{}
	reconciler.pendingRunRecovery = stub

	run := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
	run.Status = domain.RunStatusPending
	if err := repos.Runs.Update(context.Background(), run); err != nil {
		t.Fatalf("update pending run: %v", err)
	}

	if err := reconciler.RecoverInFlightRuns(context.Background()); err != nil {
		t.Fatalf("RecoverInFlightRuns: %v", err)
	}
	if len(stub.ids) != 1 || stub.ids[0] != run.ID {
		t.Fatalf("re-enqueued run ids = %v, want [%s]", stub.ids, run.ID)
	}
}

func TestRecoverPanickedRunFailsOnlyAffectedRunWithStackEvent(t *testing.T) {
	_, repos, eventStore := newRecoveryTestReconciler(t, domain.RunnerTypeCodex)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
	other := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
	orchestrator := New(nil, nil, repos.Runs, WithEvents(eventStore))
	orchestrator.recoverPanickedRun(run, obs.PanicFailure{Operation: "injected phase", Value: "boom", Stack: "stacktrace"})

	failed, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != domain.RunStatusFailed {
		t.Fatalf("affected status = %s, want failed", failed.Status)
	}
	unchanged, err := repos.Runs.Get(context.Background(), other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Status != domain.RunStatusRunning {
		t.Fatalf("unaffected run status = %s, want running", unchanged.Status)
	}
	events, err := eventStore.Get(context.Background(), run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatal(err)
	}
	foundStack := false
	for _, evt := range events {
		if log, ok := evt.Data.(*domain.LogEventData); ok && strings.Contains(log.Message, "stacktrace") {
			foundStack = true
		}
	}
	if !foundStack {
		t.Fatal("expected recovered panic stack event")
	}
}

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

func TestRecoverRun_DeadProcessWithoutTerminalEventPreservesRecoverableInterruption(t *testing.T) {
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
	if got.Status != domain.RunStatusNeedsReview {
		t.Fatalf("status = %s, want %s", got.Status, domain.RunStatusNeedsReview)
	}
	if got.TerminalClass != domain.RunTerminalClassInterruption || got.StopReason != domain.RunStopReasonCrash {
		t.Fatalf("recovery classification = (%s,%s), want (%s,%s)", got.TerminalClass, got.StopReason, domain.RunTerminalClassInterruption, domain.RunStopReasonCrash)
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

	reconciler.runStateRoot = t.TempDir()
	dir, err := runstate.RunDir(reconciler.runStateRoot, run.ID)
	if err != nil {
		t.Fatalf("run state dir: %v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	reconciler.cleanupRunStateDirs(context.Background())

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed, stat err=%v", dir, err)
	}
}

func TestCleanupRunStateDirs_ReclaimsOnlyExpiredOrphans(t *testing.T) {
	reconciler, repos, _ := newRecoveryTestReconciler(t, domain.RunnerTypeClaudeCode)
	reconciler.runStateRoot = t.TempDir()
	oldOrphan := filepath.Join(reconciler.runStateRoot, uuid.NewString())
	freshOrphan := filepath.Join(reconciler.runStateRoot, uuid.NewString())
	for _, path := range []string{oldOrphan, freshOrphan} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("create orphan dir: %v", err)
		}
	}
	old := time.Now().Add(-8 * 24 * time.Hour)
	if err := os.Chtimes(oldOrphan, old, old); err != nil {
		t.Fatalf("age orphan dir: %v", err)
	}

	// A matching row remains protected regardless of the directory's age.
	known := createRecoveryTestRun(t, repos, domain.RunnerTypeClaudeCode)
	knownDir, err := runstate.RunDir(reconciler.runStateRoot, known.ID)
	if err != nil {
		t.Fatalf("known run dir: %v", err)
	}
	if err := os.MkdirAll(knownDir, 0o755); err != nil {
		t.Fatalf("create known dir: %v", err)
	}
	if err := os.Chtimes(knownDir, old, old); err != nil {
		t.Fatalf("age known dir: %v", err)
	}

	reconciler.cleanupRunStateDirs(context.Background())
	if _, err := os.Stat(oldOrphan); !os.IsNotExist(err) {
		t.Fatalf("expired orphan still exists: %v", err)
	}
	if _, err := os.Stat(freshOrphan); err != nil {
		t.Fatalf("fresh orphan was removed: %v", err)
	}
	if _, err := os.Stat(knownDir); err != nil {
		t.Fatalf("directory with matching row was removed: %v", err)
	}
}

var (
	runStateCacheFiles = []string{
		"runtime/codex/cache/remote_plugin_catalog/catalog.json",
		"runtime/codex/plugins/cache/openai-curated-remote/reference.pptx",
		"codex/cache/remote_plugin_catalog/catalog.json",
		"codex/plugins/cache/openai-curated-remote/reference.docx",
	}
	runStateKeptFiles = []string{
		"transcript.ndjson",
		"runtime/codex/sessions/2026/09/14/rollout-thread.jsonl",
		"runtime/codex/state_5.sqlite",
		"codex/sessions/2026/09/14/rollout-thread.jsonl",
	}
)

func seedRunStateDir(t *testing.T, root string, runID uuid.UUID) string {
	t.Helper()
	dir, err := runstate.RunDir(root, runID)
	if err != nil {
		t.Fatalf("run state dir: %v", err)
	}
	for _, relative := range append(append([]string{}, runStateCacheFiles...), runStateKeptFiles...) {
		path := filepath.Join(dir, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func setRecoveryTestRunStatus(t *testing.T, repos *database.Repositories, run *domain.Run, status domain.RunStatus, endedAt *time.Time) {
	t.Helper()
	run.Status, run.EndedAt = status, endedAt
	if err := repos.Runs.Update(context.Background(), run); err != nil {
		t.Fatalf("update run: %v", err)
	}
}

func requireCachesPrunedStateKept(t *testing.T, dir string) {
	t.Helper()
	for _, relative := range runStateCacheFiles {
		if _, err := os.Stat(filepath.Join(dir, relative)); !os.IsNotExist(err) {
			t.Fatalf("runner cache %s survived: %v", relative, err)
		}
	}
	for _, relative := range runStateKeptFiles {
		if _, err := os.Stat(filepath.Join(dir, relative)); err != nil {
			t.Fatalf("run state %s was removed: %v", relative, err)
		}
	}
}

func TestCleanupRunStateDirs_AgesOutUnknownImportedRuns(t *testing.T) {
	reconciler, repos, _ := newRecoveryTestReconciler(t, domain.RunnerTypeClaudeCode)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeClaudeCode)
	setRecoveryTestRunStatus(t, repos, run, domain.RunStatusUnknown, nil)
	reconciler.runStateRoot = t.TempDir()
	dir := seedRunStateDir(t, reconciler.runStateRoot, run.ID)
	reconciler.clock = func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }

	reconciler.cleanupRunStateDirs(context.Background())

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("unknown-status run state outlived retention: %v", err)
	}
}

func TestCleanupRunStateDirs_PrunesOnlyCachesWithinRetention(t *testing.T) {
	reconciler, repos, _ := newRecoveryTestReconciler(t, domain.RunnerTypeCodex)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
	ended := time.Now()
	setRecoveryTestRunStatus(t, repos, run, domain.RunStatusComplete, &ended)
	reconciler.runStateRoot = t.TempDir()
	dir := seedRunStateDir(t, reconciler.runStateRoot, run.ID)

	reconciler.cleanupRunStateDirs(context.Background())
	for _, relative := range runStateCacheFiles {
		if _, err := os.Stat(filepath.Join(dir, relative)); err != nil {
			t.Fatalf("cache %s pruned inside the follow-up grace period: %v", relative, err)
		}
	}

	reconciler.clock = func() time.Time { return time.Now().Add(2 * time.Hour) }
	reconciler.cleanupRunStateDirs(context.Background())
	requireCachesPrunedStateKept(t, dir)
}

func TestCleanupRunStateDirs_RestingRuns(t *testing.T) {
	for _, test := range []struct {
		name      string
		status    domain.RunStatus
		idle      time.Duration
		removeDir bool
	}{
		{"abandoned review", domain.RunStatusNeedsReview, 31 * 24 * time.Hour, true},
		{"recent review", domain.RunStatusNeedsReview, 2 * time.Hour, false},
		{"long parked", domain.RunStatusParked, 90 * 24 * time.Hour, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			reconciler, repos, _ := newRecoveryTestReconciler(t, domain.RunnerTypeCodex)
			run := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
			setRecoveryTestRunStatus(t, repos, run, test.status, nil)
			reconciler.runStateRoot = t.TempDir()
			dir := seedRunStateDir(t, reconciler.runStateRoot, run.ID)
			reconciler.clock = func() time.Time { return time.Now().Add(test.idle) }

			reconciler.cleanupRunStateDirs(context.Background())

			if test.removeDir {
				if _, err := os.Stat(dir); !os.IsNotExist(err) {
					t.Fatalf("abandoned review state kept: %v", err)
				}
				return
			}
			requireCachesPrunedStateKept(t, dir)
		})
	}
}

func TestCleanupRunStateDirs_SkipsRunsUnderRecoveryTailing(t *testing.T) {
	reconciler, repos, _ := newRecoveryTestReconciler(t, domain.RunnerTypeCodex)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
	ended := time.Now()
	setRecoveryTestRunStatus(t, repos, run, domain.RunStatusComplete, &ended)
	reconciler.runStateRoot = t.TempDir()
	dir := seedRunStateDir(t, reconciler.runStateRoot, run.ID)
	reconciler.recoveryMu.Lock()
	if reconciler.tailers == nil {
		reconciler.tailers = map[uuid.UUID]context.CancelFunc{}
	}
	reconciler.tailers[run.ID] = func() {}
	reconciler.recoveryMu.Unlock()
	reconciler.clock = func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }

	reconciler.cleanupRunStateDirs(context.Background())

	for _, relative := range append(append([]string{}, runStateCacheFiles...), runStateKeptFiles...) {
		if _, err := os.Stat(filepath.Join(dir, relative)); err != nil {
			t.Fatalf("state of a run under recovery tailing was removed (%s): %v", relative, err)
		}
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
		WithReconcilerRunStateRoot(t.TempDir()),
		WithReconcilerConfig(ReconcilerConfig{
			Interval:          time.Hour,
			StaleThreshold:    time.Minute,
			MaxRecoveryAge:    10 * time.Minute,
			OrphanGracePeriod: time.Minute,
			MaxStaleRuns:      10,
			PendingThreshold:  time.Minute,
			KillOrphans:       true,
			AutoRecover:       true,
		}),
	)

	return reconciler, repos, eventStore
}

func TestReconcileReapsAgedPendingRunWithEvent(t *testing.T) {
	reconciler, repos, eventStore := newRecoveryTestReconciler(t, domain.RunnerTypeCodex)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
	run.Status = domain.RunStatusPending
	if err := repos.Runs.Update(context.Background(), run); err != nil {
		t.Fatalf("update pending run: %v", err)
	}
	// Keep the test fast while exercising the same age comparison production
	// uses for a genuinely stranded queue entry.
	reconciler.config.PendingThreshold = time.Nanosecond

	reconciler.reconcile(context.Background())
	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get reaped run: %v", err)
	}
	if got.Status != domain.RunStatusFailed {
		t.Fatalf("status = %s, want %s", got.Status, domain.RunStatusFailed)
	}
	if got.ErrorMsg == "" {
		t.Fatal("expected pending reap reason")
	}
	events, err := eventStore.Get(context.Background(), run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatalf("get events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected explanatory pending reap event")
	}
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
