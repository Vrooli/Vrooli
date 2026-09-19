package captures

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/vrooli/api-core/storage"
)

// Store persists capture metadata.
type Store interface {
	List(scenarioName string) ([]Capture, error)
	Add(capture Capture) error
	Delete(scenarioName, captureID string) error
	DeleteAll(scenarioName string) ([]Capture, error)
	Summary(scenarioName string) (CapturesSummary, error)
}

// FileStore is a JSON-file-backed Store implementation.
type FileStore struct {
	mu       sync.RWMutex
	metaPath string
	data     map[string][]Capture // keyed by scenario name
}

// NewFileStore creates a FileStore at the given metadata path, loading existing data.
func NewFileStore(metaPath string) (*FileStore, error) {
	fs := &FileStore{
		metaPath: metaPath,
		data:     make(map[string][]Capture),
	}
	if err := fs.load(); err != nil {
		return nil, err
	}
	return fs, nil
}

func (s *FileStore) load() error {
	raw, err := os.ReadFile(s.metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading captures metadata: %w", err)
	}
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, &s.data)
}

func (s *FileStore) flush() error {
	raw, err := json.Marshal(s.data)
	if err != nil {
		return fmt.Errorf("marshaling captures metadata: %w", err)
	}
	return storage.WriteFileAtomic(s.metaPath, raw, 0)
}

func (s *FileStore) List(scenarioName string) ([]Capture, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	caps := s.data[scenarioName]
	if caps == nil {
		return []Capture{}, nil
	}
	out := make([]Capture, len(caps))
	copy(out, caps)
	return out, nil
}

func (s *FileStore) Add(capture Capture) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[capture.ScenarioName] = append(s.data[capture.ScenarioName], capture)
	return s.flush()
}

func (s *FileStore) Delete(scenarioName, captureID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	caps := s.data[scenarioName]
	for i, c := range caps {
		if c.ID == captureID {
			s.data[scenarioName] = append(caps[:i], caps[i+1:]...)
			return s.flush()
		}
	}
	return fmt.Errorf("capture %q not found for scenario %q", captureID, scenarioName)
}

func (s *FileStore) DeleteAll(scenarioName string) ([]Capture, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	deleted := s.data[scenarioName]
	if deleted == nil {
		return []Capture{}, nil
	}
	out := make([]Capture, len(deleted))
	copy(out, deleted)
	delete(s.data, scenarioName)
	if err := s.flush(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *FileStore) Summary(scenarioName string) (CapturesSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	caps := s.data[scenarioName]
	var total int64
	for _, c := range caps {
		total += c.FileSizeBytes
	}
	return CapturesSummary{
		Count:      len(caps),
		TotalBytes: total,
	}, nil
}
