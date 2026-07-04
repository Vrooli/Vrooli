package providerdescriptor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadValidDescriptor(t *testing.T) {
	path := writeDescriptor(t, "search-hub", validDescriptor("search-hub", "search"))

	result := Load(LoadOptions{Paths: []string{path}})
	if err := result.Err(); err != nil {
		t.Fatalf("Load returned diagnostics: %v", err)
	}
	if len(result.Descriptors) != 1 {
		t.Fatalf("descriptor count = %d, want 1", len(result.Descriptors))
	}
	got := result.Descriptors[0]
	if got.Scenario != "search-hub" || got.Phase != "search" {
		t.Fatalf("identity = %s/%s, want search-hub/search", got.Scenario, got.Phase)
	}
	if got.DisplayName != "Search" {
		t.Fatalf("displayName = %q, want Search", got.DisplayName)
	}
	if got.TimeoutValue == 0 {
		t.Fatal("timeout was not parsed")
	}
	if got.MaturitySpec == nil || got.MaturitySpec.Provider != "search-hub" || got.MaturitySpec.Phase != "search" {
		t.Fatalf("embedded maturity identity not stamped: %#v", got.MaturitySpec)
	}
}

func TestLoadRejectsDescriptorErrors(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		want     string
		scenario string
	}{
		{
			name:     "missing required",
			body:     `{}`,
			want:     "missing_scenario",
			scenario: "search-hub",
		},
		{
			name:     "invalid enum",
			body:     strings.Replace(validDescriptor("search-hub", "search"), `"selection":"default_when_applicable"`, `"selection":"sometimes"`, 1),
			want:     "invalid_selection_policy",
			scenario: "search-hub",
		},
		{
			name:     "scenario mismatch",
			body:     validDescriptor("wrong", "search"),
			want:     "scenario_mismatch",
			scenario: "search-hub",
		},
		{
			name:     "invalid timeout",
			body:     strings.Replace(validDescriptor("search-hub", "search"), `"timeout":"120s"`, `"timeout":"soon"`, 1),
			want:     "invalid_timeout",
			scenario: "search-hub",
		},
		{
			name:     "unknown predicate rejected",
			body:     strings.Replace(validDescriptor("search-hub", "search"), `{"fileExists":".vrooli/search.json"}`, `{"unknown":"value"}`, 1),
			want:     "invalid_predicate",
			scenario: "search-hub",
		},
		{
			name:     "invalid embedded maturity",
			body:     strings.Replace(validDescriptor("search-hub", "search"), `"version":"2.0.0"`, `"version":""`, 1),
			want:     "invalid_maturity",
			scenario: "search-hub",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeDescriptor(t, tc.scenario, tc.body)
			result := Load(LoadOptions{Paths: []string{path}})
			if !hasDiagnostic(result.Diagnostics, tc.want) {
				t.Fatalf("diagnostics = %#v, want code %s", result.Diagnostics, tc.want)
			}
		})
	}
}

func TestLoadRejectsDuplicatePhase(t *testing.T) {
	first := writeDescriptor(t, "search-hub", validDescriptor("search-hub", "search"))
	second := writeDescriptor(t, "other-health", validDescriptor("other-health", "search"))

	result := Load(LoadOptions{Paths: []string{first, second}})
	if !hasDiagnostic(result.Diagnostics, "duplicate_phase") {
		t.Fatalf("diagnostics = %#v, want duplicate_phase", result.Diagnostics)
	}
}

func TestLoadRejectsLeftoverMaturityJSON(t *testing.T) {
	path := writeDescriptor(t, "search-hub", validDescriptor("search-hub", "search"))
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "maturity.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	result := Load(LoadOptions{Paths: []string{path}})
	if !hasDiagnostic(result.Diagnostics, "leftover_maturity_json") {
		t.Fatalf("diagnostics = %#v, want leftover_maturity_json", result.Diagnostics)
	}
}

func TestRepositoryDescriptorsLoadWithoutRetiredMaturityFiles(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", "..", "..", "..", ".."))
	result := Load(LoadOptions{RepoRoot: repoRoot})
	if err := result.Err(); err != nil {
		t.Fatalf("repository descriptors failed to load: %v", err)
	}
	if len(result.Descriptors) < 19 {
		t.Fatalf("repository descriptor count = %d, want at least 19 provider-backed phases", len(result.Descriptors))
	}
	for _, descriptor := range result.Descriptors {
		if strings.TrimSpace(descriptor.DisplayName) == "" {
			t.Fatalf("%s descriptor missing displayName", descriptor.Path)
		}
	}
	leftovers, err := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", ".vrooli", "maturity.json"))
	if err != nil {
		t.Fatalf("glob retired maturity specs: %v", err)
	}
	if len(leftovers) > 0 {
		t.Fatalf("retired maturity specs remain: %v", leftovers)
	}
}

func writeDescriptor(t *testing.T, scenario, body string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios", scenario, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "test-genie.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func hasDiagnostic(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func validDescriptor(scenario, phase string) string {
	return `{
  "schemaVersion":"1.0.0",
  "scenario":"` + scenario + `",
  "phase":"` + phase + `",
  "displayName":"Search",
  "description":"Validates search registration.",
  "source":"validation-provider",
  "orderHint":100,
  "timeout":"120s",
  "findingSource":"search",
  "validation":{"contract":"scenario-validation/v1","includeExecution":true},
  "applicability":{"default":"not_applicable","any":[{"fileExists":".vrooli/search.json"},{"serviceCapability":"search"}]},
  "policy":{
    "selection":"default_when_applicable",
    "providerReadiness":"required_when_applicable",
    "providerLifecycle":"start_if_needed",
    "freshness":"require_live_contract",
    "resultGating":"gating",
    "unavailable":"fail"
  },
  "runnability":{"needsUI":false,"needsAPI":false,"requiredResources":[]},
  "docs":{"path":"scenarios/test-genie/docs/phases/search/README.md"},
  "maturity":{
    "version":"2.0.0",
    "capabilities":[{"id":"registration","label":"Search registration","levels":[{"id":"L0","name":"Missing","description":"Missing search registration.","entry_criteria":[],"exit_criteria":[]}]}],
    "findings":{
      "SEARCH_REGISTRATION_MISSING":{
        "capability_id":"registration",
        "local_level_impact":"L0",
        "global_impact":"capability_gap",
        "dimension":"operational-targets",
        "severity_default":"SEVERITY_ERROR",
        "recommended_skill_ids":["search"],
        "clean_requirement":"required",
        "fix_class":"manual",
        "reason":"Requires provider-specific judgment."
      }
    },
    "fallback":{"capability_id":"registration","local_level_impact":"L0","global_impact":"unknown","dimension":"operational-targets","severity_default":"SEVERITY_WARNING","clean_requirement":"advisory"}
  }
}`
}
