package bundle

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"scenario-to-desktop/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func newTestClient(handler http.Handler) support.Dependencies {
	server := httptest.NewServer(handler)
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             "scenario-to-desktop-test",
		Version:          "test",
		Description:      "test",
		DefaultAPIBase:   server.URL,
		AllowAnonymous:   true,
		CommandGroups:    func(*cliapp.ScenarioApp) []cliapp.CommandGroup { return nil },
		SubcommandGroups: func(*cliapp.ScenarioApp) []cliapp.SubcommandGroup { return nil },
	})
	if err != nil {
		panic(err)
	}
	return support.Dependencies{Core: func() *cliapp.ScenarioApp { return app }}
}

func TestClean_MissingScenarioArg(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))

	err := cmds.Clean([]string{})
	if err == nil {
		t.Fatal("expected error for missing scenario argument")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestClean_EmptyScenarioArg(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))

	// Whitespace-only scenario should fail
	err := cmds.Clean([]string{"  "})
	if err == nil {
		t.Fatal("expected error for whitespace-only scenario")
	}
	if !strings.Contains(err.Error(), "scenario is required") {
		t.Errorf("error = %q, want 'scenario is required'", err.Error())
	}
}

func TestClean_PipelineIDRequiredForStaging(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))

	tests := []struct {
		name string
		args []string
	}{
		{"staging without pipeline-id", []string{"my-scenario", "--location-mode", "staging"}},
		{"temp without pipeline-id", []string{"my-scenario", "--location-mode", "temp"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := cmds.Clean(tc.args)
			if err == nil {
				t.Fatal("expected error for missing pipeline-id")
			}
			if !strings.Contains(err.Error(), "--pipeline-id is required") {
				t.Errorf("error = %q, want '--pipeline-id is required'", err.Error())
			}
		})
	}
}

func TestClean_PipelineIDNotRequiredForProper(t *testing.T) {
	var receivedPath string
	var receivedBody map[string]interface{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"path":    "/tmp/bundle",
			"removed": true,
		})
	})

	cmds := New(newTestClient(handler))
	err := cmds.Clean([]string{"my-scenario"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(receivedPath, "/scenarios/my-scenario/bundle/clean") {
		t.Errorf("request path = %q, want to contain '/scenarios/my-scenario/bundle/clean'", receivedPath)
	}

	if receivedBody["framework"] != "electron" {
		t.Errorf("framework = %v, want 'electron'", receivedBody["framework"])
	}
	if receivedBody["location_mode"] != "proper" {
		t.Errorf("location_mode = %v, want 'proper'", receivedBody["location_mode"])
	}
}

func TestClean_StagingWithPipelineID(t *testing.T) {
	var receivedBody map[string]interface{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"path":    "/tmp/staging-bundle",
			"removed": true,
		})
	})

	cmds := New(newTestClient(handler))
	err := cmds.Clean([]string{"my-scenario", "--location-mode", "staging", "--pipeline-id", "pipe-123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody["location_mode"] != "staging" {
		t.Errorf("location_mode = %v, want 'staging'", receivedBody["location_mode"])
	}
	if receivedBody["pipeline_id"] != "pipe-123" {
		t.Errorf("pipeline_id = %v, want 'pipe-123'", receivedBody["pipeline_id"])
	}
}

func TestClean_CustomFramework(t *testing.T) {
	var receivedBody map[string]interface{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"path":    "/tmp/bundle",
			"removed": false,
		})
	})

	cmds := New(newTestClient(handler))
	err := cmds.Clean([]string{"my-scenario", "--framework", "tauri"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody["framework"] != "tauri" {
		t.Errorf("framework = %v, want 'tauri'", receivedBody["framework"])
	}
}

func TestClean_APIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Clean([]string{"my-scenario"})
	if err == nil {
		t.Fatal("expected error from API failure")
	}
}

func TestClean_InvalidFlag(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Clean([]string{"--unknown-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}
