package facts

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestInvariantCandidateCLIReportKeepsUnknownLayersVisible(t *testing.T) {
	report := (&handlers{}).invariantCandidatesReport(cliapp.OperationContext{}, &factsv1.SearchInvariantCandidatesResponse{Scope: "one scenario source path", Results: []*factsv1.InvariantCandidate{{Title: "guard candidate", Path: "api/main.go", Line: 40, Sha256: "abc", Attestation: "CANDIDATE_ONLY", Enforcement: &factsv1.InvariantEnforcementLayers{Type: "UNKNOWN", Database: "UNKNOWN", Test: "UNKNOWN", Runtime: "UNKNOWN"}}}})
	joined := strings.Join(append(report.Summary, report.Results...), "\n")
	for _, want := range []string{"candidate source location", "CANDIDATE_ONLY", "type=UNKNOWN", "db=UNKNOWN", "test=UNKNOWN", "runtime=UNKNOWN", "api/main.go:40"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("report missing %q: %s", want, joined)
		}
	}
}
