package depsapproved

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRepositoryModuleDirRejectsPathsOutsideRepository(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, module := range []string{".", "../outside", filepath.Join(root, "outside"), ""} {
		if _, err := repositoryModuleDir(root, module); err == nil {
			t.Errorf("repositoryModuleDir(%q) accepted an invalid module path", module)
		}
	}
}

func TestRepositoryModuleDirRequiresGoMod(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := repositoryModuleDir(root, "packages/demo"); err == nil {
		t.Fatal("repositoryModuleDir accepted a module without go.mod")
	}
}

func TestRepositoryModuleDirResolvesRepositoryRelativeModule(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	module := filepath.Join(root, "packages", "demo")
	if err := os.MkdirAll(module, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := repositoryModuleDir(root, "packages/demo")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, module) {
		t.Fatalf("repositoryModuleDir() = %q, want %q", got, module)
	}
}

func TestCleanVendorEnvironmentReplacesGoModuleOverrides(t *testing.T) {
	t.Parallel()

	got := cleanVendorEnvironment([]string{"PATH=/bin", "GOWORK=/tmp/go.work", "GOFLAGS=-mod=vendor", "OTHER=value"})
	want := []string{"PATH=/bin", "OTHER=value", "GOWORK=off", "GOFLAGS="}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cleanVendorEnvironment() = %#v, want %#v", got, want)
	}
}

func TestPreserveVendorTreesRestoresNonGoInputs(t *testing.T) {
	t.Parallel()

	module := t.TempDir()
	source := filepath.Join(module, "vendor", "schemas")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "input.proto"), []byte("message Input {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	preserved, restore, err := preserveVendorTrees(module, "schemas")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(preserved, []string{"schemas"}) {
		t.Fatalf("preserved = %#v, want schemas", preserved)
	}
	if err := os.RemoveAll(filepath.Join(module, "vendor", "schemas")); err != nil {
		t.Fatal(err)
	}
	if err := restore(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(module, "vendor", "schemas", "input.proto"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "message Input {}\n" {
		t.Fatalf("restored input = %q", data)
	}
}
