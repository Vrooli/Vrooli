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
