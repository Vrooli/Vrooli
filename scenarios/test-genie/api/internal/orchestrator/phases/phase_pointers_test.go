package phases

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"

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

	writePhasePointer(env, "standards", report, nil, io.Discard)

	raw, err := os.ReadFile(filepath.Join(dir, "coverage", "runs", env.RunID, "phase-results", "standards.json"))
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

func TestRunNativePhaseLoadsExpectationsWritesSummaryAndPointer(t *testing.T) {
	dir := t.TempDir()
	env := workspace.Environment{
		RunID:        "20260610-000002-testrun",
		ScenarioName: "fixture",
		ScenarioDir:  dir,
	}
	var log bytes.Buffer
	var hookCalled bool

	report := RunNativePhase(context.Background(), env, &log, Structure,
		func(scenarioDir string) (string, error) {
			if scenarioDir != dir {
				t.Fatalf("scenarioDir = %q, want %q", scenarioDir, dir)
			}
			return "loaded", nil
		},
		func(expectations string) (StandardRunResult, error) {
			if expectations != "loaded" {
				t.Fatalf("expectations = %q, want loaded", expectations)
			}
			return fakeNativeResult{
				success: true,
				observations: []shared.Observation{
					shared.NewSuccessObservation("native check passed"),
				},
				summary: "1 check",
			}, nil
		},
		WithNativePhaseReportHook(func(report *RunReport, result StandardRunResult) {
			hookCalled = true
			if result.SummaryText() != "1 check" {
				t.Fatalf("hook summary = %q, want 1 check", result.SummaryText())
			}
			report.Findings = []*architecturev1.ArchitectureFinding{{
				Scenario: "fixture",
				Source:   architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE,
				Severity: architecturev1.FindingSeverity_FINDING_SEVERITY_INFO,
				Message:  "hook attached finding",
			}}
		}),
	)

	if !hookCalled {
		t.Fatal("report hook was not called")
	}
	if report.Err != nil {
		t.Fatalf("report error = %v, want nil", report.Err)
	}
	if len(report.Observations) != 2 {
		t.Fatalf("observations count = %d, want runner observation plus summary", len(report.Observations))
	}
	if got := report.Observations[1].Text; got != "Structure validation completed (1 check)" {
		t.Fatalf("summary observation = %q", got)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "coverage", "runs", env.RunID, "phase-results", "structure.json"))
	if err != nil {
		t.Fatalf("phase pointer not written: %v", err)
	}
	var payload struct {
		Summary  string                                `json:"summary"`
		Findings []*architecturev1.ArchitectureFinding `json:"findings"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("phase pointer not decodable: %v", err)
	}
	if payload.Summary != "1 check" {
		t.Fatalf("pointer summary = %q, want 1 check", payload.Summary)
	}
	if len(payload.Findings) != 1 {
		t.Fatalf("pointer findings count = %d, want 1", len(payload.Findings))
	}
}

func TestRunNativePhaseWritesPointerWhenExpectationsFail(t *testing.T) {
	dir := t.TempDir()
	env := workspace.Environment{
		RunID:        "20260610-000003-testrun",
		ScenarioName: "fixture",
		ScenarioDir:  dir,
	}

	report := RunNativePhase(context.Background(), env, io.Discard, Business,
		func(string) (string, error) {
			return "", errors.New("bad expectations")
		},
		func(string) (StandardRunResult, error) {
			t.Fatal("execute should not run after expectations fail")
			return fakeNativeResult{}, nil
		},
	)

	if report.Err == nil {
		t.Fatal("report error = nil, want expectation load failure")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "coverage", "runs", env.RunID, "phase-results", "business.json"))
	if err != nil {
		t.Fatalf("phase pointer not written: %v", err)
	}
	var payload struct {
		Status       string `json:"status"`
		FailureClass string `json:"failure_class"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("phase pointer not decodable: %v", err)
	}
	if payload.Status != "failed" {
		t.Fatalf("status = %q, want failed", payload.Status)
	}
	if payload.FailureClass != FailureClassMisconfiguration {
		t.Fatalf("failure_class = %q, want %q", payload.FailureClass, FailureClassMisconfiguration)
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

type fakeNativeResult struct {
	success      bool
	err          error
	failureClass shared.FailureClass
	remediation  string
	observations []shared.Observation
	summary      string
}

func (r fakeNativeResult) Succeeded() bool { return r.success }

func (r fakeNativeResult) Err() error { return r.err }

func (r fakeNativeResult) Failure() shared.FailureClass { return r.failureClass }

func (r fakeNativeResult) RemediationText() string { return r.remediation }

func (r fakeNativeResult) ObservationList() []shared.Observation { return r.observations }

func (r fakeNativeResult) SummaryText() string { return r.summary }
