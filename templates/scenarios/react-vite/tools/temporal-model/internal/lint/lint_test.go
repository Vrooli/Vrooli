package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"react-vite-temporal-model/internal/layout"
	"react-vite-temporal-model/internal/testkit"
)

func TestGoLintAcceptsValidTestFile(t *testing.T) {
	root := t.TempDir()
	flow := testkit.MustCompile(t, testkit.ValidRawContract())
	writeFile(t, root, flow.Layout.BaseDir+"/flow_test.go", `package flow

import (
	"testing"

	"`+pkgImport(flow.Layout.BaseDir)+`"
)

func TestExampleFormalReplay(t *testing.T) {
	`+layout.GeneratedDirName+`.RunReplay(t, func(s `+layout.GeneratedDirName+`.Status, e `+layout.GeneratedDirName+`.Event) (`+layout.GeneratedDirName+`.Status, error) {
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
	writeFile(t, root, flow.Layout.BaseDir+"/flow_test.go", `package flow

import (
	"testing"

	_ "`+pkgImport(flow.Layout.BaseDir)+`"
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
	writeFile(t, root, flow.Layout.BaseDir+"/flow_test.go", `package flow

import (
	"testing"

	"`+pkgImport(flow.Layout.BaseDir)+`"
)

func TestExampleFormalReplay(t *testing.T) {
	`+layout.GeneratedDirName+`.RunReplay(t, nil)
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
	writeFile(t, root, flow.Layout.BaseDir+"/flow_test.go", `package flow

import (
	"testing"

	"`+pkgImport(flow.Layout.BaseDir)+`"
)

func TestExampleFormalReplay(t *testing.T) {
	`+layout.GeneratedDirName+`.RunReplay(t, func(s `+layout.GeneratedDirName+`.Status, e `+layout.GeneratedDirName+`.Event) (`+layout.GeneratedDirName+`.Status, error) {
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
	writeFile(t, root, flow.Layout.BaseDir+"/flow.test.ts", `import { runFormalReplay } from "./generated/replay.helper";
import { transitionExample } from "./transition";
import { exampleFormalFixtures } from "./fixtures";

runFormalReplay({ transition: transitionExample, fixtures: exampleFormalFixtures });
`)
	if err := Check(root, flow); err != nil {
		t.Fatalf("unexpected lint failure: %v", err)
	}
}

func TestTSLintRejectsMissingCall(t *testing.T) {
	root := t.TempDir()
	flow := testkit.MustCompile(t, testkit.ValidTypeScriptRawContract())
	writeFile(t, root, flow.Layout.BaseDir+"/flow.test.ts", `import { runFormalReplay } from "./generated/replay.helper";
import { transitionExample } from "./transition";
import { exampleFormalFixtures } from "./fixtures";
`)
	err := Check(root, flow)
	if err == nil || !strings.Contains(err.Error(), "no top-level call") {
		t.Fatalf("expected missing-call lint failure; got %v", err)
	}
}

func TestTSLintRejectsCallInsideBlock(t *testing.T) {
	root := t.TempDir()
	flow := testkit.MustCompile(t, testkit.ValidTypeScriptRawContract())
	writeFile(t, root, flow.Layout.BaseDir+"/flow.test.ts", `import { runFormalReplay } from "./generated/replay.helper";
import { transitionExample } from "./transition";
import { exampleFormalFixtures } from "./fixtures";

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

func pkgImport(baseDir string) string {
	// Mirrors layout.SubpackageImportPath: strips api/ prefix and
	// uses {{SCENARIO_ID}} as the module anchor in templates.
	dir := baseDir + "/" + layout.GeneratedDirName
	dir = strings.TrimPrefix(dir, "api/")
	return "{{SCENARIO_ID}}/" + dir
}
