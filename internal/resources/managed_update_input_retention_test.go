package resources

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagedUpdatePrivateInputRemovalRefusesAliasedOwnerPaths(t *testing.T) {
	for _, parentAlias := range []bool{false, true} {
		t.Run(map[bool]string{false: "leaf", true: "ancestor"}[parentAlias], func(t *testing.T) {
			base, outside := t.TempDir(), t.TempDir()
			victim := filepath.Join(outside, "runtime")
			if err := os.Mkdir(victim, 0o700); err != nil {
				t.Fatal(err)
			}
			secret := filepath.Join(victim, "credential")
			if err := os.WriteFile(secret, []byte("private fixture input"), 0o600); err != nil {
				t.Fatal(err)
			}
			alias := filepath.Join(base, "role")
			target, removal := victim, alias
			if parentAlias {
				target, removal = outside, filepath.Join(alias, "runtime")
			}
			if err := os.Symlink(target, alias); err != nil {
				t.Fatal(err)
			}
			if err := removeManagedUpdatePrivateTree(base, removal); err == nil {
				t.Fatal("private cleanup accepted an aliased owner path")
			}
			if _, err := os.Stat(secret); err != nil {
				t.Fatalf("cleanup changed another owner's input: %v", err)
			}
		})
	}
}
