package main

import (
	"context"
	"fmt"
	"strings"
)

// conversationEventBatchResolver is an optional repository capability. It
// keeps ordinary repositories source-compatible while avoiding one SQL round
// trip per selected transcript event.
type conversationEventBatchResolver interface {
	ResolveEvents(ctx context.Context, sessionID string, eventIDs []string) ([]ConversationEvent, []string, error)
}

const conversationResolveBatchSize = 400

func orderedUniqueEventIDs(eventIDs []string) []string {
	seen := make(map[string]struct{}, len(eventIDs))
	ordered := make([]string, 0, len(eventIDs))
	for _, id := range eventIDs {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ordered = append(ordered, id)
	}
	return ordered
}

func (r *SQLConversationRepository) ResolveEvents(ctx context.Context, sessionID string, eventIDs []string) ([]ConversationEvent, []string, error) {
	ids := orderedUniqueEventIDs(eventIDs)
	if len(ids) == 0 {
		return []ConversationEvent{}, []string{}, nil
	}

	found := make(map[string]ConversationEvent, len(ids))
	for start := 0; start < len(ids); start += conversationResolveBatchSize {
		end := min(start+conversationResolveBatchSize, len(ids))
		batch := ids[start:end]
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",")
		query := `SELECT id, session_id, source, role, text, speech_paragraphs,
			COALESCE(original_speech_paragraphs, ''), summarized, created_at, occurred_at, sequence,
			delivery_state, tts_state, consumption_state, native_provenance_json
			FROM conversation_events WHERE session_id = ? AND id IN (` + placeholders + `)`
		args := make([]any, 0, len(batch)+1)
		args = append(args, sessionID)
		for _, id := range batch {
			args = append(args, id)
		}
		rows, err := r.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve conversation event batch: %w", err)
		}
		for rows.Next() {
			event, scanErr := scanConversationEvent(rows)
			if scanErr != nil {
				_ = rows.Close()
				return nil, nil, fmt.Errorf("scan conversation event batch: %w", scanErr)
			}
			found[event.ID] = event
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, nil, fmt.Errorf("read conversation event batch: %w", err)
		}
		if err := rows.Close(); err != nil {
			return nil, nil, fmt.Errorf("close conversation event batch: %w", err)
		}
	}
	return orderResolvedEvents(ids, found), missingResolvedEvents(ids, found), nil
}

func (r *InMemoryConversationRepository) ResolveEvents(_ context.Context, sessionID string, eventIDs []string) ([]ConversationEvent, []string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ids := orderedUniqueEventIDs(eventIDs)
	found := make(map[string]ConversationEvent, len(ids))
	if session := r.sessions[sessionID]; session != nil {
		selected := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			selected[id] = struct{}{}
		}
		for _, event := range session.events {
			if _, wanted := selected[event.ID]; wanted {
				found[event.ID] = event
			}
		}
	}
	return orderResolvedEvents(ids, found), missingResolvedEvents(ids, found), nil
}

func orderResolvedEvents(ids []string, found map[string]ConversationEvent) []ConversationEvent {
	events := make([]ConversationEvent, 0, len(found))
	for _, id := range ids {
		if event, ok := found[id]; ok {
			events = append(events, event)
		}
	}
	return events
}

func missingResolvedEvents(ids []string, found map[string]ConversationEvent) []string {
	missing := make([]string, 0)
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			missing = append(missing, id)
		}
	}
	return missing
}
