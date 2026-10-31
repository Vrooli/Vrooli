package build

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type wailsCommandRunner struct {
	command      string
	calls        []string
	env          map[string]string
	err          error
	createBinary bool
}

func (r *wailsCommandRunner) Run(ctx context.Context, _ string, command string) ([]byte, error) {
	r.command = command
	r.calls = append(r.calls, command)
	return nil, r.err
}
func (r *wailsCommandRunner) RunWithEnv(ctx context.Context, dir string, command string, env map[string]string) ([]byte, error) {
	r.command = command
	r.calls = append(r.calls, command)
	r.env = env
	if r.createBinary && r.err == nil {
		if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, "bin", "fixture"), []byte("binary"), 0o755); err != nil {
			return nil, err
		}
	}
	return []byte("Wails package output"), r.err
}

func TestScenarioBuildDispatchesWailsLinuxPackage(t *testing.T) {
	desktopPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(desktopPath, "wails-strategy.json"), []byte(`{"strategy_id":"wails-inprocess"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := &wailsCommandRunner{createBinary: true}
	store := NewStore()
	store.Save(&Status{BuildID: "wails", ScenarioName: "fixture", Status: "building", PlatformResults: map[string]*PlatformResult{"linux-amd64": {Platform: "linux-amd64"}}, Artifacts: map[string]string{}})
	service := NewService(WithStore(store), WithCommandRunner(runner))
	service.PerformScenarioDesktopBuild(context.Background(), "wails", "fixture", desktopPath, []string{"linux-amd64"}, false)
	status, _ := store.Get("wails")
	if runner.command != "go run github.com/wailsapp/wails/v3/cmd/wails3 package GOOS=linux" {
		t.Fatalf("command = %q", runner.command)
	}
	if len(runner.calls) != 2 || !strings.Contains(runner.calls[0], "scenario-dependency-analyzer deps install go/github.com/wailsapp/wails/v3@v3.0.0-beta.27") {
		t.Fatalf("commands = %#v, want SDA install followed by Wails package", runner.calls)
	}
	if runner.env["EXTRA_TAGS"] != "gtk3" {
		t.Fatalf("EXTRA_TAGS = %q, want gtk3", runner.env["EXTRA_TAGS"])
	}
	if status.Status != "ready" || status.PlatformResults["linux-amd64"].Status != "ready" {
		t.Fatalf("status = %#v", status)
	}
	if status.PlatformResults["linux-amd64"].Artifact != filepath.Join(desktopPath, "bin", "fixture") {
		t.Fatalf("artifact = %q", status.PlatformResults["linux-amd64"].Artifact)
	}
}

func TestWailsBuildRejectsNonLinuxTarget(t *testing.T) {
	desktopPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(desktopPath, "wails-strategy.json"), []byte(`{"strategy_id":"wails-inprocess"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := &wailsCommandRunner{}
	store := NewStore()
	store.Save(&Status{BuildID: "wails", Status: "building", PlatformResults: map[string]*PlatformResult{"windows-x64": {Platform: "windows-x64"}}, Artifacts: map[string]string{}})
	service := NewService(WithStore(store), WithCommandRunner(runner))
	service.PerformScenarioDesktopBuild(context.Background(), "wails", "fixture", desktopPath, []string{"windows-x64"}, false)
	status, _ := store.Get("wails")
	if len(runner.calls) != 0 {
		t.Fatalf("unexpected command for unsupported target: %#v", runner.calls)
	}
	if status.Status != "failed" || !containsBuildLog(status.PlatformResults["windows-x64"].ErrorLog, "Linux packaging only") {
		t.Fatalf("status = %#v", status)
	}
}

func TestWailsPackageCommandErrorFailsBuild(t *testing.T) {
	desktopPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(desktopPath, "wails-strategy.json"), []byte(`{"strategy_id":"wails-inprocess"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := &wailsCommandRunner{err: errors.New("missing CLI")}
	store := NewStore()
	store.Save(&Status{BuildID: "wails", Status: "building", PlatformResults: map[string]*PlatformResult{"linux": {Platform: "linux"}}, Artifacts: map[string]string{}})
	service := NewService(WithStore(store), WithCommandRunner(runner))
	service.PerformScenarioDesktopBuild(context.Background(), "wails", "fixture", desktopPath, []string{"linux"}, false)
	status, _ := store.Get("wails")
	if status.Status != "failed" {
		t.Fatalf("status = %#v", status)
	}
}

func containsBuildLog(values []string, expected string) bool {
	for _, value := range values {
		if strings.Contains(value, expected) {
			return true
		}
	}
	return false
}
