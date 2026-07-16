package phases

import (
	"encoding/json"
	"io"
	"path/filepath"
	"time"

	"test-genie/internal/orchestrator/phases/validationprovider"
	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

// writeDurableChildReference records the provider child before Wait begins.
// If Test Genie restarts, the next descriptor-driven Start reuses its
// parent+phase idempotency key and therefore reconciles this same provider run
// instead of creating duplicate execution work.
func writeDurableChildReference(env workspace.Environment, phaseName, provider string, ref validationprovider.RunReference, logWriter io.Writer) {
	payload := map[string]any{
		"phase":             phaseName,
		"scenario":          env.ScenarioName,
		"status":            "in_progress",
		"delivery_mode":     "durable-run",
		"provider":          provider,
		"parent_run_id":     ref.ParentRunID,
		"provider_run_id":   ref.RunID,
		"provider_state":    ref.State,
		"provider_eta_secs": ref.ETASeconds,
		"updated_at":        time.Now().UTC().Format(time.RFC3339),
	}
	writer := sharedartifacts.NewBaseWriter(env.ScenarioDir, env.ScenarioName, env.RunID)
	targetDir := sharedartifacts.RunPhaseResultsDir(env.ScenarioDir, env.RunID)
	if err := writer.EnsureDir(targetDir); err != nil {
		shared.LogWarn(logWriter, "failed to create phase results dir for durable child: %v", err)
		return
	}
	if err := writer.WriteJSON(filepath.Join(targetDir, phaseName+".json"), payload); err != nil {
		shared.LogWarn(logWriter, "failed to persist durable child reference: %v", err)
	}
}

// writePhasePointer persists a lightweight phase summary to coverage/runs/<runID>/phase-results/<phase>.json.
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
	if len(report.Findings) > 0 {
		// Persist the normalized findings alongside the summary so cached-
		// artifact readers (scenario-completeness-scoring) can count findings
		// per dimension without re-running the suite. Same encoding/json
		// contract as ExecutionResult.Findings: enums marshal as proto ints.
		payload["findings"] = report.Findings
	}
	if report.FindingSource != "" {
		payload["findingSource"] = report.FindingSource
	}
	for k, v := range extras {
		if v != nil {
			payload[k] = v
		}
	}

	writer := sharedartifacts.NewBaseWriter(env.ScenarioDir, env.ScenarioName, env.RunID)
	targetDir := sharedartifacts.RunPhaseResultsDir(env.ScenarioDir, env.RunID)
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

	meaningful := 0
	skips := 0
	for _, observation := range obs {
		if observation.Section != "" && observation.Text == "" && observation.Prefix == "" {
			continue
		}
		meaningful++
		if observation.Prefix == "SKIP" {
			skips++
		}
	}
	if meaningful > 0 && skips == meaningful {
		return "skipped"
	}
	return "passed"
}
