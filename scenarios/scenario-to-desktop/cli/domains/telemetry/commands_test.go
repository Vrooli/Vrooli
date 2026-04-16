package telemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// jsonHandler returns an http.HandlerFunc that responds with the given JSON body.
func jsonHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

// --- Ingest ---

func TestIngest_MissingArgs(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))

	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{}},
		{"scenario but no file", []string{"my-app"}},
		{"file but no scenario", []string{"--file", "/tmp/data.jsonl"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := cmds.Ingest(tc.args)
			if err == nil {
				t.Fatal("expected error for missing arguments")
			}
		})
	}
}

func TestIngest_ValidJSONL(t *testing.T) {
	var receivedBody map[string]interface{}
	var receivedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","events_ingested":2}`))
	})

	// Create a temp JSONL file with valid events
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "events.jsonl")
	content := `{"event":"startup","ts":1000}
{"event":"shutdown","ts":2000}
`
	if err := os.WriteFile(jsonlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmds := New(newTestClient(handler))
	err := cmds.Ingest([]string{"my-app", "--file", jsonlPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(receivedPath, "/deployment/telemetry") {
		t.Errorf("path = %q, want to contain '/deployment/telemetry'", receivedPath)
	}
	events, ok := receivedBody["events"].([]interface{})
	if !ok {
		t.Fatal("expected events array in request body")
	}
	if len(events) != 2 {
		t.Errorf("events count = %d, want 2", len(events))
	}
	if receivedBody["scenario_name"] != "my-app" {
		t.Errorf("scenario_name = %v, want 'my-app'", receivedBody["scenario_name"])
	}
	if receivedBody["source"] != "cli" {
		t.Errorf("source = %v, want 'cli' (default)", receivedBody["source"])
	}
}

func TestIngest_CustomSource(t *testing.T) {
	var receivedBody map[string]interface{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","events_ingested":1}`))
	})

	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "events.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(`{"event":"test"}`+"\n"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmds := New(newTestClient(handler))
	err := cmds.Ingest([]string{"my-app", "--file", jsonlPath, "--source", "runtime"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBody["source"] != "runtime" {
		t.Errorf("source = %v, want 'runtime'", receivedBody["source"])
	}
}

func TestIngest_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "events.jsonl")
	// Mix of valid and invalid JSONL lines
	content := `not-json
{"event":"valid"}
also not json {{{
`
	if err := os.WriteFile(jsonlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","events_ingested":1}`))
	})

	cmds := New(newTestClient(handler))
	// Should succeed since there is at least one valid event
	err := cmds.Ingest([]string{"my-app", "--file", jsonlPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIngest_AllMalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "events.jsonl")
	content := "not-json-at-all\nalso-bad\n"
	if err := os.WriteFile(jsonlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Ingest([]string{"my-app", "--file", jsonlPath})
	if err == nil {
		t.Fatal("expected error when no valid events found")
	}
	if !strings.Contains(err.Error(), "no valid events") {
		t.Errorf("error = %q, want to contain 'no valid events'", err.Error())
	}
}

func TestIngest_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "events.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Ingest([]string{"my-app", "--file", jsonlPath})
	if err == nil {
		t.Fatal("expected error for empty file")
	}
	if !strings.Contains(err.Error(), "no valid events") {
		t.Errorf("error = %q, want to contain 'no valid events'", err.Error())
	}
}

func TestIngest_FileNotFound(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Ingest([]string{"my-app", "--file", "/nonexistent/path/events.jsonl"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("error = %q, want to contain 'failed to read file'", err.Error())
	}
}

func TestIngest_JSONOutput(t *testing.T) {
	handler := jsonHandler(`{"status":"ok","events_ingested":1}`)

	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "events.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(`{"event":"test"}`+"\n"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmds := New(newTestClient(handler))
	err := cmds.Ingest([]string{"my-app", "--file", jsonlPath, "--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIngest_BlankLinesSkipped(t *testing.T) {
	var receivedBody map[string]interface{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","events_ingested":1}`))
	})

	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "events.jsonl")
	content := "\n\n{\"event\":\"test\"}\n\n\n"
	if err := os.WriteFile(jsonlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmds := New(newTestClient(handler))
	err := cmds.Ingest([]string{"my-app", "--file", jsonlPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	events, ok := receivedBody["events"].([]interface{})
	if !ok {
		t.Fatal("expected events array in request body")
	}
	if len(events) != 1 {
		t.Errorf("events count = %d, want 1 (blank lines should be skipped)", len(events))
	}
}

// --- Summary ---

func TestSummary_MissingScenario(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Summary([]string{})
	if err == nil {
		t.Fatal("expected error for missing scenario")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestSummary_Success(t *testing.T) {
	var receivedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"summary":"data"}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Summary([]string{"my-app"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(receivedPath, "/deployment/telemetry/my-app/summary") {
		t.Errorf("path = %q, want to contain '/deployment/telemetry/my-app/summary'", receivedPath)
	}
}

func TestSummary_JSONOutput(t *testing.T) {
	handler := jsonHandler(`{"summary":"data"}`)

	cmds := New(newTestClient(handler))
	err := cmds.Summary([]string{"my-app", "--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Insights ---

func TestInsights_MissingScenario(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Insights([]string{})
	if err == nil {
		t.Fatal("expected error for missing scenario")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestInsights_Success(t *testing.T) {
	var receivedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"insights":[]}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Insights([]string{"my-app"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(receivedPath, "/deployment/telemetry/my-app/insights") {
		t.Errorf("path = %q, want to contain '/deployment/telemetry/my-app/insights'", receivedPath)
	}
}

// --- Tail ---

func TestTail_MissingScenario(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Tail([]string{})
	if err == nil {
		t.Fatal("expected error for missing scenario")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestTail_DefaultLimit(t *testing.T) {
	var receivedQuery string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"events":[]}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Tail([]string{"my-app"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(receivedQuery, "limit=200") {
		t.Errorf("query = %q, want to contain 'limit=200' (default)", receivedQuery)
	}
}

func TestTail_CustomLimit(t *testing.T) {
	var receivedQuery string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"events":[]}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Tail([]string{"my-app", "--limit", "50"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(receivedQuery, "limit=50") {
		t.Errorf("query = %q, want to contain 'limit=50'", receivedQuery)
	}
}

func TestTail_JSONOutput(t *testing.T) {
	handler := jsonHandler(`{"events":[]}`)

	cmds := New(newTestClient(handler))
	err := cmds.Tail([]string{"my-app", "--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Download ---

func TestDownload_MissingScenario(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Download([]string{})
	if err == nil {
		t.Fatal("expected error for missing scenario")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestDownload_ToStdout(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("telemetry-data-here"))
	})

	cmds := New(newTestClient(handler))
	// Without --output, should print to stdout (no error)
	err := cmds.Download([]string{"my-app"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDownload_ToFile(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("telemetry-csv-data"))
	})

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "telemetry.csv")

	cmds := New(newTestClient(handler))
	err := cmds.Download([]string{"my-app", "--output", outPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(data) != "telemetry-csv-data" {
		t.Errorf("content = %q, want 'telemetry-csv-data'", string(data))
	}
}

func TestDownload_APIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"fail"}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Download([]string{"my-app"})
	if err == nil {
		t.Fatal("expected error from API failure")
	}
}

// --- Delete ---

func TestDelete_MissingScenario(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Delete([]string{})
	if err == nil {
		t.Fatal("expected error for missing scenario")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestDelete_Success(t *testing.T) {
	var receivedPath string
	var receivedMethod string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"deleted":true}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Delete([]string{"my-app"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", receivedMethod)
	}
	if !strings.Contains(receivedPath, "/deployment/telemetry/my-app") {
		t.Errorf("path = %q, want to contain '/deployment/telemetry/my-app'", receivedPath)
	}
}

func TestDelete_JSONOutput(t *testing.T) {
	handler := jsonHandler(`{"deleted":true}`)

	cmds := New(newTestClient(handler))
	err := cmds.Delete([]string{"my-app", "--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
