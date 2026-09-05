package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIStatusCommand(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`{"status":"healthy","readiness":true}`))
		case "/api/v1/status":
			_, _ = w.Write([]byte(`{"status":"warning","platform":{"platform":"linux","hasDocker":true,"supportsSystemd":true,"supportsCloudflared":true,"isWsl":false,"isHeadlessServer":true},"summary":{"total":3,"ok":2,"warning":1,"critical":0},"checks":[{"checkId":"infra-dns","status":"warning","message":"DNS degraded"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	app.core.APIOverride = ts.URL
	app.core.CLI.SetStaleChecker(nil)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"status"}); err != nil {
			t.Fatalf("Run(status) error = %v", err)
		}
	})

	if !strings.Contains(output, "Overall status: WARNING") {
		t.Fatalf("status output missing overall status: %s", output)
	}
	if !strings.Contains(output, "infra-dns") {
		t.Fatalf("status output missing warning check: %s", output)
	}
}

func TestCLITickJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`{"status":"healthy","readiness":true}`))
		case "/api/v1/tick":
			_, _ = w.Write([]byte(`{"success":true,"status":"ok","summary":{"total":1,"ok":1,"warning":0,"critical":0},"results":[{"checkId":"infra-network","status":"ok","message":"network ok"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	app.core.APIOverride = ts.URL
	app.core.CLI.SetStaleChecker(nil)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"tick", "--json"}); err != nil {
			t.Fatalf("Run(tick --json) error = %v", err)
		}
	})

	if !strings.Contains(output, `"success": true`) {
		t.Fatalf("tick json output missing success field: %s", output)
	}
}

func TestCLIWatchdogInstallJSON(t *testing.T) {
	bin := t.TempDir()
	stub := filepath.Join(bin, "vrooli")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nprintf '{\"message\": \"installed\"}\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	app.core.CLI.SetStaleChecker(nil)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"install", "--json"}); err != nil {
			t.Fatalf("Run(install --json) error = %v", err)
		}
	})

	if !strings.Contains(output, `"message": "installed"`) {
		t.Fatalf("install json output missing message: %s", output)
	}
}

func TestCLIIncidentRemediationOutcomeJSON(t *testing.T) {
	var sawOutcome bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`{"status":"healthy","readiness":true}`))
		case "/api/v1/incidents/inc_test/remediations/ubuntu-nvidia-kernel-module-mismatch/outcome":
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			sawOutcome = true
			_, _ = w.Write([]byte(`{"id":"inc_test","status":"open","outcome":{"remediationId":"ubuntu-nvidia-kernel-module-mismatch","status":"verified","note":"healthy","reportedAt":"2026-05-08T12:00:00Z"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	app.core.APIOverride = ts.URL
	app.core.CLI.SetStaleChecker(nil)

	output := captureStdout(t, func() {
		err := app.Run([]string{"incidents", "remediation", "outcome", "inc_test", "ubuntu-nvidia-kernel-module-mismatch", "--status", "verified", "--note", "healthy", "--json"})
		if err != nil {
			t.Fatalf("Run(incidents remediation outcome --json) error = %v", err)
		}
	})

	if !sawOutcome {
		t.Fatal("CLI did not call outcome endpoint")
	}
	if !strings.Contains(output, `"status": "verified"`) {
		t.Fatalf("outcome json output missing status: %s", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	_ = w.Close()
	os.Stdout = original
	return <-done
}
