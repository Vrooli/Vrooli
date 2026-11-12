package capabilities

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

// The real manifest declares exactly the services the desk calls (launch
// through Web Console; the courier sends through Vrooli Events and reads
// Notification Hub), all optional,
// each with an operator action.
func TestLoadProjectsTheManifest(t *testing.T) {
	defs, err := Load("../../../.vrooli/service.json")
	require.NoError(t, err)
	slugs := map[string]Def{}
	for _, def := range defs {
		slugs[def.DependencySlug] = def
	}
	called := []string{"web-console", "vrooli-events", "notification-hub"}
	require.Len(t, slugs, len(called), "service.json declares exactly the services the desk calls")
	for _, slug := range called {
		def, ok := slugs[slug]
		require.True(t, ok, "service.json must declare %s", slug)
		require.False(t, def.Required, "%s must be optional (charter I5)", slug)
		require.Equal(t, "ignore", def.StartupPolicy, "%s must not be started with the desk", slug)
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
		{"not running", fixedPort{err: errors.New("no port")}, capabilityregistry.StatusUnavailable, "vrooli scenario start web-console"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, reason := HealthChecker{Slug: "web-console", Resolver: tc.resolver, Client: http.DefaultClient}.Check(context.Background())
			require.Equal(t, tc.want, status)
			require.Contains(t, reason, tc.reason)
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

// E5: a peer is reached only when the registry says it is available; its
// reason (with the start command) is what launch and the courier see.
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
	require.ErrorContains(t, err, "vrooli scenario start notification-hub")

	_, err = Reach(NewRegistry(defs, up), up, "ollama")(ctx)
	require.ErrorContains(t, err, "not a dependency")
	_, err = Reach(nil, up, "web-console")(ctx)
	require.ErrorContains(t, err, "capability registry is not loaded")
}

// Every health check is bounded even when port discovery hangs.
func TestAHealthCheckIsBounded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*CheckTimeout)
	defer cancel()
	status, _ := HealthChecker{Slug: "web-console", Resolver: hang{}, Client: http.DefaultClient}.Check(ctx)
	require.Equal(t, capabilityregistry.StatusUnavailable, status)
	require.NoError(t, ctx.Err(), "the check returned within its own timeout")
}

type hang struct{}

func (hang) ResolveScenarioPort(ctx context.Context, _, _ string) (int, error) {
	<-ctx.Done()
	return 0, ctx.Err()
}
