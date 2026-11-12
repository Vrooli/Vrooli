//go:build darwin && cgo

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDarwinDirectorySubstitutionCannotNormalizeForeignACL(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	foreignUID := 0
	if os.Geteuid() == 0 {
		foreignUID = 1
	}
	explicit := PrivatePathPolicy{OwnerPrincipal: fmt.Sprintf("uid:%d", os.Geteuid()), RuntimePrincipal: fmt.Sprintf("uid:%d", foreignUID)}
	substitute := filepath.Join(root, "substitute")
	if err = ConfigurePrivateDirectoryAccess(substitute, explicit); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "target")
	opener := func(path string, _ bool) (*os.File, bool, error) {
		if err := os.Mkdir(path, 0o700); err != nil {
			return nil, false, err
		}
		if err := os.Rename(path, path+"-original"); err != nil {
			return nil, false, err
		}
		if err := os.Rename(substitute, path); err != nil {
			return nil, false, err
		}
		file, _, err := openDarwinPrivateDirectory(path, false)
		return file, true, err
	}
	if err = ensureDarwinPrivateDirectoryWith(target, PrivatePathPolicy{}, opener); err == nil {
		t.Fatal("substituted directory with an unapproved principal was normalized")
	}
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	if err = VerifyPrivatePath(target, info, explicit); err != nil {
		t.Fatal("rejected substitution changed its original ACL", err)
	}
}
