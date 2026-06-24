package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeHealthyWithProtectionAndMinimalQueryLog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireBasicAuth(t, r)
		switch r.URL.Path {
		case "/control/status":
			writeJSON(t, w, map[string]any{
				"version":           "v0.107.77",
				"protection_status": true,
			})
		case "/control/dns_info":
			writeJSON(t, w, map[string]any{"upstream_dns": []string{"https://dns.example/dns-query", "  "}})
		case "/control/querylog_info":
			writeJSON(t, w, map[string]any{"enabled": false})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	report, err := Probe(context.Background(), server.Client(), server.URL, Credentials{
		Username: "admin",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if report.Status != StatusHealthy {
		t.Fatalf("Status = %q, want %q", report.Status, StatusHealthy)
	}
	if report.ProtectionEnabled == nil || !*report.ProtectionEnabled {
		t.Fatalf("ProtectionEnabled = %v, want true", report.ProtectionEnabled)
	}
	if got, want := report.PrivacyPosture, "minimal"; got != want {
		t.Fatalf("PrivacyPosture = %q, want %q", got, want)
	}
	if got, want := len(report.Upstreams), 1; got != want {
		t.Fatalf("Upstreams len = %d, want %d", got, want)
	}
}

func TestProbeClassifiesAuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	report, err := Probe(context.Background(), server.Client(), server.URL, Credentials{})
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if report.Status != StatusAuthFailed {
		t.Fatalf("Status = %q, want %q", report.Status, StatusAuthFailed)
	}
	if report.Authenticated {
		t.Fatal("Authenticated = true, want false")
	}
}

func TestProbeClassifiesSetupRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	report, err := Probe(context.Background(), server.Client(), server.URL, Credentials{})
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if report.Status != StatusSetupRequired {
		t.Fatalf("Status = %q, want %q", report.Status, StatusSetupRequired)
	}
	if !report.SetupRequired {
		t.Fatal("SetupRequired = false, want true")
	}
}

func TestProbeClassifiesSetupRedirectRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/control/install.html", http.StatusFound)
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	report, err := Probe(context.Background(), httpClient, server.URL, Credentials{})
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if report.Status != StatusSetupRequired {
		t.Fatalf("Status = %q, want %q", report.Status, StatusSetupRequired)
	}
	if !report.SetupRequired {
		t.Fatal("SetupRequired = false, want true")
	}
}

func TestProbeDegradedWhenProtectionDisabledOrQueryLogEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/control/status":
			writeJSON(t, w, map[string]any{"protection_status": false})
		case "/control/dns_info":
			writeJSON(t, w, map[string]any{})
		case "/control/querylog_info":
			writeJSON(t, w, map[string]any{"enabled": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	report, err := Probe(context.Background(), server.Client(), server.URL, Credentials{})
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if report.Status != StatusDegraded {
		t.Fatalf("Status = %q, want %q", report.Status, StatusDegraded)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("Warnings empty, want privacy/filtering warning")
	}
}

func requireBasicAuth(t *testing.T, r *http.Request) {
	t.Helper()
	username, password, ok := r.BasicAuth()
	if !ok || username != "admin" || password != "secret" {
		t.Fatalf("BasicAuth = (%q, %q, %t), want admin credentials", username, password, ok)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
