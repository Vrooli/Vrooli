package phases

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"test-genie/internal/orchestrator/providerdescriptor"
)

func TestValidationProviderRegistryCoversDelegatingCatalogPhases(t *testing.T) {
	expected := descriptorExpectations(t)
	catalog := NewDefaultCatalog(DefaultTimeout)
	assertDelegatedCountMatchesDescriptors(t, catalog, expected)
	for phase, descriptor := range expected {
		assertProviderMatchesDescriptor(t, catalog, phase, descriptor)
	}
}

func TestEveryCatalogPhaseCanProducePhasePresentation(t *testing.T) {
	expected := descriptorExpectations(t)
	catalog := NewDefaultCatalog(DefaultTimeout)
	for _, spec := range catalog.All() {
		descriptor, ok := expected[spec.Name]
		if !ok {
			t.Fatalf("%s missing provider descriptor; native catalog phases do not produce maturity standings", spec.Name)
		}
		if spec.Delegated == nil {
			t.Fatalf("%s is native; every catalog phase must delegate to a provider that returns a maturity assessment", spec.Name)
		}
		if descriptor.MaturitySpec == nil {
			t.Fatalf("%s descriptor did not parse embedded maturity spec", spec.Name)
		}
		if !declaresMaturityLadder(descriptor) {
			t.Fatalf("%s descriptor maturity declares no local or capability ladder", spec.Name)
		}
	}
}

func descriptorExpectations(t *testing.T) map[Name]providerdescriptor.Descriptor {
	t.Helper()
	repoRoot, err := defaultRepoRoot()
	if err != nil {
		t.Fatalf("defaultRepoRoot: %v", err)
	}
	load := providerdescriptor.Load(providerdescriptor.LoadOptions{RepoRoot: repoRoot})
	if err := load.Err(); err != nil {
		t.Fatalf("load provider descriptors: %v", err)
	}
	expected := map[Name]providerdescriptor.Descriptor{}
	for _, descriptor := range load.Descriptors {
		phase, ok := NormalizeName(descriptor.Phase)
		if !ok {
			t.Fatalf("%s invalid phase %q", descriptor.Path, descriptor.Phase)
		}
		expected[phase] = descriptor
	}
	return expected
}

func assertDelegatedCountMatchesDescriptors(t *testing.T, catalog *Catalog, expected map[Name]providerdescriptor.Descriptor) {
	t.Helper()
	delegatedCount := 0
	for _, spec := range catalog.All() {
		if spec.Delegated == nil {
			continue
		}
		delegatedCount++
		if _, ok := expected[spec.Name]; !ok {
			t.Fatalf("%s is delegated but missing from provider descriptors", spec.Name)
		}
	}
	if delegatedCount != len(expected) {
		t.Fatalf("delegated phase count = %d, descriptor count = %d", delegatedCount, len(expected))
	}
}

func assertProviderMatchesDescriptor(t *testing.T, catalog *Catalog, phase Name, descriptor providerdescriptor.Descriptor) {
	t.Helper()
	spec, ok := catalog.Lookup(phase.String())
	if !ok {
		t.Fatalf("%s missing from default catalog", phase)
	}
	if spec.Delegated == nil {
		t.Fatalf("%s missing delegated catalog metadata", phase)
	}
	provider := spec.Delegated.provider()
	if provider.Phase != phase.String() {
		t.Fatalf("%s provider phase = %q, want %q", phase, provider.Phase, phase.String())
	}
	if provider.ProviderScenario != descriptor.Scenario {
		t.Fatalf("%s provider = %q, want descriptor scenario %q", phase, provider.ProviderScenario, descriptor.Scenario)
	}
	if provider.Optional != spec.Optional {
		t.Fatalf("%s provider optional = %v, want catalog optional %v", phase, provider.Optional, spec.Optional)
	}
	if provider.Timeout != spec.DefaultTimeout {
		t.Fatalf("%s provider timeout = %s, want catalog timeout %s", phase, provider.Timeout, spec.DefaultTimeout)
	}
	if provider.FindingSource != spec.FindingSource {
		t.Fatalf("%s finding source = %v, want %v", phase, spec.FindingSource, provider.FindingSource)
	}
	if provider.IncludeExecution != includeExecutionPhase(phase) {
		t.Fatalf("%s provider IncludeExecution = %v, want %v", phase, provider.IncludeExecution, includeExecutionPhase(phase))
	}
}

func declaresMaturityLadder(descriptor providerdescriptor.Descriptor) bool {
	spec := descriptor.MaturitySpec
	if spec == nil {
		return false
	}
	if len(spec.Levels) > 0 {
		return true
	}
	for _, capability := range spec.Capabilities {
		if len(capability.Levels) > 0 {
			return true
		}
	}
	return false
}

// includeExecutionPhase pins which delegates request execution-mode validation:
// Unit executes the suite, Measures runs its checks, Performance benchmarks the
// Go + UI build and Lighthouse-if-UI, Contracts runs cli-health's runtime CLI
// probe, Search runs live corpus validation, UIHealth drives the BAS render
// handshake, Workflow executes BAS validation cases, and Component Tests runs
// React Component Library contracts. Every other delegate is inspection-only.
func includeExecutionPhase(phase Name) bool {
	switch phase {
	case Unit, Measures, Performance, Contracts, Search, UIHealth, Workflow, Name("component-tests"):
		return true
	default:
		return false
	}
}

func TestValidationProviderTransportDoesNotRegressToPerPhaseRunners(t *testing.T) {
	dir := phasePackageDir(t)
	for _, filename := range []string{
		"phase_architecture.go",
		"phase_contracts.go",
		"phase_dependencies.go",
		"phase_docs.go",
		"phase_measures.go",
		"phase_proto.go",
		"phase_quality.go",
		"phase_security.go",
		"phase_tidiness.go",
		"phase_ui_health.go",
		"phase_unit.go",
	} {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("%s exists; delegated provider phases must stay in phase_validationprovider.go", filename)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", filename, err)
		}
	}
	assertNoImports(t, filepath.Join(dir, "phase_validationprovider.go"), "net/http", "os/exec")
}

func phasePackageDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}

func assertNoImports(t *testing.T, path string, banned ...string) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s imports: %v", path, err)
	}
	bannedSet := make(map[string]struct{}, len(banned))
	for _, imp := range banned {
		bannedSet[imp] = struct{}{}
	}
	for _, imp := range file.Imports {
		unquoted := imp.Path.Value
		if len(unquoted) >= 2 {
			unquoted = unquoted[1 : len(unquoted)-1]
		}
		if _, banned := bannedSet[unquoted]; banned {
			t.Fatalf("%s imports %s; provider transport must stay in validationprovider", filepath.Base(path), unquoted)
		}
	}
}
