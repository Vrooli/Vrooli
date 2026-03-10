// Package testutil provides mock implementations of repository interfaces.
// These mocks enable unit testing of handlers without database dependencies.
//
// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#Mock-Organization
// DOC: docs/internal/SEAMS.md#Repository-Seam
package testutil

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"

	"lifestyle-dashboard/domain"
	"lifestyle-dashboard/repository"
)

// =============================================================================
// MockEventRepository - In-memory event storage for testing
// =============================================================================

// MockEventRepository implements repository.EventRepository for testing.
// It stores events in memory and provides configurable error injection.
type MockEventRepository struct {
	mu       sync.RWMutex
	events   map[string]*domain.Event
	CreateFn func(ctx context.Context, event *domain.Event) error // Optional override
	GetFn    func(ctx context.Context, id string) (*domain.Event, error)
	ListFn   func(ctx context.Context, filter repository.EventFilter) ([]domain.Event, error)
}

// NewMockEventRepository creates a new mock event repository.
func NewMockEventRepository() *MockEventRepository {
	return &MockEventRepository{
		events: make(map[string]*domain.Event),
	}
}

// Create adds an event to the mock storage.
func (m *MockEventRepository) Create(ctx context.Context, event *domain.Event) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, event)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	now := time.Now().UTC().Format(time.RFC3339)
	event.CreatedAt = now
	if event.Timestamp == "" {
		event.Timestamp = now
	}

	if event.Payload == nil {
		event.Payload = json.RawMessage("{}")
	}

	// Store a copy to avoid mutation issues
	stored := *event
	m.events[event.ID] = &stored
	return nil
}

// GetByID retrieves an event from mock storage.
func (m *MockEventRepository) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, id)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	event, ok := m.events[id]
	if !ok {
		return nil, repository.ErrNotFound{Entity: "event", ID: id}
	}

	// Return a copy to avoid mutation
	copy := *event
	return &copy, nil
}

// List returns events matching the filter.
func (m *MockEventRepository) List(ctx context.Context, filter repository.EventFilter) ([]domain.Event, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, filter)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []domain.Event
	for _, e := range m.events {
		if filter.Domain != "" && e.Domain != filter.Domain {
			continue
		}
		if filter.EventType != "" && e.EventType != filter.EventType {
			continue
		}
		result = append(result, *e)
	}

	limit := filter.Limit
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// WithEvent adds a pre-existing event to the mock (builder pattern).
func (m *MockEventRepository) WithEvent(e *domain.Event) *MockEventRepository {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events[e.ID] = e
	return m
}

// WithCreateError configures the mock to return an error on Create.
func (m *MockEventRepository) WithCreateError(err error) *MockEventRepository {
	m.CreateFn = func(ctx context.Context, event *domain.Event) error {
		return err
	}
	return m
}

// =============================================================================
// MockDomainRepository - In-memory domain storage for testing
// =============================================================================

// MockDomainRepository implements repository.DomainRepository for testing.
type MockDomainRepository struct {
	mu       sync.RWMutex
	domains  map[string]*domain.Domain
	UpsertFn func(ctx context.Context, d *domain.Domain) error
	GetFn    func(ctx context.Context, name string) (*domain.Domain, error)
	ListFn   func(ctx context.Context) ([]domain.Domain, error)
}

// NewMockDomainRepository creates a new mock domain repository.
func NewMockDomainRepository() *MockDomainRepository {
	return &MockDomainRepository{
		domains: make(map[string]*domain.Domain),
	}
}

// Upsert creates or updates a domain.
func (m *MockDomainRepository) Upsert(ctx context.Context, d *domain.Domain) error {
	if m.UpsertFn != nil {
		return m.UpsertFn(ctx, d)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	if d.RegisteredAt == "" {
		d.RegisteredAt = now
	}
	d.UpdatedAt = now
	if d.Status == "" {
		d.Status = "active"
	}

	stored := *d
	m.domains[d.Name] = &stored
	return nil
}

// GetByName retrieves a domain by name.
func (m *MockDomainRepository) GetByName(ctx context.Context, name string) (*domain.Domain, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, name)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	d, ok := m.domains[name]
	if !ok {
		return nil, repository.ErrNotFound{Entity: "domain", ID: name}
	}

	copy := *d
	return &copy, nil
}

// List returns all domains.
func (m *MockDomainRepository) List(ctx context.Context) ([]domain.Domain, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []domain.Domain
	for _, d := range m.domains {
		result = append(result, *d)
	}
	return result, nil
}

// UpdateStatus updates a domain's status.
func (m *MockDomainRepository) UpdateStatus(ctx context.Context, name, status, lastHealthAt string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	d, ok := m.domains[name]
	if !ok {
		return repository.ErrNotFound{Entity: "domain", ID: name}
	}

	d.Status = status
	d.LastHealthAt = &lastHealthAt
	d.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return nil
}

// Update applies partial updates to a domain.
func (m *MockDomainRepository) Update(ctx context.Context, name string, updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	d, ok := m.domains[name]
	if !ok {
		return repository.ErrNotFound{Entity: "domain", ID: name}
	}

	if displayName, ok := updates["display_name"].(string); ok {
		d.DisplayName = displayName
	}
	if description, ok := updates["description"].(string); ok {
		d.Description = description
	}
	d.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return nil
}

// WithDomain adds a pre-existing domain to the mock (builder pattern).
func (m *MockDomainRepository) WithDomain(d *domain.Domain) *MockDomainRepository {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.domains[d.Name] = d
	return m
}

// =============================================================================
// MockStatsRepository - Stats aggregation for testing
// =============================================================================

// MockStatsRepository implements repository.StatsRepository for testing.
type MockStatsRepository struct {
	Timeline      []domain.TimelineEntry
	Summary       *domain.SummaryResponse
	Score         *domain.ScoreResponse
	TimelineError error
	SummaryError  error
	ScoreError    error
}

// NewMockStatsRepository creates a new mock stats repository.
func NewMockStatsRepository() *MockStatsRepository {
	return &MockStatsRepository{
		Timeline: []domain.TimelineEntry{},
		Summary: &domain.SummaryResponse{
			TotalEvents:    0,
			ActiveDomains:  0,
			EventsByDomain: []domain.DomainCount{},
		},
		Score: &domain.ScoreResponse{
			Current: domain.LifestyleScore{
				Score:       0,
				Date:        time.Now().Format("2006-01-02"),
				DataQuality: "insufficient",
				Message:     "No activity recorded today.",
			},
			History: []domain.ScoreHistoryEntry{},
		},
	}
}

// GetTimeline returns the configured timeline.
func (m *MockStatsRepository) GetTimeline(ctx context.Context, days int) ([]domain.TimelineEntry, error) {
	if m.TimelineError != nil {
		return nil, m.TimelineError
	}
	return m.Timeline, nil
}

// GetSummary returns the configured summary.
func (m *MockStatsRepository) GetSummary(ctx context.Context) (*domain.SummaryResponse, error) {
	if m.SummaryError != nil {
		return nil, m.SummaryError
	}
	return m.Summary, nil
}

// GetLifestyleScore returns the configured score.
// [REQ:LD-UI-SCORE] Mock implementation for testing score display.
func (m *MockStatsRepository) GetLifestyleScore(ctx context.Context, historyDays int) (*domain.ScoreResponse, error) {
	if m.ScoreError != nil {
		return nil, m.ScoreError
	}
	return m.Score, nil
}

// WithTimeline configures the timeline data (builder pattern).
func (m *MockStatsRepository) WithTimeline(entries []domain.TimelineEntry) *MockStatsRepository {
	m.Timeline = entries
	return m
}

// WithSummary configures the summary data (builder pattern).
func (m *MockStatsRepository) WithSummary(s *domain.SummaryResponse) *MockStatsRepository {
	m.Summary = s
	return m
}

// WithScore configures the score data (builder pattern).
func (m *MockStatsRepository) WithScore(s *domain.ScoreResponse) *MockStatsRepository {
	m.Score = s
	return m
}
