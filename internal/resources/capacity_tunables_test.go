package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/internal/resources/catalog"
)

func writeCapacityOperatorState(t *testing.T, root string, state []byte) string {
	t.Helper()
	t.Setenv("VROOLI_STORAGE_ROOT", root)
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		t.Fatalf("create storage resolver: %v", err)
	}
	paths, err := resolver.Resolve(storage.Options{ScenarioID: "vrooli-onboarding"})
	if err != nil {
		t.Fatalf("resolve operator state: %v", err)
	}
	if err := os.MkdirAll(paths.StateDir, 0o700); err != nil {
		t.Fatalf("create state directory: %v", err)
	}
	path := filepath.Join(paths.StateDir, operatorstate.StateFile)
	if err := os.WriteFile(path, state, 0o600); err != nil {
		t.Fatalf("write operator state: %v", err)
	}
	return path
}

func TestStatusReportsEffectiveCapacityTunableAndSource(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	stateRoot := t.TempDir()
	state := []byte(`{"version":"1.0.0","resources":{"ollama":{"capacity":{"tunables":{"num_parallel":6}}}}}`)
	writeCapacityOperatorState(t, stateRoot, state)
	controller := &Controller{Root: stateRoot}
	status := Status{Resource: catalog.Resource{
		Name: "ollama", ManifestPath: filepath.Join(repoRoot, "resources", "ollama", "resource.json"),
	}}
	decorated, err := controller.withEffectiveCapacity(status)
	if err != nil {
		t.Fatalf("withEffectiveCapacity() error = %v", err)
	}
	var raw struct {
		Capacity struct {
			Tunables map[string]effectiveCapacityTunable `json:"tunables"`
		} `json:"capacity"`
	}
	if err := json.Unmarshal(decorated.Raw, &raw); err != nil {
		t.Fatal(err)
	}
	got := raw.Capacity.Tunables["num_parallel"]
	if got.Value != float64(6) || got.Source != "operator_state" {
		t.Fatalf("effective tunable = %+v, want value 6 from operator_state", got)
	}
}

func TestStatusWithoutAccelerationDoesNotPanic(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	controller := &Controller{Root: repoRoot}
	status, err := controller.withEffectiveCapacity(Status{Resource: catalog.Resource{
		Name: "postgres", ManifestPath: filepath.Join(repoRoot, "resources", "postgres", "resource.json"),
	}})
	if err != nil {
		t.Fatalf("withEffectiveCapacity() error = %v", err)
	}
	if status.Raw != nil {
		t.Fatalf("status.Raw = %s, want unchanged", status.Raw)
	}
}

func TestStatusRejectsCapacityOverrideOutsideManifestBounds(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	stateRoot := t.TempDir()
	state := []byte(`{"version":"1.0.0","resources":{"ollama":{"capacity":{"tunables":{"num_parallel":9}}}}}`)
	writeCapacityOperatorState(t, stateRoot, state)
	controller := &Controller{Root: stateRoot}
	_, err = controller.withEffectiveCapacity(Status{Resource: catalog.Resource{
		Name: "ollama", ManifestPath: filepath.Join(repoRoot, "resources", "ollama", "resource.json"),
	}})
	if err == nil {
		t.Fatal("withEffectiveCapacity() accepted num_parallel above manifest maximum")
	}
}
