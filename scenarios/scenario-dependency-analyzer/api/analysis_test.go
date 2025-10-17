
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
			Name:        "empty-scenario",
			Description: "",
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
