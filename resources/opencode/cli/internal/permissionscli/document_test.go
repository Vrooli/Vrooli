package permissionscli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestWholeDocumentPlanAndReconcileAreIdempotent(t *testing.T) {
	h, stdout, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	document := filepath.Join(t.TempDir(), "permissions.json")
	if err := os.WriteFile(document, []byte(`{"schema_version":"v1","scope":"user","rules":[{"id":"deny-root","action":"deny","matcher":{"kind":"bash","pattern":"rm -rf /"}}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := h.Plan([]string{"--document", document, "--json"}); err != nil {
		t.Fatalf("Plan: %v", err)
	}
	stdout.Reset()
	if err := h.Reconcile([]string{"--document", document, "--json"}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	p, err := h.Adapter.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(p.BashDeny, ","), "rm -rf /"; got != want {
		t.Fatalf("deny = %q, want %q", got, want)
	}
	stdout.Reset()
	if err := h.Plan([]string{"--document", document, "--json"}); err != nil {
		t.Fatalf("second Plan: %v", err)
	}
	if !strings.Contains(stdout.String(), `"changes": []`) {
		t.Fatalf("second plan should be empty: %s", stdout.String())
	}
}
