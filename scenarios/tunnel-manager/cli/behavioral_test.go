package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestApp creates an App wired to a local httptest server so commands
// exercise real HTTP round-trips without touching a live API.
func newTestApp(t *testing.T, handler http.Handler) *App {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	app.core.APIOverride = ts.URL
	return app
}

func TestRouteCreateMissingPort(t *testing.T) {
	app := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	err := app.Run([]string{"route", "create", "--subdomain", "test", "--scenario", "test"})
	if err == nil {
		t.Fatal("expected error for missing --port, got nil")
	}
	if !strings.Contains(err.Error(), "--port") {
		t.Fatalf("error should mention --port, got: %v", err)
	}
}

func TestRouteCreateInvalidPort(t *testing.T) {
	app := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	err := app.Run([]string{"route", "create", "--port", "abc"})
	if err == nil {
		t.Fatal("expected error for invalid --port, got nil")
	}
	if !strings.Contains(err.Error(), "abc") {
		t.Fatalf("error should mention the bad value, got: %v", err)
	}
}

func TestRouteDeleteWithYes(t *testing.T) {
	app := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/routes/1") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))

	err := app.Run([]string{"route", "delete", "1", "--yes"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRouteDeleteNotFound(t *testing.T) {
	app := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not_found","message":"route 999 not found"}`))
	}))

	err := app.Run([]string{"route", "delete", "999", "--yes"})
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
}

func TestRouteDeleteCancelledWithoutYes(t *testing.T) {
	app := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("delete should not reach the server when the prompt is cancelled")
	}))
	app.Stdin = strings.NewReader("\n")

	err := app.Run([]string{"route", "delete", "1"})
	if err != nil {
		t.Fatalf("cancellation should not return error, got: %v", err)
	}
}

func TestRouteDeleteConfirmedViaStdin(t *testing.T) {
	called := false
	app := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	app.Stdin = strings.NewReader("y\n")

	err := app.Run([]string{"route", "delete", "1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected DELETE request to be sent after confirmation")
	}
}

func TestRouteGetInvalidID(t *testing.T) {
	app := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad_request","message":"invalid route ID"}`))
	}))

	err := app.Run([]string{"route", "get", "abc"})
	if err == nil {
		t.Fatal("expected error for invalid ID, got nil")
	}
}

func TestMetricsLatestNoData(t *testing.T) {
	app := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"no_data"}`))
	}))

	err := app.Run([]string{"metrics", "latest"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecoveryTriggerForce(t *testing.T) {
	var receivedBody map[string]any
	app := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if err := json.Unmarshal(body, &receivedBody); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"action":"restart","outcome":"success"}`))
	}))

	err := app.Run([]string{"recovery", "trigger", "--force"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	force, ok := receivedBody["force"]
	if !ok {
		t.Fatal("request body missing 'force' field")
	}
	if force != true {
		t.Fatalf("expected force=true, got %v", force)
	}
}
