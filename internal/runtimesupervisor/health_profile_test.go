package runtimesupervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	apihealth "github.com/vrooli/api-core/health"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestSupervisorHealthProbeUsesInstanceProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(apihealth.Response{
			Status: apihealth.StatusHealthy, Service: "alpha", Timestamp: time.Now().Format(time.RFC3339), Readiness: true,
		})
	}))
	defer server.Close()
	_, portText, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf(`{
  "service": {"name": "alpha"},
  "ports": {
    "api": {"env_var": "API_PORT"},
    "ui": {"env_var": "UI_PORT"}
  },
  "lifecycle": {"health": {"checks": [
    {"name": "api_endpoint", "type": "http", "target": "http://127.0.0.1:${API_PORT}/health", "critical": true},
    {"name": "ui_endpoint", "type": "http", "target": "http://127.0.0.1:${UI_PORT}/health"}
  ]}},
  "components": {"api": {"run": {"port": "api"}}, "ui": {"run": {"port": "ui"}}},
  "profiles": {
    "default": {"components": ["api", "ui"], "ports": ["api", "ui"]},
    "lab": {"components": ["api"], "ports": ["api"]}
  }
}`)
	root := t.TempDir()
	servicePath := filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(servicePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(servicePath, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	instance := scenarioruntime.Instance{InstanceID: "lab-instance", Scenario: "alpha", Variant: "lab", ScopePath: root}
	claims := []scenarioruntime.PortClaim{{InstanceID: instance.InstanceID, Scenario: instance.Scenario, Variant: instance.Variant, PortName: "api", EnvVar: "API_PORT", Port: mustPort(t, portText)}}
	snapshot := (&Service{}).probeHealth(context.Background(), instance, claims)
	if snapshot.Status != scenarioruntime.HealthStatusHealthy || snapshot.Error != "" {
		t.Fatalf("lab profile health = status %q error %q; want healthy without inactive UI_PORT failure", snapshot.Status, snapshot.Error)
	}
}

func mustPort(t *testing.T, value string) int {
	t.Helper()
	var port int
	if _, err := fmt.Sscanf(value, "%d", &port); err != nil {
		t.Fatal(err)
	}
	return port
}
