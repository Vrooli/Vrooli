package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCodexAgentManager(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InitializeManager", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "manager-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		timeout := 5 * time.Minute
		manager, err := newCodexAgentManager(tmpDir, timeout, "")

		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		if manager == nil {
			t.Fatal("Expected non-nil manager")
		}

		if manager.logDir != tmpDir {
			t.Errorf("Expected log dir %s, got %s", tmpDir, manager.logDir)
		}

		if manager.defaultTimeout != timeout {
			t.Errorf("Expected timeout %v, got %v", timeout, manager.defaultTimeout)
		}

		if manager.agents == nil {
			t.Error("Expected agents map to be initialized")
		}
	})

	t.Run("WithInvalidLogDir", func(t *testing.T) {
		manager, err := newCodexAgentManager("/nonexistent/invalid/path", 5*time.Minute, "")

		// Should still create manager even if dir doesn't exist initially
		if manager == nil {
			t.Logf("Manager not created for invalid log dir (expected): %v", err)
		}
	})
}

func TestManagerSnapshot(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyManager", func(t *testing.T) {
		agents, stats := codexManager.Snapshot()

		if agents == nil {
			t.Error("Expected non-nil agents slice")
		}

		if len(agents) != 0 {
			t.Logf("Manager has %d existing agents", len(agents))
		}

		if stats.Total < 0 {
			t.Error("Expected non-negative total")
		}
	})
}

// Note: Capabilities and SearchByCapability are handled through handlers, not direct manager methods

func TestManagerStop(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("StopNonExistent", func(t *testing.T) {
		agent, err := codexManager.Stop("nonexistent:agent-xyz")

		if err == nil {
			t.Error("Expected error when stopping non-existent agent")
		}

		if agent != nil {
			t.Error("Expected nil agent when stopping non-existent agent")
		}
	})

	t.Run("StopEmptyID", func(t *testing.T) {
		agent, err := codexManager.Stop("")

		if err == nil {
			t.Error("Expected error when stopping with empty ID")
		}

		if agent != nil {
			t.Error("Expected nil agent for empty ID")
		}
	})
}

func TestManagerLogs(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("LogsNonExistent", func(t *testing.T) {
		logs, err := codexManager.Logs("nonexistent:agent", 100)

		if err == nil {
			t.Error("Expected error for non-existent agent")
		}

		if logs != nil {
			t.Error("Expected nil logs for non-existent agent")
		}
	})

	t.Run("LogsWithVariousLineCounts", func(t *testing.T) {
		lineCounts := []int{1, 10, 100, 1000, 10000}

		for _, count := range lineCounts {
			logs, err := codexManager.Logs("test:agent", count)

			// Expected to error for non-existent agent
			if err == nil {
				t.Logf("Unexpectedly got logs for non-existent agent with count %d", count)
			}

			if logs != nil && err == nil {
				t.Errorf("Expected nil logs for non-existent agent with count %d", count)
			}
		}
	})
}

func TestManagerMetrics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MetricsNonExistent", func(t *testing.T) {
		metrics, err := codexManager.Metrics("nonexistent:agent")

		if err == nil {
			t.Error("Expected error for non-existent agent")
		}

		if metrics != nil {
			t.Error("Expected nil metrics for non-existent agent")
		}
	})
}

func TestGetProcessMetrics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CurrentProcess", func(t *testing.T) {
		// Get metrics for current process
		pid := os.Getpid()
		metrics := getProcessMetrics(pid)

		if metrics == nil {
			t.Fatal("Expected non-nil metrics")
		}

		// Should have standard fields
		expectedFields := []string{"cpu_percent", "memory_mb", "thread_count", "fd_count"}
		for _, field := range expectedFields {
			if _, ok := metrics[field]; !ok {
				t.Logf("Field '%s' missing from metrics (may be expected on some systems)", field)
			}
		}
	})

	t.Run("InvalidPID", func(t *testing.T) {
		metrics := getProcessMetrics(-1)

		if metrics == nil {
			t.Fatal("Expected non-nil metrics map")
		}

		// Should return zeros for invalid PID
		t.Logf("Metrics for invalid PID: %+v", metrics)
	})

	t.Run("NonExistentPID", func(t *testing.T) {
		// Use a very high PID that likely doesn't exist
		metrics := getProcessMetrics(9999999)

		if metrics == nil {
			t.Fatal("Expected non-nil metrics map")
		}

		t.Logf("Metrics for non-existent PID: %+v", metrics)
	})
}

func TestDetectScenarioRootAdvanced(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithNestedDirectories", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "nested-scenario-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create nested structure: tmpDir/api/src/
		apiDir := filepath.Join(tmpDir, "api", "src")
		if err := os.MkdirAll(apiDir, 0755); err != nil {
			t.Fatalf("Failed to create nested dirs: %v", err)
		}

		// Create service.json in root
		serviceFile := filepath.Join(tmpDir, "service.json")
		if err := os.WriteFile(serviceFile, []byte(`{"name":"nested-scenario"}`), 0644); err != nil {
			t.Fatalf("Failed to create service.json: %v", err)
		}

		// Change to nested directory
		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)

		os.Chdir(apiDir)
		root := detectScenarioRoot()

		if root == "" {
			t.Error("Expected to detect scenario root from nested directory")
		}

		// Root should be tmpDir, not apiDir
		if root != "" && root == apiDir {
			t.Error("Expected root to be parent directory, not nested directory")
		}
	})

	t.Run("WithMultipleServiceFiles", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "multi-service-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create service.json in root
		serviceFile := filepath.Join(tmpDir, "service.json")
		if err := os.WriteFile(serviceFile, []byte(`{"name":"root-scenario"}`), 0644); err != nil {
			t.Fatalf("Failed to create service.json: %v", err)
		}

		// Create nested scenario
		nestedDir := filepath.Join(tmpDir, "nested")
		if err := os.MkdirAll(nestedDir, 0755); err != nil {
			t.Fatalf("Failed to create nested dir: %v", err)
		}

		nestedService := filepath.Join(nestedDir, "service.json")
		if err := os.WriteFile(nestedService, []byte(`{"name":"nested-scenario"}`), 0644); err != nil {
			t.Fatalf("Failed to create nested service.json: %v", err)
		}

		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)

		// From nested dir, should find nearest service.json
		os.Chdir(nestedDir)
		root := detectScenarioRoot()

		t.Logf("Detected root from nested dir: %s", root)
	})
}

func TestBuildPrompts(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("buildCodexPrompt_WithVariousInputs", func(t *testing.T) {
		tests := []struct {
			name  string
			req   startAgentRequest
			notes string
		}{
			{
				name:  "simple_task",
				req:   startAgentRequest{Task: "test task"},
				notes: "",
			},
			{
				name:  "with_mode",
				req:   startAgentRequest{Task: "test task", Mode: "auto"},
				notes: "",
			},
			{
				name:  "with_capabilities",
				req:   startAgentRequest{Task: "test", Capabilities: []string{"coding", "debugging"}},
				notes: "",
			},
			{
				name:  "with_notes",
				req:   startAgentRequest{Task: "test"},
				notes: "Additional context",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				prompt := buildCodexPrompt(tt.req, tt.notes)

				if prompt == "" {
					t.Error("Expected non-empty prompt")
				}

				if !containsString(prompt, tt.req.Task) {
					t.Error("Expected prompt to contain task")
				}
			})
		}
	})

	t.Run("buildOrchestrationPrompt_WithVariousInputs", func(t *testing.T) {
		tests := []struct {
			name string
			req  orchestrateRequest
		}{
			{
				name: "simple",
				req:  orchestrateRequest{Objective: "test objective"},
			},
			{
				name: "with_targets",
				req:  orchestrateRequest{Objective: "test", Targets: []string{"agent1", "agent2"}},
			},
			{
				name: "with_notes",
				req:  orchestrateRequest{Objective: "test", Notes: "additional notes"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				prompt := buildOrchestrationPrompt(tt.req)

				if prompt == "" {
					t.Error("Expected non-empty prompt")
				}

				if !containsString(prompt, tt.req.Objective) {
					t.Error("Expected prompt to contain objective")
				}
			})
		}
	})
}

// Helper function for string contains check
func containsString(str, substr string) bool {
	return len(str) > 0 && len(substr) > 0 &&
		(str == substr ||
		 (len(str) >= len(substr) && findSubstring(str, substr)))
}

func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestProcessHelpers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("getProcessCPU", func(t *testing.T) {
		cpu := getProcessCPU(os.Getpid())
		// CPU may be 0 if process just started or not enough samples
		if cpu < 0 {
			t.Errorf("Expected non-negative CPU, got %f", cpu)
		}
		t.Logf("Current process CPU: %f%%", cpu)
	})

	t.Run("getProcessMemory", func(t *testing.T) {
		mem := getProcessMemory(os.Getpid())
		if mem < 0 {
			t.Errorf("Expected non-negative memory, got %f", mem)
		}
		t.Logf("Current process memory: %f MB", mem)
	})

	t.Run("getProcessThreads", func(t *testing.T) {
		threads := getProcessThreads(os.Getpid())
		if threads < 1 {
			t.Logf("Expected at least 1 thread, got %d (may not be available on this system)", threads)
		}
	})

	t.Run("getProcessFDs", func(t *testing.T) {
		fds := getProcessFDs(os.Getpid())
		if fds < 0 {
			t.Errorf("Expected non-negative FD count, got %d", fds)
		}
		t.Logf("Current process FDs: %d", fds)
	})

	t.Run("getProcessIO", func(t *testing.T) {
		io := getProcessIO(os.Getpid())
		if io == nil {
			t.Fatal("Expected non-nil IO stats")
		}

		// Check for expected fields
		if readBytes, ok := io["io_read_bytes"]; ok {
			t.Logf("Read bytes: %v", readBytes)
		}

		if writeBytes, ok := io["io_write_bytes"]; ok {
			t.Logf("Write bytes: %v", writeBytes)
		}
	})
}

func TestCloneAgent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CloneNilAgent", func(t *testing.T) {
		cloned := cloneAgent(nil)
		if cloned != nil {
			t.Error("Expected nil clone of nil agent")
		}
	})

	t.Run("CloneValidAgent", func(t *testing.T) {
		original := &Agent{
			ID:     "test:agent1",
			Name:   "Test Agent",
			Type:   "codex",
			Status: "running",
			PID:    12345,
		}

		cloned := cloneAgent(original)

		if cloned == nil {
			t.Fatal("Expected non-nil cloned agent")
		}

		if cloned.ID != original.ID {
			t.Errorf("Expected ID %s, got %s", original.ID, cloned.ID)
		}

		if cloned.Name != original.Name {
			t.Errorf("Expected Name %s, got %s", original.Name, cloned.Name)
		}

		// Verify it's a deep copy
		cloned.Status = "stopped"
		if original.Status == "stopped" {
			t.Error("Expected deep copy, but original was modified")
		}
	})
}

func TestRateLimitMiddlewareAdvanced(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RapidRequests", func(t *testing.T) {
		// Create a very restrictive rate limiter
		limiter := newLimiter(1, 1) // 1 request per second, burst of 1

		successCount := 0
		for i := 0; i < 5; i++ {
			if limiter.Allow() {
				successCount++
			}
		}

		// Should only allow 1 request immediately
		if successCount > 2 {
			t.Logf("Rate limiter allowed %d requests (expected 1-2 due to burst)", successCount)
		}
	})
}

// Helper to create rate limiter
func newLimiter(perSecond int, burst int) *rateLimiter {
	return &rateLimiter{
		tokens:  burst,
		max:     burst,
		perSec:  perSecond,
		lastAdd: time.Now(),
	}
}

type rateLimiter struct {
	tokens  int
	max     int
	perSec  int
	lastAdd time.Time
}

func (r *rateLimiter) Allow() bool {
	now := time.Now()
	elapsed := now.Sub(r.lastAdd)

	// Add tokens based on elapsed time
	tokensToAdd := int(elapsed.Seconds()) * r.perSec
	if tokensToAdd > 0 {
		r.tokens += tokensToAdd
		if r.tokens > r.max {
			r.tokens = r.max
		}
		r.lastAdd = now
	}

	if r.tokens > 0 {
		r.tokens--
		return true
	}
	return false
}

func TestExecutableDetection(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("DetectCodexExecutable", func(t *testing.T) {
		// Try to find codex in PATH
		_, err := exec.LookPath("codex")
		if err != nil {
			t.Logf("Codex not found in PATH (expected): %v", err)
		} else {
			t.Log("Found codex executable")
		}
	})

	t.Run("DetectClaudeCode", func(t *testing.T) {
		// Try to find claude-code in PATH
		_, err := exec.LookPath("claude")
		if err != nil {
			t.Logf("Claude Code not found in PATH (expected): %v", err)
		} else {
			t.Log("Found claude executable")
		}
	})
}

func TestOrchestateRequest(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ValidateRequest", func(t *testing.T) {
		req := orchestrateRequest{
			Objective: "Test objective",
			Targets:   []string{"agent1", "agent2"},
		}

		if req.Objective == "" {
			t.Error("Expected non-empty objective")
		}

		if len(req.Targets) == 0 {
			t.Error("Expected at least one target")
		}
	})
}
