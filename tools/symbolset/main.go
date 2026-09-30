// Command symbolset prints a filename-independent declaration inventory for a
// Go package. It is the split protocol's mechanical guard: moving a whole
// declaration between files must not change the sorted symbol set.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const symbolsetArgumentCount = 2

func main() {
	if len(os.Args) != symbolsetArgumentCount {
		fmt.Fprintln(os.Stderr, "usage: symbolset <go-package-directory>")
		os.Exit(symbolsetArgumentCount)
	}
	symbols, err := collectSymbols(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, symbol := range symbols {
		fmt.Println(symbol)
	}
}

func collectSymbols(root string) ([]string, error) {
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat package %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("package path %s is not a directory", root)
	}
	fset := token.NewFileSet()
	var files []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "vendor" || strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk package %s: %w", root, err)
	}
	sort.Strings(files)
	set := make(map[string]struct{})
	for _, path := range files {
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				if decl.Recv == nil {
					set["func "+decl.Name.Name] = struct{}{}
					continue
				}
				for _, field := range decl.Recv.List {
					set["method "+receiverName(field.Type)+" "+decl.Name.Name] = struct{}{}
				}
			case *ast.GenDecl:
				kind := ""
				switch decl.Tok {
				case token.CONST:
					kind = "const"
				case token.VAR:
					kind = "var"
				case token.TYPE:
					kind = "type"
				}
				if kind == "" {
					continue
				}
				for _, spec := range decl.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						set[kind+" "+spec.Name.Name] = struct{}{}
					case *ast.ValueSpec:
						for _, name := range spec.Names {
							set[kind+" "+name.Name] = struct{}{}
						}
					}
				}
			}
		}
	}
	symbols := make([]string, 0, len(set))
	for symbol := range set {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)
	return symbols, nil
}

func receiverName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.StarExpr:
		return "*" + receiverName(expr.X)
	case *ast.IndexExpr:
		return receiverName(expr.X)
	case *ast.IndexListExpr:
		return receiverName(expr.X)
	default:
		return "?"
	}
}
