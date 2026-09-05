package depsapproved

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/envkit-go"
)

// runVendor is the governed entry point for synchronising a module's
// committed Go vendor tree with its go.mod. It deliberately requires an
// explicit apply flag: unlike replace reconciliation, go mod vendor rewrites
// a large generated tree and may download modules.
func runVendor(_ *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps vendor")
	var module string
	var preserve string
	var apply, jsonOutput bool
	fs.StringVar(&module, "module", "", "Repository-relative directory containing go.mod")
	fs.StringVar(&preserve, "preserve", "", "Comma-separated non-Go paths under vendor/ to preserve")
	fs.BoolVar(&apply, "apply", false, "Synchronise the module vendor tree")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || strings.TrimSpace(module) == "" || !apply {
		return fmt.Errorf("usage: %s deps vendor --module <repo-relative-go-module> [--preserve <vendor-subdir,...>] --apply [--json]", support.AppName)
	}

	root := cliutil.ResolveRepoRoot()
	moduleDir, err := repositoryModuleDir(root, module)
	if err != nil {
		return err
	}
	preserved, restore, err := preserveVendorTrees(moduleDir, preserve)
	if err != nil {
		return err
	}

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "go", "mod", "vendor")
	cmd.Dir = moduleDir
	cmd.Env = envkit.Toolchain(vendorEnvironment(os.Environ()), envkit.ToolchainOptions{})
	out, err := cmd.CombinedOutput()
	result := struct {
		Module       string   `json:"module"`
		Path         string   `json:"path"`
		Apply        bool     `json:"apply"`
		Preserved    []string `json:"preserved,omitempty"`
		Output       string   `json:"output,omitempty"`
		Error        string   `json:"error,omitempty"`
		RestoreError string   `json:"restore_error,omitempty"`
	}{Module: module, Path: moduleDir, Apply: true, Preserved: preserved, Output: strings.TrimSpace(string(out))}
	if restoreErr := restore(); restoreErr != nil {
		result.RestoreError = restoreErr.Error()
	}
	if err != nil {
		result.Error = err.Error()
	}
	if jsonOutput {
		if err := support.PrintReportJSON(result); err != nil {
			return err
		}
	} else if result.Output != "" {
		fmt.Println(result.Output)
	}
	if err != nil {
		return fmt.Errorf("go mod vendor in %s: %w", module, err)
	}
	if result.RestoreError != "" {
		return fmt.Errorf("restore preserved vendor inputs in %s: %s", module, result.RestoreError)
	}
	return nil
}

func repositoryModuleDir(root, module string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(module)))
	if clean == "." || filepath.IsAbs(clean) {
		return "", fmt.Errorf("module must be a repository-relative directory containing go.mod")
	}
	dir := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("module must stay inside the repository: %s", module)
	}
	if info, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil || info.IsDir() {
		return "", fmt.Errorf("module %q has no go.mod", module)
	}
	return dir, nil
}

// vendorEnvironment switches the workspace off and keeps the inherited
// GOFLAGS; the spawn site composes the build width through envkit.Toolchain.
func vendorEnvironment(environment []string) envkit.Env {
	return envkit.WithOverlay(envkit.Env(environment), envkit.SameScenario, envkit.Env{"GOWORK=off"})
}

func preserveVendorTrees(moduleDir, raw string) ([]string, func() error, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, func() error { return nil }, nil
	}
	backup, err := os.MkdirTemp("", "sda-vendor-preserve-")
	if err != nil {
		return nil, nil, fmt.Errorf("create vendor preservation directory: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(backup) }
	preserved := make([]string, 0)
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		rel := filepath.Clean(filepath.FromSlash(item))
		if filepath.IsAbs(rel) || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			cleanup()
			return nil, nil, fmt.Errorf("preserved vendor path must stay under vendor/: %s", item)
		}
		source := filepath.Join(moduleDir, "vendor", rel)
		if _, err := os.Stat(source); err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("preserved vendor path %q does not exist", item)
		}
		if err := copyTree(source, filepath.Join(backup, rel)); err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("preserve vendor path %q: %w", item, err)
		}
		preserved = append(preserved, filepath.ToSlash(rel))
	}
	return preserved, func() error {
		var restoreErr error
		for _, rel := range preserved {
			if err := os.RemoveAll(filepath.Join(moduleDir, "vendor", filepath.FromSlash(rel))); err != nil && restoreErr == nil {
				restoreErr = err
			}
			if err := copyTree(filepath.Join(backup, filepath.FromSlash(rel)), filepath.Join(moduleDir, "vendor", filepath.FromSlash(rel))); err != nil && restoreErr == nil {
				restoreErr = err
			}
		}
		cleanup()
		return restoreErr
	}, nil
}

func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := destination
		if rel != "." {
			target = filepath.Join(destination, rel)
		}
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
