package transcriptredact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactPreservesStructureWhileRemovingSensitiveValues(t *testing.T) {
	raw := `{"api_key":"fixture", "token=demo", "command":"tool --token fixture", "path":"/home/alice/.codex/session.jsonl", "input_tokens":5}`
	redacted := Redact(raw)
	for _, forbidden := range []string{"fixture", "demo", "/home/alice"} {
		if strings.Contains(redacted, forbidden) {
			t.Fatalf("redaction retained %q: %s", forbidden, redacted)
		}
	}
	if !strings.Contains(redacted, `"input_tokens":5`) {
		t.Fatalf("redaction altered classification-relevant telemetry: %s", redacted)
	}
}

func TestScanDirRejectsUnsafeFixture(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "unsafe.jsonl"), []byte(`{"password":"not-for-commit"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	violations, err := ScanDir(root)
	if err != nil || len(violations) != 1 || !strings.Contains(violations[0], "unsafe.jsonl") {
		t.Fatalf("ScanDir() = %v, %v; want unsafe fixture violation", violations, err)
	}
}
