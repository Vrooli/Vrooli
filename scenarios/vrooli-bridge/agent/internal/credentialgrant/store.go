// Package credentialgrant is the node-side consent boundary for fleet
// credentials. It stores grant metadata only; credential values are handled by
// the authority or by an ephemeral job buffer and never by this package.
package credentialgrant

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	ClassInfrastructure      = "infrastructure"
	ClassPerInstallGenerated = "per_install_generated"
	ClassUserPrompt          = "user_prompt"
	ClassRemoteFetch         = "remote_fetch"

	RetentionDurable   = "durable"
	RetentionEphemeral = "ephemeral"
)

type Grant struct {
	ID         string `json:"id"`
	NodeID     string `json:"node_id"`
	LogicalID  string `json:"logical_id"`
	Field      string `json:"field"`
	Class      string `json:"class"`
	Retention  string `json:"retention"`
	Generation int64  `json:"generation"`
	Revoked    bool   `json:"revoked"`
}

// Store is intentionally narrower than a general repository. The agent only
// needs to answer whether a signed push is locally consented and to maintain
// the grant metadata used for startup revoke cleanup.
type Store interface {
	Lookup(logicalID, field string) (Grant, bool)
	List() []Grant
	Put(Grant) error
	Revoke(logicalID, field string) error
}

type MemoryStore struct {
	mu     sync.RWMutex
	grants map[string]Grant
}

func NewMemoryStore(grants ...Grant) *MemoryStore {
	s := &MemoryStore{grants: make(map[string]Grant, len(grants))}
	for _, grant := range grants {
		s.grants[key(grant.LogicalID, grant.Field)] = grant
	}
	return s
}

func (s *MemoryStore) Lookup(logicalID, field string) (Grant, bool) {
	s.mu.RLock()
	grant, ok := s.grants[key(logicalID, field)]
	s.mu.RUnlock()
	if !ok || grant.Revoked || grant.LogicalID == "" || grant.Field == "" {
		return Grant{}, false
	}
	return grant, true
}

func (s *MemoryStore) List() []Grant {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Grant, 0, len(s.grants))
	for _, grant := range s.grants {
		out = append(out, grant)
	}
	return out
}

func (s *MemoryStore) Put(grant Grant) error {
	if err := validate(grant); err != nil {
		return err
	}
	s.mu.Lock()
	s.grants[key(grant.LogicalID, grant.Field)] = grant
	s.mu.Unlock()
	return nil
}

func (s *MemoryStore) Revoke(logicalID, field string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	grant, ok := s.grants[key(logicalID, field)]
	if !ok {
		return nil
	}
	grant.Revoked = true
	s.grants[key(logicalID, field)] = grant
	return nil
}

// FileStore persists only grant metadata. Writes are owner-only and atomic so
// a restart cannot accidentally widen local consent because of a partial file.
type FileStore struct {
	path string
	mem  *MemoryStore
}

func LoadFile(path string) (*FileStore, error) {
	if path == "" {
		return nil, errors.New("credential grants: empty path")
	}
	s := &FileStore{path: path, mem: NewMemoryStore()}
	raw, err := os.ReadFile(path) // #nosec G304 -- path is the agent state file.
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("credential grants: read: %w", err)
	}
	var grants []Grant
	if err := json.Unmarshal(raw, &grants); err != nil {
		return nil, fmt.Errorf("credential grants: decode: %w", err)
	}
	for _, grant := range grants {
		if err := s.mem.Put(grant); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *FileStore) Lookup(logicalID, field string) (Grant, bool) {
	return s.mem.Lookup(logicalID, field)
}
func (s *FileStore) List() []Grant { return s.mem.List() }

func (s *FileStore) Put(grant Grant) error {
	if err := s.mem.Put(grant); err != nil {
		return err
	}
	return s.save()
}

func (s *FileStore) Revoke(logicalID, field string) error {
	if err := s.mem.Revoke(logicalID, field); err != nil {
		return err
	}
	return s.save()
}

func (s *FileStore) save() error {
	grants := s.mem.List()
	data, err := json.MarshalIndent(grants, "", "  ")
	if err != nil {
		return fmt.Errorf("credential grants: encode: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("credential grants: create directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".credential-grants-*")
	if err != nil {
		return fmt.Errorf("credential grants: create temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(data)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("credential grants: write: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("credential grants: install: %w", err)
	}
	return nil
}

func validate(grant Grant) error {
	if grant.LogicalID == "" || grant.Field == "" || grant.Class == "" {
		return errors.New("credential grants: logical_id and field are required")
	}
	switch grant.Class {
	case ClassInfrastructure, ClassUserPrompt, ClassRemoteFetch:
	case ClassPerInstallGenerated:
		return errors.New("credential grants: per-install-generated credentials cannot be distributed")
	default:
		return fmt.Errorf("credential grants: invalid class %q", grant.Class)
	}
	if grant.Retention != RetentionDurable && grant.Retention != RetentionEphemeral {
		return fmt.Errorf("credential grants: invalid retention %q", grant.Retention)
	}
	if grant.Retention == RetentionDurable && grant.Class == ClassInfrastructure {
		return errors.New("credential grants: infrastructure credentials cannot be durable")
	}
	if grant.Generation <= 0 {
		return errors.New("credential grants: generation must be positive")
	}
	return nil
}

func key(logicalID, field string) string { return logicalID + "\x00" + field }
