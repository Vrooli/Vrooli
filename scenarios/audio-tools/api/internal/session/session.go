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
	ErrSessionClosed           = errors.New("audio-tools/session: session is closed")
	ErrUnknownSession          = errors.New("audio-tools/session: unknown session id")
	ErrTooManySubscribers      = errors.New("audio-tools/session: subscriber limit reached")
	ErrNilContext              = errors.New("audio-tools/session: subscribe context is nil")
	ErrReplayCursorUnavailable = errors.New("audio-tools/session: replay cursor is unavailable")
)

const maxSessionObservers = 16

type observer struct {
	events            chan SessionEvent
	done              chan struct{}
	resumeFromEventID string
}

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

	mu             sync.RWMutex
	observers      map[string]observer
	history        []SessionEvent
	historyBytes   int
	historyFloorID string
	closed         atomic.Bool
	cancelHook     func(reason BargeInReason, eventID string)
	inflightMu     sync.Mutex
	inflightID     string
	clk            schedule.Clock
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
	id := uuid.NewString()
	s := &Session{
		id:             id,
		transport:      opts.Transport,
		voice:          opts.Voice,
		language:       opts.Language,
		createdAt:      clk.Now(),
		observers:      make(map[string]observer),
		historyFloorID: id,
		cancelHook:     opts.CancelHook,
		clk:            clk,
	}
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
	return s.SubscribeFrom(ctx, bufferSize, "")
}

// SubscribeFrom registers an observer and, when fromEventID is non-empty,
// replays retained events after that cursor before delivering live events.
// Replay is bounded to the current process and session lifetime.
func (s *Session) SubscribeFrom(ctx context.Context, bufferSize int, fromEventID string) (string, <-chan SessionEvent, error) {
	if ctx == nil {
		return "", nil, ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	if bufferSize <= 0 {
		bufferSize = 64
	}
	s.mu.Lock()
	if s.closed.Load() {
		s.mu.Unlock()
		return "", nil, ErrSessionClosed
	}
	if len(s.observers) >= maxSessionObservers {
		s.mu.Unlock()
		return "", nil, ErrTooManySubscribers
	}
	replay, err := s.replayAfterLocked(fromEventID)
	if err != nil {
		s.mu.Unlock()
		return "", nil, err
	}
	key := uuid.NewString()
	ch := make(chan SessionEvent, bufferSize+len(replay))
	for _, evt := range replay {
		ch <- evt
	}
	resumeFrom := fromEventID
	if resumeFrom == "" {
		if len(s.history) > 0 {
			resumeFrom = s.history[len(s.history)-1].EventID
		} else {
			resumeFrom = s.historyFloorID
		}
	}
	sub := observer{events: ch, done: make(chan struct{}), resumeFromEventID: resumeFrom}
	s.observers[key] = sub
	s.mu.Unlock()

	// Background contexts have no Done channel, so they need no watcher. A
	// cancellable observer exits its watcher on either context cancellation or
	// explicit/session-driven unsubscribe.
	if done := ctx.Done(); done != nil {
		go func() {
			select {
			case <-done:
				s.Unsubscribe(key)
			case <-sub.done:
			}
		}()
	}

	return key, ch, nil
}

// Unsubscribe drops an observer. Safe to call repeatedly.
func (s *Session) Unsubscribe(key string) {
	s.mu.Lock()
	sub, ok := s.observers[key]
	if ok {
		delete(s.observers, key)
		close(sub.done)
		close(sub.events)
	}
	s.mu.Unlock()
}

// ObserverCount returns the current subscriber count.
func (s *Session) ObserverCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.observers)
}

// EmitEvent broadcasts an event to every observer. A slow observer is
// detached with a terminal overflow event; silently dropping transcript or
// assistant events would make the stream look complete when it is not.
func (s *Session) EmitEvent(evt SessionEvent) {
	if evt.EventID == "" {
		evt.EventID = uuid.NewString()
	}
	if evt.SessionID == "" {
		evt.SessionID = s.id
	}
	if evt.EmittedAt.IsZero() {
		evt.EmittedAt = s.nowOrSystem()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed.Load() {
		return
	}
	s.appendHistoryLocked(evt)
	for key, sub := range s.observers {
		select {
		case sub.events <- evt:
		default:
			// Preserve the session for other observers, but make the gap
			// explicit and terminate this stream. The terminal event replaces
			// queued events because those events can no longer be delivered
			// reliably to this observer.
			s.overflowObserverLocked(key, sub, len(sub.events)+1)
		}
	}
}

// MarkInflight records that an assistant action (typically TTS) is in flight
// under the given eventID; barge-in cancellation references it.
func (s *Session) MarkInflight(eventID string) {
	s.inflightMu.Lock()
	s.inflightID = eventID
	s.inflightMu.Unlock()
}

// ClearInflight clears the in-flight marker only if it still belongs to the
// action that just finished. A late completion from older TTS must not clear a
// newer action.
func (s *Session) ClearInflight(eventID string) {
	s.inflightMu.Lock()
	if s.inflightID == eventID {
		s.inflightID = ""
	}
	s.inflightMu.Unlock()
}

// BargeIn cancels any in-flight assistant action and emits a BargeInCancel
// event to observers. Called by the transport on a VAD speech_start.
func (s *Session) BargeIn(reason BargeInReason) {
	s.inflightMu.Lock()
	inflight := s.inflightID
	if inflight == "" {
		s.inflightMu.Unlock()
		return
	}
	s.inflightID = ""
	if s.cancelHook != nil {
		// Keep marker updates serialized with transport cancellation. Some
		// transports cancel shared work, so allowing MarkInflight to install a
		// newer action before this hook finishes could cancel that action too.
		s.cancelHook(reason, inflight)
	}
	s.inflightMu.Unlock()
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
	s.appendHistoryLocked(final)
	for key, sub := range s.observers {
		select {
		case sub.events <- final:
			close(sub.done)
			close(sub.events)
		default:
			// Shutdown is still a data boundary: if the final close event
			// cannot be queued, report the lost tail instead of ending cleanly.
			s.overflowObserverLocked(key, sub, len(sub.events)+1)
		}
	}
	s.observers = nil
	s.mu.Unlock()
}

// overflowObserverLocked replaces an observer's undeliverable backlog with a
// terminal event that makes the missing tail explicit. Caller holds s.mu.
func (s *Session) overflowObserverLocked(key string, sub observer, unavailable int) {
	delete(s.observers, key)
	close(sub.done)
	for len(sub.events) > 0 {
		<-sub.events
	}
	sub.events <- SessionEvent{
		EventID:   uuid.NewString(),
		SessionID: s.id,
		Type:      EventClosed,
		EmittedAt: s.nowOrSystem(),
		Closed: &SessionClosed{
			Reason:            fmt.Sprintf("observer-overflow: %d events unavailable", unavailable),
			ResumeFromEventID: sub.resumeFromEventID,
		},
	}
	close(sub.events)
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
