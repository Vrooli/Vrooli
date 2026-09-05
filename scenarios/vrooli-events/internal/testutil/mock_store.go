// Package testutil provides shared test helpers, mock implementations,
// and factory functions for vrooli-events tests.
//
// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
package testutil

import (
	"context"
	"fmt"
	"sync"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
)

// MockStore implements store.Store with configurable behavior per method.
// Use the With* builder methods to set up return values or error injection.
//
// Zero-value MockStore returns empty/zero results for all methods.
type MockStore struct {
	mu sync.Mutex

	// Configurable return values
	insertID    int64
	insertErr   error
	queryResult []store.Event
	queryErr    error
	sinceResult []store.Event
	sinceErr    error
	pruneResult store.PruneResult
	pruneErr    error
	statsResult store.Stats
	statsErr    error
	closeErr    error

	// Call tracking
	InsertCalls []store.Event
	QueryCalls  []store.QueryFilters
	SinceCalls  []struct {
		LastID int64
		Limit  int
	}
	PruneCalls int
	StatsCalls int
}

// Compile-time interface check.
var _ store.Store = (*MockStore)(nil)

// WithInsertResult sets the return values for Insert.
func (m *MockStore) WithInsertResult(id int64, err error) *MockStore {
	m.insertID = id
	m.insertErr = err
	return m
}

// WithQueryResult sets the return values for Query.
func (m *MockStore) WithQueryResult(events []store.Event, err error) *MockStore {
	m.queryResult = events
	m.queryErr = err
	return m
}

// WithSinceResult sets the return values for GetSince.
func (m *MockStore) WithSinceResult(events []store.Event, err error) *MockStore {
	m.sinceResult = events
	m.sinceErr = err
	return m
}

// WithPruneResult sets the return values for Prune.
func (m *MockStore) WithPruneResult(result store.PruneResult, err error) *MockStore {
	m.pruneResult = result
	m.pruneErr = err
	return m
}

// WithStatsResult sets the return values for Stats.
func (m *MockStore) WithStatsResult(stats store.Stats, err error) *MockStore {
	m.statsResult = stats
	m.statsErr = err
	return m
}

// WithStatsError sets Stats to return an error.
func (m *MockStore) WithStatsError(err error) *MockStore {
	m.statsErr = err
	return m
}

func (m *MockStore) Insert(_ context.Context, e store.Event) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InsertCalls = append(m.InsertCalls, e)
	if m.insertErr != nil {
		return 0, m.insertErr
	}
	// Auto-increment if no explicit ID configured
	id := m.insertID
	if id == 0 {
		id = int64(len(m.InsertCalls))
	}
	return id, nil
}

func (m *MockStore) Query(_ context.Context, f store.QueryFilters) ([]store.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.QueryCalls = append(m.QueryCalls, f)
	return m.queryResult, m.queryErr
}

func (m *MockStore) GetSince(_ context.Context, lastID int64, limit int) ([]store.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SinceCalls = append(m.SinceCalls, struct {
		LastID int64
		Limit  int
	}{lastID, limit})
	return m.sinceResult, m.sinceErr
}

func (m *MockStore) Prune(_ context.Context) (store.PruneResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PruneCalls++
	return m.pruneResult, m.pruneErr
}

func (m *MockStore) Stats(_ context.Context) (store.Stats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StatsCalls++
	return m.statsResult, m.statsErr
}

func (m *MockStore) Close() error {
	return m.closeErr
}

// InsertCallCount returns the number of times Insert was called.
func (m *MockStore) InsertCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.InsertCalls)
}

// LastInsertCall returns the most recent Insert argument, or an error if none.
func (m *MockStore) LastInsertCall() (store.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.InsertCalls) == 0 {
		return store.Event{}, fmt.Errorf("no insert calls recorded")
	}
	return m.InsertCalls[len(m.InsertCalls)-1], nil
}
