package opsbridge

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/opscatalog"
	"swarm-manager/internal/opsrunner"
	"swarm-manager/internal/pathutil"
)

// fakeSimEngine is a deterministic SimulationEngine: it returns a canned
// SimulationResponse per preset (or an error), recording the last call so a test
// can assert the driver forwarded the bound mode and requested preset.
type fakeSimEngine struct {
	byPreset  map[string]operatingmode.SimulationResponse
	err       error
	gotMode   operatingmode.Mode
	gotPreset string
}

func (f *fakeSimEngine) SimulateMode(_ context.Context, mode operatingmode.Mode, preset string) (operatingmode.SimulationResponse, error) {
	f.gotMode = mode
	f.gotPreset = preset
	if f.err != nil {
		return operatingmode.SimulationResponse{}, f.err
	}
	return f.byPreset[preset], nil
}

// fakeContracts is a ContractSource backed by the real seeded operation
// contracts, so disposition resolution uses the true SSOT vocabulary.
type fakeContracts map[agentops.OperationID]agentops.OperationContract

func (f fakeContracts) Contract(id agentops.OperationID, _ string) (opscatalog.LoadedContract, bool) {
	c, ok := f[id]
	return opscatalog.LoadedContract{Contract: c}, ok
}

func seededContracts() fakeContracts {
	m := fakeContracts{}
	for _, c := range agentops.SeedOperationContracts() {
		m[c.ID] = c
	}
	return m
}

func simTrace(rounds ...operatingmode.RoundEnvelope) operatingmode.SimulationResponse {
	steps := make([]operatingmode.SimulationStep, 0, len(rounds))
	for i, r := range rounds {
		steps = append(steps, operatingmode.SimulationStep{Index: i, Phase: r.Phase, Round: r})
	}
	return operatingmode.SimulationResponse{Trace: steps}
}

func TestSimulationDriverMapsFirstRoundToContractOutcome(t *testing.T) {
	handoff := map[string]any{"handoff": map[string]any{"summary": "did a thing"}}
	cases := []struct {
		name        string
		decision    operatingmode.ProgressDecision
		wantOutcome string
		wantDisp    opsrunner.Disposition
	}{
		{"continue", operatingmode.ProgressContinue, "continue", "continue"},
		{"complete", operatingmode.ProgressComplete, "completed", "success"},
		{"blocked", operatingmode.ProgressBlocked, "blocked", "blocked"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			round := completedRound(tc.decision, handoff)
			round.RunID = "simulation-001"
			engine := &fakeSimEngine{byPreset: map[string]operatingmode.SimulationResponse{"p": simTrace(round)}}
			driver := NewSimulationDriver(engine, seededContracts())

			out, err := driver.Drive(context.Background(),
				opsrunner.Prepared{Mode: "backlog-workshop"},
				opsrunner.RunHandle{Operation: agentops.OpWorkshopRound, Preset: "p"})
			if err != nil {
				t.Fatalf("Drive: %v", err)
			}
			if out.Outcome != tc.wantOutcome {
				t.Fatalf("outcome: want %q, got %q", tc.wantOutcome, out.Outcome)
			}
			if out.Disposition != tc.wantDisp {
				t.Fatalf("disposition: want %q, got %q", tc.wantDisp, out.Disposition)
			}
			if out.RunID != "simulation-001" {
				t.Fatalf("run id not propagated, got %q", out.RunID)
			}
			// The resolved handoff plus routing progress is forwarded verbatim.
			var payload map[string]any
			if err := json.Unmarshal(out.Result, &payload); err != nil {
				t.Fatalf("decode result: %v", err)
			}
			if payload["handoff"] == nil {
				t.Fatalf("handoff dropped from result: %s", out.Result)
			}
			// The driver forwarded the bound mode + requested preset to the engine.
			if engine.gotMode != "backlog-workshop" || engine.gotPreset != "p" {
				t.Fatalf("engine call: mode=%q preset=%q", engine.gotMode, engine.gotPreset)
			}
		})
	}
}

func TestSimulationDriverAbstainsOnParkedFirstRound(t *testing.T) {
	engine := &fakeSimEngine{byPreset: map[string]operatingmode.SimulationResponse{
		"p": simTrace(operatingmode.RoundEnvelope{Status: operatingmode.RoundStatusNeedsAttention}),
	}}
	out, err := NewSimulationDriver(engine, seededContracts()).Drive(context.Background(),
		opsrunner.Prepared{Mode: "backlog-workshop"},
		opsrunner.RunHandle{Operation: agentops.OpWorkshopRound, Preset: "p"})
	if err != nil {
		t.Fatalf("Drive: %v", err)
	}
	if out.Outcome != "needs-attention" || out.Disposition != "abstain" {
		t.Fatalf("parked round must abstain to needs-attention, got outcome=%q disp=%q", out.Outcome, out.Disposition)
	}
	if out.Result != nil {
		t.Fatalf("abstain must carry no result, got %s", out.Result)
	}
}

func TestSimulationDriverUsesFirstRoundNotWholeWalk(t *testing.T) {
	// A single operation execution is a single round. Given a walk that continues
	// then completes, the driver reports the FIRST round's outcome (continue), the
	// twin of the one live round a single Invoke would start.
	handoff := map[string]any{"handoff": map[string]any{"summary": "round one"}}
	first := completedRound(operatingmode.ProgressContinue, handoff)
	second := completedRound(operatingmode.ProgressComplete, handoff)
	engine := &fakeSimEngine{byPreset: map[string]operatingmode.SimulationResponse{"happy-path": simTrace(first, second)}}

	out, err := NewSimulationDriver(engine, seededContracts()).Drive(context.Background(),
		opsrunner.Prepared{Mode: "backlog-workshop"},
		opsrunner.RunHandle{Operation: agentops.OpWorkshopRound, Preset: "happy-path"})
	if err != nil {
		t.Fatalf("Drive: %v", err)
	}
	if out.Outcome != "continue" {
		t.Fatalf("want first-round outcome continue, got %q", out.Outcome)
	}
}

func TestSimulationDriverFailClosed(t *testing.T) {
	handoff := map[string]any{"handoff": map[string]any{"summary": "x"}}
	completed := completedRound(operatingmode.ProgressComplete, handoff)

	t.Run("no engine", func(t *testing.T) {
		_, err := (&SimulationDriver{contracts: seededContracts()}).Drive(context.Background(),
			opsrunner.Prepared{Mode: "backlog-workshop"}, opsrunner.RunHandle{Operation: agentops.OpWorkshopRound})
		if err == nil {
			t.Fatal("expected error with no engine")
		}
	})
	t.Run("engine error propagates", func(t *testing.T) {
		engine := &fakeSimEngine{err: errors.New("boom")}
		_, err := NewSimulationDriver(engine, seededContracts()).Drive(context.Background(),
			opsrunner.Prepared{Mode: "backlog-workshop"}, opsrunner.RunHandle{Operation: agentops.OpWorkshopRound, Preset: "p"})
		if err == nil {
			t.Fatal("expected engine error to propagate")
		}
	})
	t.Run("empty trace", func(t *testing.T) {
		engine := &fakeSimEngine{byPreset: map[string]operatingmode.SimulationResponse{"p": {}}}
		_, err := NewSimulationDriver(engine, seededContracts()).Drive(context.Background(),
			opsrunner.Prepared{Mode: "backlog-workshop"}, opsrunner.RunHandle{Operation: agentops.OpWorkshopRound, Preset: "p"})
		if err == nil {
			t.Fatal("expected error on empty trace")
		}
	})
	t.Run("non-terminal first round", func(t *testing.T) {
		engine := &fakeSimEngine{byPreset: map[string]operatingmode.SimulationResponse{
			"p": simTrace(operatingmode.RoundEnvelope{Status: operatingmode.RoundStatusAgentRunning}),
		}}
		_, err := NewSimulationDriver(engine, seededContracts()).Drive(context.Background(),
			opsrunner.Prepared{Mode: "backlog-workshop"}, opsrunner.RunHandle{Operation: agentops.OpWorkshopRound, Preset: "p"})
		if err == nil {
			t.Fatal("expected error: a still-running first round is not a terminal outcome")
		}
	})
	t.Run("unknown operation has no disposition", func(t *testing.T) {
		engine := &fakeSimEngine{byPreset: map[string]operatingmode.SimulationResponse{"p": simTrace(completed)}}
		_, err := NewSimulationDriver(engine, seededContracts()).Drive(context.Background(),
			opsrunner.Prepared{Mode: "backlog-workshop"}, opsrunner.RunHandle{Operation: "not-an-operation", Preset: "p"})
		if err == nil {
			t.Fatal("expected fail-closed on unknown operation")
		}
	})
	t.Run("nil contract source", func(t *testing.T) {
		engine := &fakeSimEngine{byPreset: map[string]operatingmode.SimulationResponse{"p": simTrace(completed)}}
		_, err := NewSimulationDriver(engine, nil).Drive(context.Background(),
			opsrunner.Prepared{Mode: "backlog-workshop"}, opsrunner.RunHandle{Operation: agentops.OpWorkshopRound, Preset: "p"})
		if err == nil {
			t.Fatal("expected error with nil contract source")
		}
	})
}

// TestProductionRunnerConstructsAndSimulateDrives is the slice-0 acceptance: a
// real opsrunner.Runner is constructed from the real catalog + real backlog
// modes with the SimulationDriver as its ExecutionDriver and the LivePreparer as
// BOTH the runner's preparer and the resolver's ModeChecker — no stub, no
// no-op driver — and a simulated Invoke drives the mode's outcome through the
// real backlog-item transition policy.
func TestProductionRunnerConstructsAndSimulateDrives(t *testing.T) {
	scenarioRoot := pathutil.ResolveScenarioRoot("swarm-manager")
	catalog, err := opscatalog.Load(scenarioRoot)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	modeDefs, err := operatingmode.LoadModesFromDir(filepath.Join(scenarioRoot, "modes"))
	if err != nil {
		t.Fatalf("load modes: %v", err)
	}
	defsByID := map[string]operatingmode.Definition{}
	for mode, def := range modeDefs {
		defsByID[string(mode)] = def
	}
	preparer := opsrunner.NewLivePreparer(catalog, defsByID)

	tmp := t.TempDir()
	locator := opsrunner.FSLocator{
		BacklogItemDir: func(kind, name string) (string, error) { return filepath.Join(tmp, kind, name), nil },
		InitiativeDir:  func(name string) (string, error) { return filepath.Join(tmp, "initiatives", name), nil },
		ScanRoots:      []string{tmp},
	}
	repo := opsrunner.NewWorkflowRepo(locator)
	execStore := opsrunner.NewExecutionStore(locator)
	resolver := opsrunner.NewBindingResolver(catalog, opsrunner.NewFSOverrideStore(locator), preparer)
	registry := opsrunner.NewActionRegistry()
	dispatcher := opsrunner.NewDispatcher(registry, repo)

	// One deterministic first-round per preset, so each Invoke exercises a
	// distinct backlog-item transition.
	handoff := map[string]any{"handoff": map[string]any{"summary": "synthesized"}}
	engine := &fakeSimEngine{byPreset: map[string]operatingmode.SimulationResponse{
		"complete-first": simTrace(mustRound(t, operatingmode.ProgressComplete, handoff)),
		"happy-path":     simTrace(mustRound(t, operatingmode.ProgressContinue, handoff)),
		"blocked":        simTrace(mustRound(t, operatingmode.ProgressBlocked, handoff)),
	}}
	driver := NewSimulationDriver(engine, catalog)

	runner, err := opsrunner.New(opsrunner.Config{
		Catalog: catalog, Resolver: resolver, Preparer: preparer,
		Driver: driver, Repo: repo, Executions: execStore, Dispatcher: dispatcher,
	})
	if err != nil {
		t.Fatalf("construct production runner: %v", err)
	}

	cases := []struct {
		preset      string
		wantOutcome string
		wantAction  agentops.ActionName
		wantState   agentops.WorkflowState
	}{
		{"complete-first", "completed", agentops.ActionOpenReview, agentops.WorkflowState("awaiting-decision")},
		{"happy-path", "continue", agentops.ActionCommitWorkshopRound, agentops.WorkflowRunning},
		{"blocked", "blocked", agentops.ActionEscalateNeedsAttention, agentops.WorkflowState("blocked")},
	}
	for _, tc := range cases {
		t.Run(tc.preset, func(t *testing.T) {
			res, err := runner.Invoke(context.Background(), opsrunner.InvokeRequest{
				Target:           opsrunner.TargetRef{Kind: agentops.TargetBacklogItem, ID: "execute/sim-" + tc.preset},
				Operation:        agentops.OpWorkshopRound,
				OperationVersion: "1.0.0",
				Simulate:         true,
				SimulationPreset: tc.preset,
				RequestedBy:      "slice-0-test",
			})
			if err != nil {
				t.Fatalf("Invoke: %v", err)
			}
			if res.Outcome != tc.wantOutcome {
				t.Fatalf("outcome: want %q, got %q", tc.wantOutcome, res.Outcome)
			}
			if res.Action != tc.wantAction {
				t.Fatalf("action: want %q, got %q", tc.wantAction, res.Action)
			}
			if res.WorkflowState != tc.wantState {
				t.Fatalf("state: want %q, got %q", tc.wantState, res.WorkflowState)
			}
		})
	}
}

func mustRound(t *testing.T, decision operatingmode.ProgressDecision, resolved map[string]any) operatingmode.RoundEnvelope {
	t.Helper()
	r := completedRound(decision, resolved)
	r.RunID = "simulation-001"
	return r
}
