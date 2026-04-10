package bundleruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-runtime/api"
	"scenario-to-desktop-runtime/gpu"
	"scenario-to-desktop-runtime/health"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/secrets"
)

// mockSupervisorRuntime wraps a Supervisor to implement api.Runtime for testing.
type mockSupervisorRuntime struct {
	manifest      *manifest.Manifest
	secretStore   *secrets.Manager
	statuses      map[string]health.Status
	telemetryLogs []string
	gpuStatus     gpu.Status
}

func (m *mockSupervisorRuntime) Shutdown(_ context.Context) error { return nil }
func (m *mockSupervisorRuntime) ServiceStatuses() map[string]health.Status {
	return m.statuses
}

func (m *mockSupervisorRuntime) PortMap() map[string]map[string]int {
	return map[string]map[string]int{}
}
func (m *mockSupervisorRuntime) TelemetryPath() string        { return "/tmp/telemetry.jsonl" }
func (m *mockSupervisorRuntime) TelemetryUploadURL() string   { return "" }
func (m *mockSupervisorRuntime) Manifest() *manifest.Manifest { return m.manifest }
func (m *mockSupervisorRuntime) AppDataDir() string           { return "/tmp/appdata" }
func (m *mockSupervisorRuntime) FileSystem() FileSystem       { return RealFileSystem{} }
func (m *mockSupervisorRuntime) SecretStore() api.SecretStore { return m.secretStore }
func (m *mockSupervisorRuntime) StartServicesIfReady()        {}
func (m *mockSupervisorRuntime) RecordTelemetry(event string, _ map[string]interface{}) error {
	m.telemetryLogs = append(m.telemetryLogs, event)
	return nil
}

func (m *mockSupervisorRuntime) GPUStatus() gpu.Status {
	if m.gpuStatus.Method == "" {
		return gpu.Status{Available: false, Method: "mock", Reason: "test"}
	}
	return m.gpuStatus
}

func (m *mockSupervisorRuntime) ValidateBundle() *api.BundleValidationResult {
	return &api.BundleValidationResult{Valid: true}
}

func (m *mockSupervisorRuntime) RuntimeInfo() api.RuntimeInfo {
	return api.RuntimeInfo{
		InstanceID:   "test-instance",
		StartedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		AppDataDir:   "/tmp/appdata",
		BundleRoot:   "/tmp/bundle",
		DryRun:       false,
		ManifestHash: "abc123",
	}
}

func ptrBool(v bool) *bool {
	return &v
}

type readyResponse struct {
	Ready   bool                     `json:"ready"`
	Details map[string]health.Status `json:"details"`
}

func waitForReady(t *testing.T, baseURL, token string, timeout time.Duration) readyResponse {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastResp readyResponse
	var lastErr error
	for time.Now().Before(deadline) {
		var resp readyResponse
		if err := fetchJSON(baseURL, token, "/readyz", &resp); err == nil {
			lastResp = resp
			lastErr = nil
			if resp.Ready {
				return resp
			}
		} else {
			lastErr = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	if lastErr != nil {
		t.Fatalf("timeout waiting for ready (last error: %v, last response: %+v)", lastErr, lastResp)
	}
	t.Fatalf("timeout waiting for ready (last response: %+v)", lastResp)
	return readyResponse{}
}

func waitForShutdown(t *testing.T, baseURL string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 300 * time.Millisecond}
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/healthz", nil)
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		resp.Body.Close()
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for shutdown")
}

func fetchJSON(baseURL, token, path string, out interface{}) error {
	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func fetchText(baseURL, token, path string) (string, error) {
	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return "", err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func readAuthToken(t *testing.T, appData string, m *manifest.Manifest) string {
	t.Helper()
	tokenPath := manifest.ResolvePath(appData, m.IPC.AuthTokenRel)
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("read auth token: %v", err)
	}
	return strings.TrimSpace(string(data))
}

func readIPCPort(t *testing.T, appData string) int {
	t.Helper()
	portPath := filepath.Join(appData, "runtime", "ipc_port")
	data, err := os.ReadFile(portPath)
	if err != nil {
		t.Fatalf("read ipc_port: %v", err)
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("parse ipc_port: %v", err)
	}
	return port
}

func integrationManifest(binRel string) *manifest.Manifest {
	key := manifest.PlatformKey(runtime.GOOS, runtime.GOARCH)
	return &manifest.Manifest{
		SchemaVersion: "0.1",
		Target:        "desktop",
		App:           manifest.App{Name: "fixture-app", Version: "0.1.0"},
		IPC:           manifest.IPC{Host: "127.0.0.1", Port: 0, AuthTokenRel: "runtime/auth-token"},
		Telemetry:     manifest.Telemetry{File: "runtime/telemetry.jsonl"},
		Ports:         &manifest.PortRules{DefaultRange: &manifest.PortRange{Min: 48000, Max: 48100}},
		Services: []manifest.Service{
			{
				ID:   "api",
				Type: "backend",
				Binaries: map[string]manifest.Binary{
					key: {
						Path: binRel,
						Args: []string{"--mode", "api", "--port", "${api.http}"},
					},
				},
				Ports: &manifest.ServicePorts{
					Requested: []manifest.PortRequest{
						{Name: "http", Range: manifest.PortRange{Min: 48000, Max: 48100}},
					},
				},
				Health:    manifest.HealthCheck{Type: "http", Path: "/health", PortName: "http", IntervalMs: 200, TimeoutMs: 1000, Retries: 5},
				Readiness: manifest.ReadinessCheck{Type: "health_success", TimeoutMs: 5000},
				LogDir:    "logs/api.log",
			},
			{
				ID:   "worker",
				Type: "worker",
				Binaries: map[string]manifest.Binary{
					key: {
						Path: binRel,
						Args: []string{"--mode", "worker"},
					},
				},
				Health:       manifest.HealthCheck{Type: "log_match", Path: "READY", IntervalMs: 200, TimeoutMs: 1000, Retries: 5},
				Readiness:    manifest.ReadinessCheck{Type: "log_match", Pattern: "READY", TimeoutMs: 5000},
				LogDir:       "logs/worker.log",
				Dependencies: []string{"api"},
			},
		},
	}
}

func buildFixtureService(t *testing.T, bundleDir string) string {
	t.Helper()
	binDir := filepath.Join(bundleDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	srcPath := filepath.Join(binDir, "fixture-service.go")
	source := `package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	mode := flag.String("mode", "api", "mode")
	port := flag.Int("port", 0, "port")
	flag.Parse()

	switch *mode {
	case "api":
		if *port == 0 {
			fmt.Fprintln(os.Stderr, "missing --port")
			os.Exit(2)
		}
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{\"status\":\"healthy\"}"))
		})
		addr := fmt.Sprintf("127.0.0.1:%d", *port)
		log.Printf("READY api on %s", addr)
		server := &http.Server{Addr: addr, Handler: mux}
		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}()
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	case "worker":
		log.Printf("READY worker")
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		if *port > 0 {
			ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
			if err != nil {
				log.Fatal(err)
			}
			go func() {
				for {
					conn, err := ln.Accept()
					if err != nil {
						return
					}
					conn.Close()
				}
			}()
			defer ln.Close()
		}
		<-stop
	default:
		fmt.Fprintf(os.Stderr, "unknown mode: %s\n", *mode)
		os.Exit(2)
	}
}
`
	if err := os.WriteFile(srcPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write fixture source: %v", err)
	}

	exeName := "fixture-service"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	outPath := filepath.Join(binDir, exeName)
	cmd := exec.Command("go", "build", "-o", outPath, srcPath)
	cmd.Dir = binDir
	cmd.Env = append(os.Environ(), "GO111MODULE=off")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fixture service: %v\n%s", err, strings.TrimSpace(string(output)))
	}

	return filepath.Join("bin", exeName)
}
