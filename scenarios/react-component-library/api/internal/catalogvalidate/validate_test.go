package catalogvalidate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReportsSchemaViolationAsFinding(t *testing.T) {
	root := t.TempDir()
	schemaDir := filepath.Join(root, ".vrooli", "schemas")
	catalogDir := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "controls")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", ".vrooli", "schemas", "catalog-asset.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(schemaDir, "catalog-asset.schema.json"), schema, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json"), []byte(`{"kind":"catalog-config","contract":{"kind":"ui-catalog","schema":"catalog-asset/v1"},"schemaVersion":"1.0.0","domains":[{"id":"controls","description":"Controls are reusable interactive surfaces."}],"gates":[{"id":"types","description":"Strict type validation gate.","appliesTo":["component"],"blocking":true,"rung":"scaffolded"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(catalogDir, "broken.json"), []byte(`{"kind":"catalog-asset","schemaVersion":"1.0.0","asset":{"id":"controls.button","name":"Button","kind":"not-a-kind","domain":"controls","description":"A button with a clear action and accessible name.","targets":["react-vite"],"delivery":"adopted","target":{"priority":"P0","maturity":"scaffolded"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := New(root).Validate()
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) == 0 {
		t.Fatal("expected schema findings")
	}
	found := false
	for _, finding := range findings {
		if finding.Code == "catalog.schema_error" && strings.Contains(finding.Location, "broken.json") {
			found = true
		}
	}
	if !found {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestVacuousRungDetectsMissingGate(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "scenarios", "react-component-library", "catalog")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := `{"gates":[{"id":"types","rung":"scaffolded","blocking":true,"appliesTo":["foundation"]}]}`
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	findings := vacuousFindings(root, []assetDoc{{Asset: struct {
		ID     string `json:"id"`
		Kind   string `json:"kind"`
		Target struct {
			Maturity string `json:"maturity"`
		} `json:"target"`
		Targets []string `json:"targets"`
	}{ID: "foundations.tokens", Kind: "foundation", Target: struct {
		Maturity string `json:"maturity"`
	}{Maturity: "verified"}}}})
	for _, finding := range findings {
		if finding.Code == "catalog.vacuous_rung" && strings.Contains(finding.Message, "implemented") {
			return
		}
	}
	t.Fatalf("findings = %+v", findings)
}

func TestLiveCatalogValidation(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := New(root).Validate()
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) > 0 {
		t.Fatalf("live catalog has %d findings: %+v", len(findings), findings[:min(10, len(findings))])
	}
}

func min(a, b int) int {
	if a < b { return a }
	return b
}
