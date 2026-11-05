package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const modulePath = "scenario-to-cloud"

// sourceFile is one parsed, non-test Go file of the module.
type sourceFile struct {
	// Rel is the module-relative path (for example "reach/reach.go").
	Rel string
	// Pkg is the module-relative package directory ("" for the root package).
	Pkg string
	// ImportPath is the import path of the package ("scenario-to-cloud/reach").
	ImportPath string
	File       *ast.File
	Fset       *token.FileSet
	Imports    []string
}

var (
	loadOnce   sync.Once
	loadedRoot string
	loadedSrc  []sourceFile
	loadErr    error
)

// moduleRoot is the api module directory.
func moduleRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("module root %s has no go.mod: %v", root, err)
	}
	return root
}

// scenarioRoot is the scenario directory (parent of the api module).
func scenarioRoot(t *testing.T) string {
	t.Helper()
	return filepath.Dir(moduleRoot(t))
}

// loadSources parses every non-test Go file of the module once. Vendored,
// generated proto and testdata trees are skipped; the module has none today,
// but the rule keeps the census honest if one appears.
func loadSources(t *testing.T) (string, []sourceFile) {
	t.Helper()
	loadOnce.Do(func() {
		root, err := filepath.Abs("..")
		if err != nil {
			loadErr = err
			return
		}
		loadedRoot = root
		fset := token.NewFileSet()
		loadErr = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				name := d.Name()
				if path != root && (strings.HasPrefix(name, ".") || name == "testdata" || name == "vendor" || name == "node_modules") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			pkg := filepath.ToSlash(filepath.Dir(rel))
			if pkg == "." {
				pkg = ""
			}
			importPath := modulePath
			if pkg != "" {
				importPath += "/" + pkg
			}
			sf := sourceFile{Rel: rel, Pkg: pkg, ImportPath: importPath, File: file, Fset: fset}
			for _, imp := range file.Imports {
				value, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					return err
				}
				sf.Imports = append(sf.Imports, value)
			}
			loadedSrc = append(loadedSrc, sf)
			return nil
		})
		sort.Slice(loadedSrc, func(i, j int) bool { return loadedSrc[i].Rel < loadedSrc[j].Rel })
	})
	if loadErr != nil {
		t.Fatalf("parse module sources: %v", loadErr)
	}
	if len(loadedSrc) < 50 {
		t.Fatalf("parsed only %d source files; the module walk is broken", len(loadedSrc))
	}
	return loadedRoot, loadedSrc
}

// importsPackage reports whether the file imports the given import path.
func (f sourceFile) importsPackage(path string) bool {
	for _, imp := range f.Imports {
		if imp == path {
			return true
		}
	}
	return false
}

// position renders file:line for a node.
func (f sourceFile) position(node ast.Node) string {
	pos := f.Fset.Position(node.Pos())
	return f.Rel + ":" + strconv.Itoa(pos.Line)
}

// packagesImporting returns the sorted set of package directories whose
// non-test files import the path.
func packagesImporting(src []sourceFile, path string) []string {
	seen := map[string]struct{}{}
	for _, f := range src {
		if f.importsPackage(path) {
			seen[f.Pkg] = struct{}{}
		}
	}
	return sortedKeys(seen)
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// stringLiterals visits every string literal in a file with its value.
func (f sourceFile) stringLiterals(visit func(lit *ast.BasicLit, value string)) {
	ast.Inspect(f.File, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		visit(lit, value)
		return true
	})
}

// selectorCalls visits every call of the form <expr>.<Name>(...) whose
// selector name is in names.
func (f sourceFile) selectorCalls(names map[string]struct{}, visit func(call *ast.CallExpr, name string)) {
	f.selectorCallsIn(f.File, names, visit)
}

// selectorCallsIn is selectorCalls restricted to one subtree.
func (f sourceFile) selectorCallsIn(root ast.Node, names map[string]struct{}, visit func(call *ast.CallExpr, name string)) {
	ast.Inspect(root, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if _, want := names[sel.Sel.Name]; want {
			visit(call, sel.Sel.Name)
		}
		return true
	})
}

// funcBodies visits every function or method declaration with its receiver
// type name ("" for plain functions).
func (f sourceFile) funcBodies(visit func(decl *ast.FuncDecl, receiver string)) {
	for _, decl := range f.File.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		receiver := ""
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			receiver = typeName(fn.Recv.List[0].Type)
		}
		visit(fn, receiver)
	}
}

func typeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return typeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return typeName(t.X)
	}
	return ""
}
