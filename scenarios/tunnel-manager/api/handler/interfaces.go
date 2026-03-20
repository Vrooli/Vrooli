package handler

import (
	"context"
	"time"

	"tunnel-manager/domain"
)

// MetricsQuerier queries stored metrics history.
type MetricsQuerier interface {
	Query(from, to time.Time) ([]domain.MetricsRecord, error)
	Latest() (*domain.MetricsRecord, error)
}

// ProbeHistoryReader queries stored probe results.
type ProbeHistoryReader interface {
	QueryRecent(limit int) ([]domain.StoredProbeResult, error)
}

// RouteLister lists routes.
type RouteLister interface {
	List() ([]domain.Route, error)
}

// RouteManager provides full route CRUD operations.
type RouteManager interface {
	List() ([]domain.Route, error)
	GetByID(id int) (*domain.Route, error)
	Create(in domain.RouteInput) (*domain.Route, error)
	Update(id int, in domain.RouteInput) (*domain.Route, error)
	Delete(id int) error
}

// TunnelChecker checks tunnel health.
type TunnelChecker interface {
	Check(ctx context.Context) domain.TunnelStatus
}

// ProbeRunner executes probes.
type ProbeRunner interface {
	RunAll(ctx context.Context) ([]domain.ProbeResult, error)
}

// PortAuditor audits port compliance.
type PortAuditor interface {
	Audit() ([]domain.PortAuditResult, error)
}

// RecoveryManager manages recovery state.
type RecoveryManager interface {
	State() domain.RecoveryState
	TriggerManual(ctx context.Context, force bool) (*domain.RecoveryEvent, error)
	ListEvents(limit int) ([]domain.RecoveryEvent, error)
	ResetCircuit()
}
