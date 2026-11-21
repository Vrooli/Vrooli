package resources

import (
	"context"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
)

func TestRunSourceBuildUsesSharedGoInstaller(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	controller := NewController(root, home)
	manifest := ResourceManifest{
		Name: "fixture",
		CLI: &scenario.CLIConfig{
			Command:     "resource-fixture",
			Adapter:     scenario.CLIAdapterConfig{Kind: "go_module", ModuleDir: "cli"},
			SourceBuild: &scenario.CLISourceBuildConfig{Kind: "go_module"},
			Freshness:   &scenario.CLIFreshnessCheck{Inputs: []string{"cli/**", "resource.json"}},
		},
	}

	original := runSourceBuildCommandFn
	t.Cleanup(func() { runSourceBuildCommandFn = original })
	var gotPath, gotDir string
	var gotArgs []string
	runSourceBuildCommandFn = func(cmd *exec.Cmd) error {
		gotPath = cmd.Path
		gotDir = cmd.Dir
		gotArgs = append([]string(nil), cmd.Args...)
		return nil
	}

	if err := runSourceBuild(context.Background(), controller, manifest, manifest.CLI.SourceBuild); err != nil {
		t.Fatalf("runSourceBuild: %v", err)
	}
	if filepath.Base(gotPath) != "go" {
		t.Fatalf("command path = %q, want go", gotPath)
	}
	if gotDir != filepath.Join(root, "packages", "cli-core") {
		t.Fatalf("command dir = %q, want cli-core directory", gotDir)
	}
	for _, want := range []string{
		"run",
		"./cmd/cli-installer",
		"--module", filepath.Join(root, "resources", "fixture", "cli"),
		"--manifest", filepath.Join(root, "resources", "fixture", "resource.json"),
		"--install-dir", filepath.Join(home, ".vrooli", "bin"),
		"--context-root", filepath.Join(root, "resources", "fixture"),
		"--freshness-input", "cli/**",
		"resource.json",
	} {
		if !slices.Contains(gotArgs, want) {
			t.Fatalf("installer args %v missing %q", gotArgs, want)
		}
	}
}
