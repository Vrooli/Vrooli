package phases

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/playbooks"
)

// runPlaybooksPhase executes BAS playbook workflows using the playbooks package.
// This includes loading the registry, executing workflows via BAS API, and
// writing results for requirements coverage tracking.
func runPlaybooksPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{Err: err, FailureClassification: FailureClassSystem}
	}

	// Create playbooks runner with production hooks
	runner := playbooks.New(playbooks.Config{
		ScenarioDir:  env.ScenarioDir,
		ScenarioName: env.ScenarioName,
		TestDir:      env.TestDir,
		AppRoot:      env.AppRoot,
	},
		playbooks.WithLogger(logWriter),
		playbooks.WithPortResolver(resolveScenarioPort),
		playbooks.WithScenarioStarter(func(ctx context.Context, scenario string) error {
			return startScenario(ctx, scenario, logWriter)
		}),
		playbooks.WithUIBaseURLResolver(resolveScenarioBaseURL),
	)

	result := runner.Run(ctx)

	// Convert observations from playbooks package to phases package
	observations := convertPlaybooksObservations(result.Observations)

	if !result.Success {
		return RunReport{
			Err:                   result.Error,
			FailureClassification: convertPlaybooksFailureClass(result.FailureClass),
			Remediation:           result.Remediation,
			Observations:          observations,
		}
	}

	logPhaseSuccess(logWriter, "Playbooks phase complete")
	return RunReport{Observations: observations}
}

// convertPlaybooksObservations converts playbooks.Observation slice to phases.Observation slice.
func convertPlaybooksObservations(obs []playbooks.Observation) []Observation {
	result := make([]Observation, len(obs))
	for i, o := range obs {
		result[i] = convertPlaybooksObservation(o)
	}
	return result
}

// convertPlaybooksObservation converts a single playbooks.Observation to phases.Observation.
func convertPlaybooksObservation(o playbooks.Observation) Observation {
	switch o.Type {
	case playbooks.ObservationSection:
		return NewSectionObservation(o.Icon, o.Message)
	case playbooks.ObservationSuccess:
		return NewSuccessObservation(o.Message)
	case playbooks.ObservationWarning:
		return NewWarningObservation(o.Message)
	case playbooks.ObservationError:
		return NewErrorObservation(o.Message)
	case playbooks.ObservationInfo:
		return NewInfoObservation(o.Message)
	case playbooks.ObservationSkip:
		return NewSkipObservation(o.Message)
	default:
		return NewObservation(o.Message)
	}
}

// convertPlaybooksFailureClass converts playbooks.FailureClass to phases failure classification.
func convertPlaybooksFailureClass(fc playbooks.FailureClass) string {
	switch fc {
	case playbooks.FailureClassMisconfiguration:
		return FailureClassMisconfiguration
	case playbooks.FailureClassMissingDependency:
		return FailureClassMissingDependency
	case playbooks.FailureClassSystem:
		return FailureClassSystem
	case playbooks.FailureClassExecution:
		return FailureClassSystem // Map execution failures to system for phase reporting
	default:
		return FailureClassSystem
	}
}

// resolveScenarioPort resolves a port for a scenario using vrooli CLI.
func resolveScenarioPort(ctx context.Context, scenarioName, portName string) (string, error) {
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", scenarioName, portName)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("vrooli port lookup failed: %v: %s", err, stderr.String())
	}
	value := strings.TrimSpace(stdout.String())
	if value == "" {
		return "", fmt.Errorf("port lookup returned empty output")
	}
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if strings.TrimSpace(parts[0]) == portName {
				value = strings.TrimSpace(parts[1])
				break
			}
		}
	}
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, "\r")
	if _, err := strconv.Atoi(value); err != nil {
		return "", fmt.Errorf("invalid port value %q", value)
	}
	return value, nil
}

// resolveScenarioBaseURL resolves the UI base URL for a scenario.
func resolveScenarioBaseURL(ctx context.Context, scenarioName string) (string, error) {
	port, err := resolveScenarioPort(ctx, scenarioName, "UI_PORT")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://127.0.0.1:%s", port), nil
}

// startScenario starts a scenario using vrooli CLI.
func startScenario(ctx context.Context, scenarioName string, logWriter io.Writer) error {
	logPhaseStep(logWriter, "starting scenario %s for BAS workflow execution", scenarioName)
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "start", scenarioName, "--clean-stale")
	var stderr bytes.Buffer
	cmd.Stdout = logWriter
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start %s: %v %s", scenarioName, err, stderr.String())
	}
	return nil
}
