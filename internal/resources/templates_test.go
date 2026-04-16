package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
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

func TestGenerateResourceTemplateIncludesNativeCLIAndStorageSafeManifest(t *testing.T) {
	controller := NewController(projectRootForResourceTests(t), t.TempDir())
	dest := filepath.Join(t.TempDir(), "demo-db")

	_, err := controller.GenerateResourceTemplate(ResourceTemplateGenerateRequest{
		TemplateName: "docker-service",
		Destination:  dest,
		Values: map[string]string{
			"RESOURCE_NAME":         "demo-db",
			"RESOURCE_DISPLAY_NAME": "Demo DB",
		},
	})
	if err != nil {
		t.Fatalf("GenerateResourceTemplate: %v", err)
	}

	cliMain := readTestFile(t, filepath.Join(dest, "cli", "main.go"))
	if !strings.Contains(cliMain, "cliapp.NewResourceApp") {
		t.Fatalf("cli/main.go missing ResourceApp scaffold: %s", cliMain)
	}
	cliGoMod := readTestFile(t, filepath.Join(dest, "cli", "go.mod"))
	if !strings.Contains(cliGoMod, "module resource-demo-db/cli") {
		t.Fatalf("cli/go.mod missing rendered module path: %s", cliGoMod)
	}
	installSh := readTestFile(t, filepath.Join(dest, "cli", "install.sh"))
	if !strings.Contains(installSh, "resources/demo-db/cli") || !strings.Contains(installSh, "--name \"resource-demo-db\"") {
		t.Fatalf("cli/install.sh missing canonical install wiring: %s", installSh)
	}
	installPS1 := readTestFile(t, filepath.Join(dest, "cli", "install.ps1"))
	if !strings.Contains(installPS1, "resources/demo-db/cli") || !strings.Contains(installPS1, "[string]$Name = \"resource-demo-db\"") {
		t.Fatalf("cli/install.ps1 missing canonical install wiring: %s", installPS1)
	}

	resourceJSON := readTestFile(t, filepath.Join(dest, "resource.json"))
	if !strings.Contains(resourceJSON, `"command": "resource-demo-db"`) {
		t.Fatalf("resource.json missing rendered cli.command: %s", resourceJSON)
	}
	for _, expected := range []string{
		`"install": [`,
		`"run": "bash ./cli/install.sh"`,
		`"run": "powershell -ExecutionPolicy Bypass -File .\\cli\\install.ps1"`,
		`"invoke": {`,
		`"kind": "installed_command"`,
		`"freshness": {`,
		`"inputs": [`,
		`"resource.json"`,
	} {
		if !strings.Contains(resourceJSON, expected) {
			t.Fatalf("resource.json missing %s: %s", expected, resourceJSON)
		}
	}
	if !strings.Contains(resourceJSON, `"source": "${RESOURCE_DATA_DIR}"`) {
		t.Fatalf("resource.json missing canonical storage placeholder: %s", resourceJSON)
	}
	if strings.Contains(resourceJSON, `"source": "./data"`) || strings.Contains(resourceJSON, `"source": "${ROOT}/data"`) {
		t.Fatalf("resource.json still references repo-local data: %s", resourceJSON)
	}
}

func TestGenerateResourceTemplateSupportsExplicitCLICommandOverride(t *testing.T) {
	controller := NewController(projectRootForResourceTests(t), t.TempDir())
	dest := filepath.Join(t.TempDir(), "demo-db")

	_, err := controller.GenerateResourceTemplate(ResourceTemplateGenerateRequest{
		TemplateName: "docker-service",
		Destination:  dest,
		Values: map[string]string{
			"RESOURCE_NAME":         "demo-db",
			"RESOURCE_DISPLAY_NAME": "Demo DB",
			"RESOURCE_CLI_COMMAND":  "dbctl",
		},
	})
	if err != nil {
		t.Fatalf("GenerateResourceTemplate: %v", err)
	}

	resourceJSON := readTestFile(t, filepath.Join(dest, "resource.json"))
	if !strings.Contains(resourceJSON, `"command": "dbctl"`) {
		t.Fatalf("resource.json missing explicit cli command override: %s", resourceJSON)
	}
}

func TestGenerateDockerServiceTemplateIncludesInternalArchitectureScaffold(t *testing.T) {
	controller := NewController(projectRootForResourceTests(t), t.TempDir())
	dest := filepath.Join(t.TempDir(), "demo-db")

	_, err := controller.GenerateResourceTemplate(ResourceTemplateGenerateRequest{
		TemplateName: "docker-service",
		Destination:  dest,
		Values: map[string]string{
			"RESOURCE_NAME":         "demo-db",
			"RESOURCE_DISPLAY_NAME": "Demo DB",
		},
	})
	if err != nil {
		t.Fatalf("GenerateResourceTemplate: %v", err)
	}

	resourceJSON := readTestFile(t, filepath.Join(dest, "resource.json"))
	for _, expected := range []string{
		`"cli/internal/**"`,
		`"docs/**"`,
		`"README.md"`,
	} {
		if !strings.Contains(resourceJSON, expected) {
			t.Fatalf("resource.json missing freshness input %s: %s", expected, resourceJSON)
		}
	}

	readme := readTestFile(t, filepath.Join(dest, "README.md"))
	for _, expected := range []string{
		"`cli/` is the single binary entrypoint and command wiring surface.",
		"`cli/internal/` is the default home for resource-specific Go logic",
		"`cli/internal/install`: install/bootstrap behavior unique to the resource",
		"Keep `cli/main.go` focused on bootstrap and delegation; put resource-specific logic in `cli/internal/...`.",
	} {
		if !strings.Contains(readme, expected) {
			t.Fatalf("README missing architecture guidance %q: %s", expected, readme)
		}
	}

	operations := readTestFile(t, filepath.Join(dest, "docs", "OPERATIONS.md"))
	if !strings.Contains(operations, "Do not turn `cli/main.go` into the primary implementation surface.") {
		t.Fatalf("OPERATIONS.md missing CLI boundary guidance: %s", operations)
	}

	for relPath, expected := range map[string]string{
		filepath.Join("cli", "internal", "install", "install.go"): "// Package install is the default home",
		filepath.Join("cli", "internal", "runtime", "runtime.go"): "// Package runtime is the default home",
		filepath.Join("cli", "internal", "status", "status.go"):   "// Package status is the default home",
		filepath.Join("cli", "internal", "health", "health.go"):   "// Package health is the default home",
		filepath.Join("cli", "internal", "env", "env.go"):         "// Package env is the default home",
	} {
		contents := readTestFile(t, filepath.Join(dest, relPath))
		if !strings.Contains(contents, expected) {
			t.Fatalf("%s missing expected guidance: %s", relPath, contents)
		}
	}
}

func TestGenerateRemainingResourceTemplatesIncludeInternalArchitectureScaffold(t *testing.T) {
	controller := NewController(projectRootForResourceTests(t), t.TempDir())

	cases := []struct {
		templateName      string
		extraValues       map[string]string
		freshnessContains []string
		readmeContains    []string
		docsContains      map[string]string
		internalFiles     map[string]string
	}{
		{
			templateName: "compose-service",
			freshnessContains: []string{
				`"cli/internal/**"`,
				`"docs/**"`,
				`"README.md"`,
				`"compose.yaml"`,
			},
			readmeContains: []string{
				"`cli/internal/` is the default home for compose-specific Go logic",
				"`cli/internal/compose`: compose-specific graph and command helpers",
				"Keep `cli/main.go` focused on bootstrap and delegation; put compose-specific logic in `cli/internal/...`.",
			},
			docsContains: map[string]string{
				filepath.Join("docs", "OPERATIONS.md"): "Do not turn `cli/main.go` into the primary implementation surface.",
			},
			internalFiles: map[string]string{
				filepath.Join("cli", "internal", "compose", "compose.go"):   "// Package compose is the default home",
				filepath.Join("cli", "internal", "topology", "topology.go"): "// Package topology is the default home",
				filepath.Join("cli", "internal", "runtime", "runtime.go"):   "// Package runtime is the default home",
				filepath.Join("cli", "internal", "health", "health.go"):     "// Package health is the default home",
				filepath.Join("cli", "internal", "env", "env.go"):           "// Package env is the default home",
			},
		},
		{
			templateName: "external-cli",
			extraValues: map[string]string{
				"RESOURCE_BINARY": "demo-tool",
			},
			freshnessContains: []string{
				`"cli/internal/**"`,
				`"docs/**"`,
				`"README.md"`,
			},
			readmeContains: []string{
				"`cli/internal/` is the default home for external-tool-specific Go logic",
				"`cli/internal/discovery`: host binary detection and probing helpers",
				"Keep `cli/main.go` focused on bootstrap and delegation; put binary/version/auth logic in `cli/internal/...`.",
			},
			docsContains: map[string]string{
				filepath.Join("docs", "OPERATIONS.md"): "Do not turn `cli/main.go` into the primary implementation surface.",
			},
			internalFiles: map[string]string{
				filepath.Join("cli", "internal", "discovery", "discovery.go"): "// Package discovery is the default home",
				filepath.Join("cli", "internal", "install", "install.go"):     "// Package install is the default home",
				filepath.Join("cli", "internal", "version", "version.go"):     "// Package version is the default home",
				filepath.Join("cli", "internal", "env", "env.go"):             "// Package env is the default home",
				filepath.Join("cli", "internal", "auth", "auth.go"):           "// Package auth is the default home",
			},
		},
		{
			templateName: "cloud-api",
			extraValues: map[string]string{
				"RESOURCE_ENDPOINT":       "https://api.example.com/health",
				"RESOURCE_CREDENTIAL_ENV": "DEMO_API_KEY",
			},
			freshnessContains: []string{
				`"cli/internal/**"`,
				`"docs/**"`,
				`"README.md"`,
			},
			readmeContains: []string{
				"`cli/internal/` is the default home for provider-specific Go logic",
				"`cli/internal/config`: endpoint and provider configuration helpers",
				"Keep `cli/main.go` focused on bootstrap and delegation; put provider-specific config/auth/health logic in `cli/internal/...`.",
			},
			docsContains: map[string]string{
				filepath.Join("docs", "OPERATIONS.md"):  "Do not turn `cli/main.go` into the primary implementation surface.",
				filepath.Join("docs", "CREDENTIALS.md"): "Keep `resource.json` as the declarative credential contract",
			},
			internalFiles: map[string]string{
				filepath.Join("cli", "internal", "config", "config.go"): "// Package config is the default home",
				filepath.Join("cli", "internal", "auth", "auth.go"):     "// Package auth is the default home",
				filepath.Join("cli", "internal", "health", "health.go"): "// Package health is the default home",
				filepath.Join("cli", "internal", "env", "env.go"):       "// Package env is the default home",
			},
		},
		{
			templateName: "desktop-app",
			freshnessContains: []string{
				`"cli/internal/**"`,
				`"docs/**"`,
				`"README.md"`,
			},
			readmeContains: []string{
				"`cli/internal/` is the default home for desktop-app-specific Go logic",
				"`cli/internal/discovery`: host-path and application detection helpers",
				"Keep `cli/main.go` focused on bootstrap and delegation; put platform/detection logic in `cli/internal/...`.",
			},
			docsContains: map[string]string{
				filepath.Join("docs", "OPERATIONS.md"):   "Do not turn `cli/main.go` into the primary implementation surface.",
				filepath.Join("docs", "MANUAL-STEPS.md"): "Keep the operator workflow here instead of hiding it in weak automation.",
			},
			internalFiles: map[string]string{
				filepath.Join("cli", "internal", "discovery", "discovery.go"): "// Package discovery is the default home",
				filepath.Join("cli", "internal", "install", "install.go"):     "// Package install is the default home",
				filepath.Join("cli", "internal", "platform", "platform.go"):   "// Package platform is the default home",
				filepath.Join("cli", "internal", "health", "health.go"):       "// Package health is the default home",
			},
		},
		{
			templateName: "manual-resource",
			freshnessContains: []string{
				`"cli/internal/**"`,
				`"docs/**"`,
				`"README.md"`,
			},
			readmeContains: []string{
				"`cli/internal/` is optional and intentionally small;",
				"`cli/internal/validate`: validation helpers for documented manual setup",
				"Keep `cli/main.go` focused on bootstrap and delegation; keep any real validation logic under `cli/internal/...`.",
			},
			docsContains: map[string]string{
				filepath.Join("docs", "OPERATIONS.md"):      "Do not turn `cli/main.go` into the primary implementation surface.",
				filepath.Join("docs", "SETUP-CHECKLIST.md"): "Keep this checklist as the primary setup contract.",
			},
			internalFiles: map[string]string{
				filepath.Join("cli", "internal", "validate", "validate.go"): "// Package validate is the default home",
				filepath.Join("cli", "internal", "env", "env.go"):           "// Package env is the default home",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.templateName, func(t *testing.T) {
			dest := filepath.Join(t.TempDir(), tc.templateName)
			values := map[string]string{
				"RESOURCE_NAME":         tc.templateName + "-fixture",
				"RESOURCE_DISPLAY_NAME": kebabToDisplayName(tc.templateName + "-fixture"),
			}
			for key, value := range tc.extraValues {
				values[key] = value
			}

			_, err := controller.GenerateResourceTemplate(ResourceTemplateGenerateRequest{
				TemplateName: tc.templateName,
				Destination:  dest,
				Values:       values,
			})
			if err != nil {
				t.Fatalf("GenerateResourceTemplate(%s): %v", tc.templateName, err)
			}

			resourceJSON := readTestFile(t, filepath.Join(dest, "resource.json"))
			for _, expected := range tc.freshnessContains {
				if !strings.Contains(resourceJSON, expected) {
					t.Fatalf("resource.json missing %s: %s", expected, resourceJSON)
				}
			}

			readme := readTestFile(t, filepath.Join(dest, "README.md"))
			for _, expected := range tc.readmeContains {
				if !strings.Contains(readme, expected) {
					t.Fatalf("README missing %q: %s", expected, readme)
				}
			}

			for relPath, expected := range tc.docsContains {
				contents := readTestFile(t, filepath.Join(dest, relPath))
				if !strings.Contains(contents, expected) {
					t.Fatalf("%s missing %q: %s", relPath, expected, contents)
				}
			}

			for relPath, expected := range tc.internalFiles {
				contents := readTestFile(t, filepath.Join(dest, relPath))
				if !strings.Contains(contents, expected) {
					t.Fatalf("%s missing expected guidance: %s", relPath, contents)
				}
			}
		})
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
	templateDir := filepath.Join("templates", "resources", "docker-service")
	testresource.WriteResourceTemplateManifest(t, root, "docker-service", testresource.ResourceTemplate(
		"docker-service",
		testresource.WithTemplateDisplayName("Docker Service"),
		testresource.WithTemplateDescription("Fixture template"),
		testresource.WithTemplateDriver("docker-service"),
		testresource.WithTemplateRequiredVars(map[string]testresource.ResourceTemplateVar{
			"RESOURCE_NAME": {Flag: "name", Description: "Fixture"},
		}),
		testresource.WithTemplateDocs(map[string]string{
			"phase3-plan": "docs/plans/resource-cross-platform-migration-plan.md",
		}),
	))
	testkitgo.WriteRelativeFile(t, root, filepath.Join(templateDir, "README.md"), "# Fixture\n")
	testkitgo.WriteRelativeFile(t, root, filepath.Join(templateDir, "cli", "go.mod"), "module example.com/resource/cli\n\ngo 1.22\n")
	testkitgo.WriteRelativeFile(t, root, filepath.Join(templateDir, "cli", "install.sh"), "#!/usr/bin/env bash\n")
	testkitgo.WriteRelativeFile(t, root, filepath.Join(templateDir, "cli", "install.ps1"), "Write-Host 'install'\n")
	testkitgo.WriteRelativeFile(t, root, filepath.Join(templateDir, "cli", "main.go"), "package main\n")
	testkitgo.WriteJSON(t, filepath.Join(root, filepath.FromSlash(templateDir), "resource.json"), map[string]any{})
	testkitgo.WriteJSON(t, filepath.Join(root, filepath.FromSlash(templateDir), "test", "smoke.json"), map[string]any{})
	testkitgo.WriteRelativeFile(t, root, filepath.Join("docs", "plans", "resource-cross-platform-migration-plan.md"), "# Plan\n")

	_, err := NewController(root, home).ValidateResourceTemplates()
	if err == nil || !strings.Contains(err.Error(), "missing required file test/integration.json") {
		t.Fatalf("expected missing file validation error, got %v", err)
	}
}

func TestValidateResourceTemplatesRejectsMissingDocReferences(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	templateDir := filepath.Join("templates", "resources", "docker-service")
	testresource.WriteResourceTemplateManifest(t, root, "docker-service", testresource.ResourceTemplate(
		"docker-service",
		testresource.WithTemplateDisplayName("Docker Service"),
		testresource.WithTemplateDescription("Fixture template"),
		testresource.WithTemplateDriver("docker-service"),
		testresource.WithTemplateRequiredVars(map[string]testresource.ResourceTemplateVar{
			"RESOURCE_NAME": {Flag: "name", Description: "Fixture"},
		}),
		testresource.WithTemplateDocs(map[string]string{
			"phase3-plan": "docs/plans/missing.md",
		}),
	))
	testkitgo.WriteRelativeFile(t, root, filepath.Join(templateDir, "README.md"), "# Fixture\n")
	testkitgo.WriteRelativeFile(t, root, filepath.Join(templateDir, "cli", "go.mod"), "module example.com/resource/cli\n\ngo 1.22\n")
	testkitgo.WriteRelativeFile(t, root, filepath.Join(templateDir, "cli", "install.sh"), "#!/usr/bin/env bash\n")
	testkitgo.WriteRelativeFile(t, root, filepath.Join(templateDir, "cli", "install.ps1"), "Write-Host 'install'\n")
	testkitgo.WriteRelativeFile(t, root, filepath.Join(templateDir, "cli", "main.go"), "package main\n")
	testkitgo.WriteJSON(t, filepath.Join(root, filepath.FromSlash(templateDir), "resource.json"), map[string]any{})
	testkitgo.WriteJSON(t, filepath.Join(root, filepath.FromSlash(templateDir), "test", "smoke.json"), map[string]any{})
	testkitgo.WriteJSON(t, filepath.Join(root, filepath.FromSlash(templateDir), "test", "integration.json"), map[string]any{})
	testkitgo.WriteRelativeFile(t, root, filepath.Join(templateDir, "docs", "OPERATIONS.md"), "# Operations\n")

	_, err := NewController(root, home).ValidateResourceTemplates()
	if err == nil || !strings.Contains(err.Error(), "docs entry") {
		t.Fatalf("expected missing docs validation error, got %v", err)
	}
}

func TestPopulateResourceTemplatePathValuesAndCopyRenderGoModPaths(t *testing.T) {
	controller := NewController(projectRootForResourceTests(t), t.TempDir())
	templateDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(templateDir, "cli"), 0o755); err != nil {
		t.Fatalf("mkdir cli: %v", err)
	}
	goMod := `module example.com/resource/cli

go 1.22

replace github.com/vrooli/cli-core => {{PACKAGES_REL_FROM_CLI}}/cli-core
replace github.com/vrooli/repo-contract-go => {{PACKAGES_REL_FROM_CLI}}/repo-contract-go
replace github.com/vrooli/vrooli => {{REPO_ROOT_REL_FROM_CLI}}
`
	testkitgo.WriteRelativeFile(t, templateDir, filepath.Join("cli", "go.mod"), goMod)
	testkitgo.WriteRelativeFile(t, templateDir, filepath.Join("cli", "main.go"), "package main\nfunc main() {}\n")

	destination := filepath.Join(t.TempDir(), "resources", "demo")
	values := map[string]string{}
	if err := controller.populateResourceTemplatePathValues(destination, values); err != nil {
		t.Fatalf("populateResourceTemplatePathValues: %v", err)
	}
	if err := copyResourceTemplate(templateDir, destination, values); err != nil {
		t.Fatalf("copyResourceTemplate: %v", err)
	}

	rendered := readTestFile(t, filepath.Join(destination, "cli", "go.mod"))
	wantPackagesRel, err := filepath.Rel(filepath.Join(destination, "cli"), filepath.Join(controller.Root, "packages"))
	if err != nil {
		t.Fatalf("filepath.Rel(packages): %v", err)
	}
	if !strings.Contains(rendered, filepath.ToSlash(filepath.Join(wantPackagesRel, "cli-core"))) {
		t.Fatalf("rendered go.mod missing cli-core replace path: %s", rendered)
	}
	if err := verifyGeneratedResourceGoModules(destination); err != nil {
		t.Fatalf("verifyGeneratedResourceGoModules: %v", err)
	}
}

func TestValidateResourceTemplateGoModuleSourceRejectsHardcodedLocalReplace(t *testing.T) {
	templateDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(templateDir, "cli"), 0o755); err != nil {
		t.Fatalf("mkdir cli: %v", err)
	}
	testkitgo.WriteRelativeFile(t, templateDir, filepath.Join("cli", "go.mod"), "module example.com/demo\n\nreplace github.com/vrooli/cli-core => ../../../packages/cli-core\n")

	err := validateResourceTemplateGoModuleSource(ResourceTemplateInfo{Name: "docker-service", Path: templateDir})
	if err == nil || !strings.Contains(err.Error(), "hardcoded local replace target") {
		t.Fatalf("expected hardcoded replace validation error, got %v", err)
	}
}

func TestVerifyGeneratedResourceGoModulesRejectsBrokenReplaceTarget(t *testing.T) {
	destination := t.TempDir()
	moduleDir := filepath.Join(destination, "cli")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir cli: %v", err)
	}
	testkitgo.WriteRelativeFile(t, destination, filepath.Join("cli", "go.mod"), "module example.com/demo\n\ngo 1.22\n\nreplace github.com/vrooli/cli-core => ../../../packages/cli-core\n")
	testkitgo.WriteRelativeFile(t, destination, filepath.Join("cli", "main.go"), "package main\nfunc main() {}\n")

	err := verifyGeneratedResourceGoModules(destination)
	if err == nil || !strings.Contains(err.Error(), "does not resolve") {
		t.Fatalf("expected broken replace validation error, got %v", err)
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
		"cli/go.mod",
		"cli/install.sh",
		"cli/install.ps1",
		"cli/main.go",
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

func projectRootForResourceTests(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	return root
}
