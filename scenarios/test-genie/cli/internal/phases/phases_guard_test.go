package phases

import (
	"testing"

	catalog "test-genie/internal/orchestrator/phases"
)

// TestAllowedPhasesMatchCatalog is the anti-drift guard: the CLI's allowed phase
// set must equal the catalog phase set. If a phase is added/removed in the
// catalog, this fails until the CLI tracks it (which it now does automatically).
func TestAllowedPhasesMatchCatalog(t *testing.T) {
	want := make(map[string]struct{})
	for _, n := range catalog.ValidPhaseNames() {
		want[n] = struct{}{}
	}
	got := make(map[string]struct{})
	for _, n := range allowedPhaseNames() {
		got[n] = struct{}{}
	}
	if len(got) != len(want) {
		t.Fatalf("CLI allowed phase count = %d, catalog count = %d", len(got), len(want))
	}
	for n := range want {
		if _, ok := got[n]; !ok {
			t.Errorf("catalog phase %q missing from CLI allowed set", n)
		}
	}
	for n := range got {
		if _, ok := want[n]; !ok {
			t.Errorf("CLI allowed phase %q is not a catalog phase", n)
		}
	}
}

// TestE2EAliasStillValidates ensures the e2e→workflow alias still resolves even
// though "e2e" is not a catalog phase name.
func TestE2EAliasStillValidates(t *testing.T) {
	got, err := NormalizeSelection([]string{"e2e"})
	if err != nil {
		t.Fatalf("e2e should validate via alias: %v", err)
	}
	if len(got) != 1 || got[0] != "workflow" {
		t.Fatalf("e2e should normalize to [workflow], got %v", got)
	}
}

// TestPreviouslyMissingPhasesValidate locks in the bug fix: ui-health, contracts,
// and performance were absent from the hand-maintained list and could not be
// named via the CLI. They must now validate.
func TestPreviouslyMissingPhasesValidate(t *testing.T) {
	for _, p := range []string{"ui-health", "contracts", "performance"} {
		got, err := NormalizeSelection([]string{p})
		if err != nil {
			t.Errorf("phase %q should validate via CLI: %v", p, err)
			continue
		}
		if len(got) != 1 || got[0] != p {
			t.Errorf("phase %q should normalize to [%s], got %v", p, p, got)
		}
	}
}
