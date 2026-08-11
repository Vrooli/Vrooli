package testutil_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	clitest "scenario-to-android/cli/internal/testutil"
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

	app := clitest.NewTestApp(t, handler)
	body, err := app.Get("/probe", nil)
	require.NoError(t, err)
	require.Equal(t, 1, hits, "test server should have received exactly one request")
	require.Contains(t, string(body), "ok")
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

	app := clitest.NewTestApp(t, handler,
		clitest.WithAppName("custom"),
		clitest.WithAppVersion("9.9.9"),
		clitest.WithAppDescription("custom test"),
	)
	require.NotNil(t, app)
	require.NotNil(t, app.CLI, "CLI must be wired so app.Run is callable")
}
