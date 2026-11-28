package agentharness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsPermissionPublicationPreservesPrivateACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	before, err := ReadPermissionFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = PublishPermissionFile(before, []byte("private")); err != nil {
		t.Fatal(err)
	}
	before, err = ReadPermissionFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(before.Security, "D:P") || !strings.Contains(before.Security, "OW") {
		t.Fatalf("new file is not owner-only: %s", before.Security)
	}
	if err = PublishPermissionFile(before, []byte("updated")); err != nil {
		t.Fatal(err)
	}
	after, err := ReadPermissionFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if before.Security != after.Security {
		t.Fatalf("access ACL changed: %s -> %s", before.Security, after.Security)
	}
	backup, err := os.CreateTemp(filepath.Dir(path), "backup-")
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	if err = SecurePermissionFile(backup.Name(), ""); err != nil {
		t.Fatal(err)
	}
	security, err := permissionFileSecurity(backup.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(security, "D:P") || !strings.Contains(security, "OW") {
		t.Fatalf("backup ACL is not private: %s", security)
	}
}
