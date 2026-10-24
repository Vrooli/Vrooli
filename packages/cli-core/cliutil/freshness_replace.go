package cliutil

import (
	"bufio"
	"bytes"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// importHeaderBytes bounds how much of a Go file is read to find its imports.
// Generated files can be megabytes long; the import block is always near the
// top. A file whose imports do not parse inside the window is read in full.
const importHeaderBytes = 64 << 10

// WithGoReplaceFreshnessInputs extends a Go-module CLI freshness spec with the
// local `replace` targets the CLI actually builds from, so an edit to a shared
// package (for example packages/agentharness) marks the installed binary stale
// without a hand-maintained input list.
//
// The import closure is followed from SourceRoot through the main module's
// replace table (the only one Go applies), package by package. Each imported
// package directory becomes an input, plus the go.mod of each replaced module.
// A target that contains the CLI (the repo-root `=> ../../..` replace, or the
// main module itself in a shared module) never contributes its root, so an
// unrelated edit elsewhere in the repo does not mark the CLI stale. A module
// that holds a declared input is left to that input: it is the deliberate
// representation (the proto artifact stamp stands for the generated proto
// tree; a declared ../../packages/cli-core covers all of cli-core). Candidates
// under a declared input are dropped, and the main go.mod is added when no
// input covers it.
//
// Expansion is deterministic and idempotent, so the installer, the installed
// manifest check and the runtime StaleChecker derive the same input list.
func WithGoReplaceFreshnessInputs(spec FreshnessSpec) FreshnessSpec {
	sourceRoot := cleanAbs(spec.SourceRoot)
	if sourceRoot == "" {
		return spec
	}
	contextRoot := cleanAbs(spec.ContextRoot)
	if contextRoot == "" {
		contextRoot = sourceRoot
	}
	goModPath, ok := findGoMod(sourceRoot)
	if !ok {
		return spec
	}
	mainModule, replaces, err := parseGoModLocalReplaces(goModPath)
	if err != nil || len(replaces) == 0 && mainModule == "" {
		return spec
	}
	if mainModule != "" {
		if _, exists := replaces[mainModule]; !exists {
			replaces[mainModule] = filepath.Dir(goModPath)
		}
	}

	declared := declaredInputBases(contextRoot, spec.Inputs)
	if len(spec.Inputs) == 0 {
		declared = append(declared, sourceRoot)
	}
	candidates := map[string]bool{}
	consider := func(path string) {
		for _, base := range declared {
			if pathWithin(base, path) || pathWithin(path, base) {
				return
			}
		}
		candidates[path] = true
	}
	consider(goModPath)

	scanned := map[string]bool{}
	queue := sourcePackageDirs(sourceRoot)
	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]
		if scanned[dir] {
			continue
		}
		scanned[dir] = true
		for _, importPath := range packageImports(dir) {
			module, target, ok := longestReplace(replaces, importPath)
			if !ok {
				continue
			}
			pkgDir := filepath.Join(target, filepath.FromSlash(strings.TrimPrefix(strings.TrimPrefix(importPath, module), "/")))
			if pathWithin(target, sourceRoot) || pathWithin(target, contextRoot) {
				// The target contains the CLI itself (repo-root replace or the
				// main module); walking its root would take in the whole repo.
				if pkgDir != target {
					consider(pkgDir)
				}
			} else if !representedByDeclared(declared, target) {
				consider(pkgDir)
				consider(filepath.Join(target, "go.mod"))
			}
			if !scanned[pkgDir] {
				queue = append(queue, pkgDir)
			}
		}
	}

	// A directory input is walked recursively, so drop candidates that lie
	// under another candidate.
	var added []string
	for path := range candidates {
		nested := false
		for other := range candidates {
			if other != path && pathWithin(other, path) {
				nested = true
				break
			}
		}
		if !nested {
			if _, err := os.Stat(path); err == nil {
				added = append(added, path)
			}
		}
	}
	if len(added) == 0 {
		return spec
	}

	rels := make([]string, 0, len(added))
	for _, dir := range added {
		rel, err := filepath.Rel(contextRoot, dir)
		if err != nil {
			continue
		}
		rels = append(rels, filepath.ToSlash(rel))
	}
	slices.Sort(rels)
	inputs := append([]string(nil), spec.Inputs...)
	if len(inputs) == 0 {
		// An empty input list means "the whole SourceRoot"; keep that meaning
		// once explicit inputs are added.
		if rel, err := filepath.Rel(contextRoot, sourceRoot); err == nil {
			inputs = append(inputs, filepath.ToSlash(rel))
		}
	}
	spec.Inputs = append(inputs, rels...)
	return spec
}

func cleanAbs(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return filepath.Clean(path)
	}
	return abs
}

func findGoMod(dir string) (string, bool) {
	for {
		candidate := filepath.Join(dir, "go.mod")
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// parseGoModLocalReplaces reads the module path and the replace directives
// whose target is a local directory (Go treats a target starting with ./, ../
// or an absolute path as a filesystem path). Targets that do not exist are
// dropped. cli-core has no golang.org/x/mod dependency; go.mod's replace
// grammar is small enough to read line by line.
func parseGoModLocalReplaces(goModPath string) (string, map[string]string, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", nil, err
	}
	base := filepath.Dir(goModPath)
	module := ""
	replaces := map[string]string{}
	inBlock := false
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			continue
		case inBlock && line == ")":
			inBlock = false
			continue
		case strings.HasPrefix(line, "module "):
			module = unquoteGoModToken(strings.TrimSpace(strings.TrimPrefix(line, "module")))
			continue
		case line == "replace (":
			inBlock = true
			continue
		case strings.HasPrefix(line, "replace "):
			line = strings.TrimSpace(strings.TrimPrefix(line, "replace"))
		case !inBlock:
			continue
		}
		left, right, ok := strings.Cut(line, "=>")
		if !ok {
			continue
		}
		oldFields := strings.Fields(left)
		newFields := strings.Fields(right)
		if len(oldFields) == 0 || len(newFields) != 1 {
			continue // a versioned target is a module, not a directory
		}
		target := unquoteGoModToken(newFields[0])
		if !filepath.IsAbs(target) && !strings.HasPrefix(target, "./") && !strings.HasPrefix(target, "../") && target != "." && target != ".." {
			continue
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(base, filepath.FromSlash(target))
		}
		target = filepath.Clean(target)
		if info, statErr := os.Stat(target); statErr != nil || !info.IsDir() {
			continue
		}
		replaces[unquoteGoModToken(oldFields[0])] = target
	}
	return module, replaces, scanner.Err()
}

func unquoteGoModToken(token string) string {
	if unquoted, err := strconv.Unquote(token); err == nil {
		return unquoted
	}
	return token
}

func longestReplace(replaces map[string]string, importPath string) (string, string, bool) {
	best := ""
	for module := range replaces {
		if (importPath == module || strings.HasPrefix(importPath, module+"/")) && len(module) > len(best) {
			best = module
		}
	}
	if best == "" {
		return "", "", false
	}
	return best, replaces[best], true
}

// declaredInputBases resolves each declared input to the directory or file it
// names, truncated before its first glob segment ("cli/**" covers "cli").
func declaredInputBases(contextRoot string, inputs []string) []string {
	bases := make([]string, 0, len(inputs))
	for _, input := range trimNonEmpty(inputs) {
		segments := strings.Split(filepath.ToSlash(input), "/")
		kept := segments[:0]
		for _, segment := range segments {
			if hasGlobPattern(segment) {
				break
			}
			kept = append(kept, segment)
		}
		bases = append(bases, filepath.Clean(filepath.Join(contextRoot, filepath.FromSlash(strings.Join(kept, "/")))))
	}
	return bases
}

// representedByDeclared reports whether a declared input lies inside a replace
// target. That input is the module's chosen representation (the whole module,
// or the proto artifact stamp for the generated proto tree), so none of the
// module's packages are added beside it.
func representedByDeclared(declared []string, target string) bool {
	for _, base := range declared {
		if pathWithin(target, base) {
			return true
		}
	}
	return false
}

// pathWithin reports whether target is root or lies beneath it.
func pathWithin(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// sourcePackageDirs lists every directory under root that holds Go source,
// skipping the directories the freshness walk skips plus vendor and testdata.
func sourcePackageDirs(root string) []string {
	var dirs []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return nil
			}
			if path != root && (strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" || buildinfoSkipDir(root, filepath.ToSlash(rel))) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			if dir := filepath.Dir(path); len(dirs) == 0 || dirs[len(dirs)-1] != dir {
				dirs = append(dirs, dir)
			}
		}
		return nil
	})
	return dirs
}

// packageImports returns the imports of the non-test Go files in dir. Build
// constraints are ignored, so a platform-specific import counts: freshness may
// over-include an input but must never miss one.
func packageImports(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var imports []string
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		for _, importPath := range fileImports(fset, filepath.Join(dir, name)) {
			if !seen[importPath] {
				seen[importPath] = true
				imports = append(imports, importPath)
			}
		}
	}
	slices.Sort(imports)
	return imports
}

func fileImports(fset *token.FileSet, path string) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	head, err := io.ReadAll(io.LimitReader(file, importHeaderBytes))
	if err != nil {
		return nil
	}
	parsed, err := parser.ParseFile(fset, path, head, parser.ImportsOnly)
	if err != nil {
		full, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if parsed, err = parser.ParseFile(fset, path, full, parser.ImportsOnly); err != nil {
			return nil
		}
	}
	out := make([]string, 0, len(parsed.Imports))
	for _, spec := range parsed.Imports {
		if value, err := strconv.Unquote(spec.Path.Value); err == nil {
			out = append(out, value)
		}
	}
	return out
}
