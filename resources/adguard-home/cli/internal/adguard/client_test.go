package adguard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPreviewUpstreamsDiffsAndTestsWithoutMutation(t *testing.T) {
	var sawTest bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireBasicAuth(t, r)
		switch r.URL.Path {
		case DNSInfoEndpoint:
			if r.Method != http.MethodGet {
				t.Fatalf("DNSInfo method = %s, want GET", r.Method)
			}
			writeJSON(t, w, map[string]any{"upstream_dns": []string{"1.1.1.1", "9.9.9.9"}})
		case TestUpstreamDNSEndpoint:
			sawTest = true
			if r.Method != http.MethodPost {
				t.Fatalf("TestUpstreamDNS method = %s, want POST", r.Method)
			}
			var req UpstreamTestRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if got, want := len(req.UpstreamDNS), 2; got != want {
				t.Fatalf("UpstreamDNS len = %d, want %d", got, want)
			}
			writeJSON(t, w, map[string]string{"8.8.8.8": "OK", "9.9.9.9": "OK"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, Credentials{Username: "admin", Password: "secret"}, WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	preview, err := client.PreviewUpstreams(context.Background(), []string{"8.8.8.8", "9.9.9.9"}, true)
	if err != nil {
		t.Fatalf("PreviewUpstreams() error = %v", err)
	}
	if !sawTest {
		t.Fatal("expected upstream test endpoint to be called")
	}
	if !preview.Changed {
		t.Fatal("Changed = false, want true")
	}
	if got, want := preview.Added[0], "8.8.8.8"; got != want {
		t.Fatalf("Added[0] = %q, want %q", got, want)
	}
	if got, want := preview.Removed[0], "1.1.1.1"; got != want {
		t.Fatalf("Removed[0] = %q, want %q", got, want)
	}
	if !preview.ApprovalRequired || !preview.MutationRequired {
		t.Fatalf("ApprovalRequired/MutationRequired = %t/%t, want true/true", preview.ApprovalRequired, preview.MutationRequired)
	}
}

func TestQueryLogConfigFallsBackToLegacyEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case QueryLogConfigEndpoint:
			http.NotFound(w, r)
		case LegacyQueryLogInfoEndpoint:
			writeJSON(t, w, map[string]any{"enabled": false})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, Credentials{}, WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	config, endpoint, code, err := client.QueryLogConfig(context.Background())
	if err != nil {
		t.Fatalf("QueryLogConfig() error = %v", err)
	}
	if code != http.StatusOK {
		t.Fatalf("code = %d, want 200", code)
	}
	if endpoint != LegacyQueryLogInfoEndpoint {
		t.Fatalf("endpoint = %q, want %q", endpoint, LegacyQueryLogInfoEndpoint)
	}
	if config.Enabled == nil || *config.Enabled {
		t.Fatalf("Enabled = %v, want false", config.Enabled)
	}
}

func TestClientsCleansEmptyEntries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case ClientsEndpoint:
			writeJSON(t, w, map[string]any{
				"clients": []map[string]any{
					{"name": "Laptop", "ids": []string{"192.0.2.10", " "}, "tags": []string{"work"}},
					{"name": " "},
				},
				"auto_clients": []map[string]any{
					{"name": "Phone", "ip": "192.0.2.11", "source": "rdns"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, Credentials{}, WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	report, code, err := client.Clients(context.Background())
	if err != nil {
		t.Fatalf("Clients() error = %v", err)
	}
	if code != http.StatusOK {
		t.Fatalf("code = %d, want 200", code)
	}
	if got, want := report.Total, 2; got != want {
		t.Fatalf("Total = %d, want %d", got, want)
	}
	if got, want := report.Configured[0].IDs[0], "192.0.2.10"; got != want {
		t.Fatalf("configured ID = %q, want %q", got, want)
	}
	if got, want := report.Auto[0].IP, "192.0.2.11"; got != want {
		t.Fatalf("auto IP = %q, want %q", got, want)
	}
}

func TestInitialConfigureAndQueryLogDisable(t *testing.T) {
	var sawCheck bool
	var sawConfigure bool
	var sawQueryLogDisable bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case InstallCheckConfigEndpoint:
			sawCheck = true
			if r.Method != http.MethodPost {
				t.Fatalf("check method = %s, want POST", r.Method)
			}
			var req CheckConfigRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode check request: %v", err)
			}
			if got, want := req.DNS.Port, 53; got != want {
				t.Fatalf("DNS port = %d, want %d", got, want)
			}
			writeJSON(t, w, map[string]any{
				"dns": map[string]any{"status": "", "can_autofix": false},
				"web": map[string]any{"status": "", "can_autofix": false},
			})
		case InstallConfigureEndpoint:
			sawConfigure = true
			if r.Method != http.MethodPost {
				t.Fatalf("configure method = %s, want POST", r.Method)
			}
			var req InitialConfiguration
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode configure request: %v", err)
			}
			if req.Username != "admin" || req.Password != "secret" {
				t.Fatalf("credentials = (%q, %q), want admin/secret", req.Username, req.Password)
			}
			w.WriteHeader(http.StatusOK)
		case QueryLogConfigUpdateEndpoint:
			sawQueryLogDisable = true
			requireBasicAuth(t, r)
			if r.Method != http.MethodPut {
				t.Fatalf("querylog method = %s, want PUT", r.Method)
			}
			var req QueryLogConfig
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode querylog request: %v", err)
			}
			if req.Enabled == nil || *req.Enabled {
				t.Fatalf("Enabled = %v, want false", req.Enabled)
			}
			if req.Interval == nil {
				t.Fatal("Interval not preserved in query-log update")
			}
			w.WriteHeader(http.StatusOK)
		case QueryLogConfigEndpoint:
			requireBasicAuth(t, r)
			writeJSON(t, w, map[string]any{
				"enabled":             true,
				"interval":            7776000000,
				"ignored":             []string{},
				"ignored_enabled":     false,
				"anonymize_client_ip": false,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, Credentials{}, WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	cfg := InitialConfiguration{
		DNS:      AddressInfo{IP: "0.0.0.0", Port: 53},
		Web:      AddressInfo{IP: "0.0.0.0", Port: 3000},
		Username: "admin",
		Password: "secret",
	}
	if _, code, err := client.CheckInitialConfig(context.Background(), cfg); err != nil || code != http.StatusOK {
		t.Fatalf("CheckInitialConfig() = code %d err %v, want 200 nil", code, err)
	}
	if code, err := client.ConfigureInitial(context.Background(), cfg); err != nil || code != http.StatusOK {
		t.Fatalf("ConfigureInitial() = code %d err %v, want 200 nil", code, err)
	}
	authClient, err := NewClient(server.URL, Credentials{Username: "admin", Password: "secret"}, WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatalf("NewClient(auth) error = %v", err)
	}
	if code, err := authClient.DisableQueryLog(context.Background()); err != nil || code != http.StatusOK {
		t.Fatalf("DisableQueryLog() = code %d err %v, want 200 nil", code, err)
	}
	if !sawCheck || !sawConfigure || !sawQueryLogDisable {
		t.Fatalf("saw check/configure/querylog = %t/%t/%t, want all true", sawCheck, sawConfigure, sawQueryLogDisable)
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
