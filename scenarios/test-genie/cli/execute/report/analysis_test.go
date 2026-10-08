package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	execTypes "test-genie/cli/internal/execute"
)

func TestAnalyzePhaseFailuresDoesNotMisclassifyTimeoutParameterAsPhaseTimeout(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "quality.log")
	if err := os.WriteFile(logPath, []byte("running quality validation (timeout=60s)\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	insights := AnalyzePhaseFailures([]execTypes.Phase{
		{
			Name:           "quality",
			Status:         "failed",
			LogPath:        logPath,
			Error:          "quality validation exceeded policy threshold (timeout=60s)",
			Classification: "misconfiguration",
		},
	})
	if len(insights) != 1 {
		t.Fatalf("expected 1 insight, got %d", len(insights))
	}
	if insights[0].Cause == "Phase timeout" {
		t.Fatalf("expected non-timeout cause, got %q", insights[0].Cause)
	}
}

func TestAnalyzePhaseFailuresDoesNotLeakJSONFromProviderLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "quality.log")
	logContent := strings.Join([]string{
		"running quality validation",
		`{"summary":{"total":1,"highest_severity":"high","top_violations":[{"severity":"high","title":"Example","file_path":"api/main.go","line_number":1,"recommendation":"example"}]}}`,
	}, "\n")
	if err := os.WriteFile(logPath, []byte(logContent), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	insights := AnalyzePhaseFailures([]execTypes.Phase{
		{
			Name:        "quality",
			Status:      "failed",
			LogPath:     logPath,
			Error:       "quality validation failed",
			Remediation: "Run `quality-health validate scenario demo` and fix findings.",
		},
	})
	if len(insights) != 1 {
		t.Fatalf("expected 1 insight, got %d", len(insights))
	}
	if strings.Contains(insights[0].Detail, `{"summary"`) {
		t.Fatalf("expected JSON to be omitted from insight detail, got %q", insights[0].Detail)
	}
	if insights[0].Cause == "" || !strings.Contains(insights[0].Cause, "quality validation failed") {
		t.Fatalf("expected provider error to be used as cause, got %q", insights[0].Cause)
	}
	if len(insights[0].Fixes) == 0 {
		t.Fatalf("expected remediation fix to be included")
	}
}
