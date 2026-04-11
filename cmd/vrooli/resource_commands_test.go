package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/resources"
)

func TestRunResourceBlueprintListCommandJSON(t *testing.T) {
	controller := resources.NewController(projectRootForCLI(t), t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceBlueprintCommand(controller, globalOptions{json: true}, []string{"list"}, &stdout); err != nil {
		t.Fatalf("runResourceBlueprintCommand(list): %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"success": true`) || !strings.Contains(output, `"blueprints":`) {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunResourceBlueprintInfoCommandHuman(t *testing.T) {
	controller := resources.NewController(projectRootForCLI(t), t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceBlueprintCommand(controller, globalOptions{}, []string{"info", "terraform"}, &stdout); err != nil {
		t.Fatalf("runResourceBlueprintCommand(info): %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Terraform") || !strings.Contains(output, "external-cli") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunResourceBlueprintValidateCommandHuman(t *testing.T) {
	controller := resources.NewController(projectRootForCLI(t), t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceBlueprintCommand(controller, globalOptions{}, []string{"validate"}, &stdout); err != nil {
		t.Fatalf("runResourceBlueprintCommand(validate): %v", err)
	}
	if !strings.Contains(stdout.String(), "Validated") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestRunResourceBlueprintSearchCommandHuman(t *testing.T) {
	controller := resources.NewController(projectRootForCLI(t), t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceBlueprintCommand(controller, globalOptions{}, []string{"search", "network"}, &stdout); err != nil {
		t.Fatalf("runResourceBlueprintCommand(search): %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "networking") && !strings.Contains(output, "Network") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunResourceBlueprintSearchCommandJSON(t *testing.T) {
	controller := resources.NewController(projectRootForCLI(t), t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceBlueprintCommand(controller, globalOptions{json: true}, []string{"search", "network"}, &stdout); err != nil {
		t.Fatalf("runResourceBlueprintCommand(search json): %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"success": true`) || !strings.Contains(output, `"query": "network"`) || !strings.Contains(output, `"blueprints":`) {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestShowResourceHelpIncludesBlueprintCommands(t *testing.T) {
	var stdout bytes.Buffer
	showResourceHelp(&stdout)

	if !strings.Contains(stdout.String(), "resource blueprint") {
		t.Fatalf("help missing blueprint usage: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "list-deprecated") {
		t.Fatalf("help missing deprecated resource usage: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "resource template") {
		t.Fatalf("help missing resource template usage: %q", stdout.String())
	}
}

func TestRunResourceTemplateListCommandJSON(t *testing.T) {
	controller := resources.NewController(projectRootForCLI(t), t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceTemplateCommand(controller, globalOptions{json: true}, []string{"list"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("runResourceTemplateCommand(list): %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"success": true`) || !strings.Contains(output, `"templates":`) {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunResourceTemplateShowCommandHuman(t *testing.T) {
	controller := resources.NewController(projectRootForCLI(t), t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceTemplateCommand(controller, globalOptions{}, []string{"show", "docker-service"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("runResourceTemplateCommand(show): %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Docker Service") || !strings.Contains(output, "docker-service") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunResourceTemplateValidateCommandHuman(t *testing.T) {
	controller := resources.NewController(projectRootForCLI(t), t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceTemplateCommand(controller, globalOptions{}, []string{"validate"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("runResourceTemplateCommand(validate): %v", err)
	}
	if !strings.Contains(stdout.String(), "Validated") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestRunResourceTemplateGenerateCommandHuman(t *testing.T) {
	controller := resources.NewController(projectRootForCLI(t), t.TempDir())
	dest := filepath.Join(t.TempDir(), "terraform")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := runResourceTemplateCommand(controller, globalOptions{}, []string{"generate", "--from-blueprint", "terraform", "--dest", dest}, &stdout, &stderr); err != nil {
		t.Fatalf("runResourceTemplateCommand(generate): %v", err)
	}
	if !strings.Contains(stdout.String(), "Generated resource template") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dest, "resource.json")); err != nil {
		t.Fatalf("generated resource.json missing: %v", err)
	}
}

func TestRunResourceListDeprecatedCommandJSON(t *testing.T) {
	root := t.TempDir()
	controller := resources.NewController(root, t.TempDir())
	writeDeprecatedMetadataForCLI(t, root, resources.DeprecatedResource{
		Name:                "fixture",
		DeprecatedAt:        "2026-04-11",
		Reason:              "test",
		ArchivePath:         "/tmp/archive",
		ArchiveHash:         "abc",
		RetentionPolicyDays: 90,
		RestoreSupported:    true,
		PurgeAfter:          "2026-07-10",
	})
	var stdout bytes.Buffer

	if err := runResourceListDeprecatedCommand(controller, globalOptions{json: true}, nil, &stdout); err != nil {
		t.Fatalf("runResourceListDeprecatedCommand: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"success": true`) || !strings.Contains(output, `"resources":`) {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunResourceDeprecateCommandHuman(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceConfigForCLI(t, root, "fixture", true)
	writeResourceCLIForCLI(t, root, "fixture")
	var stdout bytes.Buffer

	if err := runResourceDeprecateCommand(resources.NewController(root, home), globalOptions{}, []string{"fixture"}, &stdout); err != nil {
		t.Fatalf("runResourceDeprecateCommand: %v", err)
	}
	if !strings.Contains(stdout.String(), "Deprecated fixture") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestRunResourceRestoreCommandHuman(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceConfigForCLI(t, root, "fixture", true)
	writeResourceCLIForCLI(t, root, "fixture")
	controller := resources.NewController(root, home)
	if _, err := controller.DeprecateResource("fixture"); err != nil {
		t.Fatalf("DeprecateResource: %v", err)
	}
	var stdout bytes.Buffer

	if err := runResourceRestoreCommand(controller, globalOptions{}, []string{"fixture"}, &stdout); err != nil {
		t.Fatalf("runResourceRestoreCommand: %v", err)
	}
	if !strings.Contains(stdout.String(), "Restored fixture") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestRunResourceArchiveGCCommandJSON(t *testing.T) {
	root := t.TempDir()
	controller := resources.NewController(root, t.TempDir())
	writeDeprecatedMetadataForCLI(t, root, resources.DeprecatedResource{
		Name:                "fixture",
		DeprecatedAt:        "2026-01-01",
		Reason:              "test",
		ArchivePath:         filepath.Join(root, "archive", "fixture"),
		ArchiveHash:         "abc",
		RetentionPolicyDays: 90,
		RestoreSupported:    true,
		PurgeAfter:          "2026-01-02",
	})
	if err := os.MkdirAll(filepath.Join(root, "archive", "fixture"), 0o755); err != nil {
		t.Fatalf("mkdir archive: %v", err)
	}
	var stdout bytes.Buffer

	originalNow := timeNowForResourceGC
	timeNowForResourceGC = func() time.Time { return time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC) }
	defer func() {
		timeNowForResourceGC = originalNow
	}()
	if err := runResourceArchiveGCCommand(controller, globalOptions{json: true}, nil, &stdout); err != nil {
		t.Fatalf("runResourceArchiveGCCommand: %v", err)
	}
	if !strings.Contains(stdout.String(), `"success": true`) {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestRunResourceListCommandShowsManifestMetadata(t *testing.T) {
	root := t.TempDir()
	writeResourceConfigForCLI(t, root, "fixture", true)
	writeResourceManifestForCLI(t, root, "fixture", `{
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
    "image": "fixture:latest"
  }
}`)
	controller := resources.NewController(root, t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceListCommand(controller, globalOptions{}, nil, &stdout); err != nil {
		t.Fatalf("runResourceListCommand: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "manifest-native") || !strings.Contains(output, "docker-service") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunResourceStatusCommandShowsManifestMetadata(t *testing.T) {
	root := t.TempDir()
	writeResourceConfigForCLI(t, root, "fixture", true)
	writeResourceManifestForCLI(t, root, "fixture", `{
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
	controller := resources.NewController(root, t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceStatusCommand(controller, globalOptions{}, []string{"fixture"}, &stdout); err != nil {
		t.Fatalf("runResourceStatusCommand: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "manifest-native") || !strings.Contains(output, "docker-service") || !strings.Contains(output, "platform-specific") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunResourceListCommandShowsMigratedCoreResources(t *testing.T) {
	controller := resources.NewController(projectRootForCLI(t), t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceListCommand(controller, globalOptions{}, nil, &stdout); err != nil {
		t.Fatalf("runResourceListCommand: %v", err)
	}
	output := stdout.String()
	for _, name := range []string{"postgres", "redis", "qdrant", "browserless"} {
		if !strings.Contains(output, name) {
			t.Fatalf("expected %q in output: %q", name, output)
		}
	}
	if !strings.Contains(output, "manifest-native") {
		t.Fatalf("expected manifest-native resources in output: %q", output)
	}
}

func TestRunSingleResourceControlCommandUsesManifestNativeDockerLifecycle(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceConfigForCLI(t, root, "fixture", true)
	writeResourceManifestForCLI(t, root, "fixture", `{
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

	stateFile := writeFakeDockerForCLI(t)
	controller := resources.NewController(root, home)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := runSingleResourceControlCommand(controller, "install", []string{"fixture"}, &stdout, &stderr); err != nil {
		t.Fatalf("runSingleResourceControlCommand(install): %v", err)
	}
	if err := runSingleResourceControlCommand(controller, "start", []string{"fixture"}, &stdout, &stderr); err != nil {
		t.Fatalf("runSingleResourceControlCommand(start): %v", err)
	}
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if !strings.Contains(string(data), "running") {
		t.Fatalf("state after start = %q", string(data))
	}
	if err := runSingleResourceControlCommand(controller, "logs", []string{"fixture"}, &stdout, &stderr); err != nil {
		t.Fatalf("runSingleResourceControlCommand(logs): %v", err)
	}
	if !strings.Contains(stdout.String(), "fixture logs") {
		t.Fatalf("stdout = %q, want docker logs", stdout.String())
	}
	if err := runSingleResourceControlCommand(controller, "stop", []string{"fixture"}, &stdout, &stderr); err != nil {
		t.Fatalf("runSingleResourceControlCommand(stop): %v", err)
	}
	data, err = os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if !strings.Contains(string(data), "stopped") {
		t.Fatalf("state after stop = %q", string(data))
	}
}

func writeDeprecatedMetadataForCLI(t *testing.T, root string, item resources.DeprecatedResource) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	payload, err := json.MarshalIndent(map[string]any{
		"resources": []resources.DeprecatedResource{item},
	}, "", "  ")
	if err != nil {
		t.Fatalf("marshal item: %v", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "deprecated-resources.json"), payload, 0o644); err != nil {
		t.Fatalf("write deprecated metadata: %v", err)
	}
}

func writeResourceConfigForCLI(t *testing.T, root, name string, enabled bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	payload := map[string]any{
		"dependencies": map[string]any{
			"resources": map[string]any{
				name: map[string]any{"enabled": enabled},
			},
		},
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal service config: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), data, 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
}

func writeResourceCLIForCLI(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, "resources", name, "cli.sh")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeResourceManifestForCLI(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, "resources", name, "resource.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func projectRootForCLI(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	return root
}

func writeFakeDockerForCLI(t *testing.T) string {
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
  logs)
    echo "fixture logs"
    exit 0
    ;;
esac

exit 0
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write %s: %v", scriptPath, err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_DOCKER_STATE", stateFile)
	return stateFile
}
