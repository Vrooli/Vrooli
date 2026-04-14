package resources

import (
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testfixture "github.com/vrooli/vrooli/packages/testkit-go/vrooli"
)

func writeResourceCLI(t *testing.T, root, name string) {
	t.Helper()
	testfixture.WriteResourceCLI(t, root, name, "#!/usr/bin/env bash\nexit 0\n")
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
	testfixture.WriteScenarioService(t, root, scenarioName, testfixture.ScenarioServiceManifest(
		scenarioName,
		testfixture.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				resourceName: {
					Enabled:  true,
					Required: true,
				},
			},
		}),
	))
}
