package main

import "log"

// conversationChannelBuffer sizes each subscriber's conversation event buffer.
// Sized so a briefly throttled tab (background WS reads) can absorb a burst of
// agent output without triggering the out-of-sync resync path.
const conversationChannelBuffer = 256

// conversationSubscriber tracks per-client state for the conversation fan-out.
// The resync channel is a 1-buffered signal that fires when SendConversation
// drops an event for this client because its event channel is full. The WS
// writer loop listens on this channel and emits a conversation_out_of_sync
// message so the client can refetch via GET /conversation?since_sequence=.
type conversationSubscriber struct {
	resync chan struct{}
}

// SubscribeConversation returns a buffered channel that receives conversation
// events for this session. Caller must call UnsubscribeConversation when done.
func (s *Session) SubscribeConversation() chan ConversationEvent {
	ch := make(chan ConversationEvent, conversationChannelBuffer)
	s.conversationMu.Lock()
	s.conversationClients[ch] = &conversationSubscriber{resync: make(chan struct{}, 1)}
	s.conversationMu.Unlock()
	return ch
}

// ConversationResyncSignal returns the out-of-sync signal channel bound to a
// given conversation subscription. Returns nil if ch is not (or no longer)
// subscribed. The WS writer loop selects on this channel and, on receive,
// emits conversation_out_of_sync so the client can refetch the gap via
// GET /conversation?since_sequence=N.
func (s *Session) ConversationResyncSignal(ch chan ConversationEvent) <-chan struct{} {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()
	if sub, ok := s.conversationClients[ch]; ok {
		return sub.resync
	}
	return nil
}

// UnsubscribeConversation removes and closes a conversation channel.
// close(ch) must happen inside the lock so that SendConversation (which
// iterates conversationClients under the same lock) can never write to a
// closed channel.
func (s *Session) UnsubscribeConversation(ch chan ConversationEvent) {
	s.conversationMu.Lock()
	delete(s.conversationClients, ch)
	close(ch)
	s.conversationMu.Unlock()
}

// SendConversation fans out a conversation event to all subscribed clients.
// Non-blocking: if a client's channel is full, the message is dropped and the
// subscriber's resync signal is pulsed so the WS writer emits an
// out-of-sync notice and the client refetches the gap.
func (s *Session) SendConversation(event ConversationEvent) {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()
	for ch, sub := range s.conversationClients {
		select {
		case ch <- event:
		default:
			select {
			case sub.resync <- struct{}{}:
			default:
			}
			s.conversationDropCount++
			s.conversationDropLogged = true
			log.Printf("session %s: conversation event dropped (client channel full) — seq=%d id=%s total_drops=%d — resync signaled",
				s.ID, event.Sequence, event.ID, s.conversationDropCount)
		}
	}
}
