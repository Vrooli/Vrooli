package main

import (
	"reflect"
	"testing"
)

func TestFirstNonEmpty(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{"", "  ", "first", "second"}, "first"},
		{[]string{"  trimmed  "}, "trimmed"},
		{[]string{"", ""}, ""},
		{nil, ""},
	}
	for _, c := range cases {
		if got := firstNonEmpty(c.in...); got != c.want {
			t.Errorf("firstNonEmpty(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFirstNonEmptyString(t *testing.T) {
	if got := firstNonEmptyString("", " x ", "y"); got != "x" {
		t.Errorf("firstNonEmptyString = %q, want %q", got, "x")
	}
	if got := firstNonEmptyString("", ""); got != "" {
		t.Errorf("firstNonEmptyString empty = %q, want empty", got)
	}
}

func TestSummarizePendingQuestion(t *testing.T) {
	workshopWithOptions := PendingQuestion{
		Source:  "workshop",
		Topic:   "Pick storage",
		Options: []WorkshopOption{{Label: "sqlite"}, {Label: " "}, {Label: "postgres"}},
	}
	if got := summarizePendingQuestion(workshopWithOptions); got != "Pick storage (options: sqlite, postgres)" {
		t.Errorf("workshop+options = %q", got)
	}

	workshopNoOptions := PendingQuestion{Source: "workshop", Text: "Decide approach"}
	if got := summarizePendingQuestion(workshopNoOptions); got != "Decide approach" {
		t.Errorf("workshop fallback to Text = %q", got)
	}

	workshopFallback := PendingQuestion{Source: "workshop"}
	if got := summarizePendingQuestion(workshopFallback); got != "workshop decision" {
		t.Errorf("workshop default = %q", got)
	}

	reviewWithType := PendingQuestion{Source: "review", Title: "Fix bug", ReviewType: "quality"}
	if got := summarizePendingQuestion(reviewWithType); got != "Fix bug (quality)" {
		t.Errorf("review+type = %q", got)
	}

	reviewNoType := PendingQuestion{Source: "review", Description: "desc only"}
	if got := summarizePendingQuestion(reviewNoType); got != "desc only" {
		t.Errorf("review desc fallback = %q", got)
	}
}

func TestParseCommaSeparated(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"a, b ,c", []string{"a", "b", "c"}},
		{"", []string{}},
		{" , , ", []string{}},
		{"single", []string{"single"}},
	}
	for _, c := range cases {
		got := parseCommaSeparated(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseCommaSeparated(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestIsValidInitiativeStatus(t *testing.T) {
	valid := []string{"active", "completed"}
	for _, s := range valid {
		if !isValidInitiativeStatus(s) {
			t.Errorf("isValidInitiativeStatus(%q) = false, want true", s)
		}
	}
	invalid := []string{"", "Active", "paused", "done"}
	for _, s := range invalid {
		if isValidInitiativeStatus(s) {
			t.Errorf("isValidInitiativeStatus(%q) = true, want false", s)
		}
	}
}

func TestBoolCount(t *testing.T) {
	if got := boolCount(true, false, true, true); got != 3 {
		t.Errorf("boolCount = %d, want 3", got)
	}
	if got := boolCount(); got != 0 {
		t.Errorf("boolCount() = %d, want 0", got)
	}
	if got := boolCount(false, false); got != 0 {
		t.Errorf("boolCount(false,false) = %d, want 0", got)
	}
}

func TestPromptCatalogTarget(t *testing.T) {
	if got := promptCatalogTarget(PromptCatalogEntry{SkillID: "skill-x", Builder: "b"}); got != "skill-x" {
		t.Errorf("skill priority = %q", got)
	}
	if got := promptCatalogTarget(PromptCatalogEntry{Builder: "builder-y"}); got != "builder-y" {
		t.Errorf("builder fallback = %q", got)
	}
	if got := promptCatalogTarget(PromptCatalogEntry{}); got != "(unknown)" {
		t.Errorf("unknown fallback = %q", got)
	}
}

func TestFirstOr(t *testing.T) {
	if got := firstOr([]string{" a ", "b"}, "fb"); got != "a" {
		t.Errorf("firstOr = %q, want a", got)
	}
	if got := firstOr(nil, "fb"); got != "fb" {
		t.Errorf("firstOr empty = %q, want fb", got)
	}
	if got := firstOr([]string{"  "}, "fb"); got != "fb" {
		t.Errorf("firstOr blank-first = %q, want fb", got)
	}
}

func TestEmptyDash(t *testing.T) {
	if got := emptyDash("  "); got != "—" {
		t.Errorf("emptyDash blank = %q, want dash", got)
	}
	if got := emptyDash("value"); got != "value" {
		t.Errorf("emptyDash value = %q", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("  short  ", 10); got != "short" {
		t.Errorf("truncate short = %q", got)
	}
	if got := truncate("abcdefghij", 5); got != "abcde..." {
		t.Errorf("truncate long = %q", got)
	}
	if got := truncate("exact", 5); got != "exact" {
		t.Errorf("truncate exact = %q", got)
	}
}

func TestBoolLabel(t *testing.T) {
	if boolLabel(true) != "yes" {
		t.Error("boolLabel(true) should be yes")
	}
	if boolLabel(false) != "no" {
		t.Error("boolLabel(false) should be no")
	}
}

func TestDefaultString(t *testing.T) {
	if got := defaultString("  ", "fb"); got != "fb" {
		t.Errorf("defaultString blank = %q, want fb", got)
	}
	if got := defaultString("x", "fb"); got != "x" {
		t.Errorf("defaultString value = %q", got)
	}
}

func TestModeRoundActionTitle(t *testing.T) {
	cases := map[string]string{
		"refresh": "Refreshed Round",
		"cancel":  "Canceled Round",
		"other":   "Updated Round",
		"":        "Updated Round",
	}
	for in, want := range cases {
		if got := modeRoundActionTitle(in); got != want {
			t.Errorf("modeRoundActionTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSubmissionPreview(t *testing.T) {
	if got := submissionPreview("   "); got != "" {
		t.Errorf("blank = %q", got)
	}
	if got := submissionPreview("hello"); got != "hello" {
		t.Errorf("short = %q", got)
	}
	long := make([]byte, 150)
	for i := range long {
		long[i] = 'a'
	}
	got := submissionPreview(string(long))
	if len(got) != 103 || got[100:] != "..." {
		t.Errorf("long preview len=%d suffix=%q", len(got), got[len(got)-3:])
	}
}

func TestFilterFixes(t *testing.T) {
	fixes := []ScenarioFix{
		{Name: "fix-a", Title: "Improve startup"},
		{Name: "fix-b", Title: "Fix CRASH"},
		{Name: "startup-c", Title: "Other"},
	}
	// empty search returns a copy of all.
	all := filterFixes(fixes, "  ")
	if len(all) != 3 {
		t.Fatalf("empty search len = %d, want 3", len(all))
	}
	if &all[0] == &fixes[0] {
		t.Error("empty search should return a copy, not the same backing array")
	}
	// case-insensitive title match.
	if got := filterFixes(fixes, "CRASH"); len(got) != 1 || got[0].Name != "fix-b" {
		t.Errorf("title match = %+v", got)
	}
	// match on name too.
	if got := filterFixes(fixes, "startup"); len(got) != 2 {
		t.Errorf("name+title match len = %d, want 2", len(got))
	}
	// no match.
	if got := filterFixes(fixes, "zzz"); len(got) != 0 {
		t.Errorf("no match len = %d, want 0", len(got))
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]int{"c": 3, "a": 1, "b": 2}
	got := sortedKeys(m)
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sortedKeys = %v, want %v", got, want)
	}
	if got := sortedKeys(map[string]int{}); len(got) != 0 {
		t.Errorf("empty sortedKeys len = %d", len(got))
	}
}

func TestFormatMapCounts(t *testing.T) {
	m := map[string]int{"open": 2, "closed": 5}
	if got := formatMapCounts(m); got != "closed(5) open(2)" {
		t.Errorf("formatMapCounts = %q", got)
	}
	if got := formatMapCounts(map[string]int{}); got != "" {
		t.Errorf("empty formatMapCounts = %q", got)
	}
}

func TestSortedStringMapKeys(t *testing.T) {
	m := map[string]string{"z": "1", "a": "2", "m": "3"}
	got := sortedStringMapKeys(m)
	want := []string{"a", "m", "z"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sortedStringMapKeys = %v, want %v", got, want)
	}
}

func TestRequireFlag(t *testing.T) {
	if err := requireFlag("name", " value "); err != nil {
		t.Errorf("non-empty value should pass: %v", err)
	}
	err := requireFlag("name", "  ")
	if err == nil || err.Error() != "--name is required" {
		t.Errorf("blank value err = %v, want --name is required", err)
	}
}

func TestRequireFlags(t *testing.T) {
	if err := requireFlags("a", "1", "b", "2"); err != nil {
		t.Errorf("all present should pass: %v", err)
	}
	if err := requireFlags("a", "1", "b", ""); err == nil {
		t.Error("missing second flag should error")
	}
	if err := requireFlags("a"); err == nil {
		t.Error("uneven pairs should error")
	}
}

func TestCliCommand(t *testing.T) {
	got := cliCommand("backlog", "  ", "list")
	want := appName + " backlog list"
	if got != want {
		t.Errorf("cliCommand = %q, want %q", got, want)
	}
	if got := cliCommand(); got != appName {
		t.Errorf("cliCommand() = %q, want %q", got, appName)
	}
}

func TestStringSlice(t *testing.T) {
	var s stringSlice
	if err := s.Set("a"); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("b"); err != nil {
		t.Fatal(err)
	}
	if s.String() != "a, b" {
		t.Errorf("String() = %q, want %q", s.String(), "a, b")
	}
	if len(s) != 2 {
		t.Errorf("len = %d, want 2", len(s))
	}
}
