package agentmanager

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/identity"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	"google.golang.org/protobuf/proto"
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
	for _, edit := range []func(*identity.Principal){
		func(p *identity.Principal) { p.Verified = false },
		func(p *identity.Principal) { p.Kind = identity.ActorAgent },
		func(p *identity.Principal) { p.Subject = "" },
		func(p *identity.Principal) { p.Scopes = nil },
		func(p *identity.Principal) { p.ExpiresAt = time.Now().Add(-time.Second) },
	} {
		contexts = append(contexts, auth01Caller(context.Background(), edit, false))
	}
	for _, ctx := range contexts {
		_, err := NewConnectClient(server.URL, server.Client()).Start(ctx, StartInput{AdmissionID: ownedRunID, Prompt: "fixture"})
		if !errors.Is(err, ErrCreateCaller) || calls != 0 {
			t.Fatalf("refusal %v calls=%d", err, calls)
		}
	}
}

func TestAuth01ConnectCarriesHumanProofOnlyOnCreateRun(t *testing.T) {
	fixture := &portalAgentManagerFixture{}
	client := servePortalAgentManager(t, fixture)
	parent := "123e4567-e89b-12d3-a456-426614174003"
	_, err := client.Start(auth01Caller(context.Background(), nil, false), StartInput{AdmissionID: ownedRunID, Prompt: "fixture", ParentRunID: parent})
	require.NoError(t, err)
	require.Equal(t, 1, fixture.profileCalls)
	require.Equal(t, 1, fixture.taskCalls)
	require.Equal(t, 1, fixture.createCalls)
	require.Equal(t, "", fixture.taskAuth)
	require.Equal(t, "Bearer fixture-owner", fixture.createAuth)
	require.Equal(t, parent, fixture.gotCreate.GetParentRunId())
	body, err := proto.Marshal(fixture.gotCreate)
	require.NoError(t, err)
	require.NotContains(t, string(body), "fixture-owner")
}

func TestApprovalPendingDetailBecomesDefinitiveTypedError(t *testing.T) {
	for _, tc := range []struct {
		name      string
		operation string
	}{
		{"task", "task"},
		{"run", "run"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := &portalAgentManagerFixture{}
			pending := &api.AuthorizationApprovalPending{Subject: "service:portal", Detail: "operator approval for fixture-owner required"}
			detail, err := connect.NewErrorDetail(pending)
			require.NoError(t, err)
			rpcErr := connect.NewError(connect.CodeFailedPrecondition, errors.New("approval pending"))
			rpcErr.AddDetail(detail)
			if tc.operation == "task" {
				fixture.createTaskErr = rpcErr
			} else {
				fixture.createErr = rpcErr
			}
			client := servePortalAgentManager(t, fixture)
			_, err = client.Start(auth01Caller(context.Background(), nil, false), StartInput{AdmissionID: ownedRunID, Prompt: "fixture"})
			var approval *AuthorizationApprovalPendingError
			require.ErrorAs(t, err, &approval)
			require.Equal(t, "service:portal", approval.Subject)
			require.Equal(t, "operator approval for [redacted] required", approval.Detail)
			require.Contains(t, approval.Error(), "operation was not started")
			require.NotContains(t, approval.Error(), "retry")
		})
	}
}

func TestConnectDoesNotFollowCreateRunRedirect(t *testing.T) {
	calls := 0
	fixture := &portalAgentManagerFixture{}
	path, generated := apiconnect.NewAgentManagerServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == apiconnect.AgentManagerServiceCreateRunProcedure {
			calls++
			w.Header().Set("Location", "/leak")
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
		generated.ServeHTTP(w, r)
	}))
	server := httptest.NewServer(mux)
	defer server.Close()
	_, err := NewConnectClient(server.URL, server.Client()).Start(auth01Caller(context.Background(), nil, false), StartInput{AdmissionID: ownedRunID, Prompt: "fixture"})
	require.Error(t, err)
	require.Equal(t, 1, calls)
}
