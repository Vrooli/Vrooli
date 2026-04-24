package main

import (
	"context"
	"fmt"
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
		// Send event to clients immediately so the text lands in the UI with
		// no delay — audio is handled separately below.
		sess.SendConversation(event)

		cfg := s.getTTSSummarizeConfig()
		shouldSummarize := s.ttsSummarizer != nil && cfg.Enabled && len(event.Text) >= cfg.CharThreshold
		if shouldSummarize {
			// Summarize-first path: wait for summarization to finish before
			// pre-synthesizing audio, so the cached audio matches whatever
			// paragraphs end up on the event (summary on success, original on
			// failure). This closes the pre-cache race where audio was
			// synthesized from the raw response and never invalidated.
			go func(ev ConversationEvent) {
				s.asyncSummarizeAndNotify(ev, targetSessionID, sess)
				updated, ok := s.conversations.GetEvent(targetSessionID, ev.ID)
				if !ok {
					updated = ev
				}
				s.preSynthesizeTTS(updated, targetSessionID)
			}(event)
		} else {
			go s.preSynthesizeTTS(event, targetSessionID)
		}
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

// asyncSummarizeAndNotify runs summarization in a goroutine and, on success,
// sends a conversation_event_update to all subscribed clients so the frontend
// can display the summarized version without a page refresh. On success it
// also evicts any cached TTS audio for this event so playback regenerates
// from the summary rather than the original text.
func (s *Server) asyncSummarizeAndNotify(event ConversationEvent, sessionID string, sess *Session) {
	if s.ttsSummarization == nil {
		return
	}

	cfg := s.getTTSSummarizeConfig()
	if !cfg.Enabled {
		return
	}

	if len(event.Text) < cfg.CharThreshold {
		logSummarizeSkipped("auto", cfg, event.ID, len(event.Text),
			fmt.Sprintf("text length %d < threshold %d", len(event.Text), cfg.CharThreshold))
		return
	}

	normalized := NormalizeTextForSpeech(event.Text)
	if strings.TrimSpace(normalized) == "" {
		logSummarizeSkipped("auto", cfg, event.ID, len(event.Text), "normalized text is empty")
		return
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := s.ttsSummarization.Summarize(ctx, TTSSummarizeRequest{
		EventID: event.ID,
		Path:    "auto",
		Text:    normalized,
	})
	if err != nil {
		if err == errTTSSummarizeCoolingDown {
			logSummarizeSkipped("auto", cfg, event.ID, len(normalized), err.Error())
			return
		}
		logSummarizeResult("auto", cfg, event.ID, len(normalized), result, err)
		// Notify connected clients so they can surface a persistent banner
		// with a retry affordance. Reuse the event payload (paragraphs are
		// unchanged) and mark it as an update carrying the error string.
		errEvent := event
		errEvent.IsUpdate = true
		errEvent.SummarizeError = summarizeErrorMessage(err)
		sess.SendConversation(errEvent)
		return
	}
	logSummarizeResult("auto", cfg, event.ID, len(normalized), result, nil)

	newParagraphs := result.Paragraphs
	s.conversations.UpdateSpeechParagraphs(sessionID, event.ID, newParagraphs)
	s.invalidateTTSCacheForEvent(event.ID)

	// Send update event so connected clients can display the summary.
	event.OriginalSpeechParagraphs = event.SpeechParagraphs
	event.SpeechParagraphs = newParagraphs
	event.Summarized = true
	event.IsUpdate = true
	sess.SendConversation(event)
}

// logSummarizeResult emits the unified tts-summarize log line. It is shared by
// the auto (append) and on-demand code paths so a single grep surfaces both.
// Diagnostic fields (done_reason / eval_count / raw length) are appended on
// failure so budget-exhausted truncation is distinguishable from a real empty
// response without re-running the request.
func logSummarizeResult(path string, cfg TTSSummarizeConfig, eventID string, inChars int, result TTSSummarizeResult, err error) {
	outChars := len(result.Summary)
	ratio := 0.0
	if inChars > 0 {
		ratio = float64(outChars) / float64(inChars)
	}
	if err != nil {
		log.Printf("tts-summarize: path=%s event=%s model=%s level=%s in=%d out=%d ratio=%.2f ms=%d done_reason=%s eval=%d raw=%d error=%v",
			path, eventID, cfg.Model, cfg.Level, inChars, outChars, ratio, result.ElapsedMs,
			safeDoneReason(result.DoneReason), result.EvalCount, result.RawLen, err)
		return
	}
	log.Printf("tts-summarize: path=%s event=%s model=%s level=%s in=%d out=%d ratio=%.2f ms=%d done_reason=%s eval=%d",
		path, eventID, cfg.Model, cfg.Level, inChars, outChars, ratio, result.ElapsedMs,
		safeDoneReason(result.DoneReason), result.EvalCount)
}

// safeDoneReason returns "-" when Ollama didn't emit one (e.g. request
// failed before the response was parsed) so the log line never has an empty
// value that would confuse a grep.
func safeDoneReason(r string) string {
	if r == "" {
		return "-"
	}
	return r
}

// logSummarizeSkipped records the no-op reason in the same grep-friendly shape.
func logSummarizeSkipped(path string, cfg TTSSummarizeConfig, eventID string, inChars int, reason string) {
	log.Printf("tts-summarize: path=%s event=%s model=%s level=%s in=%d out=0 ratio=0.00 ms=0 skipped=%q",
		path, eventID, cfg.Model, cfg.Level, inChars, reason)
}
