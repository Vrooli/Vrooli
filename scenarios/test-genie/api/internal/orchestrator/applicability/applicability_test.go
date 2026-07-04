package applicability

import (
	"testing"

	"test-genie/internal/orchestrator/providerdescriptor"
)

func TestEvaluateSearchApplicability(t *testing.T) {
	searchDeclaration := providerdescriptor.Applicability{
		Default: "not_applicable",
		Any: []providerdescriptor.Predicate{
			{FileExists: ".vrooli/search.json"},
			{ServiceCapability: "search"},
		},
	}

	tests := []struct {
		name       string
		ctx        Context
		wantStatus Status
		wantCode   string
	}{
		{
			name: "search file exists",
			ctx: Context{
				ScenarioName: "search-enabled",
				Files:        map[string]bool{".vrooli/search.json": true},
			},
			wantStatus: StatusApplies,
			wantCode:   CodeAnyPredicateMatched,
		},
		{
			name: "search capability declared",
			ctx: Context{
				ScenarioName:        "search-enabled",
				ServiceCapabilities: map[string]bool{"Search": true},
			},
			wantStatus: StatusApplies,
			wantCode:   CodeAnyPredicateMatched,
		},
		{
			name:       "non search target defaults not applicable",
			ctx:        Context{ScenarioName: "plain-scenario"},
			wantStatus: StatusNotApplicable,
			wantCode:   CodeDefaultNotApplicable,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate("search", searchDeclaration, tc.ctx)
			if got.Status != tc.wantStatus {
				t.Fatalf("status = %s, want %s; reasons=%#v", got.Status, tc.wantStatus, got.Reasons)
			}
			if !hasReason(got, tc.wantCode) {
				t.Fatalf("reasons = %#v, want code %s", got.Reasons, tc.wantCode)
			}
		})
	}
}

func TestEvaluateExplicitNonApplicableIsSurfaced(t *testing.T) {
	got := Evaluate("search", providerdescriptor.Applicability{
		Default: "not_applicable",
		Any:     []providerdescriptor.Predicate{{FileExists: ".vrooli/search.json"}},
	}, Context{ScenarioName: "plain-scenario"})

	if got.Phase != "search" {
		t.Fatalf("phase = %q, want search", got.Phase)
	}
	if got.Status != StatusNotApplicable {
		t.Fatalf("status = %s, want not_applicable", got.Status)
	}
	if !hasReason(got, "applicability.file_missing") {
		t.Fatalf("reasons = %#v, want file_missing detail for explanation surfaces", got.Reasons)
	}
}

func TestEvaluateAllPredicates(t *testing.T) {
	needsUIAndAPI := providerdescriptor.Applicability{
		Default: "not_applicable",
		All: []providerdescriptor.Predicate{
			{HasUI: boolPtr(true)},
			{HasAPI: boolPtr(true)},
			{TestingConfigSection: "workflow"},
		},
	}

	got := Evaluate("workflow", needsUIAndAPI, Context{
		HasUI:                 true,
		HasAPI:                true,
		TestingConfigSections: map[string]bool{"Workflow": true},
	})
	if got.Status != StatusApplies {
		t.Fatalf("status = %s, want applies; reasons=%#v", got.Status, got.Reasons)
	}
	if !hasReason(got, CodeAllPredicatesMatch) {
		t.Fatalf("reasons = %#v, want all_matched", got.Reasons)
	}

	got = Evaluate("workflow", needsUIAndAPI, Context{HasUI: true, HasAPI: false})
	if got.Status != StatusNotApplicable {
		t.Fatalf("status = %s, want not_applicable; reasons=%#v", got.Status, got.Reasons)
	}
	if !hasReason(got, "applicability.has_api_mismatched") {
		t.Fatalf("reasons = %#v, want has_api_mismatched", got.Reasons)
	}
}

func TestEvaluateInvalidDeclarations(t *testing.T) {
	tests := []struct {
		name        string
		declaration providerdescriptor.Applicability
		wantCode    string
	}{
		{
			name:        "invalid default",
			declaration: providerdescriptor.Applicability{Default: "maybe"},
			wantCode:    CodeInvalidDefault,
		},
		{
			name: "empty predicate",
			declaration: providerdescriptor.Applicability{
				Default: "not_applicable",
				Any:     []providerdescriptor.Predicate{{}},
			},
			wantCode: CodeInvalidPredicate,
		},
		{
			name: "multi field predicate",
			declaration: providerdescriptor.Applicability{
				Default: "not_applicable",
				Any: []providerdescriptor.Predicate{{
					FileExists:        ".vrooli/search.json",
					ServiceCapability: "search",
				}},
			},
			wantCode: CodeInvalidPredicate,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate("search", tc.declaration, Context{})
			if got.Status != StatusInvalid {
				t.Fatalf("status = %s, want invalid; reasons=%#v", got.Status, got.Reasons)
			}
			if !hasReason(got, tc.wantCode) {
				t.Fatalf("reasons = %#v, want code %s", got.Reasons, tc.wantCode)
			}
		})
	}
}

func hasReason(result Result, code string) bool {
	for _, reason := range result.Reasons {
		if reason.Code == code {
			return true
		}
	}
	return false
}

func boolPtr(value bool) *bool {
	return &value
}
