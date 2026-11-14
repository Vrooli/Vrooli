package agentmanager

import (
	"context"
	"encoding/json"
	"github.com/vrooli/api-core/identity"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func auth01Caller(ctx context.Context, edit func(*identity.Principal), runOffered bool) context.Context {
	p := identity.Principal{Kind: identity.ActorHuman, Verified: true, Subject: "fixture", Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}
	if edit != nil {
		edit(&p)
	}
	return WithVerifiedCreateCaller(ctx, "fixture-owner", p, runOffered)
}
func TestAuth01PortalRejectsBeforeAnyOwnerEffect(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	contexts := []context.Context{context.Background(), auth01Caller(context.Background(), nil, true)}
	for _, edit := range []func(*identity.Principal){func(p *identity.Principal) { p.Verified = false }, func(p *identity.Principal) { p.Kind = identity.ActorAgent }, func(p *identity.Principal) { p.Subject = "" }, func(p *identity.Principal) { p.Scopes = nil }, func(p *identity.Principal) { p.ExpiresAt = time.Now().Add(-time.Second) }} {
		contexts = append(contexts, auth01Caller(context.Background(), edit, false))
	}
	for _, ctx := range contexts {
		_, err := NewHTTPClient(server.URL, server.Client()).Start(ctx, StartInput{AdmissionID: ownedRunID, Prompt: "fixture"})
		if err != ErrCreateCaller || calls != 0 {
			t.Fatalf("refusal %v calls%d", err, calls)
		}
	}
}
func TestAuth01PortalHumanProofOnlyOnCreateRun(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("X-Agent-Identity-Token") != "" {
			t.Error("unexpected run proof")
		}
		if r.URL.Path == "/api/v1/runs" {
			if r.Header.Get("Authorization") != "Bearer fixture-owner" {
				t.Error("human proof lost")
			}
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			if payload["parent_run_id"] != "fixture-parent" {
				t.Error("explicit parent lost")
			}
			encoded, _ := json.Marshal(payload)
			if strings.Contains(string(encoded), "fixture-owner") {
				t.Error("credential persisted")
			}
			_, _ = io.WriteString(w, `{"run":{"id":"fixture-run"}}`)
			return
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("CreateRun proof widened to sibling")
		}
		switch r.URL.Path {
		case "/api/v1/profiles/reconcile-scenario":
			_, _ = io.WriteString(w, `{"results":[{"profile_key":"portal/agent-chat","profile_id":"fixture-profile"}]}`)
		case "/api/v1/tasks":
			_, _ = io.WriteString(w, `{"task":{"id":"fixture-task"}}`)
		default:
			t.Errorf("unexpected%s", r.URL.Path)
		}
	}))
	defer server.Close()
	_, err := NewHTTPClient(server.URL, server.Client()).Start(auth01Caller(context.Background(), nil, false), StartInput{AdmissionID: ownedRunID, Prompt: "fixture", ParentRunID: "fixture-parent"})
	if err != nil || calls != 3 {
		t.Fatalf("positive %v calls%d", err, calls)
	}
}
func TestAuth01PortalRefusalAndRedirectAreNotRetriedOrLeaked(t *testing.T) {
	for _, code := range []int{http.StatusForbidden, http.StatusTemporaryRedirect} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("Location", "/leak")
			w.WriteHeader(code)
			_, _ = io.WriteString(w, "fixture-owner")
		}))
		c := NewHTTPClient(server.URL, server.Client())
		_, err := c.createRun(auth01Caller(context.Background(), nil, false), "fixture-task", StartInput{AdmissionID: ownedRunID, Prompt: "fixture"})
		server.Close()
		if err == nil || strings.Contains(err.Error(), "fixture-owner") || calls != 1 {
			t.Fatalf("refusal/redirect %v calls%d", err, calls)
		}
	}
}

func TestAuth01PortalPreservesExistingWildcardCapabilities(t *testing.T) {
	for _, scope := range []string{"agent-manager:write", "agent-manager:*", "*:write", "*"} {
		ctx := auth01Caller(context.Background(), func(p *identity.Principal) { p.Scopes = []string{scope} }, false)
		if err := RequireCreateCaller(ctx); err != nil {
			t.Fatalf("existing capability %s refused: %v", scope, err)
		}
	}
}
