package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
)

func writeLifecycleFixture(t *testing.T, root, name string) {
	t.Helper()
	writeLifecycleFixtureManifest(t, root, lifecycleFixtureManifest(name))
}

func writeLifecycleFixtureManifest(t *testing.T, root string, manifest scenario.ServiceManifest) {
	t.Helper()

	if strings.TrimSpace(manifest.Service.Name) == "" {
		t.Fatalf("fixture manifest is missing service name")
	}

	testresource.WritePortRegistry(t, root, nil)
	// Smoke tests construct a real Runner, which loads the root manifest during
	// host-requirement resolution. Write an empty project manifest so resolution
	// succeeds with an empty scope; tests that exercise host requirements can
	// overwrite this fixture.
	testscenario.WriteProjectService(t, root, testscenario.ProjectServiceManifest())
	testscenario.WriteScenarioService(t, root, manifest.Service.Name, manifest)
	for _, component := range manifest.Components {
		if component.Build.Kind != "go_module" || component.Build.Output != "api/mock-api" {
			continue
		}
		apiDir := filepath.Join(root, "scenarios", manifest.Service.Name, "api")
		if err := os.MkdirAll(apiDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module fixture\n\ngo 1.25.0\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		mainSource := `package main
import (
 "fmt"
 "net/http"
 "os"
)
func main() {
 http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, "ok") })
 _ = http.ListenAndServe("127.0.0.1:"+os.Getenv("API_PORT"), nil)
}
`
		if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte(mainSource), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func lifecycleFixtureManifest(name string) scenario.ServiceManifest {
	return scenario.ServiceManifest{
		Version: "1.0.0",
		Service: scenario.ServiceMetadata{
			Name:        name,
			DisplayName: "Lifecycle " + name,
			Description: "Lifecycle validation fixture",
			Version:     "0.1.0",
		},
		Ports: map[string]scenario.Port{
			"api": {
				EnvVar: "API_PORT",
				Range:  "22000-22010",
			},
		},
		Components: map[string]scenario.Component{
			"api": {
				Role:  "api",
				Build: scenario.ComponentBuild{Kind: "go_module", Dir: "api", Output: "api/mock-api"},
				Run:   scenario.ComponentRun{Argv: []string{"{{bin.api}}"}, CWD: "api", Port: "api"},
			},
		},
		Lifecycle: scenario.Lifecycle{
			Version: "2.0.0",
			Health: &scenario.HealthConfig{
				Checks: []scenario.HealthCheck{
					{
						Name:     "api",
						Type:     "http",
						Target:   "http://127.0.0.1:${API_PORT}/health",
						Critical: true,
						Timeout:  1000,
					},
				},
				StartupGracePeriod: 250,
				Timeout:            5000,
				Interval:           250,
			},
			Setup:   scenario.Phase{},
			Develop: scenario.Phase{},
		},
	}
}
