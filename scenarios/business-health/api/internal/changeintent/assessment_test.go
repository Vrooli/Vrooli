package changeintent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	base := filepath.Join(root, "scenarios", "business-health")
	write := func(path, content string) {
		t.Helper()
		full := filepath.Join(base, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("requirements/index.json", `{"imports":["01-search/module.json"]}`)
	write("requirements/01-search/module.json", `{
  "requirements": [{"id":"BH-SRCH-001","title":"Intent search corpus","status":"planned","description":"Search requirement records with source identity.","tags":["intent"]}]
}`)
	write("api/handler.go", "package api\n// [REQ:BH-SRCH-001] candidate source reference\n")
	write("api/handler_test.go", "package api\n// [REQ:BH-SRCH-001] test reference only\n")
	return root
}

func TestAssessExplicitRelationAndExactSourceProvenance(t *testing.T) {
	root := fixture(t)
	response, err := Assess(root, Request{Scenario: "business-health", Intent: "Change BH-SRCH-001", Limit: 99})
	if err != nil {
		t.Fatal(err)
	}
	if response.Scope == "" || len(response.Results) != 1 {
		t.Fatalf("unexpected scope/results: %#v", response)
	}
	result := response.Results[0]
	if result.Scenario != "business-health" {
		t.Fatalf("relation lost scenario identity: %#v", result)
	}
	if result.Relation != "support" || result.RelationBasis != "explicit_requirement_id_reference_in_statement" {
		t.Fatalf("explicit link state: %#v", result)
	}
	if result.Requirement.Path != "requirements/01-search/module.json" || result.Requirement.Line != 2 || len(result.Requirement.SHA256) != 64 {
		t.Fatalf("requirement provenance: %#v", result.Requirement)
	}
	if len(result.Anchors) != 2 || result.Anchors[0].Attestation != "REFERENCE_ONLY" || result.Anchors[0].Path != "api/handler.go" || result.Anchors[1].Kind != "test_reference" {
		t.Fatalf("source anchors: %#v", result.Anchors)
	}
	if len(response.Limitations) == 0 || !strings.Contains(strings.Join(response.Limitations, " "), "not mean the requirement is implemented") {
		t.Fatalf("missing limitation: %#v", response.Limitations)
	}
	if !strings.Contains(strings.Join(response.Degradations, " "), "requested_limit_capped") {
		t.Fatalf("limit cap was not surfaced: %#v", response.Degradations)
	}
}

func TestAssessKeepsLexicalMatchUnknownAndOnlyMarksExplicitConflict(t *testing.T) {
	root := fixture(t)
	unknown, err := Assess(root, Request{Scenario: "business-health", Intent: "Search intent corpus requirement records"})
	if err != nil {
		t.Fatal(err)
	}
	if len(unknown.Results) != 1 || unknown.Results[0].Relation != "unknown" {
		t.Fatalf("lexical match overstated: %#v", unknown.Results)
	}
	conflict, err := Assess(root, Request{Scenario: "business-health", Intent: "This contradicts BH-SRCH-001"})
	if err != nil {
		t.Fatal(err)
	}
	if len(conflict.Results) != 1 || conflict.Results[0].Relation != "contradiction" {
		t.Fatalf("explicit conflict not classified: %#v", conflict.Results)
	}
}

func TestAssessChangeIntentEvidenceMatrix(t *testing.T) {
	root := fixture(t)
	cases := []struct {
		name     string
		intent   string
		wantRows int
		want     string
	}{
		{name: "exact-id-support-scope", intent: "BH-SRCH-001", wantRows: 1, want: "support"},
		{name: "exact-contradiction", intent: "This contradicts BH-SRCH-001", wantRows: 1, want: "contradiction"},
		{name: "exact-conflict", intent: "This conflicts with BH-SRCH-001", wantRows: 1, want: "contradiction"},
		{name: "negated-contradiction", intent: "This does not contradict BH-SRCH-001", wantRows: 1, want: "support"},
		{name: "qualified-negation-is-not-contradiction", intent: "It is not the case that this contradicts BH-SRCH-001", wantRows: 1, want: "support"},
		{name: "id-suffix-is-not-a-match", intent: "BH-SRCH-001A", wantRows: 0},
		{name: "id-extension-is-not-a-match", intent: "BH-SRCH-001-999", wantRows: 0},
		{name: "lexical-only-remains-unknown", intent: "Search intent corpus requirement records", wantRows: 1, want: "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response, err := Assess(root, Request{Scenario: "business-health", Intent: tc.intent})
			if err != nil {
				t.Fatal(err)
			}
			if len(response.Results) != tc.wantRows {
				t.Fatalf("rows=%d want %d: %#v", len(response.Results), tc.wantRows, response.Results)
			}
			if tc.wantRows == 1 && response.Results[0].Relation != tc.want {
				t.Fatalf("relation=%q want %q", response.Results[0].Relation, tc.want)
			}
			if tc.wantRows == 1 && (response.Results[0].Scenario != "business-health" || response.Results[0].Requirement.Path == "" || response.Results[0].Requirement.Line < 1 || len(response.Results[0].Requirement.SHA256) != 64) {
				t.Fatalf("relation lacks source or scenario identity: %#v", response.Results[0])
			}
		})
	}
}

func TestAssessRejectsAmbiguousRequirementIdentity(t *testing.T) {
	root := fixture(t)
	path := filepath.Join(root, "scenarios", "business-health", "requirements", "01-search", "module.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	duplicate := strings.Replace(string(content), `}]`, `}, {"id":"BH-SRCH-001","title":"Duplicate","status":"planned","description":"ambiguous","tags":[]}]`, 1)
	if err := os.WriteFile(path, []byte(duplicate), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Assess(root, Request{Scenario: "business-health", Intent: "BH-SRCH-001"}); err == nil || !strings.Contains(err.Error(), "duplicate ID") {
		t.Fatalf("ambiguous requirement identity was not rejected: %v", err)
	}
}

func TestAssessFailsClosedOnMalformedRequirementModule(t *testing.T) {
	root := fixture(t)
	path := filepath.Join(root, "scenarios", "business-health", "requirements", "01-search", "module.json")
	if err := os.WriteFile(path, []byte(`{"requirements":[`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Assess(root, Request{Scenario: "business-health", Intent: "BH-SRCH-001"}); err == nil || !strings.Contains(err.Error(), "parse requirements module") {
		t.Fatalf("malformed requirement parser input was not an explicit failure: %v", err)
	}
}

func TestAssessBoundsAndScenarioScope(t *testing.T) {
	root := fixture(t)
	if _, err := Assess(root, Request{Scenario: "another-scenario", Intent: "change"}); err == nil {
		t.Fatal("unsupported scenario accepted")
	}
	if _, err := Assess(root, Request{Scenario: "business-health", Intent: strings.Repeat("x", maxIntentBytes+1)}); err == nil {
		t.Fatal("oversized statement accepted")
	}
	if _, err := Assess(root, Request{Scenario: "business-health", Intent: "BH-SRCH-001"}); err != nil {
		t.Fatal(err)
	}
}
