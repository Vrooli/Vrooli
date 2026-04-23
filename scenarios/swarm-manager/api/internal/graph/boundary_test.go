package graph

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Plan §W2 calls for a boundary rule: `graph.json` is read-only to agents
// and written only by the materialization pipeline. These tests pin that
// rule structurally — adding a new exported way to mutate graph.json will
// fail CI until the allowlists below are updated, forcing a deliberate
// review.
//
// The tests are intentionally reflective / source-scanning rather than
// behavioral: the property we care about is "no other way exists", which
// cannot be proven by exercising one path.

// materializerExportedMethods is the exhaustive, sorted list of exported
// methods permitted on *Materializer. If this list needs to grow, update it
// here AND audit callers — every exported method on the materializer is a
// potential graph.json write vector.
var materializerExportedMethods = []string{
	"MaterializeAll",
	"MaterializeInitiative",
	"ReadGraph",
	"ScheduleAll",
}

// TestBoundary_MaterializerAPISurface pins the exported method set on
// *Materializer. New exported methods must be added here intentionally.
func TestBoundary_MaterializerAPISurface(t *testing.T) {
	var m *Materializer
	ty := reflect.TypeOf(m)
	got := make([]string, 0, ty.NumMethod())
	for i := 0; i < ty.NumMethod(); i++ {
		got = append(got, ty.Method(i).Name)
	}
	sort.Strings(got)
	want := append([]string(nil), materializerExportedMethods...)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Materializer exported API changed\n got:  %v\n want: %v\n"+
			"If you are adding a new method, ensure it does NOT bypass the\n"+
			"read-only-to-agents boundary and update materializerExportedMethods.",
			got, want)
	}
}

// TestBoundary_NoExportedTopLevelGraphReader guards against a top-level
// `ReadXxx` helper sneaking in alongside the `*Materializer.ReadGraph`
// method. Callers should load graph.json through the materializer so we
// retain a single choke point — if a package-level shortcut appeared, it
// would bypass any future caching or audit the materializer grows.
func TestBoundary_NoExportedTopLevelGraphReader(t *testing.T) {
	exported := exportedTopLevelFuncs(t, "Read")
	if len(exported) != 0 {
		t.Fatalf("exported top-level Read* funcs = %v; want none (use *Materializer.ReadGraph)", exported)
	}
}

// TestBoundary_NoExportedGraphWriter scans package sources for any exported
// top-level identifier whose name contains `Write` AND signature references
// MaterializedGraph, graphJSONFilename, or graph.json.
//
// The intent: writing graph.json is privileged. Only the unexported
// `writeGraph` method on *Materializer is allowed to do it. An exported
// `WriteGraph(name, MaterializedGraph)` would let any caller clobber a
// projection, making every assumption about graph.json freshness invalid.
func TestBoundary_NoExportedGraphWriter(t *testing.T) {
	files := collectPackageGoFiles(t)
	fset := token.NewFileSet()
	var offenders []string
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		file, err := parser.ParseFile(fset, path, src, parser.AllErrors)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Name == nil {
				return true
			}
			name := fn.Name.Name
			if !ast.IsExported(name) {
				return true
			}
			if !strings.Contains(name, "Write") {
				return true
			}
			// Any exported Write* function in this package is suspect.
			// (Currently none exist; add to an allowlist below if one is
			// ever added intentionally.)
			offenders = append(offenders, filepath.Base(path)+":"+name)
			return true
		})
	}
	if len(offenders) > 0 {
		t.Fatalf("graph package exports Write* identifiers (boundary violation): %v\n"+
			"graph.json must be written only via the unexported writeGraph method.",
			offenders)
	}
}

// TestBoundary_WriteGraphMethodIsUnexported asserts the Materializer has a
// lowercase `writeGraph` method and no exported equivalent. Belt-and-braces
// over TestBoundary_MaterializerAPISurface: a future refactor that renames
// `writeGraph` to `Write` or splits it into an exported helper would fail
// here first with a clearer message.
func TestBoundary_WriteGraphMethodIsUnexported(t *testing.T) {
	files := collectPackageGoFiles(t)
	fset := token.NewFileSet()
	var (
		sawUnexported bool
		exportedHits  []string
	)
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		file, err := parser.ParseFile(fset, path, src, parser.AllErrors)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name == nil {
				return true
			}
			// Only methods on *Materializer (or Materializer) are in scope.
			if !methodHasReceiverType(fn, "Materializer") {
				return true
			}
			name := fn.Name.Name
			if name == "writeGraph" {
				sawUnexported = true
			}
			if ast.IsExported(name) && strings.HasPrefix(strings.ToLower(name), "write") {
				exportedHits = append(exportedHits, filepath.Base(path)+":"+name)
			}
			return true
		})
	}
	if !sawUnexported {
		t.Fatalf("expected unexported writeGraph method on *Materializer; none found")
	}
	if len(exportedHits) > 0 {
		t.Fatalf("found exported Write* methods on *Materializer: %v", exportedHits)
	}
}

// exportedTopLevelFuncs returns exported top-level function names in the
// graph package whose name starts with the given prefix (exact prefix, not
// just "contains").
func exportedTopLevelFuncs(t *testing.T, prefix string) []string {
	t.Helper()
	files := collectPackageGoFiles(t)
	fset := token.NewFileSet()
	var names []string
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		file, err := parser.ParseFile(fset, path, src, parser.AllErrors)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name == nil {
				continue
			}
			name := fn.Name.Name
			if !ast.IsExported(name) {
				continue
			}
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// collectPackageGoFiles lists the non-test .go files that make up the
// `graph` package (this test's own package). Boundary assertions scan
// production source only — *_test.go may freely reference unexported
// identifiers and is not a risk to external callers.
func collectPackageGoFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, name)
	}
	if len(files) == 0 {
		t.Fatalf("no non-test .go files found in graph package")
	}
	return files
}

// methodHasReceiverType reports whether fn's receiver points at (or is)
// a type named typeName. Handles both `(m *Materializer)` and
// `(m Materializer)` forms.
func methodHasReceiverType(fn *ast.FuncDecl, typeName string) bool {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return false
	}
	expr := fn.Recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == typeName
}
