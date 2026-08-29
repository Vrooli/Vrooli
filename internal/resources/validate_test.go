package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
)

func TestValidateResourcesRejectsUnknownManifestField(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", manifestpkg.ResourceManifest{Name: "fixture", Driver: "external-cli", Binary: "bash"})
	path := filepath.Join(root, "resources", "fixture", "resource.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	fields["template"] = "external-cli"
	data, err = json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := NewController(root, home).ValidateResources("fixture")
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed || !strings.Contains(report.Items[0].Issues[0].Message, "unknown_manifest_field") {
		t.Fatalf("expected unknown field issue, got %#v", report)
	}
}

func TestValidateResourcesRejectsUnknownDriver(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", manifestpkg.ResourceManifest{Name: "fixture", Driver: "docker-service", Binary: "bash"})
	report, err := NewController(root, home).ValidateResources("fixture")
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed || len(report.Items) != 1 || !strings.Contains(report.Items[0].Issues[0].Message, "unknown_driver") {
		t.Fatalf("expected unknown driver issue, got %#v", report)
	}
}

func TestValidateResourcesRejectsConflictingTestDirectories(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", manifestpkg.ResourceManifest{Name: "fixture", Driver: "external-cli", Binary: "bash"})
	if err := os.Mkdir(filepath.Join(root, "resources", "fixture", "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "resources", "fixture", "tests"), 0o755); err != nil {
		t.Fatal(err)
	}
	report, err := NewController(root, home).ValidateResources("fixture")
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed || !strings.Contains(issueText(report), "conflicting_test_dir") {
		t.Fatalf("expected conflicting test directory issue, got %#v", report)
	}
}

func TestValidateResourcesRejectsEmptyRequiredDirectory(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", manifestpkg.ResourceManifest{Name: "fixture", Driver: "external-cli", Binary: "bash"})
	if err := os.Remove(filepath.Join(root, "resources", "fixture", "cli", "main.go")); err != nil {
		t.Fatal(err)
	}
	report, err := NewController(root, home).ValidateResources("fixture")
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed || !strings.Contains(issueText(report), "empty_required_dir") {
		t.Fatalf("expected empty required directory issue, got %#v", report)
	}
}

func TestValidateResourcesRejectsDeclaredCLIWithoutModule(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", manifestpkg.ResourceManifest{Name: "fixture", Driver: "external-cli", Binary: "bash"})
	if err := os.RemoveAll(filepath.Join(root, "resources", "fixture", "cli")); err != nil {
		t.Fatal(err)
	}
	report, err := NewController(root, home).ValidateResources("fixture")
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed || !strings.Contains(issueText(report), "cli_declared_without_module") {
		t.Fatalf("expected missing CLI module issue, got %#v", report)
	}
}

func issueText(report ResourceValidationReport) string {
	var messages []string
	for _, item := range report.Items {
		for _, issue := range item.Issues {
			messages = append(messages, issue.Message)
		}
	}
	return strings.Join(messages, "\n")
}

func TestValidateResourcesRejectsRepoLocalDataVolumeSourcesWithoutLegacyMarker(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", manifestpkg.ResourceManifest{
		Name:   "fixture",
		Driver: "external-cli",
		Binary: "bash",
		Runtime: manifestpkg.ResourceRuntime{
			Image: "fixture:1.0.0",
			Volumes: []manifestpkg.ResourceVolume{
				{Source: "${ROOT}/data/resources/fixture/data", Target: "/var/lib/fixture"},
			},
		},
	})

	report, err := NewController(root, home).ValidateResources("fixture")
	if err != nil {
		t.Fatalf("ValidateResources: %v", err)
	}
	if report.Passed {
		t.Fatal("expected validation to fail for repo-local data volume")
	}
	if len(report.Items) != 1 || len(report.Items[0].Issues) == 0 {
		t.Fatalf("expected validation issues, got %#v", report)
	}
	if got := report.Items[0].Issues[0].Message; !strings.Contains(got, "legacy_repo_data_allowed=true") {
		t.Fatalf("issue = %q, want legacy marker guidance", got)
	}
}

func TestValidateResourcesRejectsResourceAbsentFromRootContract(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "other", true)
	testresource.WriteResourceManifest(t, root, "fixture", manifestpkg.ResourceManifest{
		Name: "fixture", Driver: "external-cli", Binary: "bash",
	})

	report, err := NewController(root, home).ValidateResources("fixture")
	if err != nil {
		t.Fatalf("ValidateResources: %v", err)
	}
	if report.Passed {
		t.Fatal("expected validation to fail for resource absent from root contract")
	}
	for _, issue := range report.Items[0].Issues {
		if strings.Contains(issue.Message, "resource_absent_from_contract") {
			return
		}
	}
	t.Fatalf("missing resource_absent_from_contract issue: %#v", report.Items[0].Issues)
}

func TestValidateResourcesRejectsRepoLocalResourceInstanceSourcesWithoutLegacyMarker(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "postgres", true)
	testresource.WriteResourceManifest(t, root, "postgres", manifestpkg.ResourceManifest{
		Name:   "postgres",
		Driver: "external-cli",
		Binary: "bash",
		Runtime: manifestpkg.ResourceRuntime{
			Image: "postgres:16-alpine",
			Volumes: []manifestpkg.ResourceVolume{
				{Source: "resources/postgres/instances/main/data", Target: "/var/lib/postgresql/data"},
			},
		},
	})

	report, err := NewController(root, home).ValidateResources("postgres")
	if err != nil {
		t.Fatalf("ValidateResources: %v", err)
	}
	if report.Passed {
		t.Fatal("expected validation to fail for repo-local resource instance volume")
	}
	if len(report.Items) != 1 || len(report.Items[0].Issues) == 0 {
		t.Fatalf("expected validation issues, got %#v", report)
	}
	if got := report.Items[0].Issues[0].Message; !strings.Contains(got, "legacy_repo_data_allowed=true") {
		t.Fatalf("issue = %q, want legacy marker guidance", got)
	}
}

func TestValidateResourcesAllowsExplicitLegacyRepoDataVolumeSources(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", manifestpkg.ResourceManifest{
		Name:                  "fixture",
		Driver:                "external-cli",
		Binary:                "bash",
		LegacyRepoDataAllowed: true,
		Runtime: manifestpkg.ResourceRuntime{
			Image: "fixture:1.0.0",
			Volumes: []manifestpkg.ResourceVolume{
				{Source: "${ROOT}/data/resources/fixture/data", Target: "/var/lib/fixture"},
			},
		},
	})

	report, err := NewController(root, home).ValidateResources("fixture")
	if err != nil {
		t.Fatalf("ValidateResources: %v", err)
	}
	if !report.Passed {
		t.Fatalf("expected explicit legacy marker to pass validation, got %#v", report)
	}
}

func TestValidateResourcesAllowsStorageContextVariablesInDerivedExports(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", manifestpkg.ResourceManifest{
		Name:   "fixture",
		Driver: "external-cli",
		Binary: "fixture-cli",
		EnvironmentExports: manifestpkg.ResourceEnvironmentExports{
			Derived: map[string]manifestpkg.ResourceDerivedTemplate{
				"FIXTURE_DATA_DIR":  {Template: "${VROOLI_DATA}/fixture"},
				"FIXTURE_CACHE_DIR": {Template: "${RESOURCE_CACHE_DIR}/tmp"},
			},
		},
	})

	report, err := NewController(root, home).ValidateResources("fixture")
	if err != nil {
		t.Fatalf("ValidateResources: %v", err)
	}
	if !report.Passed {
		t.Fatalf("expected storage context variables to validate, got %#v", report)
	}
}
