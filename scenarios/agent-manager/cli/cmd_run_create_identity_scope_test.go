package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestRunCreateForwardsRequestedScopesAsNarrowing(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	var received apipb.CreateRunRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/runs" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(readAll(t, r), &received); err != nil {
			t.Error(err)
		}
		body, err := protojson.Marshal(&apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-1"}})
		if err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer server.Close()
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
	args := []string{
		"create", "--task-id=11111111-1111-4111-8111-111111111111",
		"--profile-id=22222222-2222-4222-8222-222222222222", "--requested-scope=treasury:read",
		"--requested-scope=treasury:write", "--expected-owner-subject=owner-1", "--json",
	}
	if err := (&App{services: NewServices(api)}).cmdRun(args); err != nil {
		t.Fatal(err)
	}
	if received.RequestedScopes == nil || received.RequestedScopes.ExpectedOwnerSubject != "owner-1" || len(received.RequestedScopes.Scopes) != 2 || received.RequestedScopes.Scopes[0] != "treasury:read" || received.RequestedScopes.Scopes[1] != "treasury:write" {
		t.Fatalf("requested identity narrowing changed in transport: %+v", received.RequestedScopes)
	}
}

func TestRunCreateRejectsEmptyRequestedScopeBeforeOwnerCall(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
	}))
	defer server.Close()
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
	args := []string{"create", "--task-id=11111111-1111-4111-8111-111111111111", "--profile-id=22222222-2222-4222-8222-222222222222", "--requested-scope="}
	if err := (&App{services: NewServices(api)}).cmdRun(args); err == nil {
		t.Fatal("empty requested scope was accepted")
	}
	if calls != 0 {
		t.Fatalf("invalid scope request reached the owner: calls=%d", calls)
	}
}
