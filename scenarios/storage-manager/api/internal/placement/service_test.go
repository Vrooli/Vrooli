package placement

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	corestorage "github.com/vrooli/api-core/storage"
)

func TestPreviewRequiresApprovalAtApplyAndAuditsVerifiedMigration(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.WriteFile(source, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	service := New(nil)
	plan, err := service.Preview(context.Background(), "cache", source, destination)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Migrate(context.Background(), plan.ID, false); err == nil {
		t.Fatal("unapproved migration succeeded")
	}
	audit, err := service.Migrate(context.Background(), plan.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if audit.Event != "migration.completed" || !audit.Verified || audit.SourcePreserved {
		t.Fatalf("audit = %+v", audit)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source remains: %v", err)
	}
	if data, err := os.ReadFile(destination); err != nil || string(data) != "payload" {
		t.Fatalf("destination = %q, err=%v", data, err)
	}
	history, err := service.Audit(context.Background(), 10)
	if err != nil || len(history) != 1 {
		t.Fatalf("history = %+v, err=%v", history, err)
	}
}

func TestVerifyUsesTargetPlatformIdentityAndDeclaredAbsence(t *testing.T) {
	service := New(nil)
	linuxOnly := corestorage.OwnerManifest{Kind: corestorage.OwnerTool, ID: "kdump-tools", Platforms: []corestorage.Platform{corestorage.PlatformLinux}, StorageEntries: []corestorage.StorageEntry{{Name: "crash_dumps", Path: corestorage.PortablePath{Value: "/var/crash"}}}}
	portable := corestorage.OwnerManifest{Kind: corestorage.OwnerTool, ID: "uv", StorageEntries: []corestorage.StorageEntry{{Name: "cache", Path: corestorage.PortablePath{Value: "$USER_CACHE_DIR/uv"}}}}
	rows := service.Verify(context.Background(), "/repo", []corestorage.OwnerManifest{linuxOnly, portable}, corestorage.PlatformWindows)
	if len(rows) != 2 {
		t.Fatalf("rows = %#v", rows)
	}
	for _, row := range rows {
		if row.Owner == "kdump-tools" {
			if row.Applicable || !row.DeclaredAbsent || row.Error != "" || !row.SyntheticIdentity {
				t.Fatalf("linux-only row = %#v", row)
			}
		}
		if row.Owner == "uv" {
			if !row.Applicable || row.Error != "" || row.Path != `C:\Users\vrooli\AppData\Local\uv` || !row.SyntheticIdentity {
				t.Fatalf("windows cache row = %#v", row)
			}
		}
	}
}
