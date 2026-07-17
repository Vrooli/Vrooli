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

// TestWritePhasePointerPersistsOnlyFindingsReference pins the canonical-owner
// contract: phase projections retain counts and a reference, never a second
// normalized findings array.
func TestWritePhasePointerPersistsOnlyFindingsReference(t *testing.T) {
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
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("phase pointer not decodable: %v", err)
	}
	if payload["status"] != "passed" {
		t.Fatalf("status = %q, want passed", payload["status"])
	}
	if payload["finding_count"] != float64(1) {
		t.Fatalf("finding_count = %#v, want 1", payload["finding_count"])
	}
	if payload["findings_artifact"] != filepath.Join("coverage", "runs", env.RunID, "findings.json") {
		t.Fatalf("findings_artifact = %#v", payload["findings_artifact"])
	}
	if _, duplicate := payload["findings"]; duplicate {
		t.Fatalf("phase pointer embeds duplicate findings: %v", payload)
	}
	if _, duplicate := payload["observations"]; duplicate {
		t.Fatalf("phase pointer embeds duplicate observations: %v", payload)
	}
}

func TestWritePhasePointerRetainsZeroCountsWithoutDetailArrays(t *testing.T) {
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
	if payload["finding_count"] != float64(0) || payload["observation_count"] != float64(1) {
		t.Fatalf("counts = %#v", payload)
	}
	if _, present := payload["findings_artifact"]; present {
		t.Fatalf("findings artifact present on zero-finding report: %v", payload)
	}
	if _, present := payload["findings"]; present {
		t.Fatalf("duplicate findings key present: %v", payload)
	}
	if _, present := payload["observations"]; present {
		t.Fatalf("duplicate observations key present: %v", payload)
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
