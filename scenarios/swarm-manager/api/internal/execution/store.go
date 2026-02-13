package execution

import (
	"path/filepath"
	"strings"

	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/storage"
)

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
		path = filepath.Join(pathutil.ResolveScenarioRoot("swarm-manager"), ".vrooli", "execution-runs.json")
	}
	return &FileStore{path: path}
}

// Load returns all execution records.
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
	return records, nil
}

// Save writes execution records atomically.
func (s *FileStore) Save(records []Record) error {
	return storage.WriteJSONAtomic(s.path, records)
}
