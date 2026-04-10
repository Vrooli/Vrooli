package service

import (
	"context"
	"time"

	"tunnel-manager/domain"
)

// RouteStore is the persistence interface for routes.
type RouteStore interface {
	List() ([]domain.Route, error)
	GetByID(id int) (*domain.Route, error)
	Create(subdomain, scenarioName string, localPort int, healthPath, publicURL string, enabled bool) (*domain.Route, error)
	Update(id int, subdomain, scenarioName string, localPort int, healthPath, publicURL string, enabled bool) (*domain.Route, error)
	Delete(id int) error
}

// ProbeResultWriter persists probe results.
type ProbeResultWriter interface {
	PersistResult(pr domain.ProbeResult) error
}

// ProbeResultReader queries stored probe results.
type ProbeResultReader interface {
	QueryRecent(limit int) ([]domain.StoredProbeResult, error)
}

// RecoveryEventStore persists and queries recovery events.
type RecoveryEventStore interface {
	PersistEvent(evt *domain.RecoveryEvent) error
	ListEvents(limit int) ([]domain.RecoveryEvent, error)
}

// MetricsReader queries stored metrics.
type MetricsReader interface {
	Query(from, to time.Time) ([]domain.MetricsRecord, error)
	Latest() (*domain.MetricsRecord, error)
}

// MetricsWriter stores scraped metrics.
type MetricsWriter interface {
	Store(m *domain.TunnelMetrics) error
}

// TunnelChecker checks tunnel health.
type TunnelChecker interface {
	Check(ctx context.Context) domain.TunnelStatus
}

// RouteLister is a narrow interface for services that only need to list routes.
type RouteLister interface {
	List() ([]domain.Route, error)
}

// ProbeRunner executes a full probe cycle.
type ProbeRunner interface {
	RunAll(ctx context.Context) ([]domain.ProbeResult, error)
}
