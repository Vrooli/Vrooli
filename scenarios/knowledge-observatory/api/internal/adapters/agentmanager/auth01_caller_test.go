package agentmanager

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/owneridentity"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

type auth01FixtureVerifier struct{}

func (auth01FixtureVerifier) Validate(_ context.Context, _ string) (owneridentity.Identity, error) {
	return owneridentity.Identity{Subject: "fixture", Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func auth01FixtureCaller(t *testing.T, ctx context.Context) context.Context {
	t.Helper()
	out, err := owneridentity.AuthorizeCreateRunCaller(ctx, http.Header{"Authorization": []string{"Bearer fixture-owner"}}, auth01FixtureVerifier{}, time.Now(), true)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestAuth01CallerTransportPositiveAndZeroEffect(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer fixture-owner" || r.Header.Get("X-Agent-Identity-Token") != "" {
			t.Error("proof lost or replaced")
		}
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "fixture-owner") {
			t.Error("proof persisted")
		}
		_, _ = io.WriteString(w, `{"run":{"id":"fixture-run"}}`)
	}))
	defer server.Close()
	c := NewClientWithBaseURL(time.Second, server.URL)
	if _, err := c.CreateRun(context.Background(), &apipb.CreateRunRequest{TaskId: "fixture-task"}); err == nil || calls != 0 {
		t.Fatal("anonymous effect")
	}
	if _, err := c.CreateRun(auth01FixtureCaller(t, context.Background()), &apipb.CreateRunRequest{TaskId: "fixture-task"}); err != nil || calls != 1 {
		t.Fatalf("positive %v calls%d", err, calls)
	}
}

func TestAuth01CallerRefusalAndRedirectDoNotRetryOrLeak(t *testing.T) {
	for _, code := range []int{403, 307} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("Location", "/leak")
			w.WriteHeader(code)
			_, _ = io.WriteString(w, "fixture-owner")
		}))
		c := NewClientWithBaseURL(time.Second, server.URL)
		_, err := c.CreateRun(auth01FixtureCaller(t, context.Background()), &apipb.CreateRunRequest{TaskId: "fixture-task"})
		server.Close()
		if err == nil || strings.Contains(err.Error(), "fixture-owner") || calls != 1 {
			t.Fatalf("refusal %v calls%d", err, calls)
		}
	}
}
