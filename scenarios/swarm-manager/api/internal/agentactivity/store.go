package agentactivity

import (
	"fmt"
	"strings"

	"swarm-manager/internal/runtimepaths"
	"swarm-manager/internal/storage"
)

type Store interface {
	Load() ([]Record, error)
	Save([]Record) error
}

type FileStore struct {
	path string
}

func NewStore(path string) *FileStore {
	if strings.TrimSpace(path) == "" {
		if resolved, err := runtimepaths.StatePath("agent-activities.json"); err == nil {
			path = resolved
		}
	}
	return &FileStore{path: path}
}

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
		if records[i].Metadata == nil {
			records[i].Metadata = map[string]string{}
		}
		if !records[i].Status.Valid() {
			return nil, fmt.Errorf("agent activity %q has unknown persisted status %q", records[i].ActivityID, records[i].Status)
		}
	}
	return records, nil
}

func (s *FileStore) Save(records []Record) error {
	for _, record := range records {
		if !record.Status.Valid() {
			return fmt.Errorf("agent activity %q has invalid status %q", record.ActivityID, record.Status)
		}
	}
	return storage.WriteJSONAtomic(s.path, records)
}
