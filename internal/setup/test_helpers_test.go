package setup

import (
	"path/filepath"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/projectstate"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
)

func writeProjectFixture(t *testing.T, root string) scenario.Scenario {
	t.Helper()
	manifest := testscenario.ProjectServiceManifest(
		testscenario.WithPorts(map[string]scenario.Port{
			"api": {EnvVar: "VROOLI_API_PORT", Port: intPtr(8092)},
		}),
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{"redis": {Enabled: false}},
		}),
	)
	return writeProjectFixtureWithServiceManifest(t, root, manifest)
}

func writeProjectFixtureWithServiceManifest(t *testing.T, root string, manifest scenario.ServiceManifest) scenario.Scenario {
	t.Helper()
	testkitgo.WriteRepoContract(t, root, "scenarios")
	testscenario.WriteProjectService(t, root, manifest)
	servicePath := scenario.ProjectServicePath(root)
	if strings.TrimSpace(manifest.Service.Name) == "" {
		manifest.Service.Name = filepath.Base(root)
	}
	return scenario.Scenario{
		Slug:        manifest.Service.Name,
		Path:        root,
		ServicePath: servicePath,
		Manifest:    manifest,
	}
}

func writeOnboardingScenarioFixture(t *testing.T, root string) {
	t.Helper()
	testkitgo.WriteFile(t, scenario.ServicePath(root, onboardingSlug), "{}\n")
}

func writeSetupCompleteMarker(t *testing.T, home, root string) error {
	t.Helper()
	locator, err := projectstate.NewLocator(home, root)
	if err != nil {
		return err
	}
	testkitgo.WriteFile(t, locator.BootstrapCompletePath(), "ok\n")
	return nil
}
