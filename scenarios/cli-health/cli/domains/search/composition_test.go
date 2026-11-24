package search

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestCompositionSearchReportMarksSourceEvidencePartial(t *testing.T) {
	response, err := structpb.NewStruct(map[string]any{"results": []any{map[string]any{"title": "cli-health ScenarioApp and command groups", "text": "NewStandardScenarioApp delegates command groups", "attestation": "PARTIAL", "evidence": []any{map[string]any{"path": "cli/app.go", "line": 22}}}}})
	if err != nil {
		t.Fatal(err)
	}
	got := (&handlers{}).compositionSearchReport(cliapp.OperationContext{}, response)
	if len(got.Summary) == 0 || got.Summary[0] == "" || len(got.Results) != 1 {
		t.Fatalf("unexpected report: %+v", got)
	}
	if got.Summary[0] != "Found 1 static CLI composition result(s); runtime dispatch remains unverified." {
		t.Fatalf("report overstates runtime proof: %+v", got)
	}
}
