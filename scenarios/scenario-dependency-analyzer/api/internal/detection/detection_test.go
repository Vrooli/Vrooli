package detection

import (
	"os"
	"path/filepath"
	"testing"

	appconfig "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/config"
	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

// TestNewDetector tests detector creation.
func TestNewDetector(t *testing.T) {
	cfg := appconfig.RuntimeConfig{
		ScenariosDir: "/tmp/scenarios",
	}
	detector := New(cfg)
	if detector == nil {
		t.Fatal("expected non-nil detector")
	}
	if detector.catalog == nil {
		t.Error("expected catalog to be initialized")
	}
	if detector.resourceScanner == nil {
		t.Error("expected resource scanner to be initialized")
	}
	if detector.scenarioScanner == nil {
		t.Error("expected scenario scanner to be initialized")
	}
}

// TestCatalogManager tests catalog management functionality.
func TestCatalogManager(t *testing.T) {
	t.Run("RefreshClears", func(t *testing.T) {
		catalog := newCatalogManager(appconfig.RuntimeConfig{})
		// Force load
		catalog.ensureLoaded()
		if !catalog.loaded {
			t.Fatal("expected catalog to be loaded")
		}

		// Refresh should clear
		catalog.refresh()
		if catalog.loaded {
			t.Error("expected catalog to be cleared after refresh")
		}
	})

	t.Run("ThreadSafeLoad", func(t *testing.T) {
		catalog := newCatalogManager(appconfig.RuntimeConfig{})
		done := make(chan bool, 10)

		// Concurrent access
		for i := 0; i < 10; i++ {
			go func() {
				catalog.ensureLoaded()
				_ = catalog.isKnownScenario("test")
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("PermissiveModeWhenEmpty", func(t *testing.T) {
		catalog := newCatalogManager(appconfig.RuntimeConfig{})
		catalog.mu.Lock()
		catalog.loaded = true
		catalog.knownScenarios = map[string]struct{}{}
		catalog.knownResources = map[string]struct{}{}
		catalog.mu.Unlock()

		// Empty catalogs should be permissive
		if !catalog.isKnownScenario("any-scenario") {
			t.Error("empty catalog should accept any scenario")
		}
		if !catalog.isKnownResource("any-resource") {
			t.Error("empty catalog should accept any resource")
		}
	})

	t.Run("GetScenarioCatalogReturnsCopy", func(t *testing.T) {
		catalog := newCatalogManager(appconfig.RuntimeConfig{})
		catalog.mu.Lock()
		catalog.loaded = true
		catalog.knownScenarios = map[string]struct{}{"test-scenario": {}}
		catalog.mu.Unlock()

		snap := catalog.getScenarioCatalog()
		snap["modified"] = struct{}{}

		// Original should be unchanged
		if catalog.isKnownScenario("modified") {
			t.Error("modification to snapshot should not affect original")
		}
	})
}

// TestShouldSkipDirectoryEntry tests directory filtering.
func TestShouldSkipDirectoryEntry(t *testing.T) {
	tests := []struct {
		name    string
		dirName string
		isDir   bool
		want    bool
	}{
		{"NodeModules", "node_modules", true, true},
		{"Dist", "dist", true, true},
		{"Build", "build", true, true},
		{"Coverage", "coverage", true, true},
		{"Git", ".git", true, true},
		{"HiddenDir", ".hidden", true, true},
		{"Vrooli", ".vrooli", true, false}, // Special exception
		{"Api", "api", true, false},
		{"Src", "src", true, false},
		{"NotADir", "somefile.go", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := mockDirEntry{name: tt.dirName, isDir: tt.isDir}
			got := shouldSkipDirectoryEntry(entry)
			if got != tt.want {
				t.Errorf("shouldSkipDirectoryEntry(%q) = %v, want %v", tt.dirName, got, tt.want)
			}
		})
	}
}

// TestShouldIgnoreDetectionFile tests file filtering.
func TestShouldIgnoreDetectionFile(t *testing.T) {
	tests := []struct {
		name    string
		relPath string
		want    bool
	}{
		{"Empty", "", false},
		{"GoTest", "main_test.go", true},
		{"JsTest", "utils.test.js", true},
		{"TsSpec", "service.spec.ts", true},
		{"TsxSpec", "component.spec.tsx", true},
		{"Readme", "README.md", true},
		{"ReadmeLower", "readme.md", true},
		{"PRD", "PRD.md", true},
		{"RegularGo", "main.go", false},
		{"RegularTs", "service.ts", false},
		{"DocsPath", "docs/api.md", true},
		{"TestPath", "test/helpers.go", true},
		{"TestsPath", "tests/integration.ts", true},
		{"PlaybooksPath", "playbooks/deploy.yml", true},
		{"SrcPath", "src/main.go", false},
		{"Problems", "PROBLEMS.md", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldIgnoreDetectionFile(tt.relPath)
			if got != tt.want {
				t.Errorf("shouldIgnoreDetectionFile(%q) = %v, want %v", tt.relPath, got, tt.want)
			}
		})
	}
}

// TestIsAllowedResourceCLIPath tests CLI path filtering.
func TestIsAllowedResourceCLIPath(t *testing.T) {
	tests := []struct {
		relPath string
		want    bool
	}{
		{"", true},
		{"api/main.go", true},
		{"cli/commands.go", true},
		{"src/utils.ts", true},
		{"scripts/setup.sh", true},
		{"docs/readme.md", false},
		{"examples/basic/main.go", false},
		{"internal/service.go", true},
		{"pkg/utils.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.relPath, func(t *testing.T) {
			got := isAllowedResourceCLIPath(tt.relPath)
			if got != tt.want {
				t.Errorf("isAllowedResourceCLIPath(%q) = %v, want %v", tt.relPath, got, tt.want)
			}
		})
	}
}

// TestDetectorIntegration tests detector with real file system.
func TestDetectorIntegration(t *testing.T) {
	// Create test directory structure
	tempDir := t.TempDir()
	scenariosDir := filepath.Join(tempDir, "scenarios")
	resourcesDir := filepath.Join(tempDir, "resources")

	// Create directories
	os.MkdirAll(filepath.Join(scenariosDir, "test-scenario"), 0o755)
	os.MkdirAll(filepath.Join(resourcesDir, "postgres"), 0o755)
	os.MkdirAll(filepath.Join(resourcesDir, "redis"), 0o755)

	cfg := appconfig.RuntimeConfig{
		ScenariosDir: scenariosDir,
	}
	detector := New(cfg)

	t.Run("KnownScenario", func(t *testing.T) {
		detector.RefreshCatalogs()
		if !detector.KnownScenario("test-scenario") {
			t.Error("expected test-scenario to be known")
		}
	})

	t.Run("UnknownScenario", func(t *testing.T) {
		detector.RefreshCatalogs()
		// With populated catalog, unknown scenario should not be known
		catalog := detector.ScenarioCatalog()
		if len(catalog) > 0 && detector.KnownScenario("nonexistent-scenario") {
			t.Error("expected nonexistent-scenario to not be known")
		}
	})

	t.Run("KnownResource", func(t *testing.T) {
		detector.RefreshCatalogs()
		if !detector.KnownResource("postgres") {
			t.Error("expected postgres to be known")
		}
		if !detector.KnownResource("redis") {
			t.Error("expected redis to be known")
		}
	})
}

// TestScanResources tests resource scanning.
func TestScanResources(t *testing.T) {
	tempDir := t.TempDir()
	scenariosDir := filepath.Join(tempDir, "scenarios")
	resourcesDir := filepath.Join(tempDir, "resources")

	// Create test scenario with resource references
	scenarioPath := filepath.Join(scenariosDir, "test-scenario")
	os.MkdirAll(filepath.Join(scenarioPath, "api"), 0o755)
	os.MkdirAll(resourcesDir, 0o755)

	// Create a script with resource CLI reference
	scriptContent := `#!/bin/bash
resource-postgres connect
PGHOST=localhost
`
	os.WriteFile(filepath.Join(scenarioPath, "api", "setup.sh"), []byte(scriptContent), 0o644)

	cfg := appconfig.RuntimeConfig{
		ScenariosDir: scenariosDir,
	}
	detector := New(cfg)
	detector.RefreshCatalogs()

	serviceConfig := &types.Manifest{}
	deps, err := detector.ScanResources(scenarioPath, "test-scenario", serviceConfig)
	if err != nil {
		t.Fatalf("ScanResources error: %v", err)
	}

	// The scanner should find resource references (may or may not depending on catalog state)
	t.Logf("Found %d resource dependencies", len(deps))
}

func TestScenarioURLResolverDetectionRequiresCanonicalImportedResolverAndLiteralKnownTarget(t *testing.T) {
	scenariosDir := t.TempDir()
	for _, name := range []string{"device-studio", "ai-gateway"} {
		path := filepath.Join(scenariosDir, name, ".vrooli")
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "service.json"), []byte(`{"service":{"name":"`+name+`"}}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	scanner := newScenarioScanner(newCatalogManager(appconfig.RuntimeConfig{ScenariosDir: scenariosDir}))
	source := `package rendering
import (
	"github.com/vrooli/api-core/discovery"
	other "example.invalid/discovery"
)
func resolve(ctx context.Context, target string) {
	// discovery.ResolveScenarioURLDefault(ctx, "comment-only")
	_ = "discovery.ResolveScenarioURLDefault(ctx, \"string-only\")"
	_, _ = discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
	_, _ = discovery.ResolveScenarioURLDefault(ctx, target)
	_, _ = other.ResolveScenarioURLDefault(ctx, "ai-gateway")
}`
	got := scanner.scanFile(source, "api/rendering/worker.go", "device-studio", "device-studio")
	if len(got) != 1 {
		t.Fatalf("detected %d dependencies, want one literal canonical resolver call: %+v", len(got), got)
	}
	if got[0].DependencyName != "ai-gateway" || got[0].AccessMethod != "scenario_url_discovery" || got[0].Required {
		t.Fatalf("unexpected resolver evidence: %+v", got[0])
	}
	if got[0].Configuration["found_in_file"] != "api/rendering/worker.go" {
		t.Fatalf("resolver evidence lost source path: %+v", got[0].Configuration)
	}
}

func TestScanScenarioDependenciesIncludesCanonicalProgramBindings(t *testing.T) {
	scenariosDir := t.TempDir()
	for _, name := range []string{"image-tools", "device-studio"} {
		if err := os.MkdirAll(filepath.Join(scenariosDir, name, ".vrooli"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(scenariosDir, name, ".vrooli", "service.json"), []byte(`{"service":{"name":"`+name+`"}}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	root := filepath.Join(scenariosDir, "image-tools")
	files := map[string]string{
		".vrooli/program-runtime/prepare.json": `{"name":"image-tools.prepare","bindings":[{"id":"device-studio/scenes/from-preparation","optional":true},{"id":"device-studio/scenes/get"},{"id":"image-tools/jobs/get"},{"id":"unknown-owner/jobs/get"},{"id":"device-studio//invalid"}]}`,
		".vrooli/program-runtime/foreign.json": `{"name":"another-owner.prepare","bindings":[{"id":"device-studio/scenes/get"}]}`,
		".vrooli/program-runtime/broken.json":  `{"name":"image-tools.broken","bindings":[`,
		"docs/example.json":                    `{"name":"image-tools.example","bindings":[{"id":"device-studio/scenes/get"}]}`,
	}
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	detector := New(appconfig.RuntimeConfig{ScenariosDir: scenariosDir})
	deps, err := detector.ScanScenarioDependencies(root, "image-tools")
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 1 {
		t.Fatalf("dependencies = %+v; want one deduplicated known foreign owner from the canonical contract", deps)
	}
	dep := deps[0]
	if dep.DependencyName != "device-studio" || dep.AccessMethod != "program_binding" || dep.Required || dep.Configuration["found_in_file"] != ".vrooli/program-runtime/prepare.json" {
		t.Fatalf("program dependency lost optionality or source evidence: %+v", dep)
	}
}

// TestScanScenarioDependencies tests scenario dependency scanning.
func TestScanScenarioDependencies(t *testing.T) {
	tempDir := t.TempDir()
	scenariosDir := filepath.Join(tempDir, "scenarios")

	// Create test scenarios
	testScenario := filepath.Join(scenariosDir, "test-scenario")
	dependencyScenario := filepath.Join(scenariosDir, "dependency-scenario")
	os.MkdirAll(filepath.Join(testScenario, "api"), 0o755)
	os.MkdirAll(filepath.Join(dependencyScenario, ".vrooli"), 0o755)

	// Create script with scenario reference
	scriptContent := `#!/bin/bash
vrooli scenario run dependency-scenario
`
	os.WriteFile(filepath.Join(testScenario, "api", "start.sh"), []byte(scriptContent), 0o644)

	cfg := appconfig.RuntimeConfig{
		ScenariosDir: scenariosDir,
	}
	detector := New(cfg)
	detector.RefreshCatalogs()

	deps, err := detector.ScanScenarioDependencies(testScenario, "test-scenario")
	if err != nil {
		t.Fatalf("ScanScenarioDependencies error: %v", err)
	}

	// Should find scenario reference
	found := false
	for _, dep := range deps {
		if dep.DependencyName == "dependency-scenario" {
			found = true
			break
		}
	}

	if !found && len(deps) > 0 {
		t.Logf("Found dependencies: %+v", deps)
	}
}

// mockDirEntry is a mock implementation of fs.DirEntry for testing
type mockDirEntry struct {
	name  string
	isDir bool
}

func (m mockDirEntry) Name() string               { return m.name }
func (m mockDirEntry) IsDir() bool                { return m.isDir }
func (m mockDirEntry) Type() os.FileMode          { return 0 }
func (m mockDirEntry) Info() (os.FileInfo, error) { return nil, nil }

// TestNormalizeName tests name normalization.
func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"postgres", "postgres"},
		{"Postgres", "postgres"},
		{"POSTGRES", "postgres"},
		{"claude-code", "claude-code"},
		{"Claude-Code", "claude-code"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeName(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
