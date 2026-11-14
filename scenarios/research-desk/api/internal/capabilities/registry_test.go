package capabilities

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"

	"research-desk/internal/home"
)

// The real manifest declares the services the desk calls: Web Console for
// launch, Vrooli Events and Notification Hub for delivery, and Data Backup
// Manager for research-home recovery.
func TestLoadProjectsTheManifest(t *testing.T) {
	defs, err := Load("../../../.vrooli/service.json")
	require.NoError(t, err)
	slugs := map[string]Def{}
	for _, def := range defs {
		slugs[def.DependencySlug] = def
	}
	called := map[string]string{
		"web-console":         "ignore",
		"vrooli-events":       "ignore",
		"notification-hub":    "ignore",
		"data-backup-manager": "try_start",
	}
	require.Len(t, slugs, len(called), "service.json declares exactly the services the desk calls")
	for slug, startupPolicy := range called {
		def, ok := slugs[slug]
		require.True(t, ok, "service.json must declare %s", slug)
		require.False(t, def.Required, "%s must be optional", slug)
		require.Equal(t, startupPolicy, def.StartupPolicy)
		require.Equal(t, "vrooli scenario start "+slug+" --json", def.OperatorCommand)
		require.NotEmpty(t, def.Uses, "%s must say what the desk uses it for", slug)
	}
	require.NoError(t, NewRegistry(defs, fixedPort{err: errors.New("unused")}).Validate())
}

type fixedPort struct {
	port int
	err  error
}

func (f fixedPort) ResolveScenarioPort(context.Context, string, string) (int, error) {
	return f.port, f.err
}

func TestHealthChecker(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer healthy.Close()
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
	defer failing.Close()

	cases := []struct {
		name     string
		resolver fixedPort
		want     capabilityregistry.Status
		reason   string
	}{
		{"healthy", fixedPort{port: portOf(t, healthy)}, capabilityregistry.StatusAvailable, "is healthy"},
		{"unhealthy", fixedPort{port: portOf(t, failing)}, capabilityregistry.StatusUnavailable, "answered 503"},
		{"not running", fixedPort{err: errors.New("no port")}, capabilityregistry.StatusUnavailable, "web-console is not running: no port"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, reason := HealthChecker{Slug: "web-console", Resolver: tc.resolver, Client: http.DefaultClient}.Check(context.Background())
			require.Equal(t, tc.want, status)
			require.Contains(t, reason, tc.reason)
			require.NotContains(t, reason, "vrooli", "a reason names no command")
		})
	}
}

func portOf(t *testing.T, s *httptest.Server) int {
	t.Helper()
	u, err := url.Parse(s.URL)
	require.NoError(t, err)
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)
	return port
}

// A peer is reached only when the registry says it is available; its
// reason is what launch and the courier see, with the start command beside it.
func TestReachAsksTheRegistry(t *testing.T) {
	defs, err := Load("../../../.vrooli/service.json")
	require.NoError(t, err)
	ctx := context.Background()

	healthy := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer healthy.Close()
	up := fixedPort{port: portOf(t, healthy)}
	base, err := Reach(NewRegistry(defs, up), up, "web-console")(ctx)
	require.NoError(t, err)
	require.Equal(t, "http://localhost:"+strconv.Itoa(up.port), base)

	down := fixedPort{err: errors.New("no port")}
	_, err = Reach(NewRegistry(defs, down), down, "notification-hub")(ctx)
	require.ErrorContains(t, err, "notification-hub is unavailable")
	require.NotContains(t, err.Error(), "vrooli", "the command travels beside the reason")
	require.Equal(t, "vrooli scenario start notification-hub", home.Command(err))

	_, err = Reach(NewRegistry(defs, up), up, "ollama")(ctx)
	require.ErrorContains(t, err, "not a dependency")
	_, err = Reach(nil, up, "web-console")(ctx)
	require.ErrorContains(t, err, "capability registry is not loaded")
}

// Every health check is bounded even when port discovery hangs.
func TestAHealthCheckIsBounded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*CheckTimeout)
	defer cancel()
	status, reason := HealthChecker{Slug: "web-console", Resolver: hang{}, Client: http.DefaultClient}.Check(ctx)
	require.Equal(t, capabilityregistry.StatusUnknown, status, "a check that timed out is unknown, not down")
	require.Contains(t, reason, "not reached just now (timed out")
	require.NoError(t, ctx.Err(), "the check returned within its own timeout")
}

// A scenario whose lifecycle lists it running but whose /health is slow is
// unknown, and no command to start it is offered; one that answers badly is
// unavailable, still with no start command, since it runs.
func TestARunningScenarioIsNeverToldToStart(t *testing.T) {
	defs, err := Load("../../../.vrooli/service.json")
	require.NoError(t, err)
	ctx := context.Background()
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	defer slow.Close()
	defer close(release)
	running := fixedPort{port: portOf(t, slow)}
	status, reason := HealthChecker{Slug: "web-console", Resolver: running, Client: &http.Client{Timeout: 50 * time.Millisecond}}.Check(ctx)
	require.Equal(t, capabilityregistry.StatusUnknown, status)
	require.Contains(t, reason, "web-console was not reached just now (timed out")

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
	defer failing.Close()
	up := fixedPort{port: portOf(t, failing)}
	_, err = Reach(NewRegistry(defs, up), up, "web-console")(ctx)
	require.ErrorContains(t, err, "answered 503")
	require.Empty(t, home.Command(err), "a running scenario is not told to start")
	require.Empty(t, StartHint(ctx, up, "web-console"))
	require.Empty(t, StartHint(ctx, hang{}, "web-console"), "a lookup that timed out does not say it is stopped")
	require.Equal(t, "vrooli scenario start web-console", StartHint(ctx, fixedPort{err: errors.New("no port")}, "web-console"))

	_, err = Reach(NewRegistry(defs, hang{}), hang{}, "web-console")(ctx)
	require.ErrorIs(t, err, ErrNotReached, "a timed-out check stays unknown for launch and the courier")
	require.Contains(t, err.Error(), "web-console was not reached just now")
	require.Empty(t, home.Command(err))
}

type hang struct{}

func (hang) ResolveScenarioPort(ctx context.Context, _, _ string) (int, error) {
	<-ctx.Done()
	return 0, ctx.Err()
}
