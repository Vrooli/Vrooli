// Package gomodreconcile converges each in-repo Go surface's go.mod and go.sum
// on its import closure. Every scenario, resource, and package Go module binds
// shared in-repo modules through local `replace` directives. A replace shares
// source, not the module graph, and Go does not propagate a dependency's
// replaces, so each consumer must declare two things itself:
//
//   - every requirement and checksum its import closure reaches, including
//     those that arrive only through a shared module's imports, and
//   - a local replace for every in-repo module among those requirements (and
//     no others: Go prunes the module graph, so other replaces are dead).
//
// When a shared module gains an in-repo edge or a third-party import, every
// consumer lacking the matching declaration fails `go build` (the 2026-09-01
// api-core incident broke 98 of 118 modules). Plan detects missing replaces
// cheaply; Converge computes the full converged state on a scratch copy; and
// Convergence.Write installs it. Deciding whether a converged state may be
// written without a new dependency decision belongs to the governance gate in
// internal/dependencygovernance, not here.
//
// The go command stays authoritative for module semantics (`go mod edit`,
// `go mod tidy`); golang.org/x/mod only parses go.mod files and orders go.sum.
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
	"fmt"
	"go/ast"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	repocontract "github.com/vrooli/repo-contract-go"
	"golang.org/x/mod/modfile"
)

// Topology maps an in-repo Go module path to its absolute on-disk directory.
type Topology map[string]string

var topologyCache = struct {
	sync.Mutex
	byRoot map[string]Topology
}{byRoot: map[string]Topology{}}

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

// LoadTopology scans repoRoot for in-repo Go modules and returns module path ->
// absolute directory. It reads the `module` declaration from each go.mod under
// the repo, skipping vendored, generated, and data directories that never carry
// reconcilable surfaces.
func LoadTopology(repoRoot string) (Topology, error) {
	repoRoot = filepath.Clean(repoRoot)
	topologyCache.Lock()
	if cached, ok := topologyCache.byRoot[repoRoot]; ok {
		out := make(Topology, len(cached))
		for path, dir := range cached {
			out[path] = dir
		}
		topologyCache.Unlock()
		return out, nil
	}
	topologyCache.Unlock()
	topo := make(Topology)
	// The repository-root Go module owns the control plane even though the
	// contract exposes cmd/ and internal/ as its concrete source targets. Add
	// that aggregate module explicitly so transitive in-repo requirements can
	// be followed through it without widening the filesystem walk to project
	// docs, templates, and other non-build targets.
	if modulePath := readModulePath(filepath.Join(repoRoot, "go.mod")); modulePath != "" {
		topo[modulePath] = repoRoot
	}
	targetRoots, err := reconcilableTargetRoots(repoRoot)
	if err != nil {
		return nil, err
	}
	skipDir := map[string]struct{}{
		".git": {}, "node_modules": {}, "vendor": {}, "data": {},
		"dist": {}, "build": {}, ".cache": {},
		// Retained snapshots are evidence, not source module providers. A
		// scenario target contains its docs too, so the target-root check alone
		// cannot distinguish an archived go.mod from a production module.
		"docs": {}, "testdata": {}, "evidence": {}, ".vrooli": {},
	}
	err = filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
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
		if !isUnderTargetRoot(filepath.Dir(path), targetRoots) {
			return nil
		}
		modulePath := readModulePath(path)
		if modulePath == "" {
			return nil
		}
		dir := filepath.Dir(path)
		if previous, exists := topo[modulePath]; exists && previous != dir {
			return fmt.Errorf("ambiguous source module %s: %s and %s", modulePath, previous, dir)
		}
		topo[modulePath] = dir
		return nil
	})
	if err != nil {
		return nil, err
	}
	topologyCache.Lock()
	topologyCache.byRoot[repoRoot] = topo
	topologyCache.Unlock()
	return topo, nil
}

// reconcilableTargetRoots derives the module search boundary from the
// repository contract. Project/docs/team targets describe governance surfaces,
// not Go module roots; source-bearing target kinds are the only roots relevant
// to dependency reconciliation. In particular, template modules are not
// targets and must never enter the topology.
func reconcilableTargetRoots(repoRoot string) ([]string, error) {
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("load repository contract for Go module topology: %w", err)
	}
	targets, err := contract.EnumerateTargets(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("enumerate repository targets for Go module topology: %w", err)
	}
	allowed := map[repocontract.TargetKind]struct{}{
		repocontract.TargetKindScenario:     {},
		repocontract.TargetKindResource:     {},
		repocontract.TargetKindPackage:      {},
		repocontract.TargetKindControlPlane: {},
	}
	roots := make([]string, 0, len(targets))
	for _, target := range targets {
		if _, ok := allowed[target.Kind]; !ok {
			continue
		}
		roots = append(roots, filepath.Join(repoRoot, filepath.FromSlash(target.Root)))
	}
	return roots, nil
}

func isUnderTargetRoot(path string, roots []string) bool {
	for _, root := range roots {
		rel, err := filepath.Rel(root, path)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
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
// go.mod without mutating anything: in-repo modules the go.mod requires without
// a matching local replace, in-repo modules its own source imports without a
// requirement (for example a scenario CLI importing its sibling API), and local
// replaces that point somewhere other than the module's source.
//
// Plan does not follow in-repo modules' own requirements. Go prunes the module
// graph: a dependency's requirements are loaded only when the consumer's go.mod
// lists them, and `go mod tidy` lists every module that provides a package in
// the import closure. Converge discovers those through tidy.
func Plan(ctx context.Context, goModPath string, topo Topology) ([]MissingReplace, error) {
	return planModFile(ctx, goModPath, filepath.Dir(goModPath), topo)
}

// planModFile plans against modFile while resolving relative replaces and
// source imports from moduleDir, so a scratch copy of a surface's go.mod can be
// planned exactly as the surface itself would be.
func planModFile(ctx context.Context, modFile, moduleDir string, topo Topology) ([]MissingReplace, error) {
	view, err := parseGoMod(ctx, modFile)
	if err != nil {
		return nil, err
	}
	requiredDirect := make(map[string]struct{}, len(view.Require))
	for _, req := range view.Require {
		requiredDirect[req.Path] = struct{}{}
	}
	replaced := make(map[string]string, len(view.Replace))
	for _, r := range view.Replace {
		if r.New.Version == "" {
			resolved := r.New.Path
			if !filepath.IsAbs(resolved) {
				resolved = filepath.Join(moduleDir, resolved)
			}
			replaced[r.Old.Path] = filepath.Clean(resolved)
		} else {
			replaced[r.Old.Path] = "versioned"
		}
	}
	imported, err := importedInRepoModules(moduleDir, view.Module.Path, topo)
	if err != nil {
		return nil, err
	}
	candidates := make(map[string]bool)
	// Existing local replacements remain build inputs even if tidy removed
	// their requirement. Never leave one pointing at an archived source copy.
	for path := range replaced {
		if _, inRepo := topo[path]; inRepo {
			candidates[path] = false
		}
	}
	for path := range requiredDirect {
		if _, inRepo := topo[path]; inRepo {
			candidates[path] = false
		}
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
		if target, ok := replaced[path]; ok && !addRequire && (target == filepath.Clean(dir) || target == "versioned") {
			continue
		}
		rel, err := filepath.Rel(moduleDir, dir)
		if err != nil {
			// Unambiguous in-repo module whose relative path cannot be derived:
			// flag it (caller surfaces a finding) but never guess a path.
			continue
		}
		missing = append(missing, MissingReplace{Module: path, RelPath: localReplacePath(rel), AddRequire: addRequire})
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
	modules := modulePrefixIndex(topo)
	seen := map[string]struct{}{}
	err := walkModuleGoFiles(moduleDir, func(path string, file *ast.File) {
		for _, spec := range file.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			if importPath == "" {
				continue
			}
			// Resolve the owning in-repo module by longest prefix first: an
			// import of a nested module (for example a scenario's `cli`
			// submodule) shares the parent module's prefix and must not be
			// mistaken for a package of the module under inspection.
			module := matchingInRepoModule(importPath, modules)
			if module == "" || module == selfModule {
				continue
			}
			seen[module] = struct{}{}
		}
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

func matchingInRepoModule(importPath string, modules map[string]string) string {
	parts := strings.Split(importPath, "/")
	for i := len(parts); i > 0; i-- {
		if module, ok := modules[strings.Join(parts[:i], "/")]; ok {
			return module
		}
	}
	return ""
}

// modulePrefixIndex turns longest-prefix module matching into one map lookup.
// Every import path is reduced to its own slash-separated prefixes, so nested
// modules retain the exact longest match without scanning the topology.
func modulePrefixIndex(topo Topology) map[string]string {
	index := make(map[string]string)
	for module := range topo {
		index[module] = module
	}
	return index
}

func importPathMatchesModule(importPath, module string) bool {
	module = strings.TrimSuffix(strings.TrimSpace(module), "/")
	if module == "" {
		return false
	}
	return importPath == module || strings.HasPrefix(importPath, module+"/")
}

// localReplacePath normalizes a module-relative directory into a path that
// `go mod edit -replace` accepts as a local target. A bare child directory such
// as `cli` is rejected ("unversioned new path must be local directory"), so it
// gains a `./` prefix; `../…`, `./…` and absolute paths are already local.
func localReplacePath(rel string) string {
	rel = filepath.ToSlash(rel)
	if rel == "" || rel == "." || rel == ".." {
		return rel
	}
	if strings.HasPrefix(rel, "../") || strings.HasPrefix(rel, "./") || strings.HasPrefix(rel, "/") {
		return rel
	}
	return "./" + rel
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

// PrepareLocalInstall seeds local replace directives for a repository-owned Go
// package before the governed installer invokes `go get`. Go otherwise tries to
// resolve the unpublished module from the network before the normal post-install
// reconcile can observe its new require. The target module's repository-local
// transitive requirements are seeded in the same edit because dependency-module
// replace directives do not propagate to the consuming main module.
//
// The function deliberately does not add a require. Dependency selection and
// version recording remain owned by `go get`. It returns false without mutating
// the manifest when packagePath does not belong to an in-repo module.
func PrepareLocalInstall(ctx context.Context, goModPath, packagePath string, topo Topology) (bool, error) {
	packagePath = strings.TrimSpace(packagePath)
	var targetModule string
	for _, module := range sortedTopologyModules(topo) {
		if importPathMatchesModule(packagePath, module) {
			targetModule = module
			break
		}
	}
	if targetModule == "" {
		return false, nil
	}

	targetDir := topo[targetModule]
	targetView, err := parseGoMod(ctx, filepath.Join(targetDir, "go.mod"))
	if err != nil {
		return false, fmt.Errorf("read local install target %s: %w", targetModule, err)
	}
	transitive, err := requiredInRepoModules(ctx, targetView, topo)
	if err != nil {
		return false, fmt.Errorf("resolve local install target %s: %w", targetModule, err)
	}
	modules := append([]string{targetModule}, transitive...)
	sort.Strings(modules)

	moduleDir := filepath.Dir(goModPath)
	args := []string{"mod", "edit"}
	for _, module := range modules {
		dir := topo[module]
		rel, err := filepath.Rel(moduleDir, dir)
		if err != nil {
			return false, fmt.Errorf("derive local replace for %s: %w", module, err)
		}
		args = append(args, "-replace="+module+"="+localReplacePath(rel))
	}
	if err := runGo(ctx, moduleDir, args...); err != nil {
		return false, fmt.Errorf("prepare local install replaces: %w", err)
	}
	return true, nil
}

func parseGoMod(ctx context.Context, goModPath string) (goModView, error) {
	select {
	case <-ctx.Done():
		return goModView{}, ctx.Err()
	default:
	}
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return goModView{}, fmt.Errorf("read go.mod %s: %w", goModPath, err)
	}
	file, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return goModView{}, fmt.Errorf("parse go.mod %s: %w", goModPath, err)
	}
	var view goModView
	if file.Module != nil {
		view.Module.Path = file.Module.Mod.Path
	}
	if view.Module.Path == "" {
		return goModView{}, fmt.Errorf("parse go.mod %s: module path is empty", goModPath)
	}
	for _, req := range file.Require {
		view.Require = append(view.Require, struct {
			Path     string
			Version  string
			Indirect bool
		}{Path: req.Mod.Path, Version: req.Mod.Version, Indirect: req.Indirect})
	}
	for _, replacement := range file.Replace {
		view.Replace = append(view.Replace, struct {
			Old struct{ Path string }
			New struct {
				Path    string
				Version string
			}
		}{Old: struct{ Path string }{Path: replacement.Old.Path}, New: struct {
			Path    string
			Version string
		}{Path: replacement.New.Path, Version: replacement.New.Version}})
	}
	return view, nil
}
