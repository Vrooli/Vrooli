package phases

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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

func TestRepositoryDescriptorsOwnDisplayNames(t *testing.T) {
	for _, spec := range DefaultCatalog().All() {
		if strings.TrimSpace(spec.DisplayName) == "" {
			t.Fatalf("phase %q missing descriptor-owned displayName", spec.Name)
		}
		for _, descriptor := range DefaultCatalog().Descriptors() {
			if descriptor.Name == spec.Name.String() && descriptor.DisplayName != spec.DisplayName {
				t.Fatalf("phase %q descriptor displayName = %q, want %q", spec.Name, descriptor.DisplayName, spec.DisplayName)
			}
		}
	}
}

func TestMergePresetsPrecedenceAndFiltering(t *testing.T) {
	allowed := map[string]struct{}{
		Name("structure").String(): {},
		Name("unit").String():      {},
		Name("docs").String():      {},
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
	for _, preset := range []Preset{PresetArchitectureAudit} {
		names, ok := presets[preset.String()]
		if !ok {
			t.Fatalf("preset %q missing from DefaultPresets", preset)
		}
		found := false
		for _, name := range names {
			if name == Name("proto").String() {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("preset %q must include %q for proto contract feedback, got %v", preset, Name("proto"), names)
		}
	}
}

func TestQuickAndSmokeAreNotFixedPhaseBundles(t *testing.T) {
	presets := DefaultPresets()
	for _, preset := range []Preset{PresetQuick, PresetSmoke} {
		if _, ok := presets[preset.String()]; ok {
			t.Fatalf("preset %q must be planned through AdaptiveProfile, not DefaultPresets fixed membership", preset)
		}
		if _, ok := AdaptiveProfile(preset.String()); !ok {
			t.Fatalf("preset %q missing adaptive profile definition", preset)
		}
	}
}

func TestQuickAndSmokeHaveAdaptiveProfileDefinitions(t *testing.T) {
	for _, preset := range []Preset{PresetQuick, PresetSmoke} {
		profile, ok := AdaptiveProfile(preset.String())
		if !ok {
			t.Fatalf("preset %q must have an adaptive profile definition", preset)
		}
		if profile.Name != preset {
			t.Fatalf("profile name = %q, want %q", profile.Name, preset)
		}
		if profile.BudgetSeconds <= 0 {
			t.Fatalf("profile %q budget must be positive, got %d", preset, profile.BudgetSeconds)
		}
		if strings.TrimSpace(profile.Strategy) == "" {
			t.Fatalf("profile %q strategy must be declared", preset)
		}
	}
}

// TestCapabilityManifestCoversEveryPhase is the anti-drift guard for the
// runnability capability manifest. Provider descriptors own surface and
// lifecycle requirements; Test Genie only normalizes catalog identity and
// optionality. This guard must therefore remain phase-agnostic.
func TestCapabilityManifestCoversEveryPhase(t *testing.T) {
	catalog := DefaultCatalog()
	for _, spec := range catalog.All() {
		caps := spec.Capabilities
		if caps.Phase != spec.Name.String() {
			t.Errorf("phase %q: Capabilities.Phase = %q, want lockstep with spec name", spec.Name, caps.Phase)
		}
		if caps.Optional != spec.Optional {
			t.Errorf("phase %q: Capabilities.Optional = %v, want %v (spec)", spec.Name, caps.Optional, spec.Optional)
		}
		if spec.Delegated == nil {
			t.Errorf("phase %q missing descriptor-backed provider metadata", spec.Name)
		}
	}
}

func TestSkipEnvVarsPreservePublishedNames(t *testing.T) {
	catalog := DefaultCatalog()
	for _, spec := range catalog.All() {
		want := "TEST_GENIE_SKIP_" + strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(spec.Name.String()))
		if spec.SkipEnvVar != want {
			t.Errorf("phase %q SkipEnvVar = %q, want %q", spec.Name, spec.SkipEnvVar, want)
		}
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
		Name("structure"):    architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE,
		Name("contracts"):    architecturev1.FindingSource_FINDING_SOURCE_CLI,
		Name("ui-health"):    architecturev1.FindingSource_FINDING_SOURCE_UI,
		Name("api"):          architecturev1.FindingSource_FINDING_SOURCE_STANDARDS,
		Name("quality"):      architecturev1.FindingSource_FINDING_SOURCE_STANDARDS,
		Name("architecture"): architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE,
		Name("dependencies"): architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY,
		Name("docs"):         architecturev1.FindingSource_FINDING_SOURCE_DOCS,
		Name("business"):     architecturev1.FindingSource_FINDING_SOURCE_BUSINESS,
		// After the hard cutover the unit phase delegates to unit-health and
		// emits coverage findings into the COVERAGE channel (the separate
		// `coverage` phase is retired), so it is now a COVERAGE producer.
		Name("unit"):                      architecturev1.FindingSource_FINDING_SOURCE_COVERAGE,
		Name("tidiness"):                  architecturev1.FindingSource_FINDING_SOURCE_TIDINESS,
		Name("security"):                  architecturev1.FindingSource_FINDING_SOURCE_SECURITY,
		Name("measures"):                  architecturev1.FindingSource_FINDING_SOURCE_MEASURES,
		Name("proto"):                     architecturev1.FindingSource_FINDING_SOURCE_PROTO,
		Name("storage"):                   architecturev1.FindingSource_FINDING_SOURCE_STORAGE,
		Name("workflow"):                  architecturev1.FindingSource_FINDING_SOURCE_WORKFLOW,
		Name("branding"):                  architecturev1.FindingSource_FINDING_SOURCE_BRANDING,
		Name("experience"):                architecturev1.FindingSource_FINDING_SOURCE_UI,
		Name("event-capture-conformance"): architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE,
		// Component Tests executes the React Component Library provider and
		// emits contract-test findings into the coverage channel.
		Name("component-tests"): architecturev1.FindingSource_FINDING_SOURCE_COVERAGE,
		Name("soak"):            architecturev1.FindingSource_FINDING_SOURCE_COVERAGE,
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

func TestTestingSchemaPhaseOverridesAreRegistryDriven(t *testing.T) {
	root := scenarioRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "schemas", "testing.schema.json"))
	if err != nil {
		t.Fatalf("read testing schema: %v", err)
	}
	var schema struct {
		Properties map[string]struct {
			Properties    map[string]json.RawMessage `json:"properties"`
			PropertyNames struct {
				Pattern string `json:"pattern"`
			} `json:"propertyNames"`
			AdditionalProperties json.RawMessage `json:"additionalProperties"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("parse testing schema: %v", err)
	}
	phaseBlock, ok := schema.Properties["phases"]
	if !ok {
		t.Fatal("testing schema missing properties.phases")
	}
	if len(phaseBlock.Properties) != 0 {
		t.Fatalf("testing schema phases block must not enumerate phase keys, got %d entries", len(phaseBlock.Properties))
	}
	if phaseBlock.PropertyNames.Pattern != "^[a-z0-9]+(?:-[a-z0-9]+)*$" {
		t.Fatalf("testing schema phase key pattern = %q", phaseBlock.PropertyNames.Pattern)
	}
	var ref struct {
		Ref string `json:"$ref"`
	}
	if err := json.Unmarshal(phaseBlock.AdditionalProperties, &ref); err != nil {
		t.Fatalf("parse phases additionalProperties: %v", err)
	}
	if ref.Ref != "#/definitions/phase_options" {
		t.Fatalf("phases additionalProperties ref = %q, want #/definitions/phase_options", ref.Ref)
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
	if phasesBlock, ok := schema.Properties["phases"]; ok {
		if len(phasesBlock.Properties) != 0 {
			t.Fatalf("testing schema phases block must not enumerate phase keys, got %d entries", len(phasesBlock.Properties))
		}
	}
	unitBlock, ok := schema.Properties["unit"]
	if !ok {
		t.Fatal("testing schema missing legacy properties.unit")
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

func TestTestingSchemaKeepsAdapterVocabularyOpen(t *testing.T) {
	root := scenarioRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "schemas", "testing.schema.json"))
	if err != nil {
		t.Fatalf("read testing schema: %v", err)
	}
	var schema struct {
		Definitions map[string]struct {
			Properties map[string]struct {
				Enum []string `json:"enum"`
				Type string   `json:"type"`
			} `json:"properties"`
		} `json:"definitions"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("parse testing schema: %v", err)
	}
	packageManager := schema.Definitions["unit_policy_class"].Properties["package_manager"]
	if packageManager.Type != "string" || len(packageManager.Enum) != 0 {
		t.Fatalf("package_manager must remain an open adapter-owned string: %+v", packageManager)
	}
	projection := schema.Definitions["unit_projection_policy"].Properties["settings"]
	if projection.Type != "object" {
		t.Fatalf("projection.settings must be an opaque object: %+v", projection)
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
	if doc.Unit.PolicyProfile.Version != "2.0.0" {
		t.Fatalf("policy profile version = %q, want 2.0.0", doc.Unit.PolicyProfile.Version)
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
