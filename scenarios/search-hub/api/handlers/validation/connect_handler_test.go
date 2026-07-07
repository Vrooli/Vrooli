package validation_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"

	handler "search-hub/handlers/validation"
	internalvalidation "search-hub/internal/validation"
)

func TestValidateScenarioReturnsSharedAssessment(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", cleanSearchConfig("demo"))
	spec := mustLoadSpec(t)
	h := handler.NewConnectHandler(handler.Deps{
		Validator:    internalvalidation.New(root),
		MaturitySpec: spec,
	})

	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED {
		t.Fatalf("status = %s, want passed", resp.Msg.GetStatus())
	}
	if resp.Msg.GetAssessment().GetProvider() != "search-hub" || resp.Msg.GetAssessment().GetPhase() != "search" {
		t.Fatalf("identity = %s/%s, want search-hub/search", resp.Msg.GetAssessment().GetProvider(), resp.Msg.GetAssessment().GetPhase())
	}
	if resp.Msg.GetMetrics() == nil {
		t.Fatal("metrics must be populated")
	}
}

func TestValidateScenarioNativeDetailIncludesFindings(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", `{
  "version":"1.0.0",
  "providers":[{
    "provider_id":"demo.docs",
    "provider_group":"other",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Docs",
    "scope":"SCOPE_PROJECT",
    "endpoint":{"http_json":{"scenario_id":"demo","path":"/search","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_COSINE_0_1"},
    "tests":{"description":"Primary corpus","cases":[{"id":"case","query":"docs","expect_ids":["doc-1"]}]}
  }]
}`)
	h := handler.NewConnectHandler(handler.Deps{
		Validator:    internalvalidation.New(root),
		MaturitySpec: mustLoadSpec(t),
	})

	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	var detail structpb.Struct
	if err := resp.Msg.GetNativeDetail().UnmarshalTo(&detail); err != nil {
		t.Fatalf("unmarshal native detail: %v", err)
	}
	findings := detail.GetFields()["findings"].GetListValue().GetValues()
	if len(findings) == 0 {
		t.Fatalf("native findings empty: %#v", detail.AsMap())
	}
	first := findings[0].GetStructValue().AsMap()
	if first["code"] == "" || first["remediation"] == "" || first["gating"] != true {
		t.Fatalf("native finding missing rich fields: %#v", first)
	}
}

func TestPreviewFixReturnsSearchHubCandidatesWithoutWriting(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", `{
  "version":"",
  "providers":[{
    "provider_id":"demo.docs",
    "provider_group":"demo",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Docs",
    "scope":"SCOPE_PROJECT",
    "endpoint":{"http_json":{"scenario_id":"demo","path":"/search","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_COSINE_0_1"},
    "tests":{"cases":[{"id":"case","query":"docs","expect_ids":["doc-1"]}]}
  }]
}`)
	h := handler.NewConnectHandler(handler.Deps{Validator: internalvalidation.New(root)})

	resp, err := h.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("PreviewFix: %v", err)
	}
	if resp.Msg.GetApplied() {
		t.Fatal("preview must not report applied")
	}
	if got := len(resp.Msg.GetCandidates()); got != 2 {
		t.Fatalf("candidates = %d, want version and eval fixes: %#v", got, resp.Msg.GetCandidates())
	}
	raw, err := os.ReadFile(filepath.Join(root, "scenarios", "demo", ".vrooli", "search.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"version": "1.0.0"`) {
		t.Fatal("PreviewFix wrote the descriptor")
	}
}

func TestApplyFixWritesAndThenReportsNoOp(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", `{
  "version":"",
  "providers":[{
    "provider_id":"demo.docs",
    "provider_group":"demo",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Docs",
    "scope":"SCOPE_PROJECT",
    "endpoint":{"http_json":{"scenario_id":"demo","path":"/search","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_COSINE_0_1"},
    "tests":{"cases":[{"id":"case","query":"docs","expect_ids":["doc-1"]}]}
  }]
}`)
	h := handler.NewConnectHandler(handler.Deps{Validator: internalvalidation.New(root)})

	resp, err := h.ApplyFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ApplyFix: %v", err)
	}
	if !resp.Msg.GetApplied() {
		t.Fatal("apply must report applied")
	}
	for _, candidate := range resp.Msg.GetCandidates() {
		if !candidate.GetApplied() {
			t.Fatalf("candidate not marked applied: %#v", candidate)
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, "scenarios", "demo", ".vrooli", "search.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"version": "1.0.0"`) || !strings.Contains(string(raw), `"suite_id": "demo.docs.primary"`) {
		t.Fatalf("ApplyFix missing expected descriptor changes:\n%s", string(raw))
	}

	again, err := h.ApplyFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ApplyFix second run: %v", err)
	}
	if len(again.Msg.GetCandidates()) != 0 || len(again.Msg.GetMessages()) == 0 {
		t.Fatalf("second apply should report no fixes with a message: %#v", again.Msg)
	}
}

func mustLoadSpec(t *testing.T) *assessment.Spec {
	t.Helper()
	spec, err := assessment.LoadSpecFromScenario("../../..")
	if err != nil {
		t.Fatalf("LoadSpecFromScenario: %v", err)
	}
	return spec
}

func writeSearchConfig(t *testing.T, root, scenario, content string) {
	t.Helper()
	dir := filepath.Join(root, "scenarios", scenario, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "search.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func cleanSearchConfig(scenario string) string {
	return `{
  "version":"1.0.0",
  "providers":[{
    "provider_id":"` + scenario + `.docs",
    "provider_group":"` + scenario + `",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Docs",
    "scope":"SCOPE_PROJECT",
    "class":"local_index",
    "endpoint":{"http_json":{"scenario_id":"` + scenario + `","path":"/search","method":"HTTP_METHOD_POST"}},
    "status_endpoint":{"http_json":{"scenario_id":"` + scenario + `","path":"/status","method":"HTTP_METHOD_POST"}},
    "reindex_endpoint":{"http_json":{"scenario_id":"` + scenario + `","path":"/reindex","method":"HTTP_METHOD_POST"}},
    "config_endpoint":{"http_json":{"scenario_id":"` + scenario + `","path":"/config","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_COSINE_0_1"},
    "tests":{"description":"Primary corpus","cases":[
      {"id":"case","query":"docs","expect_ids":["doc-1"]},
      {"id":"neg","query":"zzqxwv nonsense","expect_no_strong_hit":true,"expect_max_score":0.2}
    ]}
  }]
}`
}
