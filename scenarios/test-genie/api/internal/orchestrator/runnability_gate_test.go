package orchestrator

import (
	"context"
	"io"
	"testing"

	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/runnability"
	workspacepkg "test-genie/internal/orchestrator/workspace"
)

func passingRunner(context.Context, workspacepkg.Environment, io.Writer) phases.RunReport {
	return phases.RunReport{Observations: []phases.Observation{phases.NewSuccessObservation("ok")}}
}

func staticDef(name phases.Name) phases.Definition {
	return phases.Definition{
		Name:         name,
		Runner:       passingRunner,
		Timeout:      0,
		Capabilities: runnability.PhaseCapabilities{Phase: name.String()},
	}
}

func surfaceDef(name phases.Name) phases.Definition {
	return phases.Definition{
		Name:         name,
		Runner:       passingRunner,
		Timeout:      0,
		Capabilities: runnability.PhaseCapabilities{Phase: name.String(), NeedsUI: true},
	}
}

func TestComputeSuiteVerdict(t *testing.T) {
	defs := []phases.Definition{
		{Name: phases.Structure},
		{Name: phases.Smoke, Optional: true},
		{Name: phases.Integration},
	}
	cases := []struct {
		name    string
		results []PhaseExecutionResult
		want    string
	}{
		{
			name: "all passed → PASS",
			results: []PhaseExecutionResult{
				{Name: "structure", Status: phaseStatusPassed},
				{Name: "integration", Status: phaseStatusPassed},
			},
			want: SuiteVerdictPass,
		},
		{
			name: "any failed → FAIL",
			results: []PhaseExecutionResult{
				{Name: "structure", Status: phaseStatusPassed},
				{Name: "integration", Status: phaseStatusFailed},
			},
			want: SuiteVerdictFail,
		},
		{
			name: "non-optional skip → PARTIAL",
			results: []PhaseExecutionResult{
				{Name: "structure", Status: phaseStatusPassed},
				{Name: "integration", Status: phaseStatusSkipped},
			},
			want: SuiteVerdictPartial,
		},
		{
			name: "optional skip stays PASS",
			results: []PhaseExecutionResult{
				{Name: "structure", Status: phaseStatusPassed},
				{Name: "smoke", Status: phaseStatusSkipped},
			},
			want: SuiteVerdictPass,
		},
		{
			name: "failure dominates a skip",
			results: []PhaseExecutionResult{
				{Name: "integration", Status: phaseStatusSkipped},
				{Name: "structure", Status: phaseStatusFailed},
			},
			want: SuiteVerdictFail,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := computeSuiteVerdict(tc.results, defs); got != tc.want {
				t.Fatalf("computeSuiteVerdict = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRunSelectedPhasesSkipsUnrunnableSurfacePhaseOnSelf is the self-host gate
// at the phase level: a UI-needing phase whose surface is not live on a
// self-target is skipped (not run, not failed), while a static phase runs.
func TestRunSelectedPhasesSkipsUnrunnableSurfacePhaseOnSelf(t *testing.T) {
	o := &SuiteOrchestrator{projectRoot: t.TempDir(), phaseTimeout: phases.DefaultTimeout}
	runLogDir := t.TempDir()

	defs := []phases.Definition{
		staticDef(phases.Structure),
		surfaceDef(phases.Smoke),
	}
	// Self-target, no live surfaces: the smoke phase cannot run without a start
	// that would SIGTERM the suite, so it must skip.
	rc := runnability.RunContext{TargetIsSelf: true, LiveSurfaces: runnability.Surfaces{}}

	results, anyFailure := o.runSelectedPhasesWithEvents(context.Background(), workspacepkg.Environment{}, rc, runLogDir, defs, false, nil, nil)
	if anyFailure {
		t.Fatal("a runnability skip must not be reported as a failure")
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 phase results, got %d", len(results))
	}
	byName := map[string]PhaseExecutionResult{}
	for _, r := range results {
		byName[r.Name] = r
	}
	if byName["structure"].Status != phaseStatusPassed {
		t.Errorf("structure should pass, got %q", byName["structure"].Status)
	}
	smoke := byName["smoke"]
	if smoke.Status != phaseStatusSkipped {
		t.Fatalf("smoke should be skipped on self with no live UI, got %q", smoke.Status)
	}
	if smoke.RunnabilityVerdict != "skip" {
		t.Errorf("smoke RunnabilityVerdict = %q, want skip", smoke.RunnabilityVerdict)
	}
	if smoke.RunnabilityReason == "" {
		t.Error("skipped smoke phase should carry a runnability reason")
	}
}

// TestRunSelectedPhasesRunsSurfacePhaseWhenLive confirms the same phase runs
// when its surface is live (the live-self reuse path).
func TestRunSelectedPhasesRunsSurfacePhaseWhenLive(t *testing.T) {
	o := &SuiteOrchestrator{projectRoot: t.TempDir(), phaseTimeout: phases.DefaultTimeout}
	runLogDir := t.TempDir()

	defs := []phases.Definition{surfaceDef(phases.Smoke)}
	rc := runnability.RunContext{TargetIsSelf: true, LiveSurfaces: runnability.Surfaces{UI: true}}

	results, anyFailure := o.runSelectedPhasesWithEvents(context.Background(), workspacepkg.Environment{}, rc, runLogDir, defs, false, nil, nil)
	if anyFailure {
		t.Fatal("unexpected failure")
	}
	if len(results) != 1 || results[0].Status != phaseStatusPassed {
		t.Fatalf("smoke should run+pass when UI is live, got %+v", results)
	}
}
