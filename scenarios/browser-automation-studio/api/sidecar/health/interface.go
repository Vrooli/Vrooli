package health

import "context"

// Checker is an interface for checking the health of the sidecar.
type Checker interface {
	// Check checks the health of the sidecar.
	// Returns the health response if successful, error otherwise.
	Check(ctx context.Context) (*DriverHealthResponse, error)
}

// Monitor is an interface for monitoring sidecar health.
type Monitor interface {
	// Start begins health monitoring.
	Start(ctx context.Context) error

	// Stop stops health monitoring.
	Stop() error

	// CurrentHealth returns the current health state.
	CurrentHealth() DriverHealth

	// Subscribe returns a channel that receives health state updates.
	// The channel is closed when the monitor is stopped.
	Subscribe() <-chan DriverHealth

	// Unsubscribe removes a subscriber channel.
	Unsubscribe(ch <-chan DriverHealth)
}

// compile-time check that HealthMonitor implements Monitor
var _ Monitor = (*HealthMonitor)(nil)

// MockChecker is a Checker implementation for testing.
type MockChecker struct {
	Response *DriverHealthResponse
	Error    error
	Calls    int
}

// Check implements Checker.
func (m *MockChecker) Check(ctx context.Context) (*DriverHealthResponse, error) {
	m.Calls++
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Response, nil
}

// compile-time check
var _ Checker = (*MockChecker)(nil)
