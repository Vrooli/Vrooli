package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	hostreqspec "github.com/vrooli/vrooli/internal/hostreqspec"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=5 | LAST: 2026-04-13

func TestStatusForResourceReportsUnavailableCommand(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	controller := NewController(root, home)

	item := Resource{
		Name: "missing",
		Path: filepath.Join(root, "resources", "missing"),
	}

	status, err := controller.statusForResource(item, true)
	if err != nil {
		t.Fatalf("statusForResource: %v", err)
	}
	if status.StatusCode != StatusCodeUnavailable {
		t.Fatalf("status.StatusCode = %q, want %q", status.StatusCode, StatusCodeUnavailable)
	}
	if status.ProbeError == "" {
		t.Fatal("expected probe error for unavailable status command")
	}
}

func TestStatusForResourceCategorizesProbeFailures(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	controller := NewController(root, home)
	writeExecutableOnPath(t, "resource-fixture", "#!/usr/bin/env bash\nexit 0\n")

	originalRun := runCommandResource
	t.Cleanup(func() {
		runCommandResource = originalRun
	})

	item := Resource{
		Name:   "fixture",
		Path:   filepath.Join(root, "resources", "fixture"),
		Exists: true,
		HasCLI: true,
	}

	t.Run("timeout", func(t *testing.T) {
		runCommandResource = func(ctx context.Context, cmd *exec.Cmd) commandResult {
			return commandResult{err: context.DeadlineExceeded}
		}

		status, err := controller.statusForResource(item, true)
		if err != nil {
			t.Fatalf("statusForResource: %v", err)
		}
		if status.StatusCode != StatusCodeTimeout {
			t.Fatalf("status.StatusCode = %q, want %q", status.StatusCode, StatusCodeTimeout)
		}
	})

	t.Run("command error", func(t *testing.T) {
		runCommandResource = func(ctx context.Context, cmd *exec.Cmd) commandResult {
			return commandResult{err: errors.New("exit status 7")}
		}

		status, err := controller.statusForResource(item, true)
		if err != nil {
			t.Fatalf("statusForResource: %v", err)
		}
		if status.StatusCode != StatusCodeCommandError {
			t.Fatalf("status.StatusCode = %q, want %q", status.StatusCode, StatusCodeCommandError)
		}
		if !strings.Contains(status.ProbeError, "exit status 7") {
			t.Fatalf("status.ProbeError = %q", status.ProbeError)
		}
	})

	t.Run("invalid payload", func(t *testing.T) {
		runCommandResource = func(ctx context.Context, cmd *exec.Cmd) commandResult {
			return commandResult{output: []byte("not-json\n")}
		}

		status, err := controller.statusForResource(item, true)
		if err != nil {
			t.Fatalf("statusForResource: %v", err)
		}
		if status.StatusCode != StatusCodeInvalidStatusPayload {
			t.Fatalf("status.StatusCode = %q, want %q", status.StatusCode, StatusCodeInvalidStatusPayload)
		}
	})
}

func TestNormalizeComposePSOutputIgnoresWarnings(t *testing.T) {
	output := []byte(`time="2026-04-12T00:04:48-04:00" level=warning msg="compose warning"
{"Service":"fixture","State":"running","Health":"healthy"}`)

	normalized := normalizeComposePSOutput(output)
	if normalized != `{"Service":"fixture","State":"running","Health":"healthy"}` {
		t.Fatalf("normalizeComposePSOutput() = %q", normalized)
	}
}

func TestInspectDockerContainerUsesContainerInspectBoundary(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	controller := NewController(root, home)
	manifest := manifestpkg.ResourceManifest{
		Name: "ollama",
		Runtime: manifestpkg.ResourceRuntime{
			ContainerName: "ollama",
		},
	}

	originalRun := runCommandResource
	t.Cleanup(func() {
		runCommandResource = originalRun
	})

	runCommandResource = func(ctx context.Context, cmd *exec.Cmd) commandResult {
		if got := strings.Join(cmd.Args[:3], " "); got != "docker container inspect" {
			t.Fatalf("inspect command = %q, want docker container inspect", got)
		}
		return commandResult{
			output: []byte("Error: No such container: ollama"),
			err:    errors.New("exit status 1"),
		}
	}

	_, exists, err := inspectDockerContainer(context.Background(), controller, manifest)
	if err != nil {
		t.Fatalf("inspectDockerContainer: %v", err)
	}
	if exists {
		t.Fatal("expected missing container to report exists=false")
	}
}

func TestStatusForResourceParsesStructuredPayload(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	controller := NewController(root, home)
	writeExecutableOnPath(t, "resource-fixture", "#!/usr/bin/env bash\nexit 0\n")

	originalRun := runCommandResource
	t.Cleanup(func() {
		runCommandResource = originalRun
	})
	runCommandResource = func(ctx context.Context, cmd *exec.Cmd) commandResult {
		return commandResult{output: []byte(`{"installed":true,"running":true,"healthy":true,"message":"healthy"}`)}
	}

	item := Resource{
		Name:   "fixture",
		Path:   filepath.Join(root, "resources", "fixture"),
		Exists: true,
		HasCLI: true,
	}

	status, err := controller.statusForResource(item, true)
	if err != nil {
		t.Fatalf("statusForResource: %v", err)
	}
	if status.StatusCode != StatusCodeOK {
		t.Fatalf("status.StatusCode = %q, want %q", status.StatusCode, StatusCodeOK)
	}
	if !status.Running {
		t.Fatal("expected running status")
	}
	if status.Healthy == nil || !*status.Healthy {
		t.Fatalf("status.Healthy = %#v", status.Healthy)
	}
}

func TestRunReturnsCategorizedErrors(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	controller := NewController(root, home)

	err := controller.Run("missing", []string{"start"}, ioDiscard{}, ioDiscard{})
	var resourceErr *Error
	if !errors.As(err, &resourceErr) {
		t.Fatalf("expected *Error, got %T", err)
	}
	if resourceErr.Code != ErrorCodeCommandUnavailable {
		t.Fatalf("resourceErr.Code = %q, want %q", resourceErr.Code, ErrorCodeCommandUnavailable)
	}

	writeExecutableOnPath(t, "resource-fixture", "#!/usr/bin/env bash\nexit 7\n")

	err = controller.Run("fixture", []string{"stop"}, ioDiscard{}, ioDiscard{})
	if !errors.As(err, &resourceErr) {
		t.Fatalf("expected *Error, got %T", err)
	}
	if resourceErr.Code != ErrorCodeOperationFailed {
		t.Fatalf("resourceErr.Code = %q, want %q", resourceErr.Code, ErrorCodeOperationFailed)
	}
	if resourceErr.Operation != "stop" {
		t.Fatalf("resourceErr.Operation = %q, want stop", resourceErr.Operation)
	}
}

func TestDiscoverMarksManifestNativeResources(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("docker-service"),
		testresource.WithResourceTemplate("docker-service"),
		testresource.WithResourceDescription("Fixture resource"),
		testresource.WithResourceRuntime(manifestpkg.ResourceRuntime{
			Image:         "fixture:latest",
			ContainerName: "vrooli-fixture",
		}),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "partial",
		}),
	))

	items, err := NewController(root, home).Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ControlMode != "manifest-native" {
		t.Fatalf("ControlMode = %q, want manifest-native", items[0].ControlMode)
	}
	if items[0].Driver != "docker-service" {
		t.Fatalf("Driver = %q, want docker-service", items[0].Driver)
	}
}

func TestDiscoverHidesLegacyShellDirectoriesWithoutManifest(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceCLI(t, root, "fixture", "#!/usr/bin/env bash\nexit 0\n")

	items, err := NewController(root, home).Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0: %#v", len(items), items)
	}
}

func TestDiscoverHidesConfigOnlyResourcesWithoutManifest(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)

	items, err := NewController(root, home).Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0: %#v", len(items), items)
	}
}

func TestStatusForManifestNativeDockerResource(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("docker-service"),
		testresource.WithResourceTemplate("docker-service"),
		testresource.WithResourceDescription("Fixture resource"),
		testresource.WithResourceRuntime(manifestpkg.ResourceRuntime{
			Image:         "fixture:latest",
			ContainerName: "vrooli-fixture",
		}),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "partial",
		}),
	))
	stateFile := writeFakeDocker(t)
	if err := os.WriteFile(stateFile, []byte("running\n"), 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	status, err := NewController(root, home).Status("fixture", true)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.Running {
		t.Fatal("expected running resource")
	}
	if status.Healthy == nil || !*status.Healthy {
		t.Fatalf("Healthy = %#v, want true", status.Healthy)
	}
	if status.Resource.ControlMode != "manifest-native" {
		t.Fatalf("ControlMode = %q", status.Resource.ControlMode)
	}
}

func TestRunManifestNativeDockerLifecycle(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("docker-service"),
		testresource.WithResourceTemplate("docker-service"),
		testresource.WithResourceDescription("Fixture resource"),
		testresource.WithResourceRuntime(manifestpkg.ResourceRuntime{
			Image:         "fixture:latest",
			ContainerName: "vrooli-fixture",
		}),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "partial",
		}),
	))
	stateFile := writeFakeDocker(t)

	controller := NewController(root, home)
	if err := controller.Run("fixture", []string{"start"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatalf("Run(start): %v", err)
	}
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if strings.TrimSpace(string(data)) != "running" {
		t.Fatalf("state after start = %q, want running", string(data))
	}

	var stdout bytes.Buffer
	if err := controller.Run("fixture", []string{"logs"}, &stdout, ioDiscard{}); err != nil {
		t.Fatalf("Run(logs): %v", err)
	}
	if !strings.Contains(stdout.String(), "fixture logs") {
		t.Fatalf("logs output = %q", stdout.String())
	}

	if err := controller.Run("fixture", []string{"stop"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatalf("Run(stop): %v", err)
	}
	data, err = os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if strings.TrimSpace(string(data)) != "stopped" {
		t.Fatalf("state after stop = %q, want stopped", string(data))
	}
}

func TestStatusForManifestNativeDockerResourceAcceptsExternalHealthyService(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	port := mustAllocatePort(t)
	server := startHTTPServer(t, "127.0.0.1:"+strconv.Itoa(port), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer server.Shutdown(context.Background())

	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("docker-service"),
		testresource.WithResourceTemplate("docker-service"),
		testresource.WithResourceDescription("Fixture resource"),
		testresource.WithResourceRuntime(manifestpkg.ResourceRuntime{
			Image:         "fixture:latest",
			ContainerName: "vrooli-fixture",
		}),
		testresource.WithResourceHealthChecks(manifestpkg.ResourceHealthCheck{
			Type:           "http",
			Target:         "http://127.0.0.1:" + strconv.Itoa(port) + "/health",
			ExpectedStatus: []int{200},
			TimeoutSeconds: 5,
		}),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "partial",
		}),
	))
	writeFakeDocker(t)

	status, err := NewController(root, home).Status("fixture", false)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.Running {
		t.Fatal("expected external healthy service to satisfy running status")
	}
	if status.Healthy == nil || !*status.Healthy {
		t.Fatalf("Healthy = %#v, want true", status.Healthy)
	}
	if status.Message != "healthy (external)" {
		t.Fatalf("Message = %q, want healthy (external)", status.Message)
	}
}

func TestRunManifestNativeDockerStartNoopsWhenExternalServiceIsHealthy(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	port := mustAllocatePort(t)
	server := startHTTPServer(t, "127.0.0.1:"+strconv.Itoa(port), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer server.Shutdown(context.Background())

	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("docker-service"),
		testresource.WithResourceTemplate("docker-service"),
		testresource.WithResourceDescription("Fixture resource"),
		testresource.WithResourceRuntime(manifestpkg.ResourceRuntime{
			Image:         "fixture:latest",
			ContainerName: "vrooli-fixture",
		}),
		testresource.WithResourceHealthChecks(manifestpkg.ResourceHealthCheck{
			Type:           "http",
			Target:         "http://127.0.0.1:" + strconv.Itoa(port) + "/health",
			ExpectedStatus: []int{200},
			TimeoutSeconds: 5,
		}),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "partial",
		}),
	))
	stateFile := writeFakeDocker(t)

	controller := NewController(root, home)
	if err := controller.Run("fixture", []string{"start"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatalf("Run(start): %v", err)
	}
	if _, err := os.Stat(stateFile); !os.IsNotExist(err) {
		t.Fatalf("expected no managed container state file, got err=%v", err)
	}
}

func TestRunManifestNativeDockerStartPrefersExternalHealthyServiceOverStoppedContainer(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	port := mustAllocatePort(t)
	server := startHTTPServer(t, "127.0.0.1:"+strconv.Itoa(port), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer server.Shutdown(context.Background())

	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("docker-service"),
		testresource.WithResourceTemplate("docker-service"),
		testresource.WithResourceDescription("Fixture resource"),
		testresource.WithResourceRuntime(manifestpkg.ResourceRuntime{
			Image:         "fixture:latest",
			ContainerName: "vrooli-fixture",
		}),
		testresource.WithResourceHealthChecks(manifestpkg.ResourceHealthCheck{
			Type:           "http",
			Target:         "http://127.0.0.1:" + strconv.Itoa(port) + "/health",
			ExpectedStatus: []int{200},
			TimeoutSeconds: 5,
		}),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "partial",
		}),
	))
	stateFile := writeFakeDocker(t)
	if err := os.WriteFile(stateFile, []byte("stopped\n"), 0o644); err != nil {
		t.Fatalf("write fake docker state: %v", err)
	}

	controller := NewController(root, home)
	if err := controller.Run("fixture", []string{"start"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatalf("Run(start): %v", err)
	}
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if strings.TrimSpace(string(data)) != "stopped" {
		t.Fatalf("state after external-preferred start = %q, want stopped", string(data))
	}
}

func TestStatusForManifestNativeComposeResource(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("compose-service"),
		testresource.WithResourceTemplate("compose-service"),
		testresource.WithResourceDescription("Fixture compose resource"),
		testresource.WithResourceComposeFile("compose.yaml"),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "partial",
		}),
	))
	testkitgo.WriteRelativeFile(t, root, filepath.Join("resources", "fixture", "compose.yaml"), "services:\n  app:\n    image: fixture:latest\n")
	stateFile := writeFakeDocker(t)
	if err := os.WriteFile(stateFile, []byte("running\n"), 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	status, err := NewController(root, home).Status("fixture", true)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.Installed || !status.Running {
		t.Fatalf("status = %#v, expected installed/running", status)
	}
	if status.Healthy == nil || !*status.Healthy {
		t.Fatalf("Healthy = %#v, want true", status.Healthy)
	}
}

func TestRunManifestNativeComposeLifecycle(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("compose-service"),
		testresource.WithResourceTemplate("compose-service"),
		testresource.WithResourceDescription("Fixture compose resource"),
		testresource.WithResourceComposeFile("compose.yaml"),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "partial",
		}),
	))
	testkitgo.WriteRelativeFile(t, root, filepath.Join("resources", "fixture", "compose.yaml"), "services:\n  app:\n    image: fixture:latest\n")
	stateFile := writeFakeDocker(t)

	controller := NewController(root, home)
	if err := controller.Run("fixture", []string{"start"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatalf("Run(start): %v", err)
	}
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if strings.TrimSpace(string(data)) != "running" {
		t.Fatalf("state after start = %q, want running", string(data))
	}

	var stdout bytes.Buffer
	if err := controller.Run("fixture", []string{"logs"}, &stdout, ioDiscard{}); err != nil {
		t.Fatalf("Run(logs): %v", err)
	}
	if !strings.Contains(stdout.String(), "fixture logs") {
		t.Fatalf("logs output = %q", stdout.String())
	}

	if err := controller.Run("fixture", []string{"stop"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatalf("Run(stop): %v", err)
	}
	data, err = os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if strings.TrimSpace(string(data)) != "stopped" {
		t.Fatalf("state after stop = %q, want stopped", string(data))
	}
}

func TestStatusForManifestNativeUnsupportedPlatform(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("docker-service"),
		testresource.WithResourceTemplate("docker-service"),
		testresource.WithResourceDescription("Fixture resource"),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "unsupported",
			MacOS:   "unsupported",
			Windows: "unsupported",
		}),
		testresource.WithResourceRuntime(manifestpkg.ResourceRuntime{
			Image: "fixture:latest",
		}),
	))

	status, err := NewController(root, home).Status("fixture", true)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.StatusCode != StatusCodeUnsupportedPlatform {
		t.Fatalf("StatusCode = %q, want %q", status.StatusCode, StatusCodeUnsupportedPlatform)
	}
}

func TestStatusForManifestNativeExternalCLIResource(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	commandPath := writeExecutableOnPath(t, "fixture-cli", "#!/usr/bin/env bash\nif [[ \"$1\" == \"--version\" ]]; then\n  echo 'fixture-cli 1.0.0'\n  exit 0\nfi\nexit 0\n")
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceTemplate("external-cli"),
		testresource.WithResourceDescription("Fixture external CLI resource"),
		testresource.WithResourceBinary("fixture-cli"),
		testresource.WithResourceVersionArgs("--version"),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "supported",
		}),
		testresource.WithResourceHealthChecks(manifestpkg.ResourceHealthCheck{
			Type:    "command",
			Command: []string{commandPath, "--version"},
		}),
	))

	status, err := NewController(root, home).Status("fixture", true)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.Installed || !status.Running {
		t.Fatalf("status = %#v, expected installed/running", status)
	}
	if status.Healthy == nil || !*status.Healthy {
		t.Fatalf("Healthy = %#v, want true", status.Healthy)
	}
	if status.Message != "available" {
		t.Fatalf("Message = %q, want available", status.Message)
	}
}

func TestStatusForManifestNativeExternalCLIResourceUsesResourceScopedEnvForHealthChecks(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	scriptPath := filepath.Join(root, "fixture-health.sh")
	expectedDataDir := ""
	for _, entry := range resourceEnvForResource(root, home, "fixture") {
		if strings.HasPrefix(entry, "RESOURCE_DATA_DIR=") {
			expectedDataDir = strings.TrimPrefix(entry, "RESOURCE_DATA_DIR=")
			break
		}
	}
	if expectedDataDir == "" {
		t.Fatal("expected RESOURCE_DATA_DIR in resource env")
	}
	script := fmt.Sprintf("#!/usr/bin/env bash\nset -euo pipefail\n[[ \"$RESOURCE_DATA_DIR\" == %q ]]\n", expectedDataDir)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write health script: %v", err)
	}
	writeExecutableOnPath(t, "fixture-cli", "#!/usr/bin/env bash\nexit 0\n")
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceTemplate("external-cli"),
		testresource.WithResourceDescription("Fixture external CLI resource"),
		testresource.WithResourceBinary("fixture-cli"),
		testresource.WithResourceVersionArgs("--version"),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "supported",
		}),
		testresource.WithResourceHealthChecks(manifestpkg.ResourceHealthCheck{
			Type:    "command",
			Command: []string{"bash", scriptPath},
		}),
	))

	status, err := NewController(root, home).Status("fixture", false)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Healthy == nil || !*status.Healthy {
		t.Fatalf("status = %#v, want healthy", status)
	}
}

func TestStatusForManifestNativeExternalCLIResourceMarksUnavailableWhenVersionProbeFails(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	writeExecutableOnPath(t, "fixture-cli", "#!/usr/bin/env bash\nif [[ \"$1\" == \"--version\" ]]; then\n  exit 1\nfi\nexit 0\n")
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceTemplate("external-cli"),
		testresource.WithResourceDescription("Fixture external CLI resource"),
		testresource.WithResourceBinary("fixture-cli"),
		testresource.WithResourceVersionArgs("--version"),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "supported",
		}),
	))

	status, err := NewController(root, home).Status("fixture", true)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.StatusCode != StatusCodeUnavailable {
		t.Fatalf("StatusCode = %q, want %q", status.StatusCode, StatusCodeUnavailable)
	}
	if status.Healthy == nil || *status.Healthy {
		t.Fatalf("Healthy = %#v, want false", status.Healthy)
	}
}

func TestRunManifestNativeExternalCLIInstallRejectsUnsupportedAction(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	installMarker := filepath.Join(root, "install.marker")
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceTemplate("external-cli"),
		testresource.WithResourceDescription("Fixture external CLI resource"),
		testresource.WithResourceBinary("fixture-cli"),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "supported",
		}),
		testresource.WithResourceInstall(manifestpkg.ResourceInstall{
			Platforms: map[string][]string{
				"linux":   {"sh", "-c", "printf installed > " + shellQuote(installMarker)},
				"macos":   {"sh", "-c", "printf installed > " + shellQuote(installMarker)},
				"windows": {"sh", "-c", "printf installed > " + shellQuote(installMarker)},
			},
		}),
	))
	writeExecutableOnPath(t, "fixture-cli", "#!/usr/bin/env bash\necho 'fixture-cli 1.0.0'\n")

	controller := NewController(root, home)
	if err := controller.Run("fixture", []string{"install"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatalf("Run(install): %v", err)
	}
	if _, err := os.Stat(installMarker); err != nil {
		t.Fatalf("expected install marker: %v", err)
	}
	if err := controller.Run("fixture", []string{"custom"}, ioDiscard{}, ioDiscard{}); err == nil || !strings.Contains(err.Error(), `action "custom" is not supported`) {
		t.Fatalf("Run(custom) error = %v", err)
	}
}

func TestRunManifestNativeExternalCLIStartInstallsWhenUnavailable(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	installMarker := filepath.Join(root, "install.marker")
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceTemplate("external-cli"),
		testresource.WithResourceDescription("Fixture external CLI resource"),
		testresource.WithResourceBinary("missing-cli"),
		testresource.WithResourceVersionArgs("--version"),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "supported",
		}),
		testresource.WithResourceInstall(manifestpkg.ResourceInstall{
			Platforms: map[string][]string{
				"linux":   {"sh", "-c", "printf installed > " + shellQuote(installMarker)},
				"macos":   {"sh", "-c", "printf installed > " + shellQuote(installMarker)},
				"windows": {"sh", "-c", "printf installed > " + shellQuote(installMarker)},
			},
		}),
	))

	if err := NewController(root, home).Run("fixture", []string{"start"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatalf("Run(start): %v", err)
	}
	if _, err := os.Stat(installMarker); err != nil {
		t.Fatalf("expected install marker after start: %v", err)
	}
}

func TestStatusForManifestNativeCloudAPIResource(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	server := startHTTPServer(t, "127.0.0.1:"+strconv.Itoa(mustAllocatePort(t)), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer server.Shutdown(context.Background())

	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("cloud-api"),
		testresource.WithResourceTemplate("cloud-api"),
		testresource.WithResourceDescription("Fixture cloud API resource"),
		testresource.WithResourceEndpoint("http://"+server.Addr+"/health"),
		testresource.WithResourceCredentialsEnv("FIXTURE_API_KEY"),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "supported",
		}),
		testresource.WithResourceHealthChecks(manifestpkg.ResourceHealthCheck{
			Type:           "http",
			Target:         "http://" + server.Addr + "/health",
			ExpectedStatus: []int{http.StatusUnauthorized},
		}),
	))
	t.Setenv("FIXTURE_API_KEY", "secret")

	status, err := NewController(root, home).Status("fixture", false)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.Installed || !status.Running {
		t.Fatalf("status = %#v, expected installed/running", status)
	}
	if status.Healthy == nil || !*status.Healthy {
		t.Fatalf("Healthy = %#v, want true", status.Healthy)
	}
}

func TestStatusForManifestNativeCloudAPIMissingCredentials(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("cloud-api"),
		testresource.WithResourceTemplate("cloud-api"),
		testresource.WithResourceDescription("Fixture cloud API resource"),
		testresource.WithResourceEndpoint("https://api.example.com/health"),
		testresource.WithResourceCredentialsEnv("FIXTURE_API_KEY"),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "supported",
		}),
	))

	status, err := NewController(root, home).Status("fixture", true)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Healthy == nil || *status.Healthy {
		t.Fatalf("Healthy = %#v, want false", status.Healthy)
	}
	if !strings.Contains(status.Message, "FIXTURE_API_KEY") {
		t.Fatalf("Message = %q, want credential hint", status.Message)
	}
}

func TestProjectPhase5ResourcesAreManifestNative(t *testing.T) {
	root := projectRootForResourcesTest(t)
	controller := NewController(root, t.TempDir())

	items, err := controller.Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	expected := map[string]string{
		"postgres":              "docker-service",
		"redis":                 "docker-service",
		"qdrant":                "docker-service",
		"browserless":           "docker-service",
		"vault":                 "docker-service",
		"litellm":               "docker-service",
		"minio":                 "docker-service",
		"neo4j":                 "docker-service",
		"questdb":               "docker-service",
		"searxng":               "docker-service",
		"comfyui":               "docker-service",
		"home-assistant":        "compose-service",
		"kokoro":                "compose-service",
		"mail-in-a-box":         "compose-service",
		"sagemath":              "docker-service",
		"whisper":               "compose-service",
		"claude-code":           "external-cli",
		"codex":                 "external-cli",
		"k6":                    "external-cli",
		"opencode":              "external-cli",
		"sqlite":                "external-cli",
		"ollama":                "docker-service",
		"judge0":                "compose-service",
		"postgis":               "compose-service",
		"unstructured-io":       "docker-service",
		"gemini":                "cloud-api",
		"openrouter":            "cloud-api",
		"twilio":                "cloud-api",
		"cloudflare-ai-gateway": "cloud-api",
	}
	seen := make(map[string]Resource)
	for _, item := range items {
		if _, ok := expected[item.Name]; ok {
			seen[item.Name] = item
		}
	}
	for name, driver := range expected {
		item, ok := seen[name]
		if !ok {
			t.Fatalf("resource %q not discovered", name)
		}
		if item.ControlMode != "manifest-native" {
			t.Fatalf("%s ControlMode = %q, want manifest-native", name, item.ControlMode)
		}
		if item.Driver != driver {
			t.Fatalf("%s Driver = %q, want %q", name, item.Driver, driver)
		}
		if item.ManifestPath == "" {
			t.Fatalf("%s ManifestPath is empty", name)
		}
	}
}

func TestProjectPhase5ResourceManifestsValidate(t *testing.T) {
	root := projectRootForResourcesTest(t)
	controller := NewController(root, t.TempDir())

	for _, name := range []string{"postgres", "redis", "qdrant", "browserless", "vault", "litellm", "minio", "neo4j", "questdb", "searxng", "comfyui", "home-assistant", "kokoro", "mail-in-a-box", "sagemath", "whisper", "claude-code", "codex", "k6", "opencode", "sqlite", "ollama", "judge0", "postgis", "unstructured-io", "gemini", "openrouter", "twilio", "cloudflare-ai-gateway"} {
		manifest, err := controller.loadResourceManifest(defaultResourceManifestPath(root, name))
		if err != nil {
			t.Fatalf("loadResourceManifest(%s): %v", name, err)
		}
		if manifest.Name != name {
			t.Fatalf("%s manifest name = %q", name, manifest.Name)
		}
		if strings.TrimSpace(manifest.Driver) == "" {
			t.Fatalf("%s driver is empty", name)
		}
	}
}

func TestLoadResourceManifestParsesHostRequirements(t *testing.T) {
	root := t.TempDir()
	controller := NewController(root, t.TempDir())
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceBinary("fixture"),
		testresource.WithResourceHostTools(
			hostreqspec.Declaration{Name: "sqlite", Required: true, Reason: "resource sqlite", When: []string{"setup"}},
		),
		testresource.WithResourceHostSafeguards(
			hostreqspec.Declaration{Name: "remote_session_protection", Required: false, Reason: "resource safeguard", Platforms: []string{"linux"}},
		),
	))

	manifest, err := controller.loadResourceManifest(defaultResourceManifestPath(root, "fixture"))
	if err != nil {
		t.Fatalf("loadResourceManifest: %v", err)
	}
	if len(manifest.HostTools) != 1 || manifest.HostTools[0].Name != "sqlite" {
		t.Fatalf("hostTools = %+v", manifest.HostTools)
	}
	if len(manifest.HostSafeguards) != 1 || manifest.HostSafeguards[0].Name != "remote_session_protection" {
		t.Fatalf("hostSafeguards = %+v", manifest.HostSafeguards)
	}
}

func TestLoadResourceManifestRejectsDuplicateHostRequirements(t *testing.T) {
	root := t.TempDir()
	controller := NewController(root, t.TempDir())
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceBinary("fixture"),
		testresource.WithResourceHostTools(
			hostreqspec.Declaration{Name: "sqlite", Required: true, Reason: "one"},
			hostreqspec.Declaration{Name: "sqlite", Required: false, Reason: "two"},
		),
	))

	if _, err := controller.loadResourceManifest(defaultResourceManifestPath(root, "fixture")); err == nil || !strings.Contains(err.Error(), `duplicate tool declaration "sqlite"`) {
		t.Fatalf("loadResourceManifest error = %v", err)
	}
}

func TestProjectPhase6LegacyAdapterBacklogIsCleared(t *testing.T) {
	root := projectRootForResourcesTest(t)
	controller := NewController(root, t.TempDir())

	items, err := controller.Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	for _, item := range items {
		if item.ControlMode == "legacy-adapter" {
			t.Fatalf("%s unexpectedly remains a legacy-adapter resource", item.Name)
		}
	}
}

func TestProjectMigratedResourcesUseNativeDrivers(t *testing.T) {
	root := projectRootForResourcesTest(t)
	controller := NewController(root, t.TempDir())

	expected := map[string]string{
		"kokoro":        "compose-service",
		"mail-in-a-box": "compose-service",
		"sagemath":      "docker-service",
		"whisper":       "compose-service",
	}

	for name, driver := range expected {
		status, err := controller.Status(name, true)
		if err != nil {
			t.Fatalf("Status(%s): %v", name, err)
		}
		if status.Resource.ControlMode != "manifest-native" {
			t.Fatalf("%s ControlMode = %q, want manifest-native", name, status.Resource.ControlMode)
		}
		if status.Resource.Driver != driver {
			t.Fatalf("%s Driver = %q, want %q", name, status.Resource.Driver, driver)
		}
	}
}

func TestProjectPhase7ActiveDiscoveryOnlyIncludesManifestBackedResources(t *testing.T) {
	root := projectRootForResourcesTest(t)
	controller := NewController(root, t.TempDir())

	items, err := controller.Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one active resource")
	}

	for _, item := range items {
		if item.ManifestPath == "" {
			t.Fatalf("%s missing manifest path in active discovery", item.Name)
		}
		if item.ControlMode != "manifest-native" {
			t.Fatalf("%s ControlMode = %q, want manifest-native", item.Name, item.ControlMode)
		}
	}
}

func TestProjectDockerResourceStatusesUseNativeManifests(t *testing.T) {
	projectRoot := projectRootForResourcesTest(t)
	root := t.TempDir()
	home := t.TempDir()
	controller := NewController(root, home)
	stateFile := writeFakeDocker(t)

	testscenario.WriteProjectResourceConfig(t, root, "postgres", true)
	testscenario.WriteProjectResourceConfig(t, root, "redis", true)
	testscenario.WriteProjectResourceConfig(t, root, "qdrant", true)
	testscenario.WriteProjectResourceConfig(t, root, "browserless", true)
	testscenario.WriteProjectResourceConfig(t, root, "vault", true)

	postgresPort := mustAllocatePort(t)
	redisPort := mustAllocatePort(t)
	qdrantPort := mustAllocatePort(t)
	qdrantGRPCPort := mustAllocatePort(t)
	browserlessPort := mustAllocatePort(t)
	vaultPort := mustAllocatePort(t)

	copyManifestWithOverrides(t, projectRoot, root, "postgres", postgresPort, postgresPort, "tcp", "")
	copyManifestWithOverrides(t, projectRoot, root, "redis", redisPort, redisPort, "tcp", "")
	copyManifestWithOverrides(t, projectRoot, root, "qdrant", qdrantPort, qdrantGRPCPort, "http", "/")
	copyManifestWithOverrides(t, projectRoot, root, "browserless", browserlessPort, browserlessPort, "http", "/pressure")
	copyManifestWithOverrides(t, projectRoot, root, "vault", vaultPort, vaultPort, "http", "/v1/sys/health")

	postgresListener := mustListenTCP(t, "127.0.0.1:"+strconv.Itoa(postgresPort))
	defer postgresListener.Close()
	redisListener := mustListenTCP(t, "127.0.0.1:"+strconv.Itoa(redisPort))
	defer redisListener.Close()

	qdrantServer := startHTTPServer(t, "127.0.0.1:"+strconv.Itoa(qdrantPort), func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"title":"qdrant"}`))
	})
	defer qdrantServer.Shutdown(context.Background())

	browserlessServer := startHTTPServer(t, "127.0.0.1:"+strconv.Itoa(browserlessPort), func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pressure" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"pressure":0}`))
	})
	defer browserlessServer.Shutdown(context.Background())

	vaultServer := startHTTPServer(t, "127.0.0.1:"+strconv.Itoa(vaultPort), func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/health" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"initialized":true,"sealed":false}`))
	})
	defer vaultServer.Shutdown(context.Background())

	if err := os.WriteFile(stateFile, []byte("running\n"), 0o644); err != nil {
		t.Fatalf("write fake docker state: %v", err)
	}

	for _, name := range []string{"postgres", "redis", "qdrant", "browserless", "vault"} {
		status, err := controller.Status(name, true)
		if err != nil {
			t.Fatalf("Status(%s): %v", name, err)
		}
		if status.Resource.ControlMode != "manifest-native" {
			t.Fatalf("%s ControlMode = %q, want manifest-native", name, status.Resource.ControlMode)
		}
		if !status.Running {
			t.Fatalf("%s expected running status", name)
		}
		if status.Healthy == nil || !*status.Healthy {
			t.Fatalf("%s Healthy = %#v, want true", name, status.Healthy)
		}
	}
}

func TestManifestNativeDockerStandardCommandsDoNotFallbackToLegacyCLI(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	legacyMarker := filepath.Join(root, "legacy-docker.marker")
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDisplayName("Fixture Docker"),
		testresource.WithResourceDriver("docker-service"),
		testresource.WithResourceTemplate("docker-service"),
		testresource.WithResourceDescription("Fixture docker resource"),
		testresource.WithResourceRuntime(manifestpkg.ResourceRuntime{
			Image:         "fixture:latest",
			ContainerName: "vrooli-fixture",
		}),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "partial",
		}),
	))
	testresource.WriteResourceCLI(t, root, "fixture", "#!/usr/bin/env bash\nset -e\nprintf legacy > "+shellQuote(legacyMarker)+"\n")
	writeFakeDocker(t)

	controller := NewController(root, home)
	for _, action := range []string{"install", "start", "logs", "stop", "uninstall"} {
		if err := controller.Run("fixture", []string{action}, ioDiscard{}, ioDiscard{}); err != nil {
			t.Fatalf("Run(%s): %v", action, err)
		}
	}
	if _, err := os.Stat(legacyMarker); !os.IsNotExist(err) {
		t.Fatalf("legacy CLI marker exists after native docker commands, err=%v", err)
	}
}

func TestManifestNativeExternalCLIStandardCommandsDoNotFallbackToLegacyCLI(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	legacyMarker := filepath.Join(root, "legacy-external.marker")
	installMarker := filepath.Join(root, "install-external.marker")
	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDisplayName("Fixture CLI"),
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceTemplate("external-cli"),
		testresource.WithResourceDescription("Fixture external CLI resource"),
		testresource.WithResourceBinary("fixture-cli"),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "supported",
		}),
		testresource.WithResourceInstall(manifestpkg.ResourceInstall{
			Platforms: map[string][]string{
				"linux":   {"sh", "-c", "printf installed > " + shellQuote(installMarker)},
				"macos":   {"sh", "-c", "printf installed > " + shellQuote(installMarker)},
				"windows": {"sh", "-c", "printf installed > " + shellQuote(installMarker)},
			},
		}),
	))
	testresource.WriteResourceCLI(t, root, "fixture", "#!/usr/bin/env bash\nset -e\nprintf legacy > "+shellQuote(legacyMarker)+"\n")
	writeExecutableOnPath(t, "fixture-cli", "#!/usr/bin/env bash\necho 'fixture-cli 1.0.0'\n")

	controller := NewController(root, home)
	for _, action := range []string{"install", "status", "start", "stop", "logs"} {
		if err := controller.Run("fixture", []string{action}, ioDiscard{}, ioDiscard{}); err != nil {
			t.Fatalf("Run(%s): %v", action, err)
		}
	}
	if _, err := os.Stat(installMarker); err != nil {
		t.Fatalf("expected external CLI install marker: %v", err)
	}
	if _, err := os.Stat(legacyMarker); !os.IsNotExist(err) {
		t.Fatalf("legacy CLI marker exists after native external-cli commands, err=%v", err)
	}
}

func TestManifestNativeCloudAPIStandardCommandsDoNotFallbackToLegacyCLI(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	legacyMarker := filepath.Join(root, "legacy-cloud.marker")
	installMarker := filepath.Join(root, "install-cloud.marker")
	server := startHTTPServer(t, "127.0.0.1:"+strconv.Itoa(mustAllocatePort(t)), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer server.Shutdown(context.Background())

	testresource.WriteResourceManifest(t, root, "fixture", testresource.ResourceManifest(
		"fixture",
		testresource.WithResourceDisplayName("Fixture API"),
		testresource.WithResourceDriver("cloud-api"),
		testresource.WithResourceTemplate("cloud-api"),
		testresource.WithResourceDescription("Fixture cloud API resource"),
		testresource.WithResourceEndpoint("http://"+server.Addr+"/health"),
		testresource.WithResourceCredentialsEnv("FIXTURE_API_KEY"),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "supported",
		}),
		testresource.WithResourceInstall(manifestpkg.ResourceInstall{
			Platforms: map[string][]string{
				"linux":   {"sh", "-c", "printf installed > " + shellQuote(installMarker)},
				"macos":   {"sh", "-c", "printf installed > " + shellQuote(installMarker)},
				"windows": {"sh", "-c", "printf installed > " + shellQuote(installMarker)},
			},
		}),
		testresource.WithResourceHealthChecks(manifestpkg.ResourceHealthCheck{
			Type:           "http",
			Target:         "http://" + server.Addr + "/health",
			ExpectedStatus: []int{200, 400, 401, 403},
			TimeoutSeconds: 5,
		}),
	))
	testresource.WriteResourceCLI(t, root, "fixture", "#!/usr/bin/env bash\nset -e\nprintf legacy > "+shellQuote(legacyMarker)+"\n")
	t.Setenv("FIXTURE_API_KEY", "test-key")

	controller := NewController(root, home)
	for _, action := range []string{"install", "status", "start", "stop", "logs"} {
		if err := controller.Run("fixture", []string{action}, ioDiscard{}, ioDiscard{}); err != nil {
			t.Fatalf("Run(%s): %v", action, err)
		}
	}
	if _, err := os.Stat(installMarker); err != nil {
		t.Fatalf("expected cloud API install marker: %v", err)
	}
	if _, err := os.Stat(legacyMarker); !os.IsNotExist(err) {
		t.Fatalf("legacy CLI marker exists after native cloud-api commands, err=%v", err)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func writeExecutableOnPath(t *testing.T, name, contents string) string {
	t.Helper()
	path := testkitgo.WriteExecutableOnPath(t, name, contents)
	testresource.UseSystemLookPath(t, &lookPathCommandFn)
	testresource.UseSystemLookPath(t, &lookPathResourceFn)
	return path
}

func writeFakeDocker(t *testing.T) string {
	t.Helper()
	stateFile := testresource.WriteFakeDocker(t)
	testresource.UseSystemLookPath(t, &lookPathCommandFn)
	return stateFile
}

func projectRootForResourcesTest(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	return root
}

func mustListenTCP(t *testing.T, address string) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatalf("listen %s: %v", address, err)
	}
	return listener
}

func startHTTPServer(t *testing.T, address string, handler func(http.ResponseWriter, *http.Request)) *http.Server {
	t.Helper()
	server := &http.Server{Addr: address, Handler: http.HandlerFunc(handler)}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatalf("listen %s: %v", address, err)
	}
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		_ = listener.Close()
	})
	return server
}

func mustAllocatePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func copyManifestWithOverrides(t *testing.T, srcRoot, dstRoot, name string, primaryPort, secondaryPort int, healthType, healthPath string) {
	t.Helper()
	controller := NewController(srcRoot, t.TempDir())
	manifest, err := controller.loadResourceManifest(defaultResourceManifestPath(srcRoot, name))
	if err != nil {
		t.Fatalf("load manifest %s: %v", name, err)
	}
	manifest.Runtime.ContainerName = "fixture-" + name
	switch name {
	case "postgres", "redis":
		if len(manifest.Ports) > 0 {
			manifest.Ports[0].Host = primaryPort
		}
		if len(manifest.HealthChecks) > 0 {
			manifest.HealthChecks[0].Type = healthType
			manifest.HealthChecks[0].Target = "127.0.0.1:" + strconv.Itoa(primaryPort)
		}
	case "qdrant":
		if len(manifest.Ports) > 0 {
			manifest.Ports[0].Host = primaryPort
		}
		if len(manifest.Ports) > 1 {
			manifest.Ports[1].Host = secondaryPort
		}
		if len(manifest.HealthChecks) > 0 {
			manifest.HealthChecks[0].Type = healthType
			manifest.HealthChecks[0].Target = "http://127.0.0.1:" + strconv.Itoa(primaryPort) + healthPath
		}
	case "browserless":
		if len(manifest.Ports) > 0 {
			manifest.Ports[0].Host = primaryPort
		}
		if len(manifest.HealthChecks) > 0 {
			manifest.HealthChecks[0].Type = healthType
			manifest.HealthChecks[0].Target = "http://127.0.0.1:" + strconv.Itoa(primaryPort) + healthPath
		}
	case "vault":
		if len(manifest.Ports) > 0 {
			manifest.Ports[0].Host = primaryPort
		}
		if len(manifest.HealthChecks) > 0 {
			manifest.HealthChecks[0].Type = healthType
			manifest.HealthChecks[0].Target = "http://127.0.0.1:" + strconv.Itoa(primaryPort) + healthPath
		}
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest %s: %v", name, err)
	}
	path := defaultResourceManifestPath(dstRoot, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
