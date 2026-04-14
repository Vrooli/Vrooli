package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	packageapp "github.com/vrooli/vrooli/internal/app/package"
	"github.com/vrooli/vrooli/internal/cli/packagecli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/scenario"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testkitvrooli "github.com/vrooli/vrooli/packages/testkit-go/vrooli"
)

func showPackageHelp(w io.Writer) {
	packagecli.RenderCommandHelp(w)
}

func runPackageRootCommand(app *App, ctx *commandContext, args []string) error {
	handler := buildTopLevelHandlerMap()["package"]
	if handler == nil {
		return fmt.Errorf("package handler not registered")
	}
	return handler(ctx, args)
}

func runPackageRefreshRequest(app *App, ctx *commandContext, req packagecli.RefreshRequest) (cliout.Format, packageapp.RefreshResponse, error) {
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packageapp.RefreshResponse{}, err
	}
	stdout := ctx.Stdout
	stderr := ctx.Stderr
	if format == cliout.FormatJSON {
		stdout = stderr
	}
	resp, err := packageapp.Service{
		Root:   ctx.Root,
		Stdout: stdout,
		Stderr: stderr,
		ScenarioService: func() (packageapp.ScenarioRuntime, error) {
			return app.newScenarioService(ctx)
		},
		ScenarioRunner: func() (packageapp.ScenarioPhaseRunner, error) {
			return app.newScenarioLifecycleRunner(ctx)
		},
	}.Refresh(packageapp.RefreshRequest{
		PackageName: req.Name,
		Target:      req.Target,
		NoRestart:   req.NoRestart,
	})
	if err != nil {
		return "", packageapp.RefreshResponse{}, err
	}
	return format, resp, nil
}

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
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", repocontractmeta.ServiceManifestPathname), `{"service":{"name":"demo","parent":"vrooli"},"lifecycle":{"version":"2.0.0"}}`)
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

	_, resp, err := runPackageRefreshRequest(app, ctx, packagecli.RefreshRequest{Name: "alpha", Target: "all"})
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

	_, resp, err := runPackageRefreshRequest(app, ctx, packagecli.RefreshRequest{Name: "proto", Target: "all"})
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

	_, resp, err := runPackageRefreshRequest(app, ctx, packagecli.RefreshRequest{Name: "cli-core", Target: "all"})
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

	_, resp, err := runPackageRefreshRequest(app, ctx, packagecli.RefreshRequest{Name: "alpha", Target: "beta-ui"})
	if err != nil {
		t.Fatalf("runPackageRefreshRequest: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Consumer != "beta-ui" {
		t.Fatalf("refresh items = %#v", resp.Items)
	}
	if _, err := os.Stat(filepath.Join(fixture.Root, "scenarios", "alpha-ui", "build", "setup.txt")); !os.IsNotExist(err) {
		t.Fatalf("alpha-ui should not have been refreshed, err=%v", err)
	}
}

func TestPackageRefreshIncludesTemplateConsumersExplicitly(t *testing.T) {
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
      "allowed_consumers": ["scenario_ui", "template_ui"],
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
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "templates", "scenarios", "react-vite", "ui", "package.json"), `{
  "dependencies": {
    "@vrooli/alpha": "file:../../../packages/alpha"
  }
}`)

	app := newTestApp(fixture.Root)
	app.homeDir = func() (string, error) { return fixture.Home, nil }
	ctx := &commandContext{Root: fixture.Root, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, app: app}

	_, resp, err := runPackageRefreshRequest(app, ctx, packagecli.RefreshRequest{Name: "alpha", Target: "all"})
	if err != nil {
		t.Fatalf("runPackageRefreshRequest: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("refresh items = %#v", resp.Items)
	}
	if resp.Items[0].Consumer != "demo" || resp.Items[0].Status != "setup_only" {
		t.Fatalf("unexpected scenario refresh item = %#v", resp.Items[0])
	}
	if resp.Items[1].Consumer != "react-vite" || resp.Items[1].Status != "no_runtime_refresh" {
		t.Fatalf("unexpected template refresh item = %#v", resp.Items[1])
	}
}

func TestPackageRefreshRebuildsResourceConsumers(t *testing.T) {
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
      "allowed_consumers": ["resource_runtime"],
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
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "resources", "sqlite", "go.mod"), `module example.com/resources/sqlite

go 1.25.0

require example.com/cli-core v0.0.0

replace example.com/cli-core => ../../packages/cli-core
`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "resources", "sqlite", "main.go"), `package sqlite

import core "example.com/cli-core"

func Name() string {
	return core.Name()
}
`)

	app := newTestApp(fixture.Root)
	app.homeDir = func() (string, error) { return fixture.Home, nil }
	ctx := &commandContext{Root: fixture.Root, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, app: app}

	_, resp, err := runPackageRefreshRequest(app, ctx, packagecli.RefreshRequest{Name: "cli-core", Target: "all"})
	if err != nil {
		t.Fatalf("runPackageRefreshRequest: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Consumer != "sqlite" || resp.Items[0].Status != "rebuilt" {
		t.Fatalf("refresh items = %#v", resp.Items)
	}
}

func TestPackageRefreshDedupesMultiSurfaceScenarioSetup(t *testing.T) {
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
    "module_identifiers": ["github.com/example/proto", "@vrooli/proto-types"],
    "generated_outputs": [
      {
        "name": "proto-types",
        "identifiers": ["@vrooli/proto-types"],
        "consumers": ["scenario_ui"]
      }
    ],
    "adoption": {
      "scenario_adoptable": true,
      "allowed_consumers": ["scenario_ui", "scenario_api"],
      "adoption_modes": ["go_module_replace", "generated_artifact"]
    },
    "lifecycle": {
      "generate": [
        {
          "name": "generate",
          "run": ["bash", "-lc", "mkdir -p build && printf generate > build/generate.txt"]
        }
      ]
    },
    "refresh": {
      "strategy": "generate_then_setup",
      "restart_running_consumers": false
    }
  }
}`)
	testkitvrooli.WriteScenarioService(t, fixture.Root, "desktop", testkitvrooli.ScenarioServiceManifest("desktop",
		testkitvrooli.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name: "capture-setup",
				Run:  "mkdir -p build && printf setup >> build/setup.txt",
			}}},
		}),
	))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "desktop", "api", "go.mod"), `module example.com/desktop/api

go 1.25.0

require github.com/example/proto v0.0.0

replace github.com/example/proto => ../../../packages/proto
`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "desktop", "ui", "package.json"), `{
  "dependencies": {
    "@vrooli/proto-types": "file:../../../packages/proto/gen/typescript"
  }
}`)

	app := newTestApp(fixture.Root)
	app.homeDir = func() (string, error) { return fixture.Home, nil }
	ctx := &commandContext{Root: fixture.Root, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, app: app}

	_, resp, err := runPackageRefreshRequest(app, ctx, packagecli.RefreshRequest{Name: "proto", Target: "desktop"})
	if err != nil {
		t.Fatalf("runPackageRefreshRequest: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("refresh items = %#v", resp.Items)
	}
	if len(resp.Items[0].Classes) != 2 {
		t.Fatalf("expected merged consumer classes, got %#v", resp.Items[0])
	}
	data, err := os.ReadFile(filepath.Join(fixture.Root, "scenarios", "desktop", "build", "setup.txt"))
	if err != nil {
		t.Fatalf("read setup marker: %v", err)
	}
	if strings.Count(string(data), "setup") != 1 {
		t.Fatalf("expected setup to run once, got %q", string(data))
	}
}

func TestPackageCommandsSupportJSONOutput(t *testing.T) {
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
    },
    "docs": ["docs/package-governance.md"]
  }
}`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "docs", "package-governance.md"), "# ok\n")
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

	commands := [][]string{
		{"list"},
		{"info", "alpha"},
		{"dependents", "alpha"},
		{"validate", "--all"},
		{"build", "alpha"},
		{"refresh", "alpha", "demo"},
		{"audit", "--all"},
	}
	for _, args := range commands {
		t.Run(strings.Join(args, "-"), func(t *testing.T) {
			app, ctx := newConfiguredCommandContext(fixture.Root, globalOptions{JSON: true}, &bytes.Buffer{}, &bytes.Buffer{})
			app.homeDir = func() (string, error) { return fixture.Home, nil }
			var stdout bytes.Buffer
			ctx.Stdout = &stdout
			if err := runPackageRootCommand(app, ctx, args); err != nil {
				t.Fatalf("runPackageRootCommand(%v): %v", args, err)
			}

			var payload map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
				t.Fatalf("json output invalid: %v\n%s", err, stdout.String())
			}
			if success, ok := payload["success"].(bool); !ok || !success {
				t.Fatalf("expected success payload, got %v", payload)
			}
		})
	}
}

func TestPackageRefreshRestartsRunningScenario(t *testing.T) {
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
      "restart_running_consumers": true
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
			Develop: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name:       "stay-running",
				Run:        "sleep 30",
				Background: true,
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
	service, err := app.newScenarioService(ctx)
	if err != nil {
		t.Fatalf("newScenarioService: %v", err)
	}
	if _, err := service.StartDetailed("demo", lifecycle.StartOptions{}); err != nil {
		t.Fatalf("StartDetailed: %v", err)
	}
	t.Cleanup(func() {
		runner, runErr := app.newScenarioLifecycleRunner(ctx)
		if runErr == nil {
			_ = runner.Stop("demo", lifecycle.StopOptions{})
		}
	})

	_, resp, err := runPackageRefreshRequest(app, ctx, packagecli.RefreshRequest{Name: "alpha", Target: "demo"})
	if err != nil {
		t.Fatalf("runPackageRefreshRequest: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Status != "restarted" {
		t.Fatalf("refresh items = %#v", resp.Items)
	}
	detail, _, err := service.Lookup("demo")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if detail.Runtime.ProcessCount == 0 {
		t.Fatalf("expected demo to be running after restart, detail=%#v", detail.Runtime)
	}
	if _, err := os.Stat(filepath.Join(fixture.Root, "scenarios", "demo", "build", "setup.txt")); err != nil {
		t.Fatal("expected scenario setup marker")
	}
}

func TestPackageRefreshNoRestartLeavesScenarioStopped(t *testing.T) {
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
    "lifecycle": {},
    "refresh": {
      "strategy": "scenario_setup",
      "restart_running_consumers": true
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
			Develop: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name:       "stay-running",
				Run:        "sleep 30",
				Background: true,
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
	service, err := app.newScenarioService(ctx)
	if err != nil {
		t.Fatalf("newScenarioService: %v", err)
	}
	if _, err := service.StartDetailed("demo", lifecycle.StartOptions{}); err != nil {
		t.Fatalf("StartDetailed: %v", err)
	}
	t.Cleanup(func() {
		runner, runErr := app.newScenarioLifecycleRunner(ctx)
		if runErr == nil {
			_ = runner.Stop("demo", lifecycle.StopOptions{})
		}
	})

	_, resp, err := runPackageRefreshRequest(app, ctx, packagecli.RefreshRequest{Name: "alpha", Target: "demo", NoRestart: true})
	if err != nil {
		t.Fatalf("runPackageRefreshRequest: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Status != "stopped_after_setup" {
		t.Fatalf("refresh items = %#v", resp.Items)
	}
	detail, _, err := service.Lookup("demo")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if detail.Runtime.ProcessCount != 0 {
		t.Fatalf("expected demo to be stopped after refresh, detail=%#v", detail.Runtime)
	}
}

func writePackageManifestFixture(t *testing.T, root, name, manifest string) {
	t.Helper()
	testkitgo.WriteFile(t, filepath.Join(root, "packages", name, ".vrooli", "package.json"), manifest)
}
