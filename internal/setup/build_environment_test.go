package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/shell"
)

func TestGoBuildEnvironmentPinsConfigAndWorkspace(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "cmd", "vrooli-policy-runner")
	if err := os.MkdirAll(nested, 0700); err != nil {
		t.Fatal(err)
	}
	for path, data := range map[string]string{
		filepath.Join(root, "go.mod"):   "module example.com/root\ngo 1.23\n",
		filepath.Join(nested, "go.mod"): "module example.com/nested\ngo 1.23\n",
		filepath.Join(root, "go.work"):  "go 1.23\nuse (\n .\n ./cmd/vrooli-policy-runner\n)\n",
	} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	config := filepath.Join(t.TempDir(), "go-env")
	if err := os.WriteFile(config, []byte("GOPROXY=https://first.invalid\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOENV", config)
	t.Setenv("GOPROXY", "")
	t.Setenv("GOWORK", "")
	frozen, err := freezeGoBuildEnvironment(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, []byte("GOPROXY=https://second.invalid\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOPROXY", "https://third.invalid")
	for _, env := range [][]string{frozen.root, frozen.nested} {
		out, err := shell.Output(shell.Spec{Name: frozen.goCommand, Args: []string{"env", "GOPROXY", "GOWORK"}, Dir: root, Env: env})
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(string(out)); got != "https://first.invalid\n"+filepath.Join(root, "go.work") {
			t.Fatalf("effective environment changed: %q", got)
		}
	}
	if err := frozen.verifyWorkspaces(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.work.sum"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := frozen.verifyWorkspaces(); err == nil {
		t.Fatal("workspace change was accepted")
	}
}

func TestGoBuildInputsRejectUncoveredWorkspaceAndOverlay(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	for path, data := range map[string]string{
		filepath.Join(root, "go.mod"):     "module example.com/root\ngo 1.23\n",
		filepath.Join(external, "go.mod"): "module example.com/external\ngo 1.23\n",
	} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	workspace := filepath.Join(root, "go.work")
	for _, directive := range []string{
		"use " + external,
		"use .\nreplace example.com/external => " + external,
	} {
		if err := os.WriteFile(workspace, []byte("go 1.23\n"+directive+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := validateWorkspace(root, workspace); err == nil || !strings.Contains(err.Error(), "outside cohort input coverage") {
			t.Fatalf("%s accepted: %v", directive, err)
		}
	}
	for _, flags := range [][]string{{"-overlay=/tmp/overlay.json"}, {"-overlay", "/tmp/overlay.json"}} {
		if err := rejectUnsupportedGoFlags(flags); err == nil {
			t.Fatalf("overlay flags accepted: %v", flags)
		}
	}
	if err := rejectUnsupportedGoFlags([]string{"-trimpath", "-p=4"}); err != nil {
		t.Fatal(err)
	}
}

func TestGoBuildInputsRejectEffectiveOverlay(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "cmd", "vrooli-policy-runner")
	if err := os.MkdirAll(nested, 0700); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{root, nested} {
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\ngo 1.23\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GOENV", "off")
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "-overlay=/tmp/overlay.json")
	if _, err := freezeGoBuildEnvironment(root, t.TempDir()); err == nil || !strings.Contains(err.Error(), "overlays") {
		t.Fatalf("effective overlay accepted: %v", err)
	}
}

func TestGoBuildInputsRejectExcludedLocalModules(t *testing.T) {
	root := t.TempDir()
	for _, excluded := range []string{".git", ".vrooli", "build", "dist", "node_modules", "tmp"} {
		path := filepath.Join(root, "packages", excluded, "module")
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
		if _, err := coveredModule(root, root, path); err == nil || !strings.Contains(err.Error(), "excluded") {
			t.Fatalf("excluded path %s accepted: %v", path, err)
		}
	}
}

func TestGoBuildInputsRejectVendorModeAndVendorCreation(t *testing.T) {
	for _, flags := range [][]string{{"-mod=vendor"}, {"-mod", "vendor"}} {
		if err := rejectUnsupportedGoFlags(flags); err == nil {
			t.Fatalf("vendor mode accepted: %v", flags)
		}
	}
	root := t.TempDir()
	nested := filepath.Join(root, "cmd", "vrooli-policy-runner")
	if err := os.MkdirAll(nested, 0700); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{root, nested} {
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\ngo 1.23\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GOENV", "off")
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "")
	env, err := freezeGoBuildEnvironment(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "vendor"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := env.verifyNoVendor(); err == nil {
		t.Fatal("vendor creation accepted")
	}
	if _, err := freezeGoBuildEnvironment(root, t.TempDir()); err == nil {
		t.Fatal("existing vendor directory accepted")
	}
}
