// Package testutil provides shared test helpers, mock implementations,
// and factory functions for vrooli-events tests.
//
// This package is designed for use by the api/ test package and any other
// packages that don't create import cycles. Internal packages (store, broker)
// keep their own local test helpers to avoid cycles.
package testutil

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
)

// MakeEvent creates a store.Event with sensible defaults for testing.
// Override any field after creation as needed.
func MakeEvent(eventID, eventType, source string) store.Event {
	return store.Event{
		EventID:        eventID,
		SourceScenario: source,
		TargetScenario: "target-1",
		EventType:      eventType,
		CorrelationID:  "corr-1",
		Payload:        []byte(`{"test": true}`),
		Metadata:       map[string]string{"key": "value"},
	}
}

// NewTestStore creates an in-memory SQLiteStore for testing and registers
// cleanup via t.Cleanup. This is the standard way to get a real store in tests.
func NewTestStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	s, err := store.NewSQLiteStore(context.Background(), store.SQLiteConfig{})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// NewTestPolicyStore creates an in-memory policy SQLiteStore for testing.
func NewTestPolicyStore(t *testing.T) *policy.SQLiteStore {
	t.Helper()
	eventStore := NewTestStore(t)
	ps, err := policy.NewSQLiteStore(eventStore.DB())
	if err != nil {
		t.Fatalf("new policy store: %v", err)
	}
	return ps
}

// NewTestSubscriptionStore creates an in-memory subscription SQLiteStore for testing.
func NewTestSubscriptionStore(t *testing.T) *subscription.SQLiteStore {
	t.Helper()
	eventStore := NewTestStore(t)
	ss, err := subscription.NewSQLiteStore(eventStore.DB())
	if err != nil {
		t.Fatalf("new subscription store: %v", err)
	}
	return ss
}
