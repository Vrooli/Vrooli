package runs

import (
	"strings"
	"testing"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

func TestGetRunFindingsReportLabelsHistoricalStanding(t *testing.T) {
	report := getRunFindingsReport(nil, &runspb.GetRunFindingsResponse{
		Target: "demo",
		RunId:  "historical",
		Phases: []*runspb.RunFindingsPhase{{
			Name: "ui-health",
			MaturityStanding: &runspb.PhaseMaturityStanding{
				Provider: "ui-health",
				Phase:    "ui-health",
			},
		}},
	})
	joined := strings.Join(report.Results, "\n")
	if !strings.Contains(joined, "historical maturity standing (not canonical v1)") {
		t.Fatalf("historical findings artifact was silently dropped: %q", joined)
	}
	if strings.Contains(joined, "No phase declared a maturity standing") {
		t.Fatalf("historical evidence must not be reported as absent: %q", joined)
	}
}

func TestGetRunFindingsLinksRetainedEvidenceWithoutRerunningValidation(t *testing.T) {
	report := getRunFindingsReport(nil, &runspb.GetRunFindingsResponse{Target: "demo", RunId: "native-run"})
	if len(report.RetrievalHints) != 1 || report.RetrievalHints[0] != `test-genie runs artifacts --scenario "demo" "native-run" --kinds findings.report` {
		t.Fatalf("missing original-run artifact retrieval: %v", report.RetrievalHints)
	}
	if !strings.Contains(strings.Join(report.Summary, "\n"), "missing historical native evidence remains unknown") {
		t.Fatal("summary must retain evidence limits")
	}
}
