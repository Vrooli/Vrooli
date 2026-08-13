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
	if got.Validation.DeliveryMode != "inline" || !got.Validation.Execution {
		t.Fatalf("validation defaults = %+v, want inline execution provider", got.Validation)
	}
	if got.MaturitySpec == nil || got.MaturitySpec.Provider != "search-hub" || got.MaturitySpec.Phase != "search" {
		t.Fatalf("embedded maturity identity not stamped: %#v", got.MaturitySpec)
	}
}

func TestLoadRejectsDescriptorWithoutTargetDeclaration(t *testing.T) {
	body := strings.Replace(validDescriptor("search-hub", "search"),
		`  "targets":{"kinds":["scenario"],"selection":"enumerate"},`+"\n", "", 1)
	result := Load(LoadOptions{Paths: []string{writeDescriptor(t, "search-hub", body)}})
	if !hasDiagnostic(result.Diagnostics, "missing_targets") {
		t.Fatalf("missing target declaration was accepted: %+v", result.Diagnostics)
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
			name:     "unknown host OS rejected",
			body:     strings.Replace(validDescriptor("search-hub", "search"), `{"fileExists":".vrooli/search.json"}`, `{"hostOS":"plan9"}`, 1),
			want:     "invalid_predicate_host_os",
			scenario: "search-hub",
		},
		{
			name:     "invalid embedded maturity",
			body:     strings.Replace(validDescriptor("search-hub", "search"), `"version":"2.0.0"`, `"version":""`, 1),
			want:     "invalid_maturity",
			scenario: "search-hub",
		},
		{
			name:     "missing docs path",
			body:     strings.Replace(validDescriptor("search-hub", "search"), `  "docs":{"path":"scenarios/test-genie/docs/phases/search/README.md"},`+"\n", "", 1),
			want:     "missing_docs_path",
			scenario: "search-hub",
		},
		{
			name: "invalid evidence kind",
			body: strings.Replace(validDescriptor("search-hub", "search"),
				`"orderHint":100,`, `"orderHint":100,"evidenceKinds":["bad kind"],`, 1),
			want:     "invalid_evidence_kind",
			scenario: "search-hub",
		},
		{
			name: "any and all are mutually exclusive",
			body: strings.Replace(validDescriptor("search-hub", "search"),
				`"applicability":{"default":"not_applicable","any":[{"fileExists":".vrooli/search.json"},{"serviceCapability":"search"}]}`,
				`"applicability":{"default":"not_applicable","any":[{"serviceTag":"search"}],"all":[{"hasAPI":true}]}`, 1),
			want: "ambiguous_applicability", scenario: "search-hub",
		},
		{
			name: "durable delivery requires execution",
			body: strings.Replace(validDescriptor("search-hub", "search"),
				`"validation":{"contract":"scenario-validation/v1","includeExecution":true}`,
				`"validation":{"contract":"scenario-validation/v1","deliveryMode":"durable-run","runService":"scenario-validation/v1.DurableValidationRunService"}`, 1),
			want: "durable_delivery_requires_execution", scenario: "search-hub",
		},
		{
			name: "durable delivery rejects legacy execution switch",
			body: strings.Replace(validDescriptor("search-hub", "search"),
				`"validation":{"contract":"scenario-validation/v1","includeExecution":true}`,
				`"validation":{"contract":"scenario-validation/v1","deliveryMode":"durable-run","execution":true,"includeExecution":true,"runService":"scenario-validation/v1.DurableValidationRunService"}`, 1),
			want: "durable_delivery_rejects_legacy_include_execution", scenario: "search-hub",
		},
		{
			name: "durable delivery requires generic run service",
			body: strings.Replace(validDescriptor("search-hub", "search"),
				`"validation":{"contract":"scenario-validation/v1","includeExecution":true}`,
				`"validation":{"contract":"scenario-validation/v1","deliveryMode":"durable-run","execution":true}`, 1),
			want: "durable_delivery_requires_run_service", scenario: "search-hub",
		},
		{
			name: "static observational determinism is explicit diagnostic",
			body: strings.Replace(validDescriptor("search-hub", "search"),
				`"source":"validation-provider",`, `"source":"validation-provider","runtimeClass":"static","determinism":{"default":"observational","reason":"reads a live service"},`, 1),
			want: "static_provider_declared_observational", scenario: "search-hub",
		},
		{
			name: "static boilerplate reason is rejected",
			body: strings.Replace(validDescriptor("search-hub", "search"),
				`"source":"validation-provider",`, `"source":"validation-provider","runtimeClass":"static","determinism":{"default":"file-determined","inputs":["**/*.go"],"reason":"Provider inputs and external observations are not proven to be completely represented by a file digest."},`, 1),
			want: "boilerplate_determinism_reason", scenario: "search-hub",
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

func TestLoadRejectsUnknownRecommendedSkill(t *testing.T) {
	path := writeDescriptor(t, "search-hub", validDescriptor("search-hub", "search"))
	result := Load(LoadOptions{Paths: []string{path}, SkillIDs: map[string]struct{}{"ecosystem-fit": {}}})
	if !hasDiagnostic(result.Diagnostics, "unknown_recommended_skill") {
		t.Fatalf("diagnostics = %#v, want unknown_recommended_skill", result.Diagnostics)
	}
}

func TestLoadAcceptsKnownRecommendedSkill(t *testing.T) {
	path := writeDescriptor(t, "search-hub", validDescriptor("search-hub", "search"))
	result := Load(LoadOptions{Paths: []string{path}, SkillIDs: map[string]struct{}{"search": {}}})
	if err := result.Err(); err != nil {
		t.Fatalf("Load returned diagnostics: %v", err)
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

func TestLoadRejectsAmbiguousLineage(t *testing.T) {
	active := writeDescriptor(t, "search-hub", validDescriptor("search-hub", "search"))
	replacementBody := strings.Replace(validDescriptor("next-search", "next-search"),
		`"orderHint":100,`, `"orderHint":100,"aliases":["search"],"supersedes":["search-v0"],`, 1)
	replacement := writeDescriptor(t, "next-search", replacementBody)

	result := Load(LoadOptions{Paths: []string{active, replacement}})
	if !hasDiagnostic(result.Diagnostics, "lineage_active_phase_collision") {
		t.Fatalf("diagnostics = %#v, want lineage_active_phase_collision", result.Diagnostics)
	}
}

func TestLoadAcceptsExplicitRetiredLineageAndEvidenceKinds(t *testing.T) {
	body := strings.Replace(validDescriptor("next-search", "next-search"),
		`"orderHint":100,`, `"orderHint":100,"aliases":["legacy-search"],"supersedes":["search-v0"],"evidenceKinds":["findings.report","trace"],`, 1)
	path := writeDescriptor(t, "next-search", body)
	result := Load(LoadOptions{Paths: []string{path}})
	if err := result.Err(); err != nil {
		t.Fatalf("explicit lineage failed: %v", err)
	}
	got := result.Descriptors[0]
	if len(got.Aliases) != 1 || len(got.Supersedes) != 1 || len(got.EvidenceKinds) != 2 {
		t.Fatalf("lineage/evidence metadata = %+v", got)
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

func TestLoadRejectsFieldsForbiddenByDescriptorSchema(t *testing.T) {
	root := t.TempDir()
	schemaRoot := filepath.Join(filepath.Clean(filepath.Join("..", "..", "..", "..", "..", "..")), "scenarios", "test-genie", "schemas")
	schema, err := os.ReadFile(filepath.Join(schemaRoot, "test-genie-phase-descriptor.schema.json"))
	if err != nil {
		t.Fatalf("read descriptor schema: %v", err)
	}
	destinationSchema := filepath.Join(root, "scenarios", "test-genie", "schemas", "test-genie-phase-descriptor.schema.json")
	if err := os.MkdirAll(filepath.Dir(destinationSchema), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destinationSchema, schema, 0o644); err != nil {
		t.Fatal(err)
	}

	descriptorPath := filepath.Join(root, "scenarios", "search-hub", ".vrooli", "test-genie.json")
	if err := os.MkdirAll(filepath.Dir(descriptorPath), 0o755); err != nil {
		t.Fatal(err)
	}
	body := strings.Replace(validDescriptor("search-hub", "search"),
		`"displayName":"Search",`, `"displayName":"Search","unexpected":true,`, 1)
	body = strings.Replace(body, "{\n  \"schemaVersion\"", "{\n  \"$schema\":\"https://vrooli.dev/schemas/test-genie-phase-descriptor.schema.json\",\n  \"schemaVersion\"", 1)
	if err := os.WriteFile(descriptorPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	result := Load(LoadOptions{RepoRoot: root})
	if !hasDiagnostic(result.Diagnostics, "schema_validation_failed") {
		t.Fatalf("diagnostics = %#v, want schema_validation_failed", result.Diagnostics)
	}
}

func TestEvidenceProducingProvidersDeclareTypedKinds(t *testing.T) { // [REQ:TESTGENIE-TYPED-EVIDENCE-P0]
	repoRoot := filepath.Clean(filepath.Join("..", "..", "..", "..", "..", ".."))
	result := Load(LoadOptions{RepoRoot: repoRoot})
	if err := result.Err(); err != nil {
		t.Fatalf("repository descriptors failed to load: %v", err)
	}
	want := map[string][]string{
		"ui-health":          {"screenshot", "visual.diff", "console", "network", "dom"},
		"workflow-health":    {"workflow.video", "trace"},
		"performance-health": {"command.output", "findings.report", "trace"},
		"unit-health":        {"command.output", "coverage.report", "findings.report"},
		"tidiness-manager":   {"findings.report"},
	}
	byScenario := make(map[string]Descriptor, len(result.Descriptors))
	for _, descriptor := range result.Descriptors {
		byScenario[descriptor.Scenario] = descriptor
	}
	for scenario, kinds := range want {
		descriptor, ok := byScenario[scenario]
		if !ok {
			t.Fatalf("missing %s descriptor", scenario)
		}
		declared := make(map[string]struct{}, len(descriptor.EvidenceKinds))
		for _, kind := range descriptor.EvidenceKinds {
			declared[kind] = struct{}{}
		}
		for _, kind := range kinds {
			if _, ok := declared[kind]; !ok {
				t.Errorf("%s evidenceKinds = %v, missing %q", scenario, descriptor.EvidenceKinds, kind)
			}
		}
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
  "targets":{"kinds":["scenario"],"selection":"enumerate"},
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
