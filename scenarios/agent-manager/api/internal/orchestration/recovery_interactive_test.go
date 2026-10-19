package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"

	"github.com/google/uuid"
)

// fakeInteractiveSessions is an in-memory webconsole.SessionController for
// recovery tests: it reports whether a session is alive and records deletes.
type fakeInteractiveSessions struct {
	mu         sync.Mutex
	alive      map[string]bool
	getErr     error
	getErrOnce bool
}

type retainedGoalReplayRunner struct{ *mocks.TranscriptReplayRunner }

func (r *retainedGoalReplayRunner) NewTranscriptParser() runner.TranscriptParser {
	return codecs.NewCodexForTest().NewTranscriptParser()
}

type continuationRecoveryRunner struct{ *mocks.TranscriptReplayRunner }

func (r *continuationRecoveryRunner) NewTranscriptParser() runner.TranscriptParser {
	return continuationRecoveryParser{base: r.TranscriptReplayRunner.NewTranscriptParser()}
}

type continuationRecoveryParser struct{ base runner.TranscriptParser }

func (p continuationRecoveryParser) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	switch strings.TrimSpace(line) {
	case "user-turn":
		return runner.TranscriptParseResult{UserTurnStarted: true}
	case "provider-turn":
		return runner.TranscriptParseResult{ProviderTurnStarted: true}
	case "goal-active":
		return runner.TranscriptParseResult{Goal: &runner.GoalMarker{Objective: "finish the current objective", Status: runner.GoalStatusActive}}
	case "goal-paused":
		return runner.TranscriptParseResult{Goal: &runner.GoalMarker{Objective: "finish the current objective", Status: runner.GoalStatusPaused}}
	default:
		return p.base.ParseTranscriptLine(runID, line)
	}
}

func TestRecoverInteractiveConsumedGoalReconcilesDespiteAttachedTailer(t *testing.T) {
	reconciler, repos := newInteractiveRecoveryReconciler(t, newFakeInteractiveSessions("retained-ui"))
	if err := reconciler.runners.Register(&retainedGoalReplayRunner{mocks.NewTranscriptReplayRunner(domain.RunnerTypeCodex)}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	run := createInteractiveRecoveryRun(t, repos, "retained-ui", path)
	run.SessionID = "exact-native-session"
	run.ResolvedConfig.RunnerType = domain.RunnerTypeCodex
	run.ResolvedConfig.Until = "finish assignment"
	objective := "finish assignment"
	body, markerAt := writeAcceptedRetainedGoalTranscript(t, run, objective)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	run.TranscriptCursor = int64(len(body))
	if err := repos.Runs.Update(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	cancelled := false
	reconciler.tailers[run.ID] = func() { cancelled = true }
	result, err := reconciler.RecoverRun(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !cancelled || !result.Recovered || result.Run.Status != domain.RunStatusComplete || result.Run.Phase != domain.RunPhaseCompleted || result.Run.StopReason != domain.RunStopReasonComplete {
		t.Fatal("attached stale tailer masked accepted goal", result)
	}
	retained, err := repos.Runs.Get(t.Context(), run.ID)
	if err != nil || retained.Status != domain.RunStatusComplete || retained.TranscriptCursor != int64(len(body)) {
		t.Fatal("recovery not durable or replayed cursor", retained, err)
	}
	observed := *retained
	view := &Orchestrator{events: reconciler.events}
	if err := view.observeInteractiveRun(t.Context(), &observed); err != nil || observed.ObservedGoalStatus != "complete" || observed.ObservedGoalAt == nil || !observed.ObservedGoalAt.Equal(markerAt) {
		t.Fatal("ordinary event-derived run projection did not converge", observed.ObservedGoalStatus, observed.ObservedGoalAt, err)
	}
	goalEvents, err := reconciler.events.Get(t.Context(), run.ID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeGoalStatusChanged}})
	if err != nil || len(goalEvents) != 1 {
		t.Fatal("retained accepted goal did not converge exactly one goal status event", len(goalEvents), err)
	}
	goalData, ok := goalEvents[0].Data.(*domain.GoalStatusChangedEventData)
	if !ok || goalData.Objective != objective || goalData.Status != "complete" || !goalEvents[0].Timestamp.Equal(markerAt) {
		t.Fatalf("retained goal event did not preserve the matched marker: %#v", goalEvents[0])
	}
	if _, err := reconciler.RecoverRun(t.Context(), run.ID); err != nil {
		t.Fatal("terminal recovery replay failed", err)
	}
	goalEvents, err = reconciler.events.Get(t.Context(), run.ID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeGoalStatusChanged}})
	if err != nil || len(goalEvents) != 1 {
		t.Fatal("terminal recovery duplicated retained goal event", len(goalEvents), err)
	}
}

func writeAcceptedRetainedGoalTranscript(t *testing.T, run *domain.Run, objective string) (string, time.Time) {
	t.Helper()
	activeAt := run.StartedAt.Add(time.Second)
	markerAt := run.StartedAt.Add(2 * time.Second)
	line := func(at time.Time, eventType string, payload any) string {
		t.Helper()
		encoded, err := json.Marshal(map[string]any{"timestamp": at.Format(time.RFC3339Nano), "type": eventType, "payload": payload})
		if err != nil {
			t.Fatal(err)
		}
		return string(encoded) + "\n"
	}
	receipt, err := json.Marshal(map[string]any{"goal": map[string]any{
		"threadId": run.SessionID, "objective": objective, "status": "complete",
	}})
	if err != nil {
		t.Fatal(err)
	}
	body := strings.Join([]string{
		line(activeAt, "session_meta", map[string]any{"id": run.SessionID}),
		line(activeAt, "event_msg", map[string]any{"type": "thread_goal_updated", "threadId": run.SessionID, "goal": map[string]any{"threadId": run.SessionID, "objective": objective, "status": "active"}}),
		line(markerAt, "response_item", map[string]any{
			"type": "custom_tool_call", "name": "exec", "call_id": "complete-call",
			"input": "const r = await tools.update_goal({status:\"complete\"});\ntext(JSON.stringify(r, null, 2));",
		}),
		line(markerAt, "response_item", map[string]any{
			"type": "custom_tool_call_output", "call_id": "complete-call",
			"output": []any{
				map[string]any{"type": "input_text", "text": "Script completed\nWall time 0.1 seconds\nOutput:\n"},
				map[string]any{"type": "input_text", "text": string(receipt)},
			},
		}),
	}, "")
	return body, markerAt
}

func TestRecoverAlreadyCompletedInteractiveGoalConvergesObservedState(t *testing.T) {
	reconciler, repos := newInteractiveRecoveryReconciler(t, newFakeInteractiveSessions("retained-terminal-ui"))
	if err := reconciler.runners.Register(&retainedGoalReplayRunner{mocks.NewTranscriptReplayRunner(domain.RunnerTypeCodex)}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	run := createInteractiveRecoveryRun(t, repos, "retained-terminal-ui", path)
	run.SessionID = "exact-native-session"
	run.ResolvedConfig.RunnerType = domain.RunnerTypeCodex
	run.ResolvedConfig.Until = "finish assignment"
	body, markerAt := writeAcceptedRetainedGoalTranscript(t, run, run.ResolvedConfig.Until)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	activeAt := run.StartedAt.Add(time.Second)
	active := domain.NewGoalStatusChangedEvent(run.ID, run.ResolvedConfig.Until, "active", 0, "")
	active.Timestamp = activeAt
	if err := reconciler.events.Append(t.Context(), run.ID, active); err != nil {
		t.Fatal(err)
	}
	endedAt := run.StartedAt.Add(3 * time.Second)
	run.TranscriptCursor = int64(len(body))
	run.Status = domain.RunStatusComplete
	run.Phase = domain.RunPhaseCompleted
	run.EndedAt = &endedAt
	run.StopReason = domain.RunStopReasonComplete
	run.Result = &domain.RunResult{Success: true, TerminalReason: "goal_complete"}
	if err := repos.Runs.Update(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	result, err := reconciler.RecoverRun(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Recovered || result.Run.Status != domain.RunStatusComplete {
		t.Fatalf("terminal goal reconciliation changed run lifecycle or did not report repair: %#v", result)
	}
	retained, err := repos.Runs.Get(t.Context(), run.ID)
	if err != nil || retained.Status != domain.RunStatusComplete {
		t.Fatalf("terminal run lifecycle changed: %#v, %v", retained, err)
	}
	observed := *retained
	view := &Orchestrator{events: reconciler.events}
	if err := view.observeInteractiveRun(t.Context(), &observed); err != nil || observed.ObservedGoalStatus != "complete" || observed.ObservedGoalAt == nil || !observed.ObservedGoalAt.Equal(markerAt) {
		t.Fatalf("ordinary event-derived terminal projection did not converge: %#v, %v", observed, err)
	}
	if retained.EndedAt == nil || !retained.EndedAt.Equal(endedAt) || retained.Result == nil || retained.Result.TerminalReason != "goal_complete" {
		t.Fatal("goal projection repair changed terminal lifecycle evidence", retained)
	}
	goalEvents, err := reconciler.events.Get(t.Context(), run.ID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeGoalStatusChanged}})
	if err != nil || len(goalEvents) != 2 {
		t.Fatalf("terminal repair should preserve active and append exactly one complete marker, got %d events: %v", len(goalEvents), err)
	}
	data, ok := goalEvents[1].Data.(*domain.GoalStatusChangedEventData)
	if !ok || data.Status != "complete" || data.Objective != run.ResolvedConfig.Until || data.Iteration != 0 || data.LastReason != "" || !goalEvents[1].Timestamp.Equal(markerAt) {
		t.Fatalf("terminal repair did not preserve exact final marker: %#v", goalEvents[1])
	}
	if _, err := reconciler.RecoverRun(t.Context(), run.ID); err != nil {
		t.Fatal("second terminal repair failed", err)
	}
	goalEvents, err = reconciler.events.Get(t.Context(), run.ID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeGoalStatusChanged}})
	if err != nil || len(goalEvents) != 2 {
		t.Fatalf("second terminal repair duplicated goal event: %d, %v", len(goalEvents), err)
	}
}

func TestRecoverAlreadyCompletedInteractiveGoalIgnoresPostEndMarker(t *testing.T) {
	reconciler, repos := newInteractiveRecoveryReconciler(t, newFakeInteractiveSessions("retained-late-ui"))
	if err := reconciler.runners.Register(&retainedGoalReplayRunner{mocks.NewTranscriptReplayRunner(domain.RunnerTypeCodex)}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	run := createInteractiveRecoveryRun(t, repos, "retained-late-ui", path)
	run.SessionID = "exact-native-session"
	run.ResolvedConfig.RunnerType = domain.RunnerTypeCodex
	run.ResolvedConfig.Until = "finish assignment"
	body, markerAt := writeAcceptedRetainedGoalTranscript(t, run, run.ResolvedConfig.Until)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	active := domain.NewGoalStatusChangedEvent(run.ID, run.ResolvedConfig.Until, "active", 0, "")
	active.Timestamp = run.StartedAt.Add(time.Second)
	if err := reconciler.events.Append(t.Context(), run.ID, active); err != nil {
		t.Fatal(err)
	}
	endedAt := markerAt.Add(-time.Second)
	run.TranscriptCursor = int64(len(body))
	run.Status = domain.RunStatusComplete
	run.Phase = domain.RunPhaseCompleted
	run.EndedAt = &endedAt
	run.StopReason = domain.RunStopReasonComplete
	run.Result = &domain.RunResult{Success: true, TerminalReason: "goal_complete"}
	if err := repos.Runs.Update(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	result, err := reconciler.RecoverRun(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Idempotent || result.Run.ObservedGoalStatus != "" {
		t.Fatalf("post-EndAt goal marker was projected onto an already-completed run: %#v", result)
	}
	retained, err := repos.Runs.Get(t.Context(), run.ID)
	if err != nil || retained.ObservedGoalStatus != "" || retained.EndedAt == nil || !retained.EndedAt.Equal(endedAt) {
		t.Fatalf("post-EndAt marker changed terminal projection: %#v, %v", retained, err)
	}
	events, err := reconciler.events.Get(t.Context(), run.ID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeGoalStatusChanged}})
	if err != nil || len(events) != 1 {
		t.Fatalf("post-EndAt marker appended an event: %d, %v", len(events), err)
	}
}

func newFakeInteractiveSessions(aliveIDs ...string) *fakeInteractiveSessions {
	f := &fakeInteractiveSessions{alive: map[string]bool{}}
	for _, id := range aliveIDs {
		f.alive[id] = true
	}
	return f
}

func (f *fakeInteractiveSessions) setGone(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.alive[id] = false
}

func (f *fakeInteractiveSessions) CreateSession(context.Context, webconsole.CreateSessionParams) (string, error) {
	return "", nil
}

func (f *fakeInteractiveSessions) GetSession(_ context.Context, id string) (webconsole.SessionInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		err := f.getErr
		if f.getErrOnce {
			f.getErr = nil
		}
		return webconsole.SessionInfo{}, err
	}
	if f.alive[id] {
		return webconsole.SessionInfo{ID: id, Owner: webconsole.OwnerAgentManager}, nil
	}
	return webconsole.SessionInfo{}, webconsole.ErrSessionNotFound
}

func (f *fakeInteractiveSessions) ListSessions(context.Context) ([]webconsole.SessionInfo, error) {
	return nil, nil
}

func (f *fakeInteractiveSessions) ArchiveSession(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.alive[id] = false
	return nil
}

func (f *fakeInteractiveSessions) SendText(context.Context, string, string, string) error { return nil }
func (f *fakeInteractiveSessions) SendCommand(context.Context, string, string, string) (webconsole.PromptSubmission, error) {
	return webconsole.PromptSubmission{Verified: true, EnterPresses: 1}, nil
}
func (f *fakeInteractiveSessions) SendPrompt(context.Context, string, string, string) (webconsole.PromptSubmission, error) {
	return webconsole.PromptSubmission{Verified: true, EnterPresses: 1}, nil
}
func (f *fakeInteractiveSessions) Interrupt(context.Context, string, string) error { return nil }
func (f *fakeInteractiveSessions) Screen(context.Context, string, bool) (string, error) {
	return "", nil
}

func newInteractiveRecoveryReconciler(t *testing.T, sessions webconsole.SessionController) (*Reconciler, *database.Repositories) {
	t.Helper()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	registry := runner.NewRegistry()
	if err := registry.Register(mocks.NewTranscriptReplayRunner(domain.RunnerTypeClaudeCode)); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	reconciler := NewReconciler(
		repos.Runs,
		registry,
		WithReconcilerEvents(eventStore),
		WithReconcilerInteractive(sessions),
		WithReconcilerRunStateRoot(t.TempDir()),
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
	reconciler.interactiveDebounce = 40 * time.Millisecond
	reconciler.interactiveSessionPoll = 20 * time.Millisecond
	return reconciler, repos
}

func createInteractiveRecoveryRun(t *testing.T, repos *database.Repositories, sessionID, transcriptPath string) *domain.Run {
	t.Helper()
	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "Interactive recovery task",
		Description: "Validate interactive recovery",
		ScopePath:   ".",
		ProjectRoot: ".",
		Status:      domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	now := time.Now().UTC()
	stale := now.Add(-15 * time.Minute)
	cfg := domain.DefaultRunConfig()
	cfg.RunnerType = domain.RunnerTypeClaudeCode
	run := &domain.Run{
		ID:                  uuid.New(),
		TaskID:              task.ID,
		Tag:                 uuid.NewString(),
		RunMode:             domain.RunModeInPlace,
		Status:              domain.RunStatusRunning,
		StartedAt:           &now,
		LastHeartbeat:       &stale,
		Phase:               domain.RunPhaseExecuting,
		ApprovalState:       domain.ApprovalStateNone,
		ExecutionMode:       domain.ExecutionModeInteractive,
		WebConsoleSessionID: sessionID,
		TranscriptPath:      transcriptPath,
		ResolvedConfig:      cfg,
	}
	if err := repos.Runs.Create(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	return run
}

// TestRecoverInteractive_SessionGoneNoTerminal_FailsExplicitly proves an
// interactive run whose web-console session vanished (and has no terminal
// marker) is finalized Failed with an explicit reason naming the session —
// never left orphaned.
func TestRecoverInteractive_SessionGoneNoTerminal_FailsExplicitly(t *testing.T) {
	sessions := newFakeInteractiveSessions() // session id not alive => gone
	reconciler, repos := newInteractiveRecoveryReconciler(t, sessions)

	transcript := writeRecoveryTranscript(t, "message:partial interactive output\n")
	run := createInteractiveRecoveryRun(t, repos, "sess-gone", transcript)

	if _, err := reconciler.RecoverRun(context.Background(), run.ID); err != nil {
		t.Fatalf("RecoverRun: %v", err)
	}

	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusNeedsReview {
		t.Fatalf("status = %s, want recoverable needs_review", got.Status)
	}
	if !containsAll(got.ErrorMsg, "sess-gone", "no longer exists") {
		t.Errorf("error msg = %q, want it to name the vanished session", got.ErrorMsg)
	}
}

// TestRecoverInteractive_SessionGoneWithTerminal_Completes proves that when the
// transcript already carries a success terminal and the session is gone, the run
// completes (the last turn finished before the session was cleaned up).
func TestRecoverInteractive_SessionGoneWithTerminal_Completes(t *testing.T) {
	sessions := newFakeInteractiveSessions()
	reconciler, repos := newInteractiveRecoveryReconciler(t, sessions)

	transcript := writeRecoveryTranscript(t, "message:final interactive answer\ndone:final interactive answer\n")
	run := createInteractiveRecoveryRun(t, repos, "sess-gone", transcript)

	if _, err := reconciler.RecoverRun(context.Background(), run.ID); err != nil {
		t.Fatalf("RecoverRun: %v", err)
	}

	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusComplete {
		t.Fatalf("status = %s, want complete", got.Status)
	}
}

// TestRecoverInteractive_SessionAliveReattachesUntilTerminal is the in-process
// restart drill: it drives the REAL recovery entry point (RecoverInFlightRuns),
// which discovers the in-flight interactive run from the persisted row, verifies
// the session is alive, and reattaches the transcript tailer. A terminal marker
// appended after reattach then completes the run — proving reattachment survives
// a simulated agent-manager restart.
func TestRecoverInteractive_SessionAliveReattachesUntilTerminal(t *testing.T) {
	sessions := newFakeInteractiveSessions("sess-live")
	reconciler, repos := newInteractiveRecoveryReconciler(t, sessions)

	// Seed the transcript with pre-restart output but no terminal yet.
	transcript := filepath.Join(t.TempDir(), "transcript.ndjson")
	if err := os.WriteFile(transcript, []byte("message:output before restart\n"), 0o644); err != nil {
		t.Fatalf("seed transcript: %v", err)
	}
	run := createInteractiveRecoveryRun(t, repos, "sess-live", transcript)

	// Real restart-recovery boot path: discover + reattach.
	if err := reconciler.RecoverInFlightRuns(context.Background()); err != nil {
		t.Fatalf("RecoverInFlightRuns: %v", err)
	}

	// The run must still be running (reattached, not finalized) right after boot.
	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusRunning {
		t.Fatalf("status after reattach = %s, want still running", got.Status)
	}

	// The agent finishes its turn: append a final message + terminal marker to
	// the live session's transcript. The reattached tailer must observe it and
	// complete the run, without re-emitting the pre-restart output.
	appendRecoveryTranscript(t, transcript, "message:interactive completion\ndone:finished\n")

	waitForRecoveryStatus(t, repos, run.ID, domain.RunStatusComplete)
	final, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if final.Summary == nil || final.Summary.Description != "interactive completion" {
		t.Fatalf("summary = %+v, want interactive completion", final.Summary)
	}
}

func TestRecoverInteractive_ContinuedCompletedGoalPolicySurvivesRestart(t *testing.T) {
	for _, tc := range []struct {
		name        string
		sessionLine string
		continued   string
		wantStatus  domain.RunStatus
		wantSuccess bool
		checkResult bool
	}{
		{name: "terminal and cursor persisted before finalize", continued: "user-turn\nmessage:continued work\ndone:continued result\n", wantStatus: domain.RunStatusComplete, wantSuccess: true, checkResult: true},
		{name: "failed terminal remains failure", continued: "user-turn\nfail:provider failure\n", wantStatus: domain.RunStatusFailed},
		{name: "later user turn invalidates earlier success", continued: "user-turn\ndone:first turn\nuser-turn\n", wantStatus: domain.RunStatusRunning},
		{name: "later provider turn invalidates earlier success", continued: "user-turn\ndone:first turn\nprovider-turn\n", wantStatus: domain.RunStatusRunning},
		{name: "active goal does not use prior success", continued: "user-turn\ndone:first turn\ngoal-active\n", wantStatus: domain.RunStatusRunning},
		{name: "paused goal does not use prior success", continued: "user-turn\ndone:first turn\ngoal-paused\n", wantStatus: domain.RunStatusRunning},
		{name: "old pre-invocation terminal is outside the segment", continued: "", wantStatus: domain.RunStatusRunning},
		{name: "missing transcript session evidence fails closed", sessionLine: "", continued: "user-turn\ndone:continued result\n", wantStatus: domain.RunStatusRunning},
		{name: "other transcript session evidence fails closed", sessionLine: "session:other-session\n", continued: "user-turn\ndone:continued result\n", wantStatus: domain.RunStatusRunning},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sessionID := "sess-post-goal-continuation-" + strings.ReplaceAll(tc.name, " ", "-")
			reconciler, repos := newInteractiveRecoveryReconciler(t, newFakeInteractiveSessions(sessionID))
			base := mocks.NewTranscriptReplayRunner(domain.RunnerTypeCodex)
			if err := reconciler.runners.Register(&continuationRecoveryRunner{TranscriptReplayRunner: base}); err != nil {
				t.Fatal(err)
			}
			transcript := filepath.Join(t.TempDir(), "transcript.ndjson")
			run := createInteractiveRecoveryRun(t, repos, sessionID, transcript)
			run.SessionID = "native-session-" + strings.ReplaceAll(tc.name, " ", "-")
			run.ResolvedConfig.RunnerType = domain.RunnerTypeCodex
			run.ResolvedConfig.Until = "finish the current objective"
			transcriptSession := tc.sessionLine
			if transcriptSession == "" && !strings.HasPrefix(tc.name, "missing transcript session") {
				transcriptSession = "session:" + run.SessionID + "\n"
			}
			priorTurn := transcriptSession + "done:original completed native-goal turn\n"
			if err := os.WriteFile(transcript, []byte(priorTurn), 0o644); err != nil {
				t.Fatal(err)
			}
			invocation := time.Now().UTC().Truncate(time.Microsecond)
			startCursor := int64(len(priorTurn))
			run.InteractiveInvocationStartedAt = &invocation
			run.TranscriptCursor = startCursor
			run.ResolvedConfig.InteractiveContinuation = &domain.InteractiveContinuationPolicy{
				InvocationStartedAt: invocation, TranscriptStartCursor: &startCursor,
				CompleteAfterFirstSuccessfulTurnWithoutGoalMarker: true,
			}
			if tc.continued != "" {
				appendRecoveryTranscript(t, transcript, tc.continued)
			}
			// Simulate the crash after the transcript tailer durably advanced
			// through the new terminal but before it could finalize the run.
			contents, err := os.ReadFile(transcript)
			if err != nil {
				t.Fatal(err)
			}
			run.TranscriptCursor = int64(len(contents))
			if err := repos.Runs.Update(context.Background(), run); err != nil {
				t.Fatal(err)
			}
			result, err := reconciler.recoverInteractiveRun(context.Background(), run, false)
			if err != nil {
				t.Fatalf("recoverInteractiveRun: %v", err)
			}
			got, err := repos.Runs.Get(context.Background(), run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != tc.wantStatus || result.Run.Status != tc.wantStatus {
				t.Fatalf("status=%s result=%s want=%s (result=%+v)", got.Status, result.Run.Status, tc.wantStatus, result)
			}
			if tc.checkResult && (got.Result == nil || got.Result.Success != tc.wantSuccess || got.Result.TerminalReason != "post_goal_continuation") {
				t.Fatalf("result=%+v want success=%v post-goal-continuation", got.Result, tc.wantSuccess)
			}
			if tc.wantStatus == domain.RunStatusFailed && got.ErrorMsg != "provider failure" {
				t.Fatalf("failure terminal was not preserved: error=%q result=%+v", got.ErrorMsg, got.Result)
			}
		})
	}
}

func TestInteractiveContinuationRecoveryOptionsRequireExactActiveInvocation(t *testing.T) {
	started := time.Date(2026, 10, 18, 19, 30, 0, 0, time.UTC)
	startCursor := int64(0)
	valid := &domain.Run{Status: domain.RunStatusRunning, ExecutionMode: domain.ExecutionModeInteractive,
		InteractiveInvocationStartedAt: &started, ResolvedConfig: &domain.RunConfig{InteractiveContinuation: &domain.InteractiveContinuationPolicy{
			InvocationStartedAt: started, TranscriptStartCursor: &startCursor, CompleteAfterFirstSuccessfulTurnWithoutGoalMarker: true,
		}}}
	if got := interactiveContinuationRecoveryOptions(valid); !got.CompleteAfterFirstSuccessfulTurnWithoutGoalMarker {
		t.Fatal("exact current interactive invocation did not restore its owner completion option")
	}
	stale := *valid
	staleConfig := *valid.ResolvedConfig
	staleConfig.InteractiveContinuation = &domain.InteractiveContinuationPolicy{InvocationStartedAt: started.Add(-time.Second), CompleteAfterFirstSuccessfulTurnWithoutGoalMarker: true}
	stale.ResolvedConfig = &staleConfig
	legacy := *valid
	legacyConfig := *valid.ResolvedConfig
	legacyConfig.InteractiveContinuation = nil
	legacy.ResolvedConfig = &legacyConfig
	missingCursor := *valid
	missingCursorConfig := *valid.ResolvedConfig
	missingCursorConfig.InteractiveContinuation = &domain.InteractiveContinuationPolicy{InvocationStartedAt: started, CompleteAfterFirstSuccessfulTurnWithoutGoalMarker: true}
	missingCursor.ResolvedConfig = &missingCursorConfig
	futureCursor := *valid
	futureCursorConfig := *valid.ResolvedConfig
	future := int64(1)
	futureCursorConfig.InteractiveContinuation = &domain.InteractiveContinuationPolicy{InvocationStartedAt: started, TranscriptStartCursor: &future, CompleteAfterFirstSuccessfulTurnWithoutGoalMarker: true}
	futureCursor.ResolvedConfig = &futureCursorConfig
	inactive := *valid
	inactive.Status = domain.RunStatusComplete
	for name, run := range map[string]*domain.Run{"stale policy timestamp": &stale, "legacy missing policy": &legacy, "missing transcript cursor": &missingCursor, "cursor after observed cursor": &futureCursor, "non-running row": &inactive} {
		t.Run(name, func(t *testing.T) {
			if got := interactiveContinuationRecoveryOptions(run); got.CompleteAfterFirstSuccessfulTurnWithoutGoalMarker {
				t.Fatal("recovery enabled completion for a non-current invocation")
			}
		})
	}
}

// TestRecoverInteractive_SessionDiesMidReattach proves that if the session
// disappears while the reattached tailer is waiting for more output, the run is
// finalized Failed with an explicit reason rather than tailing forever.
func TestRecoverInteractive_SessionDiesMidReattach(t *testing.T) {
	sessions := newFakeInteractiveSessions("sess-flaky")
	reconciler, repos := newInteractiveRecoveryReconciler(t, sessions)
	// Speed up the mid-tail session watcher for this test.
	reconciler.interactiveDebounce = 40 * time.Millisecond
	// Keep the session-reattach tolerance short so the vanished session fails
	// the run promptly.
	reconciler.interactiveSessionReattachWindow = 100 * time.Millisecond
	reconciler.interactiveSessionPoll = 20 * time.Millisecond

	transcript := filepath.Join(t.TempDir(), "transcript.ndjson")
	if err := os.WriteFile(transcript, []byte("message:waiting for input\n"), 0o644); err != nil {
		t.Fatalf("seed transcript: %v", err)
	}
	run := createInteractiveRecoveryRun(t, repos, "sess-flaky", transcript)

	if err := reconciler.RecoverInFlightRuns(context.Background()); err != nil {
		t.Fatalf("RecoverInFlightRuns: %v", err)
	}

	// The session dies while the tailer waits (no terminal ever arrives).
	sessions.setGone("sess-flaky")

	waitForRecoveryStatus(t, repos, run.ID, domain.RunStatusFailed)
	got, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if !containsAll(got.ErrorMsg, "sess-flaky", "no longer exists") {
		t.Errorf("error msg = %q, want it to name the vanished session", got.ErrorMsg)
	}
}

func createUntilRecoveryRun(t *testing.T, repos *database.Repositories, sessionID, transcriptPath string) *domain.Run {
	t.Helper()
	run := createInteractiveRecoveryRun(t, repos, sessionID, transcriptPath)
	run.SessionID = "native-session-for-recovery"
	run.ResolvedConfig.RunnerType = domain.RunnerTypeCodex
	run.ResolvedConfig.Until = "finish Desktop recovery"
	started := run.StartedAt.Add(-time.Second)
	run.InteractiveInvocationStartedAt = &started
	if err := repos.Runs.Update(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	return run
}

func createOversizedTranscript(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "oversized.jsonl")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, (16<<20)+1); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRecoveryOversizedGoalEvidenceAndGoneSessionNeedsReview(t *testing.T) {
	reconciler, repos := newInteractiveRecoveryReconciler(t, newFakeInteractiveSessions())
	run := createUntilRecoveryRun(t, repos, "gone-exact-session", createOversizedTranscript(t))

	result, err := reconciler.RecoverRun(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repos.Runs.Get(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RunStatusNeedsReview || got.EndedAt == nil || got.Result != nil {
		t.Fatalf("oversized evidence with confirmed-gone session must become resumable needs_review without a success result: status=%s ended=%v result=%+v", got.Status, got.EndedAt, got.Result)
	}
	if !containsAll(got.ErrorMsg, "gone-exact-session", "retained native goal evidence unavailable", "completion remains unproven") {
		t.Fatalf("missing-proof details not retained: %q", got.ErrorMsg)
	}
	if result.Run.Status != domain.RunStatusNeedsReview {
		t.Fatalf("result status = %s, want needs_review", result.Run.Status)
	}
}

func TestRecoveryOversizedGoalEvidenceKeepsLiveSessionRunning(t *testing.T) {
	sessions := newFakeInteractiveSessions("live-exact-session")
	reconciler, repos := newInteractiveRecoveryReconciler(t, sessions)
	run := createUntilRecoveryRun(t, repos, "live-exact-session", createOversizedTranscript(t))

	result, err := reconciler.RecoverRun(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repos.Runs.Get(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RunStatusRunning || got.EndedAt != nil {
		t.Fatalf("live session must remain running when completion proof is unavailable: status=%s ended=%v", got.Status, got.EndedAt)
	}
	if !strings.Contains(result.Message, "session remains present") {
		t.Fatalf("result should report live session and unavailable evidence: %q", result.Message)
	}
	if reconciler.hasTailer(run.ID) {
		t.Fatal("ordinary read recovery must not create a new tailer when none was attached")
	}
}

func TestRecoveryEvidenceLivenessErrorRestoresJoinedObserver(t *testing.T) {
	sessions := newFakeInteractiveSessions("live-exact-session")
	sessions.getErr = errors.New("temporary Web Console lookup failure")
	sessions.getErrOnce = true
	reconciler, repos := newInteractiveRecoveryReconciler(t, sessions)
	path := filepath.Join(t.TempDir(), "empty.jsonl")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	run := createUntilRecoveryRun(t, repos, "live-exact-session", path)
	if err := reconciler.runners.Register(&retainedGoalReplayRunner{mocks.NewTranscriptReplayRunner(domain.RunnerTypeCodex)}); err != nil {
		t.Fatal(err)
	}
	if reconciler.tailers == nil {
		reconciler.tailers = make(map[uuid.UUID]context.CancelFunc)
	}
	reconciler.tailers[run.ID] = func() {} // Model a previously attached recovery observer that was joined.

	result, err := reconciler.RecoverRun(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Message, "session liveness unknown") {
		t.Fatalf("result should preserve unknown liveness: %q", result.Message)
	}
	if !reconciler.hasTailer(run.ID) {
		t.Fatal("transient session lookup error must restore the observer that recovery joined")
	}
	defer reconciler.CancelInteractiveTail(run.ID)
	got, err := repos.Runs.Get(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RunStatusRunning || got.EndedAt != nil {
		t.Fatalf("unknown session liveness must not mutate run terminal state: status=%s ended=%v", got.Status, got.EndedAt)
	}
}

func TestStartupRecoveryReattachesWhenGoalEvidenceIsEmpty(t *testing.T) {
	sessions := newFakeInteractiveSessions("live-startup-session")
	reconciler, repos := newInteractiveRecoveryReconciler(t, sessions)
	path := filepath.Join(t.TempDir(), "empty.jsonl")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	run := createUntilRecoveryRun(t, repos, "live-startup-session", path)
	if err := reconciler.runners.Register(&retainedGoalReplayRunner{mocks.NewTranscriptReplayRunner(domain.RunnerTypeCodex)}); err != nil {
		t.Fatal(err)
	}
	if err := reconciler.RecoverInFlightRuns(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !reconciler.hasTailer(run.ID) {
		t.Fatal("startup recovery should reattach a live session despite unavailable goal evidence")
	}
	defer reconciler.CancelInteractiveTail(run.ID)
	got, err := repos.Runs.Get(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RunStatusRunning || got.EndedAt != nil {
		t.Fatalf("live startup recovery must preserve running state: status=%s ended=%v", got.Status, got.EndedAt)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func TestRecoverInteractiveJoinsTailerBeforeInspectingFreshTranscript(t *testing.T) {
	reconciler, repos := newInteractiveRecoveryReconciler(t, newFakeInteractiveSessions("join-ui"))
	if err := reconciler.runners.Register(&retainedGoalReplayRunner{mocks.NewTranscriptReplayRunner(domain.RunnerTypeCodex)}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	run := createInteractiveRecoveryRun(t, repos, "join-ui", path)
	run.SessionID = "joined-session"
	run.ResolvedConfig.RunnerType, run.ResolvedConfig.Until = domain.RunnerTypeCodex, "finish assignment"
	stamp := run.StartedAt.Add(time.Second).Format(time.RFC3339Nano)
	body := fmt.Sprintf("{\"timestamp\":%q,\"type\":\"session_meta\",\"payload\":{\"id\":%q}}\n{\"timestamp\":%q,\"type\":\"event_msg\",\"payload\":{\"type\":\"thread_goal_updated\",\"goal\":{\"objective\":\"finish assignment\",\"status\":\"complete\"}}}\n", stamp, run.SessionID, stamp)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	run.TranscriptCursor = int64(len(body))
	if err := repos.Runs.Update(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	cancelled, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	reconciler.tailers[run.ID] = func() { once.Do(func() { close(cancelled) }) }
	reconciler.interactiveTailDone[run.ID] = done
	lateWrite := make(chan error, 1)
	go func() {
		<-cancelled
		<-release
		later := body + fmt.Sprintf("{\"timestamp\":%q,\"type\":\"event_msg\",\"payload\":{\"type\":\"user_message\",\"message\":\"new private work\"}}\n", run.StartedAt.Add(2*time.Second).Format(time.RFC3339Nano))
		lateWrite <- os.WriteFile(path, []byte(later), 0o600)
		close(done)
	}()
	returned := make(chan error, 1)
	go func() { _, err := reconciler.RecoverRun(t.Context(), run.ID); returned <- err }()
	<-cancelled
	var early bool
	select {
	case <-returned:
		early = true
	case <-time.After(40 * time.Millisecond):
	}
	close(release)
	if err := <-lateWrite; err != nil {
		t.Fatal(err)
	}
	if early {
		t.Fatal("recovery returned before cancelled tailer finished its final write")
	}
	if err := <-returned; err != nil {
		t.Fatal(err)
	}
	defer reconciler.CancelInteractiveTail(run.ID)
	retained, err := repos.Runs.Get(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retained.Status != domain.RunStatusRunning {
		t.Fatalf("historical completion closed newer transcript work: %s", retained.Status)
	}
}
