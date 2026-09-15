package zones

import (
	"os"
	"path/filepath"
	"testing"

	"architecture-cartographer/internal/domains"
)

func TestConfigClassify(t *testing.T) {
	cfg := defaultConfig()
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "orders", Paths: []string{"api/internal/orders/", "api/handlers/orders/", "cli/domains/orders/", "ui/src/features/orders/"}, Archetypes: domains.DeclaredArchetypes("service")},
		},
		SharedSubstrate: []string{"api/internal/customkernel/"},
	}
	cases := map[string]string{
		"api/handlers/orders":       Transport,
		"api/internal/orders":       Domain,
		"cli/domains/orders":        CLI,
		"ui/src/features/orders":    UI,
		"api/internal/server":       Substrate,
		"api/internal/customkernel": Substrate,
		"api/internal/app":          CompositionRoot,
	}
	for path, want := range cases {
		if got := cfg.Classify(path, m); got.Zone != want {
			t.Fatalf("Classify(%q).Zone = %q, want %q (%+v)", path, got.Zone, want, got)
		}
	}
	if got := cfg.Classify("api/internal/orders/service.go", m); got.Domain != "orders" || got.Archetype != "service" || !got.Declared {
		t.Fatalf("orders classification = %+v", got)
	}
}

func TestLoadForScenarioReadsManifestAndEnv(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "demo")
	templateDir := filepath.Join(root, "templates", "scenarios", "react-vite")
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), []byte(`{"generation":{"template":{"id":"react-vite"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{
	  "zones": {
	    "pathPatterns": {
	      "transport": ["server/routes/<domain>/"],
	      "domain": ["core/<domain>/"],
	      "substrate": ["core/<segment>/"],
	      "composition-root": ["core/app/"]
	    },
	    "builtinSubstrateSegments": ["platform"],
	    "compositionRootSegments": ["app"],
	    "coordinatingArchetypes": ["service"]
	  }
	}`
	if err := os.WriteFile(filepath.Join(templateDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envExtraSubstrate, "extra")
	cfg := LoadForScenario(scenarioDir)
	if _, ok := cfg.BuiltinSubstrateSegments["platform"]; !ok {
		t.Fatalf("manifest substrate not loaded: %+v", cfg.BuiltinSubstrateSegments)
	}
	if _, ok := cfg.BuiltinSubstrateSegments["extra"]; !ok {
		t.Fatalf("env substrate not loaded: %+v", cfg.BuiltinSubstrateSegments)
	}
	m := domains.DerivedDomainMap{Domains: []domains.DerivedDomain{{Name: "billing", Paths: []string{"core/billing/"}}}}
	if got := cfg.Classify("server/routes/billing", m); got.Zone != Transport || got.Domain != "billing" {
		t.Fatalf("manifest transport classification = %+v", got)
	}
	if got := cfg.Classify("core/extra", m); got.Zone != Substrate {
		t.Fatalf("env substrate classification = %+v", got)
	}
}

func TestLoadForScenarioOverlayRecordsDeviation(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "demo")
	templateDir := filepath.Join(root, "templates", "scenarios", "react-vite")
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), []byte(`{"generation":{"template":{"id":"react-vite"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"zones":{"pathPatterns":{"transport":["api/handlers/<domain>/"],"domain":["api/internal/<domain>/"],"substrate":["api/internal/<segment>/"]},"builtinSubstrateSegments":["server"]}}`
	if err := os.WriteFile(filepath.Join(templateDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	// Overlay overrides the transport pattern and substrate segments.
	overlayJSON := `{"zones":{"pathPatterns":{"transport":["http/<domain>/"]},"builtinSubstrateSegments":["kernel"]}}`
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "architecture.json"), []byte(overlayJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := LoadForScenario(scenarioDir)
	if len(cfg.Deviations) != 2 {
		t.Fatalf("want 2 deviations, got %d: %+v", len(cfg.Deviations), cfg.Deviations)
	}
	if _, ok := cfg.BuiltinSubstrateSegments["kernel"]; !ok {
		t.Fatalf("overlay substrate not applied: %+v", cfg.BuiltinSubstrateSegments)
	}
	if _, ok := cfg.BuiltinSubstrateSegments["server"]; ok {
		t.Fatalf("overlay should replace template substrate, not merge: %+v", cfg.BuiltinSubstrateSegments)
	}
	m := domains.DerivedDomainMap{Domains: []domains.DerivedDomain{{Name: "billing", Paths: []string{"api/internal/billing/"}}}}
	if got := cfg.Classify("http/billing", m); got.Zone != Transport {
		t.Fatalf("overlay transport classification = %+v", got)
	}
}
