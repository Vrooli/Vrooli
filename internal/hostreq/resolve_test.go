package hostreq

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveMergesRootScenarioAndResourceDeclarations(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	writeFile(t, filepath.Join(root, ".vrooli", "service.json"), `{
  "service": {"name": "vrooli"},
  "dependencies": {"resources": {"alpha-resource": {"enabled": true}}},
  "hostTools": [
    {"name": "docker", "required": true, "reason": "root docker", "when": ["setup", "develop"], "environments": ["development"]},
    {"name": "node", "required": true, "reason": "root node", "environments": ["development"]}
  ],
  "hostSafeguards": [
    {"name": "remote_session_protection", "required": false, "reason": "root safeguard", "platforms": ["linux"]}
  ]
}`)
	writeFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{
  "service": {"name": "alpha"},
  "hostTools": [
    {"name": "node", "required": false, "reason": "scenario node", "when": ["develop"], "notes": "scenario note"},
    {"name": "ffmpeg", "required": true, "reason": "scenario ffmpeg", "platforms": ["linux"]}
  ]
}`)
	writeFile(t, filepath.Join(root, "resources", "alpha-resource", "resource.json"), `{
  "name": "alpha-resource",
  "driver": "external-cli",
  "binary": "alpha",
  "portability_tier": "full",
  "hostTools": [
    {"name": "docker", "required": false, "reason": "resource docker", "when": ["develop"]},
    {"name": "sqlite", "required": true, "reason": "resource sqlite", "manual": true}
  ]
}`)

	resolution, err := Resolve(root, home, ResolveOptions{
		Environment: "development",
		When:        "develop",
		Resources:   "enabled",
		Scenarios:   "all",
		Platform:    "linux",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(resolution.Tools) != 4 {
		t.Fatalf("tool count = %d, want 4", len(resolution.Tools))
	}

	docker := findRequirement(t, resolution.Tools, "docker")
	if !docker.Required {
		t.Fatal("docker should remain required after merge")
	}
	if len(docker.Provenance) != 2 {
		t.Fatalf("docker provenance count = %d, want 2", len(docker.Provenance))
	}
	if got := strings.Join(docker.Reasons, ","); got != "resource docker,root docker" {
		t.Fatalf("docker reasons = %q", got)
	}

	node := findRequirement(t, resolution.Tools, "node")
	if got := strings.Join(node.When, ","); got != "develop" {
		t.Fatalf("node when = %q", got)
	}
	if got := strings.Join(node.Notes, ","); got != "scenario note" {
		t.Fatalf("node notes = %q", got)
	}

	sqlite := findRequirement(t, resolution.Tools, "sqlite")
	if !sqlite.Manual {
		t.Fatal("sqlite manual should be true")
	}

	safeguard := findRequirement(t, resolution.Safeguards, "remote_session_protection")
	if safeguard.Kind != KindSafeguard {
		t.Fatalf("safeguard kind = %q", safeguard.Kind)
	}
}

func TestResolveHonorsSelectorsAndFilters(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	writeFile(t, filepath.Join(root, ".vrooli", "service.json"), `{
  "service": {"name": "vrooli"},
  "hostTools": [
    {"name": "docker", "required": true, "reason": "root docker", "environments": ["development", "production"]},
    {"name": "python", "required": true, "reason": "root python", "environments": ["development"]},
    {"name": "openbox", "required": true, "reason": "linux only", "platforms": ["linux"]}
  ]
}`)
	writeFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{
  "service": {"name": "alpha"},
  "hostTools": [{"name": "ffmpeg", "required": true, "reason": "alpha ffmpeg"}]
}`)
	writeFile(t, filepath.Join(root, "scenarios", "beta", ".vrooli", "service.json"), `{
  "service": {"name": "beta"},
  "hostTools": [{"name": "buf", "required": true, "reason": "beta buf"}]
}`)
	writeFile(t, filepath.Join(root, "resources", "alpha-resource", "resource.json"), `{
  "name": "alpha-resource",
  "driver": "external-cli",
  "binary": "alpha",
  "portability_tier": "full",
  "hostTools": [{"name": "sqlite", "required": true, "reason": "alpha sqlite"}]
}`)
	writeFile(t, filepath.Join(root, "resources", "beta-resource", "resource.json"), `{
  "name": "beta-resource",
  "driver": "external-cli",
  "binary": "beta",
  "portability_tier": "full",
  "hostTools": [{"name": "helm", "required": true, "reason": "beta helm"}]
}`)

	resolution, err := Resolve(root, home, ResolveOptions{
		Environment: "production",
		Resources:   "beta-resource",
		Scenarios:   "beta",
		Platform:    "macos",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if findOptionalRequirement(resolution.Tools, "python") != nil {
		t.Fatal("python should be filtered out for production")
	}
	if findOptionalRequirement(resolution.Tools, "openbox") != nil {
		t.Fatal("openbox should be filtered out for macos")
	}
	if findOptionalRequirement(resolution.Tools, "ffmpeg") != nil {
		t.Fatal("alpha scenario should not be included")
	}
	if findOptionalRequirement(resolution.Tools, "sqlite") != nil {
		t.Fatal("alpha resource should not be included")
	}
	if findOptionalRequirement(resolution.Tools, "helm") == nil {
		t.Fatal("beta resource should be included")
	}
	if findOptionalRequirement(resolution.Tools, "buf") == nil {
		t.Fatal("beta scenario should be included")
	}
}

func TestResolveRejectsUnknownExplicitSelections(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeFile(t, filepath.Join(root, ".vrooli", "service.json"), `{"service":{"name":"vrooli"}}`)

	if _, err := Resolve(root, home, ResolveOptions{Resources: "missing"}); err == nil || !strings.Contains(err.Error(), `resource "missing" not found`) {
		t.Fatalf("resource error = %v", err)
	}
	if _, err := Resolve(root, home, ResolveOptions{Scenarios: "missing"}); err == nil || !strings.Contains(err.Error(), `load scenario "missing"`) {
		t.Fatalf("scenario error = %v", err)
	}
}

func TestSchemaFilesDeclareHostRequirementProperties(t *testing.T) {
	root := projectRootForHostreqTest(t)

	serviceSchema := readJSONFile(t, filepath.Join(root, ".vrooli", "schemas", "service.schema.json"))
	resourceSchema := readJSONFile(t, filepath.Join(root, ".vrooli", "schemas", "resource.schema.json"))

	assertSchemaHasHostRequirements(t, serviceSchema)
	assertSchemaHasHostRequirements(t, resourceSchema)
}

func assertSchemaHasHostRequirements(t *testing.T, schema map[string]any) {
	t.Helper()
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema properties missing")
	}
	for _, field := range []string{"hostTools", "hostSafeguards"} {
		value, ok := properties[field].(map[string]any)
		if !ok {
			t.Fatalf("%s property missing", field)
		}
		if value["type"] != "array" {
			t.Fatalf("%s type = %v", field, value["type"])
		}
	}
	definitions, ok := schema["definitions"].(map[string]any)
	if !ok {
		t.Fatal("schema definitions missing")
	}
	hostRequirement, ok := definitions["hostRequirement"].(map[string]any)
	if !ok {
		t.Fatal("hostRequirement definition missing")
	}
	required, ok := hostRequirement["required"].([]any)
	if !ok {
		t.Fatal("hostRequirement required missing")
	}
	got := make([]string, 0, len(required))
	for _, value := range required {
		got = append(got, value.(string))
	}
	if strings.Join(got, ",") != "name,required,reason" {
		t.Fatalf("hostRequirement required = %q", strings.Join(got, ","))
	}
}

func findRequirement(t *testing.T, items []ResolvedRequirement, name string) ResolvedRequirement {
	t.Helper()
	item := findOptionalRequirement(items, name)
	if item == nil {
		t.Fatalf("requirement %q not found", name)
	}
	return *item
}

func findOptionalRequirement(items []ResolvedRequirement, name string) *ResolvedRequirement {
	for i := range items {
		if items[i].Name == name {
			return &items[i]
		}
	}
	return nil
}

func projectRootForHostreqTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal(%s): %v", path, err)
	}
	return parsed
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}
