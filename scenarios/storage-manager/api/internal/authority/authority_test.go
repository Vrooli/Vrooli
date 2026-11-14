package authority

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestHostScopePermitsEverything(t *testing.T) {
	scope := Host()
	if !scope.IsHost() || scope.CheckPath("/var/lib/anything") != nil || scope.CheckHostAction("docker image prune") != nil {
		t.Fatal("host authority refused a mutation")
	}
}

func TestRedirectedScopeConfinesMutationsToItsRoot(t *testing.T) {
	root := t.TempDir()
	scope := Redirected(root)
	if scope.IsHost() {
		t.Fatal("a redirected instance claimed host authority")
	}
	for _, inside := range []string{root, filepath.Join(root, "data", "vrooli", "demo", "x.db")} {
		if err := scope.CheckPath(inside); err != nil {
			t.Fatalf("CheckPath(%q) = %v, want allowed", inside, err)
		}
	}
	for _, outside := range []string{filepath.Dir(root), root + "-sibling", filepath.Join(root, "..", "escape"), "/home/user/.local/state/vrooli/resource/ollama"} {
		if err := scope.CheckPath(outside); !errors.Is(err, ErrOutOfScope) {
			t.Fatalf("CheckPath(%q) = %v, want ErrOutOfScope", outside, err)
		}
	}
	if err := scope.CheckHostAction("owner reclaim"); !errors.Is(err, ErrOutOfScope) {
		t.Fatalf("CheckHostAction = %v, want ErrOutOfScope", err)
	}
}

func TestZeroScopeFollowsTheEnvironment(t *testing.T) {
	var scope Scope
	t.Setenv(StorageRootEnv, "")
	if !scope.IsHost() {
		t.Fatal("zero scope without an override is not host authority")
	}
	root := t.TempDir()
	t.Setenv(StorageRootEnv, root)
	if scope.IsHost() || !errors.Is(scope.CheckPath("/etc/passwd"), ErrOutOfScope) {
		t.Fatal("zero scope ignored VROOLI_STORAGE_ROOT")
	}
}
