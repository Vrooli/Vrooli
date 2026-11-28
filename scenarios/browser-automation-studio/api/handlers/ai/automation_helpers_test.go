package ai

import (
	"context"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
)

// MockAutomationRunner is a test double for AutomationRunner.
type MockAutomationRunner struct {
	Outcomes []autocontracts.StepOutcome
	Events   []autocontracts.EventEnvelope
	Err      error
	RunCalls []MockRunCall
}

// MockRunCall records a call to Run.
type MockRunCall struct {
	ViewportWidth     int
	ViewportHeight    int
	DeviceScaleFactor float64
	Instructions      []autocontracts.CompiledInstruction
}

// NewMockAutomationRunner creates a MockAutomationRunner with default successful outcomes.
func NewMockAutomationRunner() *MockAutomationRunner {
	return &MockAutomationRunner{
		Outcomes: []autocontracts.StepOutcome{
			{
				Success:   true,
				StepIndex: 0,
				NodeID:    "mock-step",
				StepType:  "navigate",
			},
		},
	}
}

// Run records the call and returns configured outcomes or error.
func (m *MockAutomationRunner) Run(ctx context.Context, viewportWidth, viewportHeight int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
	return m.run(ctx, viewportWidth, viewportHeight, 0, instructions)
}

func (m *MockAutomationRunner) run(_ context.Context, viewportWidth, viewportHeight int, deviceScaleFactor float64, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
	m.RunCalls = append(m.RunCalls, MockRunCall{
		ViewportWidth:     viewportWidth,
		ViewportHeight:    viewportHeight,
		DeviceScaleFactor: deviceScaleFactor,
		Instructions:      instructions,
	})

	if m.Err != nil {
		return nil, nil, m.Err
	}
	return m.Outcomes, m.Events, nil
}

// RunWithDeviceScale lets screenshot tests assert the same call path as the
// production runner while preserving the compact mock call record.
func (m *MockAutomationRunner) RunWithDeviceScale(ctx context.Context, viewportWidth, viewportHeight int, deviceScaleFactor float64, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
	return m.run(ctx, viewportWidth, viewportHeight, deviceScaleFactor, instructions)
}

// Reset clears recorded calls for reuse between tests.
func (m *MockAutomationRunner) Reset() {
	m.RunCalls = nil
}

// Compile-time interface enforcement
var _ AutomationRunner = (*MockAutomationRunner)(nil)
