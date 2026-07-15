package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Registry keeps bounded session ledgers alive across transport reconnects.
// It is an explicit seam: production may replace it with durable storage
// without changing either transport's replay contract.
type Registry struct {
	mu        sync.Mutex
	maxBytes  int
	directory string
	sessions  map[string]*Ledger
}

func NewRegistry(maxBytes int) *Registry {
	if maxBytes <= 0 {
		maxBytes = 64 << 20
	}
	return &Registry{maxBytes: maxBytes, sessions: make(map[string]*Ledger)}
}

// NewDiskRegistry persists each bounded ledger with owner-only permissions so
// a process restart can offer replay rather than silently starting empty.
func NewDiskRegistry(directory string, maxBytes int) (*Registry, error) {
	if directory == "" {
		return nil, fmt.Errorf("stt session: persistence directory is required")
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create session persistence directory: %w", err)
	}
	r := NewRegistry(maxBytes)
	r.directory = directory
	return r, nil
}

// Open returns the same ledger for a valid reconnect. Session identity and
// resume token are mandatory; accepting either implicitly would turn a retry
// into a fresh session and make silent loss possible.
func (r *Registry) Open(sessionID, resumeToken string) (*Ledger, bool, error) {
	if sessionID == "" || resumeToken == "" {
		return nil, false, fmt.Errorf("stt session: session id and resume token are required")
	}
	if filepath.Base(sessionID) != sessionID {
		return nil, false, fmt.Errorf("stt session: invalid session id")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if ledger, ok := r.sessions[sessionID]; ok {
		if _, err := ledger.Resume(resumeToken); err != nil {
			return nil, false, err
		}
		return ledger, true, nil
	}
	if r.directory != "" {
		if state, err := r.loadLocked(sessionID); err != nil {
			return nil, false, err
		} else if state != nil {
			ledger, err := Restore(*state)
			if err != nil {
				return nil, false, err
			}
			if _, err := ledger.Resume(resumeToken); err != nil {
				return nil, false, err
			}
			r.sessions[sessionID] = ledger
			return ledger, true, nil
		}
	}
	ledger, err := New(Config{SessionID: sessionID, ResumeToken: resumeToken, MaxBytes: r.maxBytes})
	if err != nil {
		return nil, false, err
	}
	r.sessions[sessionID] = ledger
	if err := r.persistLocked(ledger); err != nil {
		return nil, false, err
	}
	return ledger, false, nil
}

func (r *Registry) Persist(ledger *Ledger) error {
	if r == nil || r.directory == "" || ledger == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.persistLocked(ledger)
}

func (r *Registry) persistLocked(ledger *Ledger) error {
	if r.directory == "" {
		return nil
	}
	state, err := json.Marshal(ledger.PersistedState())
	if err != nil {
		return fmt.Errorf("marshal session ledger: %w", err)
	}
	path := r.path(ledger.PersistedState().SessionID)
	temporary, err := os.CreateTemp(r.directory, ".session-*")
	if err != nil {
		return fmt.Errorf("create session ledger temporary file: %w", err)
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(state); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("atomically persist session ledger: %w", err)
	}
	return nil
}

func (r *Registry) loadLocked(sessionID string) (*PersistedState, error) {
	contents, err := os.ReadFile(r.path(sessionID))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read session ledger: %w", err)
	}
	var state PersistedState
	if err := json.Unmarshal(contents, &state); err != nil {
		return nil, fmt.Errorf("decode session ledger: %w", err)
	}
	return &state, nil
}

func (r *Registry) path(sessionID string) string {
	return filepath.Join(r.directory, sessionID+".json")
}
