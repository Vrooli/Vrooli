package orchestrator

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"test-genie/internal/orchestrator/phasepolicy"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/providerreadiness"
	workspacepkg "test-genie/internal/orchestrator/workspace"
)

type providerReadinessPlan struct {
	Active   []phases.Definition
	Blocked  map[string]providerreadiness.Outcome
	Outcomes []providerreadiness.Outcome
}

func (o *SuiteOrchestrator) checkProviderReadiness(
	ctx context.Context,
	env workspacepkg.Environment,
	defs []phases.Definition,
	logWriter io.Writer,
) providerReadinessPlan {
	active := make([]phases.Definition, 0, len(defs))
	blocked := make(map[string]providerreadiness.Outcome)
	outcomes := make([]providerreadiness.Outcome, 0, len(defs))
	manager := o.readiness
	if manager == nil {
		manager = providerreadiness.NewManager()
	}

	for _, def := range defs {
		policy := def.Policy
		if policy.IsZero() {
			policy = phasepolicy.FromLegacyCatalog(def.Optional, false)
		}
		outcome := manager.Check(ctx, providerreadiness.Input{
			Phase:            def.Name.String(),
			ProviderScenario: def.ProviderScenario,
			TargetScenario:   env.ScenarioName,
			TargetPath:       env.ScenarioDir,
			Policy:           policy,
			Timeout:          def.Timeout,
		}, logWriter)
		if def.ProviderScenario != "" || outcome.Status != providerreadiness.OutcomeReady {
			outcomes = append(outcomes, outcome)
		}
		if outcome.BlocksExecution() {
			blocked[def.Name.Key()] = outcome
			continue
		}
		active = append(active, def)
	}

	return providerReadinessPlan{Active: active, Blocked: blocked, Outcomes: outcomes}
}

func (o *SuiteOrchestrator) newProviderReadinessPhaseResult(def phases.Definition, runLogDir string, outcome providerreadiness.Outcome) PhaseExecutionResult {
	logPath := phaseLogPath(runLogDir, def.Name)
	obs := providerreadiness.ResultObservation(outcome)
	if f, err := os.Create(logPath); err == nil {
		fmt.Fprintf(f, "[PROVIDER_READINESS] %s\n", obs.String())
		if remediation := providerreadiness.Remediation(outcome); strings.TrimSpace(remediation) != "" {
			fmt.Fprintf(f, "Remediation: %s\n", remediation)
		}
		_ = f.Close()
	}
	displayLogPath := logPath
	if rel, err := filepath.Rel(o.projectRoot, logPath); err == nil {
		displayLogPath = rel
	}
	status := phaseStatusProviderUnavailable
	if outcome.SkipsWithoutFailure() {
		status = phaseStatusSkipped
	}
	return PhaseExecutionResult{
		Name:            def.Name.String(),
		Status:          status,
		DurationSeconds: 0,
		LogPath:         displayLogPath,
		Error:           outcome.ErrorString(),
		Classification:  providerreadiness.FailureClass(outcome),
		Remediation:     providerreadiness.Remediation(outcome),
		Observations:    []phases.Observation{obs},
	}
}
