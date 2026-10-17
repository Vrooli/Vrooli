package dependencyhealth

import (
	"os"
	"path/filepath"
	"testing"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

func TestNodeDependencyDriftFixture(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"dependencies":{"vite":"^7.0.0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pnpm-lock.yaml"), []byte("lockfileVersion: '9.0'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := nodeDependencyDrift(&healthv1.DependencyHealthSurface{RootPath: root, PackageManager: "pnpm"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("minimal lock drift = %v", got)
	}
	if err := os.WriteFile(filepath.Join(root, "pnpm-lock.yaml"), []byte("lockfileVersion: '9.0'\nspecifiers:\n  vite: ^6.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = nodeDependencyDrift(&healthv1.DependencyHealthSurface{RootPath: root, PackageManager: "pnpm"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "vite" {
		t.Fatalf("drift = %v, want vite", got)
	}
}
