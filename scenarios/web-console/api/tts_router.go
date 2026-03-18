package main

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

const ttsDedupTTL = 30 * time.Second

// TTSCandidate is the unit routed from trusted AI event sources to a single
// terminal session. The browser correlates the candidate against the rendered
// terminal buffer before speaking it.
type TTSCandidate struct {
	EventID   string `json:"eventId"`
	Source    string `json:"source"`
	SessionID string `json:"sessionId"`
	Text      string `json:"text"`
}

// TTSRoutingResult captures backend routing outcomes before browser-side
// correlation or playback begins.
type TTSRoutingResult struct {
	Routed    bool   `json:"routed"`
	Code      string `json:"code"`
	Reason    string `json:"reason"`
	Source    string `json:"source"`
	SessionID string `json:"sessionId,omitempty"`
	EventID   string `json:"eventId,omitempty"`
	Duplicate bool   `json:"duplicate,omitempty"`
}

// TTSClientAck records terminal-side correlation and playback outcomes for a
// routed TTS candidate.
type TTSClientAck struct {
	EventID   string `json:"eventId"`
	Source    string `json:"source"`
	SessionID string `json:"sessionId"`
	Stage     string `json:"stage"`
	Backend   string `json:"backend,omitempty"`
	Message   string `json:"message,omitempty"`
}

type ttsDedup struct {
	mu   sync.Mutex
	seen map[string]time.Time
	ttl  time.Duration
}

func newTTSDedup() *ttsDedup {
	return &ttsDedup{seen: make(map[string]time.Time)}
}

func (d *ttsDedup) seenRecently(key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	ttl := d.ttl
	if ttl == 0 {
		ttl = ttsDedupTTL
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

func newTTSEventID(source, sessionID, text string) string {
	sum := sha1.Sum([]byte(source + "\n" + sessionID + "\n" + text))
	return hex.EncodeToString(sum[:])
}

func ttsRoutingFailure(code, reason, source, sessionID string) TTSRoutingResult {
	return TTSRoutingResult{
		Routed:    false,
		Code:      code,
		Reason:    reason,
		Source:    source,
		SessionID: sessionID,
	}
}

// routeTTSCandidate validates and routes a trusted assistant response to a
// single terminal session. Session ownership is determined by the source
// adapter; the backend no longer attempts to infer visible terminal content
// from PTY history.
func (s *Server) routeTTSCandidate(responseText, targetSessionID, source string) TTSRoutingResult {
	if !s.getTTSConfig().AutoEnabled {
		result := ttsRoutingFailure("tts_auto_disabled", "Auto-TTS is disabled", source, "")
		s.recordLastTTSRouting(result)
		return result
	}
	if strings.TrimSpace(targetSessionID) == "" {
		result := ttsRoutingFailure("tts_target_missing", "No web-console terminal session was identified for this TTS event", source, "")
		s.recordLastTTSRouting(result)
		return result
	}

	sess, ok := s.sessions.Get(targetSessionID)
	if !ok {
		result := ttsRoutingFailure("tts_target_missing", "The mapped terminal session is no longer available", source, targetSessionID)
		s.recordLastTTSRouting(result)
		return result
	}

	text := strings.TrimSpace(string(stripANSI([]byte(responseText))))
	if text == "" {
		result := ttsRoutingFailure("tts_input_required", "TTS event did not include assistant response text", source, targetSessionID)
		s.recordLastTTSRouting(result)
		return result
	}

	eventID := newTTSEventID(source, targetSessionID, text)
	if s.ttsDedup.seenRecently(eventID) {
		result := TTSRoutingResult{
			Routed:    true,
			Code:      "tts_duplicate",
			Reason:    "TTS candidate was already routed recently",
			Source:    source,
			SessionID: targetSessionID,
			EventID:   eventID,
			Duplicate: true,
		}
		s.recordLastTTSRouting(result)
		return result
	}

	sess.SendTTS(TTSCandidate{
		EventID:   eventID,
		Source:    source,
		SessionID: targetSessionID,
		Text:      text,
	})

	result := TTSRoutingResult{
		Routed:    true,
		Code:      "tts_candidate_routed",
		Reason:    "TTS candidate was routed to the mapped terminal session",
		Source:    source,
		SessionID: targetSessionID,
		EventID:   eventID,
	}
	s.recordLastTTSRouting(result)
	return result
}
