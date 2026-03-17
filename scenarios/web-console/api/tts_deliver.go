package main

import (
	"fmt"
	"sync"
	"time"
)

const (
	// ttsMatchNeedleLen is the number of leading characters used to match
	// TTS text against the session's output history. Long enough to avoid
	// accidental matches on common short phrases, short enough to tolerate
	// tail differences from terminal line-wrapping or truncation.
	ttsMatchNeedleLen = 200

	// ttsDeliveryDedupTTL is how long a delivered TTS text is remembered
	// to prevent duplicate delivery when both the hook and tailer paths
	// fire for the same assistant response.
	ttsDeliveryDedupTTL = 30 * time.Second
)

// ttsDedup tracks recently delivered TTS texts to prevent duplicates.
// Both the Claude Code hook and CodexTailer can fire for the same
// assistant response; the dedup cache ensures only the first delivery
// reaches subscribers.
type ttsDedup struct {
	mu   sync.Mutex
	seen map[string]time.Time
	ttl  time.Duration // injectable for testing; 0 uses ttsDeliveryDedupTTL
}

func newTTSDedup() *ttsDedup {
	return &ttsDedup{seen: make(map[string]time.Time)}
}

// TTSDeliveryResult captures the outcome of a delivery attempt so the UI and
// logs can explain why auto-TTS did or did not happen.
type TTSDeliveryResult struct {
	Delivered         bool   `json:"delivered"`
	Code              string `json:"code"`
	Reason            string `json:"reason"`
	Source            string `json:"source"`
	SessionID         string `json:"sessionId,omitempty"`
	UsedTargetSession bool   `json:"usedTargetSession"`
	Duplicate         bool   `json:"duplicate,omitempty"`
}

func ttsDeliveryFailure(code, reason, source, sessionID string, usedTarget bool) TTSDeliveryResult {
	return TTSDeliveryResult{
		Delivered:         false,
		Code:              code,
		Reason:            reason,
		Source:            source,
		SessionID:         sessionID,
		UsedTargetSession: usedTarget,
	}
}

// dedupKey returns the cache key for the given cleaned text.
// Uses the same leading-character window as the match needle.
func dedupKey(text string) string {
	if len(text) > ttsMatchNeedleLen {
		return text[:ttsMatchNeedleLen]
	}
	return text
}

// isDuplicate returns true if text was already delivered within the TTL
// window. If not a duplicate, it records the text for future checks.
// Lazy-evicts stale entries on each call.
func (d *ttsDedup) isDuplicate(text string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	ttl := d.ttl
	if ttl == 0 {
		ttl = ttsDeliveryDedupTTL
	}
	now := time.Now()

	// Lazy eviction of stale entries.
	for k, t := range d.seen {
		if now.Sub(t) > ttl {
			delete(d.seen, k)
		}
	}

	key := dedupKey(text)
	if _, ok := d.seen[key]; ok {
		return true
	}
	d.seen[key] = now
	return false
}

// deliverTTS attempts to deliver responseText to a terminal session.
// If targetSessionID is empty, it falls back to the active pane.
func (s *Server) deliverTTS(responseText, targetSessionID, source string) TTSDeliveryResult {
	if !s.getTTSConfig().AutoEnabled {
		result := ttsDeliveryFailure("tts_auto_disabled", "Auto-TTS is disabled", source, "", false)
		s.recordLastTTSDelivery(result)
		return result
	}

	sessionID := targetSessionID
	usedTarget := targetSessionID != ""
	if sessionID == "" {
		layout, err := s.workspace.GetLayout()
		if err != nil || layout.ActivePane == "" {
			result := ttsDeliveryFailure("tts_delivery_target_missing", "No active terminal session is available for TTS delivery", source, "", false)
			s.recordLastTTSDelivery(result)
			return result
		}
		sessionID = layout.ActivePane
	}

	sess, ok := s.sessions.Get(sessionID)
	if !ok {
		reason := fmt.Sprintf("Target terminal session %s is not available", sanitizeID(sessionID))
		result := ttsDeliveryFailure("tts_delivery_target_missing", reason, source, sessionID, usedTarget)
		s.recordLastTTSDelivery(result)
		return result
	}

	clean := string(stripANSI([]byte(responseText)))
	if !sess.ContainsRecentText(clean, ttsMatchNeedleLen) {
		result := ttsDeliveryFailure("tts_correlation_failed", "Assistant text did not match the session's recent terminal output", source, sessionID, usedTarget)
		s.recordLastTTSDelivery(result)
		return result
	}
	if s.ttsDedup.isDuplicate(clean) {
		result := TTSDeliveryResult{
			Delivered:         true,
			Code:              "tts_duplicate",
			Reason:            "TTS text was already delivered recently",
			Source:            source,
			SessionID:         sessionID,
			UsedTargetSession: usedTarget,
			Duplicate:         true,
		}
		s.recordLastTTSDelivery(result)
		return result
	}
	sess.SendTTS(clean)
	result := TTSDeliveryResult{
		Delivered:         true,
		Code:              "tts_delivered",
		Reason:            "TTS text was delivered to the terminal session",
		Source:            source,
		SessionID:         sessionID,
		UsedTargetSession: usedTarget,
	}
	s.recordLastTTSDelivery(result)
	return result
}
