package main

import (
	"errors"
	"flag"
	"io"
	"reflect"
	"strings"
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
	reviewWithType := PendingQuestion{Source: "review", Title: "Fix bug", ReviewType: "quality"}
	if got := summarizePendingQuestion(reviewWithType); got != "Fix bug (quality)" {
		t.Errorf("review+type = %q", got)
	}

	reviewNoType := PendingQuestion{Source: "review", Description: "desc only"}
	if got := summarizePendingQuestion(reviewNoType); got != "desc only" {
		t.Errorf("review desc fallback = %q", got)
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

func TestShellQuote(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "''"},
		{"plain-word_1.go", "plain-word_1.go"},
		{"two words", "'two words'"},
		{"it's done", `'it'\''s done'`},
		{"a $(cmd) `tick`", "'a $(cmd) `tick`'"},
	}
	for _, c := range cases {
		if got := shellQuote(c.in); got != c.want {
			t.Errorf("shellQuote(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestOutcomeLooksLikeProse(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"shipped", false},
		{"done", false},
		{"", false},
		{"all tests pass", true},
		{"All validation green with ZERO regressions across api and cli suites", true},
	}
	for _, c := range cases {
		if got := outcomeLooksLikeProse(c.in); got != c.want {
			t.Errorf("outcomeLooksLikeProse(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestRecordsCreateSuggestedCommand(t *testing.T) {
	in := recordsCreateInput{
		scenario: "web-console",
		trigger:  "mic stayed live",
		evidence: "ui vitest green; restart healthy",
		files:    stringSlice{"ui/src/a.ts", "ui/src/b.ts"},
	}
	got := in.suggestedCommand()
	for _, want := range []string{
		"swarm-manager records create",
		"--kind <idea|research|fix|execute|chore>", // missing required → placeholder
		"--scenario web-console",
		"--trigger 'mic stayed live'",
		"--evidence 'ui vitest green; restart healthy'",
		"--files ui/src/a.ts --files ui/src/b.ts",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("suggestedCommand missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "--outcome") {
		t.Errorf("default outcome should be omitted:\n%s", got)
	}
}

func TestRecordsFlagError(t *testing.T) {
	fs := flag.NewFlagSet("records create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.String("trigger", "", "")
	fs.String("scenario", "", "")

	cases := []struct {
		args     []string
		wantHint string
	}{
		{[]string{"--text", "x"}, "captures create"},
		{[]string{"--title", "x"}, "did you mean --trigger"},
		{[]string{"--description", "x"}, "did you mean --approach"},
		{[]string{"--scenari", "x"}, "did you mean --scenario?"},
	}
	for _, c := range cases {
		err := fs.Parse(c.args)
		if err == nil {
			t.Fatalf("parse %v should fail", c.args)
		}
		got := recordsFlagError(fs, err).Error()
		if !strings.Contains(got, c.wantHint) {
			t.Errorf("recordsFlagError(%v) = %q, want hint %q", c.args, got, c.wantHint)
		}
	}

	// Unrelated errors pass through untouched.
	plain := errors.New("boom")
	if recordsFlagError(fs, plain) != plain {
		t.Error("non-flag error should pass through")
	}
}
