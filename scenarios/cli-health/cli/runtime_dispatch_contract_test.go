package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// TestCompositionSearchRuntimeDispatchAndHelpContract exercises the CLI Health
// command through the same ScenarioApp registration and cli-core dispatch path
// used by the executable. Its API fixture is deliberately in-memory: source
// composition can remain PARTIAL and carry degradations even when a command is
// actually dispatchable. The static analyzer must not imply that installed
// help or runtime dispatch was proven by source joins alone.
func TestCompositionSearchRuntimeDispatchAndHelpContract(t *testing.T) {
	var compositionCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"status":"ok","service":"cli-health","version":"0.1.0","readiness":true,"dependencies":{}}`)
		case "/api/v1/cli-composition/search":
			if r.Method != http.MethodPost {
				http.Error(w, "expected POST", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				Query    string `json:"query"`
				Scenario string `json:"scenario"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			if request.Query != "ScenarioApp command-group dispatch" || request.Scenario != "cli-health" {
				http.Error(w, "unexpected search scope", http.StatusBadRequest)
				return
			}
			compositionCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"results":[{"title":"ScenarioApp and command-group registration","text":"The source analyzer found a partial group-to-manifest join.","attestation":"PARTIAL","evidence":[{"path":"cli/domains/domains.go","line":27}]}],"degradations":[{"code":"runtime_help_unverified","message":"Static source analysis does not verify installed help output."}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("CLI_HEALTH_API_BASE", server.URL)

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp(): %v", err)
	}

	// These assertions pass through cli-core's real help renderer with the
	// command groups registered from the embedded manifest.
	for _, tc := range []struct {
		name string
		args []string
		want []string
	}{
		{name: "root", args: []string{"--help"}, want: []string{"search"}},
		{name: "group", args: []string{"search", "--help"}, want: []string{"composition-search"}},
		{name: "leaf", args: []string{"search", "composition-search", "--help"}, want: []string{"composition-search", "query", "scenario"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if err := app.core.CLI.RunWithWriters(tc.args, &stdout, &stderr); err != nil {
				t.Fatalf("RunWithWriters(%v): %v; stderr=%q", tc.args, err, stderr.String())
			}
			for _, want := range tc.want {
				if !strings.Contains(stdout.String(), want) {
					t.Errorf("help for %v omitted %q:\n%s", tc.args, want, stdout.String())
				}
			}
		})
	}
	if calls := compositionCalls.Load(); calls != 0 {
		t.Fatalf("help invoked composition API %d time(s); help must remain local", calls)
	}

	var stdout, stderr bytes.Buffer
	if err := app.core.CLI.RunWithWriters([]string{"search", "composition-search", "ScenarioApp command-group dispatch", "cli-health"}, &stdout, &stderr); err != nil {
		t.Fatalf("composition dispatch: %v; stderr=%q", err, stderr.String())
	}
	if calls := compositionCalls.Load(); calls != 1 {
		t.Fatalf("composition endpoint calls = %d, want 1", calls)
	}
	for _, want := range []string{"Found 1 static CLI composition result(s); runtime dispatch remains unverified.", "PARTIAL", "cli/domains/domains.go", "runtime_help_unverified"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("human report omitted %q:\n%s", want, stdout.String())
		}
	}
}
