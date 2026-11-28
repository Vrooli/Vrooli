package drift

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestSearchReportKeepsDirectAPIEvidenceAndUnknownStatus(t *testing.T) {
	response, err := structpb.NewStruct(map[string]any{
		"scope": "scenario:demo", "results": []any{map[string]any{
			"domain": "billing", "claim": "Source Paths includes api/internal/billing", "classification": "STRUCTURAL_PATH_MATCH", "status": "PARTIAL",
			"api_status":    "ROUTE_HANDLER_DIRECT_CALL",
			"documentation": map[string]any{"path": "docs/concepts/DOMAINS.md", "line": 10, "sha256": "doc"},
			"code":          []any{}, "api": []any{
				map[string]any{"path": "api/handlers/billing/module.go", "line": 20, "sha256": "route", "basis": "POST /api/v1/billing/charge literal mux route"},
				map[string]any{"path": "api/internal/billing/service.go", "line": 4, "sha256": "callee", "basis": "direct target function"},
			},
		}}, "degradations": []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	report := (&handlers{}).searchReport(cliapp.OperationContext{}, response)
	if len(report.Result) < 2 || !strings.Contains(report.Result[1], "ROUTE_HANDLER_DIRECT_CALL") || !strings.Contains(report.Result[1], "sha256:route") || !strings.Contains(report.Result[1], "POST /api/v1/billing/charge") {
		t.Fatalf("API chain missing from CLI report: %#v", report.Result)
	}
	if !strings.Contains(report.Result[0], "Structural matches do not prove semantic conformance") {
		t.Fatalf("semantic boundary lost: %#v", report.Result)
	}
}
