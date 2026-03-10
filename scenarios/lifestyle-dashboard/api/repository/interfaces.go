// Package repository defines storage interfaces for the lifestyle dashboard.
// These interfaces abstract database operations to enable:
// - Unit testing with mock implementations
// - Swapping storage backends (SQLite, PostgreSQL, in-memory)
// - Clear separation between business logic and persistence
//
// [REQ:LD-EVENT-STORAGE] Events are persisted through EventRepository.
// [REQ:LD-DOMAIN-REGISTER] Domains are managed through DomainRepository.
package repository

import (
	"context"

	"lifestyle-dashboard/domain"
)

// EventFilter specifies query parameters for listing events.
type EventFilter struct {
	Domain    string
	EventType string
	StartTime string
	EndTime   string
	Limit     int
}

// EventRepository abstracts event storage operations.
type EventRepository interface {
	// Create persists a new event and returns it with generated ID.
	Create(ctx context.Context, event *domain.Event) error

	// GetByID retrieves a single event by its ID.
	GetByID(ctx context.Context, id string) (*domain.Event, error)

	// List retrieves events matching the given filter.
	List(ctx context.Context, filter EventFilter) ([]domain.Event, error)
}

// DomainRepository abstracts domain storage operations.
type DomainRepository interface {
	// Upsert creates or updates a domain registration.
	Upsert(ctx context.Context, d *domain.Domain) error

	// GetByName retrieves a single domain by name.
	GetByName(ctx context.Context, name string) (*domain.Domain, error)

	// List retrieves all registered domains.
	List(ctx context.Context) ([]domain.Domain, error)

	// UpdateStatus updates a domain's status and last health check time.
	UpdateStatus(ctx context.Context, name, status, lastHealthAt string) error

	// Update applies partial updates to a domain.
	Update(ctx context.Context, name string, updates map[string]interface{}) error
}

// StatsRepository abstracts statistics and aggregation queries.
type StatsRepository interface {
	// GetTimeline returns event counts grouped by day and domain.
	GetTimeline(ctx context.Context, days int) ([]domain.TimelineEntry, error)

	// GetSummary returns aggregated statistics across all domains.
	GetSummary(ctx context.Context) (*domain.SummaryResponse, error)
}

// ErrNotFound is returned when a requested entity doesn't exist.
type ErrNotFound struct {
	Entity string
	ID     string
}

func (e ErrNotFound) Error() string {
	return e.Entity + " not found: " + e.ID
}

// IsNotFound returns true if the error indicates a missing entity.
func IsNotFound(err error) bool {
	_, ok := err.(ErrNotFound)
	return ok
}
