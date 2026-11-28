package recovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestArtifactTempCleanupPreservesTempsWhenWriterInventoryIsUntrusted(t *testing.T) {
	root := t.TempDir()
	executionID := uuid.New()
	runDir := filepath.Join(root, executionID.String())
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		t.Fatal(err)
	}
	temp := filepath.Join(runDir, ".tmp-referenced-or-uncertain")
	if err := os.WriteFile(temp, []byte("retain until inventory is trusted"), 0o600); err != nil {
		t.Fatal(err)
	}
	inventory := filepath.Join(runDir, "purge-inventory.v1.json")
	if err := os.WriteFile(inventory, []byte(`{"schema_version":"bas-execution-directory/v1","execution_id":"`+executionID.String()+`","workflow_id":"`+uuid.NewString()+`","files":[],"artifact_bindings":[],"unexpected":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	removed, err := removeAbandonedArtifactTempFiles(root, executionID)
	if err == nil || removed != 0 {
		t.Fatalf("untrusted inventory cleanup = removed %d err %v", removed, err)
	}
	if _, err := os.Stat(temp); err != nil {
		t.Fatalf("uncertain temp was removed: %v", err)
	}
}
