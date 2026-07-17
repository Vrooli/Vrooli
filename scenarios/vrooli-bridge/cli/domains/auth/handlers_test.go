package auth

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/identity"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	"github.com/vrooli/cli-core/cliutil"

	clitest "vrooli-bridge/cli/internal/testutil"
)

type fakeIdentityClient struct {
	loginRequest  *identityv1.LoginRequest
	loginResponse *identityv1.LoginResponse
	loginErr      error
}

func (f *fakeIdentityClient) Login(_ context.Context, req *connect.Request[identityv1.LoginRequest]) (*connect.Response[identityv1.LoginResponse], error) {
	f.loginRequest = req.Msg
	if f.loginErr != nil {
		return nil, f.loginErr
	}
	return connect.NewResponse(f.loginResponse), nil
}

func (f *fakeIdentityClient) Register(context.Context, *connect.Request[identityv1.RegisterRequest]) (*connect.Response[identityv1.RegisterResponse], error) {
	return nil, errors.New("unexpected Register call")
}

func loginSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "email", Required: true}, {Name: "password-stdin", Bool: true}}}
}

func newTestHandlers(t *testing.T, client *fakeIdentityClient) (*handlers, *cliapp.ScenarioApp) {
	t.Helper()
	core := clitest.NewTestApp(t, nil)
	core.ConfigFile = &cliutil.ConfigFile{Path: filepath.Join(t.TempDir(), "config.json")}
	h := &handlers{core: core, client: client}
	return h, core
}

func TestLoginPersistsReturnedOwnerTokenWithoutPrintingSecrets(t *testing.T) {
	client := &fakeIdentityClient{loginResponse: &identityv1.LoginResponse{Token: "owner-jwt-secret", Email: "owner@example.com", UserId: "owner-1"}}
	h, core := newTestHandlers(t, client)
	h.password = passwordSource{stdin: strings.NewReader("correct horse battery staple\n")}
	ctx, output := cliapptest.NewCapturedRunContext(core, loginSchema(), cliapptest.TestRunContextOptions{
		Flags:     map[string]string{"email": "owner@example.com"},
		BoolFlags: map[string]bool{"password-stdin": true},
	})

	if err := h.login(ctx); err != nil {
		t.Fatalf("login: %v", err)
	}
	if client.loginRequest == nil || client.loginRequest.Email != "owner@example.com" || client.loginRequest.Password != "correct horse battery staple" {
		t.Fatalf("login request = %#v", client.loginRequest)
	}
	if core.Config.Token != "owner-jwt-secret" {
		t.Fatalf("saved token = %q", core.Config.Token)
	}
	if got := output.String(); strings.Contains(got, "owner-jwt-secret") || strings.Contains(got, "correct horse battery staple") || !strings.Contains(got, "owner@example.com") {
		t.Fatalf("unsafe login output %q", got)
	}
}

func TestLoginFailurePreservesExistingToken(t *testing.T) {
	client := &fakeIdentityClient{loginErr: connect.NewError(connect.CodeUnauthenticated, errors.New("invalid credentials"))}
	h, core := newTestHandlers(t, client)
	core.Config.Token = "existing-token"
	h.password = passwordSource{stdin: strings.NewReader("wrong\n")}
	ctx, _ := cliapptest.NewCapturedRunContext(core, loginSchema(), cliapptest.TestRunContextOptions{
		Flags:     map[string]string{"email": "owner@example.com"},
		BoolFlags: map[string]bool{"password-stdin": true},
	})

	if err := h.login(ctx); err == nil {
		t.Fatal("expected login failure")
	}
	if core.Config.Token != "existing-token" {
		t.Fatalf("token changed after failed login: %q", core.Config.Token)
	}
}

func TestLoginRequiresTTYOrPasswordStdin(t *testing.T) {
	client := &fakeIdentityClient{}
	h, core := newTestHandlers(t, client)
	h.password = passwordSource{isTerminal: func() bool { return false }, stdin: io.Reader(strings.NewReader("unused")), prompt: io.Discard}
	ctx, _ := cliapptest.NewCapturedRunContext(core, loginSchema(), cliapptest.TestRunContextOptions{Flags: map[string]string{"email": "owner@example.com"}})

	err := h.login(ctx)
	if err == nil || !strings.Contains(err.Error(), "--password-stdin") {
		t.Fatalf("error = %v, want secure non-TTY guidance", err)
	}
	if client.loginRequest != nil {
		t.Fatal("login RPC must not run before a password is available")
	}
}
