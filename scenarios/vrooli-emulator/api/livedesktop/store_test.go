package livedesktop

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreate_Success(t *testing.T) {
	store := NewInMemoryStore()
	session := &Session{ID: "s1", State: StateCreating, CreatedAt: time.Now()}
	err := store.Create(session)
	require.NoError(t, err)

	got, err := store.Get("s1")
	require.NoError(t, err)
	assert.Equal(t, "s1", got.ID)
}

func TestCreate_Duplicate(t *testing.T) {
	store := NewInMemoryStore()
	session := &Session{ID: "s1", State: StateCreating}
	require.NoError(t, store.Create(session))

	err := store.Create(&Session{ID: "s1", State: StateRunning})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestGet_NotFound(t *testing.T) {
	store := NewInMemoryStore()
	_, err := store.Get("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdate_NotFound(t *testing.T) {
	store := NewInMemoryStore()
	err := store.Update(&Session{ID: "nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDelete_NotFound(t *testing.T) {
	store := NewInMemoryStore()
	err := store.Delete("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestList_Empty(t *testing.T) {
	store := NewInMemoryStore()
	sessions := store.List()
	assert.Empty(t, sessions)
}

func TestList_Multiple(t *testing.T) {
	store := NewInMemoryStore()
	require.NoError(t, store.Create(&Session{ID: "s1", State: StateRunning}))
	require.NoError(t, store.Create(&Session{ID: "s2", State: StateStopped}))
	require.NoError(t, store.Create(&Session{ID: "s3", State: StateError}))

	sessions := store.List()
	assert.Len(t, sessions, 3)
}

func TestActiveSessions_FiltersByState(t *testing.T) {
	store := NewInMemoryStore()
	require.NoError(t, store.Create(&Session{ID: "creating", State: StateCreating}))
	require.NoError(t, store.Create(&Session{ID: "running", State: StateRunning}))
	require.NoError(t, store.Create(&Session{ID: "stopped", State: StateStopped}))
	require.NoError(t, store.Create(&Session{ID: "error", State: StateError}))
	require.NoError(t, store.Create(&Session{ID: "stopping", State: StateStopping}))

	active := store.ActiveSessions()
	assert.Len(t, active, 2)

	ids := map[string]bool{}
	for _, s := range active {
		ids[s.ID] = true
	}
	assert.True(t, ids["creating"])
	assert.True(t, ids["running"])
}

func TestConcurrentAccess(t *testing.T) {
	store := NewInMemoryStore()
	const n = 50

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			id := "session-" + time.Now().Format("150405.000000000") + "-" + string(rune('A'+idx%26))
			_ = store.Create(&Session{ID: id, State: StateRunning})
			_, _ = store.Get(id)
			_ = store.List()
		}(i)
	}
	wg.Wait()

	// Verify no panics occurred and store is consistent
	sessions := store.List()
	assert.True(t, len(sessions) > 0, "expected at least some sessions to be created")
}
