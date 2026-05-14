package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ConversationRepository persists semantic conversation history.
type ConversationRepository interface {
	AppendEvent(event ConversationEvent) (ConversationEvent, error)
	GetEvent(sessionID, eventID string) (ConversationEvent, bool, error)
	ListSession(sessionID string) (ConversationSessionState, error)
	UpdateSpeechParagraphs(sessionID, eventID string, paragraphs []string) error
	UpdateCursor(sessionID string, patch conversationCursorPatch) (ConversationCursor, error)
	RecordPlaybackStage(sessionID, eventID, stage string) error
	DeleteSession(sessionID string) error
}

type SQLConversationRepository struct {
	db *sql.DB
}

func NewSQLConversationRepository(db *sql.DB) *SQLConversationRepository {
	return &SQLConversationRepository{db: db}
}

func (r *SQLConversationRepository) AppendEvent(event ConversationEvent) (ConversationEvent, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return ConversationEvent{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	now := formatTime(event.CreatedAt)
	if now == "" {
		now = formatTime(time.Now())
	}

	if _, err := tx.Exec(`
		INSERT INTO conversation_sessions (session_id, created_at, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(session_id) DO NOTHING`,
		event.SessionID, now, now,
	); err != nil {
		return ConversationEvent{}, fmt.Errorf("ensure session: %w", err)
	}

	if err := tx.QueryRow(`
		UPDATE conversation_sessions
		SET last_sequence = last_sequence + 1, updated_at = ?
		WHERE session_id = ?
		RETURNING last_sequence`,
		now, event.SessionID,
	).Scan(&event.Sequence); err != nil {
		return ConversationEvent{}, fmt.Errorf("reserve sequence: %w", err)
	}

	speechJSON, err := marshalStringSlice(event.SpeechParagraphs)
	if err != nil {
		return ConversationEvent{}, fmt.Errorf("marshal speech paragraphs: %w", err)
	}
	var originalJSON any
	if len(event.OriginalSpeechParagraphs) > 0 {
		originalJSON, err = marshalStringSlice(event.OriginalSpeechParagraphs)
		if err != nil {
			return ConversationEvent{}, fmt.Errorf("marshal original speech paragraphs: %w", err)
		}
	}

	if _, err := tx.Exec(`
		INSERT INTO conversation_events (
			id, session_id, source, role, text, speech_paragraphs,
			original_speech_paragraphs, summarized, created_at, sequence,
			delivery_state, tts_state, consumption_state
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.SessionID,
		event.Source,
		string(event.Role),
		event.Text,
		speechJSON,
		originalJSON,
		boolToInt(event.Summarized),
		now,
		event.Sequence,
		string(event.DeliveryState),
		string(event.TTSState),
		string(event.ConsumptionState),
	); err != nil {
		return ConversationEvent{}, fmt.Errorf("insert event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return ConversationEvent{}, fmt.Errorf("commit event: %w", err)
	}
	return event, nil
}

func (r *SQLConversationRepository) GetEvent(sessionID, eventID string) (ConversationEvent, bool, error) {
	row := r.db.QueryRow(`
		SELECT id, session_id, source, role, text, speech_paragraphs,
		       COALESCE(original_speech_paragraphs, ''), summarized, created_at, sequence,
		       delivery_state, tts_state, consumption_state
		FROM conversation_events
		WHERE session_id = ? AND id = ?`,
		sessionID, eventID,
	)
	event, err := scanConversationEvent(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return ConversationEvent{}, false, nil
		}
		return ConversationEvent{}, false, fmt.Errorf("get event: %w", err)
	}
	return event, true, nil
}

func (r *SQLConversationRepository) ListSession(sessionID string) (ConversationSessionState, error) {
	state := ConversationSessionState{
		SessionID: sessionID,
		Events:    []ConversationEvent{},
	}

	row := r.db.QueryRow(`
		SELECT last_seen_sequence, last_listened_sequence
		FROM conversation_sessions
		WHERE session_id = ?`,
		sessionID,
	)
	switch err := row.Scan(&state.Cursor.LastSeenSequence, &state.Cursor.LastListenedSequence); err {
	case nil:
	case sql.ErrNoRows:
		return state, nil
	default:
		return ConversationSessionState{}, fmt.Errorf("load cursor: %w", err)
	}

	rows, err := r.db.Query(`
		SELECT id, session_id, source, role, text, speech_paragraphs,
		       COALESCE(original_speech_paragraphs, ''), summarized, created_at, sequence,
		       delivery_state, tts_state, consumption_state
		FROM conversation_events
		WHERE session_id = ?
		ORDER BY sequence`,
		sessionID,
	)
	if err != nil {
		return ConversationSessionState{}, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		event, err := scanConversationEvent(rows)
		if err != nil {
			return ConversationSessionState{}, err
		}
		state.Events = append(state.Events, event)
	}
	if err := rows.Err(); err != nil {
		return ConversationSessionState{}, fmt.Errorf("iterate events: %w", err)
	}

	return state, nil
}

func (r *SQLConversationRepository) UpdateSpeechParagraphs(sessionID, eventID string, paragraphs []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var currentSpeech string
	var currentOriginal sql.NullString
	if err := tx.QueryRow(`
		SELECT speech_paragraphs, original_speech_paragraphs
		FROM conversation_events
		WHERE session_id = ? AND id = ?`,
		sessionID, eventID,
	).Scan(&currentSpeech, &currentOriginal); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("load event speech: %w", err)
	}

	nextSpeech, err := marshalStringSlice(paragraphs)
	if err != nil {
		return fmt.Errorf("marshal summary paragraphs: %w", err)
	}

	originalSpeech := currentSpeech
	if currentOriginal.Valid && currentOriginal.String != "" {
		originalSpeech = currentOriginal.String
	}

	if _, err := tx.Exec(`
		UPDATE conversation_events
		SET original_speech_paragraphs = ?, speech_paragraphs = ?, summarized = 1
		WHERE session_id = ? AND id = ?`,
		originalSpeech, nextSpeech, sessionID, eventID,
	); err != nil {
		return fmt.Errorf("update speech paragraphs: %w", err)
	}

	return tx.Commit()
}

func (r *SQLConversationRepository) UpdateCursor(sessionID string, patch conversationCursorPatch) (ConversationCursor, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return ConversationCursor{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	now := formatTime(time.Now())
	if _, err := tx.Exec(`
		INSERT INTO conversation_sessions (session_id, created_at, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(session_id) DO NOTHING`,
		sessionID, now, now,
	); err != nil {
		return ConversationCursor{}, fmt.Errorf("ensure session: %w", err)
	}

	var cursor ConversationCursor
	if err := tx.QueryRow(`
		SELECT last_seen_sequence, last_listened_sequence
		FROM conversation_sessions
		WHERE session_id = ?`,
		sessionID,
	).Scan(&cursor.LastSeenSequence, &cursor.LastListenedSequence); err != nil {
		return ConversationCursor{}, fmt.Errorf("load cursor: %w", err)
	}

	if patch.seenSequence != nil && *patch.seenSequence > cursor.LastSeenSequence {
		cursor.LastSeenSequence = *patch.seenSequence
	}
	if patch.listenedSequence != nil && *patch.listenedSequence > cursor.LastListenedSequence {
		cursor.LastListenedSequence = *patch.listenedSequence
	}

	if _, err := tx.Exec(`
		UPDATE conversation_sessions
		SET last_seen_sequence = ?, last_listened_sequence = ?, updated_at = ?
		WHERE session_id = ?`,
		cursor.LastSeenSequence, cursor.LastListenedSequence, now, sessionID,
	); err != nil {
		return ConversationCursor{}, fmt.Errorf("update cursor: %w", err)
	}

	if err := applyCursorStateUpdates(tx, sessionID, cursor); err != nil {
		return ConversationCursor{}, err
	}

	if err := tx.Commit(); err != nil {
		return ConversationCursor{}, fmt.Errorf("commit cursor: %w", err)
	}
	return cursor, nil
}

func (r *SQLConversationRepository) RecordPlaybackStage(sessionID, eventID, stage string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var event ConversationEvent
	found, err := loadConversationEventForUpdate(tx, sessionID, eventID, &event)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	var cursor ConversationCursor
	if err := tx.QueryRow(`
		SELECT last_seen_sequence, last_listened_sequence
		FROM conversation_sessions
		WHERE session_id = ?`,
		sessionID,
	).Scan(&cursor.LastSeenSequence, &cursor.LastListenedSequence); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("load cursor: %w", err)
	}

	switch stage {
	case "received":
		if event.DeliveryState == ConversationDeliveryPending {
			event.DeliveryState = ConversationDeliveryReceived
		}
	case "seen", "correlated":
		event.DeliveryState = ConversationDeliverySeen
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
		if event.Sequence > cursor.LastSeenSequence {
			cursor.LastSeenSequence = event.Sequence
		}
		if event.Sequence > cursor.LastListenedSequence {
			cursor.LastListenedSequence = event.Sequence
		}
	case "rejected":
		event.TTSState = ConversationTTSRejected
	case "playback_failed":
		event.TTSState = ConversationTTSFailed
	}

	if err := updateConversationEventState(tx, event); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		UPDATE conversation_sessions
		SET last_seen_sequence = ?, last_listened_sequence = ?, updated_at = ?
		WHERE session_id = ?`,
		cursor.LastSeenSequence, cursor.LastListenedSequence, formatTime(time.Now()), sessionID,
	); err != nil {
		return fmt.Errorf("update cursor after playback: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit playback stage: %w", err)
	}
	return nil
}

func (r *SQLConversationRepository) DeleteSession(sessionID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(`DELETE FROM conversation_events WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("delete conversation events: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM conversation_sessions WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("delete conversation session: %w", err)
	}
	return tx.Commit()
}

type InMemoryConversationRepository struct {
	mu       sync.Mutex
	sessions map[string]*conversationSession
}

func NewInMemoryConversationRepository() *InMemoryConversationRepository {
	return &InMemoryConversationRepository{
		sessions: make(map[string]*conversationSession),
	}
}

func (r *InMemoryConversationRepository) AppendEvent(event ConversationEvent) (ConversationEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session := r.ensureSessionLocked(event.SessionID)
	session.nextSequence++
	event.Sequence = session.nextSequence
	session.events = append(session.events, event)
	return event, nil
}

func (r *InMemoryConversationRepository) GetEvent(sessionID, eventID string) (ConversationEvent, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok {
		return ConversationEvent{}, false, nil
	}
	for i := range session.events {
		if session.events[i].ID == eventID {
			return session.events[i], true, nil
		}
	}
	return ConversationEvent{}, false, nil
}

func (r *InMemoryConversationRepository) ListSession(sessionID string) (ConversationSessionState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok {
		return ConversationSessionState{
			SessionID: sessionID,
			Events:    []ConversationEvent{},
		}, nil
	}

	events := make([]ConversationEvent, len(session.events))
	copy(events, session.events)
	return ConversationSessionState{
		SessionID: sessionID,
		Events:    events,
		Cursor:    session.cursor,
	}, nil
}

func (r *InMemoryConversationRepository) UpdateSpeechParagraphs(sessionID, eventID string, paragraphs []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok {
		return nil
	}
	for i := range session.events {
		if session.events[i].ID == eventID {
			if len(session.events[i].OriginalSpeechParagraphs) == 0 {
				session.events[i].OriginalSpeechParagraphs = session.events[i].SpeechParagraphs
			}
			session.events[i].SpeechParagraphs = paragraphs
			session.events[i].Summarized = true
			return nil
		}
	}
	return nil
}

func (r *InMemoryConversationRepository) UpdateCursor(sessionID string, patch conversationCursorPatch) (ConversationCursor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session := r.ensureSessionLocked(sessionID)
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

	return session.cursor, nil
}

func (r *InMemoryConversationRepository) RecordPlaybackStage(sessionID, eventID, stage string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok {
		return nil
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
			event.DeliveryState = ConversationDeliverySeen
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
		return nil
	}
	return nil
}

func (r *InMemoryConversationRepository) DeleteSession(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, sessionID)
	return nil
}

func (r *InMemoryConversationRepository) ensureSessionLocked(sessionID string) *conversationSession {
	session, ok := r.sessions[sessionID]
	if ok {
		return session
	}
	session = &conversationSession{}
	r.sessions[sessionID] = session
	return session
}

type scannable interface {
	Scan(dest ...any) error
}

func scanConversationEvent(row scannable) (ConversationEvent, error) {
	var event ConversationEvent
	var role, createdAt string
	var speechJSON string
	var originalJSON string
	var summarized int
	var deliveryState, ttsState, consumptionState string
	if err := row.Scan(
		&event.ID,
		&event.SessionID,
		&event.Source,
		&role,
		&event.Text,
		&speechJSON,
		&originalJSON,
		&summarized,
		&createdAt,
		&event.Sequence,
		&deliveryState,
		&ttsState,
		&consumptionState,
	); err != nil {
		return ConversationEvent{}, fmt.Errorf("scan conversation event: %w", err)
	}

	event.Role = ConversationRole(role)
	event.Summarized = summarized == 1
	event.DeliveryState = ConversationDeliveryState(deliveryState)
	event.TTSState = ConversationTTSState(ttsState)
	event.ConsumptionState = ConversationConsumptionState(consumptionState)
	if parsed, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
		event.CreatedAt = parsed
	} else if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
		event.CreatedAt = parsed
	}
	if err := json.Unmarshal([]byte(speechJSON), &event.SpeechParagraphs); err != nil {
		return ConversationEvent{}, fmt.Errorf("decode speech paragraphs: %w", err)
	}
	if originalJSON != "" {
		if err := json.Unmarshal([]byte(originalJSON), &event.OriginalSpeechParagraphs); err != nil {
			return ConversationEvent{}, fmt.Errorf("decode original speech paragraphs: %w", err)
		}
	}
	return event, nil
}

func marshalStringSlice(values []string) (string, error) {
	data, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func applyCursorStateUpdates(tx *sql.Tx, sessionID string, cursor ConversationCursor) error {
	rows, err := tx.Query(`
		SELECT id, session_id, source, role, text, speech_paragraphs,
		       COALESCE(original_speech_paragraphs, ''), summarized, created_at, sequence,
		       delivery_state, tts_state, consumption_state
		FROM conversation_events
		WHERE session_id = ?
		ORDER BY sequence`,
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("query events for cursor update: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		event, err := scanConversationEvent(rows)
		if err != nil {
			return err
		}
		original := event
		if event.Sequence <= cursor.LastListenedSequence {
			event.DeliveryState = ConversationDeliverySeen
			event.ConsumptionState = ConversationConsumptionListened
			if event.TTSState == ConversationTTSIdle || event.TTSState == ConversationTTSPlaying {
				event.TTSState = ConversationTTSPlayed
			}
		} else if event.Sequence <= cursor.LastSeenSequence {
			if event.DeliveryState == ConversationDeliveryPending || event.DeliveryState == ConversationDeliveryReceived {
				event.DeliveryState = ConversationDeliverySeen
			}
			if event.ConsumptionState == ConversationConsumptionUnseen {
				event.ConsumptionState = ConversationConsumptionSeen
			}
		}
		if event.DeliveryState != original.DeliveryState || event.TTSState != original.TTSState || event.ConsumptionState != original.ConsumptionState {
			if err := updateConversationEventState(tx, event); err != nil {
				return err
			}
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate cursor update events: %w", err)
	}
	return nil
}

func loadConversationEventForUpdate(tx *sql.Tx, sessionID, eventID string, event *ConversationEvent) (bool, error) {
	row := tx.QueryRow(`
		SELECT id, session_id, source, role, text, speech_paragraphs,
		       COALESCE(original_speech_paragraphs, ''), summarized, created_at, sequence,
		       delivery_state, tts_state, consumption_state
		FROM conversation_events
		WHERE session_id = ? AND id = ?`,
		sessionID, eventID,
	)
	current, err := scanConversationEvent(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	*event = current
	return true, nil
}

func updateConversationEventState(tx *sql.Tx, event ConversationEvent) error {
	if _, err := tx.Exec(`
		UPDATE conversation_events
		SET delivery_state = ?, tts_state = ?, consumption_state = ?
		WHERE session_id = ? AND id = ?`,
		string(event.DeliveryState),
		string(event.TTSState),
		string(event.ConsumptionState),
		event.SessionID,
		event.ID,
	); err != nil {
		return fmt.Errorf("update event state: %w", err)
	}
	return nil
}

func newConversationEventID() string {
	return uuid.New().String()
}
