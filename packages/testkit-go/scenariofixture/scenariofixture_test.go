package scenariofixture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
)

func TestDefaultDisplayName(t *testing.T) {
	if got := DefaultDisplayName("swarm-manager_api"); got != "Swarm Manager Api" {
		t.Fatalf("DefaultDisplayName() = %q", got)
	}
}

func TestScenarioServiceManifestUsesTypedDefaults(t *testing.T) {
	manifest := ScenarioServiceManifest("alpha", WithDescription("custom"))
	if manifest.Service.Name != "alpha" {
		t.Fatalf("name = %q", manifest.Service.Name)
	}
	if manifest.Service.DisplayName != "Alpha" {
		t.Fatalf("display name = %q", manifest.Service.DisplayName)
	}
	if manifest.Service.Description != "custom" {
		t.Fatalf("description = %q", manifest.Service.Description)
	}
}

func TestWriteScenarioServicePersistsManifestToCanonicalPath(t *testing.T) {
	root := t.TempDir()
	WriteScenarioService(t, root, "alpha", ScenarioServiceManifest("alpha", WithLifecycle(scenario.Lifecycle{Version: "2.0.0"})))
	path := filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json")
	parsed := testkitgo.ReadJSONFile(t, path)
	service := parsed["service"].(map[string]any)
	if service["name"] != "alpha" {
		t.Fatalf("service.name = %v", service["name"])
	}
}

func TestWriteMalformedScenarioServiceWritesInvalidJSONAtCanonicalPath(t *testing.T) {
	root := t.TempDir()
	WriteMalformedScenarioService(t, root, "alpha", `{`)

	data, err := os.ReadFile(filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"))
	if err != nil {
		t.Fatalf("read malformed scenario service: %v", err)
	}
	if string(data) != "{\n" {
		t.Fatalf("malformed scenario service = %q", string(data))
	}
}

func TestScenarioTemplateManifestUsesTypedDefaults(t *testing.T) {
	manifest := ScenarioTemplateManifest("alpha-template")
	if manifest.Name != "alpha-template" {
		t.Fatalf("name = %q", manifest.Name)
	}
	if manifest.DisplayName != "Demo Template" {
		t.Fatalf("display name = %q", manifest.DisplayName)
	}
	if manifest.RequiredVars["SCENARIO_ID"].Flag != "id" {
		t.Fatalf("required SCENARIO_ID flag = %q", manifest.RequiredVars["SCENARIO_ID"].Flag)
	}
	if manifest.OptionalVars["AUTHOR"].Default != "Generator Agent" {
		t.Fatalf("optional AUTHOR default = %q", manifest.OptionalVars["AUTHOR"].Default)
	}
}

func TestWriteScenarioTemplateFixturePersistsCanonicalTemplateFiles(t *testing.T) {
	root := t.TempDir()
	WriteScenarioTemplateFixture(t, root, "alpha-template")

	manifest := testkitgo.ReadJSONFile(t, filepath.Join(root, "alpha-template", "template.json"))
	if manifest["name"] != "alpha-template" {
		t.Fatalf("template name = %v", manifest["name"])
	}

	service := testkitgo.ReadJSONFile(t, filepath.Join(root, "alpha-template", ".vrooli", "service.json"))
	serviceSection := service["service"].(map[string]any)
	if serviceSection["name"] != "{{SCENARIO_ID}}" {
		t.Fatalf("service.name = %v", serviceSection["name"])
	}

	requirements := testkitgo.ReadJSONFile(t, filepath.Join(root, "alpha-template", "requirements", "index.json"))
	if requirements["owner"] != "{{AUTHOR}}" {
		t.Fatalf("requirements.owner = %v", requirements["owner"])
	}
}

func TestWriteScenarioCLIGoModCreatesCanonicalPath(t *testing.T) {
	root := t.TempDir()
	WriteScenarioCLIGoMod(t, root, "alpha", "")

	path := filepath.Join(root, "scenarios", "alpha", "cli", "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read scenario cli go.mod: %v", err)
	}
	if string(data) != "module alpha/cli\n" {
		t.Fatalf("go.mod = %q", string(data))
	}
}
