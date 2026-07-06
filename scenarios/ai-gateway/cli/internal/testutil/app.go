package testutil

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
)

// NewTestApp wires a *cliapp.ScenarioApp pointed at a real httptest
// server fronting handler. The default APIBase resolves to the test
// server, so any code path that calls core.Get / core.Request reaches
// it without further configuration.
//
// Use this whenever a domain handler test needs the full ScenarioApp
// surface (the standard CRUD shape). For tests that exercise only the
// raw API client, NewAPIServer + a hand-built ScenarioApp is the
// lighter alternative.
//
// The returned *cliapp.ScenarioApp:
//   - resolves APIBase to the supplied test server's URL
//   - has Name / Version set to stable test defaults; override via
//     opts when a test asserts on metadata.
//   - allows anonymous requests (AllowAnonymous: true) so tests don't
//     need to seed a token.
//
// Server lifecycle is tied to t.Cleanup via NewAPIServer; tests do
// not need to close anything.
func NewTestApp(tb testing.TB, handler http.Handler, opts ...TestAppOption) *cliapp.ScenarioApp {
	tb.Helper()
	srv := NewAPIServer(tb, handler)

	cfg := testAppConfig{
		name:        "scenario-test",
		version:     "0.0.0-test",
		description: "scenario CLI test",
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:           cfg.name,
		Version:        cfg.version,
		Description:    cfg.description,
		DefaultAPIBase: srv.URL,
		AllowAnonymous: true,
	})
	require.NoError(tb, err)
	return core
}

// TestAppOption mutates the StandardScenarioOptions NewTestApp passes
// to cli-core. Use the With* helpers below; constructing a custom
// option is rarely needed.
type TestAppOption func(*testAppConfig)

type testAppConfig struct {
	name        string
	version     string
	description string
}

// WithAppName overrides the default test app name. Use when a test
// asserts on the appName metadata cli-core surfaces in --version /
// --help output.
func WithAppName(name string) TestAppOption {
	return func(c *testAppConfig) { c.name = name }
}

// WithAppVersion overrides the default test app version.
func WithAppVersion(version string) TestAppOption {
	return func(c *testAppConfig) { c.version = version }
}

// WithAppDescription overrides the default test app description.
func WithAppDescription(description string) TestAppOption {
	return func(c *testAppConfig) { c.description = description }
}

// ManifestBytes reads the CLI manifest from a domain test package.
func ManifestBytes(tb testing.TB) []byte {
	tb.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(tb, err, "read manifest")
	return raw
}
