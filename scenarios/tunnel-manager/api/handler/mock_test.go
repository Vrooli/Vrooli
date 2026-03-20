package handler

import (
	"context"
	"time"

	"tunnel-manager/domain"
)

// mockRecoveryManager implements RecoveryManager for testing.
type mockRecoveryManager struct {
	stateFn        func() domain.RecoveryState
	triggerFn      func(ctx context.Context, force bool) (*domain.RecoveryEvent, error)
	listEventsFn   func(limit int) ([]domain.RecoveryEvent, error)
	resetCircuitFn func()
}

func (m *mockRecoveryManager) State() domain.RecoveryState { return m.stateFn() }

func (m *mockRecoveryManager) TriggerManual(ctx context.Context, force bool) (*domain.RecoveryEvent, error) {
	return m.triggerFn(ctx, force)
}

func (m *mockRecoveryManager) ListEvents(limit int) ([]domain.RecoveryEvent, error) {
	return m.listEventsFn(limit)
}

func (m *mockRecoveryManager) ResetCircuit() { m.resetCircuitFn() }

// mockHandlerTunnelChecker implements TunnelChecker for handler tests.
type mockHandlerTunnelChecker struct {
	checkFn func(ctx context.Context) domain.TunnelStatus
}

func (m *mockHandlerTunnelChecker) Check(ctx context.Context) domain.TunnelStatus {
	return m.checkFn(ctx)
}

// mockHandlerRouteLister implements RouteLister for handler tests.
type mockHandlerRouteLister struct {
	listFn func() ([]domain.Route, error)
}

func (m *mockHandlerRouteLister) List() ([]domain.Route, error) { return m.listFn() }

// mockHandlerProbeRunner implements ProbeRunner for handler tests.
type mockHandlerProbeRunner struct {
	runAllFn func(ctx context.Context) ([]domain.ProbeResult, error)
}

func (m *mockHandlerProbeRunner) RunAll(ctx context.Context) ([]domain.ProbeResult, error) {
	return m.runAllFn(ctx)
}

// mockMetricsQuerier implements MetricsQuerier for testing.
type mockMetricsQuerier struct {
	queryFn  func(from, to time.Time) ([]domain.MetricsRecord, error)
	latestFn func() (*domain.MetricsRecord, error)
}

func (m *mockMetricsQuerier) Query(from, to time.Time) ([]domain.MetricsRecord, error) {
	return m.queryFn(from, to)
}

func (m *mockMetricsQuerier) Latest() (*domain.MetricsRecord, error) {
	return m.latestFn()
}

// mockProbeHistoryReader implements ProbeHistoryReader for testing.
type mockProbeHistoryReader struct {
	queryRecentFn func(limit int) ([]domain.StoredProbeResult, error)
}

func (m *mockProbeHistoryReader) QueryRecent(limit int) ([]domain.StoredProbeResult, error) {
	return m.queryRecentFn(limit)
}
