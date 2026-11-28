package cliapptest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProductionImportViolationsOnlyFindsNonTestSource(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example/cli\n")
	write("main.go", "package main\nimport _ \"example/cli/internal/testutil/fake\"\n")
	write("main_test.go", "package main\nimport _ \"example/cli/internal/testutil/fake\"\n")
	write("internal/testutil/fake.go", "package testutil\n")
	write("vendor/dependency/fake.go", "package dependency\nimport _ \"example/cli/internal/testutil/fake\"\n")

	module, err := moduleName(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	violations, err := productionImportViolations(root, module+"/internal/testutil")
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 || violations[0] != "main.go" {
		t.Fatalf("violations = %#v, want only main.go", violations)
	}
}
