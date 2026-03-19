package main

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

const conversationDedupTTL = 30 * time.Second

type ConversationRole string

const (
	ConversationRoleAssistant ConversationRole = "assistant"
)

type ConversationDeliveryState string

const (
	ConversationDeliveryPending  ConversationDeliveryState = "pending"
	ConversationDeliveryReceived ConversationDeliveryState = "received"
	ConversationDeliverySeen     ConversationDeliveryState = "seen"
)

type ConversationTTSState string

const (
	ConversationTTSIdle       ConversationTTSState = "idle"
	ConversationTTSPlaying    ConversationTTSState = "playing"
	ConversationTTSPlayed     ConversationTTSState = "played"
	ConversationTTSRejected   ConversationTTSState = "rejected"
	ConversationTTSFailed     ConversationTTSState = "failed"
)

type ConversationConsumptionState string

const (
	ConversationConsumptionUnseen    ConversationConsumptionState = "unseen"
	ConversationConsumptionSeen      ConversationConsumptionState = "seen"
	ConversationConsumptionListening ConversationConsumptionState = "listening"
	ConversationConsumptionListened  ConversationConsumptionState = "listened"
)

type ConversationEvent struct {
	ID               string                       `json:"id"`
	SessionID        string                       `json:"sessionId"`
	Source           string                       `json:"source"`
	Role             ConversationRole             `json:"role"`
	Text             string                       `json:"text"`
	SpeechParagraphs []string                     `json:"speechParagraphs"`
	CreatedAt        time.Time                    `json:"createdAt"`
	Sequence         int64                        `json:"sequence"`
	DeliveryState    ConversationDeliveryState    `json:"deliveryState"`
	TTSState         ConversationTTSState         `json:"ttsState"`
	ConsumptionState ConversationConsumptionState `json:"consumptionState"`
}

type ConversationCursor struct {
	LastSeenSequence     int64 `json:"lastSeenSequence"`
	LastListenedSequence int64 `json:"lastListenedSequence"`
}

type ConversationSessionState struct {
	SessionID string              `json:"sessionId"`
	Events    []ConversationEvent `json:"events"`
	Cursor    ConversationCursor  `json:"cursor"`
}

type ConversationAppendResult struct {
	Appended   bool   `json:"appended"`
	Code       string `json:"code"`
	Reason     string `json:"reason"`
	Source     string `json:"source"`
	SessionID  string `json:"sessionId,omitempty"`
	EventID    string `json:"eventId,omitempty"`
	Sequence   int64  `json:"sequence,omitempty"`
	Duplicate  bool   `json:"duplicate,omitempty"`
}

type conversationCursorPatch struct {
	seenSequence     *int64
	listenedSequence *int64
}

type conversationSession struct {
	nextSequence int64
	events       []ConversationEvent
	cursor       ConversationCursor
}

type conversationDedup struct {
	mu   sync.Mutex
	seen map[string]time.Time
	ttl  time.Duration
}

func newConversationDedup() *conversationDedup {
	return &conversationDedup{seen: make(map[string]time.Time)}
}

func (d *conversationDedup) seenRecently(key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	ttl := d.ttl
	if ttl == 0 {
		ttl = conversationDedupTTL
	}
	now := time.Now()
	for k, ts := range d.seen {
		if now.Sub(ts) > ttl {
			delete(d.seen, k)
		}
	}
	if _, ok := d.seen[key]; ok {
		return true
	}
	d.seen[key] = now
	return false
}

type ConversationStore struct {
	mu       sync.RWMutex
	sessions map[string]*conversationSession
	dedup    *conversationDedup
}

func NewConversationStore() *ConversationStore {
	return &ConversationStore{
		sessions: make(map[string]*conversationSession),
		dedup:    newConversationDedup(),
	}
}

func normalizeConversationText(text string) string {
	return strings.TrimSpace(string(stripANSI([]byte(text))))
}

func newConversationEventID(source, sessionID, text string) string {
	sum := sha1.Sum([]byte(source + "\n" + sessionID + "\n" + text))
	return hex.EncodeToString(sum[:])
}

func (s *ConversationStore) ensureSessionLocked(sessionID string) *conversationSession {
	session, ok := s.sessions[sessionID]
	if ok {
		return session
	}
	session = &conversationSession{}
	s.sessions[sessionID] = session
	return session
}

func (s *ConversationStore) AppendAssistantEvent(sessionID, source, text string) (ConversationEvent, ConversationAppendResult) {
	cleanText := normalizeConversationText(text)
	if strings.TrimSpace(sessionID) == "" {
		return ConversationEvent{}, ConversationAppendResult{
			Appended: false,
			Code:     "conversation_target_missing",
			Reason:   "No terminal session was identified for this conversation event",
			Source:   source,
		}
	}
	if cleanText == "" {
		return ConversationEvent{}, ConversationAppendResult{
			Appended:  false,
			Code:      "conversation_input_required",
			Reason:    "Conversation event did not include assistant response text",
			Source:    source,
			SessionID: sessionID,
		}
	}

	eventID := newConversationEventID(source, sessionID, cleanText)
	if s.dedup.seenRecently(eventID) {
		return ConversationEvent{}, ConversationAppendResult{
			Appended:  true,
			Code:      "conversation_duplicate",
			Reason:    "Conversation event was already appended recently",
			Source:    source,
			SessionID: sessionID,
			EventID:   eventID,
			Duplicate: true,
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.ensureSessionLocked(sessionID)
	session.nextSequence++
	event := ConversationEvent{
		ID:               eventID,
		SessionID:        sessionID,
		Source:           source,
		Role:             ConversationRoleAssistant,
		Text:             cleanText,
		SpeechParagraphs: SplitIntoSpeechParagraphs(cleanText),
		CreatedAt:        time.Now().UTC(),
		Sequence:         session.nextSequence,
		DeliveryState:    ConversationDeliveryPending,
		TTSState:         ConversationTTSIdle,
		ConsumptionState: ConversationConsumptionUnseen,
	}
	session.events = append(session.events, event)

	return event, ConversationAppendResult{
		Appended:  true,
		Code:      "conversation_event_appended",
		Reason:    "Conversation event was appended to the owning terminal session",
		Source:    source,
		SessionID: sessionID,
		EventID:   event.ID,
		Sequence:  event.Sequence,
	}
}

func (s *ConversationStore) ListSession(sessionID string) ConversationSessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return ConversationSessionState{
			SessionID: sessionID,
			Events:    []ConversationEvent{},
		}
	}

	events := make([]ConversationEvent, len(session.events))
	copy(events, session.events)
	return ConversationSessionState{
		SessionID: sessionID,
		Events:    events,
		Cursor:    session.cursor,
	}
}

func (s *ConversationStore) UpdateCursor(sessionID string, patch conversationCursorPatch) ConversationCursor {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.ensureSessionLocked(sessionID)
	if patch.seenSequence != nil && *patch.seenSequence > session.cursor.LastSeenSequence {
		session.cursor.LastSeenSequence = *patch.seenSequence
	}
	if patch.listenedSequence != nil && *patch.listenedSequence > session.cursor.LastListenedSequence {
		session.cursor.LastListenedSequence = *patch.listenedSequence
	}

	for i := range session.events {
		event := &session.events[i]
		if event.Sequence <= session.cursor.LastListenedSequence {
			event.DeliveryState = ConversationDeliverySeen
			event.ConsumptionState = ConversationConsumptionListened
			if event.TTSState == ConversationTTSIdle || event.TTSState == ConversationTTSPlaying {
				event.TTSState = ConversationTTSPlayed
			}
			continue
		}
		if event.Sequence <= session.cursor.LastSeenSequence {
			if event.DeliveryState == ConversationDeliveryPending || event.DeliveryState == ConversationDeliveryReceived {
				event.DeliveryState = ConversationDeliverySeen
			}
			if event.ConsumptionState == ConversationConsumptionUnseen {
				event.ConsumptionState = ConversationConsumptionSeen
			}
		}
	}

	return session.cursor
}

func (s *ConversationStore) RecordPlaybackStage(sessionID, eventID, stage string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return
	}
	for i := range session.events {
		event := &session.events[i]
		if event.ID != eventID {
			continue
		}
		switch stage {
		case "received":
			if event.DeliveryState == ConversationDeliveryPending {
				event.DeliveryState = ConversationDeliveryReceived
			}
		case "seen", "correlated":
			if event.DeliveryState != ConversationDeliverySeen {
				event.DeliveryState = ConversationDeliverySeen
			}
			if event.ConsumptionState == ConversationConsumptionUnseen {
				event.ConsumptionState = ConversationConsumptionSeen
			}
		case "playback_started":
			event.TTSState = ConversationTTSPlaying
			event.ConsumptionState = ConversationConsumptionListening
		case "playback_succeeded":
			event.TTSState = ConversationTTSPlayed
			event.DeliveryState = ConversationDeliverySeen
			event.ConsumptionState = ConversationConsumptionListened
			if event.Sequence > session.cursor.LastSeenSequence {
				session.cursor.LastSeenSequence = event.Sequence
			}
			if event.Sequence > session.cursor.LastListenedSequence {
				session.cursor.LastListenedSequence = event.Sequence
			}
		case "rejected":
			event.TTSState = ConversationTTSRejected
		case "playback_failed":
			event.TTSState = ConversationTTSFailed
		}
		return
	}
}
