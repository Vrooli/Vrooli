package operatingmode

import (
	"context"
	"testing"

	"swarm-manager/internal/agentmanager"
)

// TestRefreshRoundNotifiesObserverOnTerminal proves the terminal-round observer
// (the operation-runner completion seam) fires exactly once when a round
// transitions to a terminal status, carrying the persisted round.
func TestRefreshRoundNotifiesObserverOnTerminal(t *testing.T) {
	root := t.TempDir()
	final := `{"operating_mode_result":{"handoff":{"summary":"done"},"artifacts":[{"path":"modes/holistic-loop/findings.md","content":"# Findings"}]}}`
	agent := &fakeAgent{states: map[string]agentmanager.RunState{}}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{InitiativeName: "init-a", Phase: "investigate"})
	if err != nil {
		t.Fatalf("StartPhase: %v", err)
	}

	var observed []RoundEnvelope
	svc.SetRoundObserver(func(_ context.Context, r RoundEnvelope) { observed = append(observed, r) })

	agent.states[round.RunID] = agentmanager.RunState{
		RunID: round.RunID, Status: "complete", Summary: final, FinishedAt: "2026-04-30T12:05:00Z",
	}
	refreshed, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound: %v", err)
	}
	if refreshed.Status != RoundStatusCompleted {
		t.Fatalf("want completed round, got %q (%s)", refreshed.Status, refreshed.Error)
	}
	if len(observed) != 1 {
		t.Fatalf("observer should fire exactly once on terminal, fired %d", len(observed))
	}
	if observed[0].RunID != round.RunID || observed[0].Status != RoundStatusCompleted {
		t.Fatalf("observer received wrong round: %+v", observed[0])
	}
}

// TestRefreshRoundDoesNotNotifyObserverWhileRunning proves a still-running round
// does not fire the observer — completions are delivered once, not on every poll.
func TestRefreshRoundDoesNotNotifyObserverWhileRunning(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{states: map[string]agentmanager.RunState{}}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{InitiativeName: "init-a", Phase: "investigate"})
	if err != nil {
		t.Fatalf("StartPhase: %v", err)
	}

	fired := 0
	svc.SetRoundObserver(func(context.Context, RoundEnvelope) { fired++ })

	agent.states[round.RunID] = agentmanager.RunState{RunID: round.RunID, Status: "running"}
	if _, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round); err != nil {
		t.Fatalf("RefreshRound: %v", err)
	}
	if fired != 0 {
		t.Fatalf("observer must not fire for a still-running round, fired %d", fired)
	}
}

// TestRefreshRoundReNotifiesObserverForTerminalRound proves the recovery
// contract the completion router documents: while the owning operation record is
// still running, the refresh driver keeps re-observing the round, so a lost
// delivery (or a cancel that arrived through a raw stop surface) is recoverable.
// A refresh of an ALREADY-terminal round must therefore re-fire the observer —
// the downstream CommitResult/CancelExecution are idempotent.
func TestRefreshRoundReNotifiesObserverForTerminalRound(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{states: map[string]agentmanager.RunState{}}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{InitiativeName: "init-a", Phase: "investigate"})
	if err != nil {
		t.Fatalf("StartPhase: %v", err)
	}

	var observed []RoundEnvelope
	svc.SetRoundObserver(func(_ context.Context, r RoundEnvelope) { observed = append(observed, r) })

	// The run is stopped externally; the first refresh observes the cancel.
	agent.states[round.RunID] = agentmanager.RunState{RunID: round.RunID, Status: "cancelled", FinishedAt: "2026-04-30T12:05:00Z"}
	if _, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round); err != nil {
		t.Fatalf("RefreshRound (transition): %v", err)
	}
	if len(observed) != 1 || observed[0].Status != RoundStatusCanceled {
		t.Fatalf("first refresh should observe the canceled round once, got %d (%+v)", len(observed), observed)
	}

	// A later refresh (the driver re-polling because the operation record is
	// still running) must re-fire the observer for the terminal round instead of
	// returning silently.
	if _, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round); err != nil {
		t.Fatalf("RefreshRound (re-observe): %v", err)
	}
	if len(observed) != 2 || observed[1].Status != RoundStatusCanceled {
		t.Fatalf("re-refresh of a terminal round must re-fire the observer, got %d (%+v)", len(observed), observed)
	}
}
