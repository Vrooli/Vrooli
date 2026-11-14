package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func TestParseGrantTTL(t *testing.T) {
	cases := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"30d", 30 * 24 * time.Hour, false},
		{"720h", 720 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"", 0, true},
		{"0d", 0, true},
		{"-5d", 0, true},
		{"banana", 0, true},
		{"0", 0, true},
	}
	for _, tc := range cases {
		got, err := parseGrantTTL(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("parseGrantTTL(%q) expected error, got %v", tc.in, got)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Fatalf("parseGrantTTL(%q) = %v, %v; want %v", tc.in, got, err, tc.want)
		}
	}
}

// --local-owner must refuse inside an identified agent run so an agent cannot
// elevate itself to a human operator to mint a grant.
func TestLocalOwnerRefusedInsideAgentRun(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "some-agent-identity-token")
	if _, err := localOwnerAPIForResource(nil, standingGrantResource); err == nil {
		t.Fatal("localOwnerAPIForResource must refuse inside an identified agent run")
	}
	// Sanity: with the env cleared the guard does not trip on this check (it will
	// fail later at the socket exchange, which is fine — the guard itself passed).
	_ = os.Unsetenv(cliutil.EnvIdentityToken)
}

func TestStandingGrantListTextReportsMoreRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/standing-grants" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"grants":[{"id":"grant-1","subject":"prompt-manager","state":"approved","max_concurrent_runs":3,"expires_at":"tomorrow"}],"has_more":true}`))
	}))
	t.Cleanup(server.Close)
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
	app := &App{services: NewServices(api)}

	previousStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	err = app.standingGrantsList(nil)
	_ = writer.Close()
	os.Stdout = previousStdout
	if err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output, []byte("More standing grants exist; only the first 64 are shown.")) || !strings.Contains(string(output), "grant-1") {
		t.Fatalf("text list did not report the bounded response accurately: %s", output)
	}
}

func TestStandingGrantDiagnosticUsesFixedReadRouteAndRendersBothModes(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/standing-grants/diagnose/prompt-manager-run-create" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.ContentLength > 0 {
			t.Fatalf("diagnostic request unexpectedly carried a body: %d bytes", r.ContentLength)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"outcome":"would_refuse_limit","reason":"grant_limit_reached","observed_at":"2026-10-17T17:00:00Z","limits":[{"dimension":"max_concurrent_runs","current":2,"bound":2,"contributors":{"count":2,"runs":[{"run_id":"run-123","status":"parked"}],"truncated":true}}]}`))
	}))
	t.Cleanup(server.Close)
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
	app := &App{services: NewServices(api)}

	for _, tc := range []struct {
		name string
		args []string
		want []string
	}{
		{name: "text", args: []string{"diagnose-prompt-manager"}, want: []string{"would_refuse_limit", "grant_limit_reached", "max_concurrent_runs: 2.00 / 2.00", "counted runs: 2 (truncated)", "run-123 [parked]"}},
		{name: "json", args: []string{"diagnose-prompt-manager", "--json"}, want: []string{"\"outcome\": \"would_refuse_limit\"", "\"current\": 2", "\"bound\": 2", "\"run_id\": \"run-123\"", "\"status\": \"parked\""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, err := captureStandingGrantStdout(func() error { return app.cmdStandingGrants(tc.args) })
			if err != nil {
				t.Fatalf("diagnostic command failed: %v", err)
			}
			for _, want := range tc.want {
				if !strings.Contains(output, want) {
					t.Fatalf("output %q missing %q", output, want)
				}
			}
		})
	}
	if requests != 2 {
		t.Fatalf("want exactly two read requests, got %d", requests)
	}
}

func TestStandingGrantDiagnosticReturnsErrorForUnavailableRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/standing-grants/diagnose/prompt-manager-run-create" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		http.Error(w, `{"outcome":"unavailable","reason":"authority_state_unavailable"}`, http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
	app := &App{services: NewServices(api)}
	if _, err := captureStandingGrantStdout(func() error { return app.cmdStandingGrants([]string{"diagnose-prompt-manager"}) }); err == nil {
		t.Fatal("unavailable diagnostic was presented as a successful result")
	}
}

func captureStandingGrantStdout(run func() error) (string, error) {
	previousStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = writer
	runErr := run()
	_ = writer.Close()
	os.Stdout = previousStdout
	output, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if runErr != nil {
		return string(output), runErr
	}
	return string(output), readErr
}
