package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRepairDryRunIsBoundedAndDoesNotFollowSymlinks(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "entry"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	service := RepairService{ResolveRoot: func(string) (string, error) { return root, nil }}
	result, err := service.Repair(context.Background(), RepairRequest{
		Scope:       RepairScope{RootClass: "cache", RootPath: root},
		ExpectedUID: uint32(os.Getuid()),
		ExpectedGID: uint32(os.Getgid()),
		MaxEntries:  2,
	})
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}
	if result.Status != RepairPartial || result.Scanned != 2 {
		t.Fatalf("result = %+v, want bounded partial scan", result)
	}
	if result.Repaired != 0 {
		t.Fatalf("dry-run repaired %d entries", result.Repaired)
	}
	if data, err := os.ReadFile(outside); err != nil || string(data) != "outside" {
		t.Fatalf("outside symlink target changed: %q, %v", data, err)
	}
}

func TestRepairResumesAfterBoundedBatch(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(root, "dir", "a"), filepath.Join(root, "dir", "b"), filepath.Join(root, "other")} {
		if err := os.WriteFile(path, []byte(path), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	service := RepairService{ResolveRoot: func(string) (string, error) { return root, nil }}
	uid, gid := uint32(os.Getuid()), uint32(os.Getgid())
	first, err := service.Repair(context.Background(), RepairRequest{
		Scope:       RepairScope{RootClass: "cache", RootPath: root},
		ExpectedUID: uid,
		ExpectedGID: gid,
		MaxEntries:  2,
	})
	if err != nil {
		t.Fatalf("first Repair: %v", err)
	}
	if first.Status != RepairPartial || first.LastPath == "" {
		t.Fatalf("first result = %+v, want partial result with continuation point", first)
	}

	second, err := service.Repair(context.Background(), RepairRequest{
		Scope:       RepairScope{RootClass: "cache", RootPath: root},
		ExpectedUID: uid,
		ExpectedGID: gid,
		MaxEntries:  10,
		ResumeAfter: first.LastPath,
	})
	if err != nil {
		t.Fatalf("resumed Repair: %v", err)
	}
	if second.Status != RepairComplete || second.Scanned != 3 {
		t.Fatalf("resumed result = %+v, want complete scan of remaining entries", second)
	}
}

func TestRepairRejectsResumePathOutsideCanonicalRoot(t *testing.T) {
	root := t.TempDir()
	service := RepairService{ResolveRoot: func(string) (string, error) { return root, nil }}
	_, err := service.Repair(context.Background(), RepairRequest{
		Scope:       RepairScope{RootClass: "cache", RootPath: root},
		ExpectedUID: uint32(os.Getuid()),
		ExpectedGID: uint32(os.Getgid()),
		ResumeAfter: filepath.Join(t.TempDir(), "outside"),
	})
	if err == nil {
		t.Fatal("expected resume path outside canonical root to be rejected")
	}
}

func TestRepairRejectsNonCanonicalRootAndSymlinkFollowing(t *testing.T) {
	root := t.TempDir()
	service := RepairService{ResolveRoot: func(string) (string, error) { return root, nil }}
	_, err := service.Repair(context.Background(), RepairRequest{
		Scope:          RepairScope{RootClass: "cache", RootPath: filepath.Join(root, "child")},
		ExpectedUID:    uint32(os.Getuid()),
		ExpectedGID:    uint32(os.Getgid()),
		FollowSymlinks: true,
	})
	if err == nil {
		t.Fatal("expected symlink-following rejection")
	}
	_, err = service.Repair(context.Background(), RepairRequest{
		Scope:       RepairScope{RootClass: "cache", RootPath: filepath.Join(t.TempDir(), "outside")},
		ExpectedUID: uint32(os.Getuid()),
		ExpectedGID: uint32(os.Getgid()),
	})
	if err == nil {
		t.Fatal("expected non-canonical root rejection")
	}
}
