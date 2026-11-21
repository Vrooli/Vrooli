package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/usagemodes"

	"github.com/vrooli/api-core/identity"
)

func autoSaveServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	return &Server{usageModeNotifier: &usagemodes.Notifier{
		Modes: &usagemodes.Service{Path: filepath.Join(dir, "mode.json")},
		Path:  filepath.Join(dir, "notifier.json"),
	}}
}

func autoSaveRequest(method string, body string, p *identity.Principal) *http.Request {
	r := httptest.NewRequest(method, UsageModeAutoSavePath, bytes.NewBufferString(body))
	if p != nil {
		r = r.WithContext(identity.WithPrincipal(r.Context(), *p))
	}
	return r
}

func rulesOf(t *testing.T, s *Server) []map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	s.handleUsageModeAutoSave(w, autoSaveRequest(http.MethodGet, "", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET = %d %s", w.Code, w.Body)
	}
	var out struct {
		Rules []map[string]any `json:"rules"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out.Rules
}

// Only a verified independent human with write capability can change a rule: it
// starts Save usage by itself, so an agent, a service or an anonymous caller
// must not be able to turn it on or off.
func TestAutoSaveRuleWritesNeedAVerifiedOperator(t *testing.T) {
	human := identity.Principal{Kind: identity.ActorHuman, Subject: "operator@example.com", Verified: true, Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}
	agent := identity.Principal{Kind: identity.ActorAgent, Subject: "run-1", Verified: true, Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}
	readOnlyHuman := identity.Principal{Kind: identity.ActorHuman, Subject: "viewer@example.com", Verified: true, Scopes: []string{"agent-manager:read"}, ExpiresAt: time.Now().Add(time.Hour)}
	s := autoSaveServer(t)
	on := `{"runner":"codex","enabled":true}`
	for name, p := range map[string]*identity.Principal{"anonymous": nil, "agent": &agent, "human without write": &readOnlyHuman} {
		w := httptest.NewRecorder()
		s.handleUsageModeAutoSave(w, autoSaveRequest(http.MethodPost, on, p))
		if w.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d, want 403", name, w.Code)
		}
	}
	if len(rulesOf(t, s)) != 0 {
		t.Fatal("a refused caller created a rule")
	}

	w := httptest.NewRecorder()
	s.handleUsageModeAutoSave(w, autoSaveRequest(http.MethodPost, on, &human))
	if w.Code != http.StatusOK {
		t.Fatalf("operator POST = %d %s", w.Code, w.Body)
	}
	rules := rulesOf(t, s)
	if len(rules) != 1 || rules[0]["runner"] != "codex" || rules[0]["by"] != "operator@example.com" {
		t.Fatalf("rules = %v", rules)
	}

	w = httptest.NewRecorder()
	s.handleUsageModeAutoSave(w, autoSaveRequest(http.MethodPost, `{"runner":"gemini","enabled":true}`, &human))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown runner = %d, want 400", w.Code)
	}
	w = httptest.NewRecorder()
	s.handleUsageModeAutoSave(w, autoSaveRequest(http.MethodPost, `{"runner":"codex","enabled":false}`, &human))
	if w.Code != http.StatusOK || len(rulesOf(t, s)) != 0 {
		t.Fatalf("the operator could not turn the rule off: %d %s", w.Code, strings.TrimSpace(w.Body.String()))
	}
}
