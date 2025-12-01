package suite

import (
	"strings"
	"testing"
	"time"
)

func TestNewDefaultPhaseCatalogRegistersPhases(t *testing.T) {
	catalog := NewDefaultPhaseCatalog(time.Minute)
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
	name, ok := normalizePhaseName("  Unit  ")
	if !ok {
		t.Fatalf("expected normalization success")
	}
	if name != PhaseUnit {
		t.Fatalf("expected %s but got %s", PhaseUnit, name)
	}
	if !name.IsZero() && name.Key() != "unit" {
		t.Fatalf("unexpected map key %s", name.Key())
	}
}

func TestPhaseCatalogDescriptors(t *testing.T) {
	catalog := NewDefaultPhaseCatalog(time.Second)
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
