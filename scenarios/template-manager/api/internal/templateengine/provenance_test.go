package templateengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func TestInjectScenarioProvenancePreservesOtherServiceDeclarations(t *testing.T) {
	destination := t.TempDir()
	serviceDir := filepath.Join(destination, ".vrooli")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	servicePath := filepath.Join(serviceDir, "service.json")
	const source = `{"service":{"name":"demo"},"skills":{"usage":{"source":"skills/demo/SKILL.md"}},"operator_capabilities":[{"id":"review"}]}`
	if err := os.WriteFile(servicePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	provenance := templatecontracts.GenerationProvenance{
		Template:    templatecontracts.GenerationTemplate{ID: "react-vite", Version: "6.0.1"},
		GeneratedAt: "2026-11-09T21:30:00Z",
		Design:      templatecontracts.GenerationDesign{ID: "vrooli-default", Version: "0.2.0", Adapter: "react-vite-tailwind"},
		ManifestSha: "manifest-sha",
		ContentSha:  "content-sha",
	}
	if err := injectScenarioProvenance(destination, provenance); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatal(err)
	}
	var service map[string]json.RawMessage
	if err := json.Unmarshal(data, &service); err != nil {
		t.Fatal(err)
	}
	var skills map[string]any
	if err := json.Unmarshal(service["skills"], &skills); err != nil {
		t.Fatalf("skills declaration was lost: %v", err)
	}
	var capabilities []map[string]any
	if err := json.Unmarshal(service["operator_capabilities"], &capabilities); err != nil || len(capabilities) != 1 {
		t.Fatalf("operator capabilities = %v, err = %v; want original declaration preserved", capabilities, err)
	}
	var generation templatecontracts.GenerationProvenance
	if err := json.Unmarshal(service["generation"], &generation); err != nil {
		t.Fatal(err)
	}
	if generation.Template.ID != provenance.Template.ID || generation.Template.Version != provenance.Template.Version || generation.GeneratedAt != provenance.GeneratedAt || generation.Design.ID != provenance.Design.ID || generation.Design.Version != provenance.Design.Version || generation.Design.Adapter != provenance.Design.Adapter || generation.ManifestSha != provenance.ManifestSha || generation.ContentSha != provenance.ContentSha {
		t.Fatalf("generation metadata = %+v, want %+v", generation, provenance)
	}
}
