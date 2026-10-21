package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"web-console/backends/codex"
	"web-console/internal/sessionstore"
)

// agentVoiceLease grants one connection control of a session's audio, not of
// the agent's tools or approvals. Other connections remain observers.
type agentVoiceLease struct {
	owned      atomic.Bool
	active     atomic.Bool
	realtimeID atomic.Value
}
type agentVoiceCapability struct {
	Available    bool   `json:"available"`
	Retryable    bool   `json:"retryable,omitempty"`
	Agent        string `json:"agent"`
	Version      string `json:"version,omitempty"`
	Reason       string `json:"reason,omitempty"`
	NativeActive bool   `json:"nativeActive"`
}
type agentVoiceCommand struct {
	Type      string               `json:"type"`
	Transport string               `json:"transport,omitempty"`
	SDP       string               `json:"sdp,omitempty"`
	Audio     *codex.RealtimeAudio `json:"audio,omitempty"`
}
type agentVoiceBinding struct {
	client    *codex.Client
	threadID  string
	close     func()
	subscribe bool
	observe   func() bool
}

func (s *Server) resolveAgentVoice(ctx context.Context, id string) (agentVoiceCapability, *agentVoiceBinding, error) {
	cap := agentVoiceCapability{Reason: "Agent voice is unavailable; use Dictation."}
	if id == "" || filepath.Base(id) != id || s.sessionStore == nil {
		return cap, nil, errors.New("session not found")
	}
	meta, err := s.sessionStore.Get(ctx, id)
	if err != nil {
		return cap, nil, err
	}
	cap.Agent = string(meta.AgentType)
	cap.Version = meta.ProviderVersion
	if !meta.ArchivedAt.IsZero() {
		return cap, nil, errors.New("session is archived")
	}
	if meta.AgentType != sessionstore.AgentCodex {
		cap.Retryable = meta.AgentType == sessionstore.AgentNone || cap.Agent == "" || cap.Agent == "unknown"
		return cap, nil, nil
	}
	cap.Retryable = true // Session identity/daemon may still be starting.
	if s.sessions != nil {
		if sess, ok := s.sessions.Get(id); ok && sess != nil {
			if view := sess.ActivityScreen(); view != nil {
				cap.NativeActive = codex.ObserveVoice(view).Active
			}
		}
	}
	if s.managedCodex != nil {
		s.managedCodex.mu.RLock()
		entry, ok := s.managedCodex.entries[id]
		s.managedCodex.mu.RUnlock()
		if ok && entry.owner != nil {
			if version := entry.owner.ProviderVersion(); version != "" {
				cap.Version = version
			}
			if !codex.SupportsBrowserVoice(cap.Version) {
				cap.Retryable = false
				cap.Reason = "This Codex version has not been qualified for browser voice; use Dictation."
				return cap, nil, nil
			}
			cap.Available = true
			cap.Retryable = false
			cap.Reason = ""
			return cap, &agentVoiceBinding{client: entry.owner.VoiceClient(), threadID: entry.owner.ThreadID(), close: func() {}}, nil
		}
	}
	socket := filepath.Join(codex.SessionHomePath(resolveSessionStateRoot(), id), "app-server-control", "app-server-control.sock")
	// Absence is capability unavailability; never start a daemon as a side effect.
	if info, err := os.Stat(socket); err != nil || info.Mode()&os.ModeSocket == 0 {
		return cap, nil, nil
	}
	client, err := codex.DialVoiceDaemon(ctx, socket)
	if err != nil {
		cap.Reason = "Codex voice connection is unavailable; use Dictation."
		return cap, nil, nil
	}
	binding := &agentVoiceBinding{client: client, subscribe: true, close: func() { _ = client.Close() }}
	if s.sessions != nil {
		binding.observe = func() bool {
			if sess, ok := s.sessions.Get(id); ok && sess != nil {
				if view := sess.ActivityScreen(); view != nil {
					return codex.ObserveVoice(view).Active
				}
			}
			return false
		}
	}
	fail := func() (agentVoiceCapability, *agentVoiceBinding, error) { binding.close(); return cap, nil, nil }
	threadID := meta.NativeThreadID
	if threadID == "" {
		threadID = meta.AgentSessionID
	}
	var loaded struct {
		Data []string `json:"data"`
	}
	if client.Call(ctx, "thread/loaded/list", map[string]any{}, &loaded) != nil {
		return fail()
	}
	threadID = loadedVoiceThread(threadID, loaded.Data)
	if threadID == "" {
		return fail()
	}
	var read struct {
		Thread struct {
			ID  string `json:"id"`
			CWD string `json:"cwd"`
		} `json:"thread"`
	}
	if client.Call(ctx, "thread/read", map[string]any{"threadId": threadID}, &read) != nil || read.Thread.ID != threadID || (meta.CWD != "" && filepath.Clean(read.Thread.CWD) != filepath.Clean(meta.CWD)) {
		return fail()
	}
	binding.threadID = threadID
	cap.Version = client.ServerVersion()
	if !codex.SupportsBrowserVoice(cap.Version) {
		cap.Retryable = false
		cap.Reason = "This Codex version has not been qualified for browser voice; use Dictation."
		return fail()
	}
	cap.Available = true
	cap.Retryable = false
	cap.Reason = ""
	return cap, binding, nil
}

// A persisted thread identity alone must never cause a cold resume. Voice
// attaches only to a currently loaded thread of this session's private daemon.
func loadedVoiceThread(recorded string, loaded []string) string {
	if recorded == "" {
		if len(loaded) == 1 {
			return loaded[0]
		}
		return ""
	}
	for _, id := range loaded {
		if id == recorded {
			return id
		}
	}
	return ""
}

func sameVoiceOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}
	// The canonical UI gateway rewrites Host to the API port. Direct desktop
	// connections use the same configured UI origins as Server.Handler's CORS.
	uiPort := getEnvOrDefault("UI_PORT", "36233")
	if origin == "http://localhost:"+uiPort || origin == "http://127.0.0.1:"+uiPort {
		return true
	}
	// Only the local gateway may attest an external original host. Browser
	// clients cannot supply this header through the WebSocket constructor.
	peer, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip := net.ParseIP(peer); ip != nil && ip.IsLoopback() {
		forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
		return forwarded != "" && !strings.Contains(forwarded, ",") && strings.EqualFold(u.Host, forwarded)
	}
	return false
}

func (s *Server) acquireAgentVoice(id string, lease *agentVoiceLease) bool {
	s.agentVoiceMu.Lock()
	defer s.agentVoiceMu.Unlock()
	if s.agentVoiceControllers == nil {
		s.agentVoiceControllers = make(map[string]*agentVoiceLease)
	}
	if current := s.agentVoiceControllers[id]; current != nil && current != lease {
		return false
	}
	s.agentVoiceControllers[id] = lease
	return true
}
func (s *Server) releaseAgentVoice(id string, lease *agentVoiceLease) {
	s.agentVoiceMu.Lock()
	defer s.agentVoiceMu.Unlock()
	if s.agentVoiceControllers[id] == lease {
		delete(s.agentVoiceControllers, id)
	}
}

func validateAgentVoiceAudio(a *codex.RealtimeAudio) error {
	if a == nil || a.SampleRate != 24000 || a.NumChannels != 1 || len(a.Data) > 65536 {
		return errors.New("expected bounded 24 kHz mono PCM16 audio")
	}
	data, err := base64.StdEncoding.DecodeString(a.Data)
	if err != nil || len(data) == 0 || len(data)%2 != 0 || len(data) > 24000 {
		return errors.New("invalid PCM16 audio chunk")
	}
	if a.SamplesPerChannel != nil && *a.SamplesPerChannel != len(data)/2 {
		return errors.New("audio sample count mismatch")
	}
	return nil
}

// handleAgentVoiceWS keeps media/control outside terminal stdin. Every native
// method is selected here; the browser cannot proxy arbitrary JSON-RPC calls.
func (s *Server) handleAgentVoiceWS(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	cap, binding, err := s.resolveAgentVoice(ctx, id)
	cancel()
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if binding != nil {
		defer binding.close()
	}
	conn, err := (&websocket.Upgrader{CheckOrigin: sameVoiceOrigin}).Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(128 * 1024)
	var writeMu sync.Mutex
	write := func(value any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		return conn.WriteJSON(value)
	}
	failure := func(code, message string) {
		_ = write(map[string]any{"type": "error", "code": code, "message": message})
	}
	if write(map[string]any{"type": "capability", "capability": cap}) != nil {
		return
	}
	if binding == nil {
		return
	}
	notifications, unsubscribe := binding.client.SubscribeNotifications()
	defer unsubscribe()
	lease := &agentVoiceLease{}
	defer func() {
		if lease.owned.Swap(false) {
			stopCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
			defer done()
			_ = binding.client.Call(stopCtx, "thread/realtime/stop", map[string]string{"threadId": binding.threadID}, nil)
		}
		s.releaseAgentVoice(id, lease)
	}()
	// Subscribe to the already-loaded thread on the same daemon. No turn is
	// started and no settings/approval policy is supplied or replaced.
	if binding.subscribe {
		subCtx, done := context.WithTimeout(r.Context(), 8*time.Second)
		// Rejoin the verified running thread without hydrating on-disk turns.
		// A fresh terminal thread may not have a rollout until its first turn.
		err = binding.client.Call(subCtx, "thread/resume", map[string]any{"threadId": binding.threadID, "excludeTurns": true}, nil)
		done()
		if err != nil && !strings.Contains(err.Error(), "no rollout found for thread id") {
			failure("subscription_failed", "Could not observe Codex voice; use Dictation.")
			return
		}
		// Codex 0.161 cannot resume an unsaved, already-running thread. Keep
		// observing its footer; an explicit realtime/start attaches its own
		// listener to that verified loaded thread without a disk resume.
	}
	finished := make(chan struct{})
	defer close(finished)
	footerTicker := time.NewTicker(time.Second)
	defer footerTicker.Stop()
	go func() {
		for {
			select {
			case <-finished:
				return
			case <-footerTicker.C:
				if binding.observe != nil && !lease.owned.Load() {
					if active := binding.observe(); active != cap.NativeActive {
						cap.NativeActive = active
						_ = write(map[string]any{"type": "capability", "capability": cap})
					}
				}
			case message, ok := <-notifications:
				if !ok {
					_ = conn.Close()
					return
				}
				n, realtime := codex.ParseRealtime(message)
				if !realtime || n.ThreadID != binding.threadID {
					continue
				}
				if message.Method == "thread/realtime/started" && lease.owned.Load() {
					expected, _ := lease.realtimeID.Load().(string)
					if n.RealtimeSessionID == expected {
						lease.active.Store(true)
					} else if lease.active.Load() {
						lease.owned.Store(false)
						lease.active.Store(false)
						s.releaseAgentVoice(id, lease)
						_ = write(map[string]string{"type": "replaced"})
					}
				}
				if message.Method == "thread/realtime/closed" && lease.active.Swap(false) {
					lease.owned.Store(false)
					s.releaseAgentVoice(id, lease)
					_ = write(map[string]string{"type": "stopped"})
				}
				if message.Method == "thread/realtime/error" {
					if strings.Contains(n.Message, "requires API key auth") {
						failure("native_auth_required", "This Codex transport requires API-key authentication. Select WebRTC or Dictation.")
					} else {
						failure("native_voice_failed", "Codex voice encountered an error. Reconnect or use Dictation.")
					}
					continue
				}
				if message.Method == "thread/realtime/item/completed" && n.Item != nil {
					s.captureAgentVoiceItem(id, binding.threadID, *n.Item)
				}
				if (message.Method == "thread/realtime/outputAudio/delta" || message.Method == "thread/realtime/sdp") && !lease.owned.Load() {
					continue
				}
				if write(map[string]any{"type": "event", "method": message.Method, "params": message.Params}) != nil {
					_ = conn.Close()
					return
				}
			}
		}
	}()
	for {
		_ = conn.SetReadDeadline(time.Now().Add(45 * time.Second))
		var command agentVoiceCommand
		if conn.ReadJSON(&command) != nil {
			return
		}
		if command.Type == "ping" {
			if write(map[string]string{"type": "pong"}) != nil {
				return
			}
			continue
		}
		if command.Type == "start" {
			if lease.owned.Load() {
				failure("already_connected", "Agent voice is already connected.")
				continue
			}
			if command.Transport != "webrtc" && command.Transport != "websocket" {
				failure("invalid_transport", "Unsupported voice transport.")
				continue
			}
			if command.Transport == "webrtc" && (len(command.SDP) < 5 || len(command.SDP) > 96000) {
				failure("invalid_sdp", "Missing or invalid voice offer.")
				continue
			}
			if !s.acquireAgentVoice(id, lease) {
				failure("voice_in_use", "Another browser controls this session's voice.")
				continue
			}
			callCtx, done := context.WithTimeout(r.Context(), 25*time.Second)
			// Explicit connection transfers voice from the terminal's host device to
			// this browser. It never resumes/restarts the coding task.
			err = binding.client.Call(callCtx, "thread/realtime/stop", map[string]string{"threadId": binding.threadID}, nil)
			if err == nil {
				realtimeID := uuid.NewString()
				lease.realtimeID.Store(realtimeID)
				lease.active.Store(false)
				transport := map[string]any{"type": command.Transport}
				if command.Transport == "webrtc" {
					transport["sdp"] = command.SDP
				}
				lease.owned.Store(true)
				// Canonical transcript items are committed independently. Ending
				// voice must not submit a synthetic transcript-tail coding turn.
				err = binding.client.Call(callCtx, "thread/realtime/start", map[string]any{
					"threadId": binding.threadID, "realtimeSessionId": realtimeID,
					"transport": transport, "version": "v3", "outputModality": "audio",
					"clientManagedHandoffs": false, "codexResponsesAsItems": false,
					"includeStartupContext": false, "flushTranscriptTailOnSessionEnd": false,
				}, nil)
			}
			done()
			if err != nil {
				// A timed-out start may have reached the provider. Clean up the
				// uncertain conversation before relinquishing browser ownership.
				if lease.owned.Swap(false) {
					cleanupCtx, cleanupDone := context.WithTimeout(context.Background(), 5*time.Second)
					_ = binding.client.Call(cleanupCtx, "thread/realtime/stop", map[string]string{"threadId": binding.threadID}, nil)
					cleanupDone()
				}
				s.releaseAgentVoice(id, lease)
				failure("native_start_failed", "Codex voice could not start. Use Dictation or retry.")
				continue
			}
			if !lease.owned.Load() {
				failure("native_closed", "Codex voice ended before the media connection was ready. Reconnect or use Dictation.")
				continue
			}
			_ = write(map[string]string{"type": "accepted"})
			continue
		}
		if !lease.owned.Load() {
			failure("not_controller", "Connect agent voice before sending audio or control.")
			continue
		}
		callCtx, done := context.WithTimeout(r.Context(), 8*time.Second)
		switch command.Type {
		case "audio":
			err = validateAgentVoiceAudio(command.Audio)
			if err == nil {
				err = binding.client.Call(callCtx, "thread/realtime/appendAudio", map[string]any{"threadId": binding.threadID, "audio": command.Audio}, nil)
			}
		case "stop":
			err = binding.client.Call(callCtx, "thread/realtime/stop", map[string]string{"threadId": binding.threadID}, nil)
			if err == nil {
				lease.owned.Store(false)
				s.releaseAgentVoice(id, lease)
				_ = write(map[string]string{"type": "stopped"})
			}
		default:
			err = fmt.Errorf("unsupported voice command")
		}
		done()
		if err != nil {
			failure("native_voice_failed", "Agent voice stopped because its media connection failed.")
			return
		}
	}
}

func (s *Server) captureAgentVoiceItem(sessionID, threadID string, item codex.RealtimeItem) {
	if s.conversations == nil || item.Type != "transcriptSegment" || item.ID == "" || item.RealtimeSessionID == "" || (item.Role != "user" && item.Role != "assistant") {
		return
	}
	event, result := s.conversations.AppendNativeEvent(context.Background(), sessionID, "codex_voice", item.Role, item.Text, &NativeProvenance{Provider: "codex", SessionID: threadID, MessageID: item.ID, BoundaryID: item.RealtimeSessionID})
	if result.Appended && !result.Duplicate {
		s.publishConversationEvent(event)
	}
}
