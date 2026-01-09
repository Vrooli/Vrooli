package strategies

import (
	"context"
	"errors"
	"testing"
)

func TestNewVrooliStrategy(t *testing.T) {
	t.Run("with nil executor uses default", func(t *testing.T) {
		strategy := NewVrooliStrategy(VrooliResource, "postgres", nil)
		if strategy.executor == nil {
			t.Error("expected executor to be set to default")
		}
		if strategy.entityType != VrooliResource {
			t.Errorf("expected entity type 'resource', got %s", strategy.entityType)
		}
		if strategy.entityName != "postgres" {
			t.Errorf("expected entity name 'postgres', got %s", strategy.entityName)
		}
	})

	t.Run("with custom executor", func(t *testing.T) {
		exec := &mockExecutor{}
		strategy := NewVrooliStrategy(VrooliScenario, "landing-manager", exec)
		if strategy.executor != exec {
			t.Error("expected custom executor to be used")
		}
		if strategy.entityType != VrooliScenario {
			t.Errorf("expected entity type 'scenario', got %s", strategy.entityType)
		}
	})
}

func TestVrooliStrategy_Start(t *testing.T) {
	t.Run("successful start resource", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte("postgres started"),
		}
		strategy := NewVrooliStrategy(VrooliResource, "postgres", exec)

		result := strategy.Start(context.Background(), "resource-postgres")

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if exec.lastCommand != "vrooli" {
			t.Errorf("expected command 'vrooli', got %s", exec.lastCommand)
		}
		if len(exec.lastArgs) < 3 {
			t.Fatal("expected at least 3 args")
		}
		if exec.lastArgs[0] != "resource" || exec.lastArgs[1] != "start" || exec.lastArgs[2] != "postgres" {
			t.Errorf("expected 'resource start postgres', got %v", exec.lastArgs)
		}
		if result.ActionID != "start" {
			t.Errorf("expected action ID 'start', got %s", result.ActionID)
		}
	})

	t.Run("successful start scenario", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte("scenario started"),
		}
		strategy := NewVrooliStrategy(VrooliScenario, "landing-manager", exec)

		result := strategy.Start(context.Background(), "scenario-landing-manager")

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if exec.lastArgs[0] != "scenario" {
			t.Errorf("expected first arg 'scenario', got %s", exec.lastArgs[0])
		}
	})

	t.Run("start fails", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputErr: errors.New("resource not found"),
		}
		strategy := NewVrooliStrategy(VrooliResource, "unknown", exec)

		result := strategy.Start(context.Background(), "resource-unknown")

		if result.Success {
			t.Error("expected failure")
		}
		if result.Error != "resource not found" {
			t.Errorf("expected error 'resource not found', got %s", result.Error)
		}
	})
}

func TestVrooliStrategy_Stop(t *testing.T) {
	t.Run("successful stop", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte("stopped"),
		}
		strategy := NewVrooliStrategy(VrooliResource, "postgres", exec)

		result := strategy.Stop(context.Background(), "resource-postgres")

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if exec.lastArgs[1] != "stop" {
			t.Errorf("expected 'stop', got %s", exec.lastArgs[1])
		}
	})
}

func TestVrooliStrategy_Restart(t *testing.T) {
	t.Run("successful restart", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte("restarted"),
		}
		strategy := NewVrooliStrategy(VrooliScenario, "landing-manager", exec)

		result := strategy.Restart(context.Background(), "scenario-landing-manager")

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if exec.lastArgs[1] != "restart" {
			t.Errorf("expected 'restart', got %s", exec.lastArgs[1])
		}
	})
}

func TestVrooliStrategy_Status(t *testing.T) {
	t.Run("successful status", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte("running"),
		}
		strategy := NewVrooliStrategy(VrooliResource, "postgres", exec)

		result := strategy.Status(context.Background(), "resource-postgres")

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if result.Output != "running" {
			t.Errorf("expected output 'running', got %s", result.Output)
		}
	})

	t.Run("status fails", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputErr: errors.New("not found"),
		}
		strategy := NewVrooliStrategy(VrooliResource, "unknown", exec)

		result := strategy.Status(context.Background(), "resource-unknown")

		if result.Success {
			t.Error("expected failure")
		}
	})
}

func TestVrooliStrategy_Logs(t *testing.T) {
	t.Run("successful logs retrieval", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte("log line 1\nlog line 2"),
		}
		strategy := NewVrooliStrategy(VrooliScenario, "landing-manager", exec)

		result := strategy.Logs(context.Background(), "scenario-landing-manager", 50)

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if exec.lastArgs[1] != "logs" {
			t.Errorf("expected 'logs', got %s", exec.lastArgs[1])
		}
	})

	t.Run("uses default line count when zero", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte("logs"),
		}
		strategy := NewVrooliStrategy(VrooliResource, "postgres", exec)

		strategy.Logs(context.Background(), "resource-postgres", 0)

		// Should include --tail 100
		found := false
		for i, arg := range exec.lastArgs {
			if arg == "--tail" && i+1 < len(exec.lastArgs) && exec.lastArgs[i+1] == "100" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected '--tail 100', got args: %v", exec.lastArgs)
		}
	})
}

func TestVrooliStrategy_GetPorts(t *testing.T) {
	t.Run("successful get ports", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte("8080\n8081"),
		}
		strategy := NewVrooliStrategy(VrooliScenario, "landing-manager", exec)

		result := strategy.GetPorts(context.Background(), "scenario-landing-manager")

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if exec.lastArgs[1] != "port" {
			t.Errorf("expected 'port', got %s", exec.lastArgs[1])
		}
	})
}

func TestVrooliStrategy_CleanupPorts(t *testing.T) {
	t.Run("no ports to cleanup", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte("no ports configured"),
		}
		strategy := NewVrooliStrategy(VrooliScenario, "test", exec)

		result := strategy.CleanupPorts(context.Background(), "scenario-test")

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if result.Message != "No ports found to cleanup for scenario test" {
			t.Errorf("unexpected message: %s", result.Message)
		}
	})

	t.Run("fails to get ports", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputErr: errors.New("failed"),
		}
		strategy := NewVrooliStrategy(VrooliScenario, "test", exec)

		result := strategy.CleanupPorts(context.Background(), "scenario-test")

		if result.Success {
			t.Error("expected failure")
		}
	})

	// Note: Testing port cleanup with actual process killing requires a more
	// sophisticated mock that tracks multiple calls. The basic functionality
	// is tested through the port extraction and failure path tests.
}

func TestVrooliStrategy_Diagnose(t *testing.T) {
	t.Run("gathers diagnostic info", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte("status output"),
		}
		strategy := NewVrooliStrategy(VrooliScenario, "landing-manager", exec)

		result := strategy.Diagnose(context.Background(), "scenario-landing-manager")

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if result.ActionID != "diagnose" {
			t.Errorf("expected action ID 'diagnose', got %s", result.ActionID)
		}
		// Should include status, ports, and logs sections
		if len(result.Output) == 0 {
			t.Error("expected non-empty diagnostic output")
		}
	})
}

func TestVrooliStrategy_CleanRestart(t *testing.T) {
	t.Run("performs clean restart sequence", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte("ok"),
		}

		strategy := NewVrooliStrategy(VrooliScenario, "test", exec)

		result := strategy.CleanRestart(context.Background(), "scenario-test")

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if result.ActionID != "restart-clean" {
			t.Errorf("expected action ID 'restart-clean', got %s", result.ActionID)
		}
		// Verify output contains the expected sections
		if !containsSubstr(result.Output, "Stopping") {
			t.Error("expected output to contain 'Stopping'")
		}
		if !containsSubstr(result.Output, "Starting") {
			t.Error("expected output to contain 'Starting'")
		}
	})
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestVrooliStrategy_IsRunning(t *testing.T) {
	t.Run("running status", func(t *testing.T) {
		testCases := []struct {
			output   string
			expected bool
		}{
			{"running", true},
			{"Running", true},
			{"healthy", true},
			{"HEALTHY", true},
			{"started", true},
			{"stopped", false},
			{"not running", false},
			{"error", false},
			{"", false},
		}

		for _, tc := range testCases {
			t.Run(tc.output, func(t *testing.T) {
				exec := &mockExecutor{
					combinedOutputResult: []byte(tc.output),
				}
				strategy := NewVrooliStrategy(VrooliResource, "postgres", exec)

				got := strategy.IsRunning(context.Background())
				if got != tc.expected {
					t.Errorf("IsRunning(%q) = %v, want %v", tc.output, got, tc.expected)
				}
			})
		}
	})

	t.Run("command fails", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputErr: errors.New("command failed"),
		}
		strategy := NewVrooliStrategy(VrooliResource, "postgres", exec)

		if strategy.IsRunning(context.Background()) {
			t.Error("expected IsRunning to return false when command fails")
		}
	})
}

func TestExtractPorts(t *testing.T) {
	testCases := []struct {
		name     string
		output   string
		expected []int
	}{
		{
			name:     "single port",
			output:   "8080",
			expected: []int{8080},
		},
		{
			name:     "multiple ports",
			output:   "8080\n8081\n9000",
			expected: []int{8080, 8081, 9000},
		},
		{
			name:     "port with tcp suffix",
			output:   "8080/tcp",
			expected: []int{8080},
		},
		{
			name:     "port with colon prefix",
			output:   ":8080",
			expected: []int{8080},
		},
		{
			name:     "mixed format",
			output:   "port: 8080\nPORT=9000\n:8081/tcp",
			expected: []int{8080, 9000, 8081},
		},
		{
			name:     "no ports",
			output:   "no ports configured",
			expected: nil,
		},
		{
			name:     "filters reserved ports",
			output:   "80\n443\n8080",
			expected: []int{8080}, // 80 and 443 are filtered (< 1024)
		},
		{
			name:     "deduplicates ports",
			output:   "8080\n8080\n8080",
			expected: []int{8080},
		},
		{
			name:     "empty output",
			output:   "",
			expected: nil,
		},
		{
			name:     "invalid port numbers",
			output:   "abc\n-1\n99999\n8080",
			expected: []int{8080},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractPorts(tc.output)

			if len(got) != len(tc.expected) {
				t.Errorf("ExtractPorts(%q) = %v, want %v", tc.output, got, tc.expected)
				return
			}

			for i, port := range tc.expected {
				if got[i] != port {
					t.Errorf("ExtractPorts(%q)[%d] = %d, want %d", tc.output, i, got[i], port)
				}
			}
		})
	}
}

// Test entity type constants
func TestVrooliEntityType(t *testing.T) {
	if VrooliResource != "resource" {
		t.Errorf("expected VrooliResource to be 'resource', got %s", VrooliResource)
	}
	if VrooliScenario != "scenario" {
		t.Errorf("expected VrooliScenario to be 'scenario', got %s", VrooliScenario)
	}
}
