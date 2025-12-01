package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// runStructurePhase validates the essential scenario layout without shelling out to bash scripts.
func runStructurePhase(ctx context.Context, env PhaseEnvironment, logWriter io.Writer) PhaseRunReport {
	if err := ctx.Err(); err != nil {
		return PhaseRunReport{Err: err, FailureClassification: failureClassSystem}
	}

	var observations []string
	requiredDirs := []string{"api", "cli", "requirements", filepath.Join("test", "phases"), "ui"}
	for _, rel := range requiredDirs {
		abs := filepath.Join(env.ScenarioDir, rel)
		if err := ensureDir(abs); err != nil {
			return PhaseRunReport{
				Err:                   err,
				FailureClassification: failureClassMisconfiguration,
				Remediation:           fmt.Sprintf("Create the '%s' directory to match the scenario template.", rel),
			}
		}
		logPhaseStep(logWriter, "directory verified: %s", abs)
		observations = append(observations, fmt.Sprintf("directory present: %s", rel))
	}

	manifestPath := filepath.Join(env.ScenarioDir, ".vrooli", "service.json")
	if err := ensureFile(manifestPath); err != nil {
		return PhaseRunReport{
			Err:                   err,
			FailureClassification: failureClassMisconfiguration,
			Remediation:           "Run `vrooli scenario init` or restore .vrooli/service.json.",
		}
	}
	logPhaseStep(logWriter, "manifest located: %s", manifestPath)
	manifest, err := loadServiceManifest(manifestPath)
	if err != nil {
		return PhaseRunReport{
			Err:                   err,
			FailureClassification: failureClassMisconfiguration,
			Remediation:           fmt.Sprintf("Fix JSON syntax in %s so the manifest can be parsed.", manifestPath),
		}
	}
	if manifest.Service.Name == "" {
		return PhaseRunReport{
			Err:                   fmt.Errorf("service.name is required in %s", manifestPath),
			FailureClassification: failureClassMisconfiguration,
			Remediation:           "Set service.name to the scenario folder name in .vrooli/service.json.",
		}
	}
	if manifest.Service.Name != env.ScenarioName {
		return PhaseRunReport{
			Err:                   fmt.Errorf("service.name '%s' does not match scenario '%s'", manifest.Service.Name, env.ScenarioName),
			FailureClassification: failureClassMisconfiguration,
			Remediation:           fmt.Sprintf("Update service.name in %s to '%s'.", manifestPath, env.ScenarioName),
		}
	}
	if len(manifest.Lifecycle.Health.Checks) == 0 {
		return PhaseRunReport{
			Err:                   fmt.Errorf("lifecycle.health.checks must define at least one entry in %s", manifestPath),
			FailureClassification: failureClassMisconfiguration,
			Remediation:           "Add at least one health check entry under lifecycle.health.checks.",
		}
	}
	logPhaseStep(logWriter, "manifest validated for scenario '%s'", env.ScenarioName)

	runnerPath := filepath.Join(env.TestDir, "run-tests.sh")
	if err := ensureFile(runnerPath); err != nil {
		return PhaseRunReport{
			Err:                   err,
			FailureClassification: failureClassMisconfiguration,
			Remediation:           "Restore test/run-tests.sh from the scenario template.",
		}
	}
	content, err := os.ReadFile(runnerPath)
	if err != nil {
		return PhaseRunReport{
			Err:                   fmt.Errorf("failed to read %s: %w", runnerPath, err),
			FailureClassification: failureClassSystem,
			Remediation:           "Ensure the orchestrator script is readable by the scenario runtime.",
		}
	}
	if strings.Contains(string(content), "scripts/scenarios/testing") {
		return PhaseRunReport{
			Err:                   fmt.Errorf("%s must not reference scripts/scenarios/testing", runnerPath),
			FailureClassification: failureClassMisconfiguration,
			Remediation:           "Update run-tests.sh to call the scenario-local orchestrator instead of legacy scripts.",
		}
	}
	logPhaseStep(logWriter, "verified scenario-local orchestrator usage: %s", runnerPath)
	observations = append(observations, "scenario-local orchestrator enforced")

	logPhaseStep(logWriter, "structure validation complete")
	return PhaseRunReport{Observations: observations}
}
