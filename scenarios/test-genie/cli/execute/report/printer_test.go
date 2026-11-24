package report

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"test-genie/cli/internal/repo"

	execTypes "test-genie/cli/internal/execute"
)

func TestPrinterKeepsFailureReportConcise(t *testing.T) {
	tmp := t.TempDir()

	unitLog := filepath.Join(tmp, "unit.log")
	if err := os.WriteFile(unitLog, []byte("❌ test failed: sample\n"), 0o644); err != nil {
		t.Fatalf("write unit log: %v", err)
	}
	integrationLog := filepath.Join(tmp, "integration.log")
	if err := os.WriteFile(integrationLog, []byte("UI bundle is stale and must be rebuilt\n"), 0o644); err != nil {
		t.Fatalf("write integration log: %v", err)
	}

	resp := execTypes.Response{
		Success:     true,
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		CompletedAt: time.Now().Add(30 * time.Second).UTC().Format(time.RFC3339),
		PhaseSummary: execTypes.PhaseSummary{
			Total:           2,
			Failed:          2,
			DurationSeconds: 30,
		},
		Phases: []execTypes.Phase{
			{Name: "unit", Status: "failed", DurationSeconds: 8, LogPath: unitLog},
			{Name: "integration", Status: "failed", DurationSeconds: 12, LogPath: integrationLog},
		},
	}

	var buf bytes.Buffer
	pr := New(&buf, "demo-scenario", "", nil, nil, false, nil, nil)
	pr.Print(resp)

	out := buf.String()
	for _, token := range []string{"ERROR DIGEST:", "UI bundle", "artifact roots:"} {
		if !strings.Contains(out, token) {
			t.Fatalf("expected output to contain %q\n----\n%s\n----", token, out)
		}
	}
	for _, token := range []string{"QUICK FIX GUIDE:", "PHASE-SPECIFIC DEBUG GUIDES:", "DOCUMENTATION:"} {
		if strings.Contains(out, token) {
			t.Fatalf("failure report should not reintroduce redundant section %q\n----\n%s\n----", token, out)
		}
	}
}

func TestPrintResultsCondensedReplay(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "structure.log")
	if err := os.WriteFile(logPath, []byte("ERROR: lighthouse.json missing property 'version'\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	resp := execTypes.Response{
		PhaseSummary: execTypes.PhaseSummary{
			Total:           1,
			Failed:          1,
			DurationSeconds: 2,
		},
		Phases: []execTypes.Phase{
			{
				Name:            "structure",
				Status:          "failed",
				DurationSeconds: 2,
				LogPath:         logPath,
				Remediation:     "Add the missing version to .vrooli/lighthouse.json",
			},
		},
	}

	var buf bytes.Buffer
	pr := New(&buf, "demo", "", nil, nil, false, nil, nil)
	pr.SetStreamedObservations(true)
	pr.PrintResults(resp)

	out := buf.String()
	if strings.Contains(out, "[PHASE_START") {
		t.Fatalf("expected condensed replay without phase banners, got:\n%s", out)
	}
	if !strings.Contains(out, "structure") || !strings.Contains(out, "ERROR: lighthouse.json missing property 'version'") {
		t.Fatalf("expected headline from log to be included:\n%s", out)
	}
	if !strings.Contains(out, "log: "+logPath) {
		t.Fatalf("expected log path to be included:\n%s", out)
	}
	if !strings.Contains(out, "fix: Add the missing version") {
		t.Fatalf("expected remediation hint to be included:\n%s", out)
	}
	expectedDoc := repo.AbsPath("scenarios/test-genie/docs/phases/structure/README.md")
	if !strings.Contains(out, expectedDoc) {
		t.Fatalf("expected structure doc hint to be included:\n%s", out)
	}
}

func TestPrintPrePlanShowsDocs(t *testing.T) {
	var buf bytes.Buffer
	pr := New(&buf, "demo", "", nil, nil, false, nil, nil)
	pr.PrintPreExecution([]string{"quality"})

	out := buf.String()
	expectedDoc := repo.AbsPath("scenarios/test-genie/docs/phases/quality/README.md")
	if !strings.Contains(out, "docs: "+expectedDoc) {
		t.Fatalf("expected quality doc link in plan, got:\n%s", out)
	}
}

func TestPrinterUsesRequestedPresetFallbackAndPreservesResponsePreset(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		requested    string
		wantInOutput string
	}{
		{name: "blank response uses requested preset", response: " \t", requested: "safe-default", wantInOutput: "Preset: safe-default"},
		{name: "response preset remains unchanged", response: "  quick  ", requested: "safe-default", wantInOutput: "Preset:   quick  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			pr := New(&buf, "demo", tt.requested, nil, nil, false, nil, nil)
			pr.Print(execTypes.Response{PresetUsed: tt.response})
			if !strings.Contains(buf.String(), tt.wantInOutput) {
				t.Fatalf("Printer output = %q, want it to contain %q", buf.String(), tt.wantInOutput)
			}
		})
	}
}

func TestPassedPhaseWarningsAreSurfaced(t *testing.T) {
	resp := execTypes.Response{
		Success:     false,
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		CompletedAt: time.Now().Add(2 * time.Second).UTC().Format(time.RFC3339),
		PhaseSummary: execTypes.PhaseSummary{
			Total:           1,
			Passed:          1,
			Failed:          0,
			DurationSeconds: 2,
		},
		Phases: []execTypes.Phase{
			{
				Name:            "dependencies",
				Status:          "passed",
				DurationSeconds: 2,
				Observations: execTypes.ObservationList{
					{Prefix: "WARNING", Text: "resource telemetry unavailable"},
					{Prefix: "SUCCESS", Text: "command available: curl"},
				},
			},
		},
	}

	var buf bytes.Buffer
	pr := New(&buf, "demo", "", nil, nil, false, nil, nil)
	pr.Print(resp)

	out := buf.String()
	if !strings.Contains(out, "warnings: resource telemetry unavailable") {
		t.Fatalf("expected warning details to be surfaced for passed phase:\n%s", out)
	}
}

func TestPrinterSurfacesNotEvaluatedPhaseCount(t *testing.T) {
	var buf bytes.Buffer
	pr := New(&buf, "demo", "", nil, nil, false, nil, nil)
	pr.Print(execTypes.Response{Success: false, Verdict: "NOT_EVALUATED", PhaseSummary: execTypes.PhaseSummary{Total: 1, NotEvaluated: 1}})
	if !strings.Contains(buf.String(), "1 not evaluated") {
		t.Fatalf("human report must make missing evaluation visible: %s", buf.String())
	}
}
