package phases

import (
	"encoding/json"
	"io"
	"path/filepath"
	"time"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"
	sharedartifacts "test-genie/internal/shared/artifacts"
)

// writePhasePointer persists a lightweight phase summary to coverage/phase-results/<phase>.json.
// This keeps artifacts discoverable without moving existing outputs.
func writePhasePointer(env workspace.Environment, phaseName string, report RunReport, extras map[string]any, logWriter io.Writer) {
	status := deriveStatus(report.Observations, report.Err, report.FailureClassification)

	payload := map[string]any{
		"phase":      phaseName,
		"scenario":   env.ScenarioName,
		"status":     status,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}

	if report.FailureClassification != "" {
		payload["failure_class"] = report.FailureClassification
	}
	if report.Remediation != "" {
		payload["remediation"] = report.Remediation
	}
	if len(report.Observations) > 0 {
		payload["observations"] = ObservationsToStrings(report.Observations)
	}
	for k, v := range extras {
		if v != nil {
			payload[k] = v
		}
	}

	writer := sharedartifacts.NewBaseWriter(env.ScenarioDir, env.ScenarioName)
	targetDir := filepath.Join(env.ScenarioDir, sharedartifacts.PhaseResultsDir)
	if err := writer.EnsureDir(targetDir); err != nil {
		shared.LogWarn(logWriter, "failed to create phase results dir: %v", err)
		return
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		shared.LogWarn(logWriter, "failed to marshal %s phase pointer: %v", phaseName, err)
		return
	}

	path := filepath.Join(targetDir, phaseName+".json")
	if err := writer.FS.WriteFile(path, data, 0o644); err != nil {
		shared.LogWarn(logWriter, "failed to write %s phase pointer: %v", phaseName, err)
	}
}

// deriveStatus converts phase observations/errors into a simple status.
func deriveStatus(obs []Observation, err error, failureClass string) string {
	if err != nil || failureClass != "" {
		return "failed"
	}
	for _, o := range obs {
		if o.Prefix == "SKIP" {
			return "skipped"
		}
	}
	return "passed"
}
