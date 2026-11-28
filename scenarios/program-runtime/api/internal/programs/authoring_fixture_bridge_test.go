package programs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthoringFixtureBridgeScopesAndFailsClosed(t *testing.T) {
	bridge := NewAuthoringFixtureBridge()
	bridge.Register("authoring-session", &AuthoringFixture{Bindings: map[string][]FixtureReply{
		"agent-manager/effort/board": {{Status: http.StatusOK, Body: map[string]any{"rows": []any{}}}},
	}})
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := bridge.Wrap(next)

	request := func(sessionID, bindingID string) *httptest.ResponseRecorder {
		raw := `{"session_id":"` + sessionID + `","binding_id":"` + bindingID + `"}`
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(raw)))
		return recorder
	}
	good := request("authoring-session", "agent-manager/effort/board")
	if good.Code != http.StatusOK {
		t.Fatalf("fixture status=%d body=%s", good.Code, good.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(good.Body.Bytes(), &body); err != nil || body["rows"] == nil {
		t.Fatalf("fixture response=%s err=%v", good.Body.String(), err)
	}
	unlisted := request("authoring-session", "search-hub/query/query")
	if unlisted.Code != http.StatusBadRequest || !strings.Contains(unlisted.Body.String(), "no response") {
		t.Fatalf("unlisted fixture call must fail closed: %d %s", unlisted.Code, unlisted.Body.String())
	}
	normal := request("normal-session", "search-hub/query/query")
	if normal.Code != http.StatusNoContent {
		t.Fatalf("non-fixture sessions must pass through: %d", normal.Code)
	}
	if got := bridge.Calls("authoring-session")["agent-manager/effort/board"]; got != 1 {
		t.Fatalf("fixture call count=%d, want 1", got)
	}
}

func TestAuthoringFixtureBridgePreservesUnavailableResponse(t *testing.T) {
	bridge := NewAuthoringFixtureBridge()
	bridge.Register("authoring-session", &AuthoringFixture{Bindings: map[string][]FixtureReply{
		"agent-manager/effort/board": {{Status: http.StatusServiceUnavailable, Body: map[string]any{
			"error": "agent-manager/effort/board: fixture unavailable", "class": "unavailable", "status": "unavailable",
			"binding_id": "agent-manager/effort/board", "http_status": http.StatusServiceUnavailable,
		}}},
	}})
	handler := bridge.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("fixture call reached the live registry") }))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"session_id":"authoring-session","binding_id":"agent-manager/effort/board"}`))
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("fixture HTTP status=%d", recorder.Code)
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["class"] != "unavailable" || response["binding_id"] != "agent-manager/effort/board" {
		t.Fatalf("fixture error semantics were not preserved: %#v", response)
	}
}
