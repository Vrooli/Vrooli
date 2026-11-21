package setup

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/envkit-go"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"
	"golang.org/x/mod/modfile"
)

type goBuildEnvironment struct {
	goCommand    string
	root, nested []string
	workspaces   map[string][32]byte
	vendorDirs   []string
}

// Freeze process variables and the persistent go env file once for a cohort.
// Preserve workspace discovery separately for the nested module, then pin the
// discovered paths and reject workspace edits observed during compilation.
func freezeGoBuildEnvironment(root, stage string) (*goBuildEnvironment, error) {
	goCommand, err := exec.LookPath("go")
	if err != nil {
		return nil, err
	}
	goCommand, err = filepath.Abs(goCommand)
	if err != nil {
		return nil, err
	}
	env := envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"CGO_ENABLED=0"})
	env = envkit.Toolchain(env, envkit.ToolchainOptions{Width: tuning.BuildWidth()})
	get := func(dir string, env []string, name string) (string, error) {
		out, err := shell.Output(shell.Spec{Name: goCommand, Args: []string{"env", name}, Dir: dir, Env: env})
		if err != nil {
			return "", fmt.Errorf("go env %s: %w", name, err)
		}
		return strings.TrimSpace(string(out)), nil
	}
	goEnv, err := get(root, env, "GOENV")
	if err != nil {
		return nil, err
	}
	pinnedConfig := "off"
	if goEnv != "" && goEnv != "off" {
		data, err := os.ReadFile(goEnv)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read Go configuration: %w", err)
		}
		if err == nil {
			pinnedConfig, err = filepath.Abs(filepath.Join(stage, "go-env"))
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(pinnedConfig, data, 0600); err != nil {
				return nil, err
			}
		}
	}
	env = envkit.WithOverlay(env, envkit.SameScenario, envkit.Env{"GOENV=" + pinnedConfig})
	flags, err := get(root, env, "GOFLAGS")
	if err != nil {
		return nil, err
	}
	if err := rejectUnsupportedGoFlags(strings.Fields(flags)); err != nil {
		return nil, err
	}
	result := &goBuildEnvironment{goCommand: goCommand, workspaces: make(map[string][32]byte), vendorDirs: []string{filepath.Join(root, "vendor"), filepath.Join(root, "cmd", "vrooli-policy-runner", "vendor")}}
	for _, dir := range []string{root, filepath.Join(root, "cmd", "vrooli-policy-runner")} {
		if err := validateModuleReplacements(root, dir); err != nil {
			return nil, err
		}
	}
	pin := func(dir string) ([]string, error) {
		workspace, err := get(dir, env, "GOWORK")
		if err != nil {
			return nil, err
		}
		if workspace == "" {
			workspace = "off"
		}
		if workspace != "off" {
			result.vendorDirs = append(result.vendorDirs, filepath.Join(filepath.Dir(workspace), "vendor"))
			if err := validateWorkspace(root, workspace); err != nil {
				return nil, err
			}
			for _, path := range []string{workspace, workspace + ".sum"} {
				digest, err := workspaceDigest(path)
				if err != nil {
					return nil, err
				}
				result.workspaces[path] = digest
			}
		}
		return envkit.WithOverlay(env, envkit.SameScenario, envkit.Env{"GOWORK=" + workspace}), nil
	}
	result.root, err = pin(root)
	if err != nil {
		return nil, err
	}
	result.nested, err = pin(filepath.Join(root, "cmd", "vrooli-policy-runner"))
	if err != nil {
		return nil, err
	}
	if err := result.verifyNoVendor(); err != nil {
		return nil, err
	}
	return result, nil
}

func rejectUnsupportedGoFlags(flags []string) error {
	for i, flag := range flags {
		if flag == "-overlay" || strings.HasPrefix(flag, "-overlay=") {
			return errors.New("Go build overlays are not covered by cohort input verification; remove -overlay before building")
		}
		if flag == "-mod=vendor" || (flag == "-mod" && i+1 < len(flags) && flags[i+1] == "vendor") {
			return errors.New("Go vendor mode is not covered by cohort input verification; use module mode before building")
		}
	}
	return nil
}

func coveredModule(root, base, path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve local Go module %s: %w", path, err)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	canonicalRoot, err = filepath.Abs(canonicalRoot)
	if err != nil {
		return "", err
	}
	if resolved == canonicalRoot {
		return resolved, nil
	}
	relative, err := filepath.Rel(canonicalRoot, resolved)
	if err != nil {
		return "", err
	}
	// Keep this list aligned with buildinfo's skippedDirs: those directories
	// are excluded at any depth even when an ancestor is a covered root.
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		switch part {
		case ".git", ".vrooli", "build", "dist", "node_modules", "tmp":
			return "", fmt.Errorf("local Go module %s is in a directory excluded from cohort input coverage", resolved)
		}
	}
	for _, covered := range buildCohortSourcePaths {
		candidate := filepath.Join(canonicalRoot, covered)
		if rel, err := filepath.Rel(candidate, resolved); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return resolved, nil
		}
	}
	return "", fmt.Errorf("local Go module %s is outside cohort input coverage", resolved)
}

// A vendor directory can switch Go into vendor mode without a flag. Check
// before and after compilation to reject a cohort if that input appears.
func (e *goBuildEnvironment) verifyNoVendor() error {
	for _, dir := range e.vendorDirs {
		info, err := os.Stat(dir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect Go vendor directory %s: %w", dir, err)
		}
		if info.IsDir() {
			return fmt.Errorf("Go vendor directory %s is not covered by cohort input verification", dir)
		}
	}
	return nil
}

func validateReplacements(root, base string, replacements []*modfile.Replace) error {
	for _, replacement := range replacements {
		if replacement.New.Version == "" {
			if _, err := coveredModule(root, base, replacement.New.Path); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateModuleReplacements(root, dir string) error {
	path := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	f, err := modfile.Parse(path, data, nil)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return validateReplacements(root, dir, f.Replace)
}

func validateWorkspace(root, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	f, err := modfile.ParseWork(path, data, nil)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	base := filepath.Dir(path)
	if err := validateReplacements(root, base, f.Replace); err != nil {
		return err
	}
	for _, use := range f.Use {
		dir, err := coveredModule(root, base, use.Path)
		if err != nil {
			return err
		}
		if err := validateModuleReplacements(root, dir); err != nil {
			return err
		}
	}
	return nil
}

func workspaceDigest(path string) ([32]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return sha256.Sum256([]byte{0}), nil
	}
	if err != nil {
		return [32]byte{}, fmt.Errorf("read Go workspace %s: %w", path, err)
	}
	return sha256.Sum256(append([]byte{1}, data...)), nil
}

func (e *goBuildEnvironment) verifyWorkspaces() error {
	for path, before := range e.workspaces {
		after, err := workspaceDigest(path)
		if err != nil {
			return err
		}
		if before != after {
			return fmt.Errorf("Go workspace %s changed during compilation; existing binaries were left untouched, retry the build", path)
		}
	}
	return nil
}
