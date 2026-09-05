package testutil

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSeamRegistry(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate seam registry test")
	}
	apiRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	scenarioRoot := filepath.Dir(apiRoot)
	registryPath := filepath.Join(scenarioRoot, "docs", "internal", "SEAMS.md")
	registry, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("read seam registry: %v", err)
	}
	registered := seamRegistryEntries(string(registry))
	found := map[string]struct{}{}
	err = filepath.WalkDir(apiRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return walkErr
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if parseErr != nil {
			return parseErr
		}
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Doc == nil || !strings.Contains(general.Doc.Text(), "seam:") {
				continue
			}
			for _, spec := range general.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok {
					if _, isInterface := typeSpec.Type.(*ast.InterfaceType); isInterface {
						found[file.Name.Name+"."+typeSpec.Name.Name] = struct{}{}
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan seam declarations: %v", err)
	}
	for seam := range found {
		if _, ok := registered[seam]; !ok {
			t.Errorf("tagged seam %q is missing from %s", seam, registryPath)
		}
	}
	for seam := range registered {
		if _, ok := found[seam]; !ok {
			t.Errorf("registered seam %q has no matching // seam: interface", seam)
		}
	}
}

func seamRegistryEntries(registry string) map[string]struct{} {
	entries := map[string]struct{}{}
	inTable := false
	for _, line := range strings.Split(registry, "\n") {
		if strings.TrimSpace(line) == "## Seam Registry" {
			inTable = true
			continue
		}
		if inTable && strings.HasPrefix(strings.TrimSpace(line), "## ") {
			break
		}
		if !inTable || !strings.HasPrefix(strings.TrimSpace(line), "|") || strings.Contains(line, "---") || strings.Contains(line, "Seam |") {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) > 2 {
			entries[strings.TrimSpace(fields[1])] = struct{}{}
		}
	}
	return entries
}
