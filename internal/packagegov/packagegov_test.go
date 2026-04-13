package packagegov

import (
	"os"
	"path/filepath"
	"testing"

	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
)

func TestLoadAllReportsMissingManifest(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	packagesDir := filepath.Join(fixture.Root, "packages")
	if err := os.MkdirAll(filepath.Join(packagesDir, "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}

	items, issues, err := LoadAll(fixture.Root)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("items = %d, want 0", len(items))
	}
	if len(issues) != 1 || issues[0].Code != "missing-package-manifest" {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestValidateFlagsWorkspaceDepsAndPostinstallDebt(t *testing.T) {
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
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", ".vrooli", "service.json"), `{"service":{"name":"demo","parent":"vrooli"},"lifecycle":{"version":"2.0.0"}}`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "ui", "package.json"), `{
  "scripts": {
    "postinstall": "mkdir -p node_modules/@vrooli && cp -a ../../../packages/alpha node_modules/@vrooli/alpha"
  },
  "dependencies": {
    "@vrooli/alpha": "workspace:*"
  }
}`)

	report, err := Validate(fixture.Root, "")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(report.Issues) < 2 {
		t.Fatalf("issues = %#v", report.Issues)
	}
}

func TestValidateFlagsScenarioAdoptionOfInternalOnlyPackage(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "testkit-go", ".vrooli", "package.json"), `{
  "$schema": "schemas/package.schema.json",
  "version": "1.0.0",
  "package": {
    "name": "testkit-go",
    "display_name": "github.com/example/testkit-go",
    "kind": "go_testkit",
    "module_identifiers": ["github.com/example/testkit-go"],
    "adoption": {
      "scenario_adoptable": false,
      "allowed_consumers": ["internal_platform"],
      "adoption_modes": []
    },
    "lifecycle": {},
    "refresh": {
      "strategy": "none",
      "restart_running_consumers": false
    }
  }
}`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", ".vrooli", "service.json"), `{"service":{"name":"demo","parent":"vrooli"},"lifecycle":{"version":"2.0.0"}}`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "api", "go.mod"), `module example.com/demo/api

go 1.25.0

require github.com/example/testkit-go v0.0.0

replace github.com/example/testkit-go => ../../../packages/testkit-go
`)

	report, err := Validate(fixture.Root, "")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.Code == "package-adoption-supported" && issue.PackageName == "testkit-go" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected package-adoption-supported issue, got %#v", report.Issues)
	}
}

func TestLoadAllRejectsUnknownManifestFields(t *testing.T) {
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
    "unexpected": true,
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

	_, issues, err := LoadAll(fixture.Root)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(issues) != 1 || issues[0].Code != "invalid-package-manifest" {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestValidateRequiresGoReplaceForGovernedAdoption(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "docs", "package-governance.md"), "# ok\n")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "alpha", ".vrooli", "package.json"), `{
  "$schema": "schemas/package.schema.json",
  "version": "1.0.0",
  "package": {
    "name": "alpha",
    "display_name": "github.com/example/alpha",
    "kind": "go_runtime",
    "module_identifiers": ["github.com/example/alpha"],
    "adoption": {
      "scenario_adoptable": true,
      "allowed_consumers": ["scenario_api"],
      "adoption_modes": ["go_module_replace"]
    },
    "lifecycle": {},
    "refresh": {
      "strategy": "restart_running_consumers",
      "restart_running_consumers": true
    },
    "docs": ["docs/package-governance.md"]
  }
}`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", ".vrooli", "service.json"), `{"service":{"name":"demo","parent":"vrooli"},"lifecycle":{"version":"2.0.0"}}`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "api", "go.mod"), `module example.com/demo/api

go 1.25.0

require github.com/example/alpha v0.0.0
`)

	report, err := Validate(fixture.Root, "")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	for _, issue := range report.Issues {
		if issue.PackageName == "alpha" {
			t.Fatalf("unexpected issues for alpha without replace = %#v", report.Issues)
		}
	}
}
