package gates

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type runnerResolutionConfig struct {
	Gates []struct {
		ID     string            `json:"id"`
		Runner map[string]string `json:"runner"`
	} `json:"gates"`
}

type runnerManifestNode struct {
	Name     string               `json:"name"`
	Commands []runnerManifestNode `json:"commands"`
}

// TestCatalogRunnersResolveAgainstTheCLIManifest is the rename guard for the
// catalog's declared runner strings. It deliberately checks the authored
// config and manifest together, so a command rename fails this quality test in
// the same change that would otherwise leave a silent declaration-only gate.
func TestCatalogRunnersResolveAgainstTheCLIManifest(t *testing.T) {
	root := testRepoRoot(t)
	config := runnerResolutionConfig{}
	readJSON(t, filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json"), &config)
	var manifest struct {
		Groups []runnerManifestNode `json:"groups"`
	}
	readJSON(t, filepath.Join(root, "scenarios", "react-component-library", "cli", "manifest.json"), &manifest)

	commandPairs := map[string]struct{}{}
	collectManifestCommands(manifest.Groups, nil, commandPairs)
	seen := 0
	for _, gate := range config.Gates {
		for target, runner := range gate.Runner {
			seen++
			if err := resolveRunner(context.Background(), root, runner, commandPairs); err != nil {
				t.Fatalf("gate %s target %s runner %q: %v", gate.ID, target, runner, err)
			}
			t.Logf("resolved gate=%s target=%s runner=%s", gate.ID, target, runner)
		}
	}
	if seen != 61 {
		t.Fatalf("runner count = %d, want authored baseline 61 after stylesheet-key and token-fallback-literal gates", seen)
	}
}

func collectManifestCommands(nodes []runnerManifestNode, prefix []string, out map[string]struct{}) {
	for _, node := range nodes {
		if strings.TrimSpace(node.Name) == "" {
			continue
		}
		path := append(append([]string(nil), prefix...), node.Name)
		// A runner resolves the group/subcommand pair even when the manifest
		// continues with a positional gate or another nested command.
		if len(path) >= 2 {
			out[strings.Join(path, " ")] = struct{}{}
		}
		if len(node.Commands) == 0 {
			out[strings.Join(path, " ")] = struct{}{}
			continue
		}
		collectManifestCommands(node.Commands, path, out)
	}
}

func resolveRunner(ctx context.Context, root, runner string, commandPairs map[string]struct{}) error {
	words := strings.Fields(strings.TrimSpace(runner))
	if len(words) == 0 {
		return fmt.Errorf("empty runner")
	}
	if words[0] == "react-component-library" {
		if len(words) < 3 {
			return fmt.Errorf("internal runner does not contain <group> <subcommand>")
		}
		pair := strings.Join(words[1:3], " ")
		if _, ok := commandPairs[pair]; !ok {
			return fmt.Errorf("manifest has no command pair %q", pair)
		}
		return nil
	}
	return runExternalHelp(ctx, root, words)
}

func runExternalHelp(parent context.Context, root string, words []string) error {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	if len(words) >= 3 && words[0] == "pnpm" && words[1] == "run" {
		// `pnpm run <script> --help` executes the script, so a failing
		// catalog check would make command-resolution conflate existence with
		// health. Listing scripts proves the external command is registered.
		cmd := exec.CommandContext(ctx, "pnpm", "run")
		cmd.Dir = filepath.Join(root, "scenarios", "react-component-library", "ui")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("pnpm run failed: %w (%s)", err, strings.TrimSpace(string(output)))
		}
		if !strings.Contains(string(output), words[2]) {
			return fmt.Errorf("pnpm script %q is not registered", words[2])
		}
		return nil
	}
	args := append(append([]string(nil), words[1:]...), "--help")
	cmd := exec.CommandContext(ctx, words[0], args...)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s --help failed: %w (%s)", strings.Join(words, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func testRepoRoot(t *testing.T) string {
	t.Helper()
	working, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := working
	for {
		if _, err := os.Stat(filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json")); err == nil {
			return root
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatalf("could not locate repo root from %s", working)
		}
		root = parent
	}
}

func readJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}
