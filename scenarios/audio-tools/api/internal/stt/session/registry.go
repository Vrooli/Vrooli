package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

const defaultSessionRecoveryTTL = 15 * time.Minute

// Registry keeps bounded session ledgers alive across transport reconnects.
// It is an explicit seam: production may replace it with durable storage
// without changing either transport's replay contract.
type Registry struct {
	mu             sync.Mutex
	maxBytes       int
	directory      string
	roots          *filerouting.RoutedRoots
	sessions       map[string]*Ledger
	locks          map[string]*registryLock
	persisted      map[string]PersistedState
	logOps         map[string]int
	spoolDirs      map[string]string
	expiry         time.Duration
	timers         map[string]*time.Timer
	generation     map[string]uint64
	nextGeneration uint64
}

type registryLock struct {
	mu   sync.Mutex
	refs int
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
	return newRegistry(maxBytes, defaultSessionRecoveryTTL)
}

func newRegistry(maxBytes int, recoveryTTL time.Duration) *Registry {
	if maxBytes <= 0 {
		maxBytes = 64 << 20
	}
	if recoveryTTL <= 0 {
		recoveryTTL = defaultSessionRecoveryTTL
	}
	return &Registry{
		maxBytes: maxBytes, sessions: make(map[string]*Ledger), locks: make(map[string]*registryLock),
		persisted: make(map[string]PersistedState), logOps: make(map[string]int), spoolDirs: make(map[string]string), expiry: recoveryTTL,
		timers: make(map[string]*time.Timer), generation: make(map[string]uint64),
	}
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

func (r *Registry) touch(sessionID string) {
	r.mu.Lock()
	if timer := r.timers[sessionID]; timer != nil {
		timer.Stop()
	}
	r.nextGeneration++
	generation := r.nextGeneration
	r.generation[sessionID] = generation
	r.timers[sessionID] = time.AfterFunc(r.expiry, func() {
		r.expire(sessionID, generation)
	})
	r.mu.Unlock()
}

func (r *Registry) expire(sessionID string, generation uint64) {
	// Expiry is the bounded recovery window for a client that disappeared
	// before terminal delivery. RemoveContext also clears the persisted replay
	// spool, so abandoned sessions cannot accumulate on disk indefinitely.
	_ = r.removeContext(context.Background(), sessionID, generation)
}

func (r *Registry) OpenContext(ctx context.Context, sessionID, resumeToken string) (*Ledger, bool, error) {
	if sessionID == "" || resumeToken == "" {
		return nil, false, fmt.Errorf("stt session: session id and resume token are required")
	}
	if filepath.Base(sessionID) != sessionID {
		return nil, false, fmt.Errorf("stt session: invalid session id")
	}
	release := r.acquireSessionLock(sessionID)
	defer release()
	r.mu.Lock()
	if ledger, ok := r.sessions[sessionID]; ok {
		r.mu.Unlock()
		if _, err := ledger.Resume(resumeToken); err != nil {
			return nil, false, err
		}
		r.touch(sessionID)
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
			r.touch(sessionID)
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
	r.touch(sessionID)
	if err := r.persistLocked(ctx, ledger); err != nil {
		return nil, false, err
	}
	return ledger, false, nil
}

func (r *Registry) Persist(ledger *Ledger) error {
	return r.PersistContext(context.Background(), ledger)
}

func (r *Registry) PersistContext(ctx context.Context, ledger *Ledger) error {
	if r == nil || ledger == nil {
		return nil
	}
	release := r.acquireSessionLock(ledger.SessionID())
	defer release()
	var err error
	if r.directory != "" || r.roots != nil {
		err = r.persistLocked(ctx, ledger)
	}
	if err == nil {
		r.touch(ledger.SessionID())
	}
	return err
}

// RemoveContext releases a session from the live registry and removes its
// replay spool. Callers use it after terminal delivery; the registry also
// uses it when the bounded reconnect window expires. A bounded per-session
// audio ledger is not enough for unlimited use if committed segment metadata
// remains reachable forever. The session lock is retained as a small
// synchronization sentinel so a late opener cannot race a cleanup into a
// second live ledger.
func (r *Registry) RemoveContext(ctx context.Context, sessionID string) error {
	return r.removeContext(ctx, sessionID, 0)
}

func (r *Registry) removeContext(ctx context.Context, sessionID string, expectedGeneration uint64) error {
	if r == nil || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	if filepath.Base(sessionID) != sessionID {
		return fmt.Errorf("stt session: invalid session id")
	}
	release := r.acquireSessionLock(sessionID)
	defer release()

	r.mu.Lock()
	if expectedGeneration != 0 && r.generation[sessionID] != expectedGeneration {
		r.mu.Unlock()
		return nil
	}
	if timer := r.timers[sessionID]; timer != nil {
		timer.Stop()
	}
	delete(r.timers, sessionID)
	delete(r.generation, sessionID)
	delete(r.sessions, sessionID)
	delete(r.persisted, sessionID)
	delete(r.logOps, sessionID)
	spoolDirectory := r.spoolDirs[sessionID]
	delete(r.spoolDirs, sessionID)
	r.mu.Unlock()
	if r.directory == "" && r.roots == nil {
		return nil
	}

	if spoolDirectory == "" {
		var err error
		spoolDirectory, err = r.directoryFor(ctx)
		if err != nil {
			return err
		}
	}
	for _, suffix := range []string{".json", ".jsonl"} {
		if err := os.Remove(filepath.Join(spoolDirectory, sessionID+suffix)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove session spool %q: %w", sessionID, err)
		}
	}
	if r.roots != nil {
		r.roots.RecordWrite(ctx)
	}
	return nil
}

// Remove is the context-free convenience form used by in-process handlers.
func (r *Registry) Remove(sessionID string) error {
	return r.RemoveContext(context.Background(), sessionID)
}

func (r *Registry) acquireSessionLock(sessionID string) func() {
	r.mu.Lock()
	lock, ok := r.locks[sessionID]
	if !ok {
		lock = &registryLock{}
		r.locks[sessionID] = lock
	}
	lock.refs++
	r.mu.Unlock()
	lock.mu.Lock()
	return func() {
		lock.mu.Unlock()
		r.mu.Lock()
		lock.refs--
		if lock.refs == 0 && r.locks[sessionID] == lock {
			delete(r.locks, sessionID)
		}
		r.mu.Unlock()
	}
}

func (r *Registry) persistLocked(ctx context.Context, ledger *Ledger) error {
	if r.directory == "" && r.roots == nil {
		return nil
	}
	directory, err := r.directoryFor(ctx)
	if err != nil {
		return err
	}
	persisted := ledger.PersistedState()
	r.mu.Lock()
	previous, ok := r.persisted[persisted.SessionID]
	logOps := r.logOps[persisted.SessionID]
	r.mu.Unlock()
	if ok {
		deltas := persistenceDeltas(previous, persisted)
		if err := r.appendDeltas(ctx, persisted.SessionID, deltas); err != nil {
			return err
		}
		r.mu.Lock()
		r.persisted[persisted.SessionID] = persisted
		r.logOps[persisted.SessionID] += len(deltas)
		logOps = r.logOps[persisted.SessionID]
		r.mu.Unlock()
		if logOps < 128 {
			return nil
		}
	} else {
		r.mu.Lock()
		r.persisted[persisted.SessionID] = persisted
		r.mu.Unlock()
	}
	r.mu.Lock()
	r.spoolDirs[persisted.SessionID] = directory
	r.mu.Unlock()
	state, err := json.Marshal(persisted)
	if err != nil {
		return fmt.Errorf("marshal session ledger: %w", err)
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
	r.mu.Lock()
	r.logOps[persisted.SessionID] = 0
	r.mu.Unlock()
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
	r.mu.Lock()
	r.spoolDirs[sessionID] = directory
	r.mu.Unlock()
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
