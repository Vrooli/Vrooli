package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// tts_hook_status.go owns the web-console-internal /api/v1/tts-hook/* REST
// endpoints. These cover the Claude-hook + Codex-tailer routing diagnostics
// and the small auto/backend/startMuted preference triple. They are
// intentionally NOT a proto/Connect service — see ttsHook.ts in the UI for
// the matching RESTReasonHostHookGlue rationale.
//
// All audio synthesis, voice listing, summarize knobs flow through audio-tools
// via the audio-integration UI module; this file knows nothing about them.

// ttsRoutingResultDTO mirrors ConversationAppendResult on the wire.
type ttsRoutingResultDTO struct {
	Appended  bool   `json:"appended"`
	Code      string `json:"code"`
	Reason    string `json:"reason"`
	Source    string `json:"source"`
	SessionID string `json:"sessionId,omitempty"`
	EventID   string `json:"eventId,omitempty"`
	Sequence  int64  `json:"sequence,omitempty"`
	Duplicate bool   `json:"duplicate,omitempty"`
}

func dtoFromAppendResult(r *ConversationAppendResult) *ttsRoutingResultDTO {
	if r == nil {
		return nil
	}
	return &ttsRoutingResultDTO{
		Appended:  r.Appended,
		Code:      r.Code,
		Reason:    r.Reason,
		Source:    r.Source,
		SessionID: r.SessionID,
		EventID:   r.EventID,
		Sequence:  r.Sequence,
		Duplicate: r.Duplicate,
	}
}

// ttsClientAckDTO mirrors TTSClientAck on the wire.
type ttsClientAckDTO struct {
	EventID   string `json:"eventId"`
	Source    string `json:"source"`
	SessionID string `json:"sessionId"`
	Stage     string `json:"stage"`
	Backend   string `json:"backend,omitempty"`
	Message   string `json:"message,omitempty"`
}

func dtoFromAck(a *TTSClientAck) *ttsClientAckDTO {
	if a == nil {
		return nil
	}
	return &ttsClientAckDTO{
		EventID:   a.EventID,
		Source:    a.Source,
		SessionID: a.SessionID,
		Stage:     a.Stage,
		Backend:   a.Backend,
		Message:   a.Message,
	}
}

// ttsPlaybackEventDTO mirrors TTSPlaybackEvent on the wire.
type ttsPlaybackEventDTO struct {
	Source    string `json:"source"`
	Stage     string `json:"stage"`
	Backend   string `json:"backend,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Message   string `json:"message,omitempty"`
}

func dtoFromPlayback(p *TTSPlaybackEvent) *ttsPlaybackEventDTO {
	if p == nil {
		return nil
	}
	return &ttsPlaybackEventDTO{
		Source:    p.Source,
		Stage:     p.Stage,
		Backend:   p.Backend,
		SessionID: p.SessionID,
		Message:   p.Message,
	}
}

// ttsHookStatusDTO is the GET /api/v1/tts-hook/status response shape. The UI
// reads this once on settings open + after each persisted config change.
type ttsHookStatusDTO struct {
	Config                    TTSHookConfig        `json:"config"`
	HookRegistered            bool                 `json:"hookRegistered"`
	HookCode                  string               `json:"hookCode,omitempty"`
	HookReason                string               `json:"hookReason"`
	HookSettingsPath          string               `json:"hookSettingsPath,omitempty"`
	LastHookRouting           *ttsRoutingResultDTO `json:"lastHookRouting,omitempty"`
	LastHookRoutingAt         string               `json:"lastHookRoutingAt,omitempty"`
	LastTailerRouting         *ttsRoutingResultDTO `json:"lastTailerRouting,omitempty"`
	LastTailerRoutingAt       string               `json:"lastTailerRoutingAt,omitempty"`
	LastHookAck               *ttsClientAckDTO     `json:"lastHookAck,omitempty"`
	LastHookAckAt             string               `json:"lastHookAckAt,omitempty"`
	LastTailerAck             *ttsClientAckDTO     `json:"lastTailerAck,omitempty"`
	LastTailerAckAt           string               `json:"lastTailerAckAt,omitempty"`
	LastPlaybackEvent         *ttsPlaybackEventDTO `json:"lastPlaybackEvent,omitempty"`
	LastPlaybackAt            string               `json:"lastPlaybackAt,omitempty"`
	AudioToolsCapability      string               `json:"audioToolsCapability,omitempty"`
	AudioToolsCapabilityLabel string               `json:"audioToolsCapabilityLabel,omitempty"`
}

func formatTimeRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func (s *Server) handleTTSHookStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	hookRegistered, hookCode, hookReason, hookSettingsPath := s.getClaudeHookStatus()

	hookRouting, hookRoutingAt := s.getLastTTSRoutingBySource("claude_hook")
	tailerRouting, tailerRoutingAt := s.getLastTTSRoutingBySource("codex_tailer")
	hookAck, hookAckAt := s.getLastTTSAckBySource("claude_hook")
	tailerAck, tailerAckAt := s.getLastTTSAckBySource("codex_tailer")
	playback, playbackAt := s.getLastTTSPlaybackEvent()

	// Audio-tools capability label is best-effort — settings UI is happy
	// to render an empty label when capabilities aren't wired (tests).
	var audioStatus, audioLabel string
	if s.capabilities != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		for _, cap := range s.capabilities.ResolveLiveness(ctx) {
			if cap.ID == "audio-tools" {
				audioStatus = string(cap.Status)
				audioLabel = strings.TrimSpace(cap.Message)
				if audioLabel == "" {
					audioLabel = "audio-tools"
				}
				break
			}
		}
	}

	resp := ttsHookStatusDTO{
		Config:                    s.getTTSHookConfig(),
		HookRegistered:            hookRegistered,
		HookCode:                  hookCode,
		HookReason:                hookReason,
		HookSettingsPath:          hookSettingsPath,
		LastHookRouting:           dtoFromAppendResult(hookRouting),
		LastHookRoutingAt:         formatTimeRFC3339(hookRoutingAt),
		LastTailerRouting:         dtoFromAppendResult(tailerRouting),
		LastTailerRoutingAt:       formatTimeRFC3339(tailerRoutingAt),
		LastHookAck:               dtoFromAck(hookAck),
		LastHookAckAt:             formatTimeRFC3339(hookAckAt),
		LastTailerAck:             dtoFromAck(tailerAck),
		LastTailerAckAt:           formatTimeRFC3339(tailerAckAt),
		LastPlaybackEvent:         dtoFromPlayback(playback),
		LastPlaybackAt:            formatTimeRFC3339(playbackAt),
		AudioToolsCapability:      audioStatus,
		AudioToolsCapabilityLabel: audioLabel,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTTSHookConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var patch TTSHookConfigPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	updated := patch.Apply(s.getTTSHookConfig())
	if err := s.setTTSHookConfig(updated); err != nil {
		log.Printf("tts-hook-config: persist failed: %v", err)
		http.Error(w, "persist failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleTTSHookAck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var ack ttsClientAckDTO
	if err := json.NewDecoder(r.Body).Decode(&ack); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(ack.Source) == "" || strings.TrimSpace(ack.Stage) == "" {
		http.Error(w, "ack requires source + stage", http.StatusBadRequest)
		return
	}
	s.recordTTSAck(TTSClientAck(ack))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTTSHookPlayback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var ev ttsPlaybackEventDTO
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(ev.Source) == "" || strings.TrimSpace(ev.Stage) == "" {
		http.Error(w, "playback event requires source + stage", http.StatusBadRequest)
		return
	}
	s.recordTTSPlaybackEvent(TTSPlaybackEvent(ev))
	w.WriteHeader(http.StatusNoContent)
}

// registerTTSHookRoutes wires the four REST endpoints onto the mux router.
func (s *Server) registerTTSHookRoutes() {
	s.router.HandleFunc("/api/v1/tts-hook/status", s.handleTTSHookStatus).Methods(http.MethodGet)
	s.router.HandleFunc("/api/v1/tts-hook/config", s.handleTTSHookConfig).Methods(http.MethodPut)
	s.router.HandleFunc("/api/v1/tts-hook/ack", s.handleTTSHookAck).Methods(http.MethodPost)
	s.router.HandleFunc("/api/v1/tts-hook/playback", s.handleTTSHookPlayback).Methods(http.MethodPost)
}
