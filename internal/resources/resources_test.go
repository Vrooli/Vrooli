package resources

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=2 | LAST: 2026-04-11

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
