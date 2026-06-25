package records

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Store is the persistence seam for records. Production wiring uses
// FileStore; tests can substitute a fake.
//
// seam: records.Store
type Store interface {
	Create(r Record) error
	Get(id string) (Record, error)
	List(filter ListFilter) ([]Record, error)
	UpdateNarrative(id string, narrative Narrative, now time.Time) (Record, error)
	SetSupersededBy(id, successorID string) (Record, error)
}

// ListFilter is the read-side filter passed to Store.List.
type ListFilter struct {
	Scenario     string
	Kind         RecordKind
	BacklogRef   string
	IncludeStubs bool
	Limit        int
	Offset       int
}

// Narrative is the payload used to fill a stub record once.
type Narrative struct {
	Trigger      string
	Approach     string
	RuledOut     []string
	Commit       string
	FilesChanged []string
	Outcome      Outcome
}

// ErrNotFound is returned when a record id does not exist.
var ErrNotFound = errors.New("record not found")

// FileStore persists records as JSON under
//
//	<dataRoot>/records/<scenario>/<kind>/<id>.json
type FileStore struct {
	dataRoot string
	mu       sync.Mutex
}

// NewFileStore constructs a FileStore rooted at the given runtime-home data dir.
func NewFileStore(dataRoot string) *FileStore {
	return &FileStore{dataRoot: dataRoot}
}

func (s *FileStore) recordsDir() string {
	return filepath.Join(s.dataRoot, "records")
}

func (s *FileStore) kindDir(scenario string, kind RecordKind) string {
	return filepath.Join(s.recordsDir(), scenario, string(kind))
}

func (s *FileStore) recordPath(scenario string, kind RecordKind, id string) string {
	return filepath.Join(s.kindDir(scenario, kind), id+".json")
}

// findPath scans the records directory for a record by id (without
// requiring the caller to know scenario/kind up front).
func (s *FileStore) findPath(id string) (string, error) {
	if !validID(id) {
		return "", fmt.Errorf("invalid record id %q", id)
	}
	root := s.recordsDir()
	var match string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == id+".json" {
			match = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if match == "" {
		return "", ErrNotFound
	}
	return match, nil
}

// validID enforces a conservative allowlist on the id so it's safe as a path
// component. We use idgen.Generate() which returns 16 hex chars; allow that
// plus an optional short prefix.
func validID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '-' || c == '_':
		default:
			return false
		}
	}
	return true
}

func (s *FileStore) Create(r Record) error {
	if !validID(r.ID) {
		return fmt.Errorf("invalid record id %q", r.ID)
	}
	if r.Scenario == "" {
		return fmt.Errorf("record scenario is required")
	}
	if _, err := ParseKind(string(r.Kind)); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := s.kindDir(r.Scenario, r.Kind)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create records dir: %w", err)
	}
	path := s.recordPath(r.Scenario, r.Kind, r.ID)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("record %s already exists", r.ID)
	}
	return writeRecord(path, &r)
}

func (s *FileStore) Get(id string) (Record, error) {
	path, err := s.findPath(id)
	if err != nil {
		return Record{}, err
	}
	return readRecord(path)
}

func (s *FileStore) List(filter ListFilter) ([]Record, error) {
	root := s.recordsDir()
	out := []Record{}
	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, werr error) error {
		if werr != nil {
			if os.IsNotExist(werr) {
				return nil
			}
			return werr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		r, err := readRecord(path)
		if err != nil {
			return nil // skip unreadable
		}
		if !filter.matches(r) {
			return nil
		}
		out = append(out, r)
		return nil
	})
	if walkErr != nil && !os.IsNotExist(walkErr) {
		return nil, walkErr
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if filter.Offset > 0 {
		if filter.Offset >= len(out) {
			return []Record{}, nil
		}
		out = out[filter.Offset:]
	}
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

func (f ListFilter) matches(r Record) bool {
	if f.Scenario != "" && r.Scenario != f.Scenario {
		return false
	}
	if f.Kind != "" && r.Kind != f.Kind {
		return false
	}
	if f.BacklogRef != "" && r.BacklogRef != f.BacklogRef {
		return false
	}
	if r.Stub && !f.IncludeStubs {
		return false
	}
	return true
}

func (s *FileStore) UpdateNarrative(id string, n Narrative, now time.Time) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.findPath(id)
	if err != nil {
		return Record{}, err
	}
	r, err := readRecord(path)
	if err != nil {
		return Record{}, err
	}
	if !r.Stub {
		return Record{}, ErrStubLocked
	}
	r.Trigger = strings.TrimSpace(n.Trigger)
	r.Approach = strings.TrimSpace(n.Approach)
	r.RuledOut = trimAll(n.RuledOut)
	if n.Commit != "" {
		r.Commit = n.Commit
	}
	if len(n.FilesChanged) > 0 {
		r.FilesChanged = trimAll(n.FilesChanged)
	}
	if n.Outcome != "" {
		r.Outcome = n.Outcome
	}
	if !r.hasNarrative() {
		return Record{}, fmt.Errorf("narrative must include at least one of trigger, approach, or ruled_out")
	}
	r.Stub = false
	r.NarrativeAt = now
	if err := writeRecord(path, &r); err != nil {
		return Record{}, err
	}
	return r, nil
}

func (s *FileStore) SetSupersededBy(id, successorID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.findPath(id)
	if err != nil {
		return Record{}, err
	}
	r, err := readRecord(path)
	if err != nil {
		return Record{}, err
	}
	if r.SupersededBy != "" && r.SupersededBy != successorID {
		return Record{}, ErrAlreadySuperseded
	}
	r.SupersededBy = successorID
	if err := writeRecord(path, &r); err != nil {
		return Record{}, err
	}
	return r, nil
}

func readRecord(path string) (Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Record{}, ErrNotFound
		}
		return Record{}, fmt.Errorf("read record: %w", err)
	}
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return Record{}, fmt.Errorf("unmarshal record: %w", err)
	}
	return r, nil
}

func writeRecord(path string, r *Record) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write record: %w", err)
	}
	return os.Rename(tmp, path)
}

func trimAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
