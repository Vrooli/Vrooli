package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/vrooli/api-core/schedule"
)

var (
	ErrSessionClosed      = errors.New("audio-tools/session: session is closed")
	ErrUnknownSession     = errors.New("audio-tools/session: unknown session id")
	ErrTooManySubscribers = errors.New("audio-tools/session: subscriber limit reached")
)

// Session is a duplex voice interaction with multi-observer pub/sub.
//
// Concrete transports drive the session by calling EmitEvent and consuming
// SendText. The session owns barge-in coordination: when the transport
// reports a VAD speech_start during in-flight TTS, the session emits a
// BargeInCancel event to all observers and asks the transport to cancel.
type Session struct {
	id        string
	transport string
	voice     string
	language  string

	createdAt time.Time

	mu         sync.RWMutex
	observers  map[string]chan SessionEvent
	closed     atomic.Bool
	cancelHook func(reason BargeInReason, eventID string)
	inflightID atomic.Value // string
	clk        schedule.Clock
}

// nowOrSystem is a tiny accessor that returns the injected clock or
// falls back to schedule.System() if a Session was constructed via raw
// struct literal in older tests. New() always wires opts.Clock or
// schedule.System(), so this is a defensive read.
func (s *Session) nowOrSystem() time.Time {
	if s.clk == nil {
		return schedule.System().Now()
	}
	return s.clk.Now()
}

// Options configures a new session.
type Options struct {
	Transport string
	Voice     string
	Language  string
	// CancelHook is invoked when barge-in fires; transports register this to
	// short-circuit their TTS-out channel.
	CancelHook func(reason BargeInReason, eventID string)
	// Clock is the wall-clock seam used for createdAt + EmittedAt
	// timestamps. Defaults to schedule.System(); tests inject FakeClock for
	// deterministic timestamps.
	Clock schedule.Clock
}

// New creates a fresh Session. The session is not yet attached to a transport
// — the transport calls Open with itself to bind.
func New(opts Options) *Session {
	clk := opts.Clock
	if clk == nil {
		clk = schedule.System()
	}
	s := &Session{
		id:         uuid.NewString(),
		transport:  opts.Transport,
		voice:      opts.Voice,
		language:   opts.Language,
		createdAt:  clk.Now(),
		observers:  make(map[string]chan SessionEvent),
		cancelHook: opts.CancelHook,
		clk:        clk,
	}
	s.inflightID.Store("")
	return s
}

// ID returns the session ID.
func (s *Session) ID() string { return s.id }

// Transport returns the transport tag.
func (s *Session) Transport() string { return s.transport }

// Voice returns the configured TTS voice.
func (s *Session) Voice() string { return s.voice }

// Language returns the configured STT language.
func (s *Session) Language() string { return s.language }

// Subscribe registers a new observer with a bounded channel. Returns the
// observer key (for Unsubscribe) and the receive channel.
func (s *Session) Subscribe(ctx context.Context, bufferSize int) (string, <-chan SessionEvent, error) {
	if s.closed.Load() {
		return "", nil, ErrSessionClosed
	}
	if bufferSize <= 0 {
		bufferSize = 64
	}
	key := uuid.NewString()
	ch := make(chan SessionEvent, bufferSize)
	s.mu.Lock()
	s.observers[key] = ch
	s.mu.Unlock()

	// Auto-unsubscribe when ctx cancels.
	go func() {
		<-ctx.Done()
		s.Unsubscribe(key)
	}()

	return key, ch, nil
}

// Unsubscribe drops an observer. Safe to call repeatedly.
func (s *Session) Unsubscribe(key string) {
	s.mu.Lock()
	ch, ok := s.observers[key]
	if ok {
		delete(s.observers, key)
		close(ch)
	}
	s.mu.Unlock()
}

// ObserverCount returns the current subscriber count.
func (s *Session) ObserverCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.observers)
}

// EmitEvent broadcasts an event to every observer. Slow observers drop the
// event rather than blocking the publisher; the assumption is that observers
// keep up or accept loss.
func (s *Session) EmitEvent(evt SessionEvent) {
	if s.closed.Load() {
		return
	}
	if evt.EventID == "" {
		evt.EventID = uuid.NewString()
	}
	if evt.SessionID == "" {
		evt.SessionID = s.id
	}
	if evt.EmittedAt.IsZero() {
		evt.EmittedAt = s.nowOrSystem()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range s.observers {
		select {
		case ch <- evt:
		default:
			// Drop on backpressure; observers must read fast or accept loss.
		}
	}
}

// MarkInflight records that an assistant action (typically TTS) is in flight
// under the given eventID; barge-in cancellation references it.
func (s *Session) MarkInflight(eventID string) {
	s.inflightID.Store(eventID)
}

// ClearInflight clears the inflight marker (e.g., when TTS finishes naturally).
func (s *Session) ClearInflight() { s.inflightID.Store("") }

// BargeIn cancels any in-flight assistant action and emits a BargeInCancel
// event to observers. Called by the transport on a VAD speech_start.
func (s *Session) BargeIn(reason BargeInReason) {
	inflight, _ := s.inflightID.Load().(string)
	if inflight == "" {
		return
	}
	s.inflightID.Store("")
	if s.cancelHook != nil {
		s.cancelHook(reason, inflight)
	}
	s.EmitEvent(SessionEvent{
		Type:          EventBargeInCancel,
		BargeInCancel: &BargeInCancel{Reason: reason, CanceledEventID: inflight},
	})
}

// Close shuts the session down. Subsequent EmitEvent / Subscribe calls return
// ErrSessionClosed. Existing observers receive a SessionClosed event then
// their channels close.
func (s *Session) Close(reason string) {
	if s.closed.Swap(true) {
		return
	}
	final := SessionEvent{
		EventID:   uuid.NewString(),
		SessionID: s.id,
		Type:      EventClosed,
		EmittedAt: s.nowOrSystem(),
		Closed:    &SessionClosed{Reason: reason},
	}
	s.mu.Lock()
	for _, ch := range s.observers {
		select {
		case ch <- final:
		default:
		}
		close(ch)
	}
	s.observers = nil
	s.mu.Unlock()
}

// Registry maps session ID -> *Session. Used by the SessionService handler.
type Registry struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewRegistry() *Registry {
	return &Registry{sessions: make(map[string]*Session)}
}

func (r *Registry) Add(s *Session) { r.mu.Lock(); r.sessions[s.id] = s; r.mu.Unlock() }

func (r *Registry) Get(id string) (*Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sessions[id]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownSession, id)
	}
	return s, nil
}

func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, id)
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.sessions)
}
