package service

import (
	"context"
	"sync"

	"tunnel-manager/domain"
)

// mockRouteStore implements RouteStore for testing.
type mockRouteStore struct {
	listFn    func() ([]domain.Route, error)
	getByIDFn func(int) (*domain.Route, error)
	createFn  func(subdomain, scenarioName string, localPort int, healthPath, publicURL string, enabled bool) (*domain.Route, error)
	updateFn  func(id int, subdomain, scenarioName string, localPort int, healthPath, publicURL string, enabled bool) (*domain.Route, error)
	deleteFn  func(id int) error
}

func (m *mockRouteStore) List() ([]domain.Route, error) { return m.listFn() }

func (m *mockRouteStore) GetByID(id int) (*domain.Route, error) { return m.getByIDFn(id) }

func (m *mockRouteStore) Create(subdomain, scenarioName string, localPort int, healthPath, publicURL string, enabled bool) (*domain.Route, error) {
	return m.createFn(subdomain, scenarioName, localPort, healthPath, publicURL, enabled)
}

func (m *mockRouteStore) Update(id int, subdomain, scenarioName string, localPort int, healthPath, publicURL string, enabled bool) (*domain.Route, error) {
	return m.updateFn(id, subdomain, scenarioName, localPort, healthPath, publicURL, enabled)
}

func (m *mockRouteStore) Delete(id int) error { return m.deleteFn(id) }

// mockProbeResultWriter implements ProbeResultWriter for testing.
type mockProbeResultWriter struct {
	persistResultFn func(pr domain.ProbeResult) error
}

func (m *mockProbeResultWriter) PersistResult(pr domain.ProbeResult) error {
	return m.persistResultFn(pr)
}

// mockRecoveryEventStore implements RecoveryEventStore for testing.
type mockRecoveryEventStore struct {
	persistEventFn func(evt *domain.RecoveryEvent) error
	listEventsFn   func(limit int) ([]domain.RecoveryEvent, error)
}

func (m *mockRecoveryEventStore) PersistEvent(evt *domain.RecoveryEvent) error {
	return m.persistEventFn(evt)
}

func (m *mockRecoveryEventStore) ListEvents(limit int) ([]domain.RecoveryEvent, error) {
	return m.listEventsFn(limit)
}

// mockTunnelChecker implements TunnelChecker for testing.
type mockTunnelChecker struct {
	checkFn func(ctx context.Context) domain.TunnelStatus
}

func (m *mockTunnelChecker) Check(ctx context.Context) domain.TunnelStatus { return m.checkFn(ctx) }

// mockRouteLister implements RouteLister for testing.
type mockRouteLister struct {
	listFn func() ([]domain.Route, error)
}

func (m *mockRouteLister) List() ([]domain.Route, error) { return m.listFn() }

// mockProbeRunner implements ProbeRunner for testing.
type mockProbeRunner struct {
	runAllFn func(ctx context.Context) ([]domain.ProbeResult, error)
	mu       sync.Mutex
	calls    int
}

func (m *mockProbeRunner) RunAll(ctx context.Context) ([]domain.ProbeResult, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	return m.runAllFn(ctx)
}

func (m *mockProbeRunner) runAllCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}
