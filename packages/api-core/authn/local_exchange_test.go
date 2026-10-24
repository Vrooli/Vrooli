package authn

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	"google.golang.org/protobuf/proto"
)

type localExchangeFixture struct {
	accountsconnect.UnimplementedAccountsServiceHandler
	exchange func(context.Context, *connect.Request[accountsv1.ExchangeMachinePrincipalRequest]) (*connect.Response[accountsv1.LoginResponse], error)
}

func (f localExchangeFixture) ExchangeMachinePrincipal(ctx context.Context, req *connect.Request[accountsv1.ExchangeMachinePrincipalRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
	return f.exchange(ctx, req)
}

func isolatedExchangeDir(t *testing.T) string {
	t.Helper()
	// Keep even the canonical socket name below Unix sockaddr path limits.
	dir, err := os.MkdirTemp("/tmp", "authn-")
	if err != nil {
		t.Fatal("create fixture directory")
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	t.Setenv("TMPDIR", dir)
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "unrelated-caller")
	t.Setenv("VROOLI_AUTH_SOCKET", "")
	return dir
}

func serveLocalExchange(t *testing.T, socket string, fixture localExchangeFixture) {
	t.Helper()
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal("listen on fixture socket")
	}
	path, handler := accountsconnect.NewAccountsServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() { _ = server.Close(); <-done })
}

func TestExchangeLocalMachinePrincipalSocketSelection(t *testing.T) {
	for _, override := range []bool{false, true} {
		name := "canonical-default"
		if override {
			name = "trimmed-explicit-override"
		}
		t.Run(name, func(t *testing.T) {
			dir := isolatedExchangeDir(t)
			socket := filepath.Join(dir, "vrooli-scenario-authenticator-scenario-authenticator.sock")
			if DefaultLocalAuthenticatorSocket() != socket {
				t.Fatal("default socket must ignore caller namespace")
			}
			t.Setenv("VROOLI_AUTH_SOCKET", " \t")
			if override {
				socket = filepath.Join(dir, "custom.sock")
				t.Setenv("VROOLI_AUTH_SOCKET", " \t"+socket+" \n")
			}
			want := &accountsv1.LoginResponse{
				Account: &accountsv1.Account{},
				Tokens:  &accountsv1.TokenPair{AccessToken: " fixture-access ", RefreshToken: "fixture-refresh"},
			}
			hostname, err := os.Hostname()
			if err != nil {
				t.Fatal("fixture hostname unavailable")
			}
			requests := make(chan string, 1)
			serveLocalExchange(t, socket, localExchangeFixture{exchange: func(_ context.Context, req *connect.Request[accountsv1.ExchangeMachinePrincipalRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
				requests <- req.Msg.MachineId
				return connect.NewResponse(want), nil
			}})
			got, err := ExchangeLocalMachinePrincipal(context.Background())
			if err != nil || !proto.Equal(got, want) {
				t.Fatal("exchange must preserve successful account and token fields")
			}
			if <-requests != hostname {
				t.Fatal("exchange must send the machine hostname")
			}
		})
	}
}

func TestExchangeLocalMachinePrincipalResponses(t *testing.T) {
	for _, tc := range []struct {
		name        string
		response    *accountsv1.LoginResponse
		refusal     connect.Code
		wantSuccess bool
	}{
		{name: "permission-refused", refusal: connect.CodePermissionDenied},
		{name: "unauthenticated", refusal: connect.CodeUnauthenticated},
		{name: "missing-tokens", response: &accountsv1.LoginResponse{}},
		{name: "empty-access", response: &accountsv1.LoginResponse{Tokens: &accountsv1.TokenPair{}}},
		{name: "blank-access", response: &accountsv1.LoginResponse{Tokens: &accountsv1.TokenPair{AccessToken: " \t"}}},
		{name: "optional-refresh", response: &accountsv1.LoginResponse{Tokens: &accountsv1.TokenPair{AccessToken: "fixture-access"}}, wantSuccess: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolatedExchangeDir(t)
			serveLocalExchange(t, DefaultLocalAuthenticatorSocket(), localExchangeFixture{exchange: func(context.Context, *connect.Request[accountsv1.ExchangeMachinePrincipalRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
				if tc.refusal != 0 {
					return nil, connect.NewError(tc.refusal, errors.New("fixture refusal"))
				}
				return connect.NewResponse(tc.response), nil
			}})
			got, err := ExchangeLocalMachinePrincipal(context.Background())
			if tc.wantSuccess {
				if err != nil || !proto.Equal(got, tc.response) {
					t.Fatal("optional refresh must be accepted")
				}
				return
			}
			if err == nil || got != nil {
				t.Fatal("invalid exchange must return error and no session")
			}
			if tc.refusal != 0 && connect.CodeOf(err) != tc.refusal {
				t.Fatal("server refusal code must survive")
			}
		})
	}
}

func TestExchangeLocalMachinePrincipalForResourceSendsResource(t *testing.T) {
	isolatedExchangeDir(t)
	seen := ""
	serveLocalExchange(t, DefaultLocalAuthenticatorSocket(), localExchangeFixture{exchange: func(_ context.Context, req *connect.Request[accountsv1.ExchangeMachinePrincipalRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
		seen = req.Msg.GetResource()
		return connect.NewResponse(&accountsv1.LoginResponse{Tokens: &accountsv1.TokenPair{AccessToken: "resource-access"}}), nil
	}})
	if _, err := ExchangeLocalMachinePrincipalForResource(context.Background(), "device-sync-hub"); err != nil {
		t.Fatal(err)
	}
	if seen != "device-sync-hub" {
		t.Fatalf("resource = %q, want device-sync-hub", seen)
	}
}

func TestExchangeLocalMachinePrincipalUnavailableAndCanceled(t *testing.T) {
	dir := isolatedExchangeDir(t)
	t.Setenv("VROOLI_AUTH_SOCKET", filepath.Join(dir, "absent.sock"))
	if got, err := ExchangeLocalMachinePrincipal(context.Background()); err == nil || got != nil {
		t.Fatal("missing explicit socket must not produce a session")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := ExchangeLocalMachinePrincipal(ctx); got != nil || connect.CodeOf(err) != connect.CodeCanceled {
		t.Fatal("caller cancellation must survive")
	}
}

func TestExchangeLocalMachinePrincipalNeverUsesTCP(t *testing.T) {
	isolatedExchangeDir(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls.Add(1)
	}))
	defer server.Close()
	t.Setenv("VROOLI_AUTH_SOCKET", server.URL)
	if got, err := ExchangeLocalMachinePrincipal(context.Background()); err == nil || got != nil {
		t.Fatal("a URL is not a Unix socket")
	}
	if calls.Load() != 0 {
		t.Fatal("socket override must never select TCP")
	}
}

// A socket path that overflows sun_path cannot be bound. The canonical path
// under macOS's /var/folders/<id>/T/ is 106 bytes, so the authenticator
// exited with "bind: invalid argument" on every Mac.
func TestLocalAuthenticatorSocketFitsThePlatformLimit(t *testing.T) {
	const macTemp = "/var/folders/_m/f86gss0s08q_s04915vwk3lc0000gn/T"
	canonical := filepath.Join(macTemp, "vrooli-scenario-authenticator-scenario-authenticator.sock")

	got := localAuthenticatorSocketIn(macTemp, "scenario-authenticator", "darwin")
	if got == canonical || len(got) > 103 || filepath.Dir(got) != macTemp {
		t.Fatalf("darwin path = %q (%d bytes), want a short name in %s", got, len(got), macTemp)
	}
	if again := localAuthenticatorSocketIn(macTemp, "scenario-authenticator", "darwin"); again != got {
		t.Fatalf("path must be deterministic: %q then %q", got, again)
	}
	if other := localAuthenticatorSocketIn(macTemp, "other-namespace", "darwin"); other == got {
		t.Fatal("distinct namespaces must not share a socket")
	}
	if linux := localAuthenticatorSocketIn(macTemp, "scenario-authenticator", "linux"); linux != canonical {
		t.Fatalf("a path within the Linux limit must stay canonical, got %q", linux)
	}
	if short := localAuthenticatorSocketIn("/tmp", "scenario-authenticator", "darwin"); short != "/tmp/vrooli-scenario-authenticator-scenario-authenticator.sock" {
		t.Fatalf("a path that fits must stay canonical, got %q", short)
	}
	if name := filepath.Base(localAuthenticatorSocketIn("/tmp", "a/b c", "linux")); name != "vrooli-scenario-authenticator-a-b-c.sock" {
		t.Fatalf("namespace must be sanitized like the authenticator always did, got %q", name)
	}
}
