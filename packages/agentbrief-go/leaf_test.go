package agentbrief

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLeafDoesNotEnterSharedRuntimePackages(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	for _, relative := range []string{"packages/api-core", "packages/cli-core", "packages/agentharness"} {
		root := filepath.Join(repoRoot, relative)
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, spec := range file.Imports {
				if strings.Trim(spec.Path.Value, `"`) == "github.com/vrooli/agentbrief-go" {
					t.Errorf("forbidden leaf import in %s", path)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", relative, err)
		}
	}
}
