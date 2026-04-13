package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testkitvrooli "github.com/vrooli/vrooli/packages/testkit-go/vrooli"
)

func TestShowPackageHelpIncludesRefresh(t *testing.T) {
	var stdout bytes.Buffer
	showPackageHelp(&stdout)
	if !strings.Contains(stdout.String(), "refresh") {
		t.Fatalf("help = %q", stdout.String())
	}
}

func TestPackageListCommand(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "alpha", ".vrooli", "package.json"), `{
  "$schema": "schemas/package.schema.json",
  "version": "1.0.0",
  "package": {
    "name": "alpha",
    "display_name": "@vrooli/alpha",
    "kind": "js_runtime",
    "module_identifiers": ["@vrooli/alpha"],
    "adoption": {
      "scenario_adoptable": true,
      "allowed_consumers": ["scenario_ui"],
      "adoption_modes": ["file_dependency"]
    },
    "lifecycle": {},
    "refresh": {
      "strategy": "scenario_setup",
      "restart_running_consumers": true
    }
  }
}`)

	app, ctx := newConfiguredCommandContext(fixture.Root, globalOptions{}, &bytes.Buffer{}, &bytes.Buffer{})
	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := runPackageRootCommand(app, ctx, []string{"list"}); err != nil {
		t.Fatalf("runPackageRootCommand(list): %v", err)
	}
	if !strings.Contains(stdout.String(), "alpha") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestPackageInfoCommand(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "alpha", ".vrooli", "package.json"), `{
  "$schema": "schemas/package.schema.json",
  "version": "1.0.0",
  "package": {
    "name": "alpha",
    "display_name": "@vrooli/alpha",
    "kind": "js_runtime",
    "module_identifiers": ["@vrooli/alpha"],
    "adoption": {
      "scenario_adoptable": true,
      "allowed_consumers": ["scenario_ui"],
      "adoption_modes": ["file_dependency"]
    },
    "lifecycle": {},
    "refresh": {
      "strategy": "scenario_setup",
      "restart_running_consumers": true
    }
  }
}`)

	app, ctx := newConfiguredCommandContext(fixture.Root, globalOptions{}, &bytes.Buffer{}, &bytes.Buffer{})
	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := runPackageRootCommand(app, ctx, []string{"info", "alpha"}); err != nil {
		t.Fatalf("runPackageRootCommand(info): %v", err)
	}
	if !strings.Contains(stdout.String(), "name: alpha") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestPackageDependentsCommand(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writePackageManifestFixture(t, fixture.Root, "alpha", `{
  "$schema": "schemas/package.schema.json",
  "version": "1.0.0",
  "package": {
    "name": "alpha",
    "display_name": "@vrooli/alpha",
    "kind": "js_runtime",
    "module_identifiers": ["@vrooli/alpha"],
    "adoption": {
      "scenario_adoptable": true,
      "allowed_consumers": ["scenario_ui"],
      "adoption_modes": ["file_dependency"]
    },
    "lifecycle": {},
    "refresh": {
      "strategy": "scenario_setup",
      "restart_running_consumers": true
    }
  }
}`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", ".vrooli", "service.json"), `{"service":{"name":"demo","parent":"vrooli"},"lifecycle":{"version":"2.0.0"}}`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "ui", "package.json"), `{
  "dependencies": {
    "@vrooli/alpha": "file:../../../packages/alpha"
  }
}`)

	app, ctx := newConfiguredCommandContext(fixture.Root, globalOptions{}, &bytes.Buffer{}, &bytes.Buffer{})
	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := runPackageRootCommand(app, ctx, []string{"dependents", "alpha"}); err != nil {
		t.Fatalf("runPackageRootCommand(dependents): %v", err)
	}
	if !strings.Contains(stdout.String(), "demo") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestPackageValidateAndAuditCommands(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writePackageManifestFixture(t, fixture.Root, "alpha", `{
  "$schema": "schemas/package.schema.json",
  "version": "1.0.0",
  "package": {
    "name": "alpha",
    "display_name": "@vrooli/alpha",
    "kind": "js_runtime",
    "module_identifiers": ["@vrooli/alpha"],
    "adoption": {
      "scenario_adoptable": true,
      "allowed_consumers": ["scenario_ui"],
      "adoption_modes": ["file_dependency"]
    },
    "lifecycle": {},
    "refresh": {
      "strategy": "scenario_setup",
      "restart_running_consumers": true
    },
    "docs": ["docs/package-governance.md"]
  }
}`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "docs", "package-governance.md"), "# ok\n")

	app, ctx := newConfiguredCommandContext(fixture.Root, globalOptions{}, &bytes.Buffer{}, &bytes.Buffer{})

	var validateOut bytes.Buffer
	ctx.Stdout = &validateOut
	if err := runPackageRootCommand(app, ctx, []string{"validate", "--all"}); err != nil {
		t.Fatalf("runPackageRootCommand(validate): %v", err)
	}
	if !strings.Contains(validateOut.String(), "package governance validation passed") {
		t.Fatalf("validate stdout = %q", validateOut.String())
	}

	var auditOut bytes.Buffer
	ctx.Stdout = &auditOut
	if err := runPackageRootCommand(app, ctx, []string{"audit", "--all"}); err != nil {
		t.Fatalf("runPackageRootCommand(audit): %v", err)
	}
	if !strings.Contains(auditOut.String(), "package governance audit passed") {
		t.Fatalf("audit stdout = %q", auditOut.String())
	}
}

func TestPackageRefreshScenarioSetupRunsBuildAndSetup(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeScenarioPortRegistryFixture(t, fixture.Root)
	writePackageManifestFixture(t, fixture.Root, "alpha", `{
  "$schema": "schemas/package.schema.json",
  "version": "1.0.0",
  "package": {
    "name": "alpha",
    "display_name": "@vrooli/alpha",
    "kind": "js_runtime",
    "module_identifiers": ["@vrooli/alpha"],
    "adoption": {
      "scenario_adoptable": true,
      "allowed_consumers": ["scenario_ui"],
      "adoption_modes": ["file_dependency"]
    },
    "lifecycle": {
      "build": [
        {
          "name": "build",
          "run": ["bash", "-lc", "mkdir -p build && printf build > build/build.txt"]
        }
      ]
    },
    "refresh": {
      "strategy": "scenario_setup",
      "restart_running_consumers": false
    }
  }
}`)
	testkitvrooli.WriteScenarioService(t, fixture.Root, "demo", testkitvrooli.ScenarioServiceManifest("demo",
		testkitvrooli.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name: "capture-setup",
				Run:  "mkdir -p build && printf setup > build/setup.txt",
			}}},
		}),
	))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "ui", "package.json"), `{
  "dependencies": {
    "@vrooli/alpha": "file:../../../packages/alpha"
  }
}`)

	app := newTestApp(fixture.Root)
	app.homeDir = func() (string, error) { return fixture.Home, nil }
	ctx := &commandContext{Root: fixture.Root, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, app: app}

	_, resp, err := runPackageRefreshRequest(app, ctx, packageRefreshRequest{Name: "alpha", Target: "all"})
	if err != nil {
		t.Fatalf("runPackageRefreshRequest: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Status != "setup_only" {
		t.Fatalf("refresh items = %#v", resp.Items)
	}
	if _, err := os.Stat(filepath.Join(fixture.Root, "packages", "alpha", "build", "build.txt")); err != nil {
		t.Fatal("expected package build marker")
	}
	if _, err := os.Stat(filepath.Join(fixture.Root, "scenarios", "demo", "build", "setup.txt")); err != nil {
		t.Fatal("expected scenario setup marker")
	}
}

func TestPackageRefreshGenerateThenSetupRunsGenerateBuildAndSetup(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeScenarioPortRegistryFixture(t, fixture.Root)
	writePackageManifestFixture(t, fixture.Root, "proto", `{
  "$schema": "schemas/package.schema.json",
  "version": "1.0.0",
  "package": {
    "name": "proto",
    "display_name": "@vrooli/proto",
    "kind": "schema_or_contract",
    "module_identifiers": ["@vrooli/proto-types"],
    "adoption": {
      "scenario_adoptable": true,
      "allowed_consumers": ["scenario_ui"],
      "adoption_modes": ["file_dependency"]
    },
    "lifecycle": {
      "generate": [
        {
          "name": "generate",
          "run": ["bash", "-lc", "mkdir -p build && printf generate > build/generate.txt"]
        }
      ],
      "build": [
        {
          "name": "build",
          "run": ["bash", "-lc", "mkdir -p build && printf build > build/build.txt"]
        }
      ]
    },
    "refresh": {
      "strategy": "generate_then_setup",
      "restart_running_consumers": false
    }
  }
}`)
	testkitvrooli.WriteScenarioService(t, fixture.Root, "demo", testkitvrooli.ScenarioServiceManifest("demo",
		testkitvrooli.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name: "capture-setup",
				Run:  "mkdir -p build && printf setup > build/setup.txt",
			}}},
		}),
	))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "ui", "package.json"), `{
  "dependencies": {
    "@vrooli/proto-types": "file:../../../packages/proto"
  }
}`)

	app := newTestApp(fixture.Root)
	app.homeDir = func() (string, error) { return fixture.Home, nil }
	ctx := &commandContext{Root: fixture.Root, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, app: app}

	_, resp, err := runPackageRefreshRequest(app, ctx, packageRefreshRequest{Name: "proto", Target: "all"})
	if err != nil {
		t.Fatalf("runPackageRefreshRequest: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Status != "setup_only" {
		t.Fatalf("refresh items = %#v", resp.Items)
	}
	if _, err := os.Stat(filepath.Join(fixture.Root, "packages", "proto", "build", "generate.txt")); err != nil {
		t.Fatal("expected package generate marker")
	}
	if _, err := os.Stat(filepath.Join(fixture.Root, "packages", "proto", "build", "build.txt")); err != nil {
		t.Fatal("expected package build marker")
	}
	if _, err := os.Stat(filepath.Join(fixture.Root, "scenarios", "demo", "build", "setup.txt")); err != nil {
		t.Fatal("expected scenario setup marker")
	}
}

func TestPackageRefreshRebuildCLIRebuildsConsumer(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeScenarioPortRegistryFixture(t, fixture.Root)
	writePackageManifestFixture(t, fixture.Root, "cli-core", `{
  "$schema": "schemas/package.schema.json",
  "version": "1.0.0",
  "package": {
    "name": "cli-core",
    "display_name": "example.com/cli-core",
    "kind": "go_cli",
    "module_identifiers": ["example.com/cli-core"],
    "adoption": {
      "scenario_adoptable": true,
      "allowed_consumers": ["scenario_cli"],
      "adoption_modes": ["go_module_replace"]
    },
    "lifecycle": {},
    "refresh": {
      "strategy": "rebuild_cli_consumers",
      "restart_running_consumers": false
    }
  }
}`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "cli-core", "go.mod"), "module example.com/cli-core\n\ngo 1.25.0\n")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "cli-core", "cli_core.go"), "package clicore\n\nfunc Name() string { return \"ok\" }\n")
	testkitvrooli.WriteScenarioService(t, fixture.Root, "demo", testkitvrooli.ScenarioServiceManifest("demo", testkitvrooli.WithLifecycle(scenario.Lifecycle{Version: "2.0.0"})))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "cli", "go.mod"), `module example.com/demo/cli

go 1.25.0

require example.com/cli-core v0.0.0

replace example.com/cli-core => ../../../packages/cli-core
`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "cli", "main.go"), `package main

import core "example.com/cli-core"

func main() {
	_ = core.Name()
}
`)

	app := newTestApp(fixture.Root)
	app.homeDir = func() (string, error) { return fixture.Home, nil }
	ctx := &commandContext{Root: fixture.Root, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, app: app}

	_, resp, err := runPackageRefreshRequest(app, ctx, packageRefreshRequest{Name: "cli-core", Target: "all"})
	if err != nil {
		t.Fatalf("runPackageRefreshRequest: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Status != "rebuilt" {
		t.Fatalf("refresh items = %#v", resp.Items)
	}
}

func TestPackageRefreshTargetFiltersAffectedScenario(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeScenarioPortRegistryFixture(t, fixture.Root)
	writePackageManifestFixture(t, fixture.Root, "alpha", `{
  "$schema": "schemas/package.schema.json",
  "version": "1.0.0",
  "package": {
    "name": "alpha",
    "display_name": "@vrooli/alpha",
    "kind": "js_runtime",
    "module_identifiers": ["@vrooli/alpha"],
    "adoption": {
      "scenario_adoptable": true,
      "allowed_consumers": ["scenario_ui"],
      "adoption_modes": ["file_dependency"]
    },
    "lifecycle": {
      "build": [
        {
          "name": "build",
          "run": ["bash", "-lc", "mkdir -p build && printf build > build/build.txt"]
        }
      ]
    },
    "refresh": {
      "strategy": "scenario_setup",
      "restart_running_consumers": false
    }
  }
}`)
	for _, name := range []string{"alpha-ui", "beta-ui"} {
		testkitvrooli.WriteScenarioService(t, fixture.Root, name, testkitvrooli.ScenarioServiceManifest(name,
			testkitvrooli.WithLifecycle(scenario.Lifecycle{
				Version: "2.0.0",
				Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
					Name: "capture-setup",
					Run:  "mkdir -p build && printf setup > build/setup.txt",
				}}},
			}),
		))
		testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", name, "ui", "package.json"), `{
  "dependencies": {
    "@vrooli/alpha": "file:../../../packages/alpha"
  }
}`)
	}

	app := newTestApp(fixture.Root)
	app.homeDir = func() (string, error) { return fixture.Home, nil }
	ctx := &commandContext{Root: fixture.Root, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, app: app}

	_, resp, err := runPackageRefreshRequest(app, ctx, packageRefreshRequest{Name: "alpha", Target: "beta-ui"})
	if err != nil {
		t.Fatalf("runPackageRefreshRequest: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Scenario != "beta-ui" {
		t.Fatalf("refresh items = %#v", resp.Items)
	}
	if _, err := os.Stat(filepath.Join(fixture.Root, "scenarios", "alpha-ui", "build", "setup.txt")); !os.IsNotExist(err) {
		t.Fatalf("alpha-ui should not have been refreshed, err=%v", err)
	}
}

func writePackageManifestFixture(t *testing.T, root, name, manifest string) {
	t.Helper()
	testkitgo.WriteFile(t, filepath.Join(root, "packages", name, ".vrooli", "package.json"), manifest)
}
