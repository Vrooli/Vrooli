package build

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"deployment-manager/bundles"
)

func TestOutputPathsAndPlatformFiltering(t *testing.T) {
	config := &BuildConfig{OutputPattern: "out/{{platform}}/app{{ext}}"}
	if got := ResolveOutputPath("/work", config, SupportedPlatforms[4]); got != filepath.Join("/work", "out/win-x64/app.exe") {
		t.Fatalf("unexpected output path %q", got)
	}
	b := NewBuilder("/work", nil)
	got := b.replacePlaceholders("{{platform}} {{goos}} {{goarch}} {{ext}} {{output}}", SupportedPlatforms[0], "/tmp/a")
	if got != "linux-x64 linux amd64  /tmp/a" {
		t.Fatalf("unexpected replacements %q", got)
	}
	filtered := filterPlatforms([]string{"WIN-X64", "missing", "linux-x64"})
	if len(filtered) != 2 || filtered[0].Name != "linux-x64" || filtered[1].Name != "win-x64" {
		t.Fatalf("unexpected platform filter: %+v", filtered)
	}
}

func TestBuilderBuildsGoAndCustomTargets(t *testing.T) {
	root := t.TempDir()
	goDir := filepath.Join(root, "goapp")
	if err := os.MkdirAll(goDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goDir, "go.mod"), []byte("module example.com/test\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	logs := []string{}
	b := NewBuilder(root, func(level string, fields map[string]interface{}) {
		logs = append(logs, level+":"+fields["msg"].(string))
	})
	result, err := b.BuildAll(context.Background(), "goapp", &BuildConfig{Type: "go", SourceDir: "goapp", OutputPattern: "bin/{{platform}}/app{{ext}}"}, []string{"linux-x64"})
	if err != nil || !result.AllSucceeded || len(result.Results) != 1 {
		t.Fatalf("go build failed: result=%+v err=%v", result, err)
	}
	if _, err := os.Stat(result.Results[0].OutputPath); err != nil {
		t.Fatalf("built binary missing: %v", err)
	}
	custom := &BuildConfig{Type: "custom", SourceDir: "goapp", Args: []string{"sh", "-c", "printf custom > '{{output}}'"}, OutputPattern: "bin/{{platform}}/custom"}
	result, err = b.BuildAll(context.Background(), "custom", custom, []string{"linux-x64"})
	if err != nil || !result.AllSucceeded {
		t.Fatalf("custom build failed: result=%+v err=%v", result, err)
	}
	data, err := os.ReadFile(result.Results[0].OutputPath)
	if err != nil || string(data) != "custom" {
		t.Fatalf("unexpected custom output %q (%v)", data, err)
	}
	if len(logs) < 2 || !strings.Contains(logs[0], "build succeeded") {
		t.Fatalf("expected success logs: %v", logs)
	}
}

func TestBuilderRejectsUnsupportedAndInvalidConfigs(t *testing.T) {
	b := NewBuilder(t.TempDir(), func(string, map[string]interface{}) {})
	if _, err := b.BuildAll(context.Background(), "x", nil, nil); err == nil {
		t.Fatal("expected nil config error")
	}
	result, err := b.BuildAll(context.Background(), "x", &BuildConfig{Type: "unknown", SourceDir: "."}, []string{"linux-x64"})
	if err != nil || result.AllSucceeded || result.Results[0].Error == "" {
		t.Fatalf("expected unsupported type result: %+v %v", result, err)
	}
	if err := b.buildRust(context.Background(), &BuildConfig{}, SupportedPlatforms[0], filepath.Join(t.TempDir(), "out")); err == nil {
		t.Fatal("expected missing cargo output error")
	}
	if err := b.buildCustom(context.Background(), &BuildConfig{}, SupportedPlatforms[0], ""); err == nil {
		t.Fatal("expected custom command validation error")
	}
}

func TestAutoBuildStoreAndHelpers(t *testing.T) {
	store := NewAutoBuildStore()
	status := &AutoBuildStatus{BuildID: "b1", Targets: []AutoBuildTargetStatus{{ID: "t", Platforms: []AutoBuildPlatformStatus{{Name: "linux-x64"}}}}, BuildLog: []string{"start"}}
	store.Save(status)
	store.Update("b1", func(s *AutoBuildStatus) { s.Status = "done"; s.BuildLog = append(s.BuildLog, "done") })
	copyStatus, ok := store.Get("b1")
	if !ok || copyStatus.Status != "done" || &copyStatus.Targets[0] == &status.Targets[0] {
		t.Fatalf("unexpected store copy: %+v", copyStatus)
	}
	if store.Update("missing", func(*AutoBuildStatus) {}) {
		t.Fatal("missing update should report false")
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "example.test"
	req.Header.Set("X-Forwarded-Proto", "https")
	if got := buildAutoStatusURL(req, "b1"); got != "https://example.test/api/v1/build/auto/b1" {
		t.Fatalf("unexpected status URL %q", got)
	}
	if buildAutoStatusURL(nil, "b1") != "" || buildAutoCheckCommand("") != "" {
		t.Fatal("empty helper inputs should remain empty")
	}
	if !strings.Contains(buildAutoCheckCommand("/status"), "/status") || generateAutoBuildID() == "" {
		t.Fatal("expected check command and build id")
	}
}

func TestAutoTargetDiscoveryAndHandlerFiltering(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"api", "cli", "other"} {
		path := filepath.Join(root, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if dir != "other" {
			if err := os.WriteFile(filepath.Join(path, "go.mod"), []byte("module example.com/"+dir+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	targets := detectGoTargets(root, "demo", []string{"api"})
	if len(targets) != 1 || targets[0].ID != "demo-api" || targets[0].Config.OutputPattern == "" {
		t.Fatalf("unexpected targets: %+v", targets)
	}
	if binaryBaseName("demo", "cli") != "demo" || binaryBaseName("demo", "worker") != "demo-worker" || serviceIDForTarget("demo", "api") != "demo-api" {
		t.Fatal("unexpected target naming")
	}
	if !isGoModule(filepath.Join(root, "api")) || isGoModule(filepath.Join(root, "other")) {
		t.Fatal("unexpected module detection")
	}
	h := &Handler{}
	services := []bundles.ServiceEntry{{ID: "a", Build: &BuildConfig{}}, {ID: "b"}, {ID: "c", Build: &BuildConfig{}}}
	if got := h.filterBuildableServices(services, []string{"c"}); len(got) != 1 || got[0].ID != "c" {
		t.Fatalf("unexpected service filtering: %+v", got)
	}
}
