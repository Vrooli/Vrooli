package lifecycle

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGoModuleDriftDiagnosticsKeepsDriftAndDropsToolchainAdvice(t *testing.T) {
	output := strings.Join([]string{
		"../leaf/leaf.go:3:8: missing go.sum entry for module providing package golang.org/x/mod/module (imported by example.com/leaf); to add:",
		"\tgo get example.com/leaf@v0.0.0",
		"main.go:4:2: package golang.org/x/mod/semver imported from implicitly required module; to add missing requirements, run:",
		"\tgo get golang.org/x/mod@v0.41.0",
		"go: updates to go.mod needed; to update it:",
		"\tgo mod tidy",
		"main.go:9:2: undefined: unrelated",
	}, "\n")
	got := goModuleDriftDiagnostics(output)
	want := []string{
		"../leaf/leaf.go:3:8: missing go.sum entry for module providing package golang.org/x/mod/module (imported by example.com/leaf)",
		"main.go:4:2: package golang.org/x/mod/semver imported from implicitly required module",
		"go: updates to go.mod needed",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diagnostics = %#v, want %#v", got, want)
	}
	if got := goModuleDriftDiagnostics("main.go:9:2: undefined: unrelated\n"); len(got) != 0 {
		t.Fatalf("a compile error is not module drift: %#v", got)
	}
}

// goModuleDriftFixture is the incident shape: a shared module imports a
// third-party package the consumer's go.mod/go.sum do not record.
func goModuleDriftFixture(t *testing.T, consumerSource string) string {
	t.Helper()
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOFLAGS", "")
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Version}}", "golang.org/x/mod").Output()
	if err != nil {
		t.Skipf("resolve golang.org/x/mod from the repository module: %v", err)
	}
	version := strings.TrimSpace(string(out))
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("leaf/go.mod", "module example.com/leaf\n\ngo 1.25.0\n\nrequire golang.org/x/mod "+version+"\n")
	write("leaf/leaf.go", "package leaf\n\nimport \"golang.org/x/mod/module\"\n\ntype V = module.Version\n")
	write("api/go.mod", "module demo/api\n\ngo 1.25.0\n\nrequire example.com/leaf v0.0.0\n\nreplace example.com/leaf => ../leaf\n")
	write("api/main.go", consumerSource)
	return filepath.Join(root, "api")
}

func TestDiagnoseGoModuleDriftNamesSharedModuleRequirementGap(t *testing.T) {
	dir := goModuleDriftFixture(t, "package main\n\nimport \"example.com/leaf\"\n\nfunc main() { _ = leaf.V{} }\n")
	got := diagnoseGoModuleDrift(context.Background(), dir)
	if len(got) != 1 || !strings.Contains(got[0], "missing go.sum entry") || !strings.Contains(got[0], "golang.org/x/mod/module") {
		t.Fatalf("diagnostics = %#v, want the missing shared-module checksum", got)
	}
}

func TestDiagnoseGoModuleDriftIgnoresBuildsThatFailForOtherReasons(t *testing.T) {
	dir := goModuleDriftFixture(t, "package main\n\nfunc main() { undefinedSymbol() }\n")
	if got := diagnoseGoModuleDrift(context.Background(), dir); len(got) != 0 {
		t.Fatalf("a module whose metadata is complete has no drift: %#v", got)
	}
}
