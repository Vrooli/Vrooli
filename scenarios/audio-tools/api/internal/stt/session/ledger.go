// Package session owns the replay-safe STT session contract. Transports may
// disconnect, but only this ledger decides acknowledged coverage and terminal
// state. Audio is retained until the processed cursor covers it.
package session

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrGap                = errors.New("stt session: non-monotonic chunk sequence")
	ErrChunkConflict      = errors.New("stt session: duplicate chunk conflicts with retained digest")
	ErrResourceExhausted  = errors.New("stt session: retained audio quota exhausted")
	ErrIncompleteCoverage = errors.New("stt session: cannot complete before all audio is processed")
	ErrInvalidResumeToken = errors.New("stt session: invalid resume token")
	ErrUnknownChunk       = errors.New("stt session: processed acknowledgement exceeds received coverage")
	ErrSegmentConflict    = errors.New("stt session: committed segment identity conflicts")
	ErrTerminal           = errors.New("stt session: terminal")
)

type TerminalReason string

const (
	TerminalNone               TerminalReason = ""
	TerminalCompleted          TerminalReason = "completed"
	TerminalIncompleteCoverage TerminalReason = "incomplete_coverage"
	TerminalResourceExhausted  TerminalReason = "resource_exhausted"
)

type ReceiveResult string

const (
	ReceivedNew       ReceiveResult = "received_new"
	ReceivedDuplicate ReceiveResult = "received_duplicate"
)

type Config struct {
	SessionID   string
	ResumeToken string
	MaxBytes    int
}

type Chunk struct {
	Sequence    uint64
	StartSample int64
	EndSample   int64
	Audio       []byte
}

type Segment struct {
	ID          string
	Text        string
	StartSample int64
	EndSample   int64
}

type Snapshot struct {
	SessionID         string
	ReceivedSequence  int64
	ProcessedSequence int64
	TerminalReason    TerminalReason
	Replay            []Chunk
	Committed         []Segment
}

// PersistedState is the bounded on-disk form of a ledger. It deliberately
// excludes transcript diagnostics beyond committed segment IDs/text; the file
// exists solely to finish or replay an interrupted turn and must live in a
// service-private 0700 directory.
type PersistedState struct {
	SessionID         string
	ResumeToken       string
	MaxBytes          int
	Received          []Chunk
	Committed         []Segment
	ReceivedSequence  int64
	ProcessedSequence int64
	TerminalReason    TerminalReason
}

type retainedChunk struct {
	Chunk
	digest [sha256.Size]byte
}

// Ledger is deliberately transport-free. It is safe for a receive pump,
// provider pump, and reconnect handler to use concurrently.
type Ledger struct {
	mu                sync.Mutex
	cfg               Config
	received          map[uint64]retainedChunk
	committed         map[string]Segment
	receivedSequence  int64
	processedSequence int64
	retainedBytes     int
	terminal          TerminalReason
}

func New(cfg Config) (*Ledger, error) {
	if cfg.SessionID == "" || cfg.ResumeToken == "" || cfg.MaxBytes <= 0 {
		return nil, fmt.Errorf("stt session: session id, resume token, and positive max bytes are required")
	}
	return &Ledger{cfg: cfg, received: make(map[uint64]retainedChunk), committed: make(map[string]Segment), receivedSequence: -1, processedSequence: -1}, nil
}

func (l *Ledger) Receive(chunk Chunk) (ReceiveResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.terminal != TerminalNone {
		return "", ErrTerminal
	}
	if chunk.EndSample < chunk.StartSample {
		return "", fmt.Errorf("stt session: invalid sample range")
	}
	digest := sha256.Sum256(chunk.Audio)
	if retained, ok := l.received[chunk.Sequence]; ok {
		if retained.digest != digest || retained.StartSample != chunk.StartSample || retained.EndSample != chunk.EndSample {
			return "", ErrChunkConflict
		}
		return ReceivedDuplicate, nil
	}
	if chunk.Sequence != uint64(l.receivedSequence+1) {
		return "", ErrGap
	}
	if l.retainedBytes+len(chunk.Audio) > l.cfg.MaxBytes {
		l.terminal = TerminalResourceExhausted
		return "", ErrResourceExhausted
	}
	copyAudio := append([]byte(nil), chunk.Audio...)
	chunk.Audio = copyAudio
	l.received[chunk.Sequence] = retainedChunk{Chunk: chunk, digest: digest}
	l.receivedSequence = int64(chunk.Sequence)
	l.retainedBytes += len(chunk.Audio)
	return ReceivedNew, nil
}

func (l *Ledger) AcknowledgeProcessed(sequence uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if sequence > uint64(l.receivedSequence) {
		return ErrUnknownChunk
	}
	if int64(sequence) <= l.processedSequence {
		return nil
	}
	for next := l.processedSequence + 1; next <= int64(sequence); next++ {
		chunk, ok := l.received[uint64(next)]
		if !ok {
			return ErrUnknownChunk
		}
		delete(l.received, uint64(next))
		l.retainedBytes -= len(chunk.Audio)
	}
	l.processedSequence = int64(sequence)
	return nil
}

func (l *Ledger) Commit(segment Segment) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if segment.ID == "" {
		return false, fmt.Errorf("stt session: segment id is required")
	}
	if existing, ok := l.committed[segment.ID]; ok {
		if existing != segment {
			return false, ErrSegmentConflict
		}
		return false, nil
	}
	l.committed[segment.ID] = segment
	return true, nil
}

func (l *Ledger) Complete() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.terminal != TerminalNone {
		return ErrTerminal
	}
	if l.processedSequence != l.receivedSequence {
		l.terminal = TerminalIncompleteCoverage
		return ErrIncompleteCoverage
	}
	l.terminal = TerminalCompleted
	return nil
}

// Fail names an explicit terminal outcome while retaining replayable chunks.
// It is used for malformed input, disconnect, and provider failures; callers
// must never substitute a generic done for this outcome.
func (l *Ledger) Fail(reason TerminalReason) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.terminal == TerminalNone {
		l.terminal = reason
	}
}

func (l *Ledger) Resume(token string) (Snapshot, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if token != l.cfg.ResumeToken {
		return Snapshot{}, ErrInvalidResumeToken
	}
	return l.snapshotLocked(), nil
}

func (l *Ledger) Snapshot() Snapshot {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.snapshotLocked()
}

func (l *Ledger) PersistedState() PersistedState {
	l.mu.Lock()
	defer l.mu.Unlock()
	state := PersistedState{
		SessionID: l.cfg.SessionID, ResumeToken: l.cfg.ResumeToken, MaxBytes: l.cfg.MaxBytes,
		ReceivedSequence: l.receivedSequence, ProcessedSequence: l.processedSequence, TerminalReason: l.terminal,
	}
	for sequence := l.processedSequence + 1; sequence <= l.receivedSequence; sequence++ {
		if chunk, ok := l.received[uint64(sequence)]; ok {
			copy := chunk.Chunk
			copy.Audio = append([]byte(nil), copy.Audio...)
			state.Received = append(state.Received, copy)
		}
	}
	for _, segment := range l.committed {
		state.Committed = append(state.Committed, segment)
	}
	return state
}

func Restore(state PersistedState) (*Ledger, error) {
	ledger, err := New(Config{SessionID: state.SessionID, ResumeToken: state.ResumeToken, MaxBytes: state.MaxBytes})
	if err != nil {
		return nil, err
	}
	ledger.mu.Lock()
	ledger.receivedSequence = state.ProcessedSequence
	ledger.processedSequence = state.ProcessedSequence
	ledger.mu.Unlock()
	for _, chunk := range state.Received {
		if _, err := ledger.Receive(chunk); err != nil {
			return nil, fmt.Errorf("restore retained chunk: %w", err)
		}
	}
	ledger.mu.Lock()
	ledger.receivedSequence = state.ReceivedSequence
	ledger.terminal = state.TerminalReason
	ledger.mu.Unlock()
	for _, segment := range state.Committed {
		if _, err := ledger.Commit(segment); err != nil {
			return nil, fmt.Errorf("restore committed segment: %w", err)
		}
	}
	return ledger, nil
}

func (l *Ledger) snapshotLocked() Snapshot {
	s := Snapshot{SessionID: l.cfg.SessionID, ReceivedSequence: l.receivedSequence, ProcessedSequence: l.processedSequence, TerminalReason: l.terminal}
	for sequence := l.processedSequence + 1; sequence <= l.receivedSequence; sequence++ {
		if chunk, ok := l.received[uint64(sequence)]; ok {
			chunk.Audio = append([]byte(nil), chunk.Audio...)
			s.Replay = append(s.Replay, chunk.Chunk)
		}
	}
	for _, segment := range l.committed {
		s.Committed = append(s.Committed, segment)
	}
	return s
}
