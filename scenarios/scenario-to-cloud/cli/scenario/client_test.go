package scenario

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestDeploymentReportUsesSharedPeerResolver(t *testing.T) {
	t.Setenv("VROOLI_HOME", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_API_BASE_URL", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/scenarios/demo/deployment" || r.URL.Query().Get("refresh") != "true" {
			t.Errorf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	var targets, ports []string
	restore := cliutil.SetPortLookupRunner(func(_ context.Context, target, port string) cliutil.ScenarioPortOutcome {
		targets = append(targets, target)
		ports = append(ports, port)
		return cliutil.ScenarioPortOutcome{Port: serverURL.Port()}
	})
	defer restore()

	if _, _, err := NewClient(nil).DeploymentReport(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0] != "scenario-dependency-analyzer" || len(ports) != 1 || ports[0] != "API_PORT" {
		t.Fatalf("shared resolver targets=%v ports=%v", targets, ports)
	}
}

func TestDeploymentReportUsesConfiguredStaticBasePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/api/v1/scenarios/demo/deployment" || r.URL.Query().Get("refresh") != "true" || r.URL.Query().Get("tenant") != "test" {
			t.Errorf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_API_BASE_URL", server.URL+"/v2/?tenant=test")

	if _, _, err := NewClient(nil).DeploymentReport(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
}
