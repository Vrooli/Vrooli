package cliapptest

import (
	"net/http"
	"strings"
	"testing"
)

// TestNewTestApp_ResolvesAPIBaseToTestServer pins the canonical
// guarantee: a request issued through the returned ScenarioApp lands
// on the test handler, with no additional env-var wiring required by
// the test.
func TestNewTestApp_ResolvesAPIBaseToTestServer(t *testing.T) {
	var hits int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	app := NewTestApp(t, handler)
	body, err := app.Get("/probe", nil)
	if err != nil {
		t.Fatalf("app.Get: %v", err)
	}
	if hits != 1 {
		t.Fatalf("test server hits = %d, want 1", hits)
	}
	if !strings.Contains(string(body), "ok") {
		t.Fatalf("body = %q, want ok", body)
	}
}

// TestNewTestApp_OptionsOverrideDefaults proves the With* helpers
// reach the cli-core wiring. We can't easily inspect the configured
// metadata without exporting cli-core internals, so the smoke is that
// construction succeeds with the overrides applied (a malformed
// option would fail StandardScenarioOptions validation).
func TestNewTestApp_OptionsOverrideDefaults(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	app := NewTestApp(t, handler,
		WithAppName("custom"),
		WithAppVersion("9.9.9"),
		WithAppDescription("custom test"),
	)
	if app == nil || app.CLI == nil {
		t.Fatal("CLI must be wired so app.Run is callable")
	}
}
