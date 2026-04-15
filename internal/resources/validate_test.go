package resources

import (
	"strings"
	"testing"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

func TestValidateResourcesRejectsRepoLocalDataVolumeSourcesWithoutLegacyMarker(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", manifestpkg.ResourceManifest{
		Name:            "fixture",
		Driver:          "docker-service",
		Template:        "docker-service",
		PortabilityTier: "full",
		Runtime: manifestpkg.ResourceRuntime{
			Image: "fixture:latest",
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

func TestValidateResourcesAllowsExplicitLegacyRepoDataVolumeSources(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceManifest(t, root, "fixture", manifestpkg.ResourceManifest{
		Name:                  "fixture",
		Driver:                "docker-service",
		Template:              "docker-service",
		PortabilityTier:       "full",
		LegacyRepoDataAllowed: true,
		Runtime: manifestpkg.ResourceRuntime{
			Image: "fixture:latest",
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
		Name:            "fixture",
		Driver:          "external-cli",
		Binary:          "fixture-cli",
		PortabilityTier: "full",
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
