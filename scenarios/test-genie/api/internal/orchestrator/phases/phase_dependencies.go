package phases

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"test-genie/internal/dependencies"
	"test-genie/internal/dependencies/resources"
	"test-genie/internal/orchestrator/workspace"
)

// runDependenciesPhase validates runtime/tool requirements using the dependencies package.
// This includes baseline commands, language runtimes, package managers, and resources.
func runDependenciesPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{Err: err, FailureClassification: FailureClassSystem}
	}

	// Build configuration
	config := dependencies.Config{
		ScenarioDir:   env.ScenarioDir,
		ScenarioName:  env.ScenarioName,
		AppRoot:       env.AppRoot,
		CommandLookup: commandLookup,
	}

	// Create runner with optional resource checker
	opts := []dependencies.Option{
		dependencies.WithLogger(logWriter),
	}

	// Try to set up resource health checking if vrooli CLI is available
	resourceChecker := createResourceChecker(ctx, env, logWriter)
	if resourceChecker != nil {
		opts = append(opts, dependencies.WithResourceChecker(resourceChecker))
	}

	runner := dependencies.New(config, opts...)
	result := runner.Run(ctx)

	// Convert observations from dependencies package to phases package
	observations := convertDependencyObservations(result.Observations)

	if !result.Success {
		return RunReport{
			Err:                   result.Error,
			FailureClassification: convertDependencyFailureClass(result.FailureClass),
			Remediation:           result.Remediation,
			Observations:          observations,
		}
	}

	// Final summary
	observations = append(observations, Observation{
		Prefix: "SUCCESS",
		Text:   fmt.Sprintf("Dependency validation completed (%d checks)", result.Summary.TotalChecks()),
	})

	logPhaseSuccess(logWriter, "Dependency validation complete")
	return RunReport{Observations: observations}
}

// createResourceChecker creates a resource health checker if the vrooli CLI is available.
func createResourceChecker(ctx context.Context, env workspace.Environment, logWriter io.Writer) resources.HealthChecker {
	// Check if vrooli CLI is available
	if err := EnsureCommandAvailable("vrooli"); err != nil {
		logPhaseWarn(logWriter, "vrooli CLI unavailable, skipping resource health checks: %v", err)
		return nil
	}

	// Determine app root
	appRoot := env.AppRoot
	if appRoot == "" {
		appRoot = workspace.AppRootFromScenario(env.ScenarioDir)
	}

	// Create CLI-based status fetcher
	fetcher := &cliStatusFetcher{
		scenarioName: env.ScenarioName,
		appRoot:      appRoot,
		logWriter:    logWriter,
	}

	return resources.NewChecker(fetcher, logWriter)
}

// cliStatusFetcher implements resources.StatusFetcher using the vrooli CLI.
type cliStatusFetcher struct {
	scenarioName string
	appRoot      string
	logWriter    io.Writer
}

// Fetch implements resources.StatusFetcher.
func (f *cliStatusFetcher) Fetch(ctx context.Context) (*resources.ScenarioStatus, error) {
	return fetchScenarioStatusForResources(ctx, f.scenarioName, f.appRoot, f.logWriter)
}

// fetchScenarioStatusForResources fetches scenario status and converts to resources types.
func fetchScenarioStatusForResources(ctx context.Context, scenarioName, appRoot string, logWriter io.Writer) (*resources.ScenarioStatus, error) {
	logPhaseStep(logWriter, "collecting scenario status via 'vrooli scenario status %s --json'", scenarioName)
	output, err := phaseCommandCapture(ctx, appRoot, nil, "vrooli", "scenario", "status", scenarioName, "--json")
	if err != nil {
		return nil, fmt.Errorf("vrooli scenario status failed: %w", err)
	}

	// Parse directly into resources.ScenarioStatus
	var status resources.ScenarioStatus
	if err := parseJSON(output, &status); err != nil {
		return nil, fmt.Errorf("failed to parse scenario status JSON: %w", err)
	}

	return &status, nil
}

// parseJSON is a helper to parse JSON from a string.
func parseJSON(data string, v interface{}) error {
	return jsonUnmarshal([]byte(data), v)
}

// jsonUnmarshal is a variable to allow testing.
var jsonUnmarshal = json.Unmarshal

// convertDependencyObservations converts dependencies.Observation slice to phases.Observation slice.
func convertDependencyObservations(obs []dependencies.Observation) []Observation {
	result := make([]Observation, len(obs))
	for i, o := range obs {
		result[i] = convertDependencyObservation(o)
	}
	return result
}

// convertDependencyObservation converts a single dependencies.Observation to phases.Observation.
func convertDependencyObservation(o dependencies.Observation) Observation {
	switch o.Type {
	case dependencies.ObservationSection:
		return NewSectionObservation(o.Icon, o.Message)
	case dependencies.ObservationSuccess:
		return NewSuccessObservation(o.Message)
	case dependencies.ObservationWarning:
		return NewWarningObservation(o.Message)
	case dependencies.ObservationError:
		return NewErrorObservation(o.Message)
	case dependencies.ObservationInfo:
		return NewInfoObservation(o.Message)
	case dependencies.ObservationSkip:
		return NewSkipObservation(o.Message)
	default:
		return NewObservation(o.Message)
	}
}

// convertDependencyFailureClass converts dependencies.FailureClass to phases failure classification string.
func convertDependencyFailureClass(fc dependencies.FailureClass) string {
	switch fc {
	case dependencies.FailureClassMisconfiguration:
		return FailureClassMisconfiguration
	case dependencies.FailureClassMissingDependency:
		return FailureClassMissingDependency
	case dependencies.FailureClassSystem:
		return FailureClassSystem
	default:
		return FailureClassSystem
	}
}
