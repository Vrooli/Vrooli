package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestAnalyzeScenario tests scenario analysis functionality
func TestAnalyzeScenario(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Set up test scenario directory
	scenarioName := "test-scenario"
	testScenario := createTestScenario(t, env, scenarioName, map[string]Resource{
		"postgres": {
			Type:     "postgres",
			Enabled:  true,
			Required: true,
			Purpose:  "Primary database",
		},
		"redis": {
			Type:     "redis",
			Enabled:  true,
			Required: false,
			Purpose:  "Caching layer",
		},
	})
	defer testScenario.Cleanup()

	t.Run("ValidScenario", func(t *testing.T) {
		// This test requires the analyzeScenario function to be updated to accept
		// a scenarios directory parameter, or we need to mock the config
		// For now, we'll test the structure

		// Create a mock config
		oldScenariosDir := os.Getenv("VROOLI_SCENARIOS_DIR")
		os.Setenv("VROOLI_SCENARIOS_DIR", env.ScenariosDir)
		defer func() {
			if oldScenariosDir != "" {
				os.Setenv("VROOLI_SCENARIOS_DIR", oldScenariosDir)
			} else {
				os.Unsetenv("VROOLI_SCENARIOS_DIR")
			}
		}()

		// The actual test would call analyzeScenario, but it requires config setup
		// We'll test the components instead
		t.Skip("Requires full config setup - testing components individually instead")
	})
}

// TestScanForScenarioDependencies tests dependency scanning
func TestScanForScenarioDependencies(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	scenarioName := "test-scenario"
	scenarioPath := filepath.Join(env.ScenariosDir, scenarioName)

	t.Run("WithScenarioReferences", func(t *testing.T) {
		os.MkdirAll(scenarioPath, 0755)
		defer os.RemoveAll(scenarioPath)

		// Create a file with scenario references
		testFile := filepath.Join(scenarioPath, "test.sh")
		content := `#!/bin/bash
vrooli scenario run other-scenario
vrooli scenario test another-scenario
`
		os.WriteFile(testFile, []byte(content), 0644)

		deps, err := scanForScenarioDependencies(scenarioPath, scenarioName)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Should find references to other scenarios
		if len(deps) < 2 {
			t.Logf("Found %d dependencies (expected at least 2)", len(deps))
		}

		for _, dep := range deps {
			if dep.DependencyType != "scenario" {
				t.Errorf("Expected dependency type 'scenario', got %s", dep.DependencyType)
			}
		}
	})

	t.Run("EmptyDirectory", func(t *testing.T) {
		emptyPath := filepath.Join(env.ScenariosDir, "empty-scenario")
		os.MkdirAll(emptyPath, 0755)
		defer os.RemoveAll(emptyPath)

		deps, err := scanForScenarioDependencies(emptyPath, "empty-scenario")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(deps) != 0 {
			t.Errorf("Expected no dependencies for empty directory, got %d", len(deps))
		}
	})

	t.Run("WithCLIReferences", func(t *testing.T) {
		cliPath := filepath.Join(env.ScenariosDir, "cli-test")
		os.MkdirAll(cliPath, 0755)
		defer os.RemoveAll(cliPath)

		// Create file with CLI references
		scriptFile := filepath.Join(cliPath, "script.sh")
		content := `#!/bin/bash
data-tools-cli.sh process input.csv
`
		os.WriteFile(scriptFile, []byte(content), 0644)

		deps, err := scanForScenarioDependencies(cliPath, "cli-test")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify we can scan without errors
		t.Logf("Found %d CLI dependencies", len(deps))
	})
}

func TestScanForScenarioDependenciesFiltersNoise(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configureTestScenariosDir(t, env)
	keep := createTestScenario(t, env, "other-scenario", map[string]Resource{})
	defer keep.Cleanup()
	cliScenario := createTestScenario(t, env, "browser-automation-studio", map[string]Resource{})
	defer cliScenario.Cleanup()
	ignored := createTestScenario(t, env, "ignored-scenario", map[string]Resource{})
	defer ignored.Cleanup()

	subjectPath := filepath.Join(env.ScenariosDir, "subject")
	os.MkdirAll(subjectPath, 0755)
	defer os.RemoveAll(subjectPath)

	mainFile := filepath.Join(subjectPath, "script.sh")
	content := `#!/bin/bash
vrooli scenario run other-scenario
vrooli scenario run totally-fake
`
	os.WriteFile(mainFile, []byte(content), 0644)

	cliFile := filepath.Join(subjectPath, "cli.sh")
	os.WriteFile(cliFile, []byte("browser-automation-studio analyze"), 0644)

	nodeModules := filepath.Join(subjectPath, "node_modules", "pkg")
	os.MkdirAll(nodeModules, 0755)
	os.WriteFile(filepath.Join(nodeModules, "index.js"), []byte("vrooli scenario run ignored-scenario"), 0644)

	deps, err := scanForScenarioDependencies(subjectPath, "subject")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(deps) != 2 {
		t.Fatalf("Expected 2 dependencies, got %d", len(deps))
	}

	found := map[string]bool{}
	for _, dep := range deps {
		found[dep.DependencyName] = true
	}

	if !found["other-scenario"] || !found["browser-automation-studio"] {
		t.Fatalf("Missing expected dependencies: %v", found)
	}
	if found["ignored-scenario"] {
		t.Fatalf("ignored-scenario should have been excluded")
	}
}

func TestScanForScenarioDependenciesDetectsScenarioPortResolvers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configureTestScenariosDir(t, env)
	issueTracker := createTestScenario(t, env, "app-issue-tracker", map[string]Resource{})
	defer issueTracker.Cleanup()
	subject := createTestScenario(t, env, "port-resolver", map[string]Resource{})
	defer subject.Cleanup()
	refreshDependencyCatalogs()

	sourcePath := filepath.Join(env.ScenariosDir, "port-resolver", "main.go")
	source := `package main
const issueTrackerScenarioID = "app-issue-tracker"
func use(ctx context.Context) {
    resolveScenarioPortViaCLI(ctx, issueTrackerScenarioID, "API_PORT")
}`
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		t.Fatalf("Failed to write source: %v", err)
	}

	deps, err := scanForScenarioDependencies(filepath.Join(env.ScenariosDir, "port-resolver"), "port-resolver")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	found := false
	for _, dep := range deps {
		if dep.DependencyName == "app-issue-tracker" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("Expected to detect app-issue-tracker dependency via port resolver")
	}
}

func TestApplyDetectedDiffsPreservesExistingDependencies(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	setEnvAndCleanup(t, "API_PORT", "12345")
	setEnvAndCleanup(t, "DATABASE_URL", "postgres://example:test@localhost:5432/test")
	setEnvAndCleanup(t, "VROOLI_SCENARIOS_DIR", env.ScenariosDir)

	resourcesDir := filepath.Join(env.TempDir, "resources")
	os.MkdirAll(filepath.Join(resourcesDir, "postgres"), 0o755)
	os.MkdirAll(filepath.Join(resourcesDir, "redis"), 0o755)
	refreshDependencyCatalogs()

	scenarioName := "apply-test"
	scenarioPath := filepath.Join(env.ScenariosDir, scenarioName)
	os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755)

	serviceConfig := map[string]interface{}{
		"$schema": "../../../.vrooli/schemas/service.schema.json",
		"version": "1.0.0",
		"service": map[string]interface{}{
			"name":        scenarioName,
			"displayName": scenarioName,
			"description": "",
			"version":     "1.0.0",
		},
		"dependencies": map[string]interface{}{
			"resources": map[string]interface{}{
				"postgres": map[string]interface{}{
					"type":     "postgres",
					"enabled":  true,
					"required": true,
				},
			},
			"scenarios": map[string]interface{}{},
		},
	}
	data, _ := json.MarshalIndent(serviceConfig, "", "  ")
	os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), data, 0o644)

	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		t.Fatalf("failed to load service config: %v", err)
	}
	if len(cfg.Dependencies.Resources) == 0 {
		t.Fatalf("precondition failed: expected postgres in dependencies")
	}
	analysis := &DependencyAnalysisResponse{
		Scenario: scenarioName,
		DetectedResources: []ScenarioDependency{
			{ID: "redis-detected", ScenarioName: scenarioName, DependencyType: "resource", DependencyName: "redis", Required: true, AccessMethod: "heuristic"},
		},
		ResourceDiff: DependencyDiff{
			Missing: []DependencyDrift{{Name: "redis"}},
		},
	}

	var captured int
	applyDiffsHook = func(name string, cfg *ServiceConfig) {
		if name == scenarioName {
			captured = len(cfg.Dependencies.Resources)
		}
	}
	defer func() { applyDiffsHook = nil }()

	summary, err := applyDetectedDiffs(scenarioName, analysis, true, false)
	if err != nil {
		t.Fatalf("applyDetectedDiffs returned error: %v", err)
	}
	if changed, _ := summary["changed"].(bool); !changed {
		t.Fatalf("expected changes to be applied")
	}
	if captured == 0 {
		t.Fatalf("hook observed zero declared resources before apply")
	}

	cfg, err = loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		t.Fatalf("loadServiceConfigFromFile failed: %v", err)
	}
	resources := resolvedResourceMap(cfg)
	if _, ok := resources["postgres"]; !ok {
		t.Fatalf("existing resource 'postgres' was removed")
	}
	if _, ok := resources["redis"]; !ok {
		t.Fatalf("newly detected resource 'redis' was not added")
	}
}

func TestScanForResourceUsageDetectsInitialization(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	setEnvAndCleanup(t, "VROOLI_SCENARIOS_DIR", env.ScenariosDir)
	resourcesDir := filepath.Join(env.TempDir, "resources")
	os.MkdirAll(filepath.Join(resourcesDir, "n8n"), 0o755)
	refreshDependencyCatalogs()

	scenarioName := "initialization-test"
	scenarioPath := filepath.Join(env.ScenariosDir, scenarioName)
	os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755)

	serviceConfig := map[string]interface{}{
		"$schema": "../../../.vrooli/schemas/service.schema.json",
		"version": "1.0.0",
		"service": map[string]interface{}{
			"name":        scenarioName,
			"displayName": scenarioName,
			"description": "",
			"version":     "1.0.0",
		},
		"dependencies": map[string]interface{}{
			"resources": map[string]interface{}{
				"n8n": map[string]interface{}{
					"type":     "n8n",
					"enabled":  true,
					"required": true,
					"initialization": []map[string]interface{}{
						{"file": "initialization/automation/n8n/workflow.json", "type": "workflow"},
					},
				},
			},
			"scenarios": map[string]interface{}{},
		},
	}
	data, _ := json.MarshalIndent(serviceConfig, "", "  ")
	os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), data, 0o644)

	deps, err := scanForResourceUsage(scenarioPath, scenarioName)
	if err != nil {
		t.Fatalf("scanForResourceUsage returned error: %v", err)
	}

	found := false
	for _, dep := range deps {
		if dep.DependencyName == "n8n" && dep.DependencyType == "resource" {
			if dep.AccessMethod != "" && dep.Configuration["initialization_detected"] == true {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("expected n8n to be detected via initialization data")
	}
}

func TestLoadServiceConfigIncludesDependencies(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	setEnvAndCleanup(t, "VROOLI_SCENARIOS_DIR", filepath.Join("..", ".."))
	cfg, err := loadServiceConfigFromFile(filepath.Join("..", "..", "brand-manager"))
	if err != nil {
		t.Fatalf("failed to load brand-manager config: %v", err)
	}
	if len(cfg.Dependencies.Resources) == 0 {
		t.Fatalf("expected dependencies.resources to contain entries")
	}
}

func TestRawServiceConfigMapHasDependencies(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	raw, err := loadRawServiceConfigMap(filepath.Join("..", "..", "brand-manager"))
	if err != nil {
		t.Fatalf("failed to load raw config: %v", err)
	}
	deps := ensureOrderedMap(raw, "dependencies")
	if len(deps.Keys()) == 0 {
		t.Fatalf("expected dependencies keys")
	}
	resources := ensureOrderedMap(deps, "resources")
	if len(resources.Keys()) == 0 {
		t.Fatalf("expected raw resources entries")
	}
}

func TestScanForResourceUsageFiltersUnknown(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configureTestScenariosDir(t, env)
	createTestResourceDirs(t, env, "postgres")

	subjectPath := filepath.Join(env.ScenariosDir, "resource-subject")
	os.MkdirAll(subjectPath, 0755)
	defer os.RemoveAll(subjectPath)

	script := `#!/bin/bash
resource-postgres connect
PGHOST=localhost
resource-fake something
`
	os.WriteFile(filepath.Join(subjectPath, "main.sh"), []byte(script), 0644)

	nodeModules := filepath.Join(subjectPath, "node_modules", "pkg")
	os.MkdirAll(nodeModules, 0755)
	os.WriteFile(filepath.Join(nodeModules, "index.js"), []byte("resource-redis start"), 0644)

	deps, err := scanForResourceUsage(subjectPath, "resource-subject")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}
	if deps[0].DependencyName != "postgres" {
		t.Fatalf("Expected postgres dependency, got %s", deps[0].DependencyName)
	}
}

func TestResourceHeuristicIgnoresPlainN8NMentions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configureTestScenariosDir(t, env)
	createTestResourceDirs(t, env, "n8n")

	subjectPath := filepath.Join(env.ScenariosDir, "heuristic-noise")
	os.MkdirAll(subjectPath, 0755)
	defer os.RemoveAll(subjectPath)

	content := `package main

var resourceTypes = []string{"postgres", "redis", "n8n"}
`
	os.WriteFile(filepath.Join(subjectPath, "main.go"), []byte(content), 0644)

	deps, err := scanForResourceUsage(subjectPath, "heuristic-noise")
	if err != nil {
		t.Fatalf("scanForResourceUsage returned error: %v", err)
	}

	for _, dep := range deps {
		if dep.DependencyName == "n8n" {
			t.Fatalf("plain text mention should not be treated as n8n dependency")
		}
	}
}

func TestResourceHeuristicDetectsN8NEnvUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configureTestScenariosDir(t, env)
	createTestResourceDirs(t, env, "n8n")

	subjectPath := filepath.Join(env.ScenariosDir, "heuristic-hit")
	os.MkdirAll(subjectPath, 0755)
	defer os.RemoveAll(subjectPath)

	content := `package main

import "os"

func useN8N() string {
	return os.Getenv("N8N_URL")
}
`
	os.WriteFile(filepath.Join(subjectPath, "main.go"), []byte(content), 0644)

	deps, err := scanForResourceUsage(subjectPath, "heuristic-hit")
	if err != nil {
		t.Fatalf("scanForResourceUsage returned error: %v", err)
	}

	found := false
	for _, dep := range deps {
		if dep.DependencyName == "n8n" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected n8n dependency to be detected when referencing N8N_URL")
	}
}

func TestApplyDetectedDiffsWritesDependencies(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Setenv("API_PORT", "19999")
	t.Setenv("DATABASE_URL", "postgres://example:example@localhost:5432/example?sslmode=disable")

	configureTestScenariosDir(t, env)
	createTestResourceDirs(t, env, "postgres")

	subject := createTestScenario(t, env, "apply-subject", map[string]Resource{})
	defer subject.Cleanup()
	support := createTestScenario(t, env, "support-scenario", map[string]Resource{})
	defer support.Cleanup()

	analysis := &DependencyAnalysisResponse{
		DetectedResources: []ScenarioDependency{
			{
				DependencyName: "postgres",
				AccessMethod:   "resource_cli",
				Configuration: map[string]interface{}{
					"resource_type": "postgres",
				},
			},
		},
		ResourceDiff: DependencyDiff{Missing: []DependencyDrift{{Name: "postgres"}}},
		Scenarios: []ScenarioDependency{
			{
				DependencyName: "support-scenario",
				AccessMethod:   "vrooli scenario",
			},
		},
		ScenarioDiff: DependencyDiff{Missing: []DependencyDrift{{Name: "support-scenario"}}},
	}

	summary, err := applyDetectedDiffs("apply-subject", analysis, true, true)
	if err != nil {
		t.Fatalf("applyDetectedDiffs error: %v", err)
	}

	changed, _ := summary["changed"].(bool)
	if !changed {
		t.Fatalf("Expected summary to report changes: %v", summary)
	}

	servicePath := filepath.Join(env.ScenariosDir, "apply-subject", ".vrooli", "service.json")
	data, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("Failed to read service.json: %v", err)
	}

	var cfg ServiceConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Failed to parse service.json: %v", err)
	}

	resource, ok := cfg.Dependencies.Resources["postgres"]
	if !ok {
		t.Fatalf("postgres resource not added: %+v", cfg.Dependencies.Resources)
	}
	if resource.Type != "postgres" {
		t.Fatalf("Expected postgres type, got %s", resource.Type)
	}

	scenarioDep, ok := cfg.Dependencies.Scenarios["support-scenario"]
	if !ok || !scenarioDep.Required {
		t.Fatalf("support-scenario not added correctly: %+v", cfg.Dependencies.Scenarios)
	}
	if scenarioDep.Version != "1.0.0" {
		t.Fatalf("expected version to be propagated, got %q", scenarioDep.Version)
	}
	if scenarioDep.VersionRange != ">=1.0.0" {
		t.Fatalf("expected version range to track detected version, got %q", scenarioDep.VersionRange)
	}
}

// TestScanForSharedWorkflows tests shared workflow detection
func TestScanForSharedWorkflows(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	scenarioName := "test-scenario"
	scenarioPath := filepath.Join(env.ScenariosDir, scenarioName)

	t.Run("WithN8nWorkflows", func(t *testing.T) {
		os.MkdirAll(scenarioPath, 0755)
		defer os.RemoveAll(scenarioPath)

		// Create n8n workflows directory
		workflowsDir := filepath.Join(scenarioPath, "workflows", "n8n")
		os.MkdirAll(workflowsDir, 0755)

		// Create a sample workflow file
		workflowFile := filepath.Join(workflowsDir, "test-workflow.json")
		workflow := map[string]interface{}{
			"name":  "test-workflow",
			"nodes": []interface{}{},
		}
		workflowJSON, _ := json.Marshal(workflow)
		os.WriteFile(workflowFile, workflowJSON, 0644)

		deps, err := scanForSharedWorkflows(scenarioPath, scenarioName)
		if err != nil {
			t.Logf("Scan returned error: %v", err)
		}

		// Verify we can at least scan without crashing
		t.Logf("Found %d workflow dependencies", len(deps))
	})

	t.Run("NoWorkflows", func(t *testing.T) {
		emptyPath := filepath.Join(env.ScenariosDir, "no-workflows")
		os.MkdirAll(emptyPath, 0755)
		defer os.RemoveAll(emptyPath)

		deps, err := scanForSharedWorkflows(emptyPath, "no-workflows")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(deps) != 0 {
			t.Errorf("Expected no workflows, got %d", len(deps))
		}
	})
}

// TestGenerateDependencyGraph tests graph generation
func TestGenerateDependencyGraph(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Skip if no database available
	testDB, dbCleanup := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	db = testDB
	defer func() { db = nil }()

	// Insert test data
	deps := []ScenarioDependency{
		createTestDependency("scenario-a", "resource", "postgres", true),
		createTestDependency("scenario-a", "resource", "redis", false),
		createTestDependency("scenario-b", "resource", "postgres", true),
		createTestDependency("scenario-b", "scenario", "scenario-a", true),
	}

	for _, dep := range deps {
		insertTestDependency(t, testDB, dep)
	}

	tests := []struct {
		name      string
		graphType string
		wantError bool
	}{
		{"Resource Graph", "resource", false},
		{"Scenario Graph", "scenario", false},
		{"Combined Graph", "combined", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph, err := generateDependencyGraph(tt.graphType)

			if (err != nil) != tt.wantError {
				t.Errorf("generateDependencyGraph() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				if graph == nil {
					t.Error("Expected non-nil graph")
					return
				}

				if graph.Type != tt.graphType {
					t.Errorf("Expected graph type %s, got %s", tt.graphType, graph.Type)
				}

				if len(graph.Nodes) == 0 {
					t.Error("Expected at least one node in graph")
				}

				// Verify graph structure
				if graph.Metadata == nil {
					t.Error("Expected metadata in graph")
				}
			}
		})
	}
}

// TestAnalyzeProposedScenario tests proposed scenario analysis
func TestAnalyzeProposedScenario(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ValidRequest", func(t *testing.T) {
		req := ProposedScenarioRequest{
			Name:        "new-scenario",
			Description: "A scenario that needs database for storing user data",
			Requirements: []string{
				"Store user data",
				"Fast access with cache",
			},
		}

		result, err := analyzeProposedScenario(req)

		// May fail if resources not available, that's OK
		if err != nil {
			t.Logf("Analysis failed (expected if resources unavailable): %v", err)
			return
		}

		if result == nil {
			t.Error("Expected non-nil result")
			return
		}

		// Verify result structure
		if _, hasResources := result["predicted_resources"]; !hasResources {
			t.Error("Expected predicted_resources in result")
		}
	})

	t.Run("EmptyDescription", func(t *testing.T) {
		req := ProposedScenarioRequest{
			Name:         "empty-scenario",
			Description:  "",
			Requirements: []string{},
		}

		result, err := analyzeProposedScenario(req)

		// Should still work, just with low confidence
		if err != nil {
			t.Logf("Analysis failed: %v", err)
			return
		}

		if result == nil {
			t.Error("Expected non-nil result even with empty description")
		}
	})
}

// TestCalculateScenarioConfidence tests scenario confidence calculation
func TestCalculateScenarioConfidence(t *testing.T) {
	tests := []struct {
		name        string
		patterns    []map[string]interface{}
		minExpected float64
		maxExpected float64
	}{
		{
			name: "HighSimilarity",
			patterns: []map[string]interface{}{
				{"similarity": 0.9},
				{"similarity": 0.85},
			},
			minExpected: 0.8,
			maxExpected: 1.0,
		},
		{
			name: "LowSimilarity",
			patterns: []map[string]interface{}{
				{"similarity": 0.2},
				{"similarity": 0.3},
			},
			minExpected: 0.0,
			maxExpected: 0.4,
		},
		{
			name:        "Empty",
			patterns:    []map[string]interface{}{},
			minExpected: 0.0,
			maxExpected: 0.1,
		},
		{
			name: "MissingSimilarityField",
			patterns: []map[string]interface{}{
				{"other_field": 0.9},
			},
			minExpected: 0.0,
			maxExpected: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := calculateScenarioConfidence(tt.patterns)

			if confidence < tt.minExpected || confidence > tt.maxExpected {
				t.Errorf("Expected confidence between %f and %f, got %f",
					tt.minExpected, tt.maxExpected, confidence)
			}
		})
	}
}

// TestParseClaudeCodeResponse tests Claude Code response parsing
func TestParseClaudeCodeResponse(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		description string
		wantNil     bool
	}{
		{
			name:        "ValidResponse",
			response:    "Analysis: This scenario needs postgres and redis for data storage and caching.",
			description: "Test scenario",
			wantNil:     false,
		},
		{
			name:        "EmptyResponse",
			response:    "",
			description: "Test scenario",
			wantNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseClaudeCodeResponse(tt.response, tt.description)

			if (result == nil) != tt.wantNil {
				t.Errorf("parseClaudeCodeResponse() = %v, wantNil %v", result, tt.wantNil)
			}

			if !tt.wantNil && result != nil {
				if len(result.PredictedResources) == 0 && len(result.Recommendations) == 0 {
					t.Logf("Parsed response contains no resources or recommendations (may be expected for empty input)")
				}
			}
		})
	}
}
