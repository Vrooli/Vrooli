package captures

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *FileStore {
	t.Helper()
	metaPath := filepath.Join(t.TempDir(), "captures_meta.json")
	store, err := NewFileStore(metaPath)
	require.NoError(t, err)
	return store
}

func testCapture(scenario, id string) Capture {
	return Capture{
		ID:            id,
		ScenarioName:  scenario,
		Type:          CaptureScreenshot,
		Filename:      "screenshot-123.png",
		FileSizeBytes: 1024,
		SourceSession: "session-1",
		CreatedAt:     time.Now(),
	}
}

func TestAdd_PersistsMetadata(t *testing.T) {
	store := newTestStore(t)
	cap := testCapture("my-app", "cap-1")

	require.NoError(t, store.Add(cap))

	caps, err := store.List("my-app")
	require.NoError(t, err)
	require.Len(t, caps, 1)
	assert.Equal(t, "cap-1", caps[0].ID)

	// Reload from disk to verify persistence
	store2, err := NewFileStore(store.metaPath)
	require.NoError(t, err)
	caps2, err := store2.List("my-app")
	require.NoError(t, err)
	require.Len(t, caps2, 1)
	assert.Equal(t, "cap-1", caps2[0].ID)
}

func TestList_EmptyScenario(t *testing.T) {
	store := newTestStore(t)
	caps, err := store.List("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, caps)
}

func TestList_ReturnsCorrectScenario(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.Add(testCapture("app-a", "cap-1")))
	require.NoError(t, store.Add(testCapture("app-b", "cap-2")))

	capsA, err := store.List("app-a")
	require.NoError(t, err)
	assert.Len(t, capsA, 1)
	assert.Equal(t, "cap-1", capsA[0].ID)

	capsB, err := store.List("app-b")
	require.NoError(t, err)
	assert.Len(t, capsB, 1)
	assert.Equal(t, "cap-2", capsB[0].ID)
}

func TestDelete_RemovesEntry(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.Add(testCapture("my-app", "cap-1")))
	require.NoError(t, store.Add(testCapture("my-app", "cap-2")))

	require.NoError(t, store.Delete("my-app", "cap-1"))

	caps, err := store.List("my-app")
	require.NoError(t, err)
	require.Len(t, caps, 1)
	assert.Equal(t, "cap-2", caps[0].ID)
}

func TestDelete_NotFound(t *testing.T) {
	store := newTestStore(t)
	err := store.Delete("my-app", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteAll_ClearsScenario(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.Add(testCapture("my-app", "cap-1")))
	require.NoError(t, store.Add(testCapture("my-app", "cap-2")))

	deleted, err := store.DeleteAll("my-app")
	require.NoError(t, err)
	assert.Len(t, deleted, 2)

	caps, err := store.List("my-app")
	require.NoError(t, err)
	assert.Empty(t, caps)
}

func TestDeleteAll_EmptyScenario(t *testing.T) {
	store := newTestStore(t)
	deleted, err := store.DeleteAll("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, deleted)
}

func TestSummary_CalculatesCorrectly(t *testing.T) {
	store := newTestStore(t)
	c1 := testCapture("my-app", "cap-1")
	c1.FileSizeBytes = 1000
	c2 := testCapture("my-app", "cap-2")
	c2.FileSizeBytes = 2000
	require.NoError(t, store.Add(c1))
	require.NoError(t, store.Add(c2))

	summary, err := store.Summary("my-app")
	require.NoError(t, err)
	assert.Equal(t, 2, summary.Count)
	assert.Equal(t, int64(3000), summary.TotalBytes)
}

func TestSummary_EmptyScenario(t *testing.T) {
	store := newTestStore(t)
	summary, err := store.Summary("nonexistent")
	require.NoError(t, err)
	assert.Equal(t, 0, summary.Count)
	assert.Equal(t, int64(0), summary.TotalBytes)
}

func TestConcurrentAccess(t *testing.T) {
	store := newTestStore(t)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cap := testCapture("my-app", fmt.Sprintf("cap-%d", idx))
			_ = store.Add(cap)
		}(i)
	}
	wg.Wait()

	caps, err := store.List("my-app")
	require.NoError(t, err)
	assert.Len(t, caps, 20)
}
