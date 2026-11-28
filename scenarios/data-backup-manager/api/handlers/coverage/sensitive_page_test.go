package coverage

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"data-backup-manager/internal/coverage"
	"data-backup-manager/internal/sources"

	"github.com/vrooli/api-core/operatordecision"
)

// liveShapedSensitive reproduces the sizes of the real review that prompted the
// redesign: one AI tool's home holds about 70% of 3.3 GiB, four more owners hold
// most of the rest, and dozens of tiny host settings hold almost nothing.
func liveShapedSensitive() []coverage.Suggestion {
	items := []coverage.Suggestion{
		{ID: "claude", Owner: "claude-code", OwnerKind: "resource", Name: "home", Locator: "/h/.claude", ApproxBytes: 2_478_745_323, Rationale: "Claude Code documents ~/.claude as the cross-project user scope; this resource retains transcripts."},
		{ID: "grok", Owner: "grok", OwnerKind: "resource", Name: "home", Locator: "/h/.grok", ApproxBytes: 382_570_194},
		{ID: "gemini", Owner: "antigravity", OwnerKind: "resource", Name: "home", Locator: "/h/.gemini", ApproxBytes: 343_637_557},
		{ID: "codex", Owner: "codex", OwnerKind: "resource", Name: "home", Locator: "/h/.codex", ApproxBytes: 89_864_562},
		{ID: "sandbox", Owner: "workspace-sandbox", OwnerKind: "scenario", Name: "archives", Locator: "/h/archives", ApproxBytes: 229_446_801},
		{ID: "events", Owner: "swarm-manager", OwnerKind: "scenario", Name: "events-db", Locator: "/h/events.db", SourceKind: sources.KindSQLite, ApproxBytes: 3_076_096, Sidecars: []string{"-wal", "-shm"}},
	}
	for _, name := range []string{"sysctl_config", "journald_config", "grub_config", "apparmor_profile"} {
		items = append(items, coverage.Suggestion{ID: "host-" + name, Owner: "host_hardening", OwnerKind: "safeguard", Name: name, Locator: "/etc/" + name, ApproxBytes: 400, Rationale: "The safeguard writes Linux kernel hardening settings."})
	}
	return items
}

func renderLive(t *testing.T, earlier []time.Time) (operatordecision.Facts, string) {
	t.Helper()
	items := liveShapedSensitive()
	facts := sensitiveReviewFacts(items, snapshotCandidates(items), "ask-1", "fp", "0", "No attention items.", time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC), earlier)
	if err := facts.Validate(); err != nil {
		t.Fatalf("facts do not validate: %v", err)
	}
	if problems := operatordecision.CoverProblems(facts); len(problems) > 0 {
		t.Fatalf("cover problems: %v", problems)
	}
	return facts, facts.Explainer
}

func TestSensitivePageShowsWhereTheSizeIsAsAProportionNotATable(t *testing.T) {
	_, page := renderLive(t, nil)
	if !strings.Contains(page, `class="hub-proportion"`) {
		t.Fatal("the page has no proportion bar")
	}
	if strings.Contains(page, "<table") {
		t.Error("the items are back in a table")
	}
	legend := regexp.MustCompile(`Claude Code<small>[^<]*</small></span><span class="hub-proportion__value">[^<]*<small>(\d+)%`).FindStringSubmatch(page)
	if legend == nil || legend[1] != "70" {
		t.Errorf("legend does not say Claude Code is about 70%% of the size: %v", legend)
	}
	if !strings.Contains(page, "history, memory, settings and sign-in") {
		t.Error("the largest share is not explained in plain words")
	}
	if !regexp.MustCompile(`\d+ other places`).MatchString(page) {
		t.Error("the long tail is not folded into one share")
	}
}

func TestSensitivePageGroupsItemsInPlainWordsWithoutManifestProseOrLocators(t *testing.T) {
	_, page := renderLive(t, nil)
	for _, want := range []string{
		`class="hub-groups" data-hub-groups="groups"`, `aria-pressed="true"`, `aria-pressed="false"`, "Show the 4 items", "Show the 2 items", "Claude Code history, memory, settings and sign-in · 2.3GiB",
		"Swarm Manager&#39;s event history and approvals, with its live -wal and -shm files", "Setup&#39;s sysctl config (host hardening)",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	for _, leak := range []string{"/h/.claude", "/etc/", "documents ~/.claude", "cross-project user scope", "safeguard writes"} {
		if strings.Contains(page, leak) {
			t.Errorf("page leaks locator or manifest prose %q", leak)
		}
	}
}

func TestSensitivePageLeadsWithWhatWouldBeLostBeforeAnyCount(t *testing.T) {
	_, page := renderLive(t, nil)
	lead := strings.Index(page, "have no backup. If this computer")
	bar := strings.Index(page, "hub-proportion")
	choices := strings.Index(page, "Your choices")
	if lead < 0 || bar < 0 || choices < 0 || !(lead < bar && bar < choices) {
		t.Fatalf("order lead=%d bar=%d choices=%d; want what is lost, then where the size is, then the choices", lead, bar, choices)
	}
	if !strings.Contains(page, "Your Claude Code, Grok, Antigravity and Codex history") {
		t.Error("the lead does not name the AI history")
	}
}

func TestSensitivePageNamesTheRepeatOnlyWhenThereWasOne(t *testing.T) {
	_, first := renderLive(t, nil)
	if strings.Contains(first, "hub-recurrence") {
		t.Error("a first question claims to be a repeat")
	}
	facts, repeat := renderLive(t, []time.Time{time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC), time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC)})
	for _, want := range []string{"hub-recurrence", "asked this 3 times since 13 Nov", "ends the question"} {
		if !strings.Contains(repeat, want) {
			t.Errorf("repeat page lacks %q", want)
		}
	}
	if !strings.Contains(facts.BigPicture, "third time") {
		t.Errorf("the cover does not say it is the third time: %s", facts.BigPicture)
	}
}

func TestSensitivePageOptionCardsCarryNumbersNotTheLabelAgain(t *testing.T) {
	facts, page := renderLive(t, nil)
	for _, option := range facts.Options {
		if !strings.Contains(page, `data-od-answer="`+option.Key+`"`) {
			t.Errorf("no button answers %q", option.Key)
		}
	}
	for _, want := range []string{"All 10 are added today (3.3GiB)", "weekly note", "recorded with your name", "Adds only the groups switched on"} {
		if !strings.Contains(page, want) {
			t.Errorf("option effects lack %q", want)
		}
	}
	if !strings.Contains(page, "data-backup-manager coverage revoke-sensitive-policy") || !strings.Contains(page, "data-hub-copy") {
		t.Error("the command that turns the rule off is not shown with a copy button")
	}
	if strings.Contains(page, "`") {
		t.Error("a Markdown backtick reached the page, where it would show literally")
	}
}

func TestSensitivePageNeverUsesLiteralColoursThatBreakInDarkMode(t *testing.T) {
	_, page := renderLive(t, nil)
	if regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b|rgb\(`).MatchString(strings.ReplaceAll(page, "&#39;", "")) {
		t.Error("the page sets its own literal colours; use the Hub's tokens so light and dark both read")
	}
}

func TestSensitiveReviewPageFitsTheLimitsWithRealisticVolume(t *testing.T) {
	var items []coverage.Suggestion
	for i := 0; i < 120; i++ {
		items = append(items, coverage.Suggestion{ID: "id" + string(rune('a'+i%26)) + string(rune('a'+i/26)), Owner: "owner_" + string(rune('a'+i%26)), OwnerKind: "safeguard", Name: "item", Locator: "/x", ApproxBytes: 100})
	}
	facts := sensitiveReviewFacts(items, snapshotCandidates(items), "a", "f", "0", "", time.Now(), nil)
	if err := facts.Validate(); err != nil {
		t.Fatalf("a 120-item review does not validate: %v", err)
	}
	_ = context.Background
}

func TestSensitiveGroupsFormQuestionMatchesThePageSwitchesAndStartsHostSettingsOff(t *testing.T) {
	facts, page := renderLive(t, nil)
	if len(facts.Questions) != 1 {
		t.Fatalf("questions = %+v", facts.Questions)
	}
	q := facts.Questions[0]
	if q.ID != "groups" || q.Type != operatordecision.QuestionMultiChoice {
		t.Fatalf("question = %+v", q)
	}
	for _, c := range q.Choices {
		if !strings.Contains(page, `data-key="`+c.Key+`"`) {
			t.Errorf("no page group for choice %q", c.Key)
		}
	}
	if strings.Join(q.Recommended, ",") != "resource,scenario" {
		t.Errorf("starts on = %v, want the AI and app groups with host settings off", q.Recommended)
	}
	hasGroups := false
	for _, o := range facts.Options {
		hasGroups = hasGroups || o.Key == answerGroups
	}
	if !hasGroups {
		t.Error("no option applies the group selection")
	}
}
