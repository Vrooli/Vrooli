package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/resources/catalog"
)

func TestStatusReportsEffectiveCapacityTunableAndSource(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	stateRoot := t.TempDir()
	stateDir := filepath.Join(stateRoot, ".vrooli")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	state := []byte(`{"version":"1.0.0","resources":{"ollama":{"capacity":{"tunables":{"num_parallel":6}}}}}`)
	if err := os.WriteFile(filepath.Join(stateDir, "operator-state.json"), state, 0o600); err != nil {
		t.Fatal(err)
	}
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
	stateDir := filepath.Join(stateRoot, ".vrooli")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	state := []byte(`{"version":"1.0.0","resources":{"ollama":{"capacity":{"tunables":{"num_parallel":9}}}}}`)
	if err := os.WriteFile(filepath.Join(stateDir, "operator-state.json"), state, 0o600); err != nil {
		t.Fatal(err)
	}
	controller := &Controller{Root: stateRoot}
	_, err = controller.withEffectiveCapacity(Status{Resource: catalog.Resource{
		Name: "ollama", ManifestPath: filepath.Join(repoRoot, "resources", "ollama", "resource.json"),
	}})
	if err == nil {
		t.Fatal("withEffectiveCapacity() accepted num_parallel above manifest maximum")
	}
}
