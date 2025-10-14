package main

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type routeTestEnv struct {
	t         *testing.T
	mux       *http.ServeMux
	manager   *sessionManager
	metrics   *metricsRegistry
	workspace *workspace
}

func setupRouteTestEnv(t *testing.T) *routeTestEnv {
	cleanup := setupTestLogger()
	t.Cleanup(cleanup)

	env := setupTestDirectory(t)
	t.Cleanup(env.Cleanup)

	cfg := setupTestConfig(env.TempDir)
	cfg.maxConcurrent = 1 // exercise capacity limits easily

	manager, metrics, ws := setupTestSessionManager(t, cfg)

	mux := http.NewServeMux()
	registerRoutes(mux, manager, metrics, ws)

	return &routeTestEnv{t: t, mux: mux, manager: manager, metrics: metrics, workspace: ws}
}

func (env *routeTestEnv) do(method, path string, body any) *httptest.ResponseRecorder {
	req := makeHTTPRequest(HTTPTestRequest{Method: method, Path: path, Body: body})
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)
	return w
}

func decodeJSONResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode JSON payload: %v", err)
	}
	return payload
}

func TestRegisterRoutes_EndpointsAndErrors(t *testing.T) {
	env := setupRouteTestEnv(t)

	// PUT workspace to seed state
	tab := TestData.CreateTabRequest("tab-1", "Terminal 1", "sky")
	putBody := map[string]any{
		"activeTabId": tab.ID,
		"tabs":        []tabMeta{tab},
	}
	if w := env.do(http.MethodPut, "/api/v1/workspace", putBody); w.Code != http.StatusOK {
		t.Fatalf("expected 200 from workspace PUT, got %d (%s)", w.Code, w.Body.String())
	}

	// PATCH workspace keyboard toolbar mode
	patchBody := map[string]any{"keyboardToolbarMode": "disabled"}
	if w := env.do(http.MethodPatch, "/api/v1/workspace", patchBody); w.Code != http.StatusOK {
		t.Fatalf("expected 200 from workspace PATCH, got %d (%s)", w.Code, w.Body.String())
	}

	// POST workspace tab creation
	tabCreate := TestData.CreateTabRequest("tab-2", "Terminal 2", "emerald")
	if w := env.do(http.MethodPost, "/api/v1/workspace/tabs", tabCreate); w.Code != http.StatusCreated {
		t.Fatalf("expected 201 from tab create, got %d (%s)", w.Code, w.Body.String())
	}

	// Duplicate tab to trigger conflict path
	if w := env.do(http.MethodPost, "/api/v1/workspace/tabs", tabCreate); w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate tab, got %d (%s)", w.Code, w.Body.String())
	}

	// Update tab metadata
	tabUpdate := map[string]any{"label": "Term 2", "colorId": "violet"}
	if w := env.do(http.MethodPatch, "/api/v1/workspace/tabs/tab-2", tabUpdate); w.Code != http.StatusOK {
		t.Fatalf("expected 200 from tab update, got %d (%s)", w.Code, w.Body.String())
	}

	// Delete tab
	if w := env.do(http.MethodDelete, "/api/v1/workspace/tabs/tab-2", nil); w.Code != http.StatusOK {
		t.Fatalf("expected 200 from tab delete, got %d (%s)", w.Code, w.Body.String())
	}

	// Deleting again should surface not found branch
	if w := env.do(http.MethodDelete, "/api/v1/workspace/tabs/tab-2", nil); w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when deleting missing tab, got %d (%s)", w.Code, w.Body.String())
	}

	// Method not allowed variants
	if w := env.do(http.MethodGet, "/api/v1/workspace/tabs", nil); w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET tabs, got %d", w.Code)
	}
	if w := env.do(http.MethodPost, "/api/v1/workspace/stream", nil); w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for POST workspace stream, got %d", w.Code)
	}
	if w := env.do(http.MethodPatch, "/api/v1/workspace/tabs/", map[string]any{"label": "noop"}); w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing tab id, got %d", w.Code)
	}

	// Generate command route via mux (correct path)
	genBody := map[string]any{"prompt": "list files"}
	if w := env.do(http.MethodPost, "/api/generate-command", genBody); w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 200 or 500 from generate command, got %d", w.Code)
	}

	// Session lifecycle
	sleepReq := TestData.CreateSessionRequest("/bin/sleep", []string{"1"})
	if w := env.do(http.MethodPost, "/api/v1/sessions", sleepReq); w.Code != http.StatusCreated {
		t.Fatalf("expected 201 from session create, got %d (%s)", w.Code, w.Body.String())
	} else {
		resp := decodeJSONResponse(t, w)
		sessionID := resp["id"].(string)

		// List sessions should include capacity header
		listResp := env.do(http.MethodGet, "/api/v1/sessions", nil)
		if listResp.Code != http.StatusOK {
			t.Fatalf("expected 200 from session list, got %d", listResp.Code)
		}
		if listResp.Header().Get("X-Session-Capacity") == "" {
			t.Fatal("expected X-Session-Capacity header")
		}

		// Retrieve session details
		detailResp := env.do(http.MethodGet, "/api/v1/sessions/"+sessionID, nil)
		if detailResp.Code != http.StatusOK {
			t.Fatalf("expected 200 from session detail, got %d (%s)", detailResp.Code, detailResp.Body.String())
		}

		// Panic wrong method (session still active)
		if w := env.do(http.MethodGet, "/api/v1/sessions/"+sessionID+"/panic", nil); w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405 from panic GET, got %d", w.Code)
		}

		// Session websocket stream upgrade failure path
		streamResp := httptest.NewRecorder()
		streamReq := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+sessionID+"/stream", nil)
		env.mux.ServeHTTP(streamResp, streamReq)
		if streamResp.Code != http.StatusInternalServerError && streamResp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 or 500 from non-hijacker stream, got %d", streamResp.Code)
		}

		// Delete session by id
		if w := env.do(http.MethodDelete, "/api/v1/sessions/"+sessionID, nil); w.Code != http.StatusOK {
			t.Fatalf("expected 200 from delete session, got %d (%s)", w.Code, w.Body.String())
		}
	}

	// Panic endpoint verified on separate session
	panicReq := TestData.CreateSessionRequest("/bin/sleep", []string{"1"})
	if w := env.do(http.MethodPost, "/api/v1/sessions", panicReq); w.Code == http.StatusCreated {
		resp := decodeJSONResponse(t, w)
		panicID := resp["id"].(string)
		if w := env.do(http.MethodPost, "/api/v1/sessions/"+panicID+"/panic", nil); w.Code != http.StatusAccepted {
			t.Fatalf("expected 202 from panic endpoint, got %d (%s)", w.Code, w.Body.String())
		}
	}

	// Exceed capacity (second concurrent session)
	capacityReq := TestData.CreateSessionRequest("/bin/sleep", []string{"1"})
	first := env.do(http.MethodPost, "/api/v1/sessions", capacityReq)
	if first.Code != http.StatusCreated {
		t.Fatalf("expected to create first session, got %d (%s)", first.Code, first.Body.String())
	}
	resp := decodeJSONResponse(t, first)
	heldID := resp["id"].(string)

	// Ensure session stays active briefly
	time.Sleep(50 * time.Millisecond)

	second := env.do(http.MethodPost, "/api/v1/sessions", capacityReq)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 when capacity exceeded, got %d (%s)", second.Code, second.Body.String())
	}

	// Delete all sessions
	if w := env.do(http.MethodDelete, "/api/v1/sessions", nil); w.Code != http.StatusOK {
		t.Fatalf("expected 200 from delete all, got %d (%s)", w.Code, w.Body.String())
	}

	// Allow asynchronous close routines to finish
	time.Sleep(50 * time.Millisecond)

	// Verify manager cleaned up held session
	if _, ok := env.manager.getSession(heldID); ok {
		t.Fatal("expected held session to be removed after delete-all")
	}

	// Fetch non-existent session to cover 404
	if w := env.do(http.MethodGet, "/api/v1/sessions/does-not-exist", nil); w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing session, got %d", w.Code)
	}

	// Workspace stream upgrade failure path
	wsStreamResp := httptest.NewRecorder()
	wsStreamReq := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/stream", nil)
	env.mux.ServeHTTP(wsStreamResp, wsStreamReq)
	if wsStreamResp.Code != http.StatusInternalServerError && wsStreamResp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 or 500 from workspace stream without hijacker, got %d", wsStreamResp.Code)
	}
}

func TestLoggingResponseWriterInterfaces(t *testing.T) {
	rich := &instrumentedResponseWriter{header: http.Header{}}
	lrw := &loggingResponseWriter{ResponseWriter: rich, status: http.StatusOK}

	lrw.WriteHeader(http.StatusAccepted)
	if lrw.status != http.StatusAccepted || rich.status != http.StatusAccepted {
		t.Fatalf("expected status propagation, got lrw=%d rich=%d", lrw.status, rich.status)
	}

	conn, _, err := lrw.Hijack()
	if err != nil {
		t.Fatalf("expected successful hijack: %v", err)
	}
	_ = conn.Close()
	if !rich.hijacked {
		t.Fatal("expected underlying hijack to be called")
	}

	lrw.Flush()
	if !rich.flushed {
		t.Fatal("expected flush to be forwarded")
	}

	if n, err := lrw.ReadFrom(strings.NewReader("payload")); err != nil || n == 0 {
		t.Fatalf("expected readFrom to succeed, n=%d err=%v", n, err)
	}
	if rich.readFromData != "payload" {
		t.Fatalf("unexpected readFrom data: %q", rich.readFromData)
	}

	if err := lrw.Push("/test", nil); err != nil {
		t.Fatalf("expected push to succeed: %v", err)
	}
	if len(rich.pushTargets) == 0 || rich.pushTargets[0] != "/test" {
		t.Fatalf("expected push target to be recorded, got %v", rich.pushTargets)
	}
}

func TestLoggingResponseWriterFallbacks(t *testing.T) {
	basic := &basicResponseWriter{header: http.Header{}}
	lrw := &loggingResponseWriter{ResponseWriter: basic, status: http.StatusOK}

	if _, _, err := lrw.Hijack(); err == nil {
		t.Fatal("expected error when underlying writer lacks hijacker")
	}

	lrw.Flush() // should no-op without panic

	if n, err := lrw.ReadFrom(strings.NewReader("data")); err != nil || n == 0 {
		t.Fatalf("expected fallback ReadFrom, n=%d err=%v", n, err)
	}
	if basic.written != "data" {
		t.Fatalf("expected fallback write to store data, got %q", basic.written)
	}

	if err := lrw.Push("/push", nil); err != http.ErrNotSupported {
		t.Fatalf("expected ErrNotSupported from Push fallback, got %v", err)
	}
}

// instrumentedResponseWriter exposes optional interfaces for loggingResponseWriter tests.
type instrumentedResponseWriter struct {
	header       http.Header
	status       int
	hijacked     bool
	flushed      bool
	readFromData string
	pushTargets  []string
}

func (rw *instrumentedResponseWriter) Header() http.Header         { return rw.header }
func (rw *instrumentedResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (rw *instrumentedResponseWriter) WriteHeader(status int)      { rw.status = status }
func (rw *instrumentedResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	rw.hijacked = true
	left, right := net.Pipe()
	go right.Close()
	return left, bufio.NewReadWriter(bufio.NewReader(left), bufio.NewWriter(left)), nil
}
func (rw *instrumentedResponseWriter) Flush() { rw.flushed = true }
func (rw *instrumentedResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	rw.readFromData = string(data)
	return int64(len(data)), nil
}
func (rw *instrumentedResponseWriter) Push(target string, opts *http.PushOptions) error {
	rw.pushTargets = append(rw.pushTargets, target)
	return nil
}

// Ensure interface compliance
var _ http.ResponseWriter = (*instrumentedResponseWriter)(nil)
var _ http.Hijacker = (*instrumentedResponseWriter)(nil)
var _ http.Flusher = (*instrumentedResponseWriter)(nil)
var _ io.ReaderFrom = (*instrumentedResponseWriter)(nil)
var _ http.Pusher = (*instrumentedResponseWriter)(nil)

type basicResponseWriter struct {
	header  http.Header
	written string
}

func (rw *basicResponseWriter) Header() http.Header { return rw.header }
func (rw *basicResponseWriter) Write(b []byte) (int, error) {
	rw.written += string(b)
	return len(b), nil
}
func (rw *basicResponseWriter) WriteHeader(status int) {}

var _ http.ResponseWriter = (*basicResponseWriter)(nil)
