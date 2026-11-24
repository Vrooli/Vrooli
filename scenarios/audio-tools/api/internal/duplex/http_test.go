package duplex

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"audio-tools/internal/session"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/vrooli/api-core/identity"
)

type httpAdmission struct{ err error }

func (a httpAdmission) Authorize(context.Context, Request) error { return a.err }
func (a httpAdmission) Reserve(ctx context.Context, request Request) (Reservation, error) {
	if err := a.Authorize(ctx, request); err != nil {
		return nil, err
	}
	return &testReservation{}, nil
}

type httpProvider struct{ opens atomic.Int32 }

func (p *httpProvider) Open(context.Context, Request, Sink) (Connection, error) {
	p.opens.Add(1)
	return nil, errors.New("provider should not be contacted")
}

type liveHTTPProvider struct {
	text  chan string
	opens atomic.Int32
}

func (p *liveHTTPProvider) Open(context.Context, Request, Sink) (Connection, error) {
	p.opens.Add(1)
	return liveHTTPConnection{provider: p}, nil
}

type liveHTTPConnection struct{ provider *liveHTTPProvider }

func (c liveHTTPConnection) SendAudio(context.Context, session.AudioFrame) error { return nil }
func (c liveHTTPConnection) SendText(_ context.Context, text string) error {
	c.provider.text <- text
	return nil
}
func (liveHTTPConnection) Cancel(context.Context, session.BargeInReason, string) error { return nil }
func (liveHTTPConnection) Close() error                                                { return nil }

type failingHTTPProvider struct{}

func (failingHTTPProvider) Open(_ context.Context, _ Request, sink Sink) (Connection, error) {
	return failingHTTPConnection{sink: sink}, nil
}

type failingHTTPConnection struct{ sink Sink }

func (c failingHTTPConnection) SendAudio(context.Context, session.AudioFrame) error { return nil }
func (c failingHTTPConnection) SendText(context.Context, string) error {
	c.sink.Emit(Event{Err: errors.New("upstream disconnected")})
	return nil
}
func (failingHTTPConnection) Cancel(context.Context, session.BargeInReason, string) error { return nil }
func (failingHTTPConnection) Close() error                                                { return nil }

func TestWebSocketRequiresVerifiedHumanBeforeUpgrade(t *testing.T) {
	router := mux.NewRouter()
	HTTPModule(nil)(router)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/duplex/sessions", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want unauthorized", recorder.Code)
	}
}

func TestWebSocketAdmissionDenialNeverContactsProvider(t *testing.T) {
	provider := &httpProvider{}
	coordinator, err := NewCoordinator(provider, httpAdmission{err: ErrConsentRequired})
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	HTTPModule(coordinator)(router)
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceConsumerAccessToken, Verified: true}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(identity.WithPrincipal(r.Context(), principal))
		router.ServeHTTP(w, r)
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/duplex/sessions"
	header := http.Header{"Authorization": []string{"Bearer verified-test-token"}}
	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(openMessage{Type: "open", Consent: true}); err != nil {
		t.Fatal(err)
	}
	var response map[string]string
	if err := conn.ReadJSON(&response); err != nil {
		t.Fatal(err)
	}
	if response["type"] != "error" {
		t.Fatalf("response = %#v, want error", response)
	}
	if got := provider.opens.Load(); got != 0 {
		t.Fatalf("provider opens = %d, want 0", got)
	}
}

func TestWebSocketProviderStartupFailureDoesNotAcknowledgeReady(t *testing.T) {
	provider := &httpProvider{}
	coordinator, err := NewCoordinator(provider, httpAdmission{})
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	HTTPModule(coordinator)(router)
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceConsumerAccessToken, Verified: true}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(identity.WithPrincipal(r.Context(), principal))
		router.ServeHTTP(w, r)
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/duplex/sessions"
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Authorization": []string{"Bearer verified-test-token"}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(openMessage{Type: "open"}); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var response map[string]string
	if err := conn.ReadJSON(&response); err != nil {
		t.Fatal(err)
	}
	if response["type"] != "error" {
		t.Fatalf("first server frame = %#v, want startup error (never ready)", response)
	}
	if got := provider.opens.Load(); got != 1 {
		t.Fatalf("provider opens = %d, want 1 failed startup attempt", got)
	}
}

func TestWebSocketRejectsOversizedSessionEnvelope(t *testing.T) {
	coordinator, err := NewCoordinator(&httpProvider{}, httpAdmission{})
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	HTTPModule(coordinator)(router)
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceConsumerAccessToken, Verified: true}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(identity.WithPrincipal(r.Context(), principal))
		router.ServeHTTP(w, r)
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/duplex/sessions"
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Authorization": []string{"Bearer verified-test-token"}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	payload := strings.Repeat("x", maxClientText+1)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
		t.Fatal(err)
	}
	var response map[string]string
	if err := conn.ReadJSON(&response); err != nil {
		t.Fatal(err)
	}
	if response["type"] != "error" {
		t.Fatalf("response = %#v, want error", response)
	}
}

func TestWebSocketAdmittedSessionCanReadAfterInitialOpenTimeout(t *testing.T) {
	provider := &liveHTTPProvider{text: make(chan string, 1)}
	coordinator, err := NewCoordinator(provider, httpAdmission{})
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	HTTPModule(coordinator)(router)
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceConsumerAccessToken, Verified: true}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(identity.WithPrincipal(r.Context(), principal))
		router.ServeHTTP(w, r)
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/duplex/sessions"
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Authorization": []string{"Bearer verified-test-token"}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(openMessage{Type: "open"}); err != nil {
		t.Fatal(err)
	}
	// Exceed the initial-envelope timeout. It must not remain as an absolute
	// deadline on the upgraded socket after admission succeeds.
	time.Sleep(15*time.Second + 100*time.Millisecond)
	if err := conn.WriteJSON(clientMessage{Type: "text", Text: "still connected"}); err != nil {
		t.Fatalf("write after initial timeout: %v", err)
	}
	select {
	case text := <-provider.text:
		if text != "still connected" {
			t.Fatalf("text = %q", text)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not process post-timeout input")
	}
	_ = conn.WriteJSON(clientMessage{Type: "close"})
}

func TestWebSocketReadyAcknowledgesProviderStartupBeforeClientInput(t *testing.T) {
	provider := &liveHTTPProvider{text: make(chan string, 1)}
	coordinator, err := NewCoordinator(provider, httpAdmission{})
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	HTTPModule(coordinator)(router)
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceConsumerAccessToken, Verified: true}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(identity.WithPrincipal(r.Context(), principal))
		router.ServeHTTP(w, r)
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/duplex/sessions"
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Authorization": []string{"Bearer verified-test-token"}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(openMessage{Type: "open"}); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var ready map[string]string
	if err := conn.ReadJSON(&ready); err != nil {
		t.Fatal(err)
	}
	if ready["type"] != "ready" {
		t.Fatalf("first server frame = %#v, want admission ready", ready)
	}
	if got := provider.opens.Load(); got != 1 {
		t.Fatalf("provider opens before ready = %d, want 1", got)
	}
	if err := conn.WriteJSON(clientMessage{Type: "text", Text: "only after ready"}); err != nil {
		t.Fatal(err)
	}
	select {
	case text := <-provider.text:
		if text != "only after ready" {
			t.Fatalf("provider text = %q", text)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not process input after ready")
	}
	_ = conn.WriteJSON(clientMessage{Type: "close"})
}

func TestWebSocketClosesReaderAfterTerminalProviderFailure(t *testing.T) {
	coordinator, err := NewCoordinator(failingHTTPProvider{}, httpAdmission{})
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	HTTPModule(coordinator)(router)
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceConsumerAccessToken, Verified: true}
	handlerDone := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(handlerDone)
		r = r.WithContext(identity.WithPrincipal(r.Context(), principal))
		router.ServeHTTP(w, r)
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/duplex/sessions"
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Authorization": []string{"Bearer verified-test-token"}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(openMessage{Type: "open"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteJSON(clientMessage{Type: "text", Text: "trigger provider failure"}); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var event session.SessionEvent
	var ready map[string]string
	if err := conn.ReadJSON(&ready); err != nil {
		t.Fatalf("read admission acknowledgment: %v", err)
	}
	if ready["type"] != "ready" {
		t.Fatalf("first frame = %#v, want admission ready", ready)
	}
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read provider error event: %v", err)
	}
	if event.Type != session.EventProviderError {
		t.Fatalf("first event = %q, want %q", event.Type, session.EventProviderError)
	}
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read terminal close event: %v", err)
	}
	if event.Type != session.EventClosed {
		t.Fatalf("second event = %q, want %q", event.Type, session.EventClosed)
	}
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("socket remained open after terminal provider failure")
	}
	select {
	case <-handlerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("websocket handler remained blocked after terminal event")
	}
}

func TestWebSocketRejectsOpaqueAndForwardedOrigins(t *testing.T) {
	coordinator, err := NewCoordinator(&httpProvider{}, httpAdmission{})
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	HTTPModule(coordinator)(router)
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceConsumerAccessToken, Verified: true}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(identity.WithPrincipal(r.Context(), principal))
		router.ServeHTTP(w, r)
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/duplex/sessions"
	for _, tc := range []struct {
		name, origin, forwarded string
	}{
		{name: "opaque origin", origin: "null"},
		{name: "cross-origin through trusted forwarded host", origin: "https://attacker.example", forwarded: "audio-tools.example"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			header := http.Header{"Authorization": []string{"Bearer verified-test-token"}, "Origin": []string{tc.origin}}
			if tc.forwarded != "" {
				header.Set("X-Forwarded-Host", tc.forwarded)
			}
			conn, response, dialErr := websocket.DefaultDialer.Dial(url, header)
			if conn != nil {
				_ = conn.Close()
				t.Fatal("untrusted origin was upgraded")
			}
			if dialErr == nil || response == nil || response.StatusCode != http.StatusForbidden {
				t.Fatalf("dial error/status = %v/%v, want HTTP 403", dialErr, response)
			}
		})
	}
}

func TestSameOriginRequiresMatchingScheme(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://audio-tools.example/api/v1/duplex/sessions", nil)
	req.Host = "audio-tools.example"
	if sameOrigin(req, "https://audio-tools.example") {
		t.Fatal("same host with a different scheme was accepted")
	}
	if !sameOrigin(req, "http://audio-tools.example") {
		t.Fatal("matching origin was rejected")
	}
	for _, origin := range []string{"not a URL", "http://user@audio-tools.example", "//audio-tools.example"} {
		if sameOrigin(req, origin) {
			t.Errorf("invalid origin %q was accepted", origin)
		}
	}
}

var (
	_ Provider = (*httpProvider)(nil)
	_ Provider = (*liveHTTPProvider)(nil)
)
