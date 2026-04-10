package bundleruntime

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSupervisorIntegration_LaunchesServicesAndControlAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	bundleDir := t.TempDir()
	appData := t.TempDir()
	binRel := buildFixtureService(t, bundleDir)

	m := integrationManifest(binRel)
	s, err := NewSupervisor(Options{
		Manifest:   m,
		BundlePath: bundleDir,
		AppDataDir: appData,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Shutdown(context.Background()) })

	token := readAuthToken(t, appData, m)
	port := readIPCPort(t, appData)
	baseURL := "http://" + m.IPC.Host + ":" + strconv.Itoa(port)

	ready := waitForReady(t, baseURL, token, 10*time.Second)
	if !ready.Ready {
		t.Fatalf("expected ready=true, got false")
	}
	if st, ok := ready.Details["api"]; !ok || !st.Ready {
		t.Fatalf("expected api ready status, got %+v", ready.Details["api"])
	}
	if st, ok := ready.Details["worker"]; !ok || !st.Ready {
		t.Fatalf("expected worker ready status, got %+v", ready.Details["worker"])
	}

	var portsResp struct {
		Services map[string]map[string]int `json:"services"`
	}
	if err := fetchJSON(baseURL, token, "/ports", &portsResp); err != nil {
		t.Fatalf("fetch /ports: %v", err)
	}
	if portsResp.Services["api"]["http"] == 0 {
		t.Fatalf("expected allocated api.http port, got %+v", portsResp.Services["api"])
	}

	logs, err := fetchText(baseURL, token, "/logs/tail?serviceId=worker&lines=50")
	if err != nil {
		t.Fatalf("fetch logs: %v", err)
	}
	if !strings.Contains(logs, "READY") {
		t.Fatalf("expected worker logs to contain READY, got %q", logs)
	}

	telemetryPath := filepath.Join(appData, "runtime", "telemetry.jsonl")
	if data, err := os.ReadFile(telemetryPath); err != nil {
		t.Fatalf("read telemetry: %v", err)
	} else if !bytes.Contains(data, []byte(`"event":"service_ready"`)) {
		t.Errorf("expected telemetry to include service_ready events")
	}

	if err := fetchJSON(baseURL, token, "/shutdown", &map[string]string{}); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	waitForShutdown(t, baseURL, 10*time.Second)
}
