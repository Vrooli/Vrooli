package agentmanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/owneridentity"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
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

func TestAuth01CallerConnectTransportAndRedirectPolicy(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer fixture-owner" || r.Header.Get("X-Agent-Identity-Token") != "" {
			t.Error("caller proof lost or replaced")
		}
		w.Header().Set("Location", "/redirect-target")
		w.WriteHeader(http.StatusTemporaryRedirect)
		_, _ = w.Write([]byte("fixture-owner"))
	}))
	defer server.Close()
	c := NewClientWithBaseURL(time.Second, server.URL)
	if _, err := c.CreateRun(context.Background(), &apipb.CreateRunRequest{TaskId: "task"}); err == nil || calls != 0 {
		t.Fatal("anonymous create run was sent")
	}
	_, err := c.CreateRun(auth01FixtureCaller(t, context.Background()), &apipb.CreateRunRequest{TaskId: "task"})
	if err == nil || strings.Contains(err.Error(), "fixture-owner") || calls != 1 {
		t.Fatalf("redirect result err=%v calls=%d", err, calls)
	}
}

func TestAuth01CallerConnectRefusalDoesNotRetry(t *testing.T) {
	calls := 0
	mux := http.NewServeMux()
	procedure := apiconnect.AgentManagerServiceCreateRunProcedure
	mux.Handle(procedure, connect.NewUnaryHandler(procedure, func(context.Context, *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
		calls++
		return nil, connect.NewError(connect.CodePermissionDenied, nil)
	}))
	server := httptest.NewServer(mux)
	defer server.Close()
	_, err := NewClientWithBaseURL(time.Second, server.URL).CreateRun(auth01FixtureCaller(t, context.Background()), &apipb.CreateRunRequest{TaskId: "task"})
	if err == nil || calls != 1 {
		t.Fatalf("refusal err=%v calls=%d", err, calls)
	}
}
