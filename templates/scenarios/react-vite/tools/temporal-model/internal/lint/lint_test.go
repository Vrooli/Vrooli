package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"react-vite-temporal-model/internal/testkit"
)

func TestGoLintAcceptsValidTestFile(t *testing.T) {
	root := t.TempDir()
	flow := testkit.MustCompile(t, testkit.ValidRawContract())
	writeFile(t, root, flow.Layout.BaseDir+"/example_workflow_test.go", `package example

import (
	"testing"

	"`+pkgImport(flow.Layout.BaseDir, flow.Layout.FolderName)+`"
)

func TestExampleFormalReplay(t *testing.T) {
	`+flow.Layout.FolderName+`.RunReplay(t, func(s `+flow.Layout.FolderName+`.Status, e `+flow.Layout.FolderName+`.Event) (`+flow.Layout.FolderName+`.Status, error) {
		return s, nil
	})
}
`)
	if err := Check(root, flow); err != nil {
		t.Fatalf("unexpected lint failure: %v", err)
	}
}

func TestGoLintRejectsMissingCall(t *testing.T) {
	root := t.TempDir()
	flow := testkit.MustCompile(t, testkit.ValidRawContract())
	writeFile(t, root, flow.Layout.BaseDir+"/example_workflow_test.go", `package example

import (
	"testing"

	_ "`+pkgImport(flow.Layout.BaseDir, flow.Layout.FolderName)+`"
)

func TestNothing(t *testing.T) {}
`)
	err := Check(root, flow)
	if err == nil || !strings.Contains(err.Error(), "no Test* function") {
		t.Fatalf("expected missing-call lint failure; got %v", err)
	}
}

func TestGoLintRejectsNilTransition(t *testing.T) {
	root := t.TempDir()
	flow := testkit.MustCompile(t, testkit.ValidRawContract())
	writeFile(t, root, flow.Layout.BaseDir+"/example_workflow_test.go", `package example

import (
	"testing"

	"`+pkgImport(flow.Layout.BaseDir, flow.Layout.FolderName)+`"
)

func TestExampleFormalReplay(t *testing.T) {
	`+flow.Layout.FolderName+`.RunReplay(t, nil)
}
`)
	err := Check(root, flow)
	if err == nil || !strings.Contains(err.Error(), "nil is not a valid transition") {
		t.Fatalf("expected nil-transition lint failure; got %v", err)
	}
}

func TestGoLintRejectsEmptyTransitionBody(t *testing.T) {
	root := t.TempDir()
	flow := testkit.MustCompile(t, testkit.ValidRawContract())
	writeFile(t, root, flow.Layout.BaseDir+"/example_workflow_test.go", `package example

import (
	"testing"

	"`+pkgImport(flow.Layout.BaseDir, flow.Layout.FolderName)+`"
)

func TestExampleFormalReplay(t *testing.T) {
	`+flow.Layout.FolderName+`.RunReplay(t, func(s `+flow.Layout.FolderName+`.Status, e `+flow.Layout.FolderName+`.Event) (`+flow.Layout.FolderName+`.Status, error) {
	})
}
`)
	err := Check(root, flow)
	if err == nil || !strings.Contains(err.Error(), "empty body") {
		t.Fatalf("expected empty-body lint failure; got %v", err)
	}
}

func TestTSLintAcceptsValidTestFile(t *testing.T) {
	root := t.TempDir()
	flow := testkit.MustCompile(t, testkit.ValidTypeScriptRawContract())
	writeFile(t, root, flow.Layout.BaseDir+"/ExampleWorkflow.test.ts", `import { runFormalReplay } from "./generated/`+flow.Layout.FolderName+`/replay.helper";
import { transitionExample } from "./ExampleWorkflow";
import { exampleFormalFixtures } from "./ExampleWorkflow.fixtures";

runFormalReplay({ transition: transitionExample, fixtures: exampleFormalFixtures });
`)
	if err := Check(root, flow); err != nil {
		t.Fatalf("unexpected lint failure: %v", err)
	}
}

func TestTSLintRejectsMissingCall(t *testing.T) {
	root := t.TempDir()
	flow := testkit.MustCompile(t, testkit.ValidTypeScriptRawContract())
	writeFile(t, root, flow.Layout.BaseDir+"/ExampleWorkflow.test.ts", `import { runFormalReplay } from "./generated/`+flow.Layout.FolderName+`/replay.helper";
import { transitionExample } from "./ExampleWorkflow";
import { exampleFormalFixtures } from "./ExampleWorkflow.fixtures";
`)
	err := Check(root, flow)
	if err == nil || !strings.Contains(err.Error(), "no top-level call") {
		t.Fatalf("expected missing-call lint failure; got %v", err)
	}
}

func TestTSLintRejectsCallInsideBlock(t *testing.T) {
	root := t.TempDir()
	flow := testkit.MustCompile(t, testkit.ValidTypeScriptRawContract())
	writeFile(t, root, flow.Layout.BaseDir+"/ExampleWorkflow.test.ts", `import { runFormalReplay } from "./generated/`+flow.Layout.FolderName+`/replay.helper";
import { transitionExample } from "./ExampleWorkflow";
import { exampleFormalFixtures } from "./ExampleWorkflow.fixtures";

if (false) {
  runFormalReplay({ transition: transitionExample, fixtures: exampleFormalFixtures });
}
`)
	err := Check(root, flow)
	if err == nil || !strings.Contains(err.Error(), "no top-level call") {
		t.Fatalf("expected nested-call lint failure; got %v", err)
	}
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

func pkgImport(baseDir string, folder string) string {
	// Mirrors layout.SubpackageImportPath: strips api/ prefix and
	// uses {{SCENARIO_ID}} as the module anchor in templates. In
	// these unit tests we only care that the import string matches
	// the one the lint computes; the test parser does not resolve
	// the import.
	dir := baseDir + "/generated/" + folder
	dir = strings.TrimPrefix(dir, "api/")
	return "{{SCENARIO_ID}}/" + dir
}
