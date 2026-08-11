// Package testutil contains CLI-only fixtures shared by onboarding tests.
// Production packages must not import this package.
package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// NewTestApp provides a real HTTP boundary for domain command tests. Keeping
// this in the test-only package prevents production commands from acquiring a
// fake transport or test configuration dependency.
func NewTestApp(tb testing.TB, handler http.Handler) *cliapp.ScenarioApp {
	tb.Helper()
	srv := httptest.NewServer(handler)
	tb.Cleanup(srv.Close)
	tb.Setenv("API_BASE_URL", srv.URL)
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:           "vrooli-onboarding-test",
		Version:        "0.0.0-test",
		Description:    "onboarding CLI test",
		DefaultAPIBase: srv.URL,
		AllowAnonymous: true,
	})
	if err != nil {
		tb.Fatalf("construct test CLI: %v", err)
	}
	return app
}

// WriteJSON writes a private JSON fixture without exposing it through a
// production command path.
func WriteJSON(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
