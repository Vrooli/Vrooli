// Package session owns the replay-safe STT session contract. Transports may
// disconnect, but only this ledger decides acknowledged coverage and terminal
// state. Audio is retained until the processed cursor covers it.
package session

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"sync"

	"audio-tools/internal/protoint"
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
	ID            string
	Text          string
	StartSample   int64
	EndSample     int64
	SpeakerPolicy string
}

type Snapshot struct {
	SessionID         string
	ReceivedSequence  int64
	ProcessedSequence int64
	RetainedBytes     int
	TerminalReason    TerminalReason
	Replay            []Chunk
	Committed         []Segment
}

// MetadataSnapshot is the payload-free view used by diagnostics and
// qualification observers. Replay audio and committed transcript text remain
// available only through the ledger's session-owned replay contract.
type MetadataSnapshot struct {
	SessionID         string
	ReceivedSequence  int64
	ProcessedSequence int64
	RetainedBytes     int
	CommittedCount    int
	TerminalReason    TerminalReason
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
	committedOrder    []string
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
	if chunk.Sequence != protoint.ToUint64(l.receivedSequence+1) {
		return "", ErrGap
	}
	if l.retainedBytes+len(chunk.Audio) > l.cfg.MaxBytes {
		l.terminal = TerminalResourceExhausted
		return "", ErrResourceExhausted
	}
	copyAudio := append([]byte(nil), chunk.Audio...)
	chunk.Audio = copyAudio
	l.received[chunk.Sequence] = retainedChunk{Chunk: chunk, digest: digest}
	l.receivedSequence = protoint.FromUint64(chunk.Sequence)
	l.retainedBytes += len(chunk.Audio)
	return ReceivedNew, nil
}

func (l *Ledger) AcknowledgeProcessed(sequence uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if sequence > protoint.ToUint64(l.receivedSequence) {
		return ErrUnknownChunk
	}
	if protoint.FromUint64(sequence) <= l.processedSequence {
		return nil
	}
	for next := l.processedSequence + 1; next <= protoint.FromUint64(sequence); next++ {
		chunk, ok := l.received[protoint.ToUint64(next)]
		if !ok {
			return ErrUnknownChunk
		}
		delete(l.received, protoint.ToUint64(next))
		l.retainedBytes -= len(chunk.Audio)
	}
	l.processedSequence = protoint.FromUint64(sequence)
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
	// Commit order is the canonical order for backends that do not report
	// sample cursors (OverlapAgree today). A map cannot preserve it.
	l.committedOrder = append(l.committedOrder, segment.ID)
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

func (l *Ledger) MetadataSnapshot() MetadataSnapshot {
	if l == nil {
		return MetadataSnapshot{}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return MetadataSnapshot{
		SessionID:         l.cfg.SessionID,
		ReceivedSequence:  l.receivedSequence,
		ProcessedSequence: l.processedSequence,
		RetainedBytes:     l.retainedBytes,
		CommittedCount:    len(l.committed),
		TerminalReason:    l.terminal,
	}
}

// CommittedSegments returns an ordered copy of the durable transcript commits
// without copying the retained PCM replay journal. Reconnect handlers that
// only need transcript metadata should avoid the full Snapshot allocation.
func (l *Ledger) CommittedSegments() []Segment {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return committedInSampleOrder(l.committed, l.committedOrder)
}

func (l *Ledger) SessionID() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.cfg.SessionID
}

func (l *Ledger) PersistedState() PersistedState {
	l.mu.Lock()
	defer l.mu.Unlock()
	state := PersistedState{
		SessionID: l.cfg.SessionID, ResumeToken: l.cfg.ResumeToken, MaxBytes: l.cfg.MaxBytes,
		ReceivedSequence: l.receivedSequence, ProcessedSequence: l.processedSequence, TerminalReason: l.terminal,
	}
	for sequence := l.processedSequence + 1; sequence <= l.receivedSequence; sequence++ {
		if chunk, ok := l.received[protoint.ToUint64(sequence)]; ok {
			copy := chunk.Chunk
			copy.Audio = append([]byte(nil), copy.Audio...)
			state.Received = append(state.Received, copy)
		}
	}
	state.Committed = committedInSampleOrder(l.committed, l.committedOrder)
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
	s := Snapshot{SessionID: l.cfg.SessionID, ReceivedSequence: l.receivedSequence, ProcessedSequence: l.processedSequence, RetainedBytes: l.retainedBytes, TerminalReason: l.terminal}
	for sequence := l.processedSequence + 1; sequence <= l.receivedSequence; sequence++ {
		if chunk, ok := l.received[protoint.ToUint64(sequence)]; ok {
			chunk.Audio = append([]byte(nil), chunk.Audio...)
			s.Replay = append(s.Replay, chunk.Chunk)
		}
	}
	s.Committed = committedInSampleOrder(l.committed, l.committedOrder)
	return s
}

// committedInSampleOrder projects the committed map into canonical order:
// ascending sample range when the backend reports one, otherwise commit order
// (OverlapAgree does not populate sample cursors). Reconnect replay and
// persisted recovery promise a stable order, and map iteration order is random.
func committedInSampleOrder(committed map[string]Segment, order []string) []Segment {
	out := make([]Segment, 0, len(committed))
	seen := make(map[string]struct{}, len(order))
	for _, id := range order {
		segment, ok := committed[id]
		if !ok {
			continue
		}
		out = append(out, segment)
		seen[id] = struct{}{}
	}
	// Defensive: a segment restored without an order entry must still appear.
	for id, segment := range committed {
		if _, ok := seen[id]; !ok {
			out = append(out, segment)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].StartSample != out[j].StartSample {
			return out[i].StartSample < out[j].StartSample
		}
		return out[i].EndSample < out[j].EndSample
	})
	return out
}
