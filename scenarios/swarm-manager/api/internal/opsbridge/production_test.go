package opsbridge

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/opscatalog"
	"swarm-manager/internal/opsrunner"
	"swarm-manager/internal/pathutil"
)

// fakePhaseEngine is a deterministic live start seam: it returns a canned run
// association without spawning an agent, so the production live path can be
// driven end to end in a test.
type fakePhaseEngine struct {
	runID   string
	gotReq  operatingmode.StartTargetPhaseRequest
	started bool
}

func (f *fakePhaseEngine) StartTargetPhase(_ context.Context, req operatingmode.StartTargetPhaseRequest) (operatingmode.RoundEnvelope, error) {
	f.gotReq = req
	f.started = true
	return operatingmode.RoundEnvelope{RunID: f.runID, GeneratedAt: "2026-07-14T00:00:00Z"}, nil
}

// noopRefresher is a no-op RunRefresher (the simulation path never calls it).
type noopRefresher struct{}

func (noopRefresher) RefreshRunByID(context.Context, string) (operatingmode.RoundEnvelope, bool, error) {
	return operatingmode.RoundEnvelope{}, false, nil
}

// enrichedWorkshopRound is the resolved declared output of a workshop-round: the
// decision-B enriched result carrying everything the round file needs.
func enrichedWorkshopRound() map[string]any {
	return map[string]any{
		"handoff":   map[string]any{"summary": "synthesized the workshop"},
		"decisions": []any{map[string]any{"id": "d1", "topic": "storage", "text": "pick a store", "options": []any{map[string]any{"key": "A", "label": "sqlite", "rationale": "simple", "recommended": true}}}},
		"self_assessment": map[string]any{
			"problem_clarity": 3, "scope_defined": 2, "approach_solid": 2, "testable": 3, "risk_awareness": 1,
		},
	}
}

// buildTestRunner assembles the production runner exactly as
// registerBacklogOperationsRunner does — real catalog, real modes, real backlog
// ops handlers over a temp-rooted FileStore, the live preparer as both preparer
// and mode-checker — with deterministic operating-mode seams.
func buildTestRunner(t *testing.T, phase *fakePhaseEngine, sim *fakeSimEngine) (*BacklogRunner, *backlog.FileStore) {
	t.Helper()
	scenarioRoot := pathutil.ResolveScenarioRoot("swarm-manager")
	catalog, err := opscatalog.Load(scenarioRoot)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	modeDefs, err := operatingmode.LoadModesFromDir(filepath.Join(scenarioRoot, "modes"))
	if err != nil {
		t.Fatalf("load modes: %v", err)
	}

	tmp := t.TempDir()
	store := backlog.NewFileStore(tmp)
	locator := opsrunner.FSLocator{
		BacklogItemDir: func(kind, name string) (string, error) {
			return store.ItemDir(backlog.BacklogKind(kind), name), nil
		},
		InitiativeDir: func(name string) (string, error) { return filepath.Join(tmp, "initiatives", name), nil },
		ScanRoots:     []string{tmp},
	}
	registry := opsrunner.NewActionRegistry()
	backlog.RegisterOpsHandlers(registry, backlog.OpsHandlerDeps{Store: store})

	built, err := BuildBacklogRunner(BacklogRunnerConfig{
		Catalog:     catalog,
		ModeDefs:    modeDefs,
		PhaseEngine: phase,
		SimEngine:   sim,
		Refresher:   noopRefresher{},
		Locator:     locator,
		Registry:    registry,
		RequestedBy: "production-test",
	})
	if err != nil {
		t.Fatalf("build backlog runner: %v", err)
	}
	return built, store
}

// TestProductionRunnerSimulateWritesHandlerSideEffects proves the whole
// production shape assembled by BuildBacklogRunner: a simulated workshop-round
// Invoke drives the real backlog-item policy AND fires the real domain handler,
// materializing the artifact on the fixture item dir (round file for continue,
// review-open for completed).
func TestProductionRunnerSimulateWritesHandlerSideEffects(t *testing.T) {
	sim := &fakeSimEngine{byPreset: map[string]operatingmode.SimulationResponse{
		"continue": simTrace(mustRound(t, operatingmode.ProgressContinue, enrichedWorkshopRound())),
		"complete": simTrace(mustRound(t, operatingmode.ProgressComplete, enrichedWorkshopRound())),
	}}
	built, store := buildTestRunner(t, &fakePhaseEngine{}, sim)

	// continue -> commit-workshop-round writes round-001.json + provenance.
	contRes, err := built.Runner.Invoke(context.Background(), opsrunner.InvokeRequest{
		Target:           opsrunner.TargetRef{Kind: agentops.TargetBacklogItem, ID: "execute/cont"},
		Operation:        agentops.OpWorkshopRound,
		OperationVersion: "1.0.0",
		Simulate:         true,
		SimulationPreset: "continue",
	})
	if err != nil {
		t.Fatalf("invoke continue: %v", err)
	}
	if contRes.Action != agentops.ActionCommitWorkshopRound {
		t.Fatalf("continue action: want commit-workshop-round, got %q", contRes.Action)
	}
	roundPath := filepath.Join(store.ItemDir(backlog.BacklogKind("execute"), "cont"), "workshop", "round-001.json")
	raw, err := os.ReadFile(roundPath)
	if err != nil {
		t.Fatalf("round file not written: %v", err)
	}
	var round struct {
		RoundNum  int            `json:"round"`
		Readiness map[string]int `json:"readiness"`
		Items     []struct {
			Type    string `json:"type"`
			Options []struct {
				Key string `json:"key"`
			} `json:"options"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &round); err != nil {
		t.Fatalf("round file invalid: %v", err)
	}
	if round.RoundNum != 1 {
		t.Fatalf("round number: want 1, got %d", round.RoundNum)
	}
	if round.Readiness["problem_clarity"] != 3 || round.Readiness["risk_awareness"] != 1 {
		t.Fatalf("readiness not mapped from self_assessment: %+v", round.Readiness)
	}
	if len(round.Items) != 1 || round.Items[0].Type != "decision" || len(round.Items[0].Options) != 1 {
		t.Fatalf("decisions not mapped to items: %+v", round.Items)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(roundPath), "provenance-001.json")); err != nil {
		t.Fatalf("provenance sidecar not written: %v", err)
	}

	// complete -> open-review writes the workshop review-open artifact.
	compRes, err := built.Runner.Invoke(context.Background(), opsrunner.InvokeRequest{
		Target:           opsrunner.TargetRef{Kind: agentops.TargetBacklogItem, ID: "execute/done"},
		Operation:        agentops.OpWorkshopRound,
		OperationVersion: "1.0.0",
		Simulate:         true,
		SimulationPreset: "complete",
	})
	if err != nil {
		t.Fatalf("invoke complete: %v", err)
	}
	if compRes.Action != agentops.ActionOpenReview {
		t.Fatalf("complete action: want open-review, got %q", compRes.Action)
	}
	reviewPath := filepath.Join(store.ItemDir(backlog.BacklogKind("execute"), "done"), "workshop", "review-open.json")
	if _, err := os.Stat(reviewPath); err != nil {
		t.Fatalf("review-open artifact not written: %v", err)
	}
}

// TestProductionRunnerLiveStartThenObserveCommits proves the full async live
// lifecycle the wiring installs: a non-simulated Invoke starts a round through
// the engine run-starter (recording the run id), and delivering that round to the
// wired completion Observer routes it into CommitResult, which fires the domain
// handler — with no agent spawn and no direct runner reference from the engine.
func TestProductionRunnerLiveStartThenObserveCommits(t *testing.T) {
	phase := &fakePhaseEngine{runID: "run-live-1"}
	built, store := buildTestRunner(t, phase, &fakeSimEngine{})

	start, err := built.Runner.Invoke(context.Background(), opsrunner.InvokeRequest{
		Target:           opsrunner.TargetRef{Kind: agentops.TargetBacklogItem, ID: "execute/live"},
		Operation:        agentops.OpWorkshopRound,
		OperationVersion: "1.0.0",
		// Simulate defaults false: the live start seam is exercised.
	})
	if err != nil {
		t.Fatalf("live invoke: %v", err)
	}
	if !phase.started {
		t.Fatal("live start did not reach the phase engine")
	}
	if start.StartHandle == nil || start.StartHandle.RunID != "run-live-1" {
		t.Fatalf("live invoke did not return the run association: %+v", start.StartHandle)
	}

	// Deliver the terminal round to the wired observer, keyed to the live run id.
	round := completedRound(operatingmode.ProgressComplete, enrichedWorkshopRound())
	round.ScopeKind = string(agentops.TargetBacklogItem)
	round.ScopeID = "execute/live"
	round.RunID = "run-live-1"
	built.Observer(context.Background(), round)

	reviewPath := filepath.Join(store.ItemDir(backlog.BacklogKind("execute"), "live"), "workshop", "review-open.json")
	if _, err := os.Stat(reviewPath); err != nil {
		t.Fatalf("completion bridge did not fire the domain handler (review-open missing): %v", err)
	}
}
