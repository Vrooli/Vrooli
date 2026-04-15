package resources

import (
	"path/filepath"
	"testing"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

func writeResourceCLI(t *testing.T, root, name string) {
	t.Helper()
	testresource.WriteResourceCLI(t, root, name, "#!/usr/bin/env bash\nexit 0\n")
}

func writeDeprecatedMetadata(t *testing.T, root string, items ...DeprecatedResource) {
	t.Helper()
	testkitgo.WriteJSON(t, filepath.Join(root, filepath.FromSlash(deprecatedResourcesPath)), DeprecatedResourceList{
		Resources: items,
	})
}

func writeBlueprintArchivedMetadata(t *testing.T, root string, items ...BlueprintArchivedResource) {
	t.Helper()
	testkitgo.WriteJSON(t, filepath.Join(root, filepath.FromSlash(blueprintArchivedResourcesPath)), BlueprintArchivedResourceList{
		Resources: items,
	})
}

func writeScenarioResourceManifest(t *testing.T, root, scenarioName, resourceName string) {
	t.Helper()
	testscenario.WriteScenarioService(t, root, scenarioName, testscenario.ScenarioServiceManifest(
		scenarioName,
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				resourceName: {
					Enabled:  true,
					Required: true,
				},
			},
		}),
	))
}

func writeResourceConfig(t *testing.T, root, name string, enabled bool) {
	t.Helper()
	testscenario.WriteProjectResourceConfig(t, root, name, enabled)
}

func writeResourceManifest(t *testing.T, root, name string, manifest manifestpkg.ResourceManifest) {
	t.Helper()
	testresource.WriteResourceManifest(t, root, name, manifest)
}

func writeEnvManifestFixture(t *testing.T, root, name string, manifest manifestpkg.ResourceManifest) {
	t.Helper()
	testresource.WriteResourceManifest(t, root, name, manifest)
}
