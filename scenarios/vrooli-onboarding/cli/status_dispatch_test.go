package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"vrooli-onboarding/cli/domains"
)

// recordingAPI answers health probes and records every request path so a test
// can prove which RPCs a command did (or did not) make.
func recordingAPI(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "health") {
			_, _ = w.Write([]byte(`{"status":"healthy","readiness":true,"service":"vrooli-onboarding"}`))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"invalid_argument","message":"unexpected request in test"}`))
	}))
	t.Cleanup(server.Close)
	return server, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), paths...)
	}
}

// TestWizardCommitSendsTheSelectionDocument pins the contract Bridge's remote
// apply relies on: `wizard commit --selection <path>` reads the document and
// sends it to AcceptRecommendation. Bound to the generic connect-rpc builder,
// it failed before any request with "field selection has unsupported kind
// message", so a re-onboarded node could never apply its selection.
func TestWizardCommitSendsTheSelectionDocument(t *testing.T) {
	server, requests := recordingAPI(t)
	selectionPath := filepath.Join(t.TempDir(), "selection.json")
	if err := os.WriteFile(selectionPath, []byte(`{"schema_version":"v1","scenarios":["system-monitor"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	err = app.Run([]string{"--api-base", server.URL, "wizard", "commit", "--selection", selectionPath, "--json"})
	if err != nil && strings.Contains(err.Error(), "unsupported kind message") {
		t.Fatalf("wizard commit is bound to the generic builder: %v", err)
	}
	sent := false
	for _, path := range requests() {
		if strings.Contains(path, "SelectionService/AcceptRecommendation") {
			sent = true
		}
	}
	if !sent {
		t.Fatalf("wizard commit never sent the selection; err = %v, requests = %v", err, requests())
	}
}

// TestTopLevelStatusRunsHealthCheck pins the help contract "status — Check API
// health". The flat apply group once registered its GetApplyRun "status" after
// cli-core's health command, and the later name won the lookup.
func TestTopLevelStatusRunsHealthCheck(t *testing.T) {
	server, requests := recordingAPI(t)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if err := app.Run([]string{"--api-base", server.URL, "status"}); err != nil {
		t.Fatalf("status returned %v; requests = %v", err, requests())
	}
	sawHealth := false
	for _, path := range requests() {
		if strings.Contains(path, "ApplyService") || strings.Contains(path, "ReadinessService") {
			t.Fatalf("status must not call %s; requests = %v", path, requests())
		}
		if strings.Contains(path, "health") {
			sawHealth = true
		}
	}
	if !sawHealth {
		t.Fatalf("status did not probe API health; requests = %v", requests())
	}
}

// TestApplyRunIDIsAUsageError proves the commands that need an apply run id
// reject a missing id before any RPC is made.
func TestApplyRunIDIsAUsageError(t *testing.T) {
	for _, sub := range []string{"status", "cancel"} {
		server, requests := recordingAPI(t)
		app, err := NewApp()
		if err != nil {
			t.Fatalf("NewApp() error: %v", err)
		}
		err = app.Run([]string{"--api-base", server.URL, "apply", sub, "--target", "local"})
		if err == nil || !strings.Contains(err.Error(), "--run-id") {
			t.Fatalf("apply %s without --run-id error = %v, want a usage error naming --run-id", sub, err)
		}
		// cli-core's API preflight may probe health; the apply RPC must not run.
		for _, path := range requests() {
			if strings.Contains(path, "ApplyService") {
				t.Fatalf("apply %s without --run-id called %s; the usage error must come first", sub, path)
			}
		}
	}
}

// TestTopLevelCommandNamesAreUnique guards the root cause: cli-core resolves
// top-level names through a map, so a flat manifest group that repeats a name
// silently shadows the earlier command (including the built-in health status).
func TestTopLevelCommandNamesAreUnique(t *testing.T) {
	commandGroups, subcommandGroups, err := domains.ManifestCommandGroups(nil, manifestBytes)
	if err != nil {
		t.Fatalf("assemble command tree: %v", err)
	}
	owner := map[string]string{"help": "cli-core", "version": "cli-core", "status": "cli-core health", "configure": "cli-core"}
	for _, group := range subcommandGroups {
		if prior, ok := owner[group.Name]; ok {
			t.Errorf("subcommand group %q collides with %s", group.Name, prior)
		}
		owner[group.Name] = "group " + group.Name
	}
	for _, group := range commandGroups {
		for _, cmd := range group.Commands {
			for _, name := range append([]string{cmd.Name}, cmd.Aliases...) {
				if prior, ok := owner[name]; ok {
					t.Errorf("flat command %q in group %q shadows %s", name, group.Title, prior)
				}
				owner[name] = "flat group " + group.Title
			}
		}
	}
}
