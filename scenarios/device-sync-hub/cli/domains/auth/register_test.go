package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
)

// newAuthTestApp builds a ScenarioApp with config isolated to a temp dir so
// SaveConfig never touches the real CLI config. The API base is irrelevant —
// auth talks to the authenticator, resolved per-test via --auth-api-base.
func newAuthTestApp(t *testing.T) *cliapp.ScenarioApp {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:           "device-sync-hub-authtest",
		Version:        "0.0.0-test",
		Description:    "auth domain test",
		DefaultAPIBase: "http://127.0.0.1:0",
		AllowAnonymous: true,
	})
	require.NoError(t, err)
	return core
}

// authServer fakes scenario-authenticator's login + validate endpoints.
type authServer struct {
	loginStatus int
	loginBody   any
	loginSeen   map[string]string

	validStatus int
	validBody   any
}

func (a *authServer) start(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &a.loginSeen)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(orDefault(a.loginStatus, http.StatusOK))
		_ = json.NewEncoder(w).Encode(a.loginBody)
	})
	mux.HandleFunc("/api/v1/auth/validate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(orDefault(a.validStatus, http.StatusOK))
		_ = json.NewEncoder(w).Encode(a.validBody)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func orDefault(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

// [REQ:REQ-P0-005] Owner identity is delegated to scenario-authenticator; the CLI stores the returned token.
func TestLoginStoresToken(t *testing.T) {
	core := newAuthTestApp(t)
	as := &authServer{loginBody: authResponse{Success: true, Token: "jwt-owner-1", User: struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}{ID: "u1", Email: "owner@example.com"}}}
	srv := as.start(t)

	err := runLogin(core, []string{"--email", "owner@example.com", "--password", "hunter2", "--auth-api-base", srv.URL})
	require.NoError(t, err)
	require.Equal(t, "jwt-owner-1", core.Config.Token, "owner token persisted to config")
	require.Equal(t, "owner@example.com", as.loginSeen["email"], "email forwarded to authenticator")
	require.Equal(t, "hunter2", as.loginSeen["password"], "password forwarded to authenticator")
}

func TestLoginRequiresEmailAndPassword(t *testing.T) {
	core := newAuthTestApp(t)
	err := runLogin(core, []string{"--email", "owner@example.com"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "required")
}

func TestLoginFailureLeavesNoToken(t *testing.T) {
	core := newAuthTestApp(t)
	as := &authServer{loginBody: authResponse{Success: false, Message: "invalid credentials"}}
	srv := as.start(t)

	err := runLogin(core, []string{"--email", "x@y.z", "--password", "bad", "--auth-api-base", srv.URL})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid credentials")
	require.Empty(t, core.Config.Token, "a failed login must not store a token")
}

func TestLoginSurfacesAuthRejection(t *testing.T) {
	core := newAuthTestApp(t)
	as := &authServer{loginStatus: http.StatusUnauthorized, loginBody: map[string]string{"error": "nope"}}
	srv := as.start(t)

	err := runLogin(core, []string{"--email", "x@y.z", "--password", "bad", "--auth-api-base", srv.URL})
	require.Error(t, err)
	require.Contains(t, err.Error(), "rejected the credentials")
	require.Empty(t, core.Config.Token)
}

func TestLogoutClearsToken(t *testing.T) {
	core := newAuthTestApp(t)
	core.Config.Token = "stale"
	require.NoError(t, core.SaveConfig())

	require.NoError(t, runLogout(core, nil))
	require.Empty(t, core.Config.Token)
}

func TestWhoamiRequiresToken(t *testing.T) {
	core := newAuthTestApp(t)
	err := runWhoami(core, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not signed in")
}

func TestWhoamiValidatesStoredToken(t *testing.T) {
	core := newAuthTestApp(t)
	core.Config.Token = "jwt-owner-1"
	as := &authServer{validBody: validateResponse{Valid: true, UserID: "u1", Email: "owner@example.com", Roles: []string{"user"}}}
	srv := as.start(t)

	require.NoError(t, runWhoami(core, []string{"--auth-api-base", srv.URL}))
}

func TestWhoamiRejectsInvalidToken(t *testing.T) {
	core := newAuthTestApp(t)
	core.Config.Token = "expired"
	as := &authServer{validBody: validateResponse{Valid: false}}
	srv := as.start(t)

	err := runWhoami(core, []string{"--auth-api-base", srv.URL})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid or expired")
}

func TestResolveAuthBaseURLPrefersFlagThenEnv(t *testing.T) {
	// Flag override wins outright (and trailing slash is trimmed).
	got, err := resolveAuthBaseURL(t.Context(), "http://flag.example/")
	require.NoError(t, err)
	require.Equal(t, "http://flag.example", got)

	// With no flag, the AUTH_SERVICE_URL env var is used.
	t.Setenv(envAuthServiceURL, "http://env.example")
	got, err = resolveAuthBaseURL(t.Context(), "")
	require.NoError(t, err)
	require.Equal(t, "http://env.example", got)
}
