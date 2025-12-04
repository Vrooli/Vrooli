package orchestrator

import (
	"bytes"
	"context"
	"testing"
)

func TestWithLogger(t *testing.T) {
	var buf bytes.Buffer
	cfg := validConfig()
	orch := New(cfg, WithLogger(&buf))

	if orch.logger != &buf {
		t.Error("expected logger to be set")
	}
}

func TestWithPreflightChecker(t *testing.T) {
	cfg := validConfig()
	preflight := &mockPreflightChecker{hasUIDir: true}

	orch := New(cfg, WithPreflightChecker(preflight))

	if orch.preflight != preflight {
		t.Error("expected preflight checker to be set")
	}
}

func TestWithHealthChecker(t *testing.T) {
	cfg := validConfig()
	health := &mockHealthChecker{}

	orch := New(cfg, WithHealthChecker(health))

	if orch.healthChecker != health {
		t.Error("expected health checker to be set")
	}
	// HealthChecker also sets preflight if nil
	if orch.preflight != health {
		t.Error("expected preflight to be set to health checker")
	}
}

func TestWithHealthChecker_PreflightAlreadySet(t *testing.T) {
	cfg := validConfig()
	preflight := &mockPreflightChecker{hasUIDir: true}
	health := &mockHealthChecker{}

	orch := New(cfg,
		WithPreflightChecker(preflight),
		WithHealthChecker(health),
	)

	if orch.healthChecker != health {
		t.Error("expected health checker to be set")
	}
	// Preflight should remain as the original since it was set first
	if orch.preflight != preflight {
		t.Error("expected preflight to remain as original")
	}
}

func TestWithAutoRecovery(t *testing.T) {
	cfg := validConfig()

	orch := New(cfg, WithAutoRecovery(true))
	if !orch.autoRecoveryEnabled {
		t.Error("expected auto recovery to be enabled")
	}

	orch = New(cfg, WithAutoRecovery(false))
	if orch.autoRecoveryEnabled {
		t.Error("expected auto recovery to be disabled")
	}
}

func TestWithAutoRecoveryOptions(t *testing.T) {
	cfg := validConfig()
	opts := AutoRecoveryOptions{
		SharedMode:    true,
		MaxRetries:    3,
		ContainerName: "custom-container",
	}

	orch := New(cfg, WithAutoRecoveryOptions(opts))

	if !orch.autoRecoveryEnabled {
		t.Error("expected auto recovery to be enabled")
	}
	if orch.autoRecoveryOpts.SharedMode != true {
		t.Error("expected shared mode to be true")
	}
	if orch.autoRecoveryOpts.MaxRetries != 3 {
		t.Errorf("expected max retries 3, got %d", orch.autoRecoveryOpts.MaxRetries)
	}
	if orch.autoRecoveryOpts.ContainerName != "custom-container" {
		t.Errorf("expected container name 'custom-container', got %s", orch.autoRecoveryOpts.ContainerName)
	}
}

func TestDefaultAutoRecoveryOptions(t *testing.T) {
	opts := DefaultAutoRecoveryOptions()

	if opts.SharedMode != false {
		t.Error("expected shared mode to default to false")
	}
	if opts.MaxRetries != 1 {
		t.Errorf("expected max retries 1, got %d", opts.MaxRetries)
	}
	if opts.ContainerName != "vrooli-browserless" {
		t.Errorf("expected container name 'vrooli-browserless', got %s", opts.ContainerName)
	}
}

func TestWithBrowserClient(t *testing.T) {
	cfg := validConfig()
	browser := &mockBrowserClient{}

	orch := New(cfg, WithBrowserClient(browser))

	if orch.browser != browser {
		t.Error("expected browser client to be set")
	}
}

func TestWithArtifactWriter(t *testing.T) {
	cfg := validConfig()
	artifacts := &mockArtifactWriter{}

	orch := New(cfg, WithArtifactWriter(artifacts))

	if orch.artifacts != artifacts {
		t.Error("expected artifact writer to be set")
	}
}

func TestWithHandshakeDetector(t *testing.T) {
	cfg := validConfig()
	detector := &mockHandshakeDetector{}

	orch := New(cfg, WithHandshakeDetector(detector))

	if orch.handshake != detector {
		t.Error("expected handshake detector to be set")
	}
}

func TestWithPayloadGenerator(t *testing.T) {
	cfg := validConfig()
	gen := &mockPayloadGenerator{}

	orch := New(cfg, WithPayloadGenerator(gen))

	if orch.payloadGen != gen {
		t.Error("expected payload generator to be set")
	}
}

func TestWithScenarioStarter(t *testing.T) {
	cfg := validConfig()
	starter := &MockScenarioStarter{}

	orch := New(cfg, WithScenarioStarter(starter))

	if orch.scenarioStarter != starter {
		t.Error("expected scenario starter to be set")
	}
}

// mockHealthChecker implements HealthChecker for testing.
type mockHealthChecker struct {
	mockPreflightChecker
	healthy       bool
	healthErr     error
	diagnostics   *HealthDiagnostics
	recoveryResult *RecoveryResult
	diagnosis     *Diagnosis
}

func (m *mockHealthChecker) IsHealthy(ctx context.Context) (bool, *HealthDiagnostics, error) {
	return m.healthy, m.diagnostics, m.healthErr
}

func (m *mockHealthChecker) EnsureHealthy(ctx context.Context, opts AutoRecoveryOptions) (*RecoveryResult, error) {
	if m.recoveryResult != nil {
		return m.recoveryResult, nil
	}
	return &RecoveryResult{Attempted: false, Success: true}, nil
}

func (m *mockHealthChecker) DiagnoseBrowserlessFailure(ctx context.Context, scenarioName string) *Diagnosis {
	if m.diagnosis != nil {
		return m.diagnosis
	}
	return &Diagnosis{
		Type:           DiagnosisUnknown,
		Message:        "Unknown failure",
		Recommendation: "Check logs",
	}
}
