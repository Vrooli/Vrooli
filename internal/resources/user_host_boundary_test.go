package resources

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestUserHostPackagesStayInsideTheResourceControlPlane protects the explicit
// architecture boundary: user-host and Vault bootstrap authority cannot grow
// a dependency on API or CLI scaffolding.
func TestUserHostPackagesStayInsideTheResourceControlPlane(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	files := []string{
		"user_host.go",
		"vault_bootstrap.go",
		filepath.Join("securestore", "store.go"),
		filepath.Join("securestore", "store_linux.go"),
		filepath.Join("securestore", "store_darwin.go"),
		filepath.Join("securestore", "store_windows.go"),
	}
	for _, name := range files {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(filepath.Dir(testFile), name), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imported := range file.Imports {
			path := strings.Trim(imported.Path.Value, `"`)
			if strings.Contains(path, "/api-core") || strings.Contains(path, "/cli-core") {
				t.Fatalf("%s imports forbidden boundary package %q", name, path)
			}
		}
	}
}

func TestGenericUserHostContainsNoVaultSpecificLifecycle(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(testFile), "user_host.go"))
	if err != nil {
		t.Fatalf("read user host source: %v", err)
	}
	if strings.Contains(strings.ToLower(string(data)), "vault") {
		t.Fatal("generic user_host.go must not contain Vault-specific lifecycle or credential logic")
	}
}
