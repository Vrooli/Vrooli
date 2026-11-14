package deployment

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

type fakeProgramBindingSource struct {
	targets []types.ProgramBindingTarget
	err     error
}

func (f fakeProgramBindingSource) ProgramBindingTargets(string) ([]types.ProgramBindingTarget, string, error) {
	return f.targets, "program-binding", f.err
}

func TestProgramBindingDAGOptionIsAdditive(t *testing.T) {
	root := t.TempDir()
	scenariosDir := filepath.Join(root, "scenarios")
	if err := os.MkdirAll(filepath.Join(scenariosDir, "vrooli-memory", ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := scenariomodel.ServiceManifest{Service: scenariomodel.ServiceMetadata{Name: "vrooli-memory"}}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenariosDir, "vrooli-memory", ".vrooli", "service.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &types.Manifest{Service: scenariomodel.ServiceMetadata{Name: "demo"}}
	without := BuildDependencyNodeList(scenariosDir, "demo", cfg, map[string]struct{}{})
	withoutOptions := BuildDependencyNodeListWithOptions(scenariosDir, "demo", cfg, map[string]struct{}{}, DependencyBuildOptions{})
	if !equalNodes(without, withoutOptions) {
		t.Fatalf("option-off DAG changed: %#v != %#v", without, withoutOptions)
	}
	with := BuildDependencyNodeListWithOptions(scenariosDir, "demo", cfg, map[string]struct{}{}, DependencyBuildOptions{
		IncludeProgramBindings: true,
		ProgramBindings:        fakeProgramBindingSource{targets: []types.ProgramBindingTarget{{Scenario: "vrooli-memory", BindingID: "vrooli-memory/learning/record", Program: "demo.learn"}}},
	})
	if len(with) != 1 || with[0].Name != "vrooli-memory" || with[0].Source != "program-binding" {
		t.Fatalf("program-binding node = %#v", with)
	}
	if with[0].Metadata["program_bindings"].([]string)[0] != "vrooli-memory/learning/record" {
		t.Fatalf("program-binding metadata = %#v", with[0].Metadata)
	}
}

func TestCompositeProgramBindingSourceFallsBackToContracts(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios", "demo", ".vrooli", "program-runtime")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	contract := []byte(`{"name":"demo.learn","bindings":[{"id":"vrooli-memory/learning/record","effect":"write","optional":true}]}`)
	if err := os.WriteFile(filepath.Join(dir, "learn.json"), contract, 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	targets, source, err := (CompositeProgramBindingSource{BaseURL: server.URL, RepoRoot: root}).ProgramBindingTargets("demo")
	if err != nil {
		t.Fatal(err)
	}
	if source != "program-binding-file" || len(targets) != 1 || targets[0].Scenario != "vrooli-memory" || !targets[0].Optional {
		t.Fatalf("fallback = source %q targets %#v", source, targets)
	}
}

func TestProgramBindingSourceErrorDoesNotChangeManifestOnlyDAG(t *testing.T) {
	cfg := &types.Manifest{}
	got := BuildDependencyNodeListWithOptions(t.TempDir(), "demo", cfg, nil, DependencyBuildOptions{IncludeProgramBindings: true, ProgramBindings: fakeProgramBindingSource{err: errors.New("transport")}})
	if len(got) != 0 {
		t.Fatalf("error source changed DAG = %#v", got)
	}
}

func equalNodes(left, right []types.DeploymentDependencyNode) bool {
	return jsonEqual(left, right)
}

func jsonEqual(left, right any) bool {
	l, _ := json.Marshal(left)
	r, _ := json.Marshal(right)
	return string(l) == string(r)
}
