package lifecycle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

import (
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/values"
)

func TestLifecycleStepEnvCarriesToolchainFloor(t *testing.T) {
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOMAXPROCS", "")
	env := lifecycleStepEnv(phasesSetup, map[string]string{"FOO": "bar"})
	goflags := values.EnvValue(env, "GOFLAGS")
	if !strings.HasPrefix(goflags, "-mod=mod ") || !strings.Contains(goflags, "-p=") {
		t.Fatalf("GOFLAGS = %q, want the inherited flag composed with a -p width", goflags)
	}
	if values.EnvValue(env, "GOMAXPROCS") == "" {
		t.Fatalf("GOMAXPROCS is not set: %v", env)
	}
	if values.EnvValue(env, "FOO") != "bar" {
		t.Fatalf("override lost: %v", env)
	}
	tempRoot := t.TempDir()
	t.Setenv("VROOLI_HOME", tempRoot)
	t.Setenv("GOTMPDIR", "")
	env = lifecycleStepEnv(phasesSetup, nil)
	wantTemp := filepath.Join(tempRoot, "tmp", "go-work")
	if values.EnvValue(env, "GOTMPDIR") != wantTemp {
		t.Fatalf("GOTMPDIR = %q, want %q", values.EnvValue(env, "GOTMPDIR"), wantTemp)
	}
	if _, err := os.Stat(wantTemp); err != nil {
		t.Fatalf("GOTMPDIR was not created: %v", err)
	}
	t.Setenv("GOTMPDIR", filepath.Join(t.TempDir(), "operator-go-work"))
	if got := values.EnvValue(lifecycleStepEnv(phasesSetup, nil), "GOTMPDIR"); got != os.Getenv("GOTMPDIR") {
		t.Fatalf("explicit GOTMPDIR = %q, want %q", got, os.Getenv("GOTMPDIR"))
	}
}

func TestLifecycleStepEnvPrioritizesSelectedScenarioNode(t *testing.T) {
	nodeBin := t.TempDir()
	basePath := filepath.Join(t.TempDir(), "base-bin")
	env := lifecycleStepEnv(phasesSetup, map[string]string{
		"PATH":             basePath,
		nodeRuntimePathEnv: nodeBin,
	})
	if got := filepath.SplitList(values.EnvValue(env, "PATH"))[0]; got != nodeBin {
		t.Fatalf("PATH first entry = %q, want selected Node bin %q", got, nodeBin)
	}
	if got := values.EnvValue(env, nodeRuntimePathEnv); got != "" {
		t.Fatalf("internal Node path leaked to lifecycle child environment: %q", got)
	}
}

// Setup runs pnpm with CI=true, under which a store that differs from the one
// node_modules was linked against makes pnpm silently recreate the whole
// directory. A setup install follows the recorded store instead.
func TestSetupPnpmInstallFollowsTheStoreNodeModulesIsLinkedAgainst(t *testing.T) {
	bin, root := t.TempDir(), t.TempDir()
	log := filepath.Join(root, "pnpm.log")
	script := "#!/bin/sh\n" +
		"base=${npm_config_store_dir:-/user/config/store/v3}\n" +
		"if [ \"$1\" = store ]; then echo \"$base/v10\"; exit 0; fi\n" +
		"echo \"$@ store=$npm_config_store_dir\" >> " + log + "\n"
	if err := os.WriteFile(filepath.Join(bin, "pnpm"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	ui := filepath.Join(root, "ui")
	if err := os.MkdirAll(filepath.Join(ui, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	modules := "{\n  \"storeDir\": \"/home/u/.local/share/pnpm/store/v10\"\n}\n"
	if err := os.WriteFile(filepath.Join(ui, "node_modules", ".modules.yaml"), []byte(modules), 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	step := scenario.PhaseStep{Name: "install-ui", Exec: []string{filepath.Join(bin, "pnpm"), "install", "--ignore-workspace"}, CWD: "ui"}
	item := scenario.Scenario{Slug: "demo", Path: root}
	env := map[string]string{"PATH": bin + string(os.PathListSeparator) + os.Getenv("PATH")}
	if err := (&Runner{}).runForegroundStep(context.Background(), item, phasesSetup, step, env, &out); err != nil {
		t.Fatalf("runForegroundStep: %v\n%s", err, out.String())
	}
	logged, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("pnpm did not run: %v\n%s", err, out.String())
	}
	if !strings.Contains(string(logged), "install --ignore-workspace store=/home/u/.local/share/pnpm/store") {
		t.Fatalf("pnpm ran against %q, want the recorded store's base", logged)
	}
	if !strings.Contains(out.String(), "reusing it so pnpm does not recreate node_modules") {
		t.Fatalf("the step log must say which store was reused: %q", out.String())
	}
}
