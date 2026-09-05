package domains_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"architecture-cartographer/internal/domains"
)

// scenarioRoot is the cartographer scenario root relative to this test's
// package directory (api/internal/domains).
const scenarioRoot = "../../.."

// TestDogfood_OwnDomainsDocParses asserts cartographer's own structured
// DOMAINS.md conforms to the machine contract: it parses without error and
// the DOMAINS.md rung wins authority over the folder/CLI rungs, declaring a
// non-trivial domain set. This is the R1 brittleness guard — if a future
// edit breaks the table shape, this fails fast.
func TestDogfood_OwnDomainsDocParses(t *testing.T) {
	dir, err := filepath.Abs(scenarioRoot)
	if err != nil {
		t.Fatalf("abs scenario root: %v", err)
	}

	extractions, err := domains.RunLadder(context.Background(), dir, domains.DefaultExtractors())
	if err != nil {
		t.Fatalf("run ladder over own scenario: %v", err)
	}
	m, err := domains.Resolve("architecture-cartographer", extractions, time.Time{})
	if err != nil {
		t.Fatalf("resolve own domain map: %v", err)
	}

	if m.Authority != domains.SourceDomainsDoc {
		t.Fatalf("authority = %q, want domains_doc (the structured DOMAINS.md must be the top available rung)", m.Authority)
	}

	// The doc must declare the core product domains. We assert a stable
	// subset rather than an exact set so adding a domain doesn't break the
	// guard (exact convergence is asserted by the Phase 3 convergence test).
	want := []string{"graph", "conflicts", "signals", "apply", "analytics"}
	have := map[string]bool{}
	for _, n := range m.Names() {
		have[n] = true
	}
	for _, w := range want {
		if !have[w] {
			t.Fatalf("DOMAINS.md missing expected domain %q; derived set = %v", w, m.Names())
		}
	}

	// Every derived domain must carry at least one source path (the
	// contract requires it; Resolve trusts the extractor's validation).
	for _, d := range m.Domains {
		if len(d.Paths) == 0 {
			t.Fatalf("domain %q has no source paths", d.Name)
		}
		for _, path := range d.Paths {
			if !strings.HasPrefix(path, "scenarios/") && !strings.HasPrefix(path, "packages/") {
				t.Fatalf("domain %q path %q is not project-root-relative", d.Name, path)
			}
		}
	}

	// Shared substrate must be populated from the Non-Domains section.
	if len(m.SharedSubstrate) == 0 {
		t.Fatal("expected non-empty shared substrate derived from the Non-Domains section")
	}
}

// TestDogfood_OwnConvergenceHasNoDrift asserts cartographer's own surfaces
// (DOMAINS.md, api folders, cli groups, ui features) agree on the domain
// set with no WARN-severity convergence drift. Advisory INFO findings (e.g.
// a domain without a cli group) are tolerated; a WARN means a declared
// domain has no implementation, or a folder is undeclared — a real bug.
func TestDogfood_OwnConvergenceHasNoDrift(t *testing.T) {
	dir, err := filepath.Abs(scenarioRoot)
	if err != nil {
		t.Fatalf("abs scenario root: %v", err)
	}
	extractions, err := domains.RunLadder(context.Background(), dir, domains.DefaultExtractors())
	if err != nil {
		t.Fatalf("run ladder: %v", err)
	}
	m, err := domains.Resolve("architecture-cartographer", extractions, time.Time{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	for _, f := range domains.Convergence(m) {
		if f.Severity == domains.ConvergenceWarn {
			t.Errorf("unexpected WARN convergence drift: %s %q — %s", f.Kind, f.Domain, f.Message)
		}
	}
}
