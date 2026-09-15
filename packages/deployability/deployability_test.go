package deployability

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestAliasSurfaceIsComplete(t *testing.T) {
	root := repositoryRoot(t)
	internalDir := filepath.Join(root, "internal", "deployability")
	declared := exportedTypeNames(t, internalDir)
	aliases := exportedAliasNames(t, filepath.Join(root, "packages", "deployability", "deployability.go"))

	var missing []string
	for name := range declared {
		if !aliases[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("exported internal deployability types missing public aliases: %s", strings.Join(missing, ", "))
	}
}

func exportedTypeNames(t *testing.T, dir string) map[string]bool {
	t.Helper()
	result := map[string]bool{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok.String() != "type" {
				continue
			}
			for _, spec := range general.Specs {
				typeSpec := spec.(*ast.TypeSpec)
				if ast.IsExported(typeSpec.Name.Name) {
					result[typeSpec.Name.Name] = true
				}
			}
		}
	}
	return result
}

func exportedAliasNames(t *testing.T, path string) map[string]bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]bool{}
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok.String() != "type" {
			continue
		}
		for _, spec := range general.Specs {
			typeSpec := spec.(*ast.TypeSpec)
			if typeSpec.Assign.IsValid() && ast.IsExported(typeSpec.Name.Name) {
				result[typeSpec.Name.Name] = true
			}
		}
	}
	return result
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".vrooli", "capability-vocabulary.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root not found")
		}
		dir = parent
	}
}
