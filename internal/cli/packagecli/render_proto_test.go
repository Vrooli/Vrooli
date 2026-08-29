//nolint:goconst // test data deliberately reuses stable protocol fixtures.
package packagecli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/packagegov"
)

func samplePackage() packagegov.Package {
	return packagegov.Package{
		Name:         "proto",
		RootPath:     "/repo/packages/proto",
		ManifestPath: "/repo/packages/proto/package.json",
		Manifest: packagegov.Manifest{
			Schema:  "schemas/package.schema.json",
			Version: "1.0.0",
			Package: packagegov.ManifestEntry{
				Name:              "proto",
				DisplayName:       "Proto",
				Description:       "shared contracts",
				Kind:              packagegov.KindSchemaOrContract,
				Language:          "go",
				ModuleIdentifiers: []string{"cliv1"},
				GeneratedOutputs: []packagegov.GeneratedOutput{
					{Name: "cli/v1", Identifiers: []string{"PackageInfo"}, Consumers: []packagegov.ConsumerClass{packagegov.ConsumerScenarioAPI}},
				},
				Adoption: packagegov.AdoptionPolicy{
					ScenarioAdoptable: true,
					AllowedConsumers:  []packagegov.ConsumerClass{packagegov.ConsumerScenarioAPI},
					AdoptionModes:     []packagegov.AdoptionMode{packagegov.ModeGeneratedArtifact},
				},
				Lifecycle: packagegov.LifecyclePolicy{
					Generate: []packagegov.CommandSpec{{Name: "buf", Run: []string{"buf", "generate"}}},
				},
				Refresh: packagegov.RefreshPolicy{Strategy: packagegov.RefreshScenarioSetup, RestartRunningConsumers: true},
				Docs:    []string{"docs/proto.md"},
			},
		},
	}
}

func TestPackageListJSONContract(t *testing.T) {
	resp := ListResponse{Packages: []packagegov.Package{samplePackage()}}

	var buf bytes.Buffer
	if err := RenderList(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderList: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["success"] != true {
		t.Errorf("success: want true, got %v", got["success"])
	}
	pkgs, ok := got["packages"].([]any)
	if !ok || len(pkgs) != 1 {
		t.Fatalf("packages: want 1, got %v", got["packages"])
	}
	first := pkgs[0].(map[string]any)
	if first["name"] != "proto" || first["root_path"] != "/repo/packages/proto" {
		t.Errorf("info mismatch: %v", first)
	}
	manifest := first["manifest"].(map[string]any)
	entry := manifest["package"].(map[string]any)
	if entry["display_name"] != "Proto" {
		t.Errorf("display_name (snake_case?): %v", entry)
	}
	if _, ok := entry["module_identifiers"].([]any); !ok {
		t.Errorf("module_identifiers missing/wrong: %v", entry)
	}
	adoption := entry["adoption"].(map[string]any)
	if adoption["scenario_adoptable"] != true {
		t.Errorf("scenario_adoptable: %v", adoption)
	}
}

func TestPackageInfoJSONContract(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderInfo(&buf, cliout.FormatJSON, samplePackage()); err != nil {
		t.Fatalf("RenderInfo: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["success"] != true {
		t.Errorf("success: %v", got["success"])
	}
	pkg, ok := got["package"].(map[string]any)
	if !ok || pkg["name"] != "proto" {
		t.Fatalf("package: %v", got["package"])
	}
}

func TestPackageDependentsJSONContract(t *testing.T) {
	resp := DependentsResponse{
		PackageName: "proto",
		Dependents: []packagegov.Dependent{
			{
				PackageName:    "proto",
				ConsumerName:   "search-hub",
				ConsumerClass:  packagegov.ConsumerScenarioAPI,
				AdoptionMode:   packagegov.ModeGoModuleReplace,
				DependencyFile: "api/go.mod",
			},
		},
		// Sparse: no issues — must still emit issues: [] under EmitUnpopulated.
	}

	var buf bytes.Buffer
	if err := RenderDependents(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderDependents: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["success"] != true {
		t.Errorf("success: %v", got["success"])
	}
	dep := got["dependents"].(map[string]any)
	if dep["package_name"] != "proto" {
		t.Errorf("package_name: %v", dep)
	}
	deps := dep["dependents"].([]any)
	if len(deps) != 1 {
		t.Fatalf("dependents: %v", deps)
	}
	d := deps[0].(map[string]any)
	if d["consumer_name"] != "search-hub" || d["consumer_class"] != "scenario_api" {
		t.Errorf("dependent mismatch: %v", d)
	}
	if _, ok := dep["issues"].([]any); !ok {
		t.Errorf("issues should be emitted as [] (EmitUnpopulated): %v", dep["issues"])
	}
}

func TestPackageRunJSONContract(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderRun(&buf, cliout.FormatJSON, RunResponse{PackageName: "proto", Action: "build"}); err != nil {
		t.Fatalf("RenderRun: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	result := got["result"].(map[string]any)
	if result["package_name"] != "proto" || result["action"] != "build" {
		t.Errorf("result mismatch: %v", result)
	}
}

func TestPackageRefreshJSONContract(t *testing.T) {
	resp := RefreshResponse{
		PackageName: "proto",
		Items: []RefreshItem{
			{
				Consumer: "search-hub",
				Class:    packagegov.ConsumerScenarioAPI,
				Classes:  []packagegov.ConsumerClass{packagegov.ConsumerScenarioAPI, packagegov.ConsumerScenarioCLI},
				Action:   packagegov.RefreshActionKind("scenario_setup"),
				Status:   "ok",
			},
		},
	}
	var buf bytes.Buffer
	if err := RenderRefresh(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderRefresh: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	refresh := got["refresh"].(map[string]any)
	items := refresh["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items: %v", items)
	}
	item := items[0].(map[string]any)
	if item["consumer_class"] != "scenario_api" {
		t.Errorf("consumer_class: %v", item)
	}
	if classes, ok := item["consumer_classes"].([]any); !ok || len(classes) != 2 {
		t.Errorf("consumer_classes: %v", item["consumer_classes"])
	}
}
