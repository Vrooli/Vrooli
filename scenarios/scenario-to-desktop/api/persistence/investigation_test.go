package persistence

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"scenario-to-desktop-api/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HELPERS
// =============================================================================

// newTestStore creates an InvestigationStore backed by a temp directory.
func newTestStore(t *testing.T) *InvestigationStore {
	t.Helper()
	return NewInvestigationStore(t.TempDir())
}

// newTestStoreMemory creates an in-memory InvestigationStore (no disk persistence).
func newTestStoreMemory(t *testing.T) *InvestigationStore {
	t.Helper()
	return NewInvestigationStore("")
}

func newInvestigation(id, pipelineID string, status domain.InvestigationStatus) *domain.Investigation {
	now := time.Now()
	return &domain.Investigation{
		ID:         id,
		PipelineID: pipelineID,
		Status:     status,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// =============================================================================
// GenerateID
// =============================================================================

func TestGenerateID(t *testing.T) {
	t.Run("returns non-empty string with inv- prefix", func(t *testing.T) {
		id := GenerateID()
		assert.NotEmpty(t, id)
		assert.Contains(t, id, "inv-")
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		seen := make(map[string]struct{}, 100)
		for i := 0; i < 100; i++ {
			id := GenerateID()
			_, dup := seen[id]
			assert.False(t, dup, "duplicate ID: %s", id)
			seen[id] = struct{}{}
		}
	})
}

// =============================================================================
// CREATE
// =============================================================================

func TestCreate(t *testing.T) {
	t.Run("assigns ID if empty", func(t *testing.T) {
		s := newTestStore(t)
		inv := &domain.Investigation{PipelineID: "pipe-1"}

		err := s.Create(inv)
		require.NoError(t, err)
		assert.NotEmpty(t, inv.ID)
		assert.Contains(t, inv.ID, "inv-")
	})

	t.Run("preserves provided ID", func(t *testing.T) {
		s := newTestStore(t)
		inv := &domain.Investigation{ID: "custom-id", PipelineID: "pipe-1"}

		err := s.Create(inv)
		require.NoError(t, err)
		assert.Equal(t, "custom-id", inv.ID)
	})

	t.Run("sets timestamps", func(t *testing.T) {
		s := newTestStore(t)
		inv := &domain.Investigation{ID: "ts-test", PipelineID: "pipe-1"}

		before := time.Now().Add(-time.Second)
		err := s.Create(inv)
		require.NoError(t, err)
		after := time.Now().Add(time.Second)

		assert.True(t, inv.CreatedAt.After(before), "CreatedAt should be after test start")
		assert.True(t, inv.CreatedAt.Before(after), "CreatedAt should be before test end")
		assert.True(t, inv.UpdatedAt.After(before))
	})

	t.Run("preserves existing CreatedAt", func(t *testing.T) {
		s := newTestStore(t)
		customTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		inv := &domain.Investigation{ID: "keep-time", PipelineID: "pipe-1", CreatedAt: customTime}

		err := s.Create(inv)
		require.NoError(t, err)
		assert.Equal(t, customTime, inv.CreatedAt)
	})

	t.Run("persists to disk", func(t *testing.T) {
		dir := t.TempDir()
		s := NewInvestigationStore(dir)
		inv := &domain.Investigation{ID: "disk-test", PipelineID: "pipe-1"}

		err := s.Create(inv)
		require.NoError(t, err)

		path := filepath.Join(dir, "investigations", "disk-test.json")
		_, err = os.Stat(path)
		assert.NoError(t, err, "persisted file should exist")
	})

	t.Run("retrievable after create", func(t *testing.T) {
		s := newTestStore(t)
		inv := &domain.Investigation{ID: "get-test", PipelineID: "pipe-1", Status: domain.InvestigationStatusPending}

		require.NoError(t, s.Create(inv))

		got, err := s.Get("get-test")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "pipe-1", got.PipelineID)
		assert.Equal(t, domain.InvestigationStatusPending, got.Status)
	})
}

// =============================================================================
// GET / GET FOR PIPELINE
// =============================================================================

func TestGet(t *testing.T) {
	t.Run("returns nil for missing ID", func(t *testing.T) {
		s := newTestStore(t)
		got, err := s.Get("nonexistent")
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestGetForPipeline(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.Create(newInvestigation("inv-1", "pipe-A", domain.InvestigationStatusPending)))

	t.Run("returns investigation for matching pipeline", func(t *testing.T) {
		got, err := s.GetForPipeline("pipe-A", "inv-1")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "inv-1", got.ID)
	})

	t.Run("returns nil for wrong pipeline", func(t *testing.T) {
		got, err := s.GetForPipeline("pipe-B", "inv-1")
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("returns nil for missing ID", func(t *testing.T) {
		got, err := s.GetForPipeline("pipe-A", "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

// =============================================================================
// LIST
// =============================================================================

func TestList(t *testing.T) {
	s := newTestStore(t)

	// Create items with staggered times for deterministic ordering.
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		inv := &domain.Investigation{
			ID:         GenerateID(),
			PipelineID: "pipe-A",
			Status:     domain.InvestigationStatusCompleted,
			CreatedAt:  base.Add(time.Duration(i) * time.Hour),
		}
		require.NoError(t, s.Create(inv))
	}
	// Create one for a different pipeline.
	require.NoError(t, s.Create(newInvestigation("other", "pipe-B", domain.InvestigationStatusPending)))

	t.Run("filters by pipeline", func(t *testing.T) {
		result, err := s.List("pipe-A", 0)
		require.NoError(t, err)
		assert.Len(t, result, 5)
	})

	t.Run("excludes other pipelines", func(t *testing.T) {
		result, err := s.List("pipe-B", 0)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("sorts newest first", func(t *testing.T) {
		result, err := s.List("pipe-A", 0)
		require.NoError(t, err)
		for i := 1; i < len(result); i++ {
			assert.True(t, result[i-1].CreatedAt.After(result[i].CreatedAt) || result[i-1].CreatedAt.Equal(result[i].CreatedAt),
				"result[%d] should be >= result[%d]", i-1, i)
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		result, err := s.List("pipe-A", 2)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("empty pipeline returns empty slice", func(t *testing.T) {
		result, err := s.List("pipe-nonexistent", 0)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

// =============================================================================
// LIST ALL
// =============================================================================

func TestListAll(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.Create(newInvestigation("a", "pipe-A", domain.InvestigationStatusPending)))
	require.NoError(t, s.Create(newInvestigation("b", "pipe-B", domain.InvestigationStatusPending)))

	t.Run("returns all investigations", func(t *testing.T) {
		result, err := s.ListAll(0)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("respects limit", func(t *testing.T) {
		result, err := s.ListAll(1)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})
}

// =============================================================================
// GET ACTIVE
// =============================================================================

func TestGetActive(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.Create(newInvestigation("completed", "pipe-A", domain.InvestigationStatusCompleted)))
	require.NoError(t, s.Create(newInvestigation("pending", "pipe-A", domain.InvestigationStatusPending)))
	require.NoError(t, s.Create(newInvestigation("running", "pipe-A", domain.InvestigationStatusRunning)))
	require.NoError(t, s.Create(newInvestigation("failed", "pipe-A", domain.InvestigationStatusFailed)))

	t.Run("returns running/pending investigation", func(t *testing.T) {
		got, err := s.GetActive("pipe-A")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.True(t, got.Status == domain.InvestigationStatusPending || got.Status == domain.InvestigationStatusRunning)
	})

	t.Run("returns nil when no active for pipeline", func(t *testing.T) {
		got, err := s.GetActive("pipe-nonexistent")
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("returns latest active by CreatedAt", func(t *testing.T) {
		s2 := newTestStoreMemory(t)
		early := &domain.Investigation{ID: "early", PipelineID: "p", Status: domain.InvestigationStatusRunning, CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
		late := &domain.Investigation{ID: "late", PipelineID: "p", Status: domain.InvestigationStatusPending, CreatedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)}
		require.NoError(t, s2.Create(early))
		require.NoError(t, s2.Create(late))

		got, err := s2.GetActive("p")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "late", got.ID)
	})
}

// =============================================================================
// UPDATE
// =============================================================================

func TestUpdate(t *testing.T) {
	t.Run("modifies investigation and updates timestamp", func(t *testing.T) {
		s := newTestStoreMemory(t)
		inv := newInvestigation("upd-1", "pipe-A", domain.InvestigationStatusPending)
		require.NoError(t, s.Create(inv))

		originalUpdated := inv.UpdatedAt
		time.Sleep(time.Millisecond) // Ensure time passes

		err := s.Update("upd-1", func(inv *domain.Investigation) {
			inv.Progress = 50
		})
		require.NoError(t, err)

		got, _ := s.Get("upd-1")
		assert.Equal(t, 50, got.Progress)
		assert.True(t, got.UpdatedAt.After(originalUpdated) || got.UpdatedAt.Equal(originalUpdated))
	})

	t.Run("returns error for missing ID", func(t *testing.T) {
		s := newTestStoreMemory(t)
		err := s.Update("nonexistent", func(inv *domain.Investigation) {})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// =============================================================================
// UPDATE STATUS
// =============================================================================

func TestUpdateStatus(t *testing.T) {
	tests := []struct {
		name         string
		newStatus    domain.InvestigationStatus
		wantComplete bool
	}{
		{"to running (no CompletedAt)", domain.InvestigationStatusRunning, false},
		{"to completed (sets CompletedAt)", domain.InvestigationStatusCompleted, true},
		{"to failed (sets CompletedAt)", domain.InvestigationStatusFailed, true},
		{"to cancelled (sets CompletedAt)", domain.InvestigationStatusCancelled, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestStoreMemory(t)
			require.NoError(t, s.Create(newInvestigation("s-"+string(tc.newStatus), "pipe", domain.InvestigationStatusPending)))

			err := s.UpdateStatus("s-"+string(tc.newStatus), tc.newStatus)
			require.NoError(t, err)

			got, _ := s.Get("s-" + string(tc.newStatus))
			assert.Equal(t, tc.newStatus, got.Status)

			if tc.wantComplete {
				assert.NotNil(t, got.CompletedAt, "CompletedAt should be set for terminal status %s", tc.newStatus)
			} else {
				assert.Nil(t, got.CompletedAt)
			}
		})
	}
}

// =============================================================================
// UPDATE CONVENIENCE METHODS
// =============================================================================

func TestUpdateRunID(t *testing.T) {
	s := newTestStoreMemory(t)
	require.NoError(t, s.Create(newInvestigation("rid-1", "pipe", domain.InvestigationStatusRunning)))

	require.NoError(t, s.UpdateRunID("rid-1", "run-abc"))
	got, _ := s.Get("rid-1")
	require.NotNil(t, got.AgentRunID)
	assert.Equal(t, "run-abc", *got.AgentRunID)
}

func TestUpdateProgress(t *testing.T) {
	s := newTestStoreMemory(t)
	require.NoError(t, s.Create(newInvestigation("prog-1", "pipe", domain.InvestigationStatusRunning)))

	require.NoError(t, s.UpdateProgress("prog-1", 75))
	got, _ := s.Get("prog-1")
	assert.Equal(t, 75, got.Progress)
}

func TestUpdateFindings(t *testing.T) {
	s := newTestStoreMemory(t)
	require.NoError(t, s.Create(newInvestigation("find-1", "pipe", domain.InvestigationStatusRunning)))

	details := json.RawMessage(`{"key":"value"}`)
	require.NoError(t, s.UpdateFindings("find-1", "Found a bug", details))

	got, _ := s.Get("find-1")
	require.NotNil(t, got.Findings)
	assert.Equal(t, "Found a bug", *got.Findings)
	assert.JSONEq(t, `{"key":"value"}`, string(got.Details))
	assert.Equal(t, domain.InvestigationStatusCompleted, got.Status)
	assert.Equal(t, 100, got.Progress)
	assert.NotNil(t, got.CompletedAt)
}

func TestUpdateError(t *testing.T) {
	s := newTestStoreMemory(t)
	require.NoError(t, s.Create(newInvestigation("err-1", "pipe", domain.InvestigationStatusRunning)))

	require.NoError(t, s.UpdateError("err-1", "something broke"))
	got, _ := s.Get("err-1")
	assert.Equal(t, domain.InvestigationStatusFailed, got.Status)
	require.NotNil(t, got.ErrorMessage)
	assert.Equal(t, "something broke", *got.ErrorMessage)
	assert.NotNil(t, got.CompletedAt)
}

func TestUpdateErrorWithDetails(t *testing.T) {
	s := newTestStoreMemory(t)
	require.NoError(t, s.Create(newInvestigation("errd-1", "pipe", domain.InvestigationStatusRunning)))

	details := json.RawMessage(`{"stage":"build"}`)
	require.NoError(t, s.UpdateErrorWithDetails("errd-1", "build failed", details))

	got, _ := s.Get("errd-1")
	assert.Equal(t, domain.InvestigationStatusFailed, got.Status)
	require.NotNil(t, got.ErrorMessage)
	assert.Equal(t, "build failed", *got.ErrorMessage)
	assert.JSONEq(t, `{"stage":"build"}`, string(got.Details))
	assert.NotNil(t, got.CompletedAt)
}

// =============================================================================
// DELETE
// =============================================================================

func TestDelete(t *testing.T) {
	t.Run("removes from memory and disk", func(t *testing.T) {
		dir := t.TempDir()
		s := NewInvestigationStore(dir)
		require.NoError(t, s.Create(&domain.Investigation{ID: "del-1", PipelineID: "pipe"}))

		err := s.Delete("del-1")
		require.NoError(t, err)

		got, err := s.Get("del-1")
		require.NoError(t, err)
		assert.Nil(t, got, "should not be in memory after delete")

		path := filepath.Join(dir, "investigations", "del-1.json")
		_, err = os.Stat(path)
		assert.True(t, os.IsNotExist(err), "file should be removed from disk")
	})

	t.Run("returns error for missing ID", func(t *testing.T) {
		s := newTestStoreMemory(t)
		err := s.Delete("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// =============================================================================
// CLEANUP
// =============================================================================

func TestCleanup(t *testing.T) {
	s := newTestStoreMemory(t)

	// Old completed investigation
	old := newInvestigation("old", "pipe", domain.InvestigationStatusCompleted)
	oldTime := time.Now().Add(-48 * time.Hour)
	old.CompletedAt = &oldTime
	require.NoError(t, s.Create(old))

	// Recent completed investigation
	recent := newInvestigation("recent", "pipe", domain.InvestigationStatusCompleted)
	recentTime := time.Now().Add(-1 * time.Hour)
	recent.CompletedAt = &recentTime
	require.NoError(t, s.Create(recent))

	// Running investigation (no CompletedAt)
	require.NoError(t, s.Create(newInvestigation("running", "pipe", domain.InvestigationStatusRunning)))

	s.Cleanup(24 * time.Hour)

	got, _ := s.Get("old")
	assert.Nil(t, got, "old completed should be cleaned up")

	got, _ = s.Get("recent")
	assert.NotNil(t, got, "recent completed should remain")

	got, _ = s.Get("running")
	assert.NotNil(t, got, "running should remain")
}

// =============================================================================
// CANCELLATION TRACKING
// =============================================================================

func TestCancellation(t *testing.T) {
	t.Run("SetCancel and TakeCancel", func(t *testing.T) {
		s := newTestStoreMemory(t)
		called := false
		_, cancel := context.WithCancel(context.Background())
		wrappedCancel := context.CancelFunc(func() {
			called = true
			cancel()
		})

		s.SetCancel("inv-1", wrappedCancel)

		taken := s.TakeCancel("inv-1")
		require.NotNil(t, taken)
		taken()
		assert.True(t, called, "cancel function should be callable")

		// Take again should return nil (already taken)
		taken2 := s.TakeCancel("inv-1")
		assert.Nil(t, taken2)
	})

	t.Run("TakeCancel for nonexistent returns nil", func(t *testing.T) {
		s := newTestStoreMemory(t)
		assert.Nil(t, s.TakeCancel("nonexistent"))
	})

	t.Run("ClearCancel removes without calling", func(t *testing.T) {
		s := newTestStoreMemory(t)
		called := false
		s.SetCancel("inv-2", func() { called = true })

		s.ClearCancel("inv-2")
		assert.False(t, called, "cancel should not be called by ClearCancel")

		assert.Nil(t, s.TakeCancel("inv-2"), "should be gone after clear")
	})
}

// =============================================================================
// DISK PERSISTENCE / RELOAD
// =============================================================================

func TestDiskPersistence(t *testing.T) {
	t.Run("data survives store recreation", func(t *testing.T) {
		dir := t.TempDir()

		// Create store and add data
		s1 := NewInvestigationStore(dir)
		inv := &domain.Investigation{
			ID:         "persist-1",
			PipelineID: "pipe-A",
			Status:     domain.InvestigationStatusCompleted,
			Progress:   100,
		}
		require.NoError(t, s1.Create(inv))

		// Create a new store from the same directory
		s2 := NewInvestigationStore(dir)
		got, err := s2.Get("persist-1")
		require.NoError(t, err)
		require.NotNil(t, got, "should load from disk")
		assert.Equal(t, "pipe-A", got.PipelineID)
		assert.Equal(t, domain.InvestigationStatusCompleted, got.Status)
		assert.Equal(t, 100, got.Progress)
	})

	t.Run("skips non-json files on load", func(t *testing.T) {
		dir := t.TempDir()
		invDir := filepath.Join(dir, "investigations")
		require.NoError(t, os.MkdirAll(invDir, 0o755))

		// Write a non-json file
		require.NoError(t, os.WriteFile(filepath.Join(invDir, "readme.txt"), []byte("not json"), 0o644))

		// Write valid investigation
		inv := &domain.Investigation{ID: "valid", PipelineID: "p"}
		data, _ := json.Marshal(inv)
		require.NoError(t, os.WriteFile(filepath.Join(invDir, "valid.json"), data, 0o644))

		s := NewInvestigationStore(dir)
		got, err := s.Get("valid")
		require.NoError(t, err)
		assert.NotNil(t, got)
	})

	t.Run("skips corrupt json files on load", func(t *testing.T) {
		dir := t.TempDir()
		invDir := filepath.Join(dir, "investigations")
		require.NoError(t, os.MkdirAll(invDir, 0o755))

		require.NoError(t, os.WriteFile(filepath.Join(invDir, "bad.json"), []byte("{corrupt"), 0o644))

		// Should not panic or fail
		s := NewInvestigationStore(dir)
		all, err := s.ListAll(0)
		require.NoError(t, err)
		assert.Empty(t, all)
	})

	t.Run("empty directory is fine", func(t *testing.T) {
		dir := t.TempDir()
		s := NewInvestigationStore(dir)
		all, err := s.ListAll(0)
		require.NoError(t, err)
		assert.Empty(t, all)
	})

	t.Run("memory-only store ignores persistence", func(t *testing.T) {
		s := newTestStoreMemory(t)
		require.NoError(t, s.Create(newInvestigation("mem-1", "pipe", domain.InvestigationStatusPending)))

		got, err := s.Get("mem-1")
		require.NoError(t, err)
		assert.NotNil(t, got)
	})
}

// =============================================================================
// CONCURRENCY
// =============================================================================

func TestConcurrentAccess(t *testing.T) {
	s := newTestStoreMemory(t)
	const workers = 10
	const opsPerWorker = 50

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				id := GenerateID()
				inv := newInvestigation(id, "pipe-concurrent", domain.InvestigationStatusPending)
				_ = s.Create(inv)
				_, _ = s.Get(id)
				_, _ = s.List("pipe-concurrent", 5)
				_ = s.UpdateProgress(id, i*2)
				_, _ = s.GetActive("pipe-concurrent")
			}
		}(w)
	}

	wg.Wait()

	// Verify store is in consistent state
	all, err := s.ListAll(0)
	require.NoError(t, err)
	assert.Equal(t, workers*opsPerWorker, len(all))
}
