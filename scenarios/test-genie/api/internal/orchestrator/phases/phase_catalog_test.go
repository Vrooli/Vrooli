package phases

import (
	"strings"
	"testing"
	"time"
)

func TestNewDefaultPhaseCatalogRegistersPhases(t *testing.T) {
	catalog := NewDefaultCatalog(time.Minute)
	specs := catalog.All()
	if len(specs) < 6 {
		t.Fatalf("expected at least 6 registered phases, got %d", len(specs))
	}

	perf, ok := catalog.Lookup("PERFORMANCE")
	if !ok {
		t.Fatalf("expected performance phase to be registered")
	}
	if !perf.Optional {
		t.Fatalf("performance phase should be optional")
	}
	if perf.Runner == nil {
		t.Fatalf("performance phase should expose a runner")
	}
	if perf.DefaultTimeout <= 0 {
		t.Fatalf("expected default timeout to be set")
	}
	if strings.TrimSpace(perf.Description) == "" {
		t.Fatalf("expected description to be set for performance phase")
	}
}

func TestNormalizePhaseName(t *testing.T) {
	name, ok := NormalizeName("  Unit  ")
	if !ok {
		t.Fatalf("expected normalization success")
	}
	if name != Unit {
		t.Fatalf("expected %s but got %s", Unit, name)
	}
	if !name.IsZero() && name.Key() != "unit" {
		t.Fatalf("unexpected map key %s", name.Key())
	}
}

func TestPhaseCatalogDescriptors(t *testing.T) {
	catalog := NewDefaultCatalog(time.Second)
	descriptors := catalog.Descriptors()
	if len(descriptors) == 0 {
		t.Fatalf("expected descriptors to be returned")
	}
	for _, descriptor := range descriptors {
		if descriptor.Name == "" {
			t.Fatalf("descriptor missing name: %#v", descriptor)
		}
		if descriptor.Source == "" {
			t.Fatalf("descriptor missing source: %#v", descriptor)
		}
	}
}

func TestLintPhaseTimeout(t *testing.T) {
	catalog := NewDefaultCatalog(15 * time.Minute) // Default is 15 minutes
	lint, ok := catalog.Lookup("lint")
	if !ok {
		t.Fatalf("expected lint phase to be registered")
	}
	expected := 30 * time.Second
	if lint.DefaultTimeout != expected {
		t.Errorf("lint phase timeout = %v, want %v", lint.DefaultTimeout, expected)
	}
}

func TestLintPhaseIsRegistered(t *testing.T) {
	catalog := NewDefaultCatalog(time.Minute)
	lint, ok := catalog.Lookup("lint")
	if !ok {
		t.Fatalf("expected lint phase to be registered")
	}
	if lint.Runner == nil {
		t.Fatalf("lint phase should have a runner")
	}
	if lint.Optional {
		t.Fatalf("lint phase should not be optional")
	}
	if !strings.Contains(lint.Description, "linting") && !strings.Contains(lint.Description, "static analysis") {
		t.Errorf("lint phase description should mention linting or static analysis, got: %s", lint.Description)
	}
}

func TestDocsPhaseIsRegistered(t *testing.T) {
	catalog := NewDefaultCatalog(time.Minute)
	docsPhase, ok := catalog.Lookup("docs")
	if !ok {
		t.Fatalf("expected docs phase to be registered")
	}
	if docsPhase.Runner == nil {
		t.Fatalf("docs phase should have a runner")
	}
	if docsPhase.Optional {
		t.Fatalf("docs phase should not be optional")
	}
	if !strings.Contains(strings.ToLower(docsPhase.Description), "docs") {
		t.Errorf("docs phase description should mention docs, got: %s", docsPhase.Description)
	}
}
