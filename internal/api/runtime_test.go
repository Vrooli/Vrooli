package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/control"
)

func TestResolveRepoRootCanonicalizesContractDescendantOverride(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)

	nested := filepath.Join(fixture.Root, "cmd", "vrooli-api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	t.Setenv("VROOLI_ROOT", nested)

	if got := ResolveRepoRoot(); got != fixture.Root {
		t.Fatalf("ResolveRepoRoot() = %q, want %q", got, fixture.Root)
	}
}

func TestBuildRuntimeAppAppliesOverrides(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	startReport := control.StartReport{Message: "started"}
	stopReport := control.StopReport{Message: "stopped"}

	app := BuildRuntimeApp(RuntimeConfig{
		Root: root,
		Home: home,
		LookPathFn: func(name string) (string, error) {
			return "/custom/" + name, nil
		},
		CommandFn: func(context.Context, string, ...string) ([]byte, error) {
			return []byte("ok"), nil
		},
		StartAllScenariosFn: func() (control.StartReport, error) {
			return startReport, nil
		},
		StopAllScenariosFn: func() (control.StopReport, error) {
			return stopReport, nil
		},
		StopScenarioFn: func(name string) error {
			if name != "alpha" {
				t.Fatalf("StopScenarioFn name = %q", name)
			}
			return nil
		},
	})

	if app.Root != root {
		t.Fatalf("app.Root = %q, want %q", app.Root, root)
	}
	if app.Home != home {
		t.Fatalf("app.Home = %q, want %q", app.Home, home)
	}
	if path, err := app.LookPathFn("vrooli"); err != nil || path != "/custom/vrooli" {
		t.Fatalf("LookPathFn() = %q, %v", path, err)
	}
	if out, err := app.CommandFn(context.Background(), "vrooli"); err != nil || string(out) != "ok" {
		t.Fatalf("CommandFn() = %q, %v", string(out), err)
	}
	if report, err := app.StartAllScenariosFn(); err != nil || report.Message != startReport.Message {
		t.Fatalf("StartAllScenariosFn() = %#v, %v", report, err)
	}
	if report, err := app.StopAllScenariosFn(); err != nil || report.Message != stopReport.Message {
		t.Fatalf("StopAllScenariosFn() = %#v, %v", report, err)
	}
	if err := app.StopScenarioFn("alpha"); err != nil {
		t.Fatalf("StopScenarioFn() error = %v", err)
	}
}

func TestPerformHealthCheckPostgresSupported(t *testing.T) {
	app := BuildRuntimeApp(RuntimeConfig{
		Root: t.TempDir(),
		Home: t.TempDir(),
		LookPathFn: func(file string) (string, error) {
			return "/usr/bin/vrooli", nil
		},
		CommandFn: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte(`{"installed":true,"running":true,"healthy":true}`), nil
		},
	})

	check := HealthCheckConfig{
		Name:     "postgres_connection",
		Type:     "postgres",
		Target:   "vrooli",
		Critical: true,
		Timeout:  3000,
	}

	if err := app.PerformHealthCheck(check, "business-health", map[string]int{}); err != nil {
		t.Fatalf("PerformHealthCheck() error = %v", err)
	}
}

func TestRuntimeAppRouterServesCoreRoutes(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)

	app := BuildRuntimeApp(RuntimeConfig{
		Root: fixture.Root,
		Home: t.TempDir(),
	})
	router := app.Router()

	testCases := []struct {
		name    string
		method  string
		path    string
		allowed []int
	}{
		{name: "Health", method: http.MethodGet, path: "/health", allowed: []int{http.StatusOK, http.StatusServiceUnavailable}},
		{name: "ListScenarios", method: http.MethodGet, path: "/scenarios", allowed: []int{http.StatusOK}},
		{name: "ListApps", method: http.MethodGet, path: "/apps", allowed: []int{http.StatusOK}},
		{name: "ListResources", method: http.MethodGet, path: "/resources", allowed: []int{http.StatusOK}},
		{name: "ProcessMetrics", method: http.MethodGet, path: "/metrics/processes", allowed: []int{http.StatusOK}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			router.ServeHTTP(rec, req)

			for _, allowed := range tc.allowed {
				if rec.Code == allowed {
					return
				}
			}
			t.Fatalf("%s status = %d, allowed %v", tc.path, rec.Code, tc.allowed)
		})
	}
}
