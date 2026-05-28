package records

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// TestRecordKindMirrorsBacklogKind is an anti-drift gate. To avoid taking a
// transitive build dependency on the backlog package (which is intentionally
// large), the test reads backlog/types.go via the filesystem and extracts
// its Kind* const block. If backlog adds a new kind, this test fails until
// records adds the same value.
func TestRecordKindMirrorsBacklogKind(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	backlogTypes := filepath.Join(filepath.Dir(here), "..", "backlog", "types.go")
	data, err := os.ReadFile(backlogTypes)
	if err != nil {
		t.Fatalf("read backlog/types.go: %v", err)
	}
	re := regexp.MustCompile(`Kind(\w+)\s+BacklogKind\s*=\s*"(\w+)"`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	if len(matches) == 0 {
		t.Fatalf("no Kind* constants found in backlog/types.go")
	}
	got := map[string]string{}
	for _, m := range matches {
		got[m[1]] = m[2]
	}
	want := map[RecordKind]string{
		KindIdea:     "Idea",
		KindResearch: "Research",
		KindFix:      "Fix",
		KindExecute:  "Execute",
		KindChore:    "Chore",
	}
	for rk, ident := range want {
		v, ok := got[ident]
		if !ok {
			t.Errorf("backlog has no Kind%s; records expects it", ident)
			continue
		}
		if v != string(rk) {
			t.Errorf("kind %s drift: backlog=%q records=%q", ident, v, rk)
		}
		delete(got, ident)
	}
	for ident, v := range got {
		if !strings.HasPrefix(ident, "Idea") && !strings.HasPrefix(ident, "Research") &&
			!strings.HasPrefix(ident, "Fix") && !strings.HasPrefix(ident, "Execute") &&
			!strings.HasPrefix(ident, "Chore") {
			t.Errorf("backlog has Kind%s=%q with no records counterpart", ident, v)
		}
	}
}

func TestParseKindCaseInsensitive(t *testing.T) {
	for _, raw := range []string{"FIX", " fix ", "Fix"} {
		k, err := ParseKind(raw)
		if err != nil {
			t.Fatalf("ParseKind(%q): %v", raw, err)
		}
		if k != KindFix {
			t.Errorf("ParseKind(%q) = %q, want %q", raw, k, KindFix)
		}
	}
	if _, err := ParseKind("nope"); err == nil {
		t.Errorf("ParseKind(nope) accepted invalid value")
	}
}

func TestParseOutcomeCaseInsensitive(t *testing.T) {
	for _, raw := range []string{"SHIPPED", " shipped ", "Shipped"} {
		o, err := ParseOutcome(raw)
		if err != nil {
			t.Fatalf("ParseOutcome(%q): %v", raw, err)
		}
		if o != OutcomeShipped {
			t.Errorf("ParseOutcome(%q) = %q, want %q", raw, o, OutcomeShipped)
		}
	}
	if _, err := ParseOutcome("xxx"); err == nil {
		t.Errorf("ParseOutcome(xxx) accepted invalid value")
	}
}

func TestRecordHasNarrative(t *testing.T) {
	cases := []struct {
		name string
		r    Record
		want bool
	}{
		{"empty", Record{}, false},
		{"trigger only", Record{Trigger: "x"}, true},
		{"approach only", Record{Approach: "x"}, true},
		{"ruled-out only", Record{RuledOut: []string{"x"}}, true},
		{"whitespace trigger", Record{Trigger: "   "}, false},
	}
	for _, tc := range cases {
		if got := tc.r.hasNarrative(); got != tc.want {
			t.Errorf("%s: hasNarrative=%v, want %v", tc.name, got, tc.want)
		}
	}
}
