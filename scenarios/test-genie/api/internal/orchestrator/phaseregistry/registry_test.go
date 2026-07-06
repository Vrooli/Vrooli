package phaseregistry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/providerdescriptor"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

type testSpec struct {
	Name           string
	Source         string
	Provider       string
	FindingSource  architecturev1.FindingSource
	DefaultTimeout bool
	NeedsUI        bool
	HasPolicy      bool
}

func TestBuildDescriptorBackedProviderPhase(t *testing.T) {
	descriptor := loadDescriptor(t, "knowledge-observatory", validDescriptor("knowledge-observatory", "docs"))

	result := Build([]providerdescriptor.Descriptor{descriptor}, Options{Bindings: testBindings()})
	if hasDiagnostic(result.Diagnostics, "") {
		t.Fatalf("Build diagnostics = %#v", result.Diagnostics)
	}
	entry, ok := result.Registry.Lookup("docs")
	if !ok {
		t.Fatal("docs phase was not registered")
	}
	spec := entry.Spec.(testSpec)
	if spec.Name != "docs" {
		t.Fatalf("spec name = %q, want docs", spec.Name)
	}
	if spec.Source != SourceValidationProvider {
		t.Fatalf("source = %q, want %q", spec.Source, SourceValidationProvider)
	}
	if spec.Provider != "knowledge-observatory" {
		t.Fatalf("delegated provider = %#v, want knowledge-observatory", spec.Provider)
	}
	if spec.FindingSource != architecturev1.FindingSource_FINDING_SOURCE_DOCS {
		t.Fatalf("finding source = %v, want DOCS", spec.FindingSource)
	}
	if !spec.DefaultTimeout {
		t.Fatalf("timeout was not projected: default=%v", spec.DefaultTimeout)
	}
	if spec.NeedsUI {
		t.Fatal("docs descriptor should not require UI")
	}
	if !spec.HasPolicy {
		t.Fatal("policy was not projected")
	}
	if entry.Descriptor.Path == "" {
		t.Fatal("descriptor source path was not retained")
	}
}

func TestBuildOrdersByOrderHint(t *testing.T) {
	docs := loadDescriptor(t, "knowledge-observatory", strings.Replace(validDescriptor("knowledge-observatory", "docs"), `"orderHint":100`, `"orderHint":20`, 1))
	cliBody := strings.Replace(validDescriptor("cli-health", "contracts"), `"findingSource":"docs"`, `"findingSource":"cli"`, 1)
	cliBody = strings.Replace(cliBody, `"orderHint":100`, `"orderHint":10`, 1)
	cli := loadDescriptor(t, "cli-health", cliBody)

	result := Build([]providerdescriptor.Descriptor{docs, cli}, Options{Bindings: testBindings()})
	if hasDiagnostic(result.Diagnostics, "") {
		t.Fatalf("Build diagnostics = %#v", result.Diagnostics)
	}
	specs := result.Registry.Specs()
	if len(specs) != 2 {
		t.Fatalf("spec count = %d, want 2", len(specs))
	}
	if got := specs[0].(testSpec).Name; got != "contracts" {
		t.Fatalf("first phase = %q, want contracts", got)
	}
}

func TestBuildRejectsDuplicatePhase(t *testing.T) {
	first := loadDescriptor(t, "knowledge-observatory", validDescriptor("knowledge-observatory", "docs"))
	second := first
	second.Scenario = "other-docs"
	second.Path = filepath.Join(filepath.Dir(first.Path), "..", "..", "other-docs", ".vrooli", "test-genie.json")

	result := Build([]providerdescriptor.Descriptor{first, second}, Options{Bindings: testBindings()})
	if !hasDiagnostic(result.Diagnostics, "duplicate_phase") {
		t.Fatalf("diagnostics = %#v, want duplicate_phase", result.Diagnostics)
	}
}

func TestBuildRejectsMissingRequiredDescriptor(t *testing.T) {
	descriptor := loadDescriptor(t, "knowledge-observatory", validDescriptor("knowledge-observatory", "docs"))

	result := Build([]providerdescriptor.Descriptor{descriptor}, Options{
		Bindings:       testBindings(),
		RequiredPhases: []RequiredPhase{{Phase: "structure", ProviderScenario: "structure-health"}},
	})
	if !hasDiagnostic(result.Diagnostics, "missing_required_descriptor") {
		t.Fatalf("diagnostics = %#v, want missing_required_descriptor", result.Diagnostics)
	}
}

func TestBuildRejectsUnsupportedSource(t *testing.T) {
	descriptor := loadDescriptor(t, "knowledge-observatory", validDescriptor("knowledge-observatory", "docs"))
	descriptor.Source = "native-go"

	result := Build([]providerdescriptor.Descriptor{descriptor}, Options{Bindings: testBindings()})
	if !hasDiagnostic(result.Diagnostics, "unsupported_source") {
		t.Fatalf("diagnostics = %#v, want unsupported_source", result.Diagnostics)
	}
}

func TestBuildRejectsInvalidFindingSource(t *testing.T) {
	descriptor := loadDescriptor(t, "knowledge-observatory", validDescriptor("knowledge-observatory", "docs"))
	descriptor.FindingSource = "search"

	result := Build([]providerdescriptor.Descriptor{descriptor}, Options{Bindings: testBindings()})
	if !hasDiagnostic(result.Diagnostics, "invalid_finding_source") {
		t.Fatalf("diagnostics = %#v, want invalid_finding_source", result.Diagnostics)
	}
}

func testBindings() map[string]RunnerBinding {
	return map[string]RunnerBinding{
		SourceValidationProvider: func(descriptor providerdescriptor.Descriptor, source architecturev1.FindingSource) (any, error) {
			return testSpec{
				Name:           descriptor.Phase,
				Source:         descriptor.Source,
				Provider:       descriptor.Scenario,
				FindingSource:  source,
				DefaultTimeout: descriptor.TimeoutValue > 0,
				NeedsUI:        descriptor.Runnability.NeedsUI,
				HasPolicy:      descriptor.Policy.ProviderReadiness != "",
			}, nil
		},
	}
}

func loadDescriptor(t *testing.T, scenario, body string) providerdescriptor.Descriptor {
	t.Helper()
	path := writeDescriptor(t, scenario, body)
	result := providerdescriptor.Load(providerdescriptor.LoadOptions{Paths: []string{path}})
	if err := result.Err(); err != nil {
		t.Fatalf("descriptor load failed: %v", err)
	}
	if len(result.Descriptors) != 1 {
		t.Fatalf("descriptor count = %d, want 1", len(result.Descriptors))
	}
	return result.Descriptors[0]
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
	if code == "" {
		return len(diagnostics) > 0
	}
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
  "description":"Validates documentation health.",
  "source":"validation-provider",
  "orderHint":100,
  "timeout":"120s",
  "findingSource":"docs",
  "validation":{"contract":"scenario-validation/v1","includeExecution":true},
  "applicability":{"default":"applies"},
  "policy":{
    "selection":"default_when_applicable",
    "providerReadiness":"required_when_applicable",
    "providerLifecycle":"start_if_needed",
    "freshness":"require_live_contract",
    "resultGating":"gating",
    "unavailable":"fail"
  },
  "runnability":{"needsUI":false,"needsAPI":false,"requiredResources":[]},
  "docs":{"path":"scenarios/test-genie/docs/phases/docs/README.md"},
  "maturity":{
    "version":"2.0.0",
    "capabilities":[{"id":"content","label":"Content quality","levels":[{"id":"L0","name":"Broken","description":"Documentation health is broken.","entry_criteria":[],"exit_criteria":[]}]}],
    "findings":{
      "DOCS_BROKEN":{
        "capability_id":"content",
        "local_level_impact":"L0",
        "global_impact":"capability_gap",
        "dimension":"operational-targets",
        "severity_default":"SEVERITY_ERROR",
        "recommended_skill_ids":["documentation-health"],
        "clean_requirement":"required",
        "fix_class":"manual",
        "reason":"Requires provider-specific judgment."
      }
    },
    "fallback":{"capability_id":"content","local_level_impact":"L0","global_impact":"unknown","dimension":"operational-targets","severity_default":"SEVERITY_WARNING","clean_requirement":"advisory"}
  }
}`
}
