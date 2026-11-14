package session

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	shared "github.com/vrooli/api-core/operatorsession"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func seedLocalEnrollment(t *testing.T) {
	t.Helper()
	private, err := shared.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	store, err := shared.DefaultFileStore()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(private, shared.Enrollment{
		OperatorID:       "operator-1",
		IdentityProvider: "scenario-authenticator",
		Mode:             shared.ModePersonal,
		Reference:        "enrollment-1",
		EnrolledAt:       time.Now().UTC(),
		ScopeCeiling:     []string{"vrooli-bridge:read"},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestClientUsesLocalEnrollmentWithoutAuthenticatorOrBearer(t *testing.T) {
	t.Setenv("VROOLI_OPERATOR_SESSION_DIR", t.TempDir())
	seedLocalEnrollment(t)
	var authorization string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()

	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name: "bridge-test", Version: "0.0.0", DefaultAPIBase: srv.URL, AllowAnonymous: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	core.ConfigFile = &cliutil.ConfigFile{Path: filepath.Join(t.TempDir(), "config.json")}
	core.Config.APIBase = srv.URL
	core.Config.Token = "legacy-access"
	core.Config.RefreshToken = "legacy-refresh"
	transportClient, _ := NewConnectHTTPClient(core)
	transportClient.(*client).exchange = func(context.Context) (string, string, error) {
		t.Fatal("enrolled resolution contacted the authenticator")
		return "", "", nil
	}
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/example", bytes.NewBufferString(`{"value":"body"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := transportClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if !strings.HasPrefix(authorization, shared.LocalSessionScheme+" ") {
		t.Fatalf("authorization = %q, want local session", authorization)
	}
	if strings.Contains(authorization, "legacy-") {
		t.Fatalf("legacy bearer material crossed the transport: %q", authorization)
	}
	if core.Config.Token != "" || core.Config.RefreshToken != "" {
		t.Fatalf("legacy config was not cleared: token=%q refresh=%q", core.Config.Token, core.Config.RefreshToken)
	}
}

func TestClientReportsProviderDiagnosisAfterUnenrolledAuthenticatorFailure(t *testing.T) {
	t.Setenv("VROOLI_OPERATOR_SESSION_DIR", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"code":"unauthenticated","message":"owner required"}`)
	}))
	defer srv.Close()

	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name: "bridge-test", Version: "0.0.0", DefaultAPIBase: srv.URL, AllowAnonymous: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	core.ConfigFile = &cliutil.ConfigFile{Path: filepath.Join(t.TempDir(), "config.json")}
	core.Config.APIBase = srv.URL
	transportClient, _ := NewConnectHTTPClient(core)
	transportClient.(*client).exchange = func(context.Context) (string, string, error) {
		return "", "", io.ErrUnexpectedEOF
	}
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/owner-only", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = transportClient.Do(req)
	if err == nil || !strings.Contains(err.Error(), "provider_unavailable") {
		t.Fatalf("error = %v, want typed provider diagnosis", err)
	}
	if strings.Contains(err.Error(), "auth login") {
		t.Fatalf("provider diagnosis advised legacy auth login: %v", err)
	}
}

func TestClientWithTimeoutAppliesExplicitTransportDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name: "bridge-test", Version: "0.0.0", DefaultAPIBase: srv.URL, AllowAnonymous: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	core.ConfigFile = &cliutil.ConfigFile{Path: filepath.Join(t.TempDir(), "config.json")}
	core.Config.APIBase = srv.URL
	core.Config.Token = "test-access"

	transportClient, _ := NewConnectHTTPClientWithTimeout(core, 10*time.Millisecond)
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/example", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transportClient.Do(req); err == nil {
		t.Fatal("expected explicit transport timeout")
	}
}
