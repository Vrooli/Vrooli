package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	conversationH "web-console/handlers/conversation"
)

// conversationAdapter implements conversationH.Service against the server's
// ConversationStore, file-reference resolver, and TTS summarizer.
type conversationAdapter struct {
	srv *Server
}

func newConversationAdapter(s *Server) *conversationAdapter {
	return &conversationAdapter{srv: s}
}

func (a *conversationAdapter) Get(sessionID string, sinceSequence int64) (conversationH.SessionState, error) {
	if _, ok := a.srv.sessions.Get(sessionID); !ok {
		return conversationH.SessionState{}, fmt.Errorf("session %q: %w", sanitizeID(sessionID), conversationH.ErrSessionNotFound)
	}
	state := a.srv.conversations.ListSession(sessionID)
	events := state.Events
	if sinceSequence > 0 {
		filtered := make([]ConversationEvent, 0, len(events))
		for _, ev := range events {
			if ev.Sequence > sinceSequence {
				filtered = append(filtered, ev)
			}
		}
		events = filtered
	}
	return conversationH.SessionState{
		SessionID: state.SessionID,
		Events:    transportEvents(events),
		Cursor: conversationH.Cursor{
			LastSeenSequence:     state.Cursor.LastSeenSequence,
			LastListenedSequence: state.Cursor.LastListenedSequence,
		},
	}, nil
}

func (a *conversationAdapter) UpdateCursor(sessionID string, patch conversationH.CursorPatch) (conversationH.Cursor, error) {
	if _, ok := a.srv.sessions.Get(sessionID); !ok {
		return conversationH.Cursor{}, fmt.Errorf("session %q: %w", sanitizeID(sessionID), conversationH.ErrSessionNotFound)
	}
	storePatch := conversationCursorPatch{}
	if patch.HasLastSeenSequence {
		v := patch.LastSeenSequence
		storePatch.seenSequence = &v
	}
	if patch.HasLastListenedSequence {
		v := patch.LastListenedSequence
		storePatch.listenedSequence = &v
	}
	cur := a.srv.conversations.UpdateCursor(sessionID, storePatch)
	return conversationH.Cursor{
		LastSeenSequence:     cur.LastSeenSequence,
		LastListenedSequence: cur.LastListenedSequence,
	}, nil
}

func (a *conversationAdapter) SummarizeEvent(ctx context.Context, sessionID, eventID string) (conversationH.SummarizeResult, error) {
	if _, ok := a.srv.sessions.Get(sessionID); !ok {
		return conversationH.SummarizeResult{}, fmt.Errorf("session %q: %w", sanitizeID(sessionID), conversationH.ErrSessionNotFound)
	}
	if a.srv.ttsSummarization == nil {
		return conversationH.SummarizeResult{
			Error: "Summarizer not available — Ollama may not be running",
		}, nil
	}

	state := a.srv.conversations.ListSession(sessionID)
	var event *ConversationEvent
	for i := range state.Events {
		if state.Events[i].ID == eventID {
			event = &state.Events[i]
			break
		}
	}
	if event == nil {
		return conversationH.SummarizeResult{}, fmt.Errorf("event %q: %w", sanitizeID(eventID), conversationH.ErrNotFound)
	}
	if event.Role != ConversationRoleAssistant {
		return conversationH.SummarizeResult{}, fmt.Errorf("only assistant events can be summarized: %w", conversationH.ErrInvalidArgument)
	}

	cfg := a.srv.getTTSSummarizeConfig()
	normalized := NormalizeTextForSpeech(event.Text)
	if strings.TrimSpace(normalized) == "" {
		return conversationH.SummarizeResult{
			Error: "Event text is empty after normalization",
		}, nil
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := a.srv.ttsSummarization.Summarize(cctx, TTSSummarizeRequest{
		EventID: eventID,
		Path:    "on-demand",
		Text:    normalized,
	})
	if err != nil {
		logSummarizeResult("on-demand", cfg, eventID, len(normalized), result, err)
		return conversationH.SummarizeResult{
			Error: summarizeErrorMessage(err),
		}, nil
	}
	logSummarizeResult("on-demand", cfg, eventID, len(normalized), result, nil)

	a.srv.conversations.UpdateSpeechParagraphs(sessionID, eventID, result.Paragraphs)
	a.srv.invalidateTTSCacheForEvent(eventID)

	return conversationH.SummarizeResult{
		Summarized:       true,
		SpeechParagraphs: result.Paragraphs,
	}, nil
}

func (a *conversationAdapter) ResolveFileReference(ctx context.Context, sessionID, rawPath string) (conversationH.FileReference, error) {
	sess, ok := a.srv.sessions.Get(sessionID)
	if !ok {
		return conversationH.FileReference{}, fmt.Errorf("session %q: %w", sanitizeID(sessionID), conversationH.ErrSessionNotFound)
	}
	resolved, err := a.srv.resolveFileReference(ctx, sess, rawPath)
	if err != nil {
		return conversationH.FileReference{}, mapFileReferenceError(err)
	}
	ref := conversationH.FileReference{
		InputPath:       resolved.inputPath,
		ResolvedPath:    resolved.resolvedPath,
		Exists:          resolved.exists,
		ResolutionBasis: resolved.resolutionBasis,
		Category:        resolved.category,
		CanPreview:      resolved.canPreview,
	}
	if resolved.line != nil {
		ref.Line = *resolved.line
		ref.HasLine = true
	}
	return ref, nil
}

func (a *conversationAdapter) GetFileReferenceContent(ctx context.Context, sessionID, rawPath string) (conversationH.FileContent, error) {
	sess, ok := a.srv.sessions.Get(sessionID)
	if !ok {
		return conversationH.FileContent{}, fmt.Errorf("session %q: %w", sanitizeID(sessionID), conversationH.ErrSessionNotFound)
	}
	resolved, err := a.srv.resolveFileReference(ctx, sess, rawPath)
	if err != nil {
		return conversationH.FileContent{}, mapFileReferenceError(err)
	}
	if !resolved.canPreview {
		return conversationH.FileContent{}, fmt.Errorf("file type cannot be previewed: %w", conversationH.ErrPreviewUnavailable)
	}
	if resolved.sizeBytes > maxFilePreviewBytes {
		return conversationH.FileContent{}, fmt.Errorf("file exceeds preview limit of %d bytes: %w", maxFilePreviewBytes, conversationH.ErrPreviewUnavailable)
	}
	data, err := os.ReadFile(resolved.resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return conversationH.FileContent{}, fmt.Errorf("referenced file not found: %w", conversationH.ErrNotFound) //nolint:staticcheck // share NotFound code
		}
		return conversationH.FileContent{}, fmt.Errorf("read referenced file: %w", err)
	}
	if !utf8.Valid(data) || bytesContainNull(data) {
		return conversationH.FileContent{}, fmt.Errorf("file is not valid UTF-8 text: %w", conversationH.ErrPreviewUnavailable)
	}

	content := conversationH.FileContent{
		Path:        resolved.resolvedPath,
		Category:    resolved.category,
		ContentType: fileReferenceContentType(resolved.category),
		Content:     string(data),
		Truncated:   false,
	}
	if resolved.line != nil {
		content.Line = *resolved.line
		content.HasLine = true
	}
	return content, nil
}

// transportEvents converts the internal ConversationEvent slice into the
// transport-neutral shape used by the handler package.
func transportEvents(in []ConversationEvent) []conversationH.Event {
	out := make([]conversationH.Event, 0, len(in))
	for _, e := range in {
		out = append(out, conversationH.Event{
			ID:                       e.ID,
			SessionID:                e.SessionID,
			Source:                   e.Source,
			Role:                     string(e.Role),
			Text:                     e.Text,
			SpeechParagraphs:         append([]string(nil), e.SpeechParagraphs...),
			OriginalSpeechParagraphs: append([]string(nil), e.OriginalSpeechParagraphs...),
			Summarized:               e.Summarized,
			CreatedAt:                e.CreatedAt.Format(time.RFC3339Nano),
			Sequence:                 e.Sequence,
			DeliveryState:            string(e.DeliveryState),
			TTSState:                 string(e.TTSState),
			ConsumptionState:         string(e.ConsumptionState),
		})
	}
	return out
}

// mapFileReferenceError translates the legacy *fileReferenceError codes into
// handler-package sentinels so the Connect handler can pick the right
// Connect code.
func mapFileReferenceError(err error) error {
	var refErr *fileReferenceError
	if !asFileReferenceError(err, &refErr) {
		return err
	}
	switch refErr.code {
	case "file_reference_invalid":
		return fmt.Errorf("%s: %w", refErr.message, conversationH.ErrInvalidArgument)
	case "file_reference_not_allowed":
		return fmt.Errorf("%s: %w", refErr.message, conversationH.ErrPermissionDenied)
	case "file_reference_not_previewable":
		return fmt.Errorf("%s: %w", refErr.message, conversationH.ErrPreviewUnavailable)
	case "file_reference_too_large":
		return fmt.Errorf("%s: %w", refErr.message, conversationH.ErrPreviewUnavailable)
	case "file_reference_unresolvable", "file_reference_not_found":
		return fmt.Errorf("%s: %w", refErr.message, conversationH.ErrNotFound) //nolint:staticcheck // share NotFound code
	default:
		return fmt.Errorf("%s", refErr.message)
	}
}
