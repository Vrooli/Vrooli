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

func TestEvaluateHostOSApplicability(t *testing.T) {
	declaration := providerdescriptor.Applicability{
		Default: "not_applicable",
		Any:     []providerdescriptor.Predicate{{HostOS: "windows"}},
	}

	got := Evaluate("windows-only", declaration, Context{HostOS: "linux"})
	if got.Status != StatusNotApplicable || !hasReason(got, "applicability.host_os_mismatched") {
		t.Fatalf("non-native host OS result = %#v, want not_applicable with mismatch reason", got)
	}

	got = Evaluate("windows-only", declaration, Context{HostOS: "windows"})
	if got.Status != StatusApplies || !hasReason(got, "applicability.host_os_matched") {
		t.Fatalf("native host OS result = %#v, want applies with match reason", got)
	}
}

func TestEvaluateRejectsUndeclaredTargetKindBeforePredicatesAndDefault(t *testing.T) {
	got := Evaluate("provider", providerdescriptor.Applicability{
		Default: "applies",
		Any:     []providerdescriptor.Predicate{{TargetKind: "package"}},
	}, Context{TargetKind: "resource", DeclaredTargetKinds: []string{"package"}})
	if got.Status != StatusNotApplicable {
		t.Fatalf("status = %s, want not_applicable", got.Status)
	}
	if len(got.Reasons) != 1 || got.Reasons[0].Code != CodeTargetKindUndeclared {
		t.Fatalf("reasons = %#v, want target-kind gate", got.Reasons)
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

func TestEvaluateDependencyGlobAndTagPredicates(t *testing.T) {
	tests := []struct {
		name       string
		predicate  providerdescriptor.Predicate
		ctx        Context
		wantStatus Status
		wantCode   string
	}{
		{
			name: "enabled dependency applies", predicate: providerdescriptor.Predicate{ScenarioDependency: "agent-manager"},
			ctx:        Context{ScenarioDependencies: map[string]DependencyStatus{"agent-manager": DependencyPresent}},
			wantStatus: StatusApplies, wantCode: "applicability.scenario_dependency_present",
		},
		{
			name: "disabled dependency is distinct", predicate: providerdescriptor.Predicate{ScenarioDependency: "agent-manager"},
			ctx:        Context{ScenarioDependencies: map[string]DependencyStatus{"agent-manager": DependencyDisabled}},
			wantStatus: StatusNotApplicable, wantCode: "applicability.scenario_dependency_disabled",
		},
		{
			name: "requested glob applies with matches", predicate: providerdescriptor.Predicate{PathGlob: ".vrooli/agent-profiles/*.json"},
			ctx:        Context{PathGlobs: map[string][]string{".vrooli/agent-profiles/*.json": {".vrooli/agent-profiles/default.json"}}},
			wantStatus: StatusApplies, wantCode: "applicability.path_glob_matched",
		},
		{
			name: "tag is distinct from capability", predicate: providerdescriptor.Predicate{ServiceTag: "agentic"},
			ctx:        Context{ServiceTags: map[string]bool{"agentic": true}},
			wantStatus: StatusApplies, wantCode: "applicability.service_tag_present",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate("generic", providerdescriptor.Applicability{Default: "not_applicable", Any: []providerdescriptor.Predicate{tc.predicate}}, tc.ctx)
			if got.Status != tc.wantStatus || !hasReason(got, tc.wantCode) {
				t.Fatalf("result = %#v, want status %s and code %s", got, tc.wantStatus, tc.wantCode)
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
