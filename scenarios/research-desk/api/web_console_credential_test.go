package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"

	"github.com/vrooli/api-core/servicetoken"
	"research-desk/internal/peer"
)

func webConsoleExchange(result servicetoken.Exchanged, err error) servicetoken.Option {
	return servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) { return result, err })
}

func goodWebConsoleToken() servicetoken.Exchanged {
	return servicetoken.Exchanged{AccessToken: "desk-token", ExpiresAt: time.Now().Add(10 * time.Minute), Subject: "research-desk", Audience: webConsoleAudience, Scopes: []string{"web-console:write", "web-console:read"}}
}

func issued(context.Context) (string, error)    { return "desk-credential", nil }
func notIssued(context.Context) (string, error) { return "", errors.New("no such entry") }

// The desk keeps today's documented behavior until the operator issues the
// principal: it sends no credential and says so.
func TestWithoutAnIssuedCredentialTheDeskReachesWebConsoleUnderItsAvailabilityPolicy(t *testing.T) {
	credential := webConsoleAuthorization(newWebConsoleTokens(notIssued))
	got, err := credential(context.Background())
	if err != nil || got != "" {
		t.Fatalf("credential = %q, %v; want none and no error", got, err)
	}
	access, err := (&peer.Peer{Name: "web-console", Credential: credential}).Access(context.Background())
	if err != nil || access != peer.AccessAvailability {
		t.Fatalf("access = %q, %v", access, err)
	}
}

func TestAnIssuedCredentialIsPresentedToWebConsoleAsTheDesksBearer(t *testing.T) {
	var seen atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.Store(r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	web := &peer.Peer{
		Name:       "web-console",
		BaseURL:    func(context.Context) (string, error) { return server.URL, nil },
		Credential: webConsoleAuthorization(newWebConsoleTokens(issued, webConsoleExchange(goodWebConsoleToken(), nil))),
	}
	ctx := context.Background()
	if access, err := web.Access(ctx); err != nil || access != peer.AccessCredential {
		t.Fatalf("access = %q, %v; want the service credential", access, err)
	}
	client, base, option, err := web.Dial(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = sessionsconnect.NewSessionsServiceClient(client, base, option).List(ctx, connect.NewRequest(&sessionsv1.ListRequest{}))
	if got, _ := seen.Load().(string); got != "Bearer desk-token" {
		t.Fatalf("Authorization = %q, want the desk's bearer", got)
	}
}

// A credential the desk was given but cannot use is never a reason to drop to
// the weaker policy: the desk does not call.
func TestACredentialThatCannotBeUsedStopsTheDeskCallingWebConsole(t *testing.T) {
	wrong := goodWebConsoleToken()
	wrong.Scopes = []string{"web-console:read"}
	for name, option := range map[string]servicetoken.Option{
		"exchange refused":   webConsoleExchange(servicetoken.Exchanged{}, errors.New("credential revoked")),
		"not the desk's own": webConsoleExchange(wrong, nil),
	} {
		credential := webConsoleAuthorization(newWebConsoleTokens(issued, option))
		got, err := credential(context.Background())
		if err == nil || got != "" || errors.Is(err, servicetoken.ErrNotIssued) {
			t.Fatalf("%s: credential = %q, %v; want an error that is not 'not issued'", name, got, err)
		}
		if _, _, _, err := (&peer.Peer{Name: "web-console", BaseURL: func(context.Context) (string, error) { return "http://unused", nil }, Credential: credential}).Dial(context.Background()); err == nil {
			t.Fatalf("%s: the desk must not dial", name)
		}
	}
}
