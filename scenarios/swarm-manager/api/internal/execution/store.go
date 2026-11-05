package execution

import (
	"strings"

	"swarm-manager/internal/runtimepaths"
	"swarm-manager/internal/storage"
)

// DOC: docs/concepts/ARCHITECTURE.md#logical-architecture
// DOC: docs/internal/SEAMS.md
// DOC: docs/internal/INVARIANTS.md

// Store persists execution records.
type Store interface {
	Load() ([]Record, error)
	Save([]Record) error
}

// FileStore is a filesystem-backed execution store.
type FileStore struct {
	path string
}

// NewStore creates a store at the provided path.
func NewStore(path string) *FileStore {
	if strings.TrimSpace(path) == "" {
		if resolved, err := runtimepaths.StatePath("execution-runs.json"); err == nil {
			path = resolved
		}
	}
	return &FileStore{path: path}
}

// Load returns all execution records, migrating any stale data.
func (s *FileStore) Load() ([]Record, error) {
	var records []Record
	exists, err := storage.ReadJSON(s.path, &records)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []Record{}, nil
	}
	for i := range records {
		if strings.TrimSpace(records[i].ExecutionID) == "" {
			records[i].ExecutionID = "unknown"
		}
	}
	records = migrateRecords(records)
	return records, nil
}

// migrateRecords fixes stale records from before status expansion.
// Records with status "running" but no RunID are orphaned and can never
// be resolved — mark them failed so they surface in the UI for retry.
func migrateRecords(records []Record) []Record {
	for i := range records {
		if records[i].Status == StatusRunning && strings.TrimSpace(records[i].RunID) == "" {
			records[i].Status = StatusFailed
			records[i].FailureReason = "orphaned execution: no run ID"
		}
		// Migrate legacy scheduled records to pending/manual.
		if records[i].Status == "scheduled" {
			records[i].Status = StatusPending
		}
		if records[i].Mode == "scheduled" {
			records[i].Mode = ModeManual
		}
	}
	return records
}

// Save writes execution records atomically.
func (s *FileStore) Save(records []Record) error {
	return storage.WriteJSONAtomic(s.path, records)
}
