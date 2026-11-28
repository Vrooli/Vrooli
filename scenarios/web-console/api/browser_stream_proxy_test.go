package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"web-console/internal/sessionstore"
)

type fixedBrowserPaneReceiptResolver struct {
	receipt string
	list     paneSessionListResponse
	called  atomic.Int32
}

func (r *fixedBrowserPaneReceiptResolver) Issue(_ context.Context, webSessionID, runID, sessionID, role string) (string, error) {
	r.called.Add(1)
	if webSessionID != "web-9" || runID != "run-31" || sessionID != "session-42" || role != "observe" {
		return "", context.Canceled
	}
	return r.receipt, nil
}
func (r *fixedBrowserPaneReceiptResolver) List(context.Context, string) (paneSessionListResponse, error) {
	return r.list, nil
}
func (r *fixedBrowserPaneReceiptResolver) IssueHandle(_ context.Context, handle, role string) (paneReceiptGrant, error) {
	r.called.Add(1)
	if len(handle) != 48 || role != "observe" {
		return paneReceiptGrant{}, context.Canceled
	}
	return paneReceiptGrant{SessionID: "session-42", Receipt: r.receipt}, nil
}
func (*fixedBrowserPaneReceiptResolver) ReturnHandle(context.Context, string) error { return nil }

type browserPaneTestSessionStore struct {
	sessionstore.Store
	meta sessionstore.Metadata
}

func (s browserPaneTestSessionStore) Get(_ context.Context, id string) (sessionstore.Metadata, error) {
	if id != s.meta.ID {
		return sessionstore.Metadata{}, context.Canceled
	}
	return s.meta, nil
}

func TestBrowserStreamProxyExchangesReceiptForOwnerCookieAndRelaysFrames(t *testing.T) {
	var cookieCalls atomic.Int32
	ownerMessages := make(chan string, 2)
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/browser/sessions/session-42/stream-cookie":
			cookieCalls.Add(1)
			if r.Method != http.MethodPost {
				t.Errorf("cookie method=%s", r.Method)
			}
			var request map[string]string
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode cookie request: %v", err)
			}
			if request["receipt"] != "opaque-view" || request["role"] != "observe" {
				t.Errorf("cookie request=%v", request)
			}
			http.SetCookie(w, &http.Cookie{Name: "bas_ws_session-42", Value: "server-only-cookie", Path: "/ws/sessions/session-42", HttpOnly: true, SameSite: http.SameSiteStrictMode})
			_, _ = w.Write([]byte(`{"session_id":"session-42","role":"observe"}`))
		case "/ws/sessions/session-42":
			if !strings.Contains(r.Header.Get("Cookie"), "bas_ws_session-42=server-only-cookie") {
				t.Errorf("upstream cookie=%q", r.Header.Get("Cookie"))
			}
			conn, err := (&websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}).Upgrade(w, r, nil)
			if err != nil {
				t.Errorf("upstream upgrade: %v", err)
				return
			}
			defer conn.Close()
			for index := 0; index < 2; index++ {
				_, payload, err := conn.ReadMessage()
				if err != nil {
					return
				}
				ownerMessages <- string(payload)
				if index == 0 {
					_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"recording_subscribed","session_id":"session-42"}`))
				} else {
					_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"recording_input_applied","session_id":"session-42","input_id":"in-1"}`))
					_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"session_access_revoked","session_id":"session-42"}`))
				}
			}
		default:
			t.Errorf("unexpected owner path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer owner.Close()

	receipts := &fixedBrowserPaneReceiptResolver{receipt: "opaque-view"}
	proxy := newBrowserStreamProxy(func(context.Context) (string, error) { return owner.URL, nil }, receipts)
	router := mux.NewRouter()
	router.Handle("/api/v1/browser/stream", proxy).Methods(http.MethodGet)
	web := httptest.NewServer(router)
	defer web.Close()
	handle := strings.Repeat("a", 48)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(web.URL, "http")+"/api/v1/browser/stream", http.Header{"Origin": []string{web.URL}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(browserStreamAttach{Type: "attach", Role: "observe", PaneHandle: strings.Repeat("a", 48)}); err != nil {
		t.Fatal(err)
	}
	subscribe := `{"type":"subscribe_recording","session_id":"` + handle + `"}`
	if err := conn.WriteMessage(websocket.TextMessage, []byte(subscribe)); err != nil {
		t.Fatal(err)
	}
	_, got, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"session_id":"`+handle+`","type":"recording_subscribed"}` {
		t.Fatalf("relayed=%s", got)
	}
	input := `{"type":"recording_input","session_id":"` + handle + `","input":{"type":"keyboard","key":"a"}}`
	if err := conn.WriteMessage(websocket.TextMessage, []byte(input)); err != nil {
		t.Fatal(err)
	}
	if got := <-ownerMessages; !strings.Contains(got, `"session_id":"session-42"`) {
		t.Fatalf("first owner message=%s", got)
	}
	if got := <-ownerMessages; !strings.Contains(got, `"session_id":"session-42"`) {
		t.Fatalf("second owner message=%s", got)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, got, err = conn.ReadMessage()
	if err != nil || !strings.Contains(string(got), `"type":"recording_input_applied"`) {
		t.Fatalf("input acknowledgement relay=%s err=%v", got, err)
	}
	_, got, err = conn.ReadMessage()
	if err != nil || string(got) != `{"session_id":"`+handle+`","type":"session_access_revoked"}` {
		t.Fatalf("revocation relay=%s err=%v", got, err)
	}
	if cookieCalls.Load() != 1 {
		t.Fatalf("owner cookie calls=%d, want 1", cookieCalls.Load())
	}
	if receipts.called.Load() != 1 {
		t.Fatalf("pane receipt calls=%d, want 1", receipts.called.Load())
	}
}

func TestBrowserHubProxyMessageAllowlistIsSessionBound(t *testing.T) {
	for _, test := range []struct {
		name    string
		payload string
		allow   bool
	}{
		{name: "same session subscription", payload: `{"type":"subscribe_recording","session_id":"s-1"}`, allow: true},
		{name: "same session input", payload: `{"type":"recording_input","session_id":"s-1","input":{"type":"keyboard","key":"a"}}`, allow: true},
		{name: "other session subscription", payload: `{"type":"subscribe_recording","session_id":"s-2"}`},
		{name: "driver status escape", payload: `{"type":"subscribe_driver_status"}`},
		{name: "execution subscription escape", payload: `{"type":"subscribe_execution_frames","execution_id":"elsewhere"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := allowedBrowserHubInput([]byte(test.payload), "s-1"); got != test.allow {
				t.Fatalf("allow=%t, want %t", got, test.allow)
			}
		})
	}
	if allowedBrowserHubOutput(websocket.TextMessage, []byte(`{"type":"session_frame","session_id":"s-2","data":"abc","media_type":"image/png","width":1,"height":1}`), "s-1") {
		t.Fatal("cross-session frame was allowed")
	}
	packet := browserFramePacket(t, `{"version":1,"session_id":"s-1","page_id":"private-page","page_title":"Private title","page_url":"https://example.test/private?token=secret","captured_at":"2026-11-19T00:00:00Z"}`, []byte{0xff, 0xd8, 0xff, 0xd9})
	sanitized, err := sanitizeBrowserBinaryFrame(packet, "s-1")
	if err != nil {
		t.Fatal(err)
	}
	headerSize := binary.BigEndian.Uint32(sanitized[:4])
	var safeHeader map[string]any
	if err := json.Unmarshal(sanitized[4:4+headerSize], &safeHeader); err != nil {
		t.Fatal(err)
	}
	if _, exists := safeHeader["page_url"]; exists {
		t.Fatal("page URL reached the browser frame envelope")
	}
	if _, exists := safeHeader["page_title"]; exists {
		t.Fatal("page title reached the browser frame envelope")
	}
	if allowedBrowserHubOutput(websocket.BinaryMessage, packet, "s-2") {
		t.Fatal("cross-session binary frame was allowed")
	}
	if !allowedBrowserHubOutput(websocket.TextMessage, []byte(`{"type":"session_access_revoked","session_id":"s-1"}`), "s-1") {
		t.Fatal("owner revocation for the selected session was filtered")
	}
	for _, payload := range []string{
		`{"type":"session_access_revoked","session_id":"s-2"}`,
		`{"type":"session_access_revoked"}`,
		`{"type":"session_access_revoked","session_id":"s-1","secret":"must-not-cross"}`,
	} {
		if allowedBrowserHubOutput(websocket.TextMessage, []byte(payload), "s-1") {
			t.Fatalf("unbound owner revocation was allowed: %s", payload)
		}
	}
}

func browserFramePacket(t *testing.T, header string, image []byte) []byte {
	t.Helper()
	packet := make([]byte, 4+len(header)+len(image))
	binary.BigEndian.PutUint32(packet[:4], uint32(len(header)))
	copy(packet[4:], header)
	copy(packet[4+len(header):], image)
	return packet
}

func TestBrowserStreamProxyRejectsInvalidAttachBeforeOwnerCall(t *testing.T) {
	var calls atomic.Int32
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "unexpected", http.StatusInternalServerError)
	}))
	defer owner.Close()
	proxy := newBrowserStreamProxy(func(context.Context) (string, error) { return owner.URL, nil })
	router := mux.NewRouter()
	router.Handle("/api/v1/browser/stream", proxy).Methods(http.MethodGet)
	web := httptest.NewServer(router)
	defer web.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(web.URL, "http")+"/api/v1/browser/stream", http.Header{"Origin": []string{web.URL}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(browserStreamAttach{Type: "attach", Role: "invalid", PaneHandle: strings.Repeat("a", 48)}); err != nil {
		t.Fatal(err)
	}
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), "attach_invalid") {
		t.Fatalf("error frame=%s", payload)
	}
	if calls.Load() != 0 {
		t.Fatalf("owner contacted %d times for invalid attach", calls.Load())
	}
}

func TestBrowserStreamProxyKeepsBASReceiptDenialFailClosed(t *testing.T) {
	var streamDials atomic.Int32
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws/sessions/session-42" {
			streamDials.Add(1)
			return
		}
		http.Error(w, "receipt invalid", http.StatusForbidden)
	}))
	defer owner.Close()
	proxy := newBrowserStreamProxy(func(context.Context) (string, error) { return owner.URL, nil }, &fixedBrowserPaneReceiptResolver{receipt: "opaque-but-invalid"})
	router := mux.NewRouter()
	router.Handle("/api/v1/browser/stream", proxy).Methods(http.MethodGet)
	web := httptest.NewServer(router)
	defer web.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(web.URL, "http")+"/api/v1/browser/stream", http.Header{"Origin": []string{web.URL}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(browserStreamAttach{Type: "attach", Role: "observe", PaneHandle: strings.Repeat("a", 48)}); err != nil {
		t.Fatal(err)
	}
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), "owner_denied") {
		t.Fatalf("error frame=%s", payload)
	}
	if streamDials.Load() != 0 {
		t.Fatalf("Hub stream opened %d times after BAS denied the receipt", streamDials.Load())
	}
}
