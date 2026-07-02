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

// TestParseKindAliases asserts the curated alias map resolves common improvised
// kind names to the right canonical kind, case-insensitively and trimmed.
func TestParseKindAliases(t *testing.T) {
	cases := map[string]RecordKind{
		"improvement":          KindExecute,
		"scenario-improvement": KindExecute,
		"implementation":       KindExecute,
		"feature":              KindExecute,
		"feat":                 KindExecute,
		"refactor":             KindExecute,
		"build":                KindExecute,
		"bug":                  KindFix,
		"bugfix":               KindFix,
		"bug-fix":              KindFix,
		"fixup":                KindFix,
		"hotfix":               KindFix,
		"investigation":        KindResearch,
		"spike":                KindResearch,
		"explore":              KindResearch,
		"task":                 KindChore,
		"maintenance":          KindChore,
		"cleanup":              KindChore,
		// case/whitespace tolerance reuses the same lowercase+trim path:
		"  Feature ": KindExecute,
		"BUGFIX":     KindFix,
	}
	for raw, want := range cases {
		got, err := ParseKind(raw)
		if err != nil {
			t.Errorf("ParseKind(%q): unexpected error %v", raw, err)
			continue
		}
		if got != want {
			t.Errorf("ParseKind(%q) = %q, want %q", raw, got, want)
		}
	}
}

// TestParseKindSuggestion asserts the error is self-correcting: a near miss
// suggests the nearest canonical kind, while garbage gets no misleading hint
// (but is still rejected).
func TestParseKindSuggestion(t *testing.T) {
	// "improvment" is one edit from the "improvement" alias → execute.
	_, err := ParseKind("improvment")
	if err == nil {
		t.Fatalf("ParseKind(improvment) accepted invalid value")
	}
	if !strings.Contains(err.Error(), `did you mean "execute"?`) {
		t.Errorf("ParseKind(improvment) error %q lacks execute suggestion", err)
	}
	// The valid-kinds list is always present.
	if !strings.Contains(err.Error(), "idea, research, fix, execute, chore") {
		t.Errorf("ParseKind(improvment) error %q lacks valid-kinds list", err)
	}
	// Garbage is rejected without a misleading suggestion.
	garbageErr := func() error { _, e := ParseKind("xyzzy-not-a-kind"); return e }()
	if garbageErr == nil {
		t.Fatalf("ParseKind(garbage) accepted invalid value")
	}
	if strings.Contains(garbageErr.Error(), "did you mean") {
		t.Errorf("ParseKind(garbage) error %q has a misleading suggestion", garbageErr)
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

func TestParseOutcomeAliases(t *testing.T) {
	cases := map[string]Outcome{
		"success":     OutcomeShipped,
		"done":        OutcomeShipped,
		"green":       OutcomeShipped,
		"completed":   OutcomeShipped,
		"fixed":       OutcomeShipped,
		"resolved":    OutcomeShipped,
		"implemented": OutcomeShipped,
		"in_progress": OutcomePartial,
		"wip":         OutcomePartial,
		"cancelled":   OutcomeAbandoned,
		"dupe":        OutcomeDuplicate,
	}
	for raw, want := range cases {
		got, err := ParseOutcome(raw)
		if err != nil {
			t.Fatalf("ParseOutcome(%q): %v", raw, err)
		}
		if got != want {
			t.Errorf("ParseOutcome(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestParseOutcomeProseRejection(t *testing.T) {
	prose := "All validation green with ZERO regressions: baseline diff = preexisting; api go test green"
	_, err := ParseOutcome(prose)
	if err == nil {
		t.Fatal("ParseOutcome accepted prose narrative")
	}
	msg := err.Error()
	for _, want := range []string{"looks like a narrative", "--evidence", "--approach", "shipped"} {
		if !strings.Contains(msg, want) {
			t.Errorf("prose error %q missing %q", msg, want)
		}
	}
	// Short multi-word values are also prose-shaped, not typos.
	if _, err := ParseOutcome("all tests pass"); err == nil {
		t.Error("ParseOutcome accepted multi-word value")
	} else if !strings.Contains(err.Error(), "looks like a narrative") {
		t.Errorf("multi-word error should use prose guidance, got %q", err.Error())
	}
}

func TestParseOutcomeSuggestion(t *testing.T) {
	_, err := ParseOutcome("shiped")
	if err == nil {
		t.Fatal("ParseOutcome accepted misspelling")
	}
	if !strings.Contains(err.Error(), `did you mean "shipped"?`) {
		t.Errorf("expected did-you-mean hint, got %q", err.Error())
	}
	// Garbage gets the enum but no bogus suggestion.
	_, err = ParseOutcome("zzzzzz")
	if err == nil {
		t.Fatal("ParseOutcome accepted garbage")
	}
	if strings.Contains(err.Error(), "did you mean") {
		t.Errorf("garbage should not get a suggestion, got %q", err.Error())
	}
}

func TestEmbeddingTextIncludesEvidence(t *testing.T) {
	r := Record{Trigger: "t", Approach: "a", Evidence: "all suites green"}
	if !strings.Contains(r.EmbeddingText(), "all suites green") {
		t.Errorf("EmbeddingText missing evidence: %q", r.EmbeddingText())
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
