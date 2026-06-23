package main

import (
	"log"
	"sync"
	"time"
)

// Conversation envelope kinds carried over the global SSE channel. These are
// the `event:` field of each SSE frame and the `kind` field of the JSON
// envelope. The terminal WebSocket no longer carries any of these — the global
// hub → /api/v1/events/stream is the single conversation-event channel.
const (
	HubKindConversationEvent       = "conversation_event"
	HubKindConversationEventUpdate = "conversation_event_update"
	HubKindConversationOutOfSync   = "conversation_out_of_sync"
	HubKindSessionStatus           = "session_status"
)

// hubRingSize bounds the in-memory replay buffer keyed by global id. A
// reconnecting client presenting a Last-Event-ID newer than the oldest
// retained entry is replayed exactly; one older than the window falls back to a
// per-session out-of-sync nudge. This is a smoothness lever, not a correctness
// lever — the client always has GET /conversation?since_sequence=N as the
// authoritative backfill.
const hubRingSize = 1024

// hubSubscriberBuffer sizes each subscriber's delivery channel. A briefly
// throttled client can absorb a burst without forcing the out-of-sync path;
// when it fills, the publisher drops and signals resync rather than blocking.
const hubSubscriberBuffer = 256

// HubEnvelope is the JSON `data:` payload of every SSE frame. snake_case on the
// wire. Payload is an opaque per-kind object: a conversationEventPayload for
// the event kinds, an empty object for out-of-sync.
type HubEnvelope struct {
	ID        int64       `json:"id"`
	SessionID string      `json:"session_id"`
	Kind      string      `json:"kind"`
	Sequence  int64       `json:"sequence"`
	Payload   interface{} `json:"payload"`
}

// conversationEventPayload is the explicit per-event payload carried inside a
// HubEnvelope for the conversation_event / conversation_event_update kinds. It
// mirrors the camelCase TerminalMessage fields the UI already parses. Built
// explicitly (not via ConversationEvent marshal) because IsUpdate and
// SummarizeError are transient json:"-" fields on ConversationEvent.
type conversationEventPayload struct {
	ID                       string   `json:"id"`
	Source                   string   `json:"source"`
	Role                     string   `json:"role"`
	Text                     string   `json:"text"`
	SpeechParagraphs         []string `json:"speechParagraphs"`
	OriginalSpeechParagraphs []string `json:"originalSpeechParagraphs"`
	Summarized               bool     `json:"summarized"`
	CreatedAt                string   `json:"createdAt"`
	Sequence                 int64    `json:"sequence"`
	SummarizeError           string   `json:"summarizeError,omitempty"`
}

// hubSubscriber is one connected SSE client. events delivers live + replay
// envelopes; resync carries the session id of a dropped event so the handler
// can emit a conversation_out_of_sync nudge instead of silently losing the
// gap. The publisher never blocks on a slow subscriber.
type hubSubscriber struct {
	events chan HubEnvelope
	resync chan string
}

// ConversationHub is the process-wide conversation event channel. Every
// conversation event (assistant/user append, async summarize update,
// out-of-sync nudge) is published here and fanned out to all SSE subscribers,
// decoupled from any single session's terminal WebSocket. A bounded ring
// buffer keyed by a monotonic global id backs Last-Event-ID replay.
//
// DOC: docs/internal/SEAMS.md#conversation-hub-seam-api
type ConversationHub struct {
	mu          sync.Mutex
	nextID      int64
	ring        []HubEnvelope
	subscribers map[*hubSubscriber]struct{}
	dropCount   int64
}

// NewConversationHub returns a ready-to-use hub with an empty ring buffer.
func NewConversationHub() *ConversationHub {
	return &ConversationHub{
		ring:        make([]HubEnvelope, 0, hubRingSize),
		subscribers: make(map[*hubSubscriber]struct{}),
	}
}

// Publish assigns the next monotonic global id to env, retains it in the ring
// buffer, and fans it out to every subscriber. Non-blocking: a subscriber
// whose buffer is full has the envelope dropped and its resync channel pulsed
// with the dropped event's session id. Returns the assigned global id.
func (h *ConversationHub) Publish(env HubEnvelope) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	env.ID = h.nextID

	if len(h.ring) >= hubRingSize {
		copy(h.ring, h.ring[1:])
		h.ring[len(h.ring)-1] = env
	} else {
		h.ring = append(h.ring, env)
	}

	for sub := range h.subscribers {
		select {
		case sub.events <- env:
		default:
			h.dropCount++
			select {
			case sub.resync <- env.SessionID:
			default:
			}
			log.Printf("conversation-hub: event dropped (subscriber buffer full) — id=%d session=%s seq=%d total_drops=%d — resync signaled",
				env.ID, env.SessionID, env.Sequence, h.dropCount)
		}
	}
	return env.ID
}

// Subscribe registers a new SSE client and returns its handle. If lastEventID
// is > 0, the returned replay slice contains every retained envelope with a
// global id greater than lastEventID, in order, captured atomically with the
// subscription so no live event is missed or duplicated. The bool reports
// whether a gap was detected (lastEventID predates the retained window), in
// which case the caller must emit conversation_out_of_sync for the affected
// sessions before streaming the (possibly empty) replay + live tail.
//
// Caller must call Unsubscribe(sub) when the stream ends.
func (h *ConversationHub) Subscribe(lastEventID int64) (sub *hubSubscriber, replay []HubEnvelope, gap bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sub = &hubSubscriber{
		events: make(chan HubEnvelope, hubSubscriberBuffer),
		resync: make(chan string, hubSubscriberBuffer),
	}
	h.subscribers[sub] = struct{}{}

	if lastEventID > 0 {
		oldest := int64(0)
		if len(h.ring) > 0 {
			oldest = h.ring[0].ID
		}
		// A gap exists when the requested resume point is strictly older than
		// the oldest retained entry (its successor fell out of the buffer).
		// lastEventID == oldest-1 means the next entry is still retained, so
		// no gap. An empty ring with a non-zero cursor is also a gap.
		if len(h.ring) == 0 || lastEventID < oldest-1 {
			gap = true
		}
		for _, env := range h.ring {
			if env.ID > lastEventID {
				replay = append(replay, env)
			}
		}
	}
	return sub, replay, gap
}

// Unsubscribe removes sub from the fan-out. Idempotent. The subscriber's
// channels are owned by the handler goroutine and not closed here — the
// handler simply stops reading them after Unsubscribe returns.
func (h *ConversationHub) Unsubscribe(sub *hubSubscriber) {
	if sub == nil {
		return
	}
	h.mu.Lock()
	delete(h.subscribers, sub)
	h.mu.Unlock()
}

// RetainedSessionIDs returns the distinct session ids currently held in the
// ring buffer. Used to scope a gap-driven out-of-sync nudge to the sessions
// the client might have missed events for.
func (h *ConversationHub) RetainedSessionIDs() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	seen := make(map[string]struct{}, len(h.ring))
	ids := make([]string, 0, len(h.ring))
	for _, env := range h.ring {
		if _, ok := seen[env.SessionID]; ok {
			continue
		}
		seen[env.SessionID] = struct{}{}
		ids = append(ids, env.SessionID)
	}
	return ids
}

// DropCount returns the running count of envelopes dropped due to full
// subscriber buffers. Test-only.
func (h *ConversationHub) DropCount() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.dropCount
}

// publishConversationEvent maps a ConversationEvent onto the global hub. The
// transient IsUpdate flag selects the envelope kind; per-session sequence and
// session id come from the event. This is the single publish path that
// replaced the per-session fan-out's Send.
func (s *Server) publishConversationEvent(event ConversationEvent) {
	if s.hub == nil {
		return
	}
	kind := HubKindConversationEvent
	if event.IsUpdate {
		kind = HubKindConversationEventUpdate
	}
	s.hub.Publish(HubEnvelope{
		SessionID: event.SessionID,
		Kind:      kind,
		Sequence:  event.Sequence,
		Payload: conversationEventPayload{
			ID:                       event.ID,
			Source:                   event.Source,
			Role:                     string(event.Role),
			Text:                     event.Text,
			SpeechParagraphs:         event.SpeechParagraphs,
			OriginalSpeechParagraphs: event.OriginalSpeechParagraphs,
			Summarized:               event.Summarized,
			CreatedAt:                event.CreatedAt.UTC().Format(time.RFC3339),
			Sequence:                 event.Sequence,
			SummarizeError:           event.SummarizeError,
		},
	})
}
