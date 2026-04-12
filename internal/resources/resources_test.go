package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=3 | LAST: 2026-04-11

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
	scriptPath := filepath.Join(root, "resources", "fixture", "cli.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(scriptPath), err)
	}
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write %s: %v", scriptPath, err)
	}

	originalRun := runCommandResource
	t.Cleanup(func() {
		runCommandResource = originalRun
	})

	item := Resource{
		Name:      "fixture",
		Path:      filepath.Join(root, "resources", "fixture"),
		Exists:    true,
		HasScript: true,
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

func TestStatusForResourceParsesStructuredPayload(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	controller := NewController(root, home)
	scriptPath := filepath.Join(root, "resources", "fixture", "cli.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(scriptPath), err)
	}
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write %s: %v", scriptPath, err)
	}

	originalRun := runCommandResource
	t.Cleanup(func() {
		runCommandResource = originalRun
	})
	runCommandResource = func(ctx context.Context, cmd *exec.Cmd) commandResult {
		return commandResult{output: []byte(`{"installed":true,"running":true,"healthy":true,"message":"healthy"}`)}
	}

	item := Resource{
		Name:      "fixture",
		Path:      filepath.Join(root, "resources", "fixture"),
		Exists:    true,
		HasScript: true,
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

	scriptPath := filepath.Join(root, "resources", "fixture", "cli.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(scriptPath), err)
	}
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env bash\nexit 7\n"), 0o755); err != nil {
		t.Fatalf("write %s: %v", scriptPath, err)
	}

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

func TestStartAllUsesBestEffortWhenStatusProbeIsDegraded(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceConfig(t, root, "fixture", true)
	writeResourceManifestFixture(t, root, `{
  "name": "fixture",
  "display_name": "Fixture adapter",
  "description": "Fixture legacy adapter",
  "template": "legacy-adapter",
  "driver": "legacy-adapter",
  "legacy_adapter": {
    "owner": "Matthew Halloran",
    "decision_deadline": "2026-05-31",
    "final_disposition": "migrate",
    "legacy_cli_path": "resources/fixture/cli.sh"
  },
  "portability_tier": "partial",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "partial"
  }
}`)
	scriptPath := filepath.Join(root, "resources", "fixture", "cli.sh")
	markerPath := filepath.Join(root, "fixture-started.txt")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(scriptPath), err)
	}
	script := "#!/usr/bin/env bash\nset -e\nif [[ \"$1\" == \"start\" ]]; then\n  printf 'started' > " + shellQuote(markerPath) + "\nfi\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write %s: %v", scriptPath, err)
	}

	originalRun := runCommandResource
	t.Cleanup(func() {
		runCommandResource = originalRun
	})
	runCommandResource = func(ctx context.Context, cmd *exec.Cmd) commandResult {
		return commandResult{output: []byte("not-json\n")}
	}

	report, err := NewController(root, home).StartAll(ioDiscard{}, ioDiscard{})
	if err != nil {
		t.Fatalf("StartAll: %v", err)
	}
	if len(report.Started) != 1 {
		t.Fatalf("len(report.Started) = %d, want 1", len(report.Started))
	}
	if !strings.Contains(report.Started[0].Message, "degraded status probe") {
		t.Fatalf("report.Started[0].Message = %q", report.Started[0].Message)
	}
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("expected best-effort start marker: %v", err)
	}
}

func TestStopAllFallsBackWhenStatusProbeIsDegraded(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceConfig(t, root, "fixture", false)
	writeResourceManifestFixture(t, root, `{
  "name": "fixture",
  "display_name": "Fixture adapter",
  "description": "Fixture legacy adapter",
  "template": "legacy-adapter",
  "driver": "legacy-adapter",
  "legacy_adapter": {
    "owner": "Matthew Halloran",
    "decision_deadline": "2026-05-31",
    "final_disposition": "migrate",
    "legacy_cli_path": "resources/fixture/cli.sh"
  },
  "portability_tier": "partial",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "partial"
  }
}`)
	scriptPath := filepath.Join(root, "resources", "fixture", "cli.sh")
	markerPath := filepath.Join(root, "fixture-stopped.txt")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(scriptPath), err)
	}
	script := "#!/usr/bin/env bash\nset -e\nif [[ \"$1\" == \"stop\" ]]; then\n  printf 'stopped' > " + shellQuote(markerPath) + "\nfi\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write %s: %v", scriptPath, err)
	}

	originalRun := runCommandResource
	t.Cleanup(func() {
		runCommandResource = originalRun
	})
	runCommandResource = func(ctx context.Context, cmd *exec.Cmd) commandResult {
		return commandResult{output: []byte("not-json\n")}
	}

	report, err := NewController(root, home).StopAll(ioDiscard{}, ioDiscard{})
	if err != nil {
		t.Fatalf("StopAll: %v", err)
	}
	if len(report.Stopped) != 1 {
		t.Fatalf("len(report.Stopped) = %d, want 1", len(report.Stopped))
	}
	if !strings.Contains(report.Stopped[0].Message, "degraded status probe") {
		t.Fatalf("report.Stopped[0].Message = %q", report.Stopped[0].Message)
	}
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("expected best-effort stop marker: %v", err)
	}
}

func TestDiscoverMarksManifestNativeResources(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceConfig(t, root, "fixture", true)
	writeResourceManifestFixture(t, root, `{
  "name": "fixture",
  "display_name": "Fixture",
  "description": "Fixture resource",
  "template": "docker-service",
  "driver": "docker-service",
  "portability_tier": "partial",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "partial"
  },
  "runtime": {
    "image": "fixture:latest",
    "container_name": "vrooli-fixture"
  }
}`)

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
	writeResourceConfig(t, root, "fixture", true)
	writeResourceScript(t, root, "fixture", "#!/usr/bin/env bash\nexit 0\n")

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
	writeResourceConfig(t, root, "fixture", true)

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
	writeResourceConfig(t, root, "fixture", true)
	writeResourceManifestFixture(t, root, `{
  "name": "fixture",
  "display_name": "Fixture",
  "description": "Fixture resource",
  "template": "docker-service",
  "driver": "docker-service",
  "portability_tier": "partial",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "partial"
  },
  "runtime": {
    "image": "fixture:latest",
    "container_name": "vrooli-fixture"
  }
}`)
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
	writeResourceConfig(t, root, "fixture", true)
	writeResourceManifestFixture(t, root, `{
  "name": "fixture",
  "display_name": "Fixture",
  "description": "Fixture resource",
  "template": "docker-service",
  "driver": "docker-service",
  "portability_tier": "partial",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "partial"
  },
  "runtime": {
    "image": "fixture:latest",
    "container_name": "vrooli-fixture"
  }
}`)
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

func TestStatusForManifestNativeComposeResource(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceConfig(t, root, "fixture", true)
	writeResourceManifestFixture(t, root, `{
  "name": "fixture",
  "display_name": "Fixture Compose",
  "description": "Fixture compose resource",
  "template": "compose-service",
  "driver": "compose-service",
  "compose_file": "compose.yaml",
  "portability_tier": "partial",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "partial"
  }
}`)
	writeResourceFileFixture(t, root, filepath.Join("resources", "fixture", "compose.yaml"), "services:\n  app:\n    image: fixture:latest\n")
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
	writeResourceConfig(t, root, "fixture", true)
	writeResourceManifestFixture(t, root, `{
  "name": "fixture",
  "display_name": "Fixture Compose",
  "description": "Fixture compose resource",
  "template": "compose-service",
  "driver": "compose-service",
  "compose_file": "compose.yaml",
  "portability_tier": "partial",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "partial"
  }
}`)
	writeResourceFileFixture(t, root, filepath.Join("resources", "fixture", "compose.yaml"), "services:\n  app:\n    image: fixture:latest\n")
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
	writeResourceConfig(t, root, "fixture", true)
	writeResourceManifestFixture(t, root, `{
  "name": "fixture",
  "display_name": "Fixture",
  "description": "Fixture resource",
  "template": "docker-service",
  "driver": "docker-service",
  "portability_tier": "platform-specific",
  "platforms": {
    "linux": "unsupported",
    "macos": "unsupported",
    "windows": "unsupported"
  },
  "runtime": {
    "image": "fixture:latest"
  }
}`)

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
	writeResourceConfig(t, root, "fixture", true)
	commandPath := writeExecutableOnPath(t, "fixture-cli", "#!/usr/bin/env bash\nif [[ \"$1\" == \"--version\" ]]; then\n  echo 'fixture-cli 1.0.0'\n  exit 0\nfi\nexit 0\n")
	writeResourceManifestFixture(t, root, `{
  "name": "fixture",
  "display_name": "Fixture CLI",
  "description": "Fixture external CLI resource",
  "template": "external-cli",
  "driver": "external-cli",
  "binary": "fixture-cli",
  "version_args": ["--version"],
  "portability_tier": "full",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "supported"
  },
  "health_checks": [
    {
      "type": "command",
      "command": ["`+commandPath+`", "--version"]
    }
  ]
}`)

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

func TestRunManifestNativeExternalCLIInstallAndFallback(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceConfig(t, root, "fixture", true)
	installMarker := filepath.Join(root, "install.marker")
	customMarker := filepath.Join(root, "custom.marker")
	writeResourceManifestFixture(t, root, `{
  "name": "fixture",
  "display_name": "Fixture CLI",
  "description": "Fixture external CLI resource",
  "template": "external-cli",
  "driver": "external-cli",
  "binary": "fixture-cli",
  "portability_tier": "full",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "supported"
  },
  "install": {
    "platforms": {
      "linux": ["sh", "-c", "printf installed > `+shellQuote(installMarker)+`"],
      "macos": ["sh", "-c", "printf installed > `+shellQuote(installMarker)+`"],
      "windows": ["sh", "-c", "printf installed > `+shellQuote(installMarker)+`"]
    }
  }
}`)
	writeExecutableOnPath(t, "fixture-cli", "#!/usr/bin/env bash\necho 'fixture-cli 1.0.0'\n")
	writeResourceScript(t, root, "fixture", "#!/usr/bin/env bash\nset -e\nif [[ \"$1\" == \"custom\" ]]; then\n  printf custom > "+shellQuote(customMarker)+"\n  exit 0\nfi\nexit 0\n")

	controller := NewController(root, home)
	if err := controller.Run("fixture", []string{"install"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatalf("Run(install): %v", err)
	}
	if _, err := os.Stat(installMarker); err != nil {
		t.Fatalf("expected install marker: %v", err)
	}
	if err := controller.Run("fixture", []string{"custom"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatalf("Run(custom): %v", err)
	}
	if _, err := os.Stat(customMarker); err != nil {
		t.Fatalf("expected custom fallback marker: %v", err)
	}
}

func TestStatusForManifestNativeCloudAPIResource(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceConfig(t, root, "fixture", true)
	server := startHTTPServer(t, "127.0.0.1:"+strconv.Itoa(mustAllocatePort(t)), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer server.Shutdown(context.Background())

	writeResourceManifestFixture(t, root, `{
  "name": "fixture",
  "display_name": "Fixture API",
  "description": "Fixture cloud API resource",
  "template": "cloud-api",
  "driver": "cloud-api",
  "endpoint": "http://`+server.Addr+`/health",
  "credentials": {
    "env": ["FIXTURE_API_KEY"]
  },
  "portability_tier": "full",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "supported"
  },
  "health_checks": [
    {
      "type": "http",
      "target": "http://`+server.Addr+`/health",
      "expected_status": [401]
    }
  ]
}`)
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
	writeResourceConfig(t, root, "fixture", true)
	writeResourceManifestFixture(t, root, `{
  "name": "fixture",
  "display_name": "Fixture API",
  "description": "Fixture cloud API resource",
  "template": "cloud-api",
  "driver": "cloud-api",
  "endpoint": "https://api.example.com/health",
  "credentials": {
    "env": ["FIXTURE_API_KEY"]
  },
  "portability_tier": "full",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "supported"
  }
}`)

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
		"claude-code":           "external-cli",
		"codex":                 "external-cli",
		"k6":                    "external-cli",
		"opencode":              "external-cli",
		"sqlite":                "external-cli",
		"ollama":                "docker-service",
		"judge0":                "compose-service",
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

	for _, name := range []string{"postgres", "redis", "qdrant", "browserless", "vault", "litellm", "minio", "neo4j", "questdb", "searxng", "claude-code", "codex", "k6", "opencode", "sqlite", "ollama", "judge0", "unstructured-io", "gemini", "openrouter", "twilio", "cloudflare-ai-gateway"} {
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

func TestProjectPhase6KeepResourcesAreExplicitlyTyped(t *testing.T) {
	root := projectRootForResourcesTest(t)
	controller := NewController(root, t.TempDir())

	items, err := controller.Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	keepResources := phase0InventoryResourcesByState(t, root, "keep")
	discovered := make(map[string]Resource, len(items))
	for _, item := range items {
		discovered[item.Name] = item
	}

	for _, name := range keepResources {
		item, ok := discovered[name]
		if !ok {
			t.Fatalf("keep resource %q not discovered", name)
		}
		if item.ControlMode == "legacy-shell" {
			t.Fatalf("keep resource %q is still hidden as legacy-shell", name)
		}
		switch item.ControlMode {
		case "manifest-native":
			if item.ManifestPath == "" {
				t.Fatalf("%s manifest-native resource missing manifest path", name)
			}
		case "legacy-adapter":
			if item.LegacyAdapter.Owner == "" {
				t.Fatalf("%s legacy adapter owner is empty", name)
			}
			if item.LegacyAdapter.DecisionDeadline == "" {
				t.Fatalf("%s legacy adapter deadline is empty", name)
			}
			if item.LegacyAdapter.FinalDisposition == "" {
				t.Fatalf("%s legacy adapter final disposition is empty", name)
			}
			if item.LegacyAdapter.LegacyCLIPath == "" {
				t.Fatalf("%s legacy adapter legacy cli path is empty", name)
			}
		default:
			t.Fatalf("%s ControlMode = %q, want manifest-native or legacy-adapter", name, item.ControlMode)
		}
	}
}

func TestProjectPhase6LegacyAdaptersHaveDecisionMetadata(t *testing.T) {
	root := projectRootForResourcesTest(t)
	controller := NewController(root, t.TempDir())

	adapterNames := []string{
		"comfyui",
		"home-assistant",
		"kokoro",
		"mail-in-a-box",
		"postgis",
		"sagemath",
		"whisper",
	}
	deadlinePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

	for _, name := range adapterNames {
		manifest, err := controller.loadResourceManifest(defaultResourceManifestPath(root, name))
		if err != nil {
			t.Fatalf("loadResourceManifest(%s): %v", name, err)
		}
		if manifest.Driver != "legacy-adapter" {
			t.Fatalf("%s driver = %q, want legacy-adapter", name, manifest.Driver)
		}
		if manifest.Template != "legacy-adapter" {
			t.Fatalf("%s template = %q, want legacy-adapter", name, manifest.Template)
		}
		if manifest.LegacyAdapter.Owner == "" {
			t.Fatalf("%s owner is empty", name)
		}
		if !deadlinePattern.MatchString(manifest.LegacyAdapter.DecisionDeadline) {
			t.Fatalf("%s deadline = %q, want YYYY-MM-DD", name, manifest.LegacyAdapter.DecisionDeadline)
		}
		if manifest.LegacyAdapter.FinalDisposition != "migrate" && manifest.LegacyAdapter.FinalDisposition != "blueprint" && manifest.LegacyAdapter.FinalDisposition != "deprecate" {
			t.Fatalf("%s final disposition = %q", name, manifest.LegacyAdapter.FinalDisposition)
		}
		if manifest.LegacyAdapter.LegacyCLIPath != filepath.ToSlash(filepath.Join("resources", name, "cli.sh")) {
			t.Fatalf("%s legacy cli path = %q", name, manifest.LegacyAdapter.LegacyCLIPath)
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
		switch item.ControlMode {
		case "manifest-native", "legacy-adapter":
		default:
			t.Fatalf("%s ControlMode = %q, want manifest-native or legacy-adapter", item.Name, item.ControlMode)
		}
	}
}

func TestProjectDockerResourceStatusesUseNativeManifests(t *testing.T) {
	projectRoot := projectRootForResourcesTest(t)
	root := t.TempDir()
	home := t.TempDir()
	controller := NewController(root, home)
	stateFile := writeFakeDocker(t)

	writeResourceConfig(t, root, "postgres", true)
	writeResourceConfig(t, root, "redis", true)
	writeResourceConfig(t, root, "qdrant", true)
	writeResourceConfig(t, root, "browserless", true)
	writeResourceConfig(t, root, "vault", true)

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
	writeResourceConfig(t, root, "fixture", true)
	legacyMarker := filepath.Join(root, "legacy-docker.marker")
	writeResourceManifestFixture(t, root, `{
  "name": "fixture",
  "display_name": "Fixture Docker",
  "description": "Fixture docker resource",
  "template": "docker-service",
  "driver": "docker-service",
  "portability_tier": "full",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "partial"
  },
  "runtime": {
    "image": "fixture:latest",
    "container_name": "vrooli-fixture"
  }
}`)
	writeResourceScript(t, root, "fixture", "#!/usr/bin/env bash\nset -e\nprintf legacy > "+shellQuote(legacyMarker)+"\n")
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
	writeResourceConfig(t, root, "fixture", true)
	legacyMarker := filepath.Join(root, "legacy-external.marker")
	installMarker := filepath.Join(root, "install-external.marker")
	writeResourceManifestFixture(t, root, `{
  "name": "fixture",
  "display_name": "Fixture CLI",
  "description": "Fixture external CLI resource",
  "template": "external-cli",
  "driver": "external-cli",
  "binary": "fixture-cli",
  "portability_tier": "full",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "supported"
  },
  "install": {
    "platforms": {
      "linux": ["sh", "-c", "printf installed > `+shellQuote(installMarker)+`"],
      "macos": ["sh", "-c", "printf installed > `+shellQuote(installMarker)+`"],
      "windows": ["sh", "-c", "printf installed > `+shellQuote(installMarker)+`"]
    }
  }
}`)
	writeResourceScript(t, root, "fixture", "#!/usr/bin/env bash\nset -e\nprintf legacy > "+shellQuote(legacyMarker)+"\n")
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
	writeResourceConfig(t, root, "fixture", true)
	legacyMarker := filepath.Join(root, "legacy-cloud.marker")
	installMarker := filepath.Join(root, "install-cloud.marker")
	server := startHTTPServer(t, "127.0.0.1:"+strconv.Itoa(mustAllocatePort(t)), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer server.Shutdown(context.Background())

	writeResourceManifestFixture(t, root, `{
  "name": "fixture",
  "display_name": "Fixture API",
  "description": "Fixture cloud API resource",
  "template": "cloud-api",
  "driver": "cloud-api",
  "endpoint": "http://`+server.Addr+`/health",
  "credentials": {
    "env": ["FIXTURE_API_KEY"]
  },
  "portability_tier": "full",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "supported"
  },
  "install": {
    "platforms": {
      "linux": ["sh", "-c", "printf installed > `+shellQuote(installMarker)+`"],
      "macos": ["sh", "-c", "printf installed > `+shellQuote(installMarker)+`"],
      "windows": ["sh", "-c", "printf installed > `+shellQuote(installMarker)+`"]
    }
  },
  "health_checks": [
    {
      "type": "http",
      "target": "http://`+server.Addr+`/health",
      "expected_status": [200, 400, 401, 403],
      "timeout_seconds": 5
    }
  ]
}`)
	writeResourceScript(t, root, "fixture", "#!/usr/bin/env bash\nset -e\nprintf legacy > "+shellQuote(legacyMarker)+"\n")
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

func writeResourceConfig(t *testing.T, root, name string, enabled bool) {
	t.Helper()
	configPath := filepath.Join(root, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(configPath), err)
	}
	payload := map[string]any{
		"dependencies": map[string]any{
			"resources": map[string]any{
				name: map[string]any{"enabled": enabled},
			},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", configPath, err)
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func writeResourceManifestFixture(t *testing.T, root, contents string) {
	t.Helper()
	writeResourceManifest(t, root, "fixture", contents)
}

func writeResourceManifest(t *testing.T, root, name, contents string) {
	t.Helper()
	path := defaultResourceManifestPath(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeResourceScript(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, "resources", name, "cli.sh")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeExecutableOnPath(t *testing.T, name, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	originalLookPath := lookPathCommandFn
	lookPathCommandFn = exec.LookPath
	t.Cleanup(func() {
		lookPathCommandFn = originalLookPath
	})
	return path
}

func writeFakeDocker(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "docker-state.txt")
	scriptPath := filepath.Join(dir, "docker")
	script := `#!/usr/bin/env bash
set -e
state_file="${FAKE_DOCKER_STATE}"
cmd="${1:-}"
shift || true

case "$cmd" in
  compose)
    while [[ $# -gt 0 ]]; do
      case "$1" in
        -f|--project-name)
          shift 2
          ;;
        *)
          break
          ;;
      esac
    done
    subcmd="${1:-}"
    shift || true
    case "$subcmd" in
      ps)
        if [[ "${1:-}" == "-a" ]]; then
          shift
        fi
        if [[ "${1:-}" == "--format" ]]; then
          shift 2
        fi
        if [[ -f "$state_file" ]]; then
          state="$(tr -d '\n' < "$state_file")"
          if [[ "$state" == "running" ]]; then
            printf '[{"Service":"app","State":"running","Health":"healthy"}]'
          else
            printf '[{"Service":"app","State":"exited","Health":""}]'
          fi
        else
          printf '[]'
        fi
        exit 0
        ;;
      pull|up)
        printf 'running\n' > "$state_file"
        exit 0
        ;;
      stop)
        printf 'stopped\n' > "$state_file"
        exit 0
        ;;
      down)
        rm -f "$state_file"
        exit 0
        ;;
      logs)
        echo "fixture logs"
        exit 0
        ;;
    esac
    ;;
  image)
    if [[ "${1:-}" == "inspect" ]]; then
      exit 0
    fi
    ;;
  inspect)
    if [[ -f "$state_file" ]]; then
      state="$(tr -d '\n' < "$state_file")"
      if [[ "$state" == "running" ]]; then
        printf '{"Running":true}'
      else
        printf '{"Running":false}'
      fi
      exit 0
    fi
    echo "Error: No such object" >&2
    exit 1
    ;;
  run)
    printf 'running\n' > "$state_file"
    echo "container-id"
    exit 0
    ;;
  start)
    printf 'running\n' > "$state_file"
    exit 0
    ;;
  stop)
    printf 'stopped\n' > "$state_file"
    exit 0
    ;;
  restart)
    printf 'running\n' > "$state_file"
    exit 0
    ;;
  rm)
    rm -f "$state_file"
    exit 0
    ;;
  logs)
    echo "fixture logs"
    exit 0
    ;;
esac

echo "unexpected docker invocation: $cmd $*" >&2
exit 1
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write %s: %v", scriptPath, err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_DOCKER_STATE", stateFile)
	originalLookPath := lookPathCommandFn
	lookPathCommandFn = exec.LookPath
	t.Cleanup(func() {
		lookPathCommandFn = originalLookPath
	})
	return stateFile
}

func writeResourceFileFixture(t *testing.T, root, relPath, contents string) {
	t.Helper()
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
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

func phase0InventoryResourcesByState(t *testing.T, root, state string) []string {
	t.Helper()
	path := filepath.Join(root, "docs", "resources", "resource-phase0-inventory.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	lines := strings.Split(string(data), "\n")
	results := make([]string, 0)
	for _, line := range lines {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 7 {
			continue
		}
		name := strings.TrimSpace(parts[1])
		proposedState := strings.TrimSpace(parts[6])
		name = strings.Trim(name, "`")
		proposedState = strings.Trim(proposedState, "`")
		if proposedState == state {
			results = append(results, name)
		}
	}
	if len(results) == 0 {
		t.Fatalf("no Phase 0 inventory resources found for state %q", state)
	}
	return results
}
