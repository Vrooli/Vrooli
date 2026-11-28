package facts

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestAmbientSeamReportShowsLayerProvenanceAndUnknownRuntime(t *testing.T) {
	response, err := structpb.NewStruct(map[string]any{"scope": "scenario Go static source only", "match_status": "MATCHED", "results": []any{map[string]any{"kind": "http_client", "expression": "http.DefaultClient", "source_path": "api/handlers/facts/module.go", "line": 28, "source_sha256": "abc", "source_layer": "NON_TEST_SOURCE_CANDIDATE", "context_symbol": "Module", "argument_context_status": "STATIC_ARGUMENT_RELATIONSHIP", "argument_callee": "NewTypeScriptGraphProvider", "argument_line": 28, "matched_terms": []any{"client"}, "matched_fields": []any{"expression"}, "match_status": "MATCHED", "match_evidence": []any{map[string]any{"term": "client", "field": "expression", "value": "http.DefaultClient", "source_path": "api/handlers/facts/module.go", "line": 28, "source_sha256": "abc"}}, "limit": "No runtime proof"}}, "degradations": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	lines := (&handlers{}).ambientSeamsReport(cliapp.OperationContext{}, response).Result
	text := strings.Join(lines, "\n")
	for _, want := range []string{"match status=MATCHED", "client→expression=\"http.DefaultClient\"", "NON_TEST_SOURCE_CANDIDATE", "module.go:28", "sha256:abc", "NewTypeScriptGraphProvider", "does not establish production wiring"} {
		if !strings.Contains(text, want) {
			t.Fatalf("ambient seam report lacks %q: %s", want, text)
		}
	}
}
