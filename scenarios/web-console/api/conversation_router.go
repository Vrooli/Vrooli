package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"web-console/integrations/audiotools"
	"web-console/internal/audioports"
)

// ConversationDispatcher publishes trusted assistant and user conversation
// events to a terminal session. *Server implements it via AppendAssistant
// and AppendUser.
//
// The interface lets non-Server callers (hook handlers, tailers, future
// adapters) depend on a narrow surface instead of the whole Server god
// object, and lets tests substitute a fake dispatcher.
//
// DOC: docs/internal/SEAMS.md#conversation-dispatcher
type ConversationDispatcher interface {
	AppendAssistant(responseText, sessionID, source string) ConversationAppendResult
	AppendUser(promptText, sessionID, source string) ConversationAppendResult
}

// Compile-time check that *Server satisfies ConversationDispatcher.
var _ ConversationDispatcher = (*Server)(nil)

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

// SummarizeConfig is the narrow auto-summarize tunable web-console reads to
// decide whether to call audio-tools' Summarize RPC for long assistant
// messages. The full TTSSummarizeConfig (with `model`, `enabled`, `level`,
// etc.) lives in audio-tools; we cache only the fields the auto-path needs.
//
// CharThreshold == 0 disables auto-summarize (matches the audio-tools
// SummarizeConfig.charThreshold default semantics).
type SummarizeAutoPolicy struct {
	Enabled        bool
	CharThreshold  int
	Level          string
	TimeoutSeconds int
}

// defaultSummarizeAutoPolicy is the conservative default applied when the
// audio-tools Summarize knobs haven't been fetched yet.
func defaultSummarizeAutoPolicy() SummarizeAutoPolicy {
	return SummarizeAutoPolicy{
		Enabled:        false,
		CharThreshold:  500,
		Level:          "moderate",
		TimeoutSeconds: 120,
	}
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

// AppendAssistant validates and publishes a trusted assistant response to
// the owning terminal session. Auto-TTS enablement is not checked here:
// the conversation log is the source of truth and TTS is only one consumer.
// Satisfies ConversationDispatcher.
func (s *Server) AppendAssistant(responseText, targetSessionID, source string) ConversationAppendResult {
	if strings.TrimSpace(targetSessionID) == "" {
		result := conversationAppendFailure("conversation_target_missing", "No web-console terminal session was identified for this conversation event", source, "")
		s.recordLastTTSRouting(result)
		return result
	}

	if _, ok := s.sessions.Get(targetSessionID); !ok {
		result := conversationAppendFailure("conversation_target_missing", "The mapped terminal session is no longer available", source, targetSessionID)
		s.recordLastTTSRouting(result)
		return result
	}

	event, result := s.conversations.AppendAssistantEvent(targetSessionID, source, responseText)
	if result.Appended && !result.Duplicate {
		// Publish to the global event channel immediately so the text lands in
		// every subscribed client with no delay — audio is handled separately
		// below.
		s.publishConversationEvent(event)

		policy := s.getSummarizeAutoPolicy()
		shouldSummarize := s.summarizer != nil && policy.Enabled && len(event.Text) >= policy.CharThreshold
		if shouldSummarize {
			// Summarize-first path: wait for the audio-tools Summarize call
			// to finish before notifying clients of the summarized version.
			go func(ev ConversationEvent) {
				s.asyncSummarizeAndNotify(ev, targetSessionID, policy)
			}(event)
		}
	}
	s.recordLastTTSRouting(result)
	return result
}

// AppendUser validates and publishes a user prompt to the owning terminal
// session. No TTS routing is performed for user messages. Satisfies
// ConversationDispatcher.
func (s *Server) AppendUser(promptText, targetSessionID, source string) ConversationAppendResult {
	if strings.TrimSpace(targetSessionID) == "" {
		return conversationAppendFailure("conversation_target_missing", "No web-console terminal session was identified for this conversation event", source, "")
	}

	if _, ok := s.sessions.Get(targetSessionID); !ok {
		return conversationAppendFailure("conversation_target_missing", "The mapped terminal session is no longer available", source, targetSessionID)
	}

	event, result := s.conversations.AppendUserEvent(targetSessionID, source, promptText)
	if result.Appended && !result.Duplicate {
		s.publishConversationEvent(event)
	}
	return result
}

// asyncSummarizeAndNotify calls audio-tools' Summarize RPC and, on success,
// sends a conversation_event_update so the frontend displays the summarized
// version without a refresh. Cooldown/inflight-dedup are audio-tools'
// concern now; the remote tier handles backpressure. On failure we surface
// a one-off error event with the failure message.
func (s *Server) asyncSummarizeAndNotify(event ConversationEvent, sessionID string, policy SummarizeAutoPolicy) {
	if s.summarizer == nil {
		return
	}
	if !policy.Enabled || len(event.Text) < policy.CharThreshold {
		logSummarizeSkipped("auto", policy, event.ID, len(event.Text),
			fmt.Sprintf("text length %d < threshold %d", len(event.Text), policy.CharThreshold))
		return
	}
	if strings.TrimSpace(event.Text) == "" {
		logSummarizeSkipped("auto", policy, event.ID, len(event.Text), "event text is empty")
		return
	}

	timeout := time.Duration(policy.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	out, err := s.summarizer.Summarize(ctx, audioports.SummarizeInput{
		Text:           event.Text,
		Level:          policy.Level,
		TimeoutSeconds: policy.TimeoutSeconds,
	})
	if err != nil {
		logSummarizeResult("auto", policy, event.ID, len(event.Text), out, err)
		errEvent := event
		errEvent.IsUpdate = true
		errEvent.SummarizeError = summarizeErrorMessage(err)
		s.publishConversationEvent(errEvent)
		return
	}
	logSummarizeResult("auto", policy, event.ID, len(event.Text), out, nil)

	// audio-tools returns a flat summary; re-split into paragraphs for
	// pleasant rendering. Empty input → no update emitted.
	newParagraphs := splitIntoSpeechParagraphs(out.Text)
	if len(newParagraphs) == 0 {
		return
	}
	s.conversations.UpdateSpeechParagraphs(sessionID, event.ID, newParagraphs)

	event.OriginalSpeechParagraphs = event.SpeechParagraphs
	event.SpeechParagraphs = newParagraphs
	event.Summarized = true
	event.IsUpdate = true
	s.publishConversationEvent(event)
}

// splitIntoSpeechParagraphs is the local, dependency-free fallback for
// breaking a summarized blob into paragraphs. audio-tools owns the canonical
// normalize/split pipeline (see RemoteSpeechTextProcessor); this helper is
// only used post-summarize where the remote call has already happened.
func splitIntoSpeechParagraphs(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	raw := strings.Split(text, "\n\n")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return []string{text}
	}
	return out
}

// summarizeErrorMessage normalizes the audio-tools transport error into a
// short UI-facing string. The legacy inttts.SummarizeErrorMessage classified
// Ollama-specific failure modes; the remote tier collapses them into a
// single user-visible "summarization failed" with the wire detail.
func summarizeErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, audiotools.ErrTimeout) {
		return "Summarization timed out before audio-tools returned a result. Try again or increase the summarize timeout in voice settings."
	}
	if errors.Is(err, audiotools.ErrUnavailable) {
		return "Summarization failed: audio-tools is unavailable. Check that audio-tools and its Ollama summarizer are running."
	}
	if errors.Is(err, audiotools.ErrFailedPrecondition) {
		return "Summarization failed: selected Ollama summarizer model is not installed. Choose an installed model in voice settings or run the shown ollama pull command."
	}
	msg := err.Error()
	if msg == "" {
		return "Summarization failed"
	}
	return "Summarization failed: " + msg
}

// logSummarizeResult emits the unified tts-summarize log line. Shared by
// the auto (append) and on-demand code paths so a single grep surfaces both.
func logSummarizeResult(path string, policy SummarizeAutoPolicy, eventID string, inChars int, out audioports.SummarizeOutput, err error) {
	outChars := len(out.Text)
	ratio := 0.0
	if inChars > 0 {
		ratio = float64(outChars) / float64(inChars)
	}
	if err != nil {
		log.Printf("tts-summarize: path=%s event=%s level=%s in=%d out=%d ratio=%.2f tier=%s model=%s error=%v",
			path, eventID, policy.Level, inChars, outChars, ratio, out.ProviderTier, out.ModelID, err)
		return
	}
	log.Printf("tts-summarize: path=%s event=%s level=%s in=%d out=%d ratio=%.2f tier=%s model=%s ms=%d",
		path, eventID, policy.Level, inChars, outChars, ratio, out.ProviderTier, out.ModelID, out.Latency.Milliseconds())
}

// logSummarizeSkipped records the no-op reason in the same grep-friendly shape.
func logSummarizeSkipped(path string, policy SummarizeAutoPolicy, eventID string, inChars int, reason string) {
	log.Printf("tts-summarize: path=%s event=%s level=%s in=%d out=0 ratio=0.00 skipped=%q",
		path, eventID, policy.Level, inChars, reason)
}
