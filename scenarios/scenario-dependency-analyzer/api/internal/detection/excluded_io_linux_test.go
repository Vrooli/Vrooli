package detection

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

// An excluded input must never be opened. FIFOs make an accidental read
// observable without allocating a multi-gigabyte runtime-data fixture.
func TestScannersDoNotReadExcludedInputs(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"data/events.json", "docs/retained.go", "api/worker_test.go"} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mkfifo(path, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "api", "main.go"), []byte("// resource-redis\n// vrooli scenario start peer\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	catalog := &catalogManager{loaded: true, knownScenarios: map[string]struct{}{"peer": {}}, knownResources: map[string]struct{}{"redis": {}}}
	for name, scan := range map[string]func() ([]types.ScenarioDependency, error){
		"resources": func() ([]types.ScenarioDependency, error) {
			return newResourceScanner(catalog).scan(root, "owner", nil)
		},
		"scenarios": func() ([]types.ScenarioDependency, error) {
			return newScenarioScanner(catalog).scanDependencies(root, "owner")
		},
	} {
		t.Run(name, func(t *testing.T) {
			done := make(chan struct{})
			var deps []types.ScenarioDependency
			var err error
			go func() {
				deps, err = scan()
				close(done)
			}()
			select {
			case <-done:
				if err != nil || len(deps) != 1 {
					t.Fatalf("source detection must survive exclusion pruning: %+v %v", deps, err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("scanner opened an excluded input")
			}
		})
	}
}
