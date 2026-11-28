package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/identity"
	identityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/identity/identity_v1connect"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

// identityService is a fake IdentityService backing the CLI's connect client.
type identityService struct {
	loginResp *identityv1.LoginResponse
	loginErr  error
	regResp   *identityv1.RegisterResponse
	regErr    error

	lastLogin    *identityv1.LoginRequest
	lastRegister *identityv1.RegisterRequest
}

func (s *identityService) Login(_ context.Context, req *connect.Request[identityv1.LoginRequest]) (*connect.Response[identityv1.LoginResponse], error) {
	s.lastLogin = req.Msg
	if s.loginErr != nil {
		return nil, s.loginErr
	}
	return connect.NewResponse(s.loginResp), nil
}

func (s *identityService) Register(_ context.Context, req *connect.Request[identityv1.RegisterRequest]) (*connect.Response[identityv1.RegisterResponse], error) {
	s.lastRegister = req.Msg
	if s.regErr != nil {
		return nil, s.regErr
	}
	return connect.NewResponse(s.regResp), nil
}

// identityAPI mounts the fake IdentityService as a real Connect HTTP handler.
func identityAPI(t *testing.T, svc identityconnect.IdentityServiceHandler) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := identityconnect.NewIdentityServiceHandler(svc)
	mux.Handle(path, handler)
	return mux
}

func TestRunLogin(t *testing.T) {
	t.Run("stores the issued token", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		svc := &identityService{loginResp: &identityv1.LoginResponse{Token: "jwt-1", Email: "o@x.io", UserId: "u-1"}}
		core := clitest.NewTestApp(t, identityAPI(t, svc))

		require.NoError(t, runLogin(core, []string{"--email", "o@x.io", "--password", "pw"}))
		require.Equal(t, "jwt-1", core.Config.Token)
		require.Equal(t, "o@x.io", svc.lastLogin.GetEmail())
	})

	t.Run("surfaces an authentication failure and stores nothing", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		svc := &identityService{loginErr: connect.NewError(connect.CodeUnauthenticated, context.DeadlineExceeded)}
		core := clitest.NewTestApp(t, identityAPI(t, svc))

		require.Error(t, runLogin(core, []string{"--email", "o@x.io", "--password", "bad"}))
		require.Empty(t, core.Config.Token)
	})

	t.Run("requires email and password", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		core := clitest.NewTestApp(t, identityAPI(t, &identityService{}))
		require.Error(t, runLogin(core, []string{"--email", "o@x.io"}))
	})
}

func TestRunRegister(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	svc := &identityService{regResp: &identityv1.RegisterResponse{Token: "jwt-new", Email: "new@x.io", UserId: "u-2"}}
	core := clitest.NewTestApp(t, identityAPI(t, svc))

	require.NoError(t, runRegister(core, []string{"--email", "new@x.io", "--password", "Str0ng!pw", "--username", "New"}))
	require.Equal(t, "jwt-new", core.Config.Token)
	require.Equal(t, "New", svc.lastRegister.GetUsername())
}

func TestRunLogout(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	core := clitest.NewTestApp(t, identityAPI(t, &identityService{}))
	core.Config.Token = "jwt-x"
	require.NoError(t, core.SaveConfig())

	require.NoError(t, runLogout(core, nil))
	require.Empty(t, core.Config.Token)
}

func TestRunWhoami(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	core := clitest.NewTestApp(t, identityAPI(t, &identityService{}))

	// whoami decodes the stored token locally — no server round-trip.
	payload, _ := json.Marshal(map[string]any{"user_id": "u-9", "email": "owner@x.io", "roles": []string{"user"}, "exp": time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC).Unix()})
	token := "h." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
	core.Config.Token = token

	require.NoError(t, runWhoami(core, nil))

	// not signed in → clear error
	core.Config.Token = ""
	require.Error(t, runWhoami(core, nil))
}

func TestWhoamiReportUsesInjectedClockForExpiry(t *testing.T) {
	expires := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		now  time.Time
		want string
	}{
		{name: "before expiry", now: expires.Add(-time.Second), want: "Session expires: " + time.Unix(expires.Unix(), 0).Format(time.RFC3339)},
		{name: "at expiry", now: expires, want: "Session expires: " + time.Unix(expires.Unix(), 0).Format(time.RFC3339)},
		{name: "after expiry", now: expires.Add(time.Second), want: "Session EXPIRED at: " + time.Unix(expires.Unix(), 0).Format(time.RFC3339)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := whoamiReport(ownerTokenClaims{UserID: "owner-1", Exp: expires.Unix()}, clock{Now: func() time.Time { return tc.now }})
			require.Equal(t, []string{"Owner ID: owner-1", tc.want}, report.Results)
		})
	}
}
