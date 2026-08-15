package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
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

// writePlaintextUserSecrets drops a legacy plaintext secrets document at the
// retired location, so a test can prove the resolver ignores it.
//
// The path is written literally rather than resolved through a helper. These
// tests assert that nothing reads this file, so they must keep naming it even
// after every reader — and the contract entry that described it — is gone.
func writePlaintextUserSecrets(t *testing.T, home string, values map[string]string) {
	t.Helper()
	dir := filepath.Join(home, ".vrooli")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("create retired secrets dir: %v", err)
	}
	document := map[string]any{"_metadata": map[string]string{"managed_by": "test"}}
	for key, value := range values {
		document[key] = value
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode retired secrets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secrets.json"), encoded, 0o600); err != nil {
		t.Fatalf("write retired secrets: %v", err)
	}
}
