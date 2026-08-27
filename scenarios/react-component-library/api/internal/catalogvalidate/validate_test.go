package catalogvalidate

import (
	"encoding/json"
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
	if err := os.WriteFile(filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json"), []byte(`{"kind":"catalog-config","contract":{"kind":"ui-catalog","schema":"catalog-asset/v1"},"schemaVersion":"1.0.0","domains":[{"id":"controls","description":"Controls are reusable interactive surfaces."}],"gates":[{"id":"types","description":"Strict type validation gate.","appliesTo":["component"],"blocking":true,"rung":"scaffolded","attribution":"attributable","runner":{"react-vite":"check types"}}]}`), 0o644); err != nil {
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
		Slot   string `json:"slot"`
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

func TestVacuousAllowlistRequiresReasonsAndDetectsGrowth(t *testing.T) {
	_, findings := parseVacuousAllowlist("library/vacuous-allowlist.json", []byte(`{
  "schemaVersion": 1,
  "entries": [
    {"path":"library/components/B/versions/1.0.0/experience-contract.json","reason":"legacy"},
    {"path":"library/components/A/versions/1.0.0/experience-contract.json","reason":"legacy"},
    {"path":"library/components/A/versions/1.0.0/experience-contract.json","reason":"duplicate"},
    {"path":"library/components/C/versions/1.0.0/experience-contract.json","reason":""}
  ]
}`))
	if len(findings) < 3 {
		t.Fatalf("findings = %+v", findings)
	}

	current := []vacuousAllowlistEntry{{Path: "a"}, {Path: "b"}}
	baseline := []vacuousAllowlistEntry{{Path: "a"}}
	if got := allowlistGrowth(current, baseline); len(got) != 1 || got[0] != "b" {
		t.Fatalf("allowlistGrowth = %v, want [b]", got)
	}
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

func TestLiveCatalogHasTypedFloorAndNoGateExtensions(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		AdoptionMaturityFloor string           `json:"adoptionMaturityFloor"`
		Gates                 []map[string]any `json:"gates"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if config.AdoptionMaturityFloor == "" {
		t.Fatal("catalog config must declare adoptionMaturityFloor")
	}
	for _, gate := range config.Gates {
		for key := range gate {
			if strings.HasPrefix(key, "x-") {
				t.Fatalf("gate %v retains extension key %q", gate["id"], key)
			}
		}
	}
}

func TestCatalogSchemaRejectsMisspelledFloor(t *testing.T) {
	root := t.TempDir()
	schemaDir := filepath.Join(root, ".vrooli", "schemas")
	configDir := filepath.Join(root, "scenarios", "react-component-library", "catalog")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", ".vrooli", "schemas", "catalog-asset.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(schemaDir, "catalog-asset.schema.json"), schema, 0o644); err != nil {
		t.Fatal(err)
	}
	config := `{"kind":"catalog-config","contract":{"kind":"ui-catalog","schema":"catalog-asset/v1"},"schemaVersion":"1.0.0","adoptionMaturityFloor":"verifed","domains":[{"id":"controls","order":10,"description":"Controls are reusable interactive surfaces."}],"gates":[{"id":"types","description":"Strict type validation gate.","appliesTo":["component"],"blocking":true,"rung":"scaffolded","attribution":"attributable","runner":{"react-vite":"check types"}}]}`
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := New(root).Validate()
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding.Code == "catalog.schema_error" && strings.Contains(finding.Message, "adoptionMaturityFloor") {
			return
		}
	}
	t.Fatalf("misspelled floor did not produce a schema finding: %+v", findings)
}

func TestCatalogSchemaRequiresGateBlockingAndRunner(t *testing.T) {
	root := t.TempDir()
	schemaDir := filepath.Join(root, ".vrooli", "schemas")
	configDir := filepath.Join(root, "scenarios", "react-component-library", "catalog")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", ".vrooli", "schemas", "catalog-asset.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(schemaDir, "catalog-asset.schema.json"), schema, 0o644); err != nil {
		t.Fatal(err)
	}
	config := `{"kind":"catalog-config","contract":{"kind":"ui-catalog","schema":"catalog-asset/v1"},"schemaVersion":"1.0.0","adoptionMaturityFloor":"scaffolded","domains":[],"gates":[{"id":"types","rung":"scaffolded","attribution":"attributable","appliesTo":["component"]}]}`
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := New(root).Validate()
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) == 0 {
		t.Fatal("gate missing required blocking and runner fields was accepted")
	}
	for _, finding := range findings {
		if finding.Code == "catalog.schema_error" && strings.Contains(finding.Message, "blocking") {
			return
		}
	}
	t.Fatalf("required gate fields did not produce a blocking schema finding: %+v", findings)
}

func TestCalibrationCoverageMatchesGates(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	configData, err := os.ReadFile(filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Gates []struct {
			ID       string `json:"id"`
			Blocking bool   `json:"blocking"`
		} `json:"gates"`
	}
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatal(err)
	}
	calibrationRoot := filepath.Join(root, "scenarios", "react-component-library", "catalog", "calibration")
	entries, err := os.ReadDir(calibrationRoot)
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{}
	for _, gate := range config.Gates {
		known[gate.ID] = true
		if gate.Blocking {
			if info, statErr := os.Stat(filepath.Join(calibrationRoot, gate.ID)); statErr != nil || !info.IsDir() {
				t.Fatalf("blocking gate %q has no calibration directory", gate.ID)
			}
		}
	}
	for _, entry := range entries {
		if entry.IsDir() && !known[entry.Name()] {
			t.Fatalf("calibration directory %q has no gate", entry.Name())
		}
	}
}

func TestDomainOrderFindingsReportBothDuplicates(t *testing.T) {
	root := t.TempDir()
	catalogDir := filepath.Join(root, "scenarios", "react-component-library", "catalog")
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(catalogDir, "config.json"), []byte(`{"domains":[{"id":"a","order":10},{"id":"b","order":10},{"id":"c","order":20}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	findings := domainOrderFindings(root, catalogDir)
	duplicates := 0
	for _, finding := range findings {
		if finding.Code == "catalog.domain_order_duplicate" && finding.Severity == "error" {
			duplicates++
		}
	}
	if duplicates != 2 {
		t.Fatalf("duplicate findings=%d: %#v", duplicates, findings)
	}
}

func TestUnknownKindProducesTypedFinding(t *testing.T) {
	root := t.TempDir()
	catalogDir := filepath.Join(root, "scenarios", "react-component-library", "catalog")
	assetDir := filepath.Join(catalogDir, "assets", "controls")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(catalogDir, "config.json"), []byte(`{"domains":[{"id":"controls","order":10}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "unknown.json"), []byte(`{"kind":"catalog-asset","asset":{"id":"controls.unknown","kind":"typo-kind","targets":[]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	findings := crossRegistryFindings(root, catalogDir)
	for _, finding := range findings {
		if finding.Code == "catalog.unknown_kind" && finding.Severity == "error" && finding.Location == "controls.unknown" {
			return
		}
	}
	t.Fatalf("unknown-kind finding missing: %#v", findings)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
