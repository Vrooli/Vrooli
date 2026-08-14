package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

// Registry keeps bounded session ledgers alive across transport reconnects.
// It is an explicit seam: production may replace it with durable storage
// without changing either transport's replay contract.
type Registry struct {
	mu        sync.Mutex
	maxBytes  int
	directory string
	roots     *filerouting.RoutedRoots
	sessions  map[string]*Ledger
	locks     map[string]*sync.Mutex
	persisted map[string]PersistedState
	logOps    map[string]int
}

type persistenceDelta struct {
	Kind     string          `json:"kind"`
	Snapshot *PersistedState `json:"snapshot,omitempty"`
	Chunk    *Chunk          `json:"chunk,omitempty"`
	Sequence int64           `json:"sequence,omitempty"`
	Segment  *Segment        `json:"segment,omitempty"`
	Terminal TerminalReason  `json:"terminal,omitempty"`
}

func persistenceDeltas(previous, next PersistedState) []persistenceDelta {
	var deltas []persistenceDelta
	previousChunks := make(map[uint64]struct{}, len(previous.Received))
	for _, chunk := range previous.Received {
		previousChunks[chunk.Sequence] = struct{}{}
	}
	for index := range next.Received {
		chunk := next.Received[index]
		if _, ok := previousChunks[chunk.Sequence]; !ok {
			copyChunk := chunk
			copyChunk.Audio = append([]byte(nil), chunk.Audio...)
			deltas = append(deltas, persistenceDelta{Kind: "receive", Chunk: &copyChunk})
		}
	}
	if next.ProcessedSequence > previous.ProcessedSequence {
		deltas = append(deltas, persistenceDelta{Kind: "ack", Sequence: next.ProcessedSequence})
	}
	previousSegments := make(map[string]struct{}, len(previous.Committed))
	for _, segment := range previous.Committed {
		previousSegments[segment.ID] = struct{}{}
	}
	for index := range next.Committed {
		segment := next.Committed[index]
		if _, ok := previousSegments[segment.ID]; !ok {
			copySegment := segment
			deltas = append(deltas, persistenceDelta{Kind: "commit", Segment: &copySegment})
		}
	}
	if next.TerminalReason != previous.TerminalReason {
		deltas = append(deltas, persistenceDelta{Kind: "terminal", Terminal: next.TerminalReason})
	}
	return deltas
}

func applyPersistenceDelta(state *PersistedState, delta persistenceDelta) *PersistedState {
	if delta.Kind == "snapshot" && delta.Snapshot != nil {
		copy := *delta.Snapshot
		return &copy
	}
	if state == nil {
		return nil
	}
	switch delta.Kind {
	case "receive":
		if delta.Chunk != nil {
			state.Received = append(state.Received, *delta.Chunk)
			if int64(delta.Chunk.Sequence) > state.ReceivedSequence {
				state.ReceivedSequence = int64(delta.Chunk.Sequence)
			}
		}
	case "ack":
		state.ProcessedSequence = delta.Sequence
		retained := state.Received[:0]
		for _, chunk := range state.Received {
			if int64(chunk.Sequence) > delta.Sequence {
				retained = append(retained, chunk)
			}
		}
		state.Received = retained
	case "commit":
		if delta.Segment != nil {
			state.Committed = append(state.Committed, *delta.Segment)
		}
	case "terminal":
		state.TerminalReason = delta.Terminal
	}
	return state
}

// NewRoutedDiskRegistry persists ledgers beneath the per-request data root.
// A Test Genie lease changes that root without changing the stream protocol.
func NewRoutedDiskRegistry(roots *filerouting.RoutedRoots, maxBytes int) (*Registry, error) {
	if roots == nil {
		return nil, fmt.Errorf("stt session: routed roots are required")
	}
	// RoutedRoots.Pick(ctx, class) is the file-isolation seam. The startup
	// directory is resolved once here; request-scoped reloads resolve it again
	// through OpenContext below.
	directory, err := roots.Pick(context.Background(), storage.ClassData)
	if err != nil {
		return nil, fmt.Errorf("resolve session persistence directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(directory, "stt-session-spool"), 0o700); err != nil {
		return nil, fmt.Errorf("create session persistence directory: %w", err)
	}
	r := NewRegistry(maxBytes)
	r.roots = roots
	return r, nil
}

func NewRegistry(maxBytes int) *Registry {
	if maxBytes <= 0 {
		maxBytes = 64 << 20
	}
	return &Registry{maxBytes: maxBytes, sessions: make(map[string]*Ledger), locks: make(map[string]*sync.Mutex), persisted: make(map[string]PersistedState), logOps: make(map[string]int)}
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
	return r.OpenContext(context.Background(), sessionID, resumeToken)
}

func (r *Registry) OpenContext(ctx context.Context, sessionID, resumeToken string) (*Ledger, bool, error) {
	if sessionID == "" || resumeToken == "" {
		return nil, false, fmt.Errorf("stt session: session id and resume token are required")
	}
	if filepath.Base(sessionID) != sessionID {
		return nil, false, fmt.Errorf("stt session: invalid session id")
	}
	lock := r.sessionLock(sessionID)
	lock.Lock()
	defer lock.Unlock()
	r.mu.Lock()
	if ledger, ok := r.sessions[sessionID]; ok {
		r.mu.Unlock()
		if _, err := ledger.Resume(resumeToken); err != nil {
			return nil, false, err
		}
		return ledger, true, nil
	}
	r.mu.Unlock()
	if r.directory != "" || r.roots != nil {
		if state, err := r.loadLocked(ctx, sessionID); err != nil {
			return nil, false, err
		} else if state != nil {
			ledger, err := Restore(*state)
			if err != nil {
				return nil, false, err
			}
			if _, err := ledger.Resume(resumeToken); err != nil {
				return nil, false, err
			}
			r.mu.Lock()
			r.sessions[sessionID] = ledger
			r.persisted[sessionID] = *state
			r.mu.Unlock()
			return ledger, true, nil
		}
	}
	ledger, err := New(Config{SessionID: sessionID, ResumeToken: resumeToken, MaxBytes: r.maxBytes})
	if err != nil {
		return nil, false, err
	}
	r.mu.Lock()
	r.sessions[sessionID] = ledger
	r.mu.Unlock()
	if err := r.persistLocked(ctx, ledger); err != nil {
		return nil, false, err
	}
	return ledger, false, nil
}

func (r *Registry) Persist(ledger *Ledger) error {
	return r.PersistContext(context.Background(), ledger)
}

func (r *Registry) PersistContext(ctx context.Context, ledger *Ledger) error {
	if r == nil || (r.directory == "" && r.roots == nil) || ledger == nil {
		return nil
	}
	lock := r.sessionLock(ledger.SessionID())
	lock.Lock()
	defer lock.Unlock()
	return r.persistLocked(ctx, ledger)
}

func (r *Registry) sessionLock(sessionID string) *sync.Mutex {
	r.mu.Lock()
	defer r.mu.Unlock()
	if lock, ok := r.locks[sessionID]; ok {
		return lock
	}
	lock := &sync.Mutex{}
	r.locks[sessionID] = lock
	return lock
}

func (r *Registry) persistLocked(ctx context.Context, ledger *Ledger) error {
	if r.directory == "" && r.roots == nil {
		return nil
	}
	persisted := ledger.PersistedState()
	if previous, ok := r.persisted[persisted.SessionID]; ok {
		deltas := persistenceDeltas(previous, persisted)
		if err := r.appendDeltas(ctx, persisted.SessionID, deltas); err != nil {
			return err
		}
		r.persisted[persisted.SessionID] = persisted
		r.logOps[persisted.SessionID] += len(deltas)
		if r.logOps[persisted.SessionID] < 128 {
			return nil
		}
	} else {
		r.persisted[persisted.SessionID] = persisted
	}
	state, err := json.Marshal(persisted)
	if err != nil {
		return fmt.Errorf("marshal session ledger: %w", err)
	}
	directory, err := r.directoryFor(ctx)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create session persistence directory: %w", err)
	}
	path := filepath.Join(directory, persisted.SessionID+".json")
	temporary, err := os.CreateTemp(directory, ".session-*")
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
	_ = os.Remove(filepath.Join(directory, persisted.SessionID+".jsonl"))
	r.logOps[persisted.SessionID] = 0
	if r.roots != nil {
		r.roots.RecordWrite(ctx)
	}
	return nil
}

func (r *Registry) appendDeltas(ctx context.Context, sessionID string, deltas []persistenceDelta) error {
	if len(deltas) == 0 {
		return nil
	}
	directory, err := r.directoryFor(ctx)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create session persistence directory: %w", err)
	}
	file, err := os.OpenFile(filepath.Join(directory, sessionID+".jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open session ledger journal: %w", err)
	}
	defer file.Close()
	if err := file.Chmod(0o600); err != nil {
		return err
	}
	for _, delta := range deltas {
		encoded, err := json.Marshal(delta)
		if err != nil {
			return fmt.Errorf("marshal session ledger journal: %w", err)
		}
		if _, err := file.Write(append(encoded, '\n')); err != nil {
			return fmt.Errorf("append session ledger journal: %w", err)
		}
	}
	if r.roots != nil {
		r.roots.RecordWrite(ctx)
	}
	return nil
}

func (r *Registry) loadLocked(ctx context.Context, sessionID string) (*PersistedState, error) {
	directory, err := r.directoryFor(ctx)
	if err != nil {
		return nil, err
	}
	var state *PersistedState
	contents, err := os.ReadFile(filepath.Join(directory, sessionID+".json"))
	if err == nil {
		var decoded PersistedState
		if err := json.Unmarshal(contents, &decoded); err != nil {
			return nil, fmt.Errorf("decode session ledger: %w", err)
		}
		state = &decoded
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read session ledger: %w", err)
	}
	logContents, logErr := os.ReadFile(filepath.Join(directory, sessionID+".jsonl"))
	if logErr != nil && !os.IsNotExist(logErr) {
		return nil, fmt.Errorf("read session ledger journal: %w", logErr)
	}
	for _, line := range bytes.Split(logContents, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var delta persistenceDelta
		if err := json.Unmarshal(line, &delta); err != nil {
			return nil, fmt.Errorf("decode session ledger journal: %w", err)
		}
		state = applyPersistenceDelta(state, delta)
	}
	if state == nil {
		return nil, nil
	}
	return state, nil
}

func (r *Registry) directoryFor(ctx context.Context) (string, error) {
	if r.roots != nil {
		root, err := r.roots.Pick(ctx, storage.ClassData)
		if err != nil {
			return "", fmt.Errorf("resolve session persistence root: %w", err)
		}
		return filepath.Join(root, "stt-session-spool"), nil
	}
	return r.directory, nil
}
