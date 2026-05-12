package quint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"flow-verifier/internal/flows/contract"
	"flow-verifier/internal/flows/model"
)

func TestNormalizeTracesRejectsUnknownTags(t *testing.T) {
	dir := t.TempDir()
	body := `{"states":[{"status":{"tag":"Idle"},"event":{"tag":"Tick"},"rejected":false},{"status":{"tag":"Ghost"},"event":{"tag":"Tick"},"rejected":false}]}`
	if err := os.WriteFile(filepath.Join(dir, "trace.itf.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := NormalizeTraces(model.Flow{
		States: []contract.State{{ID: "idle", Quint: "Idle"}},
		Events: []contract.Event{{ID: "tick", Quint: "Tick"}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "unknown status tag") {
		t.Fatalf("NormalizeTraces() error = %v", err)
	}
}
