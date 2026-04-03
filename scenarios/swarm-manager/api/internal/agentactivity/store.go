package agentactivity

import (
	"path/filepath"
	"strings"

	"swarm-manager/internal/pathutil"
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
		path = filepath.Join(pathutil.ResolveScenarioRoot("swarm-manager"), ".vrooli", "agent-activities.json")
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
