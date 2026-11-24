package lab_test

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
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
	modulePath + "/internal/corpus",
	modulePath + "/handlers/corpus",
	modulePath + "/internal/eval",
	modulePath + "/internal/experiment",
	modulePath + "/internal/soak",
	modulePath + "/handlers/eval",
	modulePath + "/handlers/experiment",
	modulePath + "/handlers/soak",
	modulePath + "/handlers/validation",
}

func TestPackagePatternSeparatesProductAndLabProfiles(t *testing.T) {
	root := moduleRoot(t)
	list := func(tags ...string) map[string]bool {
		t.Helper()
		args := []string{"list"}
		if len(tags) > 0 {
			args = append(args, "-tags", strings.Join(tags, ","))
		}
		args = append(args, "-f", "{{.ImportPath}}", "./...")
		cmd := exec.Command("go", args...)
		cmd.Dir = root
		for _, item := range os.Environ() {
			if !strings.HasPrefix(item, "GOFLAGS=") {
				cmd.Env = append(cmd.Env, item)
			}
		}
		cmd.Env = append(cmd.Env, "GOFLAGS=")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("list packages with tags %v: %v\n%s", tags, err, output)
		}
		packages := make(map[string]bool)
		for _, pkg := range strings.Fields(string(output)) {
			packages[pkg] = true
		}
		return packages
	}

	product := list()
	lab := list("audio_lab")
	for _, prefix := range labImportPrefixes {
		if product[prefix] {
			t.Errorf("product package pattern includes lab package %s; lab-only packages must require audio_lab", prefix)
		}
		if !lab[prefix] {
			t.Errorf("audio_lab package pattern omits lab package %s", prefix)
		}
	}
	for _, pkg := range labPackages(product) {
		t.Errorf("product package pattern includes lab package %s; every package below a lab prefix must require audio_lab", pkg)
	}
}

func TestLabPackageClassificationIncludesDescendants(t *testing.T) {
	product := map[string]bool{
		"audio-tools/internal/eval":            true,
		"audio-tools/internal/experiment":      true,
		"audio-tools/internal/experiment/fast": true,
		"audio-tools/internal/stt":             true,
	}
	want := []string{
		"audio-tools/internal/eval",
		"audio-tools/internal/experiment",
		"audio-tools/internal/experiment/fast",
	}
	if got := labPackages(product); strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("labPackages() = %v, want %v", got, want)
	}
}

func labPackages(packages map[string]bool) []string {
	var lab []string
	for pkg := range packages {
		if isLabImport(pkg) {
			lab = append(lab, pkg)
		}
	}
	sort.Strings(lab)
	return lab
}

// servingRoots are production request-serving packages. None of them may reach
// a lab package transitively. This includes the chains bootstrap builds, so a
// provider implementation cannot grow a dependency on corpus replay or soak.
var servingRoots = []string{
	modulePath + "/handlers/stt",
	modulePath + "/handlers/tts",
	modulePath + "/handlers/summarize",
	modulePath + "/handlers/audio",
	modulePath + "/handlers/diagnostics",
	modulePath + "/handlers/models",
	modulePath + "/internal/stt",
	modulePath + "/internal/tts",
	modulePath + "/internal/summarize",
	modulePath + "/internal/audio",
	modulePath + "/internal/diagnostics",
	modulePath + "/internal/modelpromotion",
	modulePath + "/internal/ai/chains",
	modulePath + "/internal/ai/sttchain",
	modulePath + "/internal/ai/ttschain",
	modulePath + "/internal/ai/summarizechain",
}

// compositionExceptions are the only packages allowed to import lab code
// directly. They are the composition/mount layer, not a serving path:
//
//   - internal/bootstrap constructs the experiment manager and eval dependencies
//     when the process starts, then converts eval reports for retention-aware
//     persistence. It is the documented crossing point described in README.md.
//   - internal/server mounts corpus and experiment/eval/soak/validation handlers
//     and re-exports eval report mapping for the composition root.
//   - internal/modules enumerates lab endpoint and schema metadata for startup
//     migration and endpoint generation. It does not mount request handlers.
//
// Every entry is a deliberate crossing. A stale entry fails the test so the list
// cannot become a permission nobody remembers granting.
var compositionExceptions = map[string][]string{
	modulePath + "/internal/bootstrap": {
		modulePath + "/internal/corpus",
		modulePath + "/internal/eval",
		modulePath + "/internal/experiment",
		modulePath + "/internal/experiment/evaldeps",
		modulePath + "/internal/experiment/report",
	},
	modulePath + "/internal/server": {
		modulePath + "/handlers/corpus",
		modulePath + "/handlers/eval",
		modulePath + "/handlers/experiment",
		modulePath + "/handlers/soak",
		modulePath + "/handlers/validation",
		modulePath + "/internal/corpus",
		modulePath + "/internal/eval",
		modulePath + "/internal/experiment",
		modulePath + "/internal/experiment/report",
	},
	modulePath + "/internal/modules": {
		modulePath + "/handlers/corpus",
		modulePath + "/handlers/experiment",
		modulePath + "/handlers/soak",
		modulePath + "/handlers/validation",
		modulePath + "/internal/corpus",
		modulePath + "/internal/experiment",
	},
}

// TestServingPackagesDoNotReachLab proves the one-directional boundary for the
// request-serving packages. It walks the production import graph (test files
// excluded) and reports the actual chain when a lab package is reachable.
func TestServingPackagesDoNotReachLab(t *testing.T) {
	root := moduleRoot(t)
	graph := newImportGraph(t, root)
	serving := append(append([]string(nil), servingRoots...), servingHandlerRoots(t, root)...)

	for _, prefix := range labImportPrefixes {
		if _, ok := graph.imports(prefix); !ok {
			t.Fatalf("lab package %s has no Go files at %s; update labImportPrefixes", prefix, graph.dir(prefix))
		}
	}

	for _, rootPkg := range serving {
		if _, ok := graph.imports(rootPkg); !ok {
			t.Fatalf("serving package %s has no Go files; update servingRoots", rootPkg)
		}
		if chain := graph.firstLabPath(rootPkg); len(chain) > 0 {
			t.Errorf("%s reaches lab code through:\n\t%s\n\nServing code must not import lab/qualification packages; see internal/lab/README.md.", rootPkg, strings.Join(chain, "\n\t-> "))
		}
	}
}

func servingHandlerRoots(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "handlers"))
	if err != nil {
		t.Fatalf("read handler roots: %v", err)
	}
	var roots []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pkg := modulePath + "/handlers/" + entry.Name()
		if !isLabImport(pkg) {
			roots = append(roots, pkg)
		}
	}
	sort.Strings(roots)
	return roots
}

func TestImportGraphFindsDirectAndTransitiveLabImports(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{
			name: "direct serving import",
			files: map[string]string{
				"handlers/stt/handler.go": `package stt
import "audio-tools/internal/eval"
var _ = eval.DefaultNormalizeOptions`,
			},
			want: []string{"audio-tools/handlers/stt", "audio-tools/internal/eval"},
		},
		{
			name: "transitive serving import",
			files: map[string]string{
				"handlers/models/handler.go": `package models
import "audio-tools/internal/modelpromotion"
var _ = modelpromotion.Resolve`,
				"internal/modelpromotion/resolver.go": `package modelpromotion
import "audio-tools/internal/experiment"
var _ = experiment.QualificationEvidence{}`,
			},
			want: []string{"audio-tools/handlers/models", "audio-tools/internal/modelpromotion", "audio-tools/internal/experiment"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for path, source := range tc.files {
				fullPath := filepath.Join(root, filepath.FromSlash(path))
				if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(fullPath, []byte(source), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			graph := newImportGraph(t, root)
			got := graph.firstLabPath(tc.want[0])
			if strings.Join(got, "\x00") != strings.Join(tc.want, "\x00") {
				t.Fatalf("firstLabPath() = %v, want %v", got, tc.want)
			}
		})
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
