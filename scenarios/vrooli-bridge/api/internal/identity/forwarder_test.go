package identity_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/discovery"

	"vrooli-bridge/internal/identity"

	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

// fakeAccounts is a configurable Connect AccountsService stub standing in for
// scenario-authenticator. Only Register/Login are exercised by the forwarder;
// the rest inherit Unimplemented.
type fakeAccounts struct {
	accountsconnect.UnimplementedAccountsServiceHandler
	loginFn    func(*accountsv1.LoginRequest) (*accountsv1.LoginResponse, error)
	registerFn func(*accountsv1.RegisterRequest) (*accountsv1.RegisterResponse, error)
	refreshFn  func(*accountsv1.RefreshRequest) (*accountsv1.RefreshResponse, error)
}

func (f *fakeAccounts) Login(_ context.Context, req *connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
	resp, err := f.loginFn(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeAccounts) Register(_ context.Context, req *connect.Request[accountsv1.RegisterRequest]) (*connect.Response[accountsv1.RegisterResponse], error) {
	resp, err := f.registerFn(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeAccounts) Refresh(_ context.Context, req *connect.Request[accountsv1.RefreshRequest]) (*connect.Response[accountsv1.RefreshResponse], error) {
	resp, err := f.refreshFn(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func newAuthStub(t *testing.T, impl *fakeAccounts) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := accountsconnect.NewAccountsServiceHandler(impl)
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newForwarder(t *testing.T, url string) *identity.Forwarder {
	t.Helper()
	return identity.NewForwarder(identity.Config{Resolver: discovery.NewStaticResolver(url)})
}

func TestForwarderLogin(t *testing.T) {
	t.Run("ok relays token and identity", func(t *testing.T) {
		srv := newAuthStub(t, &fakeAccounts{loginFn: func(_ *accountsv1.LoginRequest) (*accountsv1.LoginResponse, error) {
			return &accountsv1.LoginResponse{
				Account: &accountsv1.Account{Id: "u-9", Email: "o@x.io"},
				Tokens:  &accountsv1.TokenPair{AccessToken: "jwt-123", RefreshToken: "r-1"},
			}, nil
		}})
		f := newForwarder(t, srv.URL)

		o, err := f.Login(context.Background(), identity.Credentials{Email: "o@x.io", Password: "pw"})
		require.NoError(t, err)
		assert.Equal(t, "jwt-123", o.Token)
		assert.Equal(t, "o@x.io", o.Email)
		assert.Equal(t, "u-9", o.UserID)
	})

	t.Run("unauthenticated is invalid credentials", func(t *testing.T) {
		srv := newAuthStub(t, &fakeAccounts{loginFn: func(_ *accountsv1.LoginRequest) (*accountsv1.LoginResponse, error) {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid email or password"))
		}})
		f := newForwarder(t, srv.URL)
		_, err := f.Login(context.Background(), identity.Credentials{Email: "o@x.io", Password: "bad"})
		assert.ErrorIs(t, err, identity.ErrInvalidCredentials)
	})

	t.Run("permission denied (locked) is invalid credentials", func(t *testing.T) {
		srv := newAuthStub(t, &fakeAccounts{loginFn: func(_ *accountsv1.LoginRequest) (*accountsv1.LoginResponse, error) {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("account locked"))
		}})
		f := newForwarder(t, srv.URL)
		_, err := f.Login(context.Background(), identity.Credentials{Email: "o@x.io", Password: "pw"})
		assert.ErrorIs(t, err, identity.ErrInvalidCredentials)
	})

	t.Run("missing fields short-circuit", func(t *testing.T) {
		f := newForwarder(t, "http://unused")
		_, err := f.Login(context.Background(), identity.Credentials{Email: "", Password: ""})
		assert.ErrorIs(t, err, identity.ErrInvalidInput)
	})

	t.Run("success without token is unavailable", func(t *testing.T) {
		srv := newAuthStub(t, &fakeAccounts{loginFn: func(_ *accountsv1.LoginRequest) (*accountsv1.LoginResponse, error) {
			return &accountsv1.LoginResponse{Account: &accountsv1.Account{Id: "u"}}, nil
		}})
		f := newForwarder(t, srv.URL)
		_, err := f.Login(context.Background(), identity.Credentials{Email: "o@x.io", Password: "pw"})
		assert.ErrorIs(t, err, identity.ErrAuthUnavailable)
	})

	t.Run("resolver failure is unavailable", func(t *testing.T) {
		f := identity.NewForwarder(identity.Config{Resolver: failingResolver{}})
		_, err := f.Login(context.Background(), identity.Credentials{Email: "o@x.io", Password: "pw"})
		assert.ErrorIs(t, err, identity.ErrAuthUnavailable)
	})

	t.Run("stopped authenticator names the dependency and remediation", func(t *testing.T) {
		f := identity.NewForwarder(identity.Config{Resolver: stoppedResolver{}})
		_, err := f.Login(context.Background(), identity.Credentials{Email: "o@x.io", Password: "pw"})
		assert.ErrorIs(t, err, identity.ErrAuthUnavailable)
		assert.Contains(t, err.Error(), "scenario-authenticator is stopped")
		assert.Contains(t, err.Error(), "vrooli scenario start scenario-authenticator")
	})
}

func TestForwarderRegister(t *testing.T) {
	t.Run("created relays token", func(t *testing.T) {
		srv := newAuthStub(t, &fakeAccounts{registerFn: func(_ *accountsv1.RegisterRequest) (*accountsv1.RegisterResponse, error) {
			return &accountsv1.RegisterResponse{
				Account: &accountsv1.Account{Id: "u-1", Email: "new@x.io"},
				Tokens:  &accountsv1.TokenPair{AccessToken: "jwt-new"},
			}, nil
		}})
		f := newForwarder(t, srv.URL)
		o, err := f.Register(context.Background(), identity.Registration{Email: "new@x.io", Password: "Str0ng!pw"})
		require.NoError(t, err)
		assert.Equal(t, "jwt-new", o.Token)
		assert.Equal(t, "new@x.io", o.Email)
	})

	t.Run("already exists is email taken", func(t *testing.T) {
		srv := newAuthStub(t, &fakeAccounts{registerFn: func(_ *accountsv1.RegisterRequest) (*accountsv1.RegisterResponse, error) {
			return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("email already registered"))
		}})
		f := newForwarder(t, srv.URL)
		_, err := f.Register(context.Background(), identity.Registration{Email: "dup@x.io", Password: "pw"})
		assert.ErrorIs(t, err, identity.ErrEmailTaken)
	})

	t.Run("invalid argument relays the authenticator's validation message", func(t *testing.T) {
		srv := newAuthStub(t, &fakeAccounts{registerFn: func(_ *accountsv1.RegisterRequest) (*accountsv1.RegisterResponse, error) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("Password must be at least 8 characters"))
		}})
		f := newForwarder(t, srv.URL)
		_, err := f.Register(context.Background(), identity.Registration{Email: "x@x.io", Password: "weak"})
		assert.ErrorIs(t, err, identity.ErrInvalidInput)
		assert.Contains(t, err.Error(), "at least 8 characters")
	})
}

func TestForwarderRefresh(t *testing.T) {
	srv := newAuthStub(t, &fakeAccounts{refreshFn: func(req *accountsv1.RefreshRequest) (*accountsv1.RefreshResponse, error) {
		assert.Equal(t, "r-1", req.GetRefreshToken())
		return &accountsv1.RefreshResponse{Tokens: &accountsv1.TokenPair{AccessToken: "jwt-new", RefreshToken: "r-2"}}, nil
	}})
	f := newForwarder(t, srv.URL)
	o, err := f.Refresh(context.Background(), "r-1")
	require.NoError(t, err)
	assert.Equal(t, "jwt-new", o.Token)
	assert.Equal(t, "r-2", o.RefreshToken)
}

type failingResolver struct{}

func (failingResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return "", errors.New("scenario not running")
}

type stoppedResolver struct{}

func (stoppedResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return "", &discovery.Error{Kind: discovery.ErrScenarioNotRunning, Scenario: "scenario-authenticator", PortKey: "API_PORT"}
}
