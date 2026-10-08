package runtime

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mutatingCalls are the operations a handler may reach only from Apply.
//
// The list is deliberately about *effects*, not about privilege: running a
// read-only command is fine from Inspect, and several handlers legitimately do
// (getent, systemctl is-active). What may not happen is writing to the host or
// escalating, because the whole consent model rests on Inspect being safe to
// call before the operator has agreed to anything.
var mutatingCalls = map[string]map[string]string{
	"hostreqkit": {
		"RunInstallCommand":               "runs an install command",
		"RunInstallCommandWithProvenance": "runs an install command",
		"RunPackageInstallCommand":        "runs a package install",
		"RunPrivilegedCommand":            "escalates privilege",
		"RunPrivilegedCommandWithOutput":  "escalates privilege",
		"RunPrivilegedCommandWithStdin":   "escalates privilege",
		"RunAsInvokingUser":               "runs a command as the invoking user",
		"RunAsInvokingUserWithInput":      "runs a command as the invoking user",
		"EnsureManagedDir":                "creates a managed directory",
		"InstallManagedContent":           "writes a managed file",
		"InstallManagedExecutable":        "writes a managed executable",
	},
	"os": {
		"WriteFile":   "writes a file",
		"Create":      "creates a file",
		"OpenFile":    "may open a file for writing",
		"Remove":      "deletes a path",
		"RemoveAll":   "deletes a tree",
		"Mkdir":       "creates a directory",
		"MkdirAll":    "creates a directory",
		"Chmod":       "changes permissions",
		"Chown":       "changes ownership",
		"Rename":      "moves a path",
		"Symlink":     "creates a symlink",
		"Link":        "creates a hard link",
		"Truncate":    "truncates a file",
		"WriteString": "writes to a file",
	},
}

// TestInspectNeverMutatesTheHost is the mechanical half of the Handler
// contract. hostreqkit.Handler separates Inspect from Apply, and both
// runtime.InspectSafeguardAt and packages/hostreq document that the read half
// "never calls Apply" -- but until this test that was a convention held by
// handler-by-handler discipline. Onboarding now samples Inspect *before* the
// operator consents to anything and prints "Nothing has been applied yet" on
// the strength of it, so the convention became load-bearing for a consent
// guarantee and needs an enforcer.
//
// The walk is transitive within each handler package: a mutation hidden one
// helper deep is still a mutation.
func TestInspectNeverMutatesTheHost(t *testing.T) {
	for _, group := range []string{"safeguards", "tools"} {
		dir := filepath.Join("..", group)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		checked := 0
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			pkgDir := filepath.Join(dir, entry.Name())
			if _, statErr := os.Stat(filepath.Join(pkgDir, "handler.go")); statErr != nil {
				continue
			}
			checked++
			t.Run(group+"/"+entry.Name(), func(t *testing.T) {
				for _, violation := range inspectViolations(t, pkgDir) {
					t.Errorf("%s", violation)
				}
			})
		}
		// A directory rename or a build-tag change must not silently turn this
		// suite into a no-op that still reports success.
		if checked == 0 {
			t.Fatalf("no handler packages found under %s; the purity check is not running", dir)
		}
	}
}

func inspectViolations(t *testing.T, pkgDir string) []string {
	t.Helper()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgDir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", pkgDir, err)
	}

	// Index every function and method in the package by name so a call into a
	// package-private helper can be followed.
	locals := map[string]*ast.FuncDecl{}
	var inspect *ast.FuncDecl
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				locals[fn.Name.Name] = fn
				if fn.Name.Name == "Inspect" && fn.Recv != nil {
					inspect = fn
				}
			}
		}
	}
	if inspect == nil {
		return nil
	}

	var violations []string
	seen := map[string]bool{}

	var walk func(fn *ast.FuncDecl, trail []string)
	walk = func(fn *ast.FuncDecl, trail []string) {
		if fn == nil || fn.Body == nil || seen[fn.Name.Name] {
			return
		}
		seen[fn.Name.Name] = true

		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch target := call.Fun.(type) {
			case *ast.SelectorExpr:
				// A handler reaching an Apply from Inspect is the exact
				// inversion the interface split exists to prevent. This is
				// checked before the package-qualified cases because the
				// receiver is usually a field or an embedded handler
				// (h.inner.Apply), not a plain package identifier.
				if target.Sel.Name == "Apply" {
					violations = append(violations, describeViolation(fset, call,
						"Apply", "applies the requirement", trail))
					return true
				}
				pkgIdent, isIdent := target.X.(*ast.Ident)
				if !isIdent {
					return true
				}
				if effect, banned := mutatingCalls[pkgIdent.Name][target.Sel.Name]; banned {
					violations = append(violations, describeViolation(fset, call,
						pkgIdent.Name+"."+target.Sel.Name, effect, trail))
				}
			case *ast.Ident:
				if helper, isLocal := locals[target.Name]; isLocal {
					walk(helper, append(trail, target.Name))
				}
			}
			return true
		})
	}

	walk(inspect, nil)
	return violations
}

func describeViolation(fset *token.FileSet, call *ast.CallExpr, name, effect string, trail []string) string {
	position := fset.Position(call.Pos())
	path := "Inspect"
	if len(trail) > 0 {
		path = "Inspect -> " + strings.Join(trail, " -> ")
	}
	return path + " calls " + name + " at " + position.String() + ", which " + effect +
		". Inspect must observe only; move this to Apply."
}

// TestInspectPurityDetectorCatchesViolations proves the guard above can fail.
// A conformance test that scans real code and finds nothing is indistinguishable
// from one whose matcher is broken, so the matcher gets its own fixtures:
// a direct mutation, one hidden behind a package-private helper, and a handler
// calling its own Apply.
func TestInspectPurityDetectorCatchesViolations(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "direct write",
			source: `package probe
import "os"
type handler struct{}
func (h handler) Inspect(a, b any) any { _ = os.WriteFile("/etc/probe", nil, 0o644); return nil }
`,
			want: "os.WriteFile",
		},
		{
			name: "mutation behind a helper",
			source: `package probe
import "os"
type handler struct{}
func (h handler) Inspect(a, b any) any { return probeState() }
func probeState() any { _ = os.MkdirAll("/var/lib/probe", 0o755); return nil }
`,
			want: "os.MkdirAll",
		},
		{
			name: "privilege escalation",
			source: `package probe
import "github.com/vrooli/vrooli/internal/hostreqkit"
type handler struct{}
func (h handler) Inspect(a, b any) any { _ = hostreqkit.RunPrivilegedCommand("", "sysctl", nil, a); return nil }
`,
			want: "hostreqkit.RunPrivilegedCommand",
		},
		{
			name: "inspect calls apply",
			source: `package probe
type handler struct{ inner interface{ Apply(a any) any } }
func (h handler) Inspect(a, b any) any { return h.inner.Apply(a) }
`,
			want: "Apply",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "handler.go"), []byte(testCase.source), 0o600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			violations := inspectViolations(t, dir)
			if len(violations) == 0 {
				t.Fatalf("detector found no violation in a knowingly impure Inspect")
			}
			if !strings.Contains(violations[0], testCase.want) {
				t.Fatalf("violation = %q, want it to name %s", violations[0], testCase.want)
			}
		})
	}
}

// TestInspectPurityDetectorAcceptsReads keeps the guard from being so strict
// that handlers cannot do their job. Inspect is allowed to run commands and
// read files; only writing and escalating are barred.
func TestInspectPurityDetectorAcceptsReads(t *testing.T) {
	dir := t.TempDir()
	source := `package probe
import (
	"os"
	"os/exec"
)
type handler struct{}
func (h handler) Inspect(a, b any) any {
	_, _ = os.ReadFile("/proc/cmdline")
	_, _ = os.Stat("/dev/tpmrm0")
	_, _ = exec.Command("systemctl", "is-active", "probe").Output()
	return nil
}
`
	if err := os.WriteFile(filepath.Join(dir, "handler.go"), []byte(source), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if violations := inspectViolations(t, dir); len(violations) != 0 {
		t.Fatalf("read-only Inspect flagged as impure: %v", violations)
	}
}
