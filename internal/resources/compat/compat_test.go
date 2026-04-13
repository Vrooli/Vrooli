package compat

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/testutil"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

func TestResolveCLIPathUsesInstalledResourceCommand(t *testing.T) {
	service := New("/repo", "/home/test", func(name string) (string, error) {
		if name != "resource-redis" {
			t.Fatalf("LookPath called with %q", name)
		}
		return "/tmp/resource-redis", nil
	})

	path, ok := service.ResolveCLIPath("redis")
	if !ok || path != "/tmp/resource-redis" {
		t.Fatalf("ResolveCLIPath() = (%q, %v)", path, ok)
	}
}

func TestCommandForResourceFallsBackToLocalScript(t *testing.T) {
	root := t.TempDir()
	scriptPath := filepath.Join(root, "resources", "redis", "cli.sh")
	testutil.WriteExecutable(t, scriptPath, "#!/usr/bin/env bash\nexit 0\n")

	service := New(root, "/home/test", func(name string) (string, error) {
		return "", os.ErrNotExist
	})

	cmd, err := service.CommandForResource("redis", "status")
	if err != nil {
		t.Fatalf("CommandForResource: %v", err)
	}
	if !strings.HasSuffix(cmd.Path, "bash") {
		t.Fatalf("cmd.Path = %q, want bash executable", cmd.Path)
	}
	if got := strings.Join(cmd.Args, " "); !strings.Contains(got, "resources/redis/cli.sh status") {
		t.Fatalf("cmd.Args = %q", got)
	}
}

func TestCommandForResourceReturnsTypedUnavailableError(t *testing.T) {
	service := New("/repo", "/home/test", func(name string) (string, error) {
		return "", os.ErrNotExist
	})

	_, err := service.CommandForResource("missing", "status")
	var resourceErr *vroolierr.Error
	if !strings.Contains(err.Error(), "no installed CLI or cli.sh") || err == nil {
		t.Fatalf("CommandForResource error = %v", err)
	}
	if !errors.As(err, &resourceErr) {
		t.Fatalf("expected *vroolierr.Error, got %T", err)
	}
	if resourceErr.Code != ErrorCodeCommandUnavailable {
		t.Fatalf("resourceErr.Code = %q", resourceErr.Code)
	}
}

func TestExtractJSONPayloadTrimsWarningsAroundJSON(t *testing.T) {
	payload, ok := ExtractJSONPayload([]byte("warning\n{\"installed\":true}\n"))
	if !ok || string(payload) != "{\"installed\":true}" {
		t.Fatalf("ExtractJSONPayload() = (%q, %v)", payload, ok)
	}
}

func TestResourceEnvOverwritesRootAndHome(t *testing.T) {
	t.Setenv("VROOLI_ROOT", "/old")
	t.Setenv("APP_ROOT", "/old-app")
	t.Setenv("HOME", "/old-home")

	env := ResourceEnv("/repo", "/tmp/home")
	joined := strings.Join(env, "\n")
	for _, expected := range []string{"VROOLI_ROOT=/repo", "APP_ROOT=/repo", "HOME=/tmp/home"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("environment missing %q in %q", expected, joined)
		}
	}
}
