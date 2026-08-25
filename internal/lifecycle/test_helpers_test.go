package lifecycle

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
)

var lifecycleFixturePorts = struct {
	sync.Mutex
	used map[string]map[int]struct{}
}{used: make(map[string]map[int]struct{})}

func writeLifecycleFixture(t *testing.T, root, name string) {
	t.Helper()
	writeLifecycleFixtureManifest(t, root, lifecycleFixtureManifest(name))
}

func cleanupRunner(t *testing.T, runner *Runner, name string, opts StopOptions) {
	t.Helper()
	t.Cleanup(func() {
		if err := runner.Stop(name, opts); err != nil {
			t.Errorf("Stop(%q, %#v) during cleanup: %v", name, opts, err)
		}
	})
}

func writeLifecycleFixtureManifest(t *testing.T, root string, manifest scenario.ServiceManifest) {
	t.Helper()

	if strings.TrimSpace(manifest.Service.Name) == "" {
		t.Fatalf("fixture manifest is missing service name")
	}
	// The real lifecycle allocator must never be coupled to a shared fixed
	// port. Allocate the fixture's API port from the OS and release it before
	// the runner starts; the runner then exercises its normal declared-port
	// path with an isolated value.
	if apiPort, ok := manifest.Ports["api"]; ok && apiPort.Port == nil && apiPort.Range == "22000-22010" {
		var listener net.Listener
		var err error
		lifecycleFixturePorts.Lock()
		defer lifecycleFixturePorts.Unlock()
		used := lifecycleFixturePorts.used[root]
		if used == nil {
			used = make(map[int]struct{})
			lifecycleFixturePorts.used[root] = used
		}
		// Port 0 is OS allocated, but Linux places it in the ephemeral band,
		// which the repository deliberately rejects for scenario APIs. Ask the
		// OS to bind candidates in the canonical API band instead.
		for port := 15000; port <= 19999; port++ {
			if _, alreadyUsed := used[port]; alreadyUsed {
				continue
			}
			listener, err = net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
			if err == nil {
				used[port] = struct{}{}
				break
			}
		}
		if err != nil || listener == nil {
			t.Fatalf("allocate lifecycle fixture port in API band: %v", err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		if err := listener.Close(); err != nil {
			t.Fatalf("release lifecycle fixture port %d: %v", port, err)
		}
		apiPort.Range = ""
		apiPort.Port = &port
		manifest.Ports["api"] = apiPort
		if health := manifest.Lifecycle.Health; health != nil {
			for i := range health.Checks {
				health.Checks[i].Target = strings.ReplaceAll(health.Checks[i].Target, "${API_PORT}", strconv.Itoa(port))
			}
		}
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
