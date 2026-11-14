package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"web-console/backends/codex"
	"web-console/internal/sessionstore"
)

func TestAgentVoiceExclusiveLease(t *testing.T) {
	s := &Server{}
	a, b := &agentVoiceLease{}, &agentVoiceLease{}
	if !s.acquireAgentVoice("s", a) || !s.acquireAgentVoice("s", a) || s.acquireAgentVoice("s", b) {
		t.Fatal("exclusive ownership failed")
	}
	s.releaseAgentVoice("s", b)
	if s.acquireAgentVoice("s", b) {
		t.Fatal("observer released controller")
	}
	s.releaseAgentVoice("s", a)
	if !s.acquireAgentVoice("s", b) {
		t.Fatal("lease not released")
	}
}
func TestAgentVoiceOrigin(t *testing.T) {
	t.Setenv("UI_PORT", "21233")
	for _, origin := range []string{"", "https://console.test"} {
		r := httptest.NewRequest("GET", "https://console.test/", nil)
		r.Header.Set("Origin", origin)
		if !sameVoiceOrigin(r) {
			t.Fatal(origin)
		}
	}
	r := httptest.NewRequest("GET", "https://console.test/", nil)
	r.Header.Set("Origin", "https://attacker.test")
	if sameVoiceOrigin(r) {
		t.Fatal("cross origin accepted")
	}
	r.Header.Set("Origin", "http://localhost:21233")
	if !sameVoiceOrigin(r) {
		t.Fatal("configured UI origin rejected behind API gateway")
	}
	r.Header.Set("Origin", "https://console.test")
	r.Host = "localhost:16382"
	r.RemoteAddr = "127.0.0.1:4321"
	r.Header.Set("X-Forwarded-Host", "console.test")
	if !sameVoiceOrigin(r) {
		t.Fatal("trusted local gateway rejected")
	}
	r.RemoteAddr = "203.0.113.1:4321"
	if sameVoiceOrigin(r) {
		t.Fatal("untrusted forwarded host accepted")
	}
}
func TestAgentVoiceAudioBounds(t *testing.T) {
	count := 2
	valid := codex.RealtimeAudio{Data: base64.StdEncoding.EncodeToString([]byte{0, 0, 1, 0}), SampleRate: 24000, NumChannels: 1, SamplesPerChannel: &count}
	if err := validateAgentVoiceAudio(&valid); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*codex.RealtimeAudio){func(a *codex.RealtimeAudio) { a.SampleRate = 48000 }, func(a *codex.RealtimeAudio) { a.NumChannels = 2 }, func(a *codex.RealtimeAudio) { a.Data = "!" }, func(a *codex.RealtimeAudio) { a.Data = base64.StdEncoding.EncodeToString([]byte{0}) }, func(a *codex.RealtimeAudio) { a.Data = strings.Repeat("A", 65540) }, func(a *codex.RealtimeAudio) { n := 3; a.SamplesPerChannel = &n }} {
		a := valid
		mutate(&a)
		if validateAgentVoiceAudio(&a) == nil {
			t.Fatalf("accepted %+v", a)
		}
	}
	if validateAgentVoiceAudio(nil) == nil {
		t.Fatal("nil audio accepted")
	}
}
func TestAgentVoiceUnsupportedDoesNotLaunchProvider(t *testing.T) {
	s := &Server{sessionStore: sessionstore.NewInMemory()}
	_ = s.sessionStore.Save(context.Background(), sessionstore.Metadata{ID: "session", AgentType: sessionstore.AgentClaude})
	cap, binding, err := s.resolveAgentVoice(context.Background(), "session")
	if err != nil || cap.Available || cap.Retryable || binding != nil {
		t.Fatalf("cap=%+v binding=%v err=%v", cap, binding, err)
	}
	if _, _, err = s.resolveAgentVoice(context.Background(), "../escape"); err == nil {
		t.Fatal("unsafe session accepted")
	}
}
func TestAgentVoiceCanonicalTranscriptIsCommittedOnce(t *testing.T) {
	s := newFakeTestServer()
	item := codex.RealtimeItem{ID: "item-1", RealtimeSessionID: "voice-1", Type: "transcriptSegment", Role: "user", Text: "Please inspect the code"}
	s.captureAgentVoiceItem("session", "thread", item)
	s.captureAgentVoiceItem("session", "thread", item)
	item.Type = "bemItemPromoted"
	s.captureAgentVoiceItem("session", "thread", item)
	state := s.conversations.ListSession(context.Background(), "session")
	if len(state.Events) != 1 || state.Events[0].Source != "codex_voice" || state.Events[0].Text != "Please inspect the code" {
		t.Fatalf("events=%+v", state.Events)
	}
}

// A live fake JSON-RPC owner exercises the real browser WebSocket handler,
// including isolation from observers, transport negotiation and teardown.
func TestAgentVoiceWebSocketOwnershipAndTranscript(t *testing.T) {
	s := newFakeTestServer()
	s.sessionStore = sessionstore.NewInMemory()
	_ = s.sessionStore.Save(context.Background(), sessionstore.Metadata{ID: "session", AgentType: sessionstore.AgentCodex, ProviderVersion: "0.161.0"})
	input, requests := io.Pipe()
	output, events := io.Pipe()
	var eventMu sync.Mutex
	send := func(value any) {
		eventMu.Lock()
		defer eventMu.Unlock()
		data, _ := json.Marshal(value)
		_, _ = events.Write(append(data, '\n'))
	}
	client := codex.NewClient(requests, output)
	owner, _ := codex.NewManagedOwner(client, "web-console:session", "thread")
	s.managedCodex = &managedCodexRegistry{entries: map[string]managedCodexEntry{"session": {owner: owner}}}
	methods := make(chan string, 32)
	go func() {
		scanner := bufio.NewScanner(input)
		for scanner.Scan() {
			var message codex.Message
			if json.Unmarshal(scanner.Bytes(), &message) != nil {
				continue
			}
			methods <- message.Method
			send(map[string]any{"id": message.ID, "result": map[string]any{"thread": map[string]string{"id": "thread"}}})
			if message.Method == "thread/realtime/start" {
				var params struct {
					Version   string `json:"version"`
					SessionID string `json:"realtimeSessionId"`
					Flush     bool   `json:"flushTranscriptTailOnSessionEnd"`
				}
				if err := json.Unmarshal(message.Params, &params); err != nil || params.Version != "v3" || params.Flush || params.SessionID == "" {
					t.Errorf("invalid native start: %s", message.Params)
				}
				send(map[string]any{"method": "thread/realtime/started", "params": map[string]string{"threadId": "thread", "realtimeSessionId": params.SessionID}})
			}
		}
	}()
	router := mux.NewRouter()
	router.HandleFunc("/sessions/{id}/voice", s.handleAgentVoiceWS)
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
		_ = input.Close()
		_ = requests.Close()
		_ = output.Close()
		_ = events.Close()
	})
	connect := func() *websocket.Conn {
		conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/sessions/session/voice", nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Close() })
		return conn
	}
	readType := func(conn *websocket.Conn, want string) map[string]any {
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		for i := 0; i < 12; i++ {
			var event map[string]any
			if err := conn.ReadJSON(&event); err != nil {
				t.Fatal(err)
			}
			if event["type"] == want {
				return event
			}
		}
		t.Fatalf("missing %s", want)
		return nil
	}
	a := connect()
	readType(a, "capability")
	_ = a.WriteJSON(agentVoiceCommand{Type: "start", Transport: "websocket"})
	readType(a, "accepted")
	b := connect()
	readType(b, "capability")
	_ = b.WriteJSON(agentVoiceCommand{Type: "start", Transport: "websocket"})
	if event := readType(b, "error"); event["code"] != "voice_in_use" {
		t.Fatal(event)
	}
	_ = b.WriteJSON(agentVoiceCommand{Type: "stop"})
	if event := readType(b, "error"); event["code"] != "not_controller" {
		t.Fatal(event)
	}
	send(map[string]any{"method": "thread/realtime/item/completed", "params": map[string]any{"threadId": "thread", "item": codex.RealtimeItem{ID: "i", RealtimeSessionID: "voice", Type: "transcriptSegment", Role: "assistant", Text: "I am listening"}}})
	readType(a, "event")
	_ = a.WriteJSON(agentVoiceCommand{Type: "stop"})
	readType(a, "stopped")
	state := s.conversations.ListSession(context.Background(), "session")
	if len(state.Events) != 1 {
		t.Fatalf("events=%+v", state.Events)
	}
	_ = a.Close()
	_ = b.Close()
	foundStart := false
	for len(methods) > 0 {
		method := <-methods
		if method == "thread/resume" {
			t.Fatal("voice observer resumed the already-owned managed thread")
		}
		if method == "thread/realtime/start" {
			foundStart = true
		}
	}
	if !foundStart {
		t.Fatal("native voice never started")
	}
}

func TestAgentVoiceSelectsOnlyLoadedThread(t *testing.T) {
	for _, tc := range []struct {
		recorded string
		loaded   []string
		want     string
	}{
		{"", []string{"one"}, "one"},
		{"", []string{"one", "two"}, ""},
		{"one", []string{"one", "two"}, "one"},
		{"stale", []string{"one"}, ""},
		{"one", nil, ""},
	} {
		if got := loadedVoiceThread(tc.recorded, tc.loaded); got != tc.want {
			t.Fatalf("selected %q, want %q", got, tc.want)
		}
	}
}

func TestAgentVoiceStartupCanBeRediscoveredWithoutLaunching(t *testing.T) {
	s := &Server{sessionStore: sessionstore.NewInMemory(), managedCodex: newManagedCodexRegistry(nil)}
	for _, agent := range []sessionstore.Agent{sessionstore.AgentNone, sessionstore.AgentCodex} {
		if err := s.sessionStore.Save(context.Background(), sessionstore.Metadata{ID: "startup-probe", AgentType: agent, ProviderVersion: "0.161.0"}); err != nil {
			t.Fatal(err)
		}
		cap, binding, err := s.resolveAgentVoice(context.Background(), "startup-probe")
		if err != nil || cap.Available || !cap.Retryable || binding != nil {
			t.Fatalf("starting capability = %+v, binding=%v err=%v", cap, binding, err)
		}
	}
	owner, err := codex.NewManagedOwner(codex.NewClient(testWriteCloser{io.Discard}, strings.NewReader("")), "owner", "thread")
	if err != nil {
		t.Fatal(err)
	}
	s.managedCodex.entries["startup-probe"] = managedCodexEntry{owner: owner}
	cap, binding, err := s.resolveAgentVoice(context.Background(), "startup-probe")
	if err != nil || !cap.Available || cap.Retryable || binding == nil {
		t.Fatalf("ready capability = %+v binding=%v err=%v", cap, binding, err)
	}
	binding.close()
}
