package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/runreport"

	"github.com/google/uuid"
)

func TestRunReportTextRendersRoutingDiscriminatorsAndNoPayloads(t *testing.T) {
	report := &runreport.RunReport{
		RunID: uuid.New(), Status: domain.RunStatusFailed, Error: "tool failed",
		Turns: 3, Tokens: 42, CostUSD: 0.25, ProjectOwnedToolCalls: 2,
		ExternalToolCalls: 1, RequestedModel: "primary", ActualModel: "fallback",
		FallbackCount: 1, RepeatedToolCalls: 2, FilesReadMoreThanOnce: 1, Result: runreport.ResultSummary{
			SelectionStatus: domain.FinalOutputSelectionAmbiguous, SelectionRule: "multiple",
			CandidateCount: 2, StructuredStatus: domain.StructuredResultInvalid,
			DiagnosticCodes: []string{"schema_invalid"},
		},
		Events: map[string]int{"tool_result": 2, "model.fallback.attempted": 1},
		Tools:  []runreport.ToolSummary{{Name: "shell", Calls: 2, Failures: 1}},
		Diff:   runreport.DiffSummary{Files: 1, Bytes: 8, Available: runreport.Availability{State: "available"}},
	}
	text := RunReportText(report)
	for _, expected := range []string{"Status: failed", "Final output: ambiguous", "schema_invalid", "fallbacks=1", "failed=1", "Efficiency: repeated tool calls=2 files reread=1", "Diff: files=1", "Next:"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("report missing %q:\n%s", expected, text)
		}
	}
	if strings.Contains(text, "unified diff") || strings.Contains(text, "message body") {
		t.Fatalf("report leaked bulk payload:\n%s", text)
	}
}

func TestRunReportTextPathologyGolden(t *testing.T) {
	report := &runreport.RunReport{
		RunID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Status: domain.RunStatusFailed, Error: "tool failed",
		Turns: 3, Tokens: 42, CostUSD: 0.25, ProjectOwnedToolCalls: 2, ExternalToolCalls: 1,
		RequestedModel: "primary", ActualModel: "fallback", FallbackCount: 1, RepeatedToolCalls: 2, FilesReadMoreThanOnce: 1,
		Result: runreport.ResultSummary{SelectionStatus: domain.FinalOutputSelectionAmbiguous, SelectionRule: "multiple", CandidateCount: 2, StructuredStatus: domain.StructuredResultInvalid, DiagnosticCodes: []string{"schema_invalid"}},
		Events: map[string]int{"tool_result": 2, "model.fallback.attempted": 1},
		Tools:  []runreport.ToolSummary{{Name: "shell", Calls: 2, Failures: 1}},
		Diff:   runreport.DiffSummary{Files: 1, Bytes: 8, Available: runreport.Availability{State: "available"}},
	}
	want, err := os.ReadFile(filepath.Join("testdata", "pathology-report.golden"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if got := RunReportText(report); got != string(want) {
		t.Fatalf("run report golden mismatch (-want +got):\n-%s\n+%s", want, got)
	}
}
