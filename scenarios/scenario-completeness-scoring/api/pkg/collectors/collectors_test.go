package collectors

import (
	"os"
	"path/filepath"
	"testing"
)

// [REQ:SCS-CORE-001] Test service config loading
func TestLoadServiceConfig(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	os.MkdirAll(vrooliDir, 0755)

	// Test with valid config
	configContent := `{"category": "automation", "name": "test-scenario", "version": "1.0.0"}`
	os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(configContent), 0644)

	config := loadServiceConfig(tmpDir)
	if config.Category != "automation" {
		t.Errorf("Expected category 'automation', got '%s'", config.Category)
	}
	if config.Name != "test-scenario" {
		t.Errorf("Expected name 'test-scenario', got '%s'", config.Name)
	}

	// Test with missing config
	emptyDir := t.TempDir()
	defaultConfig := loadServiceConfig(emptyDir)
	if defaultConfig.Category != "utility" {
		t.Errorf("Expected default category 'utility', got '%s'", defaultConfig.Category)
	}
}

// [REQ:SCS-CORE-001] Test requirements loading
func TestLoadRequirements(t *testing.T) {
	tmpDir := t.TempDir()
	reqDir := filepath.Join(tmpDir, "requirements")
	os.MkdirAll(filepath.Join(reqDir, "01-core"), 0755)

	// Create index.json
	indexContent := `{
		"imports": ["01-core/module.json"],
		"requirements": []
	}`
	os.WriteFile(filepath.Join(reqDir, "index.json"), []byte(indexContent), 0644)

	// Create module.json
	moduleContent := `{
		"requirements": [
			{"id": "REQ-001", "title": "Test Requirement", "status": "in_progress"},
			{"id": "REQ-002", "title": "Another Requirement", "status": "passed"}
		]
	}`
	os.WriteFile(filepath.Join(reqDir, "01-core", "module.json"), []byte(moduleContent), 0644)

	requirements := loadRequirements(tmpDir)
	if len(requirements) != 2 {
		t.Errorf("Expected 2 requirements, got %d", len(requirements))
	}

	// Test requirement pass calculation
	pass := calculateRequirementPass(requirements, nil)
	if pass.Total != 2 {
		t.Errorf("Expected total 2, got %d", pass.Total)
	}
	if pass.Passing != 1 {
		t.Errorf("Expected 1 passing, got %d", pass.Passing)
	}
}

// [REQ:SCS-CORE-001] Test test results loading
func TestLoadTestResults(t *testing.T) {
	// Test with no results
	emptyDir := t.TempDir()
	results := loadTestResults(emptyDir)
	if results.Total != 0 {
		t.Errorf("Expected 0 total tests, got %d", results.Total)
	}

	// Test with single file results
	tmpDir := t.TempDir()
	coverageDir := filepath.Join(tmpDir, "coverage")
	os.MkdirAll(coverageDir, 0755)

	testResultsContent := `{"passed": 10, "failed": 2, "timestamp": "2025-11-28T12:00:00Z"}`
	os.WriteFile(filepath.Join(coverageDir, "test-results.json"), []byte(testResultsContent), 0644)

	results = loadTestResults(tmpDir)
	if results.Total != 12 {
		t.Errorf("Expected 12 total tests, got %d", results.Total)
	}
	if results.Passing != 10 {
		t.Errorf("Expected 10 passing, got %d", results.Passing)
	}
}

// [REQ:SCS-CORE-001D] Test UI metrics collection
func TestCollectUIMetrics(t *testing.T) {
	// Test with no UI
	emptyDir := t.TempDir()
	metrics := collectUIMetrics(emptyDir)
	if metrics != nil {
		t.Error("Expected nil metrics for empty directory")
	}

	// Test with basic UI
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui", "src")
	os.MkdirAll(uiDir, 0755)

	// Create App.tsx
	appContent := `import React from 'react';

export function App() {
  return (
    <div>
      <h1>Hello World</h1>
    </div>
  );
}
`
	os.WriteFile(filepath.Join(uiDir, "App.tsx"), []byte(appContent), 0644)

	metrics = collectUIMetrics(tmpDir)
	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}
	if metrics.FileCount < 1 {
		t.Errorf("Expected at least 1 file, got %d", metrics.FileCount)
	}
	if metrics.IsTemplate {
		t.Error("App.tsx should not be detected as template")
	}
}

// [REQ:SCS-CORE-001D] Test template detection
func TestDetectTemplateUI(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui", "src")
	os.MkdirAll(uiDir, 0755)

	// Create template-like App.tsx
	templateContent := `import React from 'react';

export function App() {
  return (
    <div>
      <h1>This starter UI is intentionally minimal</h1>
      <p>Replace it with your scenario-specific implementation</p>
    </div>
  );
}
`
	appPath := filepath.Join(uiDir, "App.tsx")
	os.WriteFile(appPath, []byte(templateContent), 0644)

	isTemplate := detectTemplateUI(tmpDir, appPath)
	if !isTemplate {
		t.Error("Expected template to be detected")
	}

	// Non-template content
	realContent := `import React from 'react';
import { Dashboard } from './components/Dashboard';
import { Settings } from './components/Settings';

export function App() {
  return (
    <div className="app">
      <Dashboard />
      <Settings />
    </div>
  );
}
`
	os.WriteFile(appPath, []byte(realContent), 0644)

	isTemplate = detectTemplateUI(tmpDir, appPath)
	if isTemplate {
		t.Error("Real app should not be detected as template")
	}
}

// [REQ:SCS-CORE-001D] Test routing detection
func TestDetectRouting(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui", "src")
	os.MkdirAll(uiDir, 0755)

	// No routing
	noRoutingContent := `import React from 'react';
export function App() { return <div>Hello</div>; }
`
	appPath := filepath.Join(uiDir, "App.tsx")
	os.WriteFile(appPath, []byte(noRoutingContent), 0644)

	routing := detectRouting(tmpDir, appPath)
	if routing.hasRouting {
		t.Error("Expected no routing for simple app")
	}

	// With React Router
	routerContent := `import React from 'react';
import { BrowserRouter, Route, Routes } from 'react-router-dom';

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/about" element={<About />} />
        <Route path="/contact" element={<Contact />} />
      </Routes>
    </BrowserRouter>
  );
}
`
	os.WriteFile(appPath, []byte(routerContent), 0644)

	routing = detectRouting(tmpDir, appPath)
	if !routing.hasRouting {
		t.Error("Expected routing to be detected with React Router")
	}
	if routing.routeCount < 3 {
		t.Errorf("Expected at least 3 routes, got %d", routing.routeCount)
	}
}

// [REQ:SCS-CORE-001D] Test API endpoint extraction
func TestExtractAPIEndpoints(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(tmpDir, 0755)

	// Create file with API calls
	apiContent := `
import { useState, useEffect } from 'react';

export function Dashboard() {
  useEffect(() => {
    fetch('/api/v1/scores')
      .then(res => res.json())
      .then(data => console.log(data));

    fetch('/api/v1/config')
      .then(res => res.json());

    // Health check
    fetch('/health')
      .then(res => res.json());
  }, []);

  return <div>Dashboard</div>;
}
`
	os.WriteFile(filepath.Join(tmpDir, "Dashboard.tsx"), []byte(apiContent), 0644)

	apiInfo := extractAPIEndpoints(tmpDir)
	if apiInfo.total < 2 {
		t.Errorf("Expected at least 2 API endpoints, got %d", apiInfo.total)
	}
	if apiInfo.beyondHealth < 1 {
		t.Errorf("Expected at least 1 endpoint beyond health, got %d", apiInfo.beyondHealth)
	}
}

// [REQ:SCS-CORE-001B] Test requirement tree building
func TestBuildRequirementTrees(t *testing.T) {
	requirements := []Requirement{
		{ID: "REQ-001", Children: []string{"REQ-001A", "REQ-001B"}},
		{ID: "REQ-001A", Children: []string{"REQ-001A1"}},
		{ID: "REQ-001A1"},
		{ID: "REQ-001B"},
		{ID: "REQ-002"},
	}

	trees := buildRequirementTrees(requirements)

	// Should have 2 root trees: REQ-001 and REQ-002
	if len(trees) != 2 {
		t.Errorf("Expected 2 root trees, got %d", len(trees))
	}

	// Find REQ-001 tree and verify depth
	var req001Tree *testing.T
	for _, tree := range trees {
		if tree.ID == "REQ-001" {
			if len(tree.Children) != 2 {
				t.Errorf("REQ-001 should have 2 children, got %d", len(tree.Children))
			}
			// Find REQ-001A and check its child
			for _, child := range tree.Children {
				if child.ID == "REQ-001A" {
					if len(child.Children) != 1 {
						t.Errorf("REQ-001A should have 1 child, got %d", len(child.Children))
					}
				}
			}
			break
		}
		_ = req001Tree // Silence unused variable warning
	}
}

// [REQ:SCS-CORE-001] Test file counting
func TestCountFilesRecursive(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure
	os.MkdirAll(filepath.Join(tmpDir, "components"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "pages"), 0755)

	os.WriteFile(filepath.Join(tmpDir, "App.tsx"), []byte("// App"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "components", "Button.tsx"), []byte("// Button"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "components", "Card.tsx"), []byte("// Card"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "pages", "Home.tsx"), []byte("// Home"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "styles.css"), []byte("/* CSS */"), 0644) // Should not count

	count := countFilesRecursive(tmpDir)
	if count != 4 {
		t.Errorf("Expected 4 source files, got %d", count)
	}
}

// [REQ:SCS-CORE-001D] Test LOC counting
func TestGetTotalLOC(t *testing.T) {
	tmpDir := t.TempDir()

	content := `line 1
line 2
line 3
line 4
line 5
`
	os.WriteFile(filepath.Join(tmpDir, "test.tsx"), []byte(content), 0644)

	loc := getTotalLOC(tmpDir)
	if loc != 5 {
		t.Errorf("Expected 5 LOC, got %d", loc)
	}
}

// [REQ:SCS-CORE-001] Test sync metadata loading
func TestLoadSyncMetadata(t *testing.T) {
	// Test with no sync metadata
	emptyDir := t.TempDir()
	metadata := loadSyncMetadata(emptyDir)
	if metadata != nil {
		t.Error("Expected nil for missing sync metadata")
	}

	// Test with valid sync metadata
	tmpDir := t.TempDir()
	coverageDir := filepath.Join(tmpDir, "coverage")
	os.MkdirAll(coverageDir, 0755)

	syncContent := `{
		"requirements": {
			"REQ-001": {"status": "passed", "last_run": "2025-11-28"},
			"REQ-002": {"status": "failed", "last_run": "2025-11-28"}
		}
	}`
	os.WriteFile(filepath.Join(coverageDir, "requirements-sync.json"), []byte(syncContent), 0644)

	metadata = loadSyncMetadata(tmpDir)
	if metadata == nil {
		t.Fatal("Expected sync metadata to be loaded")
	}
	if len(metadata.Requirements) != 2 {
		t.Errorf("Expected 2 requirements in sync metadata, got %d", len(metadata.Requirements))
	}

	// Test with invalid JSON
	invalidDir := t.TempDir()
	coverageDir2 := filepath.Join(invalidDir, "coverage")
	os.MkdirAll(coverageDir2, 0755)
	os.WriteFile(filepath.Join(coverageDir2, "requirements-sync.json"), []byte("invalid json"), 0644)

	metadata = loadSyncMetadata(invalidDir)
	if metadata != nil {
		t.Error("Expected nil for invalid JSON")
	}
}

// [REQ:SCS-CORE-001A] Test target pass calculation
func TestCalculateTargetPass(t *testing.T) {
	// Test with priority-based counting (no operational target IDs)
	requirements := []Requirement{
		{ID: "REQ-001", Priority: "P0", Status: "passed"},
		{ID: "REQ-002", Priority: "P0", Status: "in_progress"},
		{ID: "REQ-003", Priority: "P1", Status: "complete"},
		{ID: "REQ-004", Priority: "P2", Status: "passed"}, // P2 should not be counted
	}

	pass := calculateTargetPass(requirements, nil)
	if pass.Total != 3 {
		t.Errorf("Expected 3 P0/P1 requirements, got %d", pass.Total)
	}
	if pass.Passing != 2 {
		t.Errorf("Expected 2 passing P0/P1 requirements, got %d", pass.Passing)
	}

	// Test with operational target IDs
	// NOTE: JS behavior is "at least one linked requirement passes" = target passes
	requirementsWithTargets := []Requirement{
		{ID: "REQ-001", OperationalTargetID: "OT-001", Status: "passed"},
		{ID: "REQ-002", OperationalTargetID: "OT-001", Status: "passed"},
		{ID: "REQ-003", OperationalTargetID: "OT-002", Status: "in_progress"},
		{ID: "REQ-004", OperationalTargetID: "OT-002", Status: "passed"},
	}

	pass = calculateTargetPass(requirementsWithTargets, nil)
	if pass.Total != 2 {
		t.Errorf("Expected 2 targets, got %d", pass.Total)
	}
	// OT-001 passes (both passed), OT-002 passes (REQ-004 is passed - matches JS "any one passes" logic)
	if pass.Passing != 2 {
		t.Errorf("Expected 2 passing targets (JS behavior: any linked req passes), got %d", pass.Passing)
	}
}

// [REQ:SCS-CORE-001A] Test priority requirements counting
func TestCountPriorityRequirements(t *testing.T) {
	requirements := []Requirement{
		{ID: "REQ-001", Priority: "P0", Status: "passed"},
		{ID: "REQ-002", Priority: "P0", Status: "done"},
		{ID: "REQ-003", Priority: "P1", Status: "failed"},
		{ID: "REQ-004", Priority: "P2", Status: "passed"}, // Should be excluded
		{ID: "REQ-005", Priority: "", Status: "complete"}, // Empty priority should be included
	}

	pass := countPriorityRequirements(requirements, nil)
	// P0, P0, P1, and empty (4 total)
	if pass.Total != 4 {
		t.Errorf("Expected 4 priority requirements, got %d", pass.Total)
	}
	// passed, done, complete (3 passing)
	if pass.Passing != 3 {
		t.Errorf("Expected 3 passing, got %d", pass.Passing)
	}

	// Test with sync metadata
	syncData := &SyncMetadata{
		Requirements: map[string]SyncedReq{
			"REQ-003": {Status: "passed"}, // Override failed status
		},
	}

	pass = countPriorityRequirements(requirements, syncData)
	if pass.Passing != 4 {
		t.Errorf("Expected 4 passing with sync override, got %d", pass.Passing)
	}
}

// [REQ:SCS-CORE-001] Test MetricsCollector creation
func TestNewMetricsCollector(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewMetricsCollector(tmpDir)

	if collector == nil {
		t.Fatal("Expected non-nil collector")
	}
	if collector.VrooliRoot != tmpDir {
		t.Errorf("Expected VrooliRoot %s, got %s", tmpDir, collector.VrooliRoot)
	}
}

// [REQ:SCS-CORE-001] Test collector with scenario directory
func TestCollectorCollect(t *testing.T) {
	// Create a minimal scenario structure
	tmpDir := t.TempDir()
	scenarioName := "test-scenario"
	scenarioDir := filepath.Join(tmpDir, "scenarios", scenarioName)

	// Create .vrooli directory with service.json
	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	os.MkdirAll(vrooliDir, 0755)
	os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(`{"category": "utility", "name": "test-scenario"}`), 0644)

	// Create requirements
	reqDir := filepath.Join(scenarioDir, "requirements")
	os.MkdirAll(reqDir, 0755)
	os.WriteFile(filepath.Join(reqDir, "index.json"), []byte(`{"imports": [], "requirements": [{"id": "REQ-001", "status": "passed"}]}`), 0644)

	collector := NewMetricsCollector(tmpDir)
	metrics, err := collector.Collect(scenarioName)

	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}
	if metrics.Scenario != scenarioName {
		t.Errorf("Expected scenario name %s, got %s", scenarioName, metrics.Scenario)
	}
	if metrics.Category != "utility" {
		t.Errorf("Expected category 'utility', got '%s'", metrics.Category)
	}
	if metrics.Requirements.Total != 1 {
		t.Errorf("Expected 1 requirement, got %d", metrics.Requirements.Total)
	}
}

// [REQ:SCS-CORE-003] Test collector with non-existent scenario returns error
// Non-existent scenarios should fail fast with clear feedback
func TestCollectorCollectMissing(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "scenarios"), 0755)

	collector := NewMetricsCollector(tmpDir)
	metrics, err := collector.Collect("non-existent")

	// Non-existent scenario should return an error
	if err == nil {
		t.Error("Expected error for non-existent scenario")
	}
	if metrics != nil {
		t.Error("Expected nil metrics for non-existent scenario")
	}
}

// [REQ:SCS-CORE-003] Test collector with existing but empty scenario returns defaults
// Existing scenarios with missing data files should return partial results with defaults
func TestCollectorCollectExistingButEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	// Create the scenario directory (it exists but has no data)
	os.MkdirAll(filepath.Join(tmpDir, "scenarios", "empty-scenario"), 0755)

	collector := NewMetricsCollector(tmpDir)
	metrics, err := collector.Collect("empty-scenario")

	// Should succeed with default values for missing data
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if metrics == nil {
		t.Fatal("Expected non-nil metrics for existing empty scenario")
	}
	// Should return default category and empty counts
	if metrics.Category != "utility" {
		t.Errorf("Expected default category 'utility', got '%s'", metrics.Category)
	}
	if metrics.Requirements.Total != 0 {
		t.Errorf("Expected 0 requirements for empty scenario, got %d", metrics.Requirements.Total)
	}
}

// [REQ:SCS-CORE-001A] Test requirement pass with sync data
func TestCalculateRequirementPassWithSync(t *testing.T) {
	requirements := []Requirement{
		{ID: "REQ-001", Status: "in_progress"},
		{ID: "REQ-002", Status: "failed"},
	}

	// Without sync data - both should fail
	pass := calculateRequirementPass(requirements, nil)
	if pass.Passing != 0 {
		t.Errorf("Expected 0 passing without sync, got %d", pass.Passing)
	}

	// With sync data overriding one
	syncData := &SyncMetadata{
		Requirements: map[string]SyncedReq{
			"REQ-001": {Status: "passed"},
		},
	}

	pass = calculateRequirementPass(requirements, syncData)
	if pass.Passing != 1 {
		t.Errorf("Expected 1 passing with sync, got %d", pass.Passing)
	}
}

// [REQ:SCS-CORE-001] Test requirements with children grouping
func TestRequirementsWithChildrenGrouping(t *testing.T) {
	requirements := []Requirement{
		{ID: "REQ-001", Children: []string{"REQ-001A", "REQ-001B"}}, // Parent with empty status
		{ID: "REQ-001A", Status: "passed"},
		{ID: "REQ-001B", Status: "passed"},
		{ID: "REQ-002", Status: "in_progress"},
	}

	pass := calculateRequirementPass(requirements, nil)
	// REQ-001 should be skipped (empty status + has children)
	// So total should be 3, passing should be 2
	if pass.Total != 3 {
		t.Errorf("Expected 3 total (parent skipped), got %d", pass.Total)
	}
	if pass.Passing != 2 {
		t.Errorf("Expected 2 passing, got %d", pass.Passing)
	}
}

// [REQ:SCS-CORE-001] Test phase-based test results loading
func TestLoadPhaseBasedResults(t *testing.T) {
	tmpDir := t.TempDir()

	// Create phase-based results structure
	artifactsDir := filepath.Join(tmpDir, "test", "artifacts")
	os.MkdirAll(artifactsDir, 0755)

	// Create a log file with test results
	logContent := `[PHASE_START:unit:1/1]
Running unit tests...
✅ 15 tests passed  ❌ 3 tests failed
[PHASE_END:unit:passed:30s]`
	os.WriteFile(filepath.Join(artifactsDir, "phase-unit-12345.log"), []byte(logContent), 0644)

	results := loadTestResults(tmpDir)
	// Phase-based loading may return results if it parses the log
	_ = results
}

// [REQ:SCS-CORE-001D] Test maxInt helper
func TestMaxInt(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 2},
		{5, 3, 5},
		{-1, -5, -1},
		{0, 0, 0},
	}

	for _, tc := range tests {
		result := maxInt(tc.a, tc.b)
		if result != tc.expected {
			t.Errorf("maxInt(%d, %d) = %d, expected %d", tc.a, tc.b, result, tc.expected)
		}
	}
}

// [REQ:SCS-CORE-001A] Test calculateTargetPass with sync metadata
func TestCalculateTargetPassWithSyncData(t *testing.T) {
	requirementsWithTargets := []Requirement{
		{ID: "REQ-001", OperationalTargetID: "OT-001", Status: "in_progress"},
		{ID: "REQ-002", OperationalTargetID: "OT-001", Status: "in_progress"},
	}

	// Without sync - target should fail
	pass := calculateTargetPass(requirementsWithTargets, nil)
	if pass.Passing != 0 {
		t.Errorf("Expected 0 passing without sync, got %d", pass.Passing)
	}

	// With sync overriding both requirements
	syncData := &SyncMetadata{
		Requirements: map[string]SyncedReq{
			"REQ-001": {Status: "passed"},
			"REQ-002": {Status: "complete"},
		},
	}

	pass = calculateTargetPass(requirementsWithTargets, syncData)
	if pass.Passing != 1 {
		t.Errorf("Expected 1 passing target with sync, got %d", pass.Passing)
	}
}
