package phases

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidationProviderRegistryCoversDelegatingCatalogPhases(t *testing.T) {
	expected := map[Name]string{
		Structure:    "structure-health",
		Business:     "business-health",
		Contracts:    "cli-health",
		Standards:    "scenario-auditor",
		Proto:        "proto-health",
		UIHealth:     "ui-health",
		Security:     "security-health",
		Quality:      "quality-health",
		Unit:         "unit-health",
		Measures:     "measures-health",
		Dependencies: "scenario-dependency-analyzer",
		Architecture: "architecture-cartographer",
		Docs:         "knowledge-observatory",
		Tidiness:     "tidiness-manager",
		Performance:  "performance-health",
		Storage:      "storage-health",
		Workflow:     "workflow-health",
		Branding:     "brand-manager",
	}
	catalog := NewDefaultCatalog(DefaultTimeout)
	delegatedCount := 0
	for _, spec := range catalog.All() {
		if spec.Delegated != nil {
			delegatedCount++
			if _, ok := expected[spec.Name]; !ok {
				t.Fatalf("%s is delegated but missing from provider guard expectations", spec.Name)
			}
		}
	}
	if delegatedCount != len(expected) {
		t.Fatalf("delegated phase count = %d, want %d", delegatedCount, len(expected))
	}
	for phase, providerScenario := range expected {
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
		if provider.ProviderScenario != providerScenario {
			t.Fatalf("%s provider = %q, want %q", phase, provider.ProviderScenario, providerScenario)
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
		// IncludeExecution pins which delegates request execution-mode validation:
		// the provider actually runs its measurements (not just inspects) and gates
		// on the result. Unit executes the suite, Measures runs its checks,
		// Performance benchmarks the Go + UI build and runs Lighthouse-if-UI,
		// Contracts asks cli-health to run its runtime CLI probe on top of the
		// static manifest↔proto cross-check, and UIHealth drives the BAS render +
		// iframe-bridge handshake runtime group on top of its static UI checks.
		// Workflow executes BAS validation cases through workflow-health when
		// requested, preserving the old playbooks runtime semantics through the
		// shared provider contract.
		// Every other delegate is inspection-only.
		executionPhases := map[Name]bool{Unit: true, Measures: true, Performance: true, Contracts: true, UIHealth: true, Workflow: true}
		if provider.IncludeExecution != executionPhases[phase] {
			t.Fatalf("%s provider IncludeExecution = %v, want %v", phase, provider.IncludeExecution, executionPhases[phase])
		}
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
