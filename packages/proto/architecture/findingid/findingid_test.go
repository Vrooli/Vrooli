package findingid

import (
	"strings"
	"testing"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func finding(scenario string, src architecturev1.FindingSource, code string, locs ...string) *architecturev1.ArchitectureFinding {
	return &architecturev1.ArchitectureFinding{
		Scenario:  scenario,
		Source:    src,
		Code:      code,
		Locations: locs,
	}
}

// TestComputeDeterministic: the same inputs always hash to the same ID.
func TestComputeDeterministic(t *testing.T) {
	f := finding("swarm-manager", architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE, "cycle/cross-domain", "api/internal/a", "api/internal/b")
	a := For(f)
	b := For(f)
	if a != b {
		t.Fatalf("non-deterministic: %s != %s", a, b)
	}
	if !strings.HasPrefix(a, Prefix) {
		t.Fatalf("missing prefix %q: %s", Prefix, a)
	}
	if len(a) != len(Prefix)+16 {
		t.Fatalf("want %d-char id, got %d: %s", len(Prefix)+16, len(a), a)
	}
}

// TestComputeOrderInsensitiveLocations: location order/casing/slashing
// must not change the ID (cosmetic perturbation → no false regression).
func TestComputeOrderInsensitiveLocations(t *testing.T) {
	base := For(finding("s", architecturev1.FindingSource_FINDING_SOURCE_CLI, "c", "api/x.go", "api/y.go"))
	perms := [][]string{
		{"api/y.go", "api/x.go"},                // reordered
		{"api\\y.go", "api\\x.go"},              // backslashes
		{"  api/x.go ", "api/y.go", "api/y.go"}, // whitespace + dup
		{"api/x.go", "", "api/y.go"},            // empty entry dropped
	}
	for i, p := range perms {
		got := For(finding("s", architecturev1.FindingSource_FINDING_SOURCE_CLI, "c", p...))
		if got != base {
			t.Errorf("perm %d: %v hashed to %s, want %s", i, p, got, base)
		}
	}
}

// TestExcludedFieldsDoNotChangeID: severity/message/suggestion/domains are
// not hash inputs.
func TestExcludedFieldsDoNotChangeID(t *testing.T) {
	a := finding("s", architecturev1.FindingSource_FINDING_SOURCE_DOCS, "missing_doc", "docs/x.md")
	b := finding("s", architecturev1.FindingSource_FINDING_SOURCE_DOCS, "missing_doc", "docs/x.md")
	b.Severity = architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER
	b.Message = "totally different message"
	b.Suggestion = "do something else"
	b.Domains = []string{"docs"}
	if For(a) != For(b) {
		t.Fatalf("excluded fields changed ID: %s != %s", For(a), For(b))
	}
}

// TestNoCollisions: a corpus of meaningfully-distinct findings must all
// produce distinct IDs. Each varied hash input (scenario, source, code,
// locations) must move the ID.
func TestNoCollisions(t *testing.T) {
	corpus := []*architecturev1.ArchitectureFinding{
		finding("s1", architecturev1.FindingSource_FINDING_SOURCE_CLI, "c", "a"),
		finding("s2", architecturev1.FindingSource_FINDING_SOURCE_CLI, "c", "a"),       // scenario differs
		finding("s1", architecturev1.FindingSource_FINDING_SOURCE_UI, "c", "a"),        // source differs
		finding("s1", architecturev1.FindingSource_FINDING_SOURCE_CLI, "d", "a"),       // code differs
		finding("s1", architecturev1.FindingSource_FINDING_SOURCE_CLI, "c", "b"),       // location differs
		finding("s1", architecturev1.FindingSource_FINDING_SOURCE_CLI, "c", "a", "b"),  // extra location
		finding("s1", architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE, "c", "a"), // source differs
		finding("s1", architecturev1.FindingSource_FINDING_SOURCE_STANDARDS, "c", "a"),
		finding("s1", architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE, "cycle/within-domain", "a"),
		finding("s1", architecturev1.FindingSource_FINDING_SOURCE_TIDINESS, "long_func", "a"),
	}
	seen := map[string]int{}
	for i, f := range corpus {
		id := For(f)
		if prev, ok := seen[id]; ok {
			t.Errorf("collision: corpus[%d] and corpus[%d] both → %s", prev, i, id)
		}
		seen[id] = i
	}
}

// TestStampWritesField: Stamp populates StableId in place.
func TestStampWritesField(t *testing.T) {
	f := finding("s", architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE, "cycle", "a")
	Stamp(f)
	if f.StableId != For(f) {
		t.Fatalf("Stamp wrote %s, want %s", f.StableId, For(f))
	}
}

// TestNilSafe: helpers tolerate nil.
func TestNilSafe(t *testing.T) {
	if For(nil) != "" {
		t.Errorf("For(nil) should be empty")
	}
	if Stamp(nil) != nil {
		t.Errorf("Stamp(nil) should be nil")
	}
}
