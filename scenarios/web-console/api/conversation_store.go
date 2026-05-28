package main

import (
	"strings"
	"sync"
	"time"

	"web-console/internal/audioports"
	"web-console/terminal"
)

const conversationDedupTTL = 30 * time.Second

type ConversationRole string

const (
	ConversationRoleAssistant ConversationRole = "assistant"
	ConversationRoleUser      ConversationRole = "user"
)

type ConversationDeliveryState string

const (
	ConversationDeliveryPending  ConversationDeliveryState = "pending"
	ConversationDeliveryReceived ConversationDeliveryState = "received"
	ConversationDeliverySeen     ConversationDeliveryState = "seen"
)

type ConversationTTSState string

const (
	ConversationTTSIdle     ConversationTTSState = "idle"
	ConversationTTSPlaying  ConversationTTSState = "playing"
	ConversationTTSPlayed   ConversationTTSState = "played"
	ConversationTTSRejected ConversationTTSState = "rejected"
	ConversationTTSFailed   ConversationTTSState = "failed"
)

type ConversationConsumptionState string

const (
	ConversationConsumptionUnseen    ConversationConsumptionState = "unseen"
	ConversationConsumptionSeen      ConversationConsumptionState = "seen"
	ConversationConsumptionListening ConversationConsumptionState = "listening"
	ConversationConsumptionListened  ConversationConsumptionState = "listened"
)

type ConversationEvent struct {
	ID                       string                       `json:"id"`
	SessionID                string                       `json:"sessionId"`
	Source                   string                       `json:"source"`
	Role                     ConversationRole             `json:"role"`
	Text                     string                       `json:"text"`
	SpeechParagraphs         []string                     `json:"speechParagraphs"`
	OriginalSpeechParagraphs []string                     `json:"originalSpeechParagraphs,omitempty"`
	Summarized               bool                         `json:"summarized"`
	CreatedAt                time.Time                    `json:"createdAt"`
	Sequence                 int64                        `json:"sequence"`
	DeliveryState            ConversationDeliveryState    `json:"deliveryState"`
	TTSState                 ConversationTTSState         `json:"ttsState"`
	ConsumptionState         ConversationConsumptionState `json:"consumptionState"`
	// IsUpdate is a transient flag (not persisted) indicating this event is an
	// async update to a previously delivered event (e.g. summarization result).
	// The WS forwarder uses this to send a conversation_event_update message.
	IsUpdate bool `json:"-"`
	// SummarizeError is a transient field (not persisted) that carries an
	// auto-summarization failure message to connected clients. When set, the
	// WS forwarder includes it in the conversation_event_update payload so the
	// UI can surface a persistent banner with retry affordance.
	SummarizeError string `json:"-"`
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

type conversationSession struct {
	nextSequence int64
	events       []ConversationEvent
	cursor       ConversationCursor
}

type ConversationAppendResult struct {
	Appended  bool   `json:"appended"`
	Code      string `json:"code"`
	Reason    string `json:"reason"`
	Source    string `json:"source"`
	SessionID string `json:"sessionId,omitempty"`
	EventID   string `json:"eventId,omitempty"`
	Sequence  int64  `json:"sequence,omitempty"`
	Duplicate bool   `json:"duplicate,omitempty"`
}

type conversationCursorPatch struct {
	seenSequence     *int64
	listenedSequence *int64
}

type conversationDedup struct {
	mu   sync.Mutex
	seen map[string]conversationDedupEntry
	ttl  time.Duration
}

type conversationDedupEntry struct {
	at      time.Time
	eventID string
}

func newConversationDedup() *conversationDedup {
	return &conversationDedup{seen: make(map[string]conversationDedupEntry)}
}

func (d *conversationDedup) seenRecently(key string) (string, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	ttl := d.ttl
	if ttl == 0 {
		ttl = conversationDedupTTL
	}
	now := time.Now()
	for k, entry := range d.seen {
		if now.Sub(entry.at) > ttl {
			delete(d.seen, k)
		}
	}
	if entry, ok := d.seen[key]; ok {
		return entry.eventID, true
	}
	return "", false
}

func (d *conversationDedup) remember(key, eventID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	ttl := d.ttl
	if ttl == 0 {
		ttl = conversationDedupTTL
	}
	now := time.Now()
	for k, entry := range d.seen {
		if now.Sub(entry.at) > ttl {
			delete(d.seen, k)
		}
	}
	d.seen[key] = conversationDedupEntry{
		at:      now,
		eventID: eventID,
	}
}

type ConversationStore struct {
	repository ConversationRepository
	dedup      *conversationDedup
	// processor is the audio capability port for speech-shaped paragraph
	// splitting. Defaulted to audioports.PassthroughSpeechTextProcessor{} via
	// the speechProcessor() accessor below so tests and the in-memory
	// constructor don't have to wire it explicitly. Production wires the
	// audioports.RemoteSpeechTextProcessor backed by audio-tools.
	processor audioports.SpeechTextProcessor
}

// SetSpeechProcessor lets package main inject the active port. The passthrough
// default remains in use until called.
func (s *ConversationStore) SetSpeechProcessor(p audioports.SpeechTextProcessor) {
	s.processor = p
}

func (s *ConversationStore) speechProcessor() audioports.SpeechTextProcessor {
	if s.processor == nil {
		return audioports.PassthroughSpeechTextProcessor{}
	}
	return s.processor
}

func NewConversationStore() *ConversationStore {
	return NewConversationStoreWithRepository(NewInMemoryConversationRepository())
}

func NewConversationStoreWithRepository(repository ConversationRepository) *ConversationStore {
	if repository == nil {
		repository = NewInMemoryConversationRepository()
	}
	return &ConversationStore{
		repository: repository,
		dedup:      newConversationDedup(),
	}
}

func normalizeConversationText(text string) string {
	return strings.TrimSpace(string(terminal.StripEscapes([]byte(text))))
}

// conversationDedupKey intentionally omits the event source so that the same
// assistant text arriving from two transports (e.g. codex_tailer + claude_hook)
// collapses into a single event. The dedup window is short (30s) so this only
// suppresses true same-message replays, not semantically distinct utterances
// that happen to repeat a prior phrase later in the conversation.
func conversationDedupKey(sessionID string, role ConversationRole, text string) string {
	return strings.Join([]string{sessionID, string(role), text}, "\n")
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

	dedupKey := conversationDedupKey(sessionID, ConversationRoleAssistant, cleanText)
	if eventID, ok := s.dedup.seenRecently(dedupKey); ok {
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

	event := ConversationEvent{
		ID:               newConversationEventID(),
		SessionID:        sessionID,
		Source:           source,
		Role:             ConversationRoleAssistant,
		Text:             cleanText,
		SpeechParagraphs: s.speechProcessor().SplitIntoParagraphs(cleanText),
		CreatedAt:        time.Now().UTC(),
		DeliveryState:    ConversationDeliveryPending,
		TTSState:         ConversationTTSIdle,
		ConsumptionState: ConversationConsumptionUnseen,
	}
	persisted, err := s.repository.AppendEvent(event)
	if err != nil {
		return ConversationEvent{}, ConversationAppendResult{
			Appended:  false,
			Code:      "conversation_store_failed",
			Reason:    "Conversation event could not be persisted",
			Source:    source,
			SessionID: sessionID,
		}
	}
	s.dedup.remember(dedupKey, persisted.ID)

	return persisted, ConversationAppendResult{
		Appended:  true,
		Code:      "conversation_event_appended",
		Reason:    "Conversation event was appended to the owning terminal session",
		Source:    source,
		SessionID: sessionID,
		EventID:   persisted.ID,
		Sequence:  persisted.Sequence,
	}
}

func (s *ConversationStore) AppendUserEvent(sessionID, source, text string) (ConversationEvent, ConversationAppendResult) {
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
			Reason:    "Conversation event did not include user prompt text",
			Source:    source,
			SessionID: sessionID,
		}
	}

	dedupKey := conversationDedupKey(sessionID, ConversationRoleUser, cleanText)
	if eventID, ok := s.dedup.seenRecently(dedupKey); ok {
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

	event := ConversationEvent{
		ID:               newConversationEventID(),
		SessionID:        sessionID,
		Source:           source,
		Role:             ConversationRoleUser,
		Text:             cleanText,
		SpeechParagraphs: nil,
		CreatedAt:        time.Now().UTC(),
		DeliveryState:    ConversationDeliveryPending,
		TTSState:         ConversationTTSIdle,
		ConsumptionState: ConversationConsumptionUnseen,
	}
	persisted, err := s.repository.AppendEvent(event)
	if err != nil {
		return ConversationEvent{}, ConversationAppendResult{
			Appended:  false,
			Code:      "conversation_store_failed",
			Reason:    "Conversation event could not be persisted",
			Source:    source,
			SessionID: sessionID,
		}
	}
	s.dedup.remember(dedupKey, persisted.ID)

	return persisted, ConversationAppendResult{
		Appended:  true,
		Code:      "conversation_event_appended",
		Reason:    "User conversation event was appended to the owning terminal session",
		Source:    source,
		SessionID: sessionID,
		EventID:   persisted.ID,
		Sequence:  persisted.Sequence,
	}
}

// UpdateSpeechParagraphs replaces the SpeechParagraphs field for a stored event
// with a summarized version. The original paragraphs are preserved so the
// frontend can toggle between summarized and original playback.
func (s *ConversationStore) UpdateSpeechParagraphs(sessionID, eventID string, paragraphs []string) {
	_ = s.repository.UpdateSpeechParagraphs(sessionID, eventID, paragraphs)
}

// GetEvent returns a copy of a single event by ID. The bool is false when the
// session or event is unknown.
func (s *ConversationStore) GetEvent(sessionID, eventID string) (ConversationEvent, bool) {
	event, ok, err := s.repository.GetEvent(sessionID, eventID)
	if err != nil {
		return ConversationEvent{}, false
	}
	return event, ok
}

func (s *ConversationStore) ListSession(sessionID string) ConversationSessionState {
	state, err := s.repository.ListSession(sessionID)
	if err != nil {
		return ConversationSessionState{
			SessionID: sessionID,
			Events:    []ConversationEvent{},
		}
	}
	return state
}

func (s *ConversationStore) UpdateCursor(sessionID string, patch conversationCursorPatch) ConversationCursor {
	cursor, err := s.repository.UpdateCursor(sessionID, patch)
	if err != nil {
		return ConversationCursor{}
	}
	return cursor
}

func (s *ConversationStore) RecordPlaybackStage(sessionID, eventID, stage string) {
	_ = s.repository.RecordPlaybackStage(sessionID, eventID, stage)
}

func (s *ConversationStore) DeleteSession(sessionID string) {
	_ = s.repository.DeleteSession(sessionID)
}

// CopySession duplicates the conversation history from oldID onto newID so a
// recovered session pane shows the prior conversation. See
// ConversationRepository.CopySession for the preservation semantics.
func (s *ConversationStore) CopySession(oldID, newID string) error {
	return s.repository.CopySession(oldID, newID)
}
