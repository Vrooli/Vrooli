package heartbeat

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// recordingResumer records owner wake calls and can be made to fail.
type recordingResumer struct {
	mu    sync.Mutex
	calls []string
	err   error
	block chan struct{}
}

func (r *recordingResumer) WakePausedRun(_ context.Context, runID, _ string) error {
	r.mu.Lock()
	r.calls = append(r.calls, runID)
	block := r.block
	err := r.err
	r.mu.Unlock()
	if block != nil {
		<-block
	}
	return err
}

func (r *recordingResumer) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func (r *recordingResumer) setErr(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = err
}

// syntheticEligibilitySource replays a fixed list of owner events and then
// reports exhaustion.
type syntheticEligibilitySource struct {
	mu     sync.Mutex
	events []ResetEligibility
	i      int
}

func (s *syntheticEligibilitySource) Next(_ context.Context) (ResetEligibility, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.i >= len(s.events) {
		return ResetEligibility{}, false, nil
	}
	ev := s.events[s.i]
	s.i++
	return ev, true, nil
}

// seedPausedObligation writes a parked run to disk and recovers it so the
// in-memory obligation is classified ObligationPaused.
func seedPausedObligation(t *testing.T, dir, teamID, agentID, runID string) *TeamExecutionContext {
	t.Helper()
	writeQueueFile(t, dir, teamID, persistedTeamQueue{
		TeamID:            teamID,
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: agentID, ProfileKey: "p", RunID: runID},
		},
	})
	client := newMockAgentClient().WithGetRunResponse(runID, &Run{ID: runID, Status: "RUN_STATUS_PARKED"})
	tec := newTeamExecutionContext(teamID, &captureExecutor{}, dir, client)
	tec.Recover(context.Background())
	status := tec.Status()
	if len(status.PausedAgentIDs) != 1 {
		t.Fatalf("expected a paused obligation after recovery, got %+v", status)
	}
	return tec
}

func TestTeamExecutionConsumeResetEligibility_KnownResetResumesOnce(t *testing.T) {
	dir := t.TempDir()
	tec := seedPausedObligation(t, dir, "team-1", "agent-1", "run-parked")
	resumer := &recordingResumer{}
	tec.SetPausedRunResumer(resumer)

	base := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	clock := base
	tec.SetClock(func() time.Time { return clock })

	ev := ResetEligibility{
		EventID:   "reset-1",
		TeamID:    "team-1",
		AgentID:   "agent-1",
		RunID:     "run-parked",
		Known:     true,
		ResumesAt: base.Add(time.Hour),
	}

	ctx := context.Background()

	// The hourly tick stays refused while the obligation is paused, even after
	// the clock passes the known reset: time alone does not re-admit work.
	clock = base.Add(2 * time.Hour)
	if _, err := tec.Enqueue(ctx, "agent-1", "p"); !IsMemberAlreadyQueued(err) {
		t.Fatalf("expected paused obligation to refuse Enqueue, got %v", err)
	}

	// Before the reset time the event is deferred and the owner is not woken.
	clock = base
	if got, err := tec.ConsumeResetEligibility(ctx, ev); err != nil || got != ResetConsumeDeferred {
		t.Fatalf("before reset: got (%q, %v), want deferred", got, err)
	}
	if resumer.count() != 0 {
		t.Fatalf("expected no wake before reset, got %d", resumer.count())
	}

	// At the reset time the owner is woken exactly once.
	clock = base.Add(time.Hour)
	if got, err := tec.ConsumeResetEligibility(ctx, ev); err != nil || got != ResetConsumeResumed {
		t.Fatalf("at reset: got (%q, %v), want resumed", got, err)
	}
	if resumer.count() != 1 {
		t.Fatalf("expected exactly one wake, got %d", resumer.count())
	}
	status := tec.Status()
	if len(status.PausedAgentIDs) != 0 || len(status.RunningAgentIDs) != 1 {
		t.Fatalf("expected running obligation after resume, got %+v", status)
	}

	// A replayed event cannot wake the run a second time: the member is no
	// longer paused, so the event is inert.
	if got, err := tec.ConsumeResetEligibility(ctx, ev); err != nil || got != ResetConsumeNone {
		t.Fatalf("replayed event: got (%q, %v), want none", got, err)
	}
	if resumer.count() != 1 {
		t.Fatalf("replayed event woke the run; calls=%d", resumer.count())
	}
}

func TestTeamExecutionConsumeResetEligibility_UnknownResetIsNeverConsumed(t *testing.T) {
	dir := t.TempDir()
	tec := seedPausedObligation(t, dir, "team-1", "agent-1", "run-parked")
	resumer := &recordingResumer{}
	tec.SetPausedRunResumer(resumer)
	tec.SetClock(func() time.Time { return time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC) })

	ev := ResetEligibility{
		EventID: "reset-unknown",
		TeamID:  "team-1",
		AgentID: "agent-1",
		RunID:   "run-parked",
		Known:   false,
	}
	for i := 0; i < 3; i++ {
		if got, err := tec.ConsumeResetEligibility(context.Background(), ev); err != nil || got != ResetConsumeDeferred {
			t.Fatalf("unknown reset: got (%q, %v), want deferred", got, err)
		}
	}
	if resumer.count() != 0 {
		t.Fatalf("unknown reset must not wake the run, got %d calls", resumer.count())
	}
	if status := tec.Status(); len(status.PausedAgentIDs) != 1 {
		t.Fatalf("unknown reset must retain the paused obligation, got %+v", status)
	}
}

func TestTeamExecutionConsumeResetEligibility_OwnerWakeFailureKeepsPauseForRetry(t *testing.T) {
	dir := t.TempDir()
	tec := seedPausedObligation(t, dir, "team-1", "agent-1", "run-parked")
	resumer := &recordingResumer{err: errors.New("owner unreachable")}
	tec.SetPausedRunResumer(resumer)
	base := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	tec.SetClock(func() time.Time { return base.Add(2 * time.Hour) })

	ev := ResetEligibility{
		EventID:   "reset-1",
		TeamID:    "team-1",
		AgentID:   "agent-1",
		RunID:     "run-parked",
		Known:     true,
		ResumesAt: base,
	}
	if _, err := tec.ConsumeResetEligibility(context.Background(), ev); err == nil {
		t.Fatal("expected owner wake failure to surface")
	}
	if resumer.count() != 1 {
		t.Fatalf("expected one attempted wake, got %d", resumer.count())
	}
	if status := tec.Status(); len(status.PausedAgentIDs) != 1 {
		t.Fatalf("failed wake must retain the paused obligation, got %+v", status)
	}

	// A distinct owner event can retry once the owner is reachable.
	resumer.setErr(nil)
	retry := ev
	retry.EventID = "reset-2"
	if got, err := tec.ConsumeResetEligibility(context.Background(), retry); err != nil || got != ResetConsumeResumed {
		t.Fatalf("retry: got (%q, %v), want resumed", got, err)
	}
	if status := tec.Status(); len(status.PausedAgentIDs) != 0 || len(status.RunningAgentIDs) != 1 {
		t.Fatalf("expected running obligation after retry, got %+v", status)
	}
}

func TestTeamExecutionConsumeResetEligibility_IgnoresNonPausedAndStaleRuns(t *testing.T) {
	dir := t.TempDir()

	// A running (non-parked) obligation is not eligible for reset consumption.
	writeQueueFile(t, dir, "team-1", persistedTeamQueue{
		TeamID:            "team-1",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-active"},
		},
	})
	client := newMockAgentClient().WithGetRunResponse("run-active", &Run{ID: "run-active", Status: "RUN_STATUS_RUNNING"})
	running := newTeamExecutionContext("team-1", &captureExecutor{}, dir, client)
	resumer := &recordingResumer{}
	running.SetPausedRunResumer(resumer)
	running.Recover(context.Background())

	activeEvent := ResetEligibility{EventID: "e1", TeamID: "team-1", AgentID: "agent-1", RunID: "run-active", Known: true}
	if got, err := running.ConsumeResetEligibility(context.Background(), activeEvent); err != nil || got != ResetConsumeNone {
		t.Fatalf("non-paused member: got (%q, %v), want none", got, err)
	}
	if resumer.count() != 0 {
		t.Fatalf("non-paused member must not be woken, got %d", resumer.count())
	}

	// A stale event naming a different run is ignored.
	staleDir := t.TempDir()
	paused := seedPausedObligation(t, staleDir, "team-2", "agent-2", "run-current")
	staleResumer := &recordingResumer{}
	paused.SetPausedRunResumer(staleResumer)
	staleEvent := ResetEligibility{EventID: "e2", TeamID: "team-2", AgentID: "agent-2", RunID: "run-old", Known: true}
	if got, err := paused.ConsumeResetEligibility(context.Background(), staleEvent); err != nil || got != ResetConsumeNone {
		t.Fatalf("stale run: got (%q, %v), want none", got, err)
	}
	if staleResumer.count() != 0 {
		t.Fatalf("stale run event must not wake, got %d", staleResumer.count())
	}
	if status := paused.Status(); len(status.PausedAgentIDs) != 1 {
		t.Fatalf("stale event must retain the paused obligation, got %+v", status)
	}
}

func TestTeamExecutionConsumeResetEligibility_ConcurrentDuplicateEventsWakeOnce(t *testing.T) {
	dir := t.TempDir()
	tec := seedPausedObligation(t, dir, "team-1", "agent-1", "run-parked")
	block := make(chan struct{})
	resumer := &recordingResumer{block: block}
	tec.SetPausedRunResumer(resumer)
	tec.SetClock(func() time.Time { return time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC) })

	ev := ResetEligibility{EventID: "reset-1", TeamID: "team-1", AgentID: "agent-1", RunID: "run-parked", Known: true}

	const callers = 8
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer wg.Done()
			tec.ConsumeResetEligibility(context.Background(), ev)
		}()
	}
	waitFor(t, 2*time.Second, func() bool { return resumer.count() == 1 }, "one in-flight wake")
	close(block)
	wg.Wait()

	if got := resumer.count(); got != 1 {
		t.Fatalf("expected exactly one wake across concurrent duplicates, got %d", got)
	}
	if status := tec.Status(); len(status.RunningAgentIDs) != 1 || len(status.PausedAgentIDs) != 0 {
		t.Fatalf("expected running obligation, got %+v", status)
	}
}

func TestTeamExecutionRunResetEligibility_SyntheticStreamResumesPausedObligation(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-1", persistedTeamQueue{
		TeamID:            "team-1",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-parked"},
		},
	})
	client := newMockAgentClient().WithGetRunResponse("run-parked", &Run{ID: "run-parked", Status: "RUN_STATUS_PARKED"})
	store := NewTeamExecutionStore(nil, &captureExecutor{}, dir, client)

	base := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	resumer := &recordingResumer{}
	store.SetPausedRunResumer(resumer)
	store.SetClock(func() time.Time { return base.Add(time.Hour) })
	store.Recover(context.Background())

	if got := store.Status("team-1"); len(got.PausedAgentIDs) != 1 {
		t.Fatalf("expected paused obligation after recovery, got %+v", got)
	}

	src := &syntheticEligibilitySource{events: []ResetEligibility{{
		EventID:   "reset-1",
		TeamID:    "team-1",
		AgentID:   "agent-1",
		RunID:     "run-parked",
		Known:     true,
		ResumesAt: base,
	}}}
	store.RunResetEligibility(context.Background(), src)

	if resumer.count() != 1 {
		t.Fatalf("expected the synthetic event to wake the run once, got %d", resumer.count())
	}
	if got := store.Status("team-1"); len(got.PausedAgentIDs) != 0 || len(got.RunningAgentIDs) != 1 {
		t.Fatalf("expected running obligation after stream consumption, got %+v", got)
	}
}

func TestTeamExecutionConsumeResetEligibility_StoreRoutesByTeam(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-1", persistedTeamQueue{
		TeamID:            "team-1",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-parked"},
		},
	})
	client := newMockAgentClient().WithGetRunResponse("run-parked", &Run{ID: "run-parked", Status: "RUN_STATUS_PARKED"})
	store := NewTeamExecutionStore(nil, &captureExecutor{}, dir, client)
	resumer := &recordingResumer{}
	store.SetPausedRunResumer(resumer)
	store.SetClock(func() time.Time { return time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC) })
	store.Recover(context.Background())

	// An event for a different team is not applied to team-1.
	if got, err := store.ConsumeResetEligibility(context.Background(), ResetEligibility{
		EventID: "e-other", TeamID: "team-other", AgentID: "agent-1", Known: true,
	}); err != nil || got != ResetConsumeNone {
		t.Fatalf("other team event: got (%q, %v), want none", got, err)
	}
	if resumer.count() != 0 {
		t.Fatalf("other team event must not wake team-1, got %d", resumer.count())
	}

	if got, err := store.ConsumeResetEligibility(context.Background(), ResetEligibility{
		EventID: "e1", TeamID: "team-1", AgentID: "agent-1", RunID: "run-parked", Known: true,
	}); err != nil || got != ResetConsumeResumed {
		t.Fatalf("team-1 event: got (%q, %v), want resumed", got, err)
	}
	if resumer.count() != 1 {
		t.Fatalf("expected team-1 to be woken once, got %d", resumer.count())
	}
}
