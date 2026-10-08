package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var runbookPath = filepath.Join("..", "..", "docs", "guides", "recovery-runbook.md")

// TestRunbookDocumentIsRendered [REQ:STC-P0-030] keeps
// docs/guides/recovery-runbook.md equal to the rendering of the owner-verb
// table, so the published procedure and the verbs cannot drift. Regenerate
// with STC_WRITE_RUNBOOK=1.
func TestRunbookDocumentIsRendered(t *testing.T) {
	want := RenderRunbook()
	if os.Getenv("STC_WRITE_RUNBOOK") == "1" {
		if err := os.WriteFile(runbookPath, []byte(want), 0o644); err != nil { //nolint:gosec // generated documentation
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(runbookPath)
	if err != nil {
		t.Fatalf("read runbook: %v (regenerate with STC_WRITE_RUNBOOK=1)", err)
	}
	if string(got) != want {
		t.Fatalf("%s is stale; regenerate with STC_WRITE_RUNBOOK=1 go test ./backup/ -run TestRunbookDocumentIsRendered", runbookPath)
	}
	for _, verb := range []string{"cloud-target:data.backup", "cloud-target:data.restore", "cloud-target:data.verify"} {
		if !strings.Contains(want, verb) {
			t.Fatalf("runbook must name %s", verb)
		}
	}
	for _, code := range []string{"recovery_point_corrupt", "recovery_key_unavailable", "restore_target_not_clean", "rollback_incompatible", "recovery_point_protected"} {
		if !strings.Contains(want, code) {
			t.Fatalf("runbook must name refusal %s", code)
		}
	}
}
