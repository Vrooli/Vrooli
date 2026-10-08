package agentactivity

import (
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
	}
	return records, nil
}

func (s *FileStore) Save(records []Record) error {
	return storage.WriteJSONAtomic(s.path, records)
}
