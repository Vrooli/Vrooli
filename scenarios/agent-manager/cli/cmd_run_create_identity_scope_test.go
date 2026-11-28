package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
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

func TestRunCreateLocalOwnerRefusesAgentBeforeAnyRequest(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "identified-agent-fixture")
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	t.Cleanup(server.Close)
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
	err := (&App{services: NewServices(api)}).cmdRun([]string{
		"create", "--task-id=11111111-1111-4111-8111-111111111111",
		"--profile-id=22222222-2222-4222-8222-222222222222", "--local-owner",
	})
	if err == nil || !strings.Contains(err.Error(), "unavailable inside an identified agent run") {
		t.Fatalf("local owner agent refusal = %v", err)
	}
	if calls != 0 {
		t.Fatalf("refused agent reached run creation: %d requests", calls)
	}
}

func TestRunCreateLocalOwnerValidatesBeforeAuthentication(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "identified-agent-fixture")
	err := (&App{}).cmdRun([]string{
		"create", "--task-id=invalid", "--profile-id=22222222-2222-4222-8222-222222222222", "--local-owner",
	})
	if err == nil || !strings.Contains(err.Error(), "--task-id") || strings.Contains(err.Error(), "identified agent") {
		t.Fatalf("invalid request was not rejected before authentication: %v", err)
	}
}

func TestRunCreateLocalOwnerTransport(t *testing.T) {
	for _, tc := range []struct {
		name       string
		brokerMode string
		refusal    connect.Code
		empty      bool
		wantOK     bool
	}{
		{name: "operator credential is request local", wantOK: true},
		{name: "broker identifies agent", brokerMode: "broker-agent"},
		{name: "owner exchange refuses", refusal: connect.CodePermissionDenied},
		{name: "owner exchange returns no token", empty: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := &exactSetupOwnerFixture{resource: standingGrantResource, brokerMode: tc.brokerMode, refusal: tc.refusal, empty: tc.empty}
			calls := 0
			app, base, _ := newExactSetupOwnerApp(t, fixture, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				wantToken := "Bearer synthetic-owner-access"
				if calls == 2 {
					wantToken = "Bearer synthetic-ordinary-operator"
				}
				if r.Method != http.MethodPost || r.URL.Path != "/api/v1/runs" || r.Header.Get("Authorization") != wantToken || r.Header.Get(cliutil.HeaderAgentIdentityToken) != "" {
					t.Error("run creation lost request-local operator authentication")
				}
				if r.Header.Get(cliutil.HeaderInvocationCommand) != "run create" {
					t.Error("run creation lost CLI provenance")
				}
				body, err := protojson.Marshal(&apipb.CreateRunResponse{Run: &domainpb.Run{Id: "11111111-1111-4111-8111-111111111111"}})
				if err != nil {
					t.Error(err)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(body)
			}))
			args := []string{"--api-base", base, "run", "create", "--task-id=11111111-1111-4111-8111-111111111111", "--profile-id=22222222-2222-4222-8222-222222222222", "--json"}
			var runErr error
			output := captureStdout(t, func() error { runErr = app.Run(append(append([]string{}, args...), "--local-owner")); return nil })
			if (runErr == nil) != tc.wantOK {
				t.Fatalf("owner outcome = %v, want success=%t", runErr, tc.wantOK)
			}
			if strings.Contains(output, "synthetic-owner-access") || strings.Contains(output, "synthetic-unused-refresh") || runErr != nil && strings.Contains(runErr.Error(), "synthetic-owner-access") {
				t.Fatal("owner credential leaked into output")
			}
			if !tc.wantOK {
				if calls != 0 {
					t.Fatalf("refused owner fell back to ordinary credentials: %d requests", calls)
				}
				if tc.brokerMode != "" && fixture.calls.Load() != 0 {
					t.Fatal("identified broker caller reached the human exchange")
				}
				return
			}
			if fixture.calls.Load() != 1 || calls != 1 {
				t.Fatalf("owner exchange/create calls = %d/%d", fixture.calls.Load(), calls)
			}
			if err := app.Run(args); err != nil || calls != 2 || fixture.calls.Load() != 1 {
				t.Fatalf("ordinary credential was mutated after owner request: err=%v calls=%d exchanges=%d", err, calls, fixture.calls.Load())
			}
		})
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
