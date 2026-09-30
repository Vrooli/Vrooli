package permissionscli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/resources/grok/cli/internal/permissions"
)

func TestWholeDocumentPlanAndReconcileAreIdempotent(t *testing.T) {
	h, stdout, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	document := filepath.Join(t.TempDir(), "permissions.json")
	if err := os.WriteFile(document, []byte(`{"schema_version":"v1","scope":"admin","rules":[{"id":"deny-root","action":"deny","matcher":{"kind":"bash","pattern":"rm -rf /"}}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := h.Plan([]string{"--scope", "admin", "--document", document, "--json"}); err != nil {
		t.Fatalf("Plan: %v", err)
	}
	stdout.Reset()
	if err := h.Reconcile([]string{"--scope", "admin", "--document", document, "--json"}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	a, _ := h.AdapterFor(permissions.ScopeAdmin)
	p, err := a.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(p.BashDeny, ","), "Bash(rm -rf /)"; got != want {
		t.Fatalf("deny = %q, want %q", got, want)
	}
	stdout.Reset()
	if err := h.Plan([]string{"--scope", "admin", "--document", document, "--json"}); err != nil {
		t.Fatalf("second Plan: %v", err)
	}
	if !strings.Contains(stdout.String(), `"changes": []`) {
		t.Fatalf("second plan should be empty: %s", stdout.String())
	}
}
