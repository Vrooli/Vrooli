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
