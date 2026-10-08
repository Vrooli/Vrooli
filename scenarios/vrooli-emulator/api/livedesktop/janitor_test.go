package livedesktop

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReapIdleSessions(t *testing.T) {
	store := NewInMemoryStore()
	svc := newTestService(store, newMockBackend())

	// Create a session with a very old heartbeat
	session := &Session{
		ID:            "idle-session",
		ScenarioName:  "test",
		State:         StateRunning,
		CreatedAt:     time.Now().Add(-2 * time.Hour),
		LastHeartbeat: time.Now().Add(-2 * time.Hour),
	}
	require.NoError(t, store.Create(session))

	reapIdleSessions(svc, 1*time.Hour)

	got, err := store.Get("idle-session")
	require.NoError(t, err)
	assert.Equal(t, StateStopped, got.State)
}

func TestReapIdleSessions_ActiveNotReaped(t *testing.T) {
	store := NewInMemoryStore()
	svc := newTestService(store, newMockBackend())

	// Create a session with a recent heartbeat
	session := &Session{
		ID:            "active-session",
		ScenarioName:  "test",
		State:         StateRunning,
		CreatedAt:     time.Now(),
		LastHeartbeat: time.Now(),
	}
	require.NoError(t, store.Create(session))

	reapIdleSessions(svc, 1*time.Hour)

	got, err := store.Get("active-session")
	require.NoError(t, err)
	assert.Equal(t, StateRunning, got.State)
}
