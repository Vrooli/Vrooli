package lab_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "audio-tools"

// labImportPrefixes is the authoritative lab/qualification package set. Matching
// is by path segment, so a new subpackage under internal/experiment (or a new
// handler under handlers/eval) is lab by default instead of quietly escaping
// the boundary. Keep in sync with README.md.
var labImportPrefixes = []string{
	modulePath + "/internal/eval",
	modulePath + "/internal/experiment",
	modulePath + "/internal/soak",
	modulePath + "/handlers/eval",
	modulePath + "/handlers/experiment",
	modulePath + "/handlers/soak",
	modulePath + "/handlers/validation",
}

// servingRoots are production request-serving packages. None of them may reach
// a lab package transitively. This includes the chains bootstrap builds, so a
// provider implementation cannot grow a dependency on corpus replay or soak.
var servingRoots = []string{
	modulePath + "/handlers/stt",
	modulePath + "/handlers/tts",
	modulePath + "/handlers/summarize",
	modulePath + "/handlers/audio",
	modulePath + "/internal/stt",
	modulePath + "/internal/tts",
	modulePath + "/internal/summarize",
	modulePath + "/internal/audio",
	modulePath + "/internal/ai/chains",
	modulePath + "/internal/ai/sttchain",
	modulePath + "/internal/ai/ttschain",
	modulePath + "/internal/ai/summarizechain",
}

// compositionExceptions are the only packages allowed to import lab code
// directly. They are the composition/mount layer, not a serving path:
//
//   - internal/bootstrap constructs the experiment manager and eval dependencies
//     when the process starts. It is the documented crossing point described in
//     README.md.
//   - internal/server mounts the experiment/eval/soak/validation handler modules
//     and re-exports eval report mapping for the composition root.
//
// Every entry is a deliberate crossing. A stale entry fails the test so the list
// cannot become a permission nobody remembers granting.
var compositionExceptions = map[string][]string{
	modulePath + "/internal/bootstrap": {
		modulePath + "/internal/experiment",
		modulePath + "/internal/experiment/evaldeps",
		modulePath + "/internal/experiment/report",
	},
	modulePath + "/internal/server": {
		modulePath + "/handlers/eval",
		modulePath + "/handlers/experiment",
		modulePath + "/handlers/soak",
		modulePath + "/handlers/validation",
		modulePath + "/internal/eval",
		modulePath + "/internal/experiment",
		modulePath + "/internal/experiment/report",
	},
}

// TestServingPackagesDoNotReachLab proves the one-directional boundary for the
// request-serving packages. It walks the production import graph (test files
// excluded) and reports the actual chain when a lab package is reachable.
func TestServingPackagesDoNotReachLab(t *testing.T) {
	root := moduleRoot(t)
	graph := newImportGraph(t, root)

	for _, prefix := range labImportPrefixes {
		if _, ok := graph.imports(prefix); !ok {
			t.Fatalf("lab package %s has no Go files at %s; update labImportPrefixes", prefix, graph.dir(prefix))
		}
	}

	for _, rootPkg := range servingRoots {
		if _, ok := graph.imports(rootPkg); !ok {
			t.Fatalf("serving package %s has no Go files; update servingRoots", rootPkg)
		}
		if chain := graph.firstLabPath(rootPkg); len(chain) > 0 {
			t.Errorf("%s reaches lab code through:\n\t%s\n\nServing code must not import lab/qualification packages; see internal/lab/README.md.", rootPkg, strings.Join(chain, "\n\t-> "))
		}
	}
}

// TestCompositionLabImportsAreExplicit proves the only crossings are the
// enumerated ones. It checks direct imports (not transitively), because the
// composition roots are allowed to mount a lab module whose own internals are
// lab-to-lab dependencies.
func TestCompositionLabImportsAreExplicit(t *testing.T) {
	root := moduleRoot(t)
	graph := newImportGraph(t, root)

	for composition, allowed := range compositionExceptions {
		imports, ok := graph.imports(composition)
		if !ok {
			t.Fatalf("composition package %s has no Go files; update compositionExceptions", composition)
		}
		actual := findLabImports(imports)
		allowedSet := make(map[string]bool, len(allowed))
		for _, pkg := range allowed {
			allowedSet[pkg] = true
		}
		for _, pkg := range actual {
			if !allowedSet[pkg] {
				t.Errorf("%s imports %s without an allowlist entry in compositionExceptions.\nIf this is a real mount point, add it with a comment naming the surface; otherwise remove the import.", composition, pkg)
			}
		}
		actualSet := make(map[string]bool, len(actual))
		for _, pkg := range actual {
			actualSet[pkg] = true
		}
		for _, pkg := range allowed {
			if !actualSet[pkg] {
				t.Errorf("%s no longer imports %s; remove the stale compositionExceptions entry so the allowlist matches reality.", composition, pkg)
			}
		}
	}
}

// importGraph is a hermetic parser-based view of the module's production import
// graph. Only intra-module imports are retained because that is all the
// boundary can govern.
type importGraph struct {
	t      *testing.T
	root   string
	fset   *token.FileSet
	cache  map[string][]string
	parsed map[string]bool
}

func newImportGraph(t *testing.T, root string) *importGraph {
	t.Helper()
	return &importGraph{
		t:      t,
		root:   root,
		fset:   token.NewFileSet(),
		cache:  make(map[string][]string),
		parsed: make(map[string]bool),
	}
}

func (g *importGraph) dir(pkgPath string) string {
	return filepath.Join(g.root, filepath.FromSlash(strings.TrimPrefix(pkgPath, modulePath)))
}

// imports returns the production imports of pkgPath and whether the package
// exists in the tree. Test files are excluded: tests may exercise lab code.
func (g *importGraph) imports(pkgPath string) ([]string, bool) {
	if g.parsed[pkgPath] {
		return g.cache[pkgPath], true
	}
	dir := g.dir(pkgPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, false
	}

	var imports []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(g.fset, filepath.Join(dir, name), nil, parser.ImportsOnly)
		if err != nil {
			g.t.Fatalf("parse %s: %v", filepath.Join(dir, name), err)
		}
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				g.t.Fatalf("unquote import %s in %s: %v", spec.Path.Value, name, err)
			}
			if path == modulePath || strings.HasPrefix(path, modulePath+"/") {
				imports = append(imports, path)
			}
		}
	}
	sort.Strings(imports)
	g.cache[pkgPath] = imports
	g.parsed[pkgPath] = true
	return imports, true
}

// firstLabPath returns the shortest import chain from start to a lab package,
// or nil when the package cannot reach lab code.
func (g *importGraph) firstLabPath(start string) []string {
	type node struct {
		path   string
		parent int
	}
	nodes := []node{{path: start, parent: -1}}
	seen := map[string]bool{start: true}

	for index := 0; index < len(nodes); index++ {
		current := nodes[index]
		imports, ok := g.imports(current.path)
		if !ok {
			continue
		}
		for _, imported := range imports {
			if isLabImport(imported) {
				chain := []string{imported}
				for at := index; at >= 0; at = nodes[at].parent {
					chain = append([]string{nodes[at].path}, chain...)
				}
				return chain
			}
			if !seen[imported] {
				seen[imported] = true
				nodes = append(nodes, node{path: imported, parent: index})
			}
		}
	}
	return nil
}

func findLabImports(imports []string) []string {
	var lab []string
	for _, imported := range imports {
		if isLabImport(imported) {
			lab = append(lab, imported)
		}
	}
	return lab
}

func isLabImport(importPath string) bool {
	for _, prefix := range labImportPrefixes {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}
	return false
}

// moduleRoot walks up from the test's working directory to the directory that
// holds go.mod for this module.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find api/go.mod above the test working directory")
		}
		dir = parent
	}
}
