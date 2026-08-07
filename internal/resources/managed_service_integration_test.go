package resources

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// These tests are opt-in because they launch the signed native artifacts and
// require the local artifact staging directory. They intentionally use
// ephemeral ports, so they can validate startup/readiness without competing
// with operator-managed resource instances on the canonical ports.
func TestMinioManagedArtifactReadinessIntegration(t *testing.T) {
	if os.Getenv("VROOLI_RESOURCE_INTEGRATION") != "1" {
		t.Skip("set VROOLI_RESOURCE_INTEGRATION=1 to run native managed-service integration tests")
	}
	artifact := stagedArtifact(t, "minio", "minio")
	data := t.TempDir()
	apiPort := freePort(t)
	consolePort := freePort(t)
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/minio/health/ready", apiPort)
	if probeHTTP(endpoint) {
		t.Fatal("MinIO readiness endpoint unexpectedly served before process start")
	}
	cmd := exec.Command(artifact, "server", "--address", fmt.Sprintf("127.0.0.1:%d", apiPort), data, "--console-address", fmt.Sprintf("127.0.0.1:%d", consolePort))
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	cmd.Env = append(os.Environ(), "MINIO_ROOT_USER=minioadmin", "MINIO_ROOT_PASSWORD=minioadmin", "MINIO_BROWSER=off")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start MinIO artifact: %v", err)
	}
	defer stopIntegrationProcess(t, cmd, &output)
	if !waitForHTTP(endpoint, 30*time.Second) {
		t.Fatalf("MinIO readiness never succeeded; output: %s", output.String())
	}
}

func TestQdrantManagedArtifactReadinessIntegration(t *testing.T) {
	if os.Getenv("VROOLI_RESOURCE_INTEGRATION") != "1" {
		t.Skip("set VROOLI_RESOURCE_INTEGRATION=1 to run native managed-service integration tests")
	}
	artifact := stagedArtifact(t, "qdrant", "qdrant")
	data := t.TempDir()
	config := filepath.Join(t.TempDir(), "config.yaml")
	httpPort := freePort(t)
	grpcPort := freePort(t)
	contents := fmt.Sprintf("storage:\n  storage_path: %s\nservice:\n  host: 127.0.0.1\n  http_port: %d\n  grpc_port: %d\n", data, httpPort, grpcPort)
	if err := os.WriteFile(config, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/readyz", httpPort)
	if probeHTTP(endpoint) {
		t.Fatal("Qdrant readiness endpoint unexpectedly served before process start")
	}
	cmd := exec.Command(artifact, "--config-path", config)
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	cmd.Env = append(os.Environ(), "QDRANT__SERVICE__HOST=127.0.0.1", fmt.Sprintf("QDRANT__SERVICE__HTTP_PORT=%d", httpPort), fmt.Sprintf("QDRANT__SERVICE__GRPC_PORT=%d", grpcPort), "QDRANT__TELEMETRY_DISABLED=true")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start Qdrant artifact: %v", err)
	}
	defer stopIntegrationProcess(t, cmd, &output)
	if !waitForHTTP(endpoint, 30*time.Second) {
		t.Fatalf("Qdrant readiness never succeeded; output: %s", output.String())
	}
}

// TestManagedArtifactPrivateLifecycleIntegration exercises the same native
// supervisor and manifest path used by the resource control plane. It selects
// managed-private explicitly so a headless CI host does not need an unlocked
// desktop keyring; the control-plane default remains managed-shared for normal
// operator runs.
func TestManagedArtifactPrivateLifecycleIntegration(t *testing.T) {
	if os.Getenv("VROOLI_RESOURCE_INTEGRATION") != "1" {
		t.Skip("set VROOLI_RESOURCE_INTEGRATION=1 to run native managed-service integration tests")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, resource := range []struct {
		name   string
		binary string
	}{
		{name: "minio", binary: "minio"},
		{name: "qdrant", binary: "qdrant"},
	} {
		t.Run(resource.name, func(t *testing.T) {
			artifactPath := stagedArtifact(t, resource.name, resource.binary)
			controller := NewController(repoRoot, t.TempDir())
			t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", t.TempDir())
			manifest, err := controller.LoadManifest(filepath.Join(repoRoot, "resources", resource.name, "resource.json"))
			if err != nil {
				t.Fatal(err)
			}
			portMap := make(map[int]int, len(manifest.Ports))
			for index := range manifest.Ports {
				oldPort := manifest.Ports[index].Host
				newPort := freePort(t)
				manifest.Ports[index].Host = newPort
				portMap[oldPort] = newPort
			}
			for index := range manifest.HealthChecks {
				target := manifest.HealthChecks[index].Target
				for oldPort, newPort := range portMap {
					target = strings.ReplaceAll(target, ":"+strconv.Itoa(oldPort), ":"+strconv.Itoa(newPort))
				}
				manifest.HealthChecks[index].Target = target
			}

			driver := managedServiceDriver{}
			item := Resource{Name: resource.name}
			private := []string{"--provider=managed-private"}
			if err := driver.Run(context.Background(), controller, item, manifest, "start", private, io.Discard, io.Discard); err != nil {
				t.Fatalf("start verified private artifact %s: %v", artifactPath, err)
			}
			t.Cleanup(func() {
				_ = driver.Run(context.Background(), controller, item, manifest, "stop", private, io.Discard, io.Discard)
			})

			status, err := driver.Status(context.Background(), controller, item, manifest, false)
			if err != nil || !status.Installed || !status.Running || status.Health != "healthy" {
				t.Fatalf("private %s status = %+v, %v", resource.name, status, err)
			}
			var logs bytes.Buffer
			if err := driver.Run(context.Background(), controller, item, manifest, "logs", private, &logs, io.Discard); err != nil {
				t.Fatalf("logs %s: %v", resource.name, err)
			}
			if err := driver.Run(context.Background(), controller, item, manifest, "restart", private, io.Discard, io.Discard); err != nil {
				t.Fatalf("restart %s: %v", resource.name, err)
			}
			if err := driver.Run(context.Background(), controller, item, manifest, "stop", private, io.Discard, io.Discard); err != nil {
				t.Fatalf("stop %s: %v", resource.name, err)
			}
			status, err = driver.Status(context.Background(), controller, item, manifest, true)
			if err != nil || status.Running {
				t.Fatalf("stopped %s status = %+v, %v", resource.name, status, err)
			}
		})
	}
}

func stagedArtifact(t *testing.T, resource, binary string) string {
	t.Helper()
	root := os.Getenv("VROOLI_RESOURCE_ARTIFACT_DIR")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		root = filepath.Join(home, ".vrooli", "artifacts")
	}
	name := fmt.Sprintf("%s_%s_%s", binary, runtime.GOOS, runtime.GOARCH)
	path := filepath.Join(root, resource, map[string]string{"minio": "RELEASE.2025-07-23T15-54-02Z", "qdrant": "1.15.3"}[resource], name)
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("staged %s artifact unavailable: %v", resource, err)
	}
	return path
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func probeHTTP(endpoint string) bool {
	client := &http.Client{Timeout: 250 * time.Millisecond}
	response, err := client.Get(endpoint)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	return response.StatusCode >= 200 && response.StatusCode < 300
}

func waitForHTTP(endpoint string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if probeHTTP(endpoint) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func stopIntegrationProcess(t *testing.T, cmd *exec.Cmd, output *bytes.Buffer) {
	t.Helper()
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if err := cmd.Wait(); err != nil && !isProcessKilled(err) {
		t.Logf("managed artifact exit: %v; output: %s", err, output.String())
	}
}

func isProcessKilled(err error) bool {
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ProcessState != nil && !exit.ProcessState.Success()
	}
	return false
}
