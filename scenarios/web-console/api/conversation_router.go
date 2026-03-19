package main

import (
	"context"
	"log"
	"strings"
	"time"
)

// TTSClientAck records terminal-side event playback outcomes for a routed
// assistant conversation event.
type TTSClientAck struct {
	EventID   string `json:"eventId"`
	Source    string `json:"source"`
	SessionID string `json:"sessionId"`
	Stage     string `json:"stage"`
	Backend   string `json:"backend,omitempty"`
	Message   string `json:"message,omitempty"`
}

func conversationAppendFailure(code, reason, source, sessionID string) ConversationAppendResult {
	return ConversationAppendResult{
		Appended:  false,
		Code:      code,
		Reason:    reason,
		Source:    source,
		SessionID: sessionID,
	}
}

// appendConversationEvent validates and publishes a trusted assistant response
// to the owning terminal session. Auto-TTS enablement is not checked here:
// the conversation log is the source of truth and TTS is only one consumer.
func (s *Server) appendConversationEvent(responseText, targetSessionID, source string) ConversationAppendResult {
	if strings.TrimSpace(targetSessionID) == "" {
		result := conversationAppendFailure("conversation_target_missing", "No web-console terminal session was identified for this conversation event", source, "")
		s.recordLastTTSRouting(result)
		return result
	}

	sess, ok := s.sessions.Get(targetSessionID)
	if !ok {
		result := conversationAppendFailure("conversation_target_missing", "The mapped terminal session is no longer available", source, targetSessionID)
		s.recordLastTTSRouting(result)
		return result
	}

	event, result := s.conversations.AppendAssistantEvent(targetSessionID, source, responseText)
	if result.Appended && !result.Duplicate {
		s.maybeSummarizeSpeechParagraphs(&event, targetSessionID)
		sess.SendConversation(event)
	}
	s.recordLastTTSRouting(result)
	return result
}

// appendUserConversationEvent validates and publishes a user prompt to
// the owning terminal session. No TTS routing is performed for user messages.
func (s *Server) appendUserConversationEvent(promptText, targetSessionID, source string) ConversationAppendResult {
	if strings.TrimSpace(targetSessionID) == "" {
		return conversationAppendFailure("conversation_target_missing", "No web-console terminal session was identified for this conversation event", source, "")
	}

	sess, ok := s.sessions.Get(targetSessionID)
	if !ok {
		return conversationAppendFailure("conversation_target_missing", "The mapped terminal session is no longer available", source, targetSessionID)
	}

	event, result := s.conversations.AppendUserEvent(targetSessionID, source, promptText)
	if result.Appended && !result.Duplicate {
		sess.SendConversation(event)
	}
	return result
}

// maybeSummarizeSpeechParagraphs optionally summarizes long assistant responses
// via the TTS summarizer before they are broadcast. The full original text is
// preserved in the event; only speechParagraphs is replaced with the summary.
// The original paragraphs are saved in OriginalSpeechParagraphs so the frontend
// can toggle between summarized and full playback.
func (s *Server) maybeSummarizeSpeechParagraphs(event *ConversationEvent, sessionID string) {
	if s.ttsSummarizer == nil {
		log.Printf("tts-summarize: skipped (no summarizer configured)")
		return
	}

	cfg := s.getTTSSummarizeConfig()
	if !cfg.Enabled {
		log.Printf("tts-summarize: skipped (disabled in config)")
		return
	}

	if len(event.Text) < cfg.CharThreshold {
		log.Printf("tts-summarize: skipped (text length %d < threshold %d)", len(event.Text), cfg.CharThreshold)
		return
	}

	normalized := NormalizeTextForSpeech(event.Text)
	if strings.TrimSpace(normalized) == "" {
		log.Printf("tts-summarize: skipped (normalized text is empty)")
		return
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Printf("tts-summarize: summarizing %d chars with model=%s level=%s timeout=%s",
		len(normalized), cfg.Model, cfg.Level, timeout)

	summary, err := s.ttsSummarizer.Summarize(ctx, normalized, cfg.Model, cfg.Level)
	if err != nil {
		log.Printf("tts-summarize: fallback to unsummarized (error: %v)", err)
		return
	}

	summary = strings.TrimSpace(summary)
	if summary == "" {
		log.Printf("tts-summarize: fallback to unsummarized (empty summary returned)")
		return
	}

	log.Printf("tts-summarize: success — reduced %d chars to %d chars (%.0f%% reduction)",
		len(normalized), len(summary), float64(len(normalized)-len(summary))/float64(len(normalized))*100)

	newParagraphs := SplitIntoSpeechParagraphs(summary)
	event.OriginalSpeechParagraphs = event.SpeechParagraphs
	event.SpeechParagraphs = newParagraphs
	event.Summarized = true
	s.conversations.UpdateSpeechParagraphs(sessionID, event.ID, newParagraphs)
}
