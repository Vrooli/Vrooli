package strategies

import (
	"context"
	"errors"
	"testing"

	"vrooli-autoheal/internal/checks"
)

// mockExecutor implements checks.CommandExecutor for testing.
type mockExecutor struct {
	outputResult        []byte
	outputErr           error
	combinedOutputResult []byte
	combinedOutputErr   error
	runErr              error
	lastCommand         string
	lastArgs            []string
}

func (m *mockExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	m.lastCommand = name
	m.lastArgs = args
	return m.outputResult, m.outputErr
}

func (m *mockExecutor) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	m.lastCommand = name
	m.lastArgs = args
	return m.combinedOutputResult, m.combinedOutputErr
}

func (m *mockExecutor) Run(ctx context.Context, name string, args ...string) error {
	m.lastCommand = name
	m.lastArgs = args
	return m.runErr
}

func TestNewSystemdStrategy(t *testing.T) {
	t.Run("with nil executor uses default", func(t *testing.T) {
		strategy := NewSystemdStrategy("docker", nil)
		if strategy.executor == nil {
			t.Error("expected executor to be set to default")
		}
		if strategy.serviceName != "docker" {
			t.Errorf("expected service name 'docker', got %s", strategy.serviceName)
		}
	})

	t.Run("with custom executor", func(t *testing.T) {
		exec := &mockExecutor{}
		strategy := NewSystemdStrategy("docker", exec)
		if strategy.executor != exec {
			t.Error("expected custom executor to be used")
		}
	})
}

func TestSystemdStrategy_Restart(t *testing.T) {
	t.Run("successful restart", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte(""),
		}
		strategy := NewSystemdStrategy("docker", exec)

		result := strategy.Restart(context.Background(), "infra-docker")

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if exec.lastCommand != "sudo" {
			t.Errorf("expected command 'sudo', got %s", exec.lastCommand)
		}
		if len(exec.lastArgs) < 3 || exec.lastArgs[0] != "systemctl" || exec.lastArgs[1] != "restart" {
			t.Errorf("expected 'systemctl restart', got %v", exec.lastArgs)
		}
		if result.ActionID != "restart" {
			t.Errorf("expected action ID 'restart', got %s", result.ActionID)
		}
		if result.CheckID != "infra-docker" {
			t.Errorf("expected check ID 'infra-docker', got %s", result.CheckID)
		}
	})

	t.Run("restart fails", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputErr: errors.New("permission denied"),
		}
		strategy := NewSystemdStrategy("docker", exec)

		result := strategy.Restart(context.Background(), "infra-docker")

		if result.Success {
			t.Error("expected failure")
		}
		if result.Error != "permission denied" {
			t.Errorf("expected error 'permission denied', got %s", result.Error)
		}
	})
}

func TestSystemdStrategy_Start(t *testing.T) {
	t.Run("successful start", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte(""),
		}
		strategy := NewSystemdStrategy("docker", exec)

		result := strategy.Start(context.Background(), "infra-docker")

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if len(exec.lastArgs) < 3 || exec.lastArgs[1] != "start" {
			t.Errorf("expected 'systemctl start', got %v", exec.lastArgs)
		}
		if result.ActionID != "start" {
			t.Errorf("expected action ID 'start', got %s", result.ActionID)
		}
	})

	t.Run("start fails", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputErr: errors.New("service not found"),
		}
		strategy := NewSystemdStrategy("nonexistent", exec)

		result := strategy.Start(context.Background(), "test-check")

		if result.Success {
			t.Error("expected failure")
		}
	})
}

func TestSystemdStrategy_Stop(t *testing.T) {
	t.Run("successful stop", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte(""),
		}
		strategy := NewSystemdStrategy("docker", exec)

		result := strategy.Stop(context.Background(), "infra-docker")

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if len(exec.lastArgs) < 3 || exec.lastArgs[1] != "stop" {
			t.Errorf("expected 'systemctl stop', got %v", exec.lastArgs)
		}
	})
}

func TestSystemdStrategy_Status(t *testing.T) {
	t.Run("active service", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte("active (running)"),
		}
		strategy := NewSystemdStrategy("docker", exec)

		result := strategy.Status(context.Background(), "infra-docker")

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if result.Output != "active (running)" {
			t.Errorf("expected output 'active (running)', got %s", result.Output)
		}
	})

	t.Run("inactive service still succeeds", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte("inactive (dead)"),
			combinedOutputErr:    errors.New("exit status 3"), // systemctl returns non-zero for inactive
		}
		strategy := NewSystemdStrategy("docker", exec)

		result := strategy.Status(context.Background(), "infra-docker")

		// Status should still succeed even if service is inactive
		if !result.Success {
			t.Errorf("expected success for status check, got error: %s", result.Error)
		}
	})
}

func TestSystemdStrategy_Logs(t *testing.T) {
	t.Run("successful logs retrieval", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte("Jan 01 12:00:00 server docker[1234]: Started"),
		}
		strategy := NewSystemdStrategy("docker", exec)

		result := strategy.Logs(context.Background(), "infra-docker", 50)

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if exec.lastCommand != "journalctl" {
			t.Errorf("expected command 'journalctl', got %s", exec.lastCommand)
		}
	})

	t.Run("uses default line count when zero", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputResult: []byte("logs"),
		}
		strategy := NewSystemdStrategy("docker", exec)

		strategy.Logs(context.Background(), "infra-docker", 0)

		// Should use default of 100
		found := false
		for i, arg := range exec.lastArgs {
			if arg == "-n" && i+1 < len(exec.lastArgs) && exec.lastArgs[i+1] == "100" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected -n 100, got args: %v", exec.lastArgs)
		}
	})

	t.Run("logs retrieval fails", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutputErr: errors.New("no journal files found"),
		}
		strategy := NewSystemdStrategy("docker", exec)

		result := strategy.Logs(context.Background(), "infra-docker", 100)

		if result.Success {
			t.Error("expected failure")
		}
	})
}

func TestSystemdStrategy_IsActive(t *testing.T) {
	t.Run("active service", func(t *testing.T) {
		exec := &mockExecutor{
			runErr: nil,
		}
		strategy := NewSystemdStrategy("docker", exec)

		if !strategy.IsActive(context.Background()) {
			t.Error("expected IsActive to return true")
		}
		if exec.lastCommand != "systemctl" {
			t.Errorf("expected command 'systemctl', got %s", exec.lastCommand)
		}
	})

	t.Run("inactive service", func(t *testing.T) {
		exec := &mockExecutor{
			runErr: errors.New("exit status 3"),
		}
		strategy := NewSystemdStrategy("docker", exec)

		if strategy.IsActive(context.Background()) {
			t.Error("expected IsActive to return false")
		}
	})
}

// Test that strategy uses correct service name
func TestSystemdStrategy_ServiceName(t *testing.T) {
	testCases := []struct {
		serviceName string
		actionFn    func(*SystemdStrategy, context.Context, string) checks.ActionResult
		expected    string
	}{
		{"docker", (*SystemdStrategy).Restart, "docker"},
		{"systemd-resolved", (*SystemdStrategy).Start, "systemd-resolved"},
		{"cloudflared", (*SystemdStrategy).Stop, "cloudflared"},
	}

	for _, tc := range testCases {
		t.Run(tc.serviceName, func(t *testing.T) {
			exec := &mockExecutor{
				combinedOutputResult: []byte(""),
			}
			strategy := NewSystemdStrategy(tc.serviceName, exec)

			tc.actionFn(strategy, context.Background(), "test-check")

			found := false
			for _, arg := range exec.lastArgs {
				if arg == tc.expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected service name '%s' in args, got %v", tc.expected, exec.lastArgs)
			}
		})
	}
}
