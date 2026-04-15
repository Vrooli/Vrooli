package testkitgo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRootPackageExportedSurfaceMatchesPhase1BaseLayer(t *testing.T) {
	root := filepath.Join(ProjectRoot(t), "packages", "testkit-go")
	matches, err := filepath.Glob(filepath.Join(root, "*.go"))
	if err != nil {
		t.Fatalf("glob root package files: %v", err)
	}

	expected := []string{
		"NewRepoFixture",
		"ProjectRoot",
		"ReadJSONFile",
		"ReadJSONFileInto",
		"RepoFixture",
		"RepoFixture.WriteRepoContract",
		"RepoFixture.WriteResourceStub",
		"RepoFixture.WriteScenarioStub",
		"RepoFixtureOption",
		"ReserveFreePort",
		"WaitForFile",
		"WithScenarioDir",
		"WriteExecutable",
		"WriteExecutableOnPath",
		"WriteFile",
		"WriteJSON",
		"WriteJSONMode",
		"WriteMalformedJSON",
		"WriteRawJSON",
		"WriteRelativeExecutable",
		"WriteRelativeFile",
		"WriteRelativeMalformedJSON",
		"WriteRepoContract",
		"WriteResourceStub",
		"WriteScenarioStub",
	}

	found := make([]string, 0, len(expected))
	fset := token.NewFileSet()
	for _, path := range matches {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, decl := range file.Decls {
			switch typed := decl.(type) {
			case *ast.FuncDecl:
				if !ast.IsExported(typed.Name.Name) {
					continue
				}
				if typed.Recv == nil {
					found = append(found, typed.Name.Name)
					continue
				}
				if receiverName := exportedReceiverName(typed.Recv.List); receiverName != "" {
					found = append(found, receiverName+"."+typed.Name.Name)
				}
			case *ast.GenDecl:
				for _, spec := range typed.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if ok && ast.IsExported(typeSpec.Name.Name) {
						found = append(found, typeSpec.Name.Name)
					}
				}
			}
		}
	}

	slices.Sort(found)
	slices.Sort(expected)
	if !slices.Equal(found, expected) {
		t.Fatalf("root exported surface changed\nfound:    %v\nexpected: %v", found, expected)
	}
}

func exportedReceiverName(receivers []*ast.Field) string {
	if len(receivers) != 1 {
		return ""
	}
	return receiverTypeName(receivers[0].Type)
}

func receiverTypeName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		if ast.IsExported(typed.Name) {
			return typed.Name
		}
	case *ast.StarExpr:
		return receiverTypeName(typed.X)
	}
	return ""
}
