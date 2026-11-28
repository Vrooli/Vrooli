package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/servicetoken"
	"web-console/internal/sessionstore"
)

// This composes the production Web Console candidate-list handler, opaque-handle
// resolver and stream proxy. AM and BAS remain fixture owners; this is source
// integration evidence, never a real-session qualification.
func TestBrowserPaneCandidateToSameOriginFrameUsesOneOwnerSession(t *testing.T) {
	const (
		webSession = "wc-session"
		basSession = "bas-session"
		ownerRun   = "d28e41fa-c82a-4e6f-8c3b-0e6c50d9dc6c"
		thread     = "thread-17"
		human      = "human-3"
	)
	var ownerCookieRequests, ownerStreamRequests int
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/browser/sessions/bas-session/stream-cookie":
			ownerCookieRequests++
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode BAS receipt request: %v", err)
			}
			if body["receipt"] != "bas-owner-receipt" || body["role"] != "observe" {
				t.Errorf("BAS cookie receipt binding: %v", body)
			}
			http.SetCookie(w, &http.Cookie{Name: "bas_ws_bas-session", Value: "owner-cookie-secret", Path: "/ws/sessions/bas-session", HttpOnly: true, SameSite: http.SameSiteStrictMode})
			_, _ = w.Write([]byte(`{"session_id":"bas-session","role":"observe"}`))
		case "/ws/sessions/bas-session":
			ownerStreamRequests++
			if !strings.Contains(r.Header.Get("Cookie"), "bas_ws_bas-session=owner-cookie-secret") {
				t.Errorf("BAS owner cookie missing: %q", r.Header.Get("Cookie"))
			}
			conn, err := (&websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}).Upgrade(w, r, nil)
			if err != nil {
				t.Errorf("upgrade BAS stream: %v", err)
				return
			}
			defer conn.Close()
			_, _, err = conn.ReadMessage() // exact-session subscribe from the pane
			if err != nil {
				return
			}
			header, _ := json.Marshal(map[string]any{"version": 1, "session_id": basSession, "captured_at": "2026-11-19T22:00:00Z", "page_url": "https://private.example/?token=secret"})
			packet := make([]byte, 4+len(header)+4)
			binary.BigEndian.PutUint32(packet[:4], uint32(len(header)))
			copy(packet[4:], header)
			copy(packet[4+len(header):], []byte{0xff, 0xd8, 0xff, 0xd9})
			_ = conn.WriteMessage(websocket.BinaryMessage, packet)
		default:
			http.NotFound(w, r)
		}
	}))
	defer owner.Close()

	am := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer web-console-service-token" {
			t.Errorf("AM service identity missing: %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/api/v1/browser/pane/sessions":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode AM candidate request: %v", err)
			}
			if body["web_console_session_id"] != webSession || body["thread_id"] != thread || body["owner_subject"] != human {
				t.Errorf("AM candidate binding: %v", body)
			}
			_ = json.NewEncoder(w).Encode(paneSessionInternalResponse{RunID: ownerRun, ThreadID: thread, Sessions: []paneSessionInternal{{SessionID: basSession, Mode: "managed"}}})
		case "/api/v1/browser/pane/resolve":
			var body paneReceiptIssueRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode AM receipt request: %v", err)
			}
			if body.RunID != ownerRun || body.SessionID != basSession || body.ThreadID != thread || body.Role != "observe" || body.OwnerSubject != human {
				t.Errorf("AM exact-session receipt binding: %+v", body)
			}
			_ = json.NewEncoder(w).Encode(paneReceiptIssueResponse{SessionID: basSession, Role: "observe", Receipt: "bas-owner-receipt"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer am.Close()

	store := browserPaneTestSessionStore{meta: sessionstore.Metadata{ID: webSession, Owner: human, Status: sessionstore.StatusLive, NativeThreadID: thread}}
	resolver := newWebConsoleBrowserPaneReceiptResolver(store)
	resolver.resolve = func(context.Context) (string, error) { return am.URL, nil }
	resolver.tokens = webConsoleBrowserPaneRole.NewTokens("web-console", func(context.Context) (string, error) { return "synthetic-credential", nil }, servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) {
		return servicetoken.Exchanged{AccessToken: "web-console-service-token", ExpiresAt: time.Now().Add(time.Hour), Subject: webConsoleBrowserPaneRole.Principal("web-console"), Audience: webConsoleBrowserPaneRole.Audience, Scopes: []string{webConsoleBrowserPaneRole.Scope}}, nil
	}))
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: human, Verified: true, Source: identity.SourcePersonalLocal, Scopes: []string{"web-console:write"}, ExpiresAt: time.Now().Add(time.Hour)}
	apiRouter := mux.NewRouter()
	apiRouter.HandleFunc("/api/v1/browser/pane/sessions", func(w http.ResponseWriter, r *http.Request) {
		browserPaneSessionsHandler(resolver)(w, r.WithContext(identity.WithPrincipal(r.Context(), principal)))
	}).Methods(http.MethodPost)
	streamProxy := newBrowserStreamProxy(func(context.Context) (string, error) { return owner.URL, nil }, resolver)
	apiRouter.Handle("/api/v1/browser/stream", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		streamProxy.ServeHTTP(w, r.WithContext(identity.WithPrincipal(r.Context(), principal)))
	})).Methods(http.MethodGet)
	api := httptest.NewServer(apiRouter)
	defer api.Close()

	response, err := http.Post(api.URL+"/api/v1/browser/pane/sessions", "application/json", strings.NewReader(`{"web_console_session_id":"wc-session"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("candidate status=%d", response.StatusCode)
	}
	var listed paneSessionListResponse
	if err := json.NewDecoder(response.Body).Decode(&listed); err != nil || len(listed.Sessions) != 1 {
		t.Fatalf("candidate list=%+v err=%v", listed, err)
	}
	handle := listed.Sessions[0].Handle
	if len(handle) != 48 || handle == basSession {
		t.Fatalf("browser received non-opaque selector %q", handle)
	}
	public, _ := json.Marshal(listed)
	if strings.Contains(string(public), ownerRun) || strings.Contains(string(public), thread) || strings.Contains(string(public), basSession) {
		t.Fatalf("browser pane response leaked owner identifiers: %s", public)
	}
	conn, handshake, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(api.URL, "http")+"/api/v1/browser/stream", http.Header{"Origin": []string{api.URL}})
	if err != nil {
		if handshake != nil && handshake.Body != nil {
			defer handshake.Body.Close()
			body, _ := io.ReadAll(handshake.Body)
			t.Fatalf("open proxied stream: %v status=%d body=%s", err, handshake.StatusCode, body)
		}
		t.Fatalf("open proxied stream: %v response=%v", err, handshake)
	}
	defer conn.Close()
	_ = conn.WriteJSON(browserStreamAttach{Type: "attach", Role: "observe", PaneHandle: handle})
	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"subscribe_recording","session_id":"`+handle+`"}`))
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	messageType, frame, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read owner frame: %v", err)
	}
	if messageType != websocket.BinaryMessage {
		t.Fatalf("frame type=%d, want binary: %s", messageType, frame)
	}
	if len(frame) < 6 {
		t.Fatalf("frame too short: %d", len(frame))
	}
	headerLength := int(binary.BigEndian.Uint32(frame[:4]))
	var safe map[string]any
	if 4+headerLength > len(frame) || json.Unmarshal(frame[4:4+headerLength], &safe) != nil {
		t.Fatalf("invalid proxied frame header")
	}
	if safe["session_id"] != handle || safe["captured_at"] != "2026-11-19T22:00:00Z" {
		t.Fatalf("frame lost pane/session provenance: %v", safe)
	}
	if _, leaksURL := safe["page_url"]; leaksURL || strings.Contains(string(frame), "token=secret") || strings.Contains(string(frame), "owner-cookie-secret") {
		t.Fatalf("frame leaked private owner metadata: %v", safe)
	}
	if ownerCookieRequests != 1 || ownerStreamRequests != 1 {
		t.Fatalf("owner calls cookie=%d stream=%d", ownerCookieRequests, ownerStreamRequests)
	}
}
