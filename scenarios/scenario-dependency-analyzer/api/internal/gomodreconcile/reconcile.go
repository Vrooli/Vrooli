// Package gomodreconcile is the single reconcile primitive for in-repo Go module
// replace directives. Go does not propagate a dependency's replace directives to
// downstream main modules, so every surface go.mod that (transitively) requires
// an in-repo module must declare its own local `replace`. When a shared leaf
// package quietly takes on a new in-repo edge, consumers that lack the replace
// fail `go build` only at restart time. This primitive detects and repairs that
// gap, converging on the golden per-surface shape: each surface declares its own
// require + a local replace pointing at the in-repo module's directory, with no
// go.sum entry for the replaced in-repo module.
//
// It is intentionally toolchain-driven (`go mod edit`/`go mod tidy`) rather than
// pulling in golang.org/x/mod: the go binary is authoritative for go.mod
// formatting (`go mod edit -fmt`) and is always present, and SDA — the
// dependency authority — should not grow a new third-party dependency just to
// rewrite go.mod files.
//
// Topology (which module paths are in-repo and where they live) is ground truth
// read from the repo's go.mod `module` declarations. This is the authoritative
// source for a replace target (module path -> directory) and is a superset of
// the governed package registry surfaced by `vrooli package list`; the leaf
// policy in internal/packagegov governs which couplings are *allowed*, while this
// primitive only ever adds the local replace that makes a surface build.
package gomodreconcile

import (
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Topology maps an in-repo Go module path to its absolute on-disk directory.
type Topology map[string]string

// goModView is the subset of `go mod edit -json` output we consume.
type goModView struct {
	Module  struct{ Path string }
	Require []struct {
		Path     string
		Version  string
		Indirect bool
	}
	Replace []struct {
		Old struct{ Path string }
		New struct {
			Path    string
			Version string
		}
	}
}

// MissingReplace is one in-repo module that a surface requires without a matching
// local replace.
type MissingReplace struct {
	// Module is the required in-repo module path (e.g. github.com/vrooli/cli-core).
	Module string
	// RelPath is the surface-relative path to the module directory that the
	// replace must point at (e.g. ../../../packages/cli-core).
	RelPath string
	// AddRequire is true when the surface imports the in-repo module but go.mod
	// does not yet require it. The deterministic fix adds require v0.0.0 first,
	// then the local replace.
	AddRequire bool
}

// Candidate is a proposed (or applied) reconcile edit for a single surface
// go.mod. Before/After hold the full file content so callers can present a diff.
type Candidate struct {
	// GoModPath is the absolute path of the surface go.mod.
	GoModPath string
	// Missing lists the in-repo replaces that were (or would be) added.
	Missing []MissingReplace
	Before  string
	After   string
	Applied bool
}

// LoadTopology scans repoRoot for in-repo Go modules and returns module path ->
// absolute directory. It reads the `module` declaration from each go.mod under
// the repo, skipping vendored, generated, and data directories that never carry
// reconcilable surfaces.
func LoadTopology(repoRoot string) (Topology, error) {
	repoRoot = filepath.Clean(repoRoot)
	topo := make(Topology)
	skipDir := map[string]struct{}{
		".git": {}, "node_modules": {}, "vendor": {}, "data": {},
		"dist": {}, "build": {}, ".cache": {},
	}
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if _, skip := skipDir[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		modulePath := readModulePath(path)
		if modulePath == "" {
			return nil
		}
		topo[modulePath] = filepath.Dir(path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return topo, nil
}

func readModulePath(goModPath string) string {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "//") {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// Plan computes the missing in-repo require/replace wiring for a single surface
// go.mod without mutating anything. It covers both modules already present in
// the dependency graph and in-repo modules imported by source files but not yet
// declared in go.mod (for example a scenario CLI importing its sibling API).
func Plan(ctx context.Context, goModPath string, topo Topology) ([]MissingReplace, error) {
	view, err := parseGoMod(ctx, goModPath)
	if err != nil {
		return nil, err
	}
	moduleDir := filepath.Dir(goModPath)
	requiredDirect := make(map[string]struct{}, len(view.Require))
	for _, req := range view.Require {
		requiredDirect[req.Path] = struct{}{}
	}
	replaced := make(map[string]struct{}, len(view.Replace))
	for _, r := range view.Replace {
		replaced[r.Old.Path] = struct{}{}
	}
	required, err := requiredInRepoModules(ctx, view, topo)
	if err != nil {
		return nil, err
	}
	imported, err := importedInRepoModules(moduleDir, view.Module.Path, topo)
	if err != nil {
		return nil, err
	}
	candidates := make(map[string]bool)
	for _, path := range required {
		candidates[path] = false
	}
	for _, path := range imported {
		if _, ok := requiredDirect[path]; !ok {
			candidates[path] = true
		} else if _, ok := candidates[path]; !ok {
			candidates[path] = false
		}
	}

	var missing []MissingReplace
	for _, path := range sortedBoolKeys(candidates) {
		if path == view.Module.Path {
			continue
		}
		dir, inRepo := topo[path]
		if !inRepo {
			continue // third-party — governed by approved-dependencies, not here
		}
		addRequire := candidates[path]
		if _, ok := replaced[path]; ok && !addRequire {
			continue
		}
		rel, err := filepath.Rel(moduleDir, dir)
		if err != nil {
			// Unambiguous in-repo module whose relative path cannot be derived:
			// flag it (caller surfaces a finding) but never guess a path.
			continue
		}
		missing = append(missing, MissingReplace{Module: path, RelPath: filepath.ToSlash(rel), AddRequire: addRequire})
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i].Module < missing[j].Module })
	return missing, nil
}

func requiredInRepoModules(ctx context.Context, view goModView, topo Topology) ([]string, error) {
	seen := map[string]struct{}{}
	var walk func(string) error
	walk = func(path string) error {
		if _, ok := seen[path]; ok {
			return nil
		}
		dir, inRepo := topo[path]
		if !inRepo {
			return nil
		}
		seen[path] = struct{}{}
		child, err := parseGoMod(ctx, filepath.Join(dir, "go.mod"))
		if err != nil {
			return err
		}
		for _, req := range child.Require {
			if req.Path == child.Module.Path {
				continue
			}
			if err := walk(req.Path); err != nil {
				return err
			}
		}
		return nil
	}
	for _, req := range view.Require {
		if req.Path == view.Module.Path {
			continue
		}
		if err := walk(req.Path); err != nil {
			return nil, err
		}
	}
	out := make([]string, 0, len(seen))
	for path := range seen {
		out = append(out, path)
	}
	sort.Strings(out)
	return out, nil
}

func importedInRepoModules(moduleDir, selfModule string, topo Topology) ([]string, error) {
	modules := sortedTopologyModules(topo)
	seen := map[string]struct{}{}
	skipDir := map[string]struct{}{
		".git": {}, "node_modules": {}, "vendor": {}, "data": {},
		"dist": {}, "build": {}, ".cache": {}, "coverage": {},
	}
	err := filepath.WalkDir(moduleDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if _, skip := skipDir[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return nil
		}
		for _, spec := range file.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			if importPath == "" || importPathMatchesModule(importPath, selfModule) {
				continue
			}
			if module := matchingInRepoModule(importPath, modules); module != "" {
				seen[module] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(seen))
	for module := range seen {
		out = append(out, module)
	}
	sort.Strings(out)
	return out, nil
}

func matchingInRepoModule(importPath string, modules []string) string {
	for _, module := range modules {
		if importPathMatchesModule(importPath, module) {
			return module
		}
	}
	return ""
}

func importPathMatchesModule(importPath, module string) bool {
	module = strings.TrimSuffix(strings.TrimSpace(module), "/")
	if module == "" {
		return false
	}
	return importPath == module || strings.HasPrefix(importPath, module+"/")
}

func sortedTopologyModules(topo Topology) []string {
	modules := make([]string, 0, len(topo))
	for module := range topo {
		modules = append(modules, module)
	}
	sort.Slice(modules, func(i, j int) bool {
		if len(modules[i]) == len(modules[j]) {
			return modules[i] < modules[j]
		}
		return len(modules[i]) > len(modules[j])
	})
	return modules
}

func sortedBoolKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// PreviewSurface returns the deterministic before/after for adding the currently
// missing in-repo replaces to one surface, without writing or tidying. Returns
// nil when the surface is already converged.
func PreviewSurface(ctx context.Context, goModPath string, topo Topology) (*Candidate, error) {
	missing, err := Plan(ctx, goModPath, topo)
	if err != nil {
		return nil, err
	}
	if len(missing) == 0 {
		return nil, nil
	}
	before, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, err
	}
	after, err := editedGoMod(ctx, goModPath, string(before), missing)
	if err != nil {
		return nil, err
	}
	return &Candidate{
		GoModPath: goModPath,
		Missing:   missing,
		Before:    string(before),
		After:     after,
	}, nil
}

// ApplySurface writes the missing replaces and runs `go mod tidy`, iterating to a
// fixpoint so that in-repo edges newly materialized by tidy (as indirect
// requires) also receive their replace. It is idempotent: an already-converged
// surface produces no change. Returns nil when nothing needed fixing.
func ApplySurface(ctx context.Context, goModPath string, topo Topology) (*Candidate, error) {
	before, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, err
	}
	moduleDir := filepath.Dir(goModPath)
	var allMissing []MissingReplace
	changed := false
	const maxIters = 8
	for i := 0; i < maxIters; i++ {
		missing, err := Plan(ctx, goModPath, topo)
		if err != nil {
			return nil, err
		}
		if len(missing) == 0 {
			if !changed {
				return nil, nil
			}
			// Converged after at least one edit — final tidy already ran below.
			break
		}
		args := []string{"mod", "edit"}
		for _, m := range missing {
			if m.AddRequire {
				args = append(args, "-require="+m.Module+"@v0.0.0")
			}
			args = append(args, "-replace="+m.Module+"="+m.RelPath)
		}
		if err := runGo(ctx, moduleDir, args...); err != nil {
			return nil, fmt.Errorf("go mod edit: %w", err)
		}
		allMissing = append(allMissing, missing...)
		changed = true
		if err := runGo(ctx, moduleDir, "mod", "tidy"); err != nil {
			return nil, fmt.Errorf("go mod tidy after adding replaces: %w", err)
		}
	}
	after, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, err
	}
	allMissing = dedupeMissing(allMissing)
	return &Candidate{
		GoModPath: goModPath,
		Missing:   allMissing,
		Before:    string(before),
		After:     string(after),
		Applied:   true,
	}, nil
}

// editedGoMod returns goMod content with the given replaces added, using a temp
// copy so the original is untouched (`go mod edit -fmt` formatting).
func editedGoMod(ctx context.Context, goModPath, content string, missing []MissingReplace) (string, error) {
	tmpDir, err := os.MkdirTemp("", "gomodreconcile-preview-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	tmp := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return "", err
	}
	args := []string{"mod", "edit"}
	for _, m := range missing {
		if m.AddRequire {
			args = append(args, "-require="+m.Module+"@v0.0.0")
		}
		args = append(args, "-replace="+m.Module+"="+m.RelPath)
	}
	args = append(args, "-fmt", tmp)
	if err := runGo(ctx, tmpDir, args...); err != nil {
		return "", err
	}
	out, err := os.ReadFile(tmp)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func parseGoMod(ctx context.Context, goModPath string) (goModView, error) {
	var view goModView
	cmd := exec.CommandContext(ctx, "go", "mod", "edit", "-json", goModPath)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.Output()
	if err != nil {
		return view, fmt.Errorf("go mod edit -json %s: %w", goModPath, err)
	}
	if err := json.Unmarshal(out, &view); err != nil {
		return view, fmt.Errorf("parse go.mod json: %w", err)
	}
	return view, nil
}

func runGo(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func dedupeMissing(in []MissingReplace) []MissingReplace {
	seen := make(map[string]struct{}, len(in))
	out := make([]MissingReplace, 0, len(in))
	for _, m := range in {
		if _, ok := seen[m.Module]; ok {
			continue
		}
		seen[m.Module] = struct{}{}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Module < out[j].Module })
	return out
}
