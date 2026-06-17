package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quality-health/internal/surfaces"
)

func writeShell(t *testing.T, root, rel, content string) string {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func withShellSeams(t *testing.T, resolve func([]string) (string, bool), run func(string, ...string) (string, bool)) {
	t.Helper()
	or, ru := shellResolve, shellRun
	t.Cleanup(func() { shellResolve, shellRun = or, ru })
	shellResolve, shellRun = resolve, run
}

func TestShellSyntaxNoScriptsNoFindings(t *testing.T) {
	root := t.TempDir()
	withShellSeams(t,
		func([]string) (string, bool) {
			t.Fatal("resolve should not be called when no scripts")
			return "", false
		},
		func(string, ...string) (string, bool) { t.Fatal("run should not be called"); return "", true },
	)
	if got := evalShellSyntax(EvalContext{Inventory: surfaces.Inventory{RootPath: root}}); got != nil {
		t.Fatalf("expected nil findings, got %+v", got)
	}
}

func TestShellSyntaxCleanScript(t *testing.T) {
	root := t.TempDir()
	writeShell(t, root, "cli/run.sh", "#!/usr/bin/env bash\necho ok\n")
	withShellSeams(t,
		func(c []string) (string, bool) {
			if c[0] == "bash" {
				return "bash", true
			}
			return "", false // no shellcheck
		},
		func(string, ...string) (string, bool) { return "", true },
	)
	if got := evalShellSyntax(EvalContext{Inventory: surfaces.Inventory{RootPath: root}}); got != nil {
		t.Fatalf("clean script should produce no findings, got %+v", got)
	}
}

func TestShellSyntaxErrorProducesFinding(t *testing.T) {
	root := t.TempDir()
	bad := writeShell(t, root, "lib/bad.sh", "if [ ; then\n")
	writeShell(t, root, "lib/good.sh", "echo hi\n")
	withShellSeams(t,
		func(c []string) (string, bool) {
			if c[0] == "bash" {
				return "bash", true
			}
			return "", false
		},
		func(name string, args ...string) (string, bool) {
			// Last arg is the file path.
			file := args[len(args)-1]
			if file == bad {
				return "bad.sh: line 1: syntax error near unexpected token", false
			}
			return "", true
		},
	)
	got := evalShellSyntax(EvalContext{Inventory: surfaces.Inventory{RootPath: root}})
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %+v", got)
	}
	if got[0].Severity != "warning" {
		t.Errorf("severity = %q, want warning", got[0].Severity)
	}
	if !strings.Contains(got[0].Evidence, "bad.sh") {
		t.Errorf("evidence should name the failing file: %q", got[0].Evidence)
	}
}

func TestShellSyntaxDegradesWhenNoBash(t *testing.T) {
	root := t.TempDir()
	writeShell(t, root, "cli/run.sh", "echo hi\n")
	withShellSeams(t,
		func([]string) (string, bool) { return "", false }, // no bash, no shellcheck
		func(string, ...string) (string, bool) {
			t.Fatal("run should not be called without bash")
			return "", true
		},
	)
	got := evalShellSyntax(EvalContext{Inventory: surfaces.Inventory{RootPath: root}})
	if len(got) != 1 || got[0].Severity != "info" {
		t.Fatalf("expected one info degrade finding, got %+v", got)
	}
	if !strings.Contains(strings.ToLower(got[0].Message), "no bash") {
		t.Errorf("degrade message should mention missing bash: %q", got[0].Message)
	}
}
