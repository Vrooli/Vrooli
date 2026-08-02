package placement

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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
