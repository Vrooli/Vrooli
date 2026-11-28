package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/decisionpage"
	"github.com/vrooli/api-core/servicetoken"
)

func pageWriterToken() servicetoken.Exchanged {
	return servicetoken.Exchanged{AccessToken: "writer-token", ExpiresAt: time.Now().Add(10 * time.Minute), Subject: pageWriterPrincipal, Audience: pageWriterAudience, Scopes: []string{"agent-manager:write"}}
}

// The writer creates runs only as the desk's own Agent Manager principal: with
// the issued credential it presents exactly that bearer, and without it, or
// with a token that is not exactly the desk's, it creates nothing.
func TestThePageWriterCreatesRunsOnlyAsTheDesksAgentManagerPrincipal(t *testing.T) {
	var seen atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.Store(r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/api/v1/tasks":
			_, _ = w.Write([]byte(`{"task":{"id":"task-1"}}`))
		default:
			_, _ = w.Write([]byte(`{"run":{"id":"run-1","task_id":"task-1","status":"RUN_STATUS_RUNNING"}}`))
		}
	}))
	defer server.Close()
	base := func(context.Context) (string, error) { return server.URL, nil }
	launch := func(tokens *servicetoken.Tokens) error {
		_, err := (&decisionpage.AgentManager{BaseURL: base, Authorization: tokens.Authorization}).Launch(context.Background(), decisionpage.LaunchSpec{Title: "t", Prompt: "p", IdempotencyKey: "k"})
		return err
	}
	exchange := func(result servicetoken.Exchanged) servicetoken.Option {
		return servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) { return result, nil })
	}
	if err := launch(newPageWriterTokens(issued, exchange(pageWriterToken()))); err != nil {
		t.Fatal(err)
	}
	if got, _ := seen.Load().(string); got != "Bearer writer-token" {
		t.Fatalf("Authorization = %q, want the desk's Agent Manager bearer", got)
	}
	seen.Store("")
	wrong := pageWriterToken()
	wrong.Subject = "research-desk"
	for name, tokens := range map[string]*servicetoken.Tokens{
		"not issued":         newPageWriterTokens(notIssued),
		"not the desk's own": newPageWriterTokens(issued, exchange(wrong)),
	} {
		if err := launch(tokens); !errors.Is(err, decisionpage.ErrRefused) {
			t.Fatalf("%s: launch = %v; want a refusal", name, err)
		}
		if got, _ := seen.Load().(string); got != "" {
			t.Fatalf("%s: a call reached Agent Manager with %q", name, got)
		}
	}
}
