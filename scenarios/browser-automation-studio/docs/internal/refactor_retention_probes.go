// Investigation of real retention selection with an in-memory index/filesystem.
// No production data or disk is changed. From api/:
// GOPROXY=off GOTOOLCHAIN=local go run ../docs/internal/refactor_retention_probes.go
// Exit 0 means diagnostics completed; inspect expected_behavior_met.
package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/retention"
)

type observation struct {
	ID       string `json:"id"`
	Expected string `json:"expected"`
	Actual   any    `json:"actual"`
	Met      bool   `json:"expected_behavior_met"`
}

type memoryStore struct {
	rows    []*database.ExecutionIndex
	deleted []uuid.UUID
}

func (s *memoryStore) GetExecution(_ context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	for _, row := range s.rows {
		if row.ID == id {
			return row, nil
		}
	}
	return nil, database.ErrNotFound
}
func (s *memoryStore) ListExecutions(_ context.Context, workflow, project *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error) {
	var out []*database.ExecutionIndex
	for _, row := range s.rows {
		if workflow == nil || row.WorkflowID == *workflow {
			out = append(out, row)
		}
	}
	return out, nil
}
func (s *memoryStore) byStatus(status string, limit, offset int, oldest bool) []*database.ExecutionIndex {
	var out []*database.ExecutionIndex
	for _, row := range s.rows {
		if row.Status == status {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if oldest {
			return out[i].StartedAt.Before(out[j].StartedAt)
		}
		return out[i].StartedAt.After(out[j].StartedAt)
	})
	if offset >= len(out) {
		return nil
	}
	out = out[offset:]
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
func (s *memoryStore) ListExecutionsByStatus(_ context.Context, status string, limit, offset int) ([]*database.ExecutionIndex, error) {
	return s.byStatus(status, limit, offset, false), nil
}

// Implements the optional production optimization, including its real oldest-first LIMIT semantics.
func (s *memoryStore) ListExecutionsByStatusOldest(_ context.Context, status string, limit, offset int) ([]*database.ExecutionIndex, error) {
	return s.byStatus(status, limit, offset, true), nil
}
func (s *memoryStore) DeleteExecution(_ context.Context, id uuid.UUID) error {
	s.deleted = append(s.deleted, id)
	for i, row := range s.rows {
		if row.ID == id {
			s.rows = append(s.rows[:i], s.rows[i+1:]...)
			break
		}
	}
	return nil
}

type memoryFS struct{ removed []string }

func (f *memoryFS) DirSize(string) (int64, bool, error) { return 100, true, nil }
func (f *memoryFS) RemoveAll(name string) error         { f.removed = append(f.removed, name); return nil }

var now = time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)

const root = "/synthetic/retention-investigation"

func fixture() (*memoryStore, *memoryFS, *retention.Service) {
	wf := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	store, fs := &memoryStore{}, &memoryFS{}
	for i, age := range []int{10, 5, 1} {
		id := uuid.UUID{}
		id[15] = byte(i + 1)
		started := now.Add(-time.Duration(age) * 24 * time.Hour)
		store.rows = append(store.rows, &database.ExecutionIndex{ID: id, WorkflowID: wf, Status: database.ExecutionStatusCompleted,
			StartedAt: started, CompletedAt: &started, ResultPath: filepath.Join(root, id.String(), "result.json")})
	}
	return store, fs, retention.NewService(store, fs, root, nil).WithClock(func() time.Time { return now })
}
func sweep(s *retention.Service, options retention.Options) *retention.Report {
	r, err := s.Sweep(context.Background(), options)
	if err != nil {
		panic(err)
	}
	return r
}
func main() {
	var results []observation
	store, _, service := fixture()
	r := sweep(service, retention.Options{KeepLatest: 1, Apply: true})
	results = append(results, observation{"unbounded-keep-latest-control", "Unbounded sweep deletes two older rows and protects newest", r, r.RemovedCount == 2 && len(store.rows) == 1 && store.rows[0].ID[15] == 3})

	store, _, service = fixture()
	var counts []int
	for i := 0; i < 3; i++ {
		counts = append(counts, sweep(service, retention.Options{KeepLatest: 1, MaxItems: 1, Apply: true}).RemovedCount)
	}
	results = append(results, observation{"bounded-keep-latest-progress", "Repeated size-one batches delete eligible old rows while preserving the actual newest row",
		map[string]any{"removed_per_sweep": counts, "remaining_rows": len(store.rows)}, len(store.rows) == 1 && store.rows[0].ID[15] == 3})

	store, _, service = fixture()
	old := store.rows[0].ID
	preview := sweep(service, retention.Options{KeepLatest: 1})
	apply := sweep(service, retention.Options{KeepLatest: 1, ExecutionIDs: []uuid.UUID{old}, Apply: true})
	results = append(results, observation{"preview-subset-keep-latest", "Applying an eligible old preview ID does not turn that row into the protected newest row",
		map[string]any{"preview_eligible_count": preview.RemovedCount, "apply_removed": apply.RemovedCount, "apply_skipped": apply.Skipped}, apply.RemovedCount == 1})

	store, fs, service := fixture()
	store.rows[0].Status = "running"
	r = sweep(service, retention.Options{ExecutionIDs: []uuid.UUID{store.rows[0].ID}, Apply: true})
	results = append(results, observation{"active-execution-retention-control", "An explicitly selected running execution is not deleted", map[string]any{"removed": r.RemovedCount, "filesystem_deletes": len(fs.removed)}, r.RemovedCount == 0 && len(fs.removed) == 0})

	store, _, service = fixture()
	r = sweep(service, retention.Options{MaxItems: 1, Apply: true})
	results = append(results, observation{"bounded-no-keep-control", "A bounded sweep without keep_latest deletes the oldest row", map[string]any{"removed": r.RemovedCount, "deleted_ids": store.deleted}, r.RemovedCount == 1 && store.deleted[0][15] == 1})

	out := map[string]any{"schema_version": 1, "observed_at": time.Now().UTC(), "scope": "Actual retention service with in-memory store/filesystem; production oldest-first LIMIT behavior modelled; no database or filesystem deletion", "results": results}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		panic(err)
	}
}
