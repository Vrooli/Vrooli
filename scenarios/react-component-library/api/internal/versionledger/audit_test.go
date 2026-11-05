package versionledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAuditReleaseHashLedgerClassifiesAuthoredFilesAndIgnoresLocks(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios/react-component-library/library/components/Button/versions/1.0.0")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	matching := []byte("matching\n")
	mutated := []byte("mutated\n")
	if err := os.WriteFile(filepath.Join(dir, "Button.tsx"), matching, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "story.tsx"), mutated, 0o600); err != nil {
		t.Fatal(err)
	}
	hash := func(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
	ledger := map[string]any{"schemaVersion": 1, "entries": []map[string]string{
		{"path": "components/Button/versions/1.0.0/Button.tsx", "sha256": hash(matching)},
		{"path": "components/Button/versions/1.0.0/story.tsx", "sha256": hash([]byte("old\n"))},
		{"path": "components/Button/versions/1.0.0/removed.tsx", "sha256": hash([]byte("gone\n"))},
		{"path": "components/Button/versions/1.0.0/dependencies.json", "sha256": hash([]byte("derived\n"))},
	}}
	raw, _ := json.Marshal(ledger)
	if err := os.WriteFile(filepath.Join(root, "scenarios/react-component-library/library/released-version-hashes.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	audit, err := AuditReleaseHashLedger(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(audit.Matching) != 1 || len(audit.Mutated) != 1 || len(audit.Missing) != 1 {
		t.Fatalf("unexpected audit: %+v", audit)
	}
	if audit.Mutated[0].Asset != "Button@1.0.0" || audit.Missing[0].Asset != "Button@1.0.0" {
		t.Fatalf("asset attribution lost: %+v", audit)
	}
}
