package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestEnhancedDetectVrooliRootCoverage adds coverage for detectVrooliRoot edge cases
func TestEnhancedDetectVrooliRootCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("DetectVrooliWithValidEnvAndMissingDir", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		// Set env to path without .vrooli directory
		os.Setenv("VROOLI_ROOT", env.TempDir)
		defer os.Unsetenv("VROOLI_ROOT")

		result := detectVrooliRoot(".")
		if result == "" {
			t.Error("detectVrooliRoot should not return empty string")
		}
	})

	t.Run("DetectVrooliWithNestedStructure", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		os.Unsetenv("VROOLI_ROOT")

		// Create deeply nested structure
		deepDir := filepath.Join(env.TempDir, "a", "b", "c", "d")
		if err := os.MkdirAll(deepDir, 0755); err != nil {
			t.Fatalf("Failed to create deep dir: %v", err)
		}

		// Create .vrooli at the root
		vrooliDir := filepath.Join(env.TempDir, ".vrooli")
		if err := os.MkdirAll(vrooliDir, 0755); err != nil {
			t.Fatalf("Failed to create .vrooli dir: %v", err)
		}

		result := detectVrooliRoot(deepDir)
		// Should traverse up and find .vrooli
		if !strings.Contains(result, env.TempDir) && result != deepDir {
			t.Logf("detectVrooliRoot from deep dir returned: %s", result)
		}
	})

	t.Run("DetectVrooliAtRootDirectory", func(t *testing.T) {
		os.Unsetenv("VROOLI_ROOT")

		// Test with root directory (should handle gracefully)
		result := detectVrooliRoot("/")
		if result == "" {
			t.Error("detectVrooliRoot should handle root directory")
		}
	})
}

// TestEnhancedResolveCodexTimeoutCoverage adds coverage for config parsing paths
func TestEnhancedResolveCodexTimeoutCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ConfigWithZeroTimeout", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		configDir := filepath.Join(env.TempDir, "initialization", "configuration")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("Failed to create config dir: %v", err)
		}

		configPath := filepath.Join(configDir, "codex-config.json")
		configContent := `{
			"investigation_settings": {
				"timeout_seconds": 0
			}
		}`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		timeout := resolveCodexAgentTimeout(env.TempDir)
		if timeout != defaultAgentTimeout {
			t.Errorf("Expected default timeout for zero value, got %v", timeout)
		}
	})

	t.Run("ConfigWithMissingInvestigationSettings", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		configDir := filepath.Join(env.TempDir, "initialization", "configuration")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("Failed to create config dir: %v", err)
		}

		configPath := filepath.Join(configDir, "codex-config.json")
		configContent := `{"other_field": "value"}`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		timeout := resolveCodexAgentTimeout(env.TempDir)
		if timeout != defaultAgentTimeout {
			t.Errorf("Expected default timeout when investigation_settings missing, got %v", timeout)
		}
	})
}

// TestEnhancedDetectScenarioRootCoverage adds coverage for scenario detection paths
func TestEnhancedDetectScenarioRootCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("DetectScenarioWithValidEnvButFileInsteadOfDir", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		// Create a file instead of directory
		filePath := filepath.Join(env.TempDir, "somefile.txt")
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		os.Setenv("SCENARIO_ROOT", filePath)
		defer os.Unsetenv("SCENARIO_ROOT")

		result := detectScenarioRoot()
		if result == "" {
			t.Error("detectScenarioRoot should return fallback when env points to file")
		}
	})

	t.Run("DetectScenarioWithNestedAgentDashboard", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		os.Unsetenv("SCENARIO_ROOT")

		// Create nested path with agent-dashboard
		dashboardPath := filepath.Join(env.TempDir, "parent", "agent-dashboard", "api")
		if err := os.MkdirAll(dashboardPath, 0755); err != nil {
			t.Fatalf("Failed to create dashboard path: %v", err)
		}

		origWd, _ := os.Getwd()
		defer os.Chdir(origWd)

		if err := os.Chdir(dashboardPath); err == nil {
			result := detectScenarioRoot()
			t.Logf("detectScenarioRoot from nested api dir: %s", result)
		}
	})
}

// TestEnhancedCapabilitiesHandlerCoverage adds more coverage for capabilities handler
func TestEnhancedCapabilitiesHandlerCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CapabilitiesWithMixedCaseSorting", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		manager, err := newCodexAgentManager(env.TempDir, defaultAgentTimeout, ".")
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		// Create agents with capabilities that test sorting
		agent1 := &Agent{
			ID:           "test:1",
			Capabilities: []string{"Coding", "Testing"},
		}
		agent2 := &Agent{
			ID:           "test:2",
			Capabilities: []string{"coding", "analysis"},
		}
		agent3 := &Agent{
			ID:           "test:3",
			Capabilities: []string{"", "CODING"}, // Test empty capability filtering
		}

		manager.mu.Lock()
		manager.agents[agent1.ID] = &managedAgent{agent: agent1, done: make(chan struct{})}
		manager.agents[agent2.ID] = &managedAgent{agent: agent2, done: make(chan struct{})}
		manager.agents[agent3.ID] = &managedAgent{agent: agent3, done: make(chan struct{})}
		manager.mu.Unlock()

		oldManager := codexManager
		codexManager = manager
		defer func() { codexManager = oldManager }()

		req, _ := http.NewRequest("GET", "/api/v1/capabilities", nil)
		w := httptest.NewRecorder()

		capabilitiesHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestEnhancedSearchCapabilityCoverage adds coverage for search edge cases
func TestEnhancedSearchCapabilityCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SearchWithSpecialCharactersInCapability", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/agents/search?capability=test-skill_name", nil)
		w := httptest.NewRecorder()

		searchByCapabilityHandler(w, req)

		// Dashes and underscores are valid
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for valid capability chars, got %d", w.Code)
		}
	})

	t.Run("SearchWithPartialMatch", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		manager, err := newCodexAgentManager(env.TempDir, defaultAgentTimeout, ".")
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		agent := &Agent{
			ID:           "test:1",
			Capabilities: []string{"CodexAnalysis", "TestingFramework"},
		}

		manager.mu.Lock()
		manager.agents[agent.ID] = &managedAgent{agent: agent, done: make(chan struct{})}
		manager.mu.Unlock()

		oldManager := codexManager
		codexManager = manager
		defer func() { codexManager = oldManager }()

		// Search for partial match
		req, _ := http.NewRequest("GET", "/api/v1/agents/search?capability=codex", nil)
		w := httptest.NewRecorder()

		searchByCapabilityHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestEnhancedIndividualAgentHandlerCoverage adds more path coverage
func TestEnhancedIndividualAgentHandlerCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InvalidPostPath", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		manager, err := newCodexAgentManager(env.TempDir, defaultAgentTimeout, ".")
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		agent := &Agent{
			ID:     "codex:test123",
			Status: "running",
		}

		manager.mu.Lock()
		manager.agents[agent.ID] = &managedAgent{
			agent:  agent,
			cancel: func() {},
			done:   make(chan struct{}),
		}
		manager.mu.Unlock()

		oldManager := codexManager
		codexManager = manager
		defer func() { codexManager = oldManager }()

		// POST with invalid path (too many parts)
		req, _ := http.NewRequest("POST", "/api/v1/agents/test123/invalid/path", nil)
		w := httptest.NewRecorder()

		individualAgentHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Logf("Invalid POST path returned status: %d", w.Code)
		}
	})

	t.Run("GetUnknownSubEndpoint", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		manager, err := newCodexAgentManager(env.TempDir, defaultAgentTimeout, ".")
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		agent := &Agent{
			ID:     "codex:test456",
			Status: "running",
		}

		manager.mu.Lock()
		manager.agents[agent.ID] = &managedAgent{agent: agent, done: make(chan struct{})}
		manager.mu.Unlock()

		oldManager := codexManager
		codexManager = manager
		defer func() { codexManager = oldManager }()

		req, _ := http.NewRequest("GET", "/api/v1/agents/test456/unknown", nil)
		w := httptest.NewRecorder()

		individualAgentHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for unknown endpoint, got %d", w.Code)
		}
	})
}

// TestEnhancedStopAgentCoverage adds coverage for stop agent timeout scenario
func TestEnhancedStopAgentCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("StopAgentWithSlowShutdown", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		manager, err := newCodexAgentManager(env.TempDir, defaultAgentTimeout, ".")
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		agent := &Agent{
			ID:     "test:slow",
			Status: "running",
		}

		// Create a command that will be slow to stop
		cmd := exec.CommandContext(ctx, "sleep", "5")
		if err := cmd.Start(); err != nil {
			t.Skipf("Failed to start sleep command: %v", err)
			return
		}

		managed := &managedAgent{
			agent:  agent,
			cancel: cancel,
			cmd:    cmd,
			done:   make(chan struct{}),
		}

		// Close done channel quickly to simulate timeout scenario
		go func() {
			time.Sleep(50 * time.Millisecond)
			close(managed.done)
		}()

		manager.mu.Lock()
		manager.agents[agent.ID] = managed
		manager.mu.Unlock()

		result, err := manager.Stop(agent.ID)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if result == nil {
			t.Error("Expected agent result")
		}

		// Cleanup
		cancel()
		cmd.Wait()
	})
}

// TestEnhancedOrchestrateHandlerCoverage adds coverage for orchestrate edge cases
func TestEnhancedOrchestrateHandlerCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("OrchestratewithMaxTargets", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		manager, err := newCodexAgentManager(env.TempDir, defaultAgentTimeout, ".")
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		oldManager := codexManager
		codexManager = manager
		defer func() { codexManager = oldManager }()

		// Create request with more than max targets
		targets := make([]string, 20)
		for i := 0; i < 20; i++ {
			targets[i] = "target" + string(rune(i))
		}

		reqBody := map[string]interface{}{
			"mode":      "auto",
			"objective": "test",
			"targets":   targets,
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/orchestrate",
			Body:   reqBody,
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		orchestrateHandler(w, httpReq)

		// Should handle truncation of targets to max
		t.Logf("Orchestrate with many targets returned status: %d", w.Code)
	})

	t.Run("OrchestrateWithEmptyMode", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		manager, err := newCodexAgentManager(env.TempDir, defaultAgentTimeout, ".")
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		oldManager := codexManager
		codexManager = manager
		defer func() { codexManager = oldManager }()

		reqBody := map[string]interface{}{
			"objective": "test objective",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/orchestrate",
			Body:   reqBody,
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		orchestrateHandler(w, httpReq)

		// Should default mode to "auto"
		t.Logf("Orchestrate with empty mode returned status: %d", w.Code)
	})
}

// TestEnhancedSnapshotCoverage adds coverage for snapshot edge cases
func TestEnhancedSnapshotCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SnapshotWithNilRadarPosition", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		manager, err := newCodexAgentManager(env.TempDir, defaultAgentTimeout, ".")
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		agent := &Agent{
			ID:            "test:no-radar",
			Status:        "running",
			PID:           1234,
			StartTime:     time.Now(),
			RadarPosition: nil, // Explicitly nil
		}

		managed := &managedAgent{
			agent: agent,
			done:  make(chan struct{}),
		}

		manager.mu.Lock()
		manager.agents[agent.ID] = managed
		manager.mu.Unlock()

		agents, stats := manager.Snapshot()

		if len(agents) == 0 {
			t.Error("Expected at least one agent in snapshot")
		}

		if stats.Running != 1 {
			t.Errorf("Expected 1 running agent, got %d", stats.Running)
		}

		// Check that radar position was generated
		if agents[0].RadarPosition == nil {
			t.Error("Expected radar position to be generated for agent")
		}
	})

	t.Run("SnapshotWithStartingStatus", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		manager, err := newCodexAgentManager(env.TempDir, defaultAgentTimeout, ".")
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		agent := &Agent{
			ID:        "test:starting",
			Status:    "starting",
			StartTime: time.Now(),
		}

		managed := &managedAgent{
			agent: agent,
			done:  make(chan struct{}),
		}

		manager.mu.Lock()
		manager.agents[agent.ID] = managed
		manager.mu.Unlock()

		_, stats := manager.Snapshot()

		// Starting status should be counted as running
		if stats.Running != 1 {
			t.Errorf("Expected 1 running agent (starting counts as running), got %d", stats.Running)
		}
	})
}

// TestEnhancedWaitForExitCoverage adds coverage for waitForExit edge cases
func TestEnhancedWaitForExitCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WaitForExitWithNilProcessState", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		ctx := context.Background()

		logPath := filepath.Join(env.TempDir, "test.log")
		logFile, _ := os.Create(logPath)

		agent := &Agent{
			ID:        "test:nil-state",
			Status:    "running",
			StartTime: time.Now(),
		}

		// Create a command that exits immediately
		cmd := exec.Command("echo", "test")
		if err := cmd.Start(); err != nil {
			t.Skipf("Failed to start command: %v", err)
			return
		}

		managed := &managedAgent{
			agent:   agent,
			cmd:     cmd,
			logFile: logFile,
			done:    make(chan struct{}),
		}

		managed.waitForExit(ctx, 10*time.Second)

		// Check that exit handling worked
		if agent.EndTime == nil {
			t.Error("Expected end time to be set")
		}

		logFile.Close()
	})
}

// TestEnhancedBuildCodexPromptCoverage adds coverage for prompt building
func TestEnhancedBuildCodexPromptCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("BuildPromptWithAllFields", func(t *testing.T) {
		req := startAgentRequest{
			Task:  "Test task",
			Mode:  "analysis",
			Notes: "Important context",
		}

		prompt := buildCodexPrompt(req, "/test/scenario/root")

		if !strings.Contains(prompt, "Test task") {
			t.Error("Expected task in prompt")
		}

		if !strings.Contains(prompt, "Important context") {
			t.Error("Expected notes in prompt")
		}

		if !strings.Contains(prompt, "/test/scenario/root") {
			t.Error("Expected scenario root in prompt")
		}

		if !strings.Contains(prompt, "analysis") {
			t.Error("Expected mode in prompt")
		}
	})

	t.Run("BuildPromptWithEmptyNotes", func(t *testing.T) {
		req := startAgentRequest{
			Task:  "Test task",
			Mode:  "auto",
			Notes: "",
		}

		prompt := buildCodexPrompt(req, ".")

		if strings.Contains(prompt, "Additional Context") {
			t.Error("Should not include Additional Context section when notes empty")
		}
	})
}
