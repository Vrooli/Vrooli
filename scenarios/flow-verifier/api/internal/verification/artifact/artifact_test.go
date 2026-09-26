package artifact

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"flow-verifier/internal/testkit"
	"flow-verifier/internal/verification/quint"
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
	runner := testkit.FakeRunner{}
	flow := testkit.MustCompile(t, testkit.ValidRawContract())
	writeFile(t, root, flow.ContractPath, "{}")
	built, err := Build(context.Background(), flow, BuildOptions{
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
	if got := strings.Join(built.GeneratedChecks, ","); got != GeneratedCheckTransitionTable {
		t.Fatalf("generated checks = %q, want transitionTable", got)
	}
	if !built.Coverage.TransitionMatrixComplete {
		t.Fatalf("transition matrix should be complete")
	}
	if !built.Coverage.NamedTraces.AllStatesCovered || !built.Coverage.NamedTraces.AllEventsCovered {
		t.Fatalf("named traces should cover all states and events: %+v", built.Coverage.NamedTraces)
	}
	if built.Coverage.GeneratedTraces.AllPairsCovered == nil || *built.Coverage.GeneratedTraces.AllPairsCovered {
		t.Fatalf("generated trace pair coverage should not be inferred from matrix completeness: %+v", built.Coverage.GeneratedTraces)
	}
	if got := strings.Join(built.Coverage.GeneratedTraces.CoveredPairs, ","); got != "idle/start" {
		t.Fatalf("generated trace covered pairs = %q, want idle/start", got)
	}
}

func TestBuildReportsRunnerFailures(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, GeneratorPath), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, filepath.Join(GeneratorPath, "go.mod"), "module test\n")
	flow := testkit.MustCompile(t, testkit.ValidRawContract())
	writeFile(t, root, flow.ContractPath, "{}")
	_, err := Build(context.Background(), flow, BuildOptions{
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

type failingRunner struct{}

func (f failingRunner) Run(_ context.Context, command quint.Command) (quint.Result, error) {
	if len(command.Args) >= 2 && command.Args[1] == "test" {
		return quint.Result{Stdout: "stdout", Stderr: "stderr"}, os.ErrPermission
	}
	return testkit.FakeRunner{}.Run(context.Background(), command)
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
