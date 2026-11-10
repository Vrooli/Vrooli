package execution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileStoreRejectsUnknownPersistedExecutionStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "executions.json")
	if err := os.WriteFile(path, []byte(`[{"execution_id":"exec-unknown","status":"future_state"}]`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := NewStore(path).Load()
	if err == nil || !strings.Contains(err.Error(), `record "exec-unknown" has unknown status "future_state"`) {
		t.Fatalf("Load error = %v, want unknown persisted status", err)
	}
}

func TestFileStoreRefusesToPersistUnknownExecutionStatus(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "executions.json"))
	err := store.Save([]Record{{ExecutionID: "exec-unknown", Status: Status("future_state")}})
	if err == nil || !strings.Contains(err.Error(), `record "exec-unknown" has unknown status "future_state"`) {
		t.Fatalf("Save error = %v, want unknown persisted status", err)
	}
}
