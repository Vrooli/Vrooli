package phases

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/orchestrator/phases/validationprovider"
	"test-genie/internal/orchestrator/workspace"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// TestWritePhasePointerPersistsFindings pins the cached-artifact contract
// consumed by scenario-completeness-scoring: the per-run phase-results file
// carries the phase's normalized findings, round-trippable through
// encoding/json into ArchitectureFinding with enum values intact.
func TestWritePhasePointerPersistsFindings(t *testing.T) {
	dir := t.TempDir()
	env := workspace.Environment{
		RunID:        "20260610-000000-testrun",
		ScenarioName: "fixture",
		ScenarioDir:  dir,
	}
	report := RunReport{
		Observations: []Observation{NewSuccessObservation("checked")},
		Findings: []*architecturev1.ArchitectureFinding{
			{
				Scenario: "fixture",
				Source:   architecturev1.FindingSource_FINDING_SOURCE_STANDARDS,
				Severity: architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
				Message:  "placeholder PRD content",
			},
		},
	}

	writePhasePointer(env, "quality", report, nil, io.Discard)

	raw, err := os.ReadFile(filepath.Join(dir, "coverage", "runs", env.RunID, "phase-results", "quality.json"))
	if err != nil {
		t.Fatalf("phase pointer not written: %v", err)
	}
	var payload struct {
		Status   string                                `json:"status"`
		Findings []*architecturev1.ArchitectureFinding `json:"findings"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("phase pointer not decodable: %v", err)
	}
	if payload.Status != "passed" {
		t.Fatalf("status = %q, want passed", payload.Status)
	}
	if len(payload.Findings) != 1 {
		t.Fatalf("findings count = %d, want 1", len(payload.Findings))
	}
	got := payload.Findings[0]
	if got.Severity != architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR {
		t.Fatalf("severity = %v, want ERROR", got.Severity)
	}
	if got.Source != architecturev1.FindingSource_FINDING_SOURCE_STANDARDS {
		t.Fatalf("source = %v, want STANDARDS", got.Source)
	}
}

// TestWritePhasePointerOmitsEmptyFindings keeps older-shape parity: a phase
// with no findings writes no findings key at all (consumers use key
// presence to distinguish "clean pass" from "writer predates findings").
func TestWritePhasePointerOmitsEmptyFindings(t *testing.T) {
	dir := t.TempDir()
	env := workspace.Environment{
		RunID:        "20260610-000001-testrun",
		ScenarioName: "fixture",
		ScenarioDir:  dir,
	}
	writePhasePointer(env, "smoke", RunReport{
		Observations: []Observation{NewSuccessObservation("ok")},
	}, nil, io.Discard)

	raw, err := os.ReadFile(filepath.Join(dir, "coverage", "runs", env.RunID, "phase-results", "smoke.json"))
	if err != nil {
		t.Fatalf("phase pointer not written: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("phase pointer not decodable: %v", err)
	}
	if _, present := payload["findings"]; present {
		t.Fatalf("findings key present on findings-less report: %v", payload)
	}
}

func TestWritePhasePointerPersistsExtras(t *testing.T) {
	dir := t.TempDir()
	env := workspace.Environment{
		RunID:        "20260610-000002-testrun",
		ScenarioName: "fixture",
		ScenarioDir:  dir,
	}
	report := RunReport{Observations: []Observation{NewSuccessObservation("provider check passed")}}
	writePhasePointer(env, "workflow", report, map[string]any{"summary": "1 check"}, io.Discard)

	raw, err := os.ReadFile(filepath.Join(dir, "coverage", "runs", env.RunID, "phase-results", "workflow.json"))
	if err != nil {
		t.Fatalf("phase pointer not written: %v", err)
	}
	var payload struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("phase pointer not decodable: %v", err)
	}
	if payload.Summary != "1 check" {
		t.Fatalf("pointer summary = %q, want 1 check", payload.Summary)
	}
}

func TestWriteDurableChildReferencePersistsBeforeWait(t *testing.T) {
	dir := t.TempDir()
	env := workspace.Environment{RunID: "20260610-000003-testrun", ScenarioName: "fixture", ScenarioDir: dir}
	writeDurableChildReference(env, "workflow", "workflow-health", validationprovider.RunReference{
		RunID: "provider-run-1", ParentRunID: env.RunID, ETASeconds: 120, State: "queued",
	}, io.Discard)

	raw, err := os.ReadFile(filepath.Join(dir, "coverage", "runs", env.RunID, "phase-results", "workflow.json"))
	if err != nil {
		t.Fatalf("durable child reference not written: %v", err)
	}
	var payload struct {
		Status        string `json:"status"`
		DeliveryMode  string `json:"delivery_mode"`
		ProviderRunID string `json:"provider_run_id"`
		ParentRunID   string `json:"parent_run_id"`
		ETASeconds    int64  `json:"provider_eta_secs"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("durable child reference not decodable: %v", err)
	}
	if payload.Status != "in_progress" || payload.DeliveryMode != "durable-run" || payload.ProviderRunID != "provider-run-1" || payload.ParentRunID != env.RunID || payload.ETASeconds != 120 {
		t.Fatalf("durable child reference = %+v", payload)
	}
}

func TestDeriveStatus(t *testing.T) {
	t.Run("returns failed when execution failed", func(t *testing.T) {
		if got := deriveStatus(nil, assertErr{}, ""); got != "failed" {
			t.Fatalf("deriveStatus() = %q, want failed", got)
		}
	})

	t.Run("returns failed when failure class is set", func(t *testing.T) {
		if got := deriveStatus(nil, nil, FailureClassSystem); got != "failed" {
			t.Fatalf("deriveStatus() = %q, want failed", got)
		}
	})

	t.Run("returns skipped only when all meaningful observations are skips", func(t *testing.T) {
		obs := []Observation{
			NewSectionObservation("🔍", "Checks"),
			NewSkipObservation("python not detected"),
			NewSkipObservation("websocket not configured"),
		}
		if got := deriveStatus(obs, nil, ""); got != "skipped" {
			t.Fatalf("deriveStatus() = %q, want skipped", got)
		}
	})

	t.Run("returns passed when skips are mixed with passing observations", func(t *testing.T) {
		obs := []Observation{
			NewSectionObservation("🔍", "Checks"),
			NewSuccessObservation("go tests passed"),
			NewSkipObservation("python not detected"),
		}
		if got := deriveStatus(obs, nil, ""); got != "passed" {
			t.Fatalf("deriveStatus() = %q, want passed", got)
		}
	})
}

type assertErr struct{}

func (assertErr) Error() string { return "boom" }
