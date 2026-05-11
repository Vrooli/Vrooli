package artifact

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/quint"
)

func TestCanonicalJSONIsDeterministic(t *testing.T) {
	value := map[string]any{"b": 2, "a": []string{"x", "y"}}
	first, err := CanonicalJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CanonicalJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("canonical JSON was not deterministic")
	}
}

func TestBuildUsesRunnerAndNormalizesGeneratedTraces(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, GeneratorPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, GeneratorPath, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := validContract()
	writeFile(t, root, c.ContractPath, "{}")
	runner := fakeRunner{}
	built, err := Build(context.Background(), c, BuildOptions{
		Root:         root,
		Rendered:     "module Example {}",
		QuintVersion: "0.32.0",
		RunQuint:     true,
		Runner:       runner,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(built.GeneratedTraces) != 1 {
		t.Fatalf("generated traces = %d, want 1", len(built.GeneratedTraces))
	}
}

func TestBuildReportsRunnerFailures(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, GeneratorPath), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, filepath.Join(GeneratorPath, "go.mod"), "module test\n")
	c := validContract()
	writeFile(t, root, c.ContractPath, "{}")
	_, err := Build(context.Background(), c, BuildOptions{
		Root:         root,
		Rendered:     "module Example {}",
		QuintVersion: "0.32.0",
		RunQuint:     true,
		Runner:       failingRunner{},
	})
	if err == nil || !strings.Contains(err.Error(), "command failed: quint test") || !strings.Contains(err.Error(), "stderr") {
		t.Fatalf("Build() error = %v", err)
	}
}

type fakeRunner struct{}

func (fakeRunner) Run(_ context.Context, command quint.Command) (quint.Result, error) {
	if len(command.Args) >= 2 && command.Args[1] == "run" {
		pattern := command.Args[len(command.Args)-1]
		path := strings.Replace(pattern, "{seq}", "1", 1)
		body := `{"states":[{"status":{"tag":"Idle"},"event":{"tag":"Start"},"rejected":false},{"status":{"tag":"Busy"},"event":{"tag":"Start"},"rejected":false}]}`
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return quint.Result{}, err
		}
	}
	return quint.Result{Stdout: "ok"}, nil
}

type failingRunner struct{}

func (f failingRunner) Run(_ context.Context, command quint.Command) (quint.Result, error) {
	if len(command.Args) >= 2 && command.Args[1] == "test" {
		return quint.Result{Stdout: "stdout", Stderr: "stderr"}, os.ErrPermission
	}
	return fakeRunner{}.Run(context.Background(), command)
}

func validContract() contract.Contract {
	c := contract.Contract{
		SchemaVersion: 2,
		FlowID:        "example.flow",
		Domain:        "example",
		Description:   "Example.",
		ContractPath:  "example.flow.json",
		Model: contract.Model{
			Module:     "Example",
			Seed:       "1",
			MaxSteps:   2,
			TraceCount: 1,
			Verify:     contract.Verify{Invariants: []string{"TypeOK"}},
		},
		Outputs:            contract.Outputs{ModelPath: "model.qnt", ArtifactPath: "model.formal.generated.json"},
		States:             []contract.State{{ID: "idle", Quint: "Idle", Initial: true}, {ID: "busy", Quint: "Busy"}},
		Events:             []contract.Event{{ID: "start", Quint: "Start"}},
		TransitionDefaults: contract.TransitionDefaults{Invalid: &contract.DefaultTransition{To: "self", WantError: true}},
		Transitions:        []contract.Transition{{From: contract.StringList{"idle"}, Event: contract.StringList{"start"}, To: "busy"}},
		Invariants:         []contract.Invariant{{ID: "type_ok", Quint: "TypeOK", Description: "Type OK."}},
		Traces:             []contract.Trace{{Name: "success", Initial: "idle", Steps: []contract.TraceStep{{Event: "start", Want: "busy"}}}},
	}
	if err := contract.ValidateAndExpand(&c); err != nil {
		panic(err)
	}
	return c
}

func writeFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
