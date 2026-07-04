package phases

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/runnability"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// scenarioRoot resolves scenarios/test-genie from this test file's location so
// doc-existence checks don't depend on the working directory.
func scenarioRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// .../scenarios/test-genie/api/internal/orchestrator/phases/<file>
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

// TestPresetsResolveAgainstCatalog is the anti-drift guard for presets: every
// phase referenced by a built-in preset must exist in the catalog.
func TestPresetsResolveAgainstCatalog(t *testing.T) {
	if err := ValidatePresets(DefaultCatalog()); err != nil {
		t.Fatalf("default presets must resolve against the catalog: %v", err)
	}
	valid := make(map[string]struct{})
	for _, n := range ValidPhaseNames() {
		valid[n] = struct{}{}
	}
	for preset, phases := range DefaultPresets() {
		for _, p := range phases {
			if _, ok := valid[p]; !ok {
				t.Errorf("preset %q references unknown phase %q", preset, p)
			}
		}
	}
}

func TestCatalogSourceDoesNotDuplicateProviderMetadata(t *testing.T) {
	dir := phasePackageDir(t)
	path := filepath.Join(dir, "catalog.go")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read catalog.go: %v", err)
	}
	source := string(raw)
	for _, forbidden := range []string{
		"ProviderScenario:",
		"delegatedSpec(Delegated{",
		"ValidationProviderSpec(Delegated{",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("catalog.go contains %q; provider-backed phase metadata must live in provider descriptors", forbidden)
		}
	}
}

func TestMergePresetsPrecedenceAndFiltering(t *testing.T) {
	allowed := map[string]struct{}{
		Structure.String(): {},
		Unit.String():      {},
		Docs.String():      {},
	}
	defaults := map[string][]string{
		"quick": {"structure", "unit"},
		"full":  {"structure", "docs"},
	}
	fileOverrides := map[string][]string{
		"quick": {"docs", "missing", "docs"},
		"file":  {"unit"},
	}
	configOverrides := map[string][]string{
		"quick": {"structure"},
		"file":  {},
	}

	got := MergePresets(defaults, fileOverrides, configOverrides, allowed)
	if phases := strings.Join(got["quick"], ","); phases != "structure" {
		t.Fatalf("quick preset = %q, want config override to replace file/default", phases)
	}
	if _, ok := got["file"]; ok {
		t.Fatalf("empty config override should delete file preset, got %v", got["file"])
	}
	if phases := strings.Join(got["full"], ","); phases != "structure,docs" {
		t.Fatalf("full preset = %q, want defaults to fill missing preset", phases)
	}
}

func TestCuratedPresetsIncludeProto(t *testing.T) {
	presets := DefaultPresets()
	for _, preset := range []Preset{PresetQuick, PresetSmoke, PresetArchitectureAudit} {
		names, ok := presets[preset.String()]
		if !ok {
			t.Fatalf("preset %q missing from DefaultPresets", preset)
		}
		found := false
		for _, name := range names {
			if name == Proto.String() {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("preset %q must include %q for proto contract feedback, got %v", preset, Proto, names)
		}
	}
}

// TestCapabilityManifestCoversEveryPhase is the anti-drift guard for the
// runnability capability manifest. Every catalog phase must carry a manifest
// whose Phase/Optional mirror the spec, and the surface declarations are pinned
// to the behavior the old hand-maintained runtimeNeeds switch encoded — so a
// future capability edit that silently changes which phases need UI/API breaks
// the build instead of changing runtime behavior unnoticed.
func TestCapabilityManifestCoversEveryPhase(t *testing.T) {
	// Pinned expectations transcribed from the pre-refactor runtimeNeeds switch
	// (ui-health/workflow/performance → UI) plus the workflow
	// DB-isolation/lifecycle-mutation contract.
	type want struct {
		ui, api, mutates, deferred bool
		dbiso                      runnability.DBIsolation
	}
	expected := map[Name]want{
		UIHealth:    {ui: true},
		Performance: {ui: true},
		Workflow:    {ui: true, mutates: true, deferred: true, dbiso: runnability.DBIsolationRouted},
	}

	catalog := DefaultCatalog()
	for _, spec := range catalog.All() {
		caps := spec.Capabilities
		if caps.Phase != spec.Name.String() {
			t.Errorf("phase %q: Capabilities.Phase = %q, want lockstep with spec name", spec.Name, caps.Phase)
		}
		if caps.Optional != spec.Optional {
			t.Errorf("phase %q: Capabilities.Optional = %v, want %v (spec)", spec.Name, caps.Optional, spec.Optional)
		}
		w := expected[spec.Name] // zero value = static phase with no surface
		if caps.NeedsUI != w.ui || caps.NeedsAPI != w.api {
			t.Errorf("phase %q surfaces: NeedsUI=%v NeedsAPI=%v, want UI=%v API=%v",
				spec.Name, caps.NeedsUI, caps.NeedsAPI, w.ui, w.api)
		}
		if caps.MutatesLifecycle != w.mutates {
			t.Errorf("phase %q: MutatesLifecycle=%v, want %v", spec.Name, caps.MutatesLifecycle, w.mutates)
		}
		if caps.LifecycleDecisionDeferred != w.deferred {
			t.Errorf("phase %q: LifecycleDecisionDeferred=%v, want %v", spec.Name, caps.LifecycleDecisionDeferred, w.deferred)
		}
		if caps.DBIsolation != w.dbiso {
			t.Errorf("phase %q: DBIsolation=%v, want %v", spec.Name, caps.DBIsolation, w.dbiso)
		}
	}
}

func TestSkipEnvVarsPreservePublishedNames(t *testing.T) {
	expected := map[Name]string{
		Structure:    "TEST_GENIE_SKIP_STRUCTURE",
		Contracts:    "TEST_GENIE_SKIP_CONTRACTS",
		UIHealth:     "TEST_GENIE_SKIP_UI_HEALTH",
		API:          "TEST_GENIE_SKIP_API",
		Architecture: "TEST_GENIE_SKIP_ARCHITECTURE",
		Dependencies: "TEST_GENIE_SKIP_DEPENDENCIES",
		Quality:      "TEST_GENIE_SKIP_QUALITY",
		Docs:         "TEST_GENIE_SKIP_DOCS",
		Unit:         "TEST_GENIE_SKIP_UNIT",
		Storage:      "TEST_GENIE_SKIP_STORAGE",
		Workflow:     "TEST_GENIE_SKIP_WORKFLOW",
		Business:     "TEST_GENIE_SKIP_BUSINESS",
		Performance:  "TEST_GENIE_SKIP_PERFORMANCE",
		Tidiness:     "TEST_GENIE_SKIP_TIDINESS",
		Security:     "TEST_GENIE_SKIP_SECURITY",
		Measures:     "TEST_GENIE_SKIP_MEASURES",
		Proto:        "TEST_GENIE_SKIP_PROTO",
		Branding:     "TEST_GENIE_SKIP_BRANDING",
		Search:       "TEST_GENIE_SKIP_SEARCH",
	}
	catalog := DefaultCatalog()
	for _, spec := range catalog.All() {
		want, ok := expected[spec.Name]
		if !ok {
			t.Fatalf("phase %q missing skip env-var expectation", spec.Name)
		}
		if spec.SkipEnvVar != want {
			t.Errorf("phase %q SkipEnvVar = %q, want %q", spec.Name, spec.SkipEnvVar, want)
		}
	}
	if len(expected) != len(catalog.All()) {
		t.Fatalf("skip env-var expectations = %d, catalog phases = %d", len(expected), len(catalog.All()))
	}
}

// TestFindingSourceCoversEveryProducingPhase is the anti-drift guard for the
// per-phase finding-source tokens that the combined findings artifact carries
// and a campaign reaudit derives covered-sources from. Every finding-producing
// phase MUST declare a non-UNSPECIFIED FindingSource; non-producing phases
// (unit, smoke, …) MUST leave it UNSPECIFIED so they never contribute a
// phantom source to reaudit coverage. The expected map is pinned to the
// producer set so adding a finding-emitting phase without a source breaks the
// build instead of silently producing un-attributed findings.
func TestFindingSourceCoversEveryProducingPhase(t *testing.T) {
	producing := map[Name]architecturev1.FindingSource{
		Structure:    architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE,
		Contracts:    architecturev1.FindingSource_FINDING_SOURCE_CLI,
		UIHealth:     architecturev1.FindingSource_FINDING_SOURCE_UI,
		API:          architecturev1.FindingSource_FINDING_SOURCE_STANDARDS,
		Quality:      architecturev1.FindingSource_FINDING_SOURCE_STANDARDS,
		Architecture: architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE,
		Dependencies: architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY,
		Docs:         architecturev1.FindingSource_FINDING_SOURCE_DOCS,
		Business:     architecturev1.FindingSource_FINDING_SOURCE_BUSINESS,
		// After the hard cutover the unit phase delegates to unit-health and
		// emits coverage findings into the COVERAGE channel (the separate
		// `coverage` phase is retired), so it is now a COVERAGE producer.
		Unit:     architecturev1.FindingSource_FINDING_SOURCE_COVERAGE,
		Tidiness: architecturev1.FindingSource_FINDING_SOURCE_TIDINESS,
		Security: architecturev1.FindingSource_FINDING_SOURCE_SECURITY,
		Measures: architecturev1.FindingSource_FINDING_SOURCE_MEASURES,
		Proto:    architecturev1.FindingSource_FINDING_SOURCE_PROTO,
		Storage:  architecturev1.FindingSource_FINDING_SOURCE_STORAGE,
		Workflow: architecturev1.FindingSource_FINDING_SOURCE_WORKFLOW,
		Branding: architecturev1.FindingSource_FINDING_SOURCE_BRANDING,
	}
	catalog := DefaultCatalog()
	for _, spec := range catalog.All() {
		want, isProducer := producing[spec.Name]
		if isProducer {
			if spec.FindingSource != want {
				t.Errorf("phase %q: FindingSource = %v, want %v", spec.Name, spec.FindingSource, want)
			}
			continue
		}
		if spec.FindingSource != architecturev1.FindingSource_FINDING_SOURCE_UNSPECIFIED {
			t.Errorf("non-producing phase %q declares FindingSource %v, want UNSPECIFIED", spec.Name, spec.FindingSource)
		}
	}
}

// TestDocPathsCoverEveryCatalogPhase is the anti-drift guard for documentation:
// every catalog phase resolves to a doc path that exists on disk, and unknown
// phases resolve to nothing.
func TestDocPathsCoverEveryCatalogPhase(t *testing.T) {
	root := scenarioRoot(t)
	for _, name := range ValidPhaseNames() {
		docs := DocPaths(name)
		if len(docs) == 0 {
			t.Errorf("phase %q has no documentation path", name)
			continue
		}
		for _, rel := range docs {
			abs := filepath.Join(root, strings.TrimPrefix(rel, "scenarios/test-genie/"))
			if _, err := os.Stat(abs); err != nil {
				t.Errorf("phase %q doc %q missing on disk (%s): %v", name, rel, abs, err)
			}
		}
	}
	if got := DocPaths("nonexistent-phase"); got != nil {
		t.Errorf("DocPaths(nonexistent) = %v, want nil", got)
	}
}

func TestGeneratedPhaseDocsMatchCommitted(t *testing.T) {
	root := scenarioRoot(t)
	assertGeneratedDoc(t, filepath.Join(root, "docs", "phases", "README.md"), RenderPhasesMarkdown(DefaultCatalog()))
	assertGeneratedDoc(t, filepath.Join(root, "docs", "reference", "presets.md"), RenderPresetsMarkdown(DefaultCatalog()))
}

func assertGeneratedDoc(t *testing.T, path string, want string) {
	t.Helper()
	if os.Getenv("UPDATE_TEST_GENIE_DOCS") == "1" {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatalf("update generated doc %s: %v", path, err)
		}
		return
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated doc %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("%s is out of date; run UPDATE_TEST_GENIE_DOCS=1 go test ./internal/orchestrator/phases -run TestGeneratedPhaseDocsMatchCommitted", path)
	}
}

func TestTestingSchemaPhasePropertiesMatchCatalog(t *testing.T) {
	root := scenarioRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "schemas", "testing.schema.json"))
	if err != nil {
		t.Fatalf("read testing schema: %v", err)
	}
	var schema struct {
		Properties map[string]struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("parse testing schema: %v", err)
	}
	phaseBlock, ok := schema.Properties["phases"]
	if !ok {
		t.Fatal("testing schema missing properties.phases")
	}
	if len(phaseBlock.Properties) == 0 {
		t.Fatal("testing schema properties.phases.properties is empty")
	}
	want := ValidPhaseNames()
	if len(phaseBlock.Properties) != len(want) {
		t.Fatalf("testing schema phase property count = %d, want %d (%v)", len(phaseBlock.Properties), len(want), want)
	}
	for _, name := range want {
		if _, ok := phaseBlock.Properties[name]; !ok {
			t.Errorf("testing schema missing phases.%s property", name)
		}
	}
}

func TestTestingSchemaDefinesUnitPolicyProfile(t *testing.T) {
	root := scenarioRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "schemas", "testing.schema.json"))
	if err != nil {
		t.Fatalf("read testing schema: %v", err)
	}
	var schema struct {
		Properties map[string]struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"properties"`
		Definitions map[string]json.RawMessage `json:"definitions"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("parse testing schema: %v", err)
	}
	unitBlock, ok := schema.Properties["unit"]
	if !ok {
		t.Fatal("testing schema missing properties.unit")
	}
	var policyRef struct {
		Ref string `json:"$ref"`
	}
	if err := json.Unmarshal(unitBlock.Properties["policy_profile"], &policyRef); err != nil {
		t.Fatalf("parse unit.policy_profile schema ref: %v", err)
	}
	if policyRef.Ref != "#/definitions/unit_policy_profile" {
		t.Fatalf("unit.policy_profile ref = %q, want #/definitions/unit_policy_profile", policyRef.Ref)
	}
	for _, name := range []string{
		"unit_policy_profile",
		"unit_required_role",
		"unit_policy_class",
		"unit_policy_customization",
		"unit_policy_waiver",
	} {
		if _, ok := schema.Definitions[name]; !ok {
			t.Errorf("testing schema missing definitions.%s", name)
		}
	}
}

func TestReactViteTemplateDeclaresUnitPolicyProfile(t *testing.T) {
	root := scenarioRoot(t)
	templateTestingPath := filepath.Clean(filepath.Join(root, "..", "..", "templates", "scenarios", "react-vite", ".vrooli", "testing.json"))
	raw, err := os.ReadFile(templateTestingPath)
	if err != nil {
		t.Fatalf("read react-vite testing.json: %v", err)
	}
	var doc struct {
		Unit struct {
			PolicyProfile struct {
				Version       string            `json:"version"`
				Template      map[string]string `json:"template"`
				RequiredRoles []struct {
					Role        string `json:"role"`
					PolicyClass string `json:"policy_class"`
				} `json:"required_roles"`
				PolicyClasses map[string]json.RawMessage `json:"policy_classes"`
				Customization struct {
					Mode string `json:"mode"`
				} `json:"customization"`
			} `json:"policy_profile"`
		} `json:"unit"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse react-vite testing.json: %v", err)
	}
	if doc.Unit.PolicyProfile.Version != "1.0.0" {
		t.Fatalf("policy profile version = %q, want 1.0.0", doc.Unit.PolicyProfile.Version)
	}
	if doc.Unit.PolicyProfile.Template["id"] != "react-vite" {
		t.Fatalf("template id = %q, want react-vite", doc.Unit.PolicyProfile.Template["id"])
	}
	wantRoles := map[string]string{"api": "go_service", "cli": "go_cli", "ui": "react_vite_ui"}
	for _, role := range doc.Unit.PolicyProfile.RequiredRoles {
		if wantRoles[role.Role] != role.PolicyClass {
			t.Errorf("unexpected required role mapping %s -> %s", role.Role, role.PolicyClass)
		}
		delete(wantRoles, role.Role)
	}
	for role := range wantRoles {
		t.Errorf("react-vite policy profile missing required role %s", role)
	}
	for _, class := range []string{"go_service", "go_cli", "react_vite_ui"} {
		if _, ok := doc.Unit.PolicyProfile.PolicyClasses[class]; !ok {
			t.Errorf("react-vite policy profile missing policy class %s", class)
		}
	}
	if doc.Unit.PolicyProfile.Customization.Mode != "monotonic" {
		t.Fatalf("customization mode = %q, want monotonic", doc.Unit.PolicyProfile.Customization.Mode)
	}
}

func TestMaturityGoPhaseArtifactMatchesCatalog(t *testing.T) {
	root := scenarioRoot(t)
	path := filepath.Clean(filepath.Join(root, "..", "..", "packages", "maturity-go", "dimensions", "testdata", "testgenie_phase_names.json"))
	payload := struct {
		Source string   `json:"source"`
		Phases []string `json:"phases"`
	}{
		Source: "test-genie/api/internal/orchestrator/phases.ValidPhaseNames",
		Phases: ValidPhaseNames(),
	}
	want, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal phase artifact: %v", err)
	}
	want = append(want, '\n')
	if os.Getenv("UPDATE_TEST_GENIE_DOCS") == "1" {
		if err := os.WriteFile(path, want, 0o644); err != nil {
			t.Fatalf("update phase artifact %s: %v", path, err)
		}
		return
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read phase artifact: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is out of date; run UPDATE_TEST_GENIE_DOCS=1 go test ./internal/orchestrator/phases -run TestMaturityGoPhaseArtifactMatchesCatalog", path)
	}
}
