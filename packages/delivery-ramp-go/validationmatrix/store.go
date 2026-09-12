package validationmatrix

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/proto"
)

type Store interface {
	Save(*MatrixRun) error
	Get(string) (*MatrixRun, bool)
	GetByIdempotencyKey(string) (*MatrixRun, bool)
	Update(string, func(*MatrixRun)) bool
	List() []*MatrixRun
}

type FileStore struct {
	mu      sync.RWMutex
	dataDir string
	runs    map[string]*MatrixRun
}

func NewFileStore(dataDir string) (*FileStore, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("validation matrix data directory is required")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create validation matrix data directory: %w", err)
	}
	s := &FileStore{dataDir: dataDir, runs: make(map[string]*MatrixRun)}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *FileStore) Save(run *MatrixRun) error {
	if run == nil || run.RunID == "" {
		return fmt.Errorf("matrix run identity is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal matrix run: %w", err)
	}
	path := filepath.Join(s.dataDir, run.RunID+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write matrix run: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("persist matrix run: %w", err)
	}
	s.runs[run.RunID] = cloneRun(run)
	return nil
}

func (s *FileStore) Get(runID string) (*MatrixRun, bool) {
	s.mu.RLock()
	run, ok := s.runs[runID]
	if ok {
		run = cloneRun(run)
	}
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return run, true
}

func (s *FileStore) GetByIdempotencyKey(key string) (*MatrixRun, bool) {
	if key == "" {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, run := range s.runs {
		if run.IdempotencyKey == key {
			return cloneRun(run), true
		}
	}
	return nil, false
}

func (s *FileStore) Update(runID string, mutate func(*MatrixRun)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok {
		return false
	}
	mutate(run)
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return false
	}
	path := filepath.Join(s.dataDir, run.RunID+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return false
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return false
	}
	return true
}

func (s *FileStore) List() []*MatrixRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*MatrixRun, 0, len(s.runs))
	for _, run := range s.runs {
		result = append(result, cloneRun(run))
	}
	return result
}

func (s *FileStore) load() error {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		return fmt.Errorf("read validation matrix data directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dataDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read matrix run %q: %w", entry.Name(), err)
		}
		var run MatrixRun
		if err := json.Unmarshal(data, &run); err != nil {
			return fmt.Errorf("decode matrix run %q: %w", entry.Name(), err)
		}
		if run.RunID == "" || run.Matrix == nil {
			return fmt.Errorf("matrix run %q is missing immutable identity", entry.Name())
		}
		s.runs[run.RunID] = cloneRun(&run)
	}
	return nil
}

func cloneRun(run *MatrixRun) *MatrixRun {
	if run == nil {
		return nil
	}
	copy := *run
	copy.Selection = cloneSelection(run.Selection)
	if run.Matrix != nil {
		copy.Matrix = proto.Clone(run.Matrix).(*domainv1.ValidationMatrix)
	}
	copy.Cells = make([]*CellRecord, len(run.Cells))
	for index, cell := range run.Cells {
		copy.Cells[index] = cloneCellRecord(cell)
	}
	if run.Gate != nil {
		copy.Gate = proto.Clone(run.Gate).(*domainv1.ReleaseGate)
	}
	return &copy
}
