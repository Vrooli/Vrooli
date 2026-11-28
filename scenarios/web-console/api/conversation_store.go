package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/cli-core/agenttranscript"

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

// NativeProvenance is optional harness-owned identity. Projection ids and
// sequences are display coordinates only and must never mutate a harness.
type NativeProvenance struct {
	Provider          string `json:"provider"`
	SessionID         string `json:"sessionId"`
	TurnID            string `json:"turnId"`
	MessageID         string `json:"messageId"`
	BoundaryID        string `json:"boundaryId"`
	CompactionLineage string `json:"compactionLineage"`
}

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
	OccurredAt               time.Time                    `json:"-"`
	Sequence                 int64                        `json:"sequence"`
	DeliveryState            ConversationDeliveryState    `json:"deliveryState"`
	TTSState                 ConversationTTSState         `json:"ttsState"`
	ConsumptionState         ConversationConsumptionState `json:"consumptionState"`
	NativeProvenance         *NativeProvenance            `json:"nativeProvenance,omitempty"`
	// Kind, Author, AuthorName, Label and ThreadID classify the row; see
	// agenttranscript.Classification. Kind is "work" only on projection rows
	// that summarise the hidden rows between two conversation rows.
	Kind       string `json:"kind"`
	Author     string `json:"author"`
	AuthorName string `json:"authorName,omitempty"`
	Label      string `json:"label,omitempty"`
	ThreadID   string `json:"threadId,omitempty"`
	// Work is set on projection "work" rows only; it is never persisted.
	Work *ConversationWorkSpan `json:"work,omitempty"`
	// TextTruncated marks a work-detail row whose text is a preview.
	TextTruncated bool `json:"textTruncated,omitempty"`
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

type nativeEventIdentity struct {
	provider          string
	sessionID         string
	turnID            string
	messageID         string
	boundaryID        string
	compactionLineage string
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
	nativeMu   sync.Mutex
	// processor is the audio capability port for speech-shaped paragraph
	// splitting. Defaulted to audioports.PassthroughSpeechTextProcessor{} via
	// the speechProcessor() accessor below so tests and the in-memory
	// constructor don't have to wire it explicitly. Production wires the
	// audioports.RemoteSpeechTextProcessor backed by audio-tools.
	processor audioports.SpeechTextProcessor
	// launchPrompts attributes a programmatic session's first prompt to the
	// program that launched it. Nil leaves every typed prompt the human's.
	launchPrompts launchPromptAttributor
}

// SetSpeechProcessor lets package main inject the active port. The passthrough
// default remains in use until called.
func (s *ConversationStore) SetSpeechProcessor(ctx context.Context, p audioports.SpeechTextProcessor) {
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

func (s *ConversationStore) AppendAssistantEvent(ctx context.Context, sessionID, source, text string) (ConversationEvent, ConversationAppendResult) {
	return s.AppendAssistantEventAt(ctx, sessionID, source, text, time.Time{})
}

// AppendAssistantEventAt stores a transcript source's occurrence time without
// changing CreatedAt, which remains the local ingestion time.
func (s *ConversationStore) AppendAssistantEventAt(ctx context.Context, sessionID, source, text string, occurredAt time.Time) (ConversationEvent, ConversationAppendResult) {
	return s.AppendClassifiedEvent(ctx, sessionID, source, ConversationRoleAssistant, text, occurredAt, nil)
}

// AppendClassifiedEvent appends one transcript row with the classification
// its parser derived from native structure. A nil classification is derived
// from the text (agenttranscript.ClassifyText), so every capture path stores
// what a row is and who wrote it.
func (s *ConversationStore) AppendClassifiedEvent(ctx context.Context, sessionID, source string, role ConversationRole, text string, occurredAt time.Time, class *agenttranscript.Classification) (ConversationEvent, ConversationAppendResult) {
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
		reason := "Conversation event did not include assistant response text"
		if role == ConversationRoleUser {
			reason = "Conversation event did not include user prompt text"
		}
		return ConversationEvent{}, ConversationAppendResult{
			Appended:  false,
			Code:      "conversation_input_required",
			Reason:    reason,
			Source:    source,
			SessionID: sessionID,
		}
	}

	dedupKey := conversationDedupKey(sessionID, role, cleanText)
	// A confirmed voice handoff is a distinct delivery fact. A matching typed
	// prompt must not swallow its receipt through cross-transport deduplication.
	if source == codexVoiceHandoffSource {
		dedupKey += "\nvoice_handoff"
	}
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
		Role:             role,
		Text:             cleanText,
		CreatedAt:        time.Now().UTC(),
		OccurredAt:       occurredAt.UTC(),
		DeliveryState:    ConversationDeliveryPending,
		TTSState:         ConversationTTSIdle,
		ConsumptionState: ConversationConsumptionUnseen,
	}
	thread := s.classifyEvent(ctx, &event, class)
	persisted, err := s.repository.AppendEvent(ctx, event)
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
	s.registerThread(ctx, persisted, thread)
	reason := "Conversation event was appended to the owning terminal session"
	if role == ConversationRoleUser {
		reason = "User conversation event was appended to the owning terminal session"
	}
	return persisted, ConversationAppendResult{
		Appended:  true,
		Code:      "conversation_event_appended",
		Reason:    reason,
		Source:    source,
		SessionID: sessionID,
		EventID:   persisted.ID,
		Sequence:  persisted.Sequence,
	}
}

// classifyEvent records what the row is, who wrote it and which agent thread
// it belongs to, and prepares speech only for the agent's own messages
// (playback never reads tool calls or context). It returns the thread for
// registerThread once the row is persisted.
func (s *ConversationStore) classifyEvent(ctx context.Context, event *ConversationEvent, class *agenttranscript.Classification) agenttranscript.ThreadRef {
	resolved := classifyConversationRow(event.Source, string(event.Role), event.Text)
	if class != nil && class.Kind != "" {
		resolved = *class
	}
	if resolved.Author == "" {
		resolved.Author = agenttranscript.ClassifyText("", string(event.Role), "").Author
	}
	event.Kind = string(resolved.Kind)
	event.Author = string(resolved.Author)
	event.AuthorName = resolved.AuthorName
	event.Label = resolved.Label
	event.ThreadID = resolved.Thread.Key()
	if s.launchPrompts != nil && event.Role == ConversationRoleUser && event.ThreadID == "" &&
		event.Kind == string(agenttranscript.KindMessage) && event.Author == string(agenttranscript.AuthorHuman) {
		if owner, ok := s.launchPrompts.LaunchPromptOwner(ctx, event.SessionID); ok {
			event.Author = string(agenttranscript.AuthorAutomation)
			event.AuthorName = owner
			event.Label = "launch_prompt"
		}
	}
	if event.Role == ConversationRoleAssistant && event.Kind == string(agenttranscript.KindMessage) && event.SpeechParagraphs == nil {
		event.SpeechParagraphs = s.speechProcessor().SplitIntoParagraphs(event.Text)
	}
	return resolved.Thread
}

// registerThread records the agent thread a persisted row belongs to, so the
// projection can name it. A failure only loses the name, never the row.
func (s *ConversationStore) registerThread(ctx context.Context, event ConversationEvent, thread agenttranscript.ThreadRef) {
	if event.ThreadID == "" {
		return
	}
	threads, ok := s.repository.(conversationThreadStore)
	if !ok {
		return
	}
	if err := threads.UpsertThread(ctx, event.SessionID, conversationThreadFromRef(thread, event.CreatedAt)); err != nil {
		log.Printf("conversation: register thread %s for %s: %v", sanitizeID(event.ThreadID), sanitizeID(event.SessionID), err)
	}
}

// AppendNativeEvent is the capture path for providers whose native message
// identity is required for later control. The caller owns idempotency through
// the provider cursor; the native locator is persisted atomically with the
// projection event.
func (s *ConversationStore) AppendNativeEvent(ctx context.Context, sessionID, source, role, text string, provenance *NativeProvenance) (ConversationEvent, ConversationAppendResult) {
	return s.AppendNativeEventAt(ctx, sessionID, source, role, text, provenance, time.Time{})
}

// AppendNativeEventAt stores the provider's explicit occurrence time separately
// from CreatedAt, which remains the local ingestion time. A zero occurrence time
// means that the source format did not provide one.
func (s *ConversationStore) AppendNativeEventAt(ctx context.Context, sessionID, source, role, text string, provenance *NativeProvenance, occurredAt time.Time) (ConversationEvent, ConversationAppendResult) {
	s.nativeMu.Lock()
	defer s.nativeMu.Unlock()
	cleanText := normalizeConversationText(text)
	if strings.TrimSpace(sessionID) == "" || cleanText == "" || provenance == nil {
		return ConversationEvent{}, ConversationAppendResult{Appended: false, Code: "conversation_input_required", Source: source, SessionID: sessionID}
	}
	conversationRole := ConversationRoleAssistant
	if role == "user" {
		conversationRole = ConversationRoleUser
	}
	existing, found, err := s.repository.FindNativeEvent(ctx, sessionID, provenance)
	if err != nil {
		return ConversationEvent{}, ConversationAppendResult{Appended: false, Code: "conversation_store_failed", Reason: "Native event identity lookup failed", Source: source, SessionID: sessionID}
	}
	if found {
		if !occurredAt.IsZero() && !existing.OccurredAt.Equal(occurredAt) {
			if updater, ok := s.repository.(nativeOccurredAtUpdater); ok {
				updated, updateErr := updater.UpdateNativeEventOccurredAt(ctx, sessionID, provenance, occurredAt)
				if updateErr != nil {
					return ConversationEvent{}, ConversationAppendResult{Appended: false, Code: "conversation_store_failed", Reason: "Native occurrence timestamp update failed", Source: source, SessionID: sessionID}
				}
				if updated {
					existing.OccurredAt = occurredAt
				}
			}
		}
		return existing, ConversationAppendResult{Appended: true, Code: "conversation_duplicate", Reason: "Native provider identity was already appended", Source: source, SessionID: sessionID, EventID: existing.ID, Sequence: existing.Sequence, Duplicate: true}
	}
	event := ConversationEvent{ID: newConversationEventID(), SessionID: sessionID, Source: source, Role: conversationRole, Text: cleanText, CreatedAt: time.Now().UTC(), NativeProvenance: provenance, DeliveryState: ConversationDeliveryPending, TTSState: ConversationTTSIdle, ConsumptionState: ConversationConsumptionUnseen}
	event.OccurredAt = occurredAt.UTC()
	thread := s.classifyEvent(ctx, &event, nil)
	persisted, err := s.repository.AppendEvent(ctx, event)
	if err != nil {
		return ConversationEvent{}, ConversationAppendResult{Appended: false, Code: "conversation_store_failed", Source: source, SessionID: sessionID}
	}
	s.registerThread(ctx, persisted, thread)
	return persisted, ConversationAppendResult{Appended: true, Code: "conversation_event_appended", Source: source, SessionID: sessionID, EventID: persisted.ID, Sequence: persisted.Sequence}
}

type nativeOccurredAtUpdater interface {
	UpdateNativeEventOccurredAt(context.Context, string, *NativeProvenance, time.Time) (bool, error)
}

func nativeIdentity(provenance *NativeProvenance) nativeEventIdentity {
	if provenance == nil {
		return nativeEventIdentity{}
	}
	return nativeEventIdentity{
		provider: provenance.Provider, sessionID: provenance.SessionID, turnID: provenance.TurnID,
		messageID: provenance.MessageID, boundaryID: provenance.BoundaryID, compactionLineage: provenance.CompactionLineage,
	}
}

func nativeProvenanceEqual(left, right *NativeProvenance) bool {
	if left == nil || right == nil {
		return false
	}
	return left.Provider == right.Provider && left.SessionID == right.SessionID && left.TurnID == right.TurnID && left.MessageID == right.MessageID && left.BoundaryID == right.BoundaryID && left.CompactionLineage == right.CompactionLineage
}

func (s *ConversationStore) TruncateSessionAfter(ctx context.Context, sessionID string, sequence int64) error {
	if s == nil || s.repository == nil || strings.TrimSpace(sessionID) == "" || sequence < 0 {
		return fmt.Errorf("invalid conversation truncation target")
	}
	if err := s.repository.TruncateSessionAfter(ctx, sessionID, sequence); err != nil {
		return err
	}
	return nil
}

func (s *ConversationStore) AppendUserEvent(ctx context.Context, sessionID, source, text string) (ConversationEvent, ConversationAppendResult) {
	return s.AppendUserEventAt(ctx, sessionID, source, text, time.Time{})
}

// AppendUserEventAt stores a transcript source's occurrence time without
// changing CreatedAt, which remains the local ingestion time.
func (s *ConversationStore) AppendUserEventAt(ctx context.Context, sessionID, source, text string, occurredAt time.Time) (ConversationEvent, ConversationAppendResult) {
	return s.AppendClassifiedEvent(ctx, sessionID, source, ConversationRoleUser, text, occurredAt, nil)
}

// UpdateSpeechParagraphs replaces the SpeechParagraphs field for a stored event
// with a summarized version. The original paragraphs are preserved so the
// frontend can toggle between summarized and original playback.
func (s *ConversationStore) UpdateSpeechParagraphs(ctx context.Context, sessionID, eventID string, paragraphs []string) error {
	return s.repository.UpdateSpeechParagraphs(ctx, sessionID, eventID, paragraphs)
}

// GetEvent returns a copy of a single event by ID. The bool is false when the
// session or event is unknown.
func (s *ConversationStore) GetEvent(ctx context.Context, sessionID, eventID string) (ConversationEvent, bool) {
	event, ok, err := s.repository.GetEvent(ctx, sessionID, eventID)
	if err != nil {
		return ConversationEvent{}, false
	}
	return event, ok
}

// ResolveEvents returns complete records for stable IDs and explicitly lists
// IDs that no longer exist. Unlike GetEvent's legacy convenience wrapper, it
// preserves repository errors so bulk export cannot silently become partial.
func (s *ConversationStore) ResolveEvents(ctx context.Context, sessionID string, eventIDs []string) ([]ConversationEvent, []string, error) {
	if resolver, ok := s.repository.(conversationEventBatchResolver); ok {
		return resolver.ResolveEvents(ctx, sessionID, eventIDs)
	}

	events := make([]ConversationEvent, 0, len(eventIDs))
	missing := make([]string, 0)
	seen := make(map[string]struct{}, len(eventIDs))
	for _, id := range eventIDs {
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		event, found, err := s.repository.GetEvent(ctx, sessionID, id)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve conversation event %q: %w", sanitizeID(id), err)
		}
		if !found {
			missing = append(missing, id)
			continue
		}
		events = append(events, event)
	}
	return events, missing, nil
}

func (s *ConversationStore) ListSession(ctx context.Context, sessionID string) ConversationSessionState {
	state, err := s.repository.ListSession(ctx, sessionID)
	if err != nil {
		return ConversationSessionState{
			SessionID: sessionID,
			Events:    []ConversationEvent{},
		}
	}
	return state
}

func (s *ConversationStore) ListSessionPage(ctx context.Context, sessionID string, page ConversationPageQuery) (ConversationSessionState, bool) {
	state, hasMore, err := s.repository.ListSessionPage(ctx, sessionID, page)
	if err != nil {
		return ConversationSessionState{SessionID: sessionID, Events: []ConversationEvent{}}, false
	}
	return state, hasMore
}

// SessionCursor reads a session's cursor without loading its events.
func (s *ConversationStore) SessionCursor(ctx context.Context, sessionID string) ConversationCursor {
	cursor, err := s.repository.SessionCursor(ctx, sessionID)
	if err != nil {
		return ConversationCursor{}
	}
	return cursor
}

// LatestEventCreatedAt is the ingestion time of the session's newest event.
func (s *ConversationStore) LatestEventCreatedAt(ctx context.Context, sessionID string) (time.Time, bool) {
	createdAt, found, err := s.repository.LatestEventCreatedAt(ctx, sessionID)
	if err != nil {
		return time.Time{}, false
	}
	return createdAt, found
}

func (s *ConversationStore) CountSessionEvents(ctx context.Context, sessionID string) int64 {
	count, err := s.repository.CountSessionEvents(ctx, sessionID)
	if err != nil {
		return 0
	}
	return count
}

func (s *ConversationStore) SessionStorageBytes(ctx context.Context, sessionID string) int64 {
	size, err := s.repository.SessionStorageBytes(ctx, sessionID)
	if err != nil {
		return 0
	}
	return size
}

func (s *ConversationStore) SearchSession(ctx context.Context, sessionID string, query ConversationSearchQuery) (ConversationSearchResult, error) {
	return s.repository.SearchSession(ctx, sessionID, query)
}

func (s *ConversationStore) SearchArchived(ctx context.Context, filter ArchivedConversationSearchFilter) (ArchivedConversationSearchResult, error) {
	return s.repository.SearchArchived(ctx, filter)
}

func (s *ConversationStore) ListSessionRange(ctx context.Context, sessionID string, from, to int64) ([]ConversationEvent, error) {
	return s.repository.ListSessionRange(ctx, sessionID, from, to)
}

// HasConversationAfter reports whether tracking recorded an event after the
// supplied instant. It is used by the recovered-session health signal, once
// per recovered session in a session list, so it reads only the newest event.
func (s *ConversationStore) HasConversationAfter(ctx context.Context, sessionID string, after time.Time) bool {
	latest, found := s.LatestEventCreatedAt(ctx, sessionID)
	return found && latest.After(after)
}

func (s *ConversationStore) UpdateCursor(ctx context.Context, sessionID string, patch conversationCursorPatch) ConversationCursor {
	cursor, err := s.repository.UpdateCursor(ctx, sessionID, patch)
	if err != nil {
		return ConversationCursor{}
	}
	return cursor
}

func (s *ConversationStore) RecordPlaybackStage(ctx context.Context, sessionID, eventID, stage string) error {
	return s.repository.RecordPlaybackStage(ctx, sessionID, eventID, stage)
}

func (s *ConversationStore) DeleteSession(ctx context.Context, sessionID string) error {
	if err := s.repository.DeleteSession(ctx, sessionID); err != nil {
		return err
	}
	return nil
}

func (s *ConversationStore) DeleteSessionsByPrefix(ctx context.Context, prefix string) error {
	lister, ok := s.repository.(conversationSessionPrefixLister)
	if !ok {
		return fmt.Errorf("conversation repository does not support session prefix listing")
	}
	ids, err := lister.ListSessionIDsByPrefix(ctx, prefix)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.DeleteSession(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// CopySession duplicates the conversation history from oldID onto newID so a
// recovered session pane shows the prior conversation. See
// ConversationRepository.CopySession for the preservation semantics.
func (s *ConversationStore) CopySession(ctx context.Context, oldID, newID string) error {
	return s.repository.CopySession(ctx, oldID, newID)
}
