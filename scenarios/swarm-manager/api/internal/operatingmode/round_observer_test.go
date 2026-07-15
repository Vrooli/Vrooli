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
