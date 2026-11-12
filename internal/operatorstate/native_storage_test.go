package operatorstate

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/config"
)

func TestLogicalOperatorHomeIgnoresMachineNamespace(t *testing.T) {
	home := t.TempDir()
	machine := t.TempDir()
	t.Setenv("HOME", machine)
	t.Setenv("VROOLI_HOME", filepath.Join(machine, "runtime"))
	t.Setenv("VROOLI_STORAGE_ROOT", machine)
	s := New(Config{LogicalOperatorHome: home})
	path, err := s.Path(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, home+string(filepath.Separator)) || strings.HasPrefix(path, machine+string(filepath.Separator)) {
		t.Fatalf("operator path escaped logical home: %s", path)
	}
	if _, err = os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("path resolution created storage: %v", err)
	}
	p, err := s.nativeStatePolicy()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(home)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := config.PrivatePathOwner(home, info)
	if err != nil {
		t.Fatal(err)
	}
	if p.OwnerPrincipal != owner || p.RuntimePrincipal != "" {
		t.Fatalf("operator policy %+v", p)
	}
}

func TestNativeOperatorMissingLoadHasNoEffects(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("native storage boundary")
	}
	home := t.TempDir()
	path := filepath.Join(home, "missing", "state.json")
	s := New(Config{LogicalOperatorHome: home, StatePath: func(context.Context) (string, error) { return path, nil }})
	got, err := s.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if Revision(got) != Revision(Default()) {
		t.Fatal("missing state differs from default")
	}
	if _, err = os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("missing read created storage: %v", err)
	}
}
