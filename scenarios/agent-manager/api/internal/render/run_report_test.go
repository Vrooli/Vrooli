package render

import (
	"fmt"
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
		Tools:  []runreport.ToolSummary{{Name: "shell", Calls: 2, Returned: 2, Failures: 1, UnknownOutcome: 1, Unresolved: 0}},
		Diff:   runreport.DiffSummary{Files: 1, Bytes: 8, Available: runreport.Availability{State: "available"}},
	}
	text := RunReportText(report)
	for _, expected := range []string{"Status: failed", "Final output: ambiguous", "schema_invalid", "fallbacks=1", "shell: calls=2 returned=2 success=0 failed=1 unknown=1 unresolved=0", "Efficiency: recognized command retries=2 files reread=1", "Diff: files=1", "Next:"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("report missing %q:\n%s", expected, text)
		}
	}
	// Follow-up commands must retain every public routing discriminator.
	for _, command := range []string{"result", "events", "tools", "diff", "receipts"} {
		expected := fmt.Sprintf("agent-manager run %s %s", command, report.RunID)
		if command == "tools" {
			expected += " --failed"
		}
		if !strings.Contains(text, expected) {
			t.Fatalf("report missing route %q:\n%s", expected, text)
		}
	}
	if strings.Contains(text, "unified diff") || strings.Contains(text, "message body") {
		t.Fatalf("report leaked bulk payload:\n%s", text)
	}
}
