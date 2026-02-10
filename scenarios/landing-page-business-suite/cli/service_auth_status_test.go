package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func withAPIBase(t *testing.T, base string) {
	t.Helper()
	const envKey = "LANDING_PAGE_BUSINESS_SUITE_API_BASE"
	previous, had := os.LookupEnv(envKey)
	if err := os.Setenv(envKey, base); err != nil {
		t.Fatalf("set env %s: %v", envKey, err)
	}
	t.Cleanup(func() {
		if !had {
			_ = os.Unsetenv(envKey)
			return
		}
		_ = os.Setenv(envKey, previous)
	})
}

func TestServiceAuthStatusRequireEnabledPassesWhenConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/usage/health" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"healthy":true,"database_connected":true,"service_auth_configured":true,"service_auth_mode":"token"}`))
	}))
	defer server.Close()

	withAPIBase(t, server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	if err := app.cmdServiceAuthStatus([]string{"--require-enabled"}); err != nil {
		t.Fatalf("cmdServiceAuthStatus returned error: %v", err)
	}
}

func TestServiceAuthStatusRequireEnabledFailsWhenDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/usage/health" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"healthy":true,"database_connected":true,"service_auth_configured":false,"service_auth_mode":"disabled"}`))
	}))
	defer server.Close()

	withAPIBase(t, server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	err = app.cmdServiceAuthStatus([]string{"--require-enabled"})
	if err == nil {
		t.Fatal("expected error when service auth is disabled")
	}
	if !strings.Contains(err.Error(), "Next steps:") {
		t.Fatalf("expected next-step guidance, got: %v", err)
	}
}

func withAdminSession(t *testing.T, app *App, apiBase string) {
	t.Helper()
	if err := app.saveAdminSession(adminSessionConfig{
		Session: "test-admin-session",
		APIBase: apiBase,
	}); err != nil {
		t.Fatalf("save admin session: %v", err)
	}
}

func TestRemoteProfilesLoginAcceptsTagSelector(t *testing.T) {
	var sawList, sawLogin bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/remote-profiles":
			sawList = true
			_, _ = w.Write([]byte(`{"profiles":[{"id":12,"tag":"prod"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/admin/remote-profiles/12/login":
			sawLogin = true
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode login payload: %v", err)
			}
			if payload["email"] != "admin@example.com" {
				t.Fatalf("unexpected email: %q", payload["email"])
			}
			if payload["password"] != "secret" {
				t.Fatalf("unexpected password payload")
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	withAPIBase(t, server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	withAdminSession(t, app, server.URL)

	if err := app.cmdRemoteProfilesLogin([]string{"--tag", "prod", "--email", "admin@example.com", "--password", "secret"}); err != nil {
		t.Fatalf("cmdRemoteProfilesLogin returned error: %v", err)
	}
	if !sawList {
		t.Fatal("expected remote profile list request for --tag selector")
	}
	if !sawLogin {
		t.Fatal("expected remote profile login request")
	}
}

func TestRemoteProfilesLoginAcceptsProfileIDFlag(t *testing.T) {
	var sawList, sawLogin bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/remote-profiles":
			sawList = true
			t.Fatalf("did not expect list request when --profile-id is used")
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/admin/remote-profiles/22/login":
			sawLogin = true
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	withAPIBase(t, server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	withAdminSession(t, app, server.URL)

	if err := app.cmdRemoteProfilesLogin([]string{"--profile-id", "22", "--email", "admin@example.com", "--password", "secret"}); err != nil {
		t.Fatalf("cmdRemoteProfilesLogin returned error: %v", err)
	}
	if sawList {
		t.Fatal("unexpected list request when --profile-id is provided")
	}
	if !sawLogin {
		t.Fatal("expected remote profile login request")
	}
}

func TestRemoteProfilesTestAcceptsTagSelector(t *testing.T) {
	var sawList, sawTest bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/remote-profiles":
			sawList = true
			_, _ = w.Write([]byte(`{"profiles":[{"id":"44","tag":"prod"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/admin/remote-profiles/44/test":
			sawTest = true
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	withAPIBase(t, server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	withAdminSession(t, app, server.URL)

	if err := app.cmdRemoteProfilesTest([]string{"--tag", "prod"}); err != nil {
		t.Fatalf("cmdRemoteProfilesTest returned error: %v", err)
	}
	if !sawList {
		t.Fatal("expected remote profile list request for --tag selector")
	}
	if !sawTest {
		t.Fatal("expected remote profile test request")
	}
}

func TestDeployReadinessReportsFailuresWhenUnconfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/usage/health":
			_, _ = w.Write([]byte(`{"healthy":true,"database_connected":true,"service_auth_configured":false,"service_auth_mode":"disabled"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	withAPIBase(t, server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	err = app.cmdDeployReadiness([]string{"--domain", "example.com"})
	if err == nil {
		t.Fatal("expected deploy readiness to fail without admin session/service auth")
	}
	if !strings.Contains(err.Error(), "deploy readiness checks failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeployReadinessPassesWithProfileTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/admin/download-storage/test":
			_, _ = w.Write([]byte(`{"ok":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/remote-profiles":
			_, _ = w.Write([]byte(`{"profiles":[{"id":"9","tag":"prod"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/admin/remote-profiles/9/test":
			_, _ = w.Write([]byte(`{"ok":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/usage/health":
			_, _ = w.Write([]byte(`{"healthy":true,"database_connected":true,"service_auth_configured":true,"service_auth_mode":"token"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	withAPIBase(t, server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	withAdminSession(t, app, server.URL)

	if err := app.cmdDeployReadiness([]string{"--profile-tag", "prod"}); err != nil {
		t.Fatalf("cmdDeployReadiness returned error: %v", err)
	}
}

func TestDeployReadinessDoesNotSuggestRemoteLoginWhenBlockedByAdminSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/usage/health":
			_, _ = w.Write([]byte(`{"healthy":true,"database_connected":true,"service_auth_configured":false,"service_auth_mode":"disabled"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	withAPIBase(t, server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	originalStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe() error: %v", pipeErr)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = originalStdout })

	runErr := app.cmdDeployReadiness([]string{"--profile-tag", "prod", "--domain", "example.com"})
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	if runErr == nil {
		t.Fatal("expected deploy readiness to fail")
	}
	out := buf.String()
	if !strings.Contains(out, "[BLOCKED] remote_profile_session") {
		t.Fatalf("expected blocked remote profile check, got output:\n%s", out)
	}
	if strings.Contains(out, "remote-profiles-login --tag prod") {
		t.Fatalf("did not expect remote login suggestion when admin session is missing, got output:\n%s", out)
	}
}
