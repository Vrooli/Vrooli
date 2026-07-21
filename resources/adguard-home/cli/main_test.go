package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewAppConfiguresResourceApp(t *testing.T) {
	app, err := newApp()
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	if app == nil {
		t.Fatal("newApp() returned nil app")
	}
	if app.CLI == nil {
		t.Fatal("newApp() returned nil CLI")
	}
	if app.StaleChecker == nil {
		t.Fatal("newApp() returned nil stale checker")
	}
	if app.StaleChecker.SourceContextPath != ".." {
		t.Fatalf("SourceContextPath = %q, want %q", app.StaleChecker.SourceContextPath, "..")
	}
	if app.StaleChecker.ManifestSourcePath != "resource.json" {
		t.Fatalf("ManifestSourcePath = %q, want %q", app.StaleChecker.ManifestSourcePath, "resource.json")
	}
	if len(app.StaleChecker.FreshnessInputs) != 3 {
		t.Fatalf("FreshnessInputs len = %d, want 3", len(app.StaleChecker.FreshnessInputs))
	}
	if got, want := app.StaleChecker.FreshnessInputs[0], "cli/**"; got != want {
		t.Fatalf("FreshnessInputs[0] = %q, want %q", got, want)
	}
	if got, want := app.StaleChecker.FreshnessInputs[1], "resource.json"; got != want {
		t.Fatalf("FreshnessInputs[1] = %q, want %q", got, want)
	}
	if got, want := app.StaleChecker.FreshnessInputs[2], "../../packages/cli-core"; got != want {
		t.Fatalf("FreshnessInputs[2] = %q, want %q", got, want)
	}
}

func TestRunConfigPreviewJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/control/dns_info":
			writeJSON(t, w, map[string]any{"upstream_dns": []string{"1.1.1.1"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var out bytes.Buffer
	err := runConfigPreview([]string{"--base-url", server.URL, "--password", "secret", "--upstream", "9.9.9.9", "--json"}, &out)
	if err != nil {
		t.Fatalf("runConfigPreview() error = %v", err)
	}
	var payload struct {
		Changed bool     `json:"changed"`
		Added   []string `json:"added"`
		Removed []string `json:"removed"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if !payload.Changed {
		t.Fatal("changed = false, want true")
	}
	if got, want := payload.Added[0], "9.9.9.9"; got != want {
		t.Fatalf("added[0] = %q, want %q", got, want)
	}
	if got, want := payload.Removed[0], "1.1.1.1"; got != want {
		t.Fatalf("removed[0] = %q, want %q", got, want)
	}
}

func TestRunClientsListJSONDoesNotIncludeQueries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/control/clients":
			writeJSON(t, w, map[string]any{
				"clients": []map[string]any{{"name": "Laptop", "ids": []string{"192.0.2.10"}}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var out bytes.Buffer
	err := runClientsList([]string{"--base-url", server.URL, "--password", "secret", "--json"}, &out)
	if err != nil {
		t.Fatalf("runClientsList() error = %v", err)
	}
	if strings.Contains(out.String(), "query") {
		t.Fatalf("output contains query-level wording: %s", out.String())
	}
	var payload struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload.Total != 1 {
		t.Fatalf("total = %d, want 1", payload.Total)
	}
}

func TestRunQueryLogPrivacyWarnsWhenEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/control/querylog/config":
			writeJSON(t, w, map[string]any{"enabled": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var out bytes.Buffer
	err := runQueryLogPrivacy([]string{"--base-url", server.URL, "--password", "secret", "--json"}, &out)
	if err != nil {
		t.Fatalf("runQueryLogPrivacy() error = %v", err)
	}
	var payload struct {
		PrivacyPosture string   `json:"privacy_posture"`
		Warnings       []string `json:"warnings"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if got, want := payload.PrivacyPosture, "query_log_enabled"; got != want {
		t.Fatalf("privacy_posture = %q, want %q", got, want)
	}
	if len(payload.Warnings) == 0 {
		t.Fatal("warnings empty, want privacy warning")
	}
}

func TestEnsureBootstrapCredentialsGeneratesAndStoresSecret(t *testing.T) {
	vault := &fakeVault{values: map[string]string{}}
	creds, generated, err := ensureBootstrapCredentials(context.Background(), vault, "secret://secret/resources/adguard-home/admin", "", "", false)
	if err != nil {
		t.Fatalf("ensureBootstrapCredentials() error = %v", err)
	}
	if !generated {
		t.Fatal("generated = false, want true")
	}
	if creds.Username != "admin" {
		t.Fatalf("username = %q, want admin", creds.Username)
	}
	if len(creds.Password) < 32 {
		t.Fatalf("password len = %d, want >= 32", len(creds.Password))
	}
	if got := vault.values["secret/resources/adguard-home/admin:password"]; got != creds.Password {
		t.Fatal("stored password does not match generated password")
	}
	if got := vault.values["secret/resources/adguard-home/admin:username"]; got != "admin" {
		t.Fatalf("stored username = %q, want admin", got)
	}
}

func TestEnsureBootstrapCredentialsReusesExistingSecret(t *testing.T) {
	vault := &fakeVault{values: map[string]string{
		"secret/resources/adguard-home/admin:password": "existing-password",
	}}
	creds, generated, err := ensureBootstrapCredentials(context.Background(), vault, "secret/resources/adguard-home/admin", "operator", "", false)
	if err != nil {
		t.Fatalf("ensureBootstrapCredentials() error = %v", err)
	}
	if generated {
		t.Fatal("generated = true, want false")
	}
	if creds.Password != "existing-password" {
		t.Fatalf("password = %q, want existing password", creds.Password)
	}
	if got := vault.values["secret/resources/adguard-home/admin:username"]; got != "operator" {
		t.Fatalf("stored username = %q, want operator", got)
	}
}

func TestWriteBootstrapReportDoesNotPrintPassword(t *testing.T) {
	var out bytes.Buffer
	report := bootstrapReport{
		Status:             "configured",
		BaseURL:            "http://localhost:3000",
		CredentialRef:      "secret/resources/adguard-home/admin",
		Username:           "admin",
		PasswordGenerated:  true,
		CredentialsStored:  true,
		QueryLogHardening:  "disabled",
		NetworkManagerHint: "network-manager resolver configure-adguard --token-ref secret/resources/adguard-home/admin --json",
	}
	if err := writeBootstrapReport(&out, true, report, nil); err != nil {
		t.Fatalf("writeBootstrapReport() error = %v", err)
	}
	if strings.Contains(out.String(), "secret-password") {
		t.Fatalf("output leaked password: %s", out.String())
	}
	var payload bootstrapReport
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload.CredentialRef == "" || !payload.CredentialsStored {
		t.Fatalf("credential metadata missing: %+v", payload)
	}
}

type fakeVault struct {
	values map[string]string
}

func (v *fakeVault) GetSecret(_ context.Context, path, key string) (string, bool, error) {
	value, ok := v.values[path+":"+key]
	return value, ok, nil
}

func (v *fakeVault) PutSecret(_ context.Context, path, key, value string) error {
	if v.values == nil {
		v.values = map[string]string{}
	}
	v.values[path+":"+key] = value
	return nil
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
