// Package session owns the policy and lifecycle of interactive sessions. It
// deliberately does not know about WebSockets or protobufs; transports call
// this seam and relay the bytes unchanged.
package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"vrooli-bridge/internal/audit"

	"github.com/vrooli/api-core/schedule"
)

const (
	Scope              = "vrooli-bridge:session"
	DefaultWindow      = uint32(64)
	DefaultIdle        = 5 * time.Minute
	DefaultMaxLifetime = 2 * time.Hour
	historyLimit       = 256
)

var (
	ErrScopeDenied   = errors.New("session scope is not granted")
	ErrOwnerReauth   = errors.New("owner re-authentication is required")
	ErrUnknown       = errors.New("session not found")
	ErrSequenceGap   = errors.New("session data sequence gap")
	ErrWindowFull    = errors.New("session receive window is full")
	ErrAlreadyClosed = errors.New("session is closed")
)

type OpenRequest struct {
	ID          string
	NodeID      string
	OwnerID     string
	Scopes      []string
	Reauth      bool
	Window      uint32
	Idle        time.Duration
	MaxLifetime time.Duration
}

type ResizeRequest struct{ Columns, Rows uint32 }

type DataResult struct {
	SessionID string
	Sequence  uint64
	Data      []byte
}

type State struct {
	ID, NodeID, OwnerID    string
	OpenedAt, LastActivity time.Time
	Idle, MaxLifetime      time.Duration
	Window, Outstanding    uint32
	NextSequence           uint64
	NextOutputSequence     uint64
	Closed                 bool
	done                   chan struct{}
}

type Manager struct {
	mu       sync.Mutex
	clock    schedule.Clock
	audit    audit.Sink
	sessions map[string]*State
	outputs  map[string]map[chan DataResult]struct{}
	history  map[string][]DataResult
	newID    func() string
}

func NewManager(clk schedule.Clock, sink audit.Sink) *Manager {
	if clk == nil {
		clk = schedule.System()
	}
	return &Manager{clock: clk, audit: sink, sessions: make(map[string]*State), outputs: make(map[string]map[chan DataResult]struct{}), history: make(map[string][]DataResult)}
}

// SubscribeOutput returns the node-to-control-plane byte stream for a live
// session. The caller must invoke the returned unsubscribe function. Output
// subscribers are deliberately separate from the browser transport so a node
// reconnect cannot race the WebSocket lifecycle.
func (m *Manager) SubscribeOutput(id string) (<-chan DataResult, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[id]; !ok {
		return nil, func() {}, ErrUnknown
	}
	replay := append([]DataResult(nil), m.history[id]...)
	ch := make(chan DataResult, len(replay)+32)
	set := m.outputs[id]
	if set == nil {
		set = make(map[chan DataResult]struct{})
		m.outputs[id] = set
	}
	set[ch] = struct{}{}
	for _, result := range replay {
		ch <- DataResult{SessionID: result.SessionID, Sequence: result.Sequence, Data: append([]byte(nil), result.Data...)}
	}
	return ch, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if current := m.outputs[id]; current != nil {
			delete(current, ch)
			if len(current) == 0 {
				delete(m.outputs, id)
			}
		}
	}, nil
}

// DeliverOutput accepts bytes produced by the node backend and fans them out
// to the authenticated owner transport. Sequence numbers are node-owned and
// must be monotonic per session; a gap is rejected before any byte is exposed.
func (m *Manager) DeliverOutput(ctx context.Context, id string, sequence uint64, data []byte) error {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return ErrUnknown
	}
	if err := m.expiredLocked(s); err != nil {
		m.mu.Unlock()
		_ = m.Close(ctx, id, err.Error())
		return err
	}
	if s.Closed {
		m.mu.Unlock()
		return ErrAlreadyClosed
	}
	if sequence != s.NextOutputSequence {
		if sequence < s.NextOutputSequence {
			for _, prior := range m.history[id] {
				if prior.Sequence == sequence && string(prior.Data) == string(data) {
					m.mu.Unlock()
					return nil
				}
			}
		}
		m.mu.Unlock()
		return fmt.Errorf("%w: output got %d want %d", ErrSequenceGap, sequence, s.NextOutputSequence)
	}
	s.NextOutputSequence++
	s.LastActivity = m.clock.Now().UTC()
	m.history[id] = append(m.history[id], DataResult{SessionID: id, Sequence: sequence, Data: append([]byte(nil), data...)})
	if len(m.history[id]) > historyLimit {
		m.history[id] = m.history[id][len(m.history[id])-historyLimit:]
	}
	listeners := make([]chan DataResult, 0, len(m.outputs[id]))
	for ch := range m.outputs[id] {
		listeners = append(listeners, ch)
	}
	m.mu.Unlock()
	result := DataResult{SessionID: id, Sequence: sequence, Data: append([]byte(nil), data...)}
	for _, ch := range listeners {
		select {
		case ch <- result:
		case <-ctx.Done():
			return ctx.Err()
		default:
			_ = m.Close(ctx, id, "output_backpressure")
			return ErrWindowFull
		}
	}
	return nil
}

func (m *Manager) Open(ctx context.Context, req OpenRequest) (State, error) {
	if !hasScope(req.Scopes) {
		return State{}, ErrScopeDenied
	}
	if req.OwnerID == "" || !req.Reauth {
		return State{}, ErrOwnerReauth
	}
	if req.ID == "" || req.NodeID == "" {
		return State{}, fmt.Errorf("session id and node id are required")
	}
	now := m.clock.Now().UTC()
	s := &State{ID: req.ID, NodeID: req.NodeID, OwnerID: req.OwnerID, OpenedAt: now, LastActivity: now,
		Window: boundedWindow(req.Window), Idle: boundedIdle(req.Idle), MaxLifetime: boundedLifetime(req.MaxLifetime), done: make(chan struct{})}
	m.mu.Lock()
	if _, exists := m.sessions[req.ID]; exists {
		existing := m.sessions[req.ID]
		if !existing.Closed && existing.NodeID == req.NodeID && existing.OwnerID == req.OwnerID {
			existing.LastActivity = now
			state := *existing
			m.mu.Unlock()
			return state, nil
		}
		m.mu.Unlock()
		return State{}, fmt.Errorf("session %q already exists", req.ID)
	}
	m.sessions[req.ID] = s
	m.mu.Unlock()
	if err := m.record(ctx, s, audit.ActionSessionOpen, audit.OutcomeAccepted, "session opened"); err != nil {
		m.Close(ctx, req.ID, "audit_failed")
		return State{}, err
	}
	return *s, nil
}

func (m *Manager) AcceptData(ctx context.Context, id string, sequence uint64, data []byte) (DataResult, error) {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return DataResult{}, ErrUnknown
	}
	if err := m.expiredLocked(s); err != nil {
		m.mu.Unlock()
		_ = m.Close(ctx, id, err.Error())
		return DataResult{}, err
	}
	if s.Closed {
		m.mu.Unlock()
		return DataResult{}, ErrAlreadyClosed
	}
	if s.Outstanding >= s.Window {
		m.mu.Unlock()
		return DataResult{}, ErrWindowFull
	}
	if sequence != s.NextSequence {
		m.mu.Unlock()
		return DataResult{}, fmt.Errorf("%w: got %d want %d", ErrSequenceGap, sequence, s.NextSequence)
	}
	s.NextSequence++
	s.Outstanding++
	s.LastActivity = m.clock.Now().UTC()
	m.mu.Unlock()
	return DataResult{SessionID: id, Sequence: sequence, Data: append([]byte(nil), data...)}, nil
}

func (m *Manager) Acknowledge(id string, count uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return ErrUnknown
	}
	if count >= s.Outstanding {
		s.Outstanding = 0
	} else {
		s.Outstanding -= count
	}
	s.LastActivity = m.clock.Now().UTC()
	return nil
}

func (m *Manager) Resize(ctx context.Context, id string, req ResizeRequest) error {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return ErrUnknown
	}
	if err := m.expiredLocked(s); err != nil {
		m.mu.Unlock()
		_ = m.Close(ctx, id, err.Error())
		return err
	}
	s.LastActivity = m.clock.Now().UTC()
	copyState := *s
	m.mu.Unlock()
	return m.record(ctx, &copyState, audit.ActionSessionResize, audit.OutcomeAccepted, fmt.Sprintf("resize %dx%d", req.Columns, req.Rows))
}

func (m *Manager) Close(ctx context.Context, id, reason string) error {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return ErrUnknown
	}
	if s.Closed {
		m.mu.Unlock()
		return nil
	}
	s.Closed = true
	close(s.done)
	delete(m.history, id)
	copyState := *s
	m.mu.Unlock()
	return m.record(ctx, &copyState, audit.ActionSessionClose, audit.OutcomeCompleted, reason)
}

// Kill is the operator kill switch. It is deliberately idempotent and uses the
// same close path as a client close so the audit trail has one terminal record.
func (m *Manager) Kill(ctx context.Context, id string) error {
	return m.Close(ctx, id, "operator_kill")
}

// CloseByNode terminates every live session owned by a node whose channel has
// disappeared. An interactive session cannot safely remain open after its
// remote PTY transport is gone; closing it promptly gives the owner a typed,
// auditable outcome instead of waiting for the idle timeout.
func (m *Manager) CloseByNode(ctx context.Context, nodeID, reason string) {
	m.mu.Lock()
	ids := make([]string, 0)
	for id, s := range m.sessions {
		if s.NodeID == nodeID && !s.Closed {
			ids = append(ids, id)
		}
	}
	m.mu.Unlock()
	for _, id := range ids {
		_ = m.Close(ctx, id, reason)
	}
}

// Done returns a channel that closes when the session is terminated. HTTP
// transports use it to interrupt a blocked read and close the remote stream.
func (m *Manager) Done(id string) (<-chan struct{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, ErrUnknown
	}
	return s.done, nil
}

func (m *Manager) Get(id string) (State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return State{}, ErrUnknown
	}
	return *s, nil
}

func (m *Manager) Expire(ctx context.Context, id string) error {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return ErrUnknown
	}
	expired := m.expiredLocked(s)
	m.mu.Unlock()
	if expired != nil {
		return m.Close(ctx, id, expired.Error())
	}
	return nil
}

func (m *Manager) expiredLocked(s *State) error {
	now := m.clock.Now().UTC()
	if now.Sub(s.OpenedAt) >= s.MaxLifetime {
		return fmt.Errorf("session lifetime exceeded")
	}
	if now.Sub(s.LastActivity) >= s.Idle {
		return fmt.Errorf("session idle timeout exceeded")
	}
	return nil
}

func (m *Manager) record(ctx context.Context, s *State, action audit.Action, outcome audit.Outcome, detail string) error {
	if m.audit == nil {
		return nil
	}
	_, err := m.audit.Append(ctx, audit.Record{Action: action, Outcome: outcome, Actor: s.OwnerID, NodeID: s.NodeID, RunID: s.ID, Detail: detail})
	return err
}

func hasScope(scopes []string) bool {
	for _, scope := range scopes {
		if scope == Scope {
			return true
		}
	}
	return false
}
func boundedWindow(v uint32) uint32 {
	if v == 0 {
		return DefaultWindow
	}
	if v > 1024 {
		return 1024
	}
	return v
}
func boundedIdle(v time.Duration) time.Duration {
	if v <= 0 {
		return DefaultIdle
	}
	if v < time.Second {
		return time.Second
	}
	if v > 24*time.Hour {
		return 24 * time.Hour
	}
	return v
}
func boundedLifetime(v time.Duration) time.Duration {
	if v <= 0 {
		return DefaultMaxLifetime
	}
	if v < time.Second {
		return time.Second
	}
	if v > 7*24*time.Hour {
		return 7 * 24 * time.Hour
	}
	return v
}
