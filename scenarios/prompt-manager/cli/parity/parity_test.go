package parity

import (
	"sort"
	"strings"
	"testing"
)

// TestParityNoUnmappedAPIRoutes is the load-bearing CI guard: it fails when
// a v1.HandleFunc registration in the API source has no entry in
// coverage.json. Adding a new endpoint without recording its CLI exposure
// (or marking it intentionally-absent / audit-pending with a reason) breaks
// the build.
func TestParityNoUnmappedAPIRoutes(t *testing.T) {
	mainPath, err := APIMainPath()
	if err != nil {
		t.Fatalf("APIMainPath: %v", err)
	}
	routes, err := ExtractAPIRoutes(mainPath)
	if err != nil {
		t.Fatalf("ExtractAPIRoutes(%s): %v", mainPath, err)
	}
	if len(routes) == 0 {
		t.Fatal("extracted 0 routes — regex broken or main.go moved?")
	}

	coverage, err := LoadCoverage()
	if err != nil {
		t.Fatalf("LoadCoverage: %v", err)
	}

	missing := FindUnmapped(routes, coverage)
	if len(missing) > 0 {
		var b strings.Builder
		b.WriteString("the following API routes have no entry in coverage.json:\n")
		for _, r := range missing {
			b.WriteString("  - ")
			b.WriteString(FormatRoute(r))
			b.WriteString("\n  (key: ")
			b.WriteString(r.Key())
			b.WriteString(")\n")
		}
		b.WriteString("\nAdd an entry to scenarios/prompt-manager/cli/parity/coverage.json with status \"covered\", \"intentionally-absent\", or \"audit-pending\".")
		t.Fatal(b.String())
	}
}

// TestParityNoStaleCoverageEntries fails if coverage.json refers to routes
// that no longer exist in the API source — keeping the audit doc honest.
func TestParityNoStaleCoverageEntries(t *testing.T) {
	mainPath, err := APIMainPath()
	if err != nil {
		t.Fatalf("APIMainPath: %v", err)
	}
	routes, err := ExtractAPIRoutes(mainPath)
	if err != nil {
		t.Fatalf("ExtractAPIRoutes: %v", err)
	}
	coverage, err := LoadCoverage()
	if err != nil {
		t.Fatalf("LoadCoverage: %v", err)
	}

	stale := FindStale(routes, coverage)
	if len(stale) > 0 {
		t.Fatalf("coverage.json references routes that no longer exist in the API:\n  - %s\nRemove these entries.", strings.Join(stale, "\n  - "))
	}
}

// TestParityRouteKeyShape sanity-checks the regex by ensuring every route
// has a non-empty path and at least one HTTP method. Catches a class of
// silent regex regressions.
func TestParityRouteKeyShape(t *testing.T) {
	mainPath, err := APIMainPath()
	if err != nil {
		t.Fatalf("APIMainPath: %v", err)
	}
	routes, err := ExtractAPIRoutes(mainPath)
	if err != nil {
		t.Fatalf("ExtractAPIRoutes: %v", err)
	}
	for _, r := range routes {
		if r.Path == "" {
			t.Errorf("line %d: empty path for handler %s", r.Line, r.Handler)
		}
		if len(r.Methods) == 0 {
			t.Errorf("line %d: empty methods for path %s", r.Line, r.Path)
		}
		for _, m := range r.Methods {
			if m != strings.ToUpper(m) {
				t.Errorf("line %d: method %q not uppercase", r.Line, m)
			}
		}
	}
}

// TestParityProgressMetric reports the audit-pending count as a non-failing
// baseline so future PRs that drain the queue can be appreciated. It does
// not enforce a target — the goal is visibility, not pressure.
func TestParityProgressMetric(t *testing.T) {
	coverage, err := LoadCoverage()
	if err != nil {
		t.Fatalf("LoadCoverage: %v", err)
	}
	pending := []string{}
	covered := 0
	absent := 0
	for k, v := range coverage {
		switch v.Status {
		case StatusAuditPending:
			pending = append(pending, k)
		case StatusCovered:
			covered++
		case StatusIntentionallyAbsent:
			absent++
		}
	}
	sort.Strings(pending)
	t.Logf("coverage progress: %d covered, %d intentionally-absent, %d audit-pending (total %d)",
		covered, absent, len(pending), covered+absent+len(pending))
	if len(pending) > 0 && testing.Verbose() {
		for _, k := range pending {
			t.Logf("  pending: %s", k)
		}
	}
}
