package cliutil

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestOrdinaryAttachedCleanupForwardsOnlyItsHeldSelfIdentity(t *testing.T) {
	t.Setenv(AgentManagerIdentityTokenEnv, "different-parent-fixture")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/runs/owned-fixture/detach" || r.Header.Get(HeaderAgentIdentityToken) != "held-attached-fixture" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	if err := detachCodingAgent(context.Background(), AgentLaunchRequest{APIBase: server.URL}, "owned-fixture", "held-attached-fixture"); err != nil {
		t.Fatalf("attached self cleanup lost its own identity: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatal("cleanup did not use exactly one held-identity request")
	}
}

func TestAttachedCleanupDoesNotForwardIdentityThroughRedirect(t *testing.T) {
	var redirected atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected.Add(1); w.WriteHeader(http.StatusOK) }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	if err := detachCodingAgent(context.Background(), AgentLaunchRequest{APIBase: server.URL}, "owned-fixture", "held-attached-fixture"); err == nil {
		t.Fatal("redirected cleanup reported success")
	}
	if redirected.Load() != 0 {
		t.Fatal("attached identity cleanup followed a redirect")
	}
}

func TestAttachedCleanupRejectsIncompleteEndpoint(t *testing.T) {
	if err := detachCodingAgent(context.Background(), AgentLaunchRequest{APIBase: "localhost"}, "owned-fixture", "held-attached-fixture"); err == nil {
		t.Fatal("invalid cleanup endpoint reported success")
	}
}
