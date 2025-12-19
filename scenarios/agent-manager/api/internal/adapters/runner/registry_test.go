package runner_test

import (
	"context"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// [REQ:REQ-P0-006] Tests for runner adapter interface and registry

func TestDefaultRegistry_Register(t *testing.T) {
	reg := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)

	if err := reg.Register(mock); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify runner was registered
	r, err := reg.Get(domain.RunnerTypeClaudeCode)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if r.Type() != domain.RunnerTypeClaudeCode {
		t.Errorf("expected type %s, got %s", domain.RunnerTypeClaudeCode, r.Type())
	}
}

func TestDefaultRegistry_Register_Duplicate(t *testing.T) {
	reg := runner.NewRegistry()
	mock1 := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock2 := runner.NewMockRunner(domain.RunnerTypeClaudeCode)

	if err := reg.Register(mock1); err != nil {
		t.Fatalf("First register failed: %v", err)
	}

	err := reg.Register(mock2)
	if err == nil {
		t.Error("expected error when registering duplicate runner type")
	}
}

func TestDefaultRegistry_Get_NotFound(t *testing.T) {
	reg := runner.NewRegistry()

	_, err := reg.Get(domain.RunnerTypeClaudeCode)
	if err == nil {
		t.Error("expected error when getting unregistered runner")
	}
}

func TestDefaultRegistry_List(t *testing.T) {
	reg := runner.NewRegistry()

	// Register multiple runners
	reg.Register(runner.NewMockRunner(domain.RunnerTypeClaudeCode))
	reg.Register(runner.NewMockRunner(domain.RunnerTypeCodex))
	reg.Register(runner.NewMockRunner(domain.RunnerTypeOpenCode))

	runners := reg.List()
	if len(runners) != 3 {
		t.Errorf("expected 3 runners, got %d", len(runners))
	}
}

func TestDefaultRegistry_Available(t *testing.T) {
	reg := runner.NewRegistry()
	ctx := context.Background()

	// Register runners with different availability
	available := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	available.SetAvailable(true, "ready")

	unavailable := runner.NewMockRunner(domain.RunnerTypeCodex)
	unavailable.SetAvailable(false, "not configured")

	reg.Register(available)
	reg.Register(unavailable)

	// Check available runners
	avail := reg.Available(ctx)
	if len(avail) != 1 {
		t.Errorf("expected 1 available runner, got %d", len(avail))
	}

	if len(avail) > 0 && avail[0].Type() != domain.RunnerTypeClaudeCode {
		t.Errorf("expected claude-code to be available, got %s", avail[0].Type())
	}
}

func TestMockRunner_Defaults(t *testing.T) {
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)

	// Check type
	if mock.Type() != domain.RunnerTypeClaudeCode {
		t.Errorf("expected type %s, got %s", domain.RunnerTypeClaudeCode, mock.Type())
	}

	// Check default availability
	avail, msg := mock.IsAvailable(context.Background())
	if !avail {
		t.Errorf("expected mock to be available by default, got: %s", msg)
	}

	// Check capabilities
	caps := mock.Capabilities()
	if !caps.SupportsMessages {
		t.Error("expected mock to support messages by default")
	}
	if !caps.SupportsStreaming {
		t.Error("expected mock to support streaming by default")
	}
}

func TestMockRunner_Execute_Default(t *testing.T) {
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	ctx := context.Background()

	req := runner.ExecuteRequest{
		RunID:      uuid.New(),
		Prompt:     "test prompt",
		WorkingDir: "/tmp/test",
	}

	result, err := mock.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Error("expected default execution to succeed")
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestMockRunner_Execute_Custom(t *testing.T) {
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	ctx := context.Background()

	// Set custom execute behavior
	customResult := &runner.ExecuteResult{
		Success:  false,
		ExitCode: 1,
	}
	mock.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return customResult, nil
	}

	result, err := mock.Execute(ctx, runner.ExecuteRequest{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Error("expected custom execution to fail")
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
}

func TestMockRunner_Stop(t *testing.T) {
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	ctx := context.Background()
	runID := uuid.New()

	// Default stop should succeed
	if err := mock.Stop(ctx, runID); err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// Custom stop behavior
	stopCalled := false
	mock.StopFunc = func(ctx context.Context, id uuid.UUID) error {
		stopCalled = true
		return nil
	}

	mock.Stop(ctx, runID)
	if !stopCalled {
		t.Error("custom StopFunc was not called")
	}
}

func TestMockRunner_SetAvailable(t *testing.T) {
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	ctx := context.Background()

	// Set unavailable
	mock.SetAvailable(false, "maintenance")

	avail, msg := mock.IsAvailable(ctx)
	if avail {
		t.Error("expected mock to be unavailable")
	}
	if msg != "maintenance" {
		t.Errorf("expected message 'maintenance', got '%s'", msg)
	}
}

func TestMockRunner_SetCapabilities(t *testing.T) {
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)

	caps := runner.Capabilities{
		SupportsMessages:     false,
		SupportsCostTracking: true,
		MaxTurns:             50,
	}
	mock.SetCapabilities(caps)

	result := mock.Capabilities()
	if result.SupportsMessages {
		t.Error("expected messages to be disabled")
	}
	if !result.SupportsCostTracking {
		t.Error("expected cost tracking to be enabled")
	}
	if result.MaxTurns != 50 {
		t.Errorf("expected max turns 50, got %d", result.MaxTurns)
	}
}

func TestStubRunner(t *testing.T) {
	stub := runner.NewStubRunner(domain.RunnerTypeCodex, "not installed")
	ctx := context.Background()

	// Check type
	if stub.Type() != domain.RunnerTypeCodex {
		t.Errorf("expected type %s, got %s", domain.RunnerTypeCodex, stub.Type())
	}

	// Check unavailability
	avail, msg := stub.IsAvailable(ctx)
	if avail {
		t.Error("expected stub to be unavailable")
	}
	if msg != "not installed" {
		t.Errorf("expected message 'not installed', got '%s'", msg)
	}

	// Check empty capabilities
	caps := stub.Capabilities()
	if caps.SupportsMessages || caps.SupportsStreaming {
		t.Error("expected stub to have no capabilities")
	}
}

func TestStubRunner_Execute_Fails(t *testing.T) {
	stub := runner.NewStubRunner(domain.RunnerTypeCodex, "not available")
	ctx := context.Background()

	_, err := stub.Execute(ctx, runner.ExecuteRequest{})
	if err == nil {
		t.Error("expected stub execute to fail")
	}
}

func TestStubRunner_Stop_Fails(t *testing.T) {
	stub := runner.NewStubRunner(domain.RunnerTypeCodex, "not available")
	ctx := context.Background()

	err := stub.Stop(ctx, uuid.New())
	if err == nil {
		t.Error("expected stub stop to fail")
	}
}
