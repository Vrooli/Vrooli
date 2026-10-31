package orchestration

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"

	"github.com/google/uuid"
)

type failNextRunStatusEventStore struct {
	event.Store
	err       error
	remaining atomic.Int32
}

func (s *failNextRunStatusEventStore) Append(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error {
	for _, evt := range events {
		if evt != nil && evt.EventType == domain.EventTypeStatus && s.remaining.CompareAndSwap(1, 0) {
			return s.err
		}
	}
	return s.Store.Append(ctx, runID, events...)
}

func parkInteractiveRunForWake(t *testing.T) (*recordingSessions, *Orchestrator, *domain.Run) {
	t.Helper()

	runs, sessions, svc, run := setupNativeGoalContinuation(t, "finish the exact parked task", false)
	svc.runStateRoot = t.TempDir()
	run.Status = domain.RunStatusRunning
	run.Phase = domain.RunPhaseExecuting
	run.EndedAt = nil
	if err := runs.Update(context.Background(), run); err != nil {
		t.Fatalf("mark interactive run running before park: %v", err)
	}
	parked, err := svc.ParkRun(context.Background(), ParkRunInput{
		RunID:    run.ID,
		Producer: "test-genie",
		Key:      "interactive-wake",
	})
	if err != nil {
		t.Fatalf("ParkRun: %v", err)
	}
	if parked.Status != domain.RunStatusParked || parked.AwaitHandle == nil {
		t.Fatalf("parked interactive run = status %s, handle %+v", parked.Status, parked.AwaitHandle)
	}
	return sessions, svc, parked
}

func wakeCodecContinueCounter(t *testing.T, svc *Orchestrator) *atomic.Int32 {
	t.Helper()
	selected, err := svc.runners.Get(domain.RunnerTypeClaudeCode)
	if err != nil {
		t.Fatalf("get test runner: %v", err)
	}
	wrapped, ok := selected.(*nativeObjectiveReplayRunner)
	if !ok {
		t.Fatalf("test runner type = %T, want native objective wrapper", selected)
	}
	wrapped.capabilities.SupportsContinuation = true
	replay, ok := wrapped.Runner.(*mocks.TranscriptReplayRunner)
	if !ok {
		t.Fatalf("wrapped runner type = %T, want transcript replay runner", wrapped.Runner)
	}
	calls := &atomic.Int32{}
	replay.ContinueFunc = func(context.Context, runner.ContinueRequest) (*runner.ExecuteResult, error) {
		calls.Add(1)
		return &runner.ExecuteResult{Success: true}, nil
	}
	return calls
}

func TestWakeRunContinuesParkedInteractiveSessionInPlaceAndReplayIsNoOp(t *testing.T) {
	ctx := context.Background()
	sessions, svc, parked := parkInteractiveRunForWake(t)
	codecContinues := wakeCodecContinueCounter(t, svc)

	// A parked run can still have its prior interactive tail registered. Wake
	// must drain that exact old driver before attaching the next turn's driver.
	oldCtx, oldDriver := svc.interactiveDrivers.register(context.Background(), parked.ID)
	oldDriverDone := make(chan struct{})
	go func() {
		<-oldCtx.Done()
		svc.interactiveDrivers.finish(parked.ID, oldDriver)
		close(oldDriverDone)
	}()

	const result = "the exact producer returned this result"
	woken, err := svc.WakeRun(ctx, WakeRunInput{RunID: parked.ID, Result: result})
	if err != nil {
		t.Fatalf("WakeRun: %v", err)
	}
	if woken.Status != domain.RunStatusRunning || woken.AwaitHandle != nil {
		t.Fatalf("woken interactive run = status %s handle %+v, want running with no await handle", woken.Status, woken.AwaitHandle)
	}
	if woken.LastAwaitKey != "test-genie:interactive-wake" || woken.LastAwaitResult != result || woken.LastAwaitResolvedAt == nil {
		t.Fatalf("wake bookkeeping = key %q result %q resolved %v", woken.LastAwaitKey, woken.LastAwaitResult, woken.LastAwaitResolvedAt)
	}
	if woken.LastWakeSeq != parked.TranscriptLastSeq {
		t.Fatalf("LastWakeSeq=%d, want exact parked transcript sequence %d", woken.LastWakeSeq, parked.TranscriptLastSeq)
	}
	select {
	case <-oldDriverDone:
	case <-time.After(time.Second):
		t.Fatal("prior parked interactive driver was not cancelled and drained")
	}

	sessions.mu.Lock()
	promptText := append([]string(nil), sessions.promptText...)
	sessionIDs := append([]string(nil), sessions.sessionIDs...)
	promptSources := append([]string(nil), sessions.promptSrc...)
	sessions.mu.Unlock()
	if len(promptText) != 1 || !strings.Contains(promptText[0], result) {
		t.Fatalf("interactive wake prompts = %q, want one formatted prompt containing the exact result", promptText)
	}
	if len(sessionIDs) != 1 || sessionIDs[0] != parked.WebConsoleSessionID {
		t.Fatalf("wake session IDs = %q, want retained session %q", sessionIDs, parked.WebConsoleSessionID)
	}
	if len(promptSources) != 1 || promptSources[0] != interactiveRunSource(parked.ID) {
		t.Fatalf("wake prompt sources = %q, want exact run attribution %q", promptSources, interactiveRunSource(parked.ID))
	}
	if codecContinues.Load() != 0 {
		t.Fatalf("parked interactive wake used codec Continue %d times", codecContinues.Load())
	}

	// Replaying the same notification sees the durable running state and cannot
	// type a second prompt into the retained session.
	if _, err := svc.WakeRun(ctx, WakeRunInput{RunID: parked.ID, Result: "duplicate must be ignored"}); err != nil {
		t.Fatalf("duplicate WakeRun: %v", err)
	}
	sessions.mu.Lock()
	defer sessions.mu.Unlock()
	if len(sessions.promptText) != 1 {
		t.Fatalf("duplicate wake typed another prompt: %q", sessions.promptText)
	}
}

func TestWakeRunInteractiveContinuesCommittedWakeWhenStatusEventAppendFails(t *testing.T) {
	ctx := context.Background()
	sessions, svc, parked := parkInteractiveRunForWake(t)
	codecContinues := wakeCodecContinueCounter(t, svc)
	eventErr := errors.New("interactive wake status event append unavailable")
	failingStore := &failNextRunStatusEventStore{Store: svc.events, err: eventErr}
	failingStore.remaining.Store(1)
	svc.events = failingStore

	woken, err := svc.WakeRun(ctx, WakeRunInput{RunID: parked.ID, Result: "reread current effort feedback"})
	if !errors.Is(err, eventErr) {
		t.Fatalf("WakeRun error = %v, want committed status-event error", err)
	}
	if woken == nil || woken.Status != domain.RunStatusRunning || woken.AwaitHandle != nil {
		t.Fatalf("WakeRun did not return the committed running owner: %+v", woken)
	}
	if codecContinues.Load() != 0 {
		t.Fatalf("interactive status event failure incorrectly fell through to codec Continue: %d", codecContinues.Load())
	}
	sessions.mu.Lock()
	got := append([]string(nil), sessions.promptText...)
	sessions.mu.Unlock()
	if len(got) != 1 || !strings.Contains(got[0], "reread current effort feedback") {
		t.Fatalf("committed interactive wake did not submit one retained-session prompt: %q", got)
	}
	svc.interactiveDrivers.cancelAndWait(parked.ID)
}

func TestWakeRunInteractiveMissingSessionRetainsParkedAwait(t *testing.T) {
	ctx := context.Background()
	sessions, svc, parked := parkInteractiveRunForWake(t)
	codecContinues := wakeCodecContinueCounter(t, svc)
	sessions.getNotFound = true

	if _, err := svc.WakeRun(ctx, WakeRunInput{RunID: parked.ID, Result: "must not dispatch"}); err == nil || !domain.IsPreEffectRefusal(err) {
		t.Fatalf("missing interactive session error = %v, want pre-effect refusal", err)
	}
	stored, err := svc.GetStoredRun(ctx, parked.ID)
	if err != nil {
		t.Fatalf("read parked run after missing session: %v", err)
	}
	if stored.Status != domain.RunStatusParked || stored.AwaitHandle == nil || stored.AwaitHandle.Key != parked.AwaitHandle.Key {
		t.Fatalf("missing-session wake changed parked owner: status=%s handle=%+v", stored.Status, stored.AwaitHandle)
	}
	if stored.LastAwaitResolvedAt != nil || stored.LastAwaitResult != "" {
		t.Fatalf("missing-session wake persisted a resolution: key=%q result=%q at=%v", stored.LastAwaitKey, stored.LastAwaitResult, stored.LastAwaitResolvedAt)
	}
	if codecContinues.Load() != 0 {
		t.Fatalf("missing-session wake used codec Continue %d times", codecContinues.Load())
	}
	if got := sessions.callLog(); len(got) != 0 {
		t.Fatalf("missing-session wake reached prompt transport: %v", got)
	}
}

func TestWakeRunCodexRefusesMissingRunScopedTailContextBeforeEffects(t *testing.T) {
	ctx := context.Background()
	sessions, svc, parked := parkInteractiveRunForWake(t)
	parked.ResolvedConfig.RunnerType = domain.RunnerTypeCodex
	if err := svc.runs.Update(ctx, parked); err != nil {
		t.Fatalf("mark parked run as Codex: %v", err)
	}
	// Model a failed run-state-root resolution. The prior driver must remain
	// untouched because WakeRun cannot safely attach a Codex transcript tail.
	svc.runStateRoot = ""
	oldCtx, oldDriver := svc.interactiveDrivers.register(context.Background(), parked.ID)
	defer svc.interactiveDrivers.finish(parked.ID, oldDriver)

	if _, err := svc.WakeRun(ctx, WakeRunInput{RunID: parked.ID, Result: "must remain parked"}); err == nil || !domain.IsPreEffectRefusal(err) || !strings.Contains(err.Error(), "run-scoped transcript directory") {
		t.Fatalf("unscoped Codex wake error = %v, want a pre-effect scope refusal", err)
	}
	stored, err := svc.GetStoredRun(ctx, parked.ID)
	if err != nil {
		t.Fatalf("read retained run: %v", err)
	}
	if stored.Status != domain.RunStatusParked || stored.AwaitHandle == nil || stored.LastAwaitResolvedAt != nil || stored.AwaitHandle.Key != parked.AwaitHandle.Key {
		t.Fatalf("unscoped wake changed retained owner: status=%s handle=%+v resolved=%v", stored.Status, stored.AwaitHandle, stored.LastAwaitResolvedAt)
	}
	select {
	case <-oldCtx.Done():
		t.Fatal("unscoped wake cancelled the prior interactive driver before refusing")
	default:
	}
	if got := sessions.callLog(); len(got) != 0 {
		t.Fatalf("unscoped wake reached session effects: %v", got)
	}
}

func TestWakeRunInteractiveStillFencesDifferentCurrentTaskOwner(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	sessions := newRecordingSessions()
	svc := New(repos.Profiles, repos.Tasks, repos.Runs,
		WithInteractiveSessions(sessions), WithEvents(eventStore), WithRunners(newNativeObjectiveRegistry(t)),
		WithRunStateRoot(t.TempDir()), newCurrentModelPolicyFixtureOption(t))
	task := interactiveTestTask(t, svc)
	owner := &domain.Run{
		ID: uuid.New(), TaskID: task.ID, Tag: "current-interactive-owner",
		RunMode: domain.RunModeInPlace, Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued,
		ApprovalState: domain.ApprovalStateNone,
	}
	if err := repos.Runs.CreateTaskAttempt(ctx, task, task, owner, nil); err != nil {
		t.Fatalf("create exact current Task attempt: %v", err)
	}
	old := time.Now().Add(-time.Minute)
	runID := uuid.New()
	parked := &domain.Run{
		ID: runID, TaskID: task.ID, Tag: runID.String(), RunMode: domain.RunModeInPlace,
		ExecutionMode: domain.ExecutionModeInteractive, WebConsoleSessionID: "wc-wrong-owner",
		SessionID: "provider-wrong-owner", Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting,
		StartedAt: &old, LastHeartbeat: &old, ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
		ApprovalState: domain.ApprovalStateNone, CreatedAt: old, UpdatedAt: old,
	}
	if err := repos.Runs.Create(ctx, parked); err != nil {
		t.Fatalf("create named interactive run: %v", err)
	}
	if _, err := svc.ParkRun(ctx, ParkRunInput{RunID: parked.ID, Producer: "test-genie", Key: "interactive-wrong-owner"}); err != nil {
		t.Fatalf("ParkRun: %v", err)
	}
	codecContinues := wakeCodecContinueCounter(t, svc)

	if _, err := svc.WakeRun(ctx, WakeRunInput{RunID: parked.ID, Result: "must remain with the current owner"}); err == nil || !strings.Contains(err.Error(), "requested Run no longer owns the Task") {
		t.Fatalf("wrong-owner interactive wake error = %v", err)
	}
	stored, err := svc.GetStoredRun(ctx, parked.ID)
	if err != nil || stored.Status != domain.RunStatusParked || stored.AwaitHandle == nil {
		t.Fatalf("wrong-owner wake changed parked run: %+v err=%v", stored, err)
	}
	if codecContinues.Load() != 0 || len(sessions.callLog()) != 0 {
		t.Fatalf("wrong-owner wake reached a runner or session: codec=%d calls=%v", codecContinues.Load(), sessions.callLog())
	}
}
