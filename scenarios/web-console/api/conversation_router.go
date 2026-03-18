package main

import "strings"

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
		sess.SendConversation(event)
	}
	s.recordLastTTSRouting(result)
	return result
}
