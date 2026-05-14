package main

import (
	"log"
	"sync"
)

// defaultSessionEnv returns the per-session environment variables injected
// into the spawned PTY. Wired into session.Manager.envForSession by both
// constructors; tests can override the field for hermetic spawning.
func defaultSessionEnv(sessionID string) map[string]string {
	return map[string]string{
		"WC_WEB_CONSOLE_SESSION_ID": sessionID,
		"CODEX_HOME":                sessionCodexHome(sessionID),
		"WC_CODEX_SESSIONS_DIR":     sessionCodexSessionsDir(sessionID),
	}
}

// conversationChannelBuffer sizes each subscriber's conversation event buffer.
// Sized so a briefly throttled tab (background WS reads) can absorb a burst of
// agent output without triggering the out-of-sync resync path.
const conversationChannelBuffer = 256

// conversationSubscriber tracks per-client state for the conversation fan-out.
// The resync channel is a 1-buffered signal that fires when Send drops an
// event for this client because its event channel is full. The WS writer loop
// listens on this channel and emits a conversation_out_of_sync message so the
// client can refetch via GET /conversation?since_sequence=.
type conversationSubscriber struct {
	resync chan struct{}
}

// ConversationFanout owns the per-session conversation event fan-out. One
// instance per Session, created by session.Manager alongside the session and
// retrieved via session.Manager.ConversationFanout.
//
// Extracted from Session so that session lifecycle/IO can move into the
// internal/session/ sub-package without dragging the conversation event
// type with it.
type ConversationFanout struct {
	sessionID  string
	mu         sync.Mutex
	clients    map[chan ConversationEvent]*conversationSubscriber
	dropLogged bool
	dropCount  int64
}

// NewConversationFanout returns a ready-to-use fanout for a single session.
func NewConversationFanout(sessionID string) *ConversationFanout {
	return &ConversationFanout{
		sessionID: sessionID,
		clients:   make(map[chan ConversationEvent]*conversationSubscriber),
	}
}

// Subscribe returns a buffered channel that receives conversation events for
// this session. Caller must call Unsubscribe when done.
func (f *ConversationFanout) Subscribe() chan ConversationEvent {
	ch := make(chan ConversationEvent, conversationChannelBuffer)
	f.mu.Lock()
	f.clients[ch] = &conversationSubscriber{resync: make(chan struct{}, 1)}
	f.mu.Unlock()
	return ch
}

// ResyncSignal returns the out-of-sync signal channel bound to a given
// subscription. Returns nil if ch is not (or no longer) subscribed.
func (f *ConversationFanout) ResyncSignal(ch chan ConversationEvent) <-chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	if sub, ok := f.clients[ch]; ok {
		return sub.resync
	}
	return nil
}

// Unsubscribe removes and closes a channel. close(ch) must happen inside the
// lock so that Send (which iterates clients under the same lock) can never
// write to a closed channel.
func (f *ConversationFanout) Unsubscribe(ch chan ConversationEvent) {
	f.mu.Lock()
	delete(f.clients, ch)
	close(ch)
	f.mu.Unlock()
}

// Send fans out a conversation event to all subscribed clients. Non-blocking:
// if a client's channel is full, the message is dropped and the subscriber's
// resync signal is pulsed so the WS writer emits an out-of-sync notice and
// the client refetches the gap.
func (f *ConversationFanout) Send(event ConversationEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for ch, sub := range f.clients {
		select {
		case ch <- event:
		default:
			select {
			case sub.resync <- struct{}{}:
			default:
			}
			f.dropCount++
			f.dropLogged = true
			log.Printf("session %s: conversation event dropped (client channel full) — seq=%d id=%s total_drops=%d — resync signaled",
				f.sessionID, event.Sequence, event.ID, f.dropCount)
		}
	}
}

// DropCount returns the running count of dropped events. Test-only.
func (f *ConversationFanout) DropCount() int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.dropCount
}

// DropLogged reports whether any drop has been logged. Test-only.
func (f *ConversationFanout) DropLogged() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.dropLogged
}
