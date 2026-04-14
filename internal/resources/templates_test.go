package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestValidateResourceTemplates(t *testing.T) {
	controller := NewController(projectRootForResourceTests(t), t.TempDir())

	report, err := controller.ValidateResourceTemplates()
	if err != nil {
		t.Fatalf("ValidateResourceTemplates: %v", err)
	}
	if report.Count < len(AllowedSuggestedTemplates()) {
		t.Fatalf("ValidateResourceTemplates count = %d, want at least %d", report.Count, len(AllowedSuggestedTemplates()))
	}
}

func TestGenerateResourceTemplateDryRun(t *testing.T) {
	controller := NewController(projectRootForResourceTests(t), t.TempDir())
	dest := filepath.Join(t.TempDir(), "demo-db")

	report, err := controller.GenerateResourceTemplate(ResourceTemplateGenerateRequest{
		TemplateName: "docker-service",
		Destination:  dest,
		DryRun:       true,
		Values: map[string]string{
			"RESOURCE_NAME":         "demo-db",
			"RESOURCE_DISPLAY_NAME": "Demo DB",
		},
	})
	if err != nil {
		t.Fatalf("GenerateResourceTemplate dry-run: %v", err)
	}
	if !report.DryRun {
		t.Fatalf("report.DryRun = false, want true")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("destination should not exist after dry-run, err=%v", err)
	}
	if len(report.Files) == 0 {
		t.Fatal("expected generated file preview")
	}
}

func TestGenerateResourceTemplateAllCanonicalTemplates(t *testing.T) {
	controller := NewController(projectRootForResourceTests(t), t.TempDir())

	for _, templateName := range AllowedSuggestedTemplates() {
		templateName := templateName
		t.Run(templateName, func(t *testing.T) {
			dest := filepath.Join(t.TempDir(), templateName)
			report, err := controller.GenerateResourceTemplate(ResourceTemplateGenerateRequest{
				TemplateName: templateName,
				Destination:  dest,
				Values: map[string]string{
					"RESOURCE_NAME":         templateName + "-fixture",
					"RESOURCE_DISPLAY_NAME": kebabToDisplayName(templateName + "-fixture"),
				},
			})
			if err != nil {
				t.Fatalf("GenerateResourceTemplate(%s): %v", templateName, err)
			}
			if report.Template.Name != templateName {
				t.Fatalf("template = %s, want %s", report.Template.Name, templateName)
			}
			assertGeneratedTemplateLayout(t, dest)
		})
	}
}

func TestGenerateResourceTemplateFromBlueprint(t *testing.T) {
	controller := NewController(projectRootForResourceTests(t), t.TempDir())
	dest := filepath.Join(t.TempDir(), "terraform")

	report, err := controller.GenerateResourceTemplate(ResourceTemplateGenerateRequest{
		BlueprintName: "terraform",
		Destination:   dest,
	})
	if err != nil {
		t.Fatalf("GenerateResourceTemplate from blueprint: %v", err)
	}
	if report.Template.Name != "external-cli" {
		t.Fatalf("template = %s, want external-cli", report.Template.Name)
	}
	if report.BlueprintName != "terraform" {
		t.Fatalf("blueprint = %s, want terraform", report.BlueprintName)
	}

	assertJSONFile(t, filepath.Join(dest, "resource.json"))
	assertJSONFile(t, filepath.Join(dest, "test", "smoke.json"))
	assertJSONFile(t, filepath.Join(dest, "test", "integration.json"))

	readme := readTestFile(t, filepath.Join(dest, "README.md"))
	if strings.Contains(readme, "{{") {
		t.Fatalf("README still contains placeholders: %s", readme)
	}
	resourceJSON := readTestFile(t, filepath.Join(dest, "resource.json"))
	if !strings.Contains(resourceJSON, `"name": "terraform"`) {
		t.Fatalf("resource.json missing blueprint-seeded name: %s", resourceJSON)
	}
}

func TestGenerateResourceTemplateFromBlueprintRepresentativeArchetypes(t *testing.T) {
	controller := NewController(projectRootForResourceTests(t), t.TempDir())

	blueprints, err := controller.ListBlueprints()
	if err != nil {
		t.Fatalf("ListBlueprints: %v", err)
	}

	representatives := make(map[string]string)
	for _, item := range blueprints {
		if _, ok := representatives[item.SuggestedTemplate]; ok {
			continue
		}
		representatives[item.SuggestedTemplate] = item.Name
	}

	templates := make([]string, 0, len(representatives))
	for templateName := range representatives {
		templates = append(templates, templateName)
	}
	sort.Strings(templates)

	for _, templateName := range templates {
		blueprintName := representatives[templateName]
		t.Run(templateName, func(t *testing.T) {
			dest := filepath.Join(t.TempDir(), blueprintName)
			report, err := controller.GenerateResourceTemplate(ResourceTemplateGenerateRequest{
				BlueprintName: blueprintName,
				Destination:   dest,
			})
			if err != nil {
				t.Fatalf("GenerateResourceTemplate(%s from %s): %v", templateName, blueprintName, err)
			}
			if report.Template.Name != templateName {
				t.Fatalf("template = %s, want %s", report.Template.Name, templateName)
			}
			assertGeneratedTemplateLayout(t, dest)
		})
	}
}

func TestGenerateResourceTemplateRejectsMissingRequiredValues(t *testing.T) {
	controller := NewController(projectRootForResourceTests(t), t.TempDir())

	_, err := controller.GenerateResourceTemplate(ResourceTemplateGenerateRequest{
		TemplateName: "docker-service",
		Destination:  filepath.Join(t.TempDir(), "missing"),
	})
	if err == nil || !strings.Contains(err.Error(), "--name") {
		t.Fatalf("expected required value error, got %v", err)
	}
}

func TestGenerateResourceTemplateRejectsBlueprintTemplateMismatch(t *testing.T) {
	controller := NewController(projectRootForResourceTests(t), t.TempDir())

	_, err := controller.GenerateResourceTemplate(ResourceTemplateGenerateRequest{
		TemplateName:  "docker-service",
		BlueprintName: "terraform",
		Destination:   filepath.Join(t.TempDir(), "mismatch"),
		Values: map[string]string{
			"RESOURCE_NAME": "mismatch",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "does not match blueprint") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}

func TestGenerateResourceTemplateRequiresForceToOverwrite(t *testing.T) {
	controller := NewController(projectRootForResourceTests(t), t.TempDir())
	dest := filepath.Join(t.TempDir(), "existing")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dest, err)
	}
	if err := os.WriteFile(filepath.Join(dest, "stale.txt"), []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	_, err := controller.GenerateResourceTemplate(ResourceTemplateGenerateRequest{
		TemplateName: "docker-service",
		Destination:  dest,
		Values: map[string]string{
			"RESOURCE_NAME":         "existing",
			"RESOURCE_DISPLAY_NAME": "Existing",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "destination already exists") {
		t.Fatalf("expected destination exists error, got %v", err)
	}

	_, err = controller.GenerateResourceTemplate(ResourceTemplateGenerateRequest{
		TemplateName: "docker-service",
		Destination:  dest,
		Force:        true,
		Values: map[string]string{
			"RESOURCE_NAME":         "existing",
			"RESOURCE_DISPLAY_NAME": "Existing",
		},
	})
	if err != nil {
		t.Fatalf("GenerateResourceTemplate with force: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "stale.txt")); !os.IsNotExist(err) {
		t.Fatalf("stale file should be removed, err=%v", err)
	}
	assertGeneratedTemplateLayout(t, dest)
}

func TestValidateResourceTemplatesRejectsMissingRequiredFiles(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	templateDir := filepath.Join(root, "templates", "resources", "docker-service")
	writeTestFile(t, filepath.Join(templateDir, "template.json"), `{
  "name": "docker-service",
  "displayName": "Docker Service",
  "description": "Fixture template",
  "driver": "docker-service",
  "requiredVars": {
    "RESOURCE_NAME": {"flag": "name", "description": "Fixture"}
  },
  "docs": {
    "phase3-plan": "docs/plans/resource-cross-platform-migration-plan.md"
  }
}`)
	writeTestFile(t, filepath.Join(templateDir, "README.md"), "# Fixture\n")
	writeTestFile(t, filepath.Join(templateDir, "resource.json"), "{}\n")
	writeTestFile(t, filepath.Join(templateDir, "test", "smoke.json"), "{}\n")
	writeTestFile(t, filepath.Join(root, "docs", "plans", "resource-cross-platform-migration-plan.md"), "# Plan\n")

	_, err := NewController(root, home).ValidateResourceTemplates()
	if err == nil || !strings.Contains(err.Error(), "missing required file test/integration.json") {
		t.Fatalf("expected missing file validation error, got %v", err)
	}
}

func TestValidateResourceTemplatesRejectsMissingDocReferences(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	templateDir := filepath.Join(root, "templates", "resources", "docker-service")
	writeTestFile(t, filepath.Join(templateDir, "template.json"), `{
  "name": "docker-service",
  "displayName": "Docker Service",
  "description": "Fixture template",
  "driver": "docker-service",
  "requiredVars": {
    "RESOURCE_NAME": {"flag": "name", "description": "Fixture"}
  },
  "docs": {
    "phase3-plan": "docs/plans/missing.md"
  }
}`)
	writeTestFile(t, filepath.Join(templateDir, "README.md"), "# Fixture\n")
	writeTestFile(t, filepath.Join(templateDir, "resource.json"), "{}\n")
	writeTestFile(t, filepath.Join(templateDir, "test", "smoke.json"), "{}\n")
	writeTestFile(t, filepath.Join(templateDir, "test", "integration.json"), "{}\n")
	writeTestFile(t, filepath.Join(templateDir, "docs", "OPERATIONS.md"), "# Operations\n")

	_, err := NewController(root, home).ValidateResourceTemplates()
	if err == nil || !strings.Contains(err.Error(), "docs entry") {
		t.Fatalf("expected missing docs validation error, got %v", err)
	}
}

func assertJSONFile(t *testing.T, path string) {
	t.Helper()
	data := readTestFile(t, path)
	var payload any
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		t.Fatalf("json unmarshal %s: %v", path, err)
	}
	if strings.Contains(data, "{{") {
		t.Fatalf("file %s still contains placeholders", path)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func assertGeneratedTemplateLayout(t *testing.T, dest string) {
	t.Helper()
	requiredFiles := []string{
		"README.md",
		"resource.json",
		"test/smoke.json",
		"test/integration.json",
		"docs/OPERATIONS.md",
	}
	for _, relPath := range requiredFiles {
		path := filepath.Join(dest, relPath)
		if strings.HasSuffix(relPath, ".json") {
			assertJSONFile(t, path)
			continue
		}
		if contents := readTestFile(t, path); strings.Contains(contents, "{{") {
			t.Fatalf("file %s still contains placeholders", path)
		}
	}
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func projectRootForResourceTests(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	return root
}
