package operatordecision

import (
	"reflect"
	"strings"
	"testing"
)

func TestInternalLabelsFlagsRequesterLabelsAndSparesOrdinaryText(t *testing.T) {
	for name, tc := range map[string]struct {
		text string
		want []string
	}{
		"letter and hyphen digits":         {"Should we pick S-14 now?", []string{"S-14"}},
		"letter and digits":                {"Do O7 first or later?", []string{"O7"}},
		"long segmented id":                {"Ship F49-MOR-CHECKOUT this week?", []string{"F49-MOR-CHECKOUT"}},
		"decision id":                      {"Close BAS-D-065 or keep it open?", []string{"BAS-D-065"}},
		"prefixed number":                  {"Is BAS-065 still wanted?", []string{"BAS-065"}},
		"uuid":                             {"Retry 3f2a9c1b-0d4e-4f6a-8b7c-1e2d3c4b5a69?", []string{"3f2a9c1b-0d4e-4f6a-8b7c-1e2d3c4b5a69"}},
		"several in order, repeats once":   {"S-14 then O7, then S-14 again", []string{"S-14", "O7"}},
		"id at the start and end":          {"O7", []string{"O7"}},
		"inside brackets and quotes":       {`Skip "S-14" (O7)?`, []string{"S-14", "O7"}},
		"nested label reported as longest": {"Work on F49-MOR-CHECKOUT", []string{"F49-MOR-CHECKOUT"}},
		"dated slugs":                      {"Apply um-20261109-57fc87 from research-run-20261114?", []string{"um-20261109-57fc87", "research-run-20261114"}},
		"hash":                             {"Ship commit 57fc87a1b2c3d4e5 now?", []string{"57fc87a1b2c3d4e5"}},

		"plain words":            {"Should the backup run tonight or wait until the drive is back?", nil},
		"year":                   {"Renew the certificate before 2026?", nil},
		"date":                   {"Keep logs until 2026-11-13?", nil},
		"time of day":            {"Run at 10:30 or 22:00?", nil},
		"money":                  {"Spend $50 or $500 on the upgrade?", nil},
		"lowercase version":      {"Upgrade to v2 or stay on v1.4?", nil},
		"capital version":        {"Upgrade to V2.1 or stay?", nil},
		"quarter":                {"Plan for Q3 or Q4?", nil},
		"half year":              {"Ship in H1 or H2?", nil},
		"apple chip":             {"Should the build target M1 or M3?", nil},
		"latency percentile":     {"Is a P95 of 300 ms acceptable?", nil},
		"paper size":             {"Print on A4?", nil},
		"storage product":        {"Move the archive to S3?", nil},
		"encoding standard":      {"Use UTF-8 and SHA-256 for the export?", nil},
		"model name":             {"Switch to GPT-4 for summaries?", nil},
		"cve":                    {"Patch CVE-2026-1234 now?", nil},
		"gpu names":              {"Rent an A100 or an H100?", nil},
		"mixed letters digits":   {"Is the B2B plan or the E2E suite first?", nil},
		"all caps words":         {"Is the DO-NOT-DISTURB mode OK?", nil},
		"iso week":               {"Deploy in 2026-W45?", nil},
		"hyphenated lower":       {"Use the follow-up-2 plan?", nil},
		"percent and ranges":     {"Raise the limit 10-20% or 5-10?", nil},
		"acronym with plural":    {"Review the APIs and CLIs?", nil},
		"sentence ending in a-b": {"Pick option a-1 or b-2?", nil},
		"long number":            {"Call 441555012345 or wait?", nil},
		"long hex-letter word":   {"Is the deadbeefcafe banner fine?", nil},
		"email and url":          {"Mail run-20261114@example.com or see https://example.com/57fc87a1b2c3d4e5", nil},
		"snake-case folder":      {"Delete node_modules and build_cache?", nil},
		"file names are paths":   {"Is /tmp/run-20261114.txt still needed?", nil},
	} {
		t.Run(name, func(t *testing.T) {
			if got := InternalLabels(tc.text); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("InternalLabels(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}

func TestLeadingTag(t *testing.T) {
	for text, want := range map[string]string{
		"[BAS] Close the goal?": "[BAS]",
		"  [S-14] retry?":       "[S-14]",
		"Close the goal [now]?": "", // a bracket inside the sentence is not a prefix tag
		"Close the goal?":       "",
	} {
		if got := LeadingTag(text); got != want {
			t.Errorf("LeadingTag(%q) = %q, want %q", text, got, want)
		}
	}
}

func TestPathsFindFilePathsButNotUrlsOrRoutes(t *testing.T) {
	for name, tc := range map[string]struct {
		text string
		want []string
	}{
		"absolute under a host root": {"See /home/matthew/Vrooli/docs/plan.md for details.", []string{"/home/matthew/Vrooli/docs/plan.md"}},
		"tmp file":                   {"Output is in /tmp/run-42/out.txt.", []string{"/tmp/run-42/out.txt"}},
		"in parentheses":             {"the plan (/var/lib/app/state.json)", []string{"/var/lib/app/state.json"}},
		"in backticks":               {"edit `/etc/hosts` first", []string{"/etc/hosts"}},
		"home relative":              {"open ~/notes/todo.md", []string{"~/notes/todo.md"}},
		"dot relative":               {"run ./scripts/fix.sh and ../out/log.txt", []string{"./scripts/fix.sh", "../out/log.txt"}},
		"windows":                    {`copy C:\Users\me\file.txt over`, []string{`C:\Users\me\file.txt`}},
		"other root with extension":  {"Read /project/docs/plan.md carefully", []string{"/project/docs/plan.md"}},
		"two in order":               {"/home/a/b.txt and /home/a/c.txt", []string{"/home/a/b.txt", "/home/a/c.txt"}},
		"repository file":            {"It edits api/main.go and .vrooli/work/plan.md.", []string{"api/main.go", ".vrooli/work/plan.md"}},

		"https url":             {"See https://example.com/docs/plan.md for details", nil},
		"http url with path":    {"Visit http://example.com/a/b/c.html", nil},
		"host with a path":      {"Visit example.com/plan.html", nil},
		"api route":             {"Call /api/v1/asks to list them", nil},
		"single segment route":  {"The /health page is green", nil},
		"fraction and ratios":   {"About 3/4 or 24/7, and/or both", nil},
		"slashes in a date":     {"On 2026/11/13 we decide", nil},
		"word with slash":       {"Pick read/write access", nil},
		"words with slashes":    {"Allow read/write/execute?", nil},
		"protocol relative":     {"load //cdn.example.com/lib.js", nil},
		"sentence period only":  {"Done. Now what?", nil},
		"extension without dir": {"Edit plan.md and notes.txt", nil},
	} {
		t.Run(name, func(t *testing.T) {
			if got := findPaths(tc.text, true); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("findPaths(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
	if got := LocalPaths("It edits api/main.go and /tmp/out.txt"); !reflect.DeepEqual(got, []string{"/tmp/out.txt"}) {
		t.Fatalf("a repository-relative file is not a file on this host to attach: %q", got)
	}
}

func coldFacts() Facts {
	return Facts{
		Subject:              "Skip tonight's backup?",
		Question:             "Should the nightly backup skip the Elements drive until it is plugged back in?",
		BigPicture:           "The drive has been unplugged since Tuesday, so tonight's backup would fail.",
		Options:              []Option{{Key: "skip", Label: "Skip it"}, {Key: "run", Label: "Run anyway"}},
		Recommended:          "skip",
		RecommendationReason: "Skipping avoids a failed run and a noisy alert.",
		TechnicalDetails:     "Destination 3f2a9c1b-0d4e-4f6a-8b7c-1e2d3c4b5a69 at /mnt/elements (internal names belong here).",
	}
}

func TestCheckCoverReportsEachRuleOnItsField(t *testing.T) {
	codes := func(c Facts) []string {
		var got []string
		for _, f := range CheckCover(c) {
			got = append(got, f.Code+"@"+f.Field)
		}
		return got
	}
	clean := coldFacts()
	if got := codes(clean); got != nil {
		t.Fatalf("a cold-readable cover was flagged: %v", got)
	}
	c := clean
	c.Question = "[BAS] Close the goal?"
	c.BigPicture = "This moves O5 through ~/notes/todo.md."
	c.RecommendationReason = ""
	c.Options = append(c.Options, Option{Key: "later", Label: "Wait for F11 first"})
	want := []string{
		"internal_id_in_question@question", "internal_id_in_question@big_picture", "internal_id_in_question@options[2].label",
		"path_in_question@big_picture", "missing_recommendation_reason@recommendation_reason",
	}
	if got := codes(c); !reflect.DeepEqual(got, want) {
		t.Fatalf("codes = %v, want %v", got, want)
	}
	c = clean
	c.Question, c.BigPicture = strings.Repeat("Should we keep going with this plan? ", 6), " "
	if got := codes(c); !reflect.DeepEqual(got, []string{"long_question@question", "missing_big_picture@big_picture"}) {
		t.Fatalf("codes = %v", got)
	}
}

func TestCoverProblemsAlsoReadsTheSubjectButNotTheTechnicalDetails(t *testing.T) {
	f := coldFacts()
	if problems := CoverProblems(f); len(problems) != 0 {
		t.Fatalf("a cold-readable decision was flagged: %v", problems)
	}
	f.Subject = "Run 1a2b3c4d5e6f needs review"
	problems := CoverProblems(f)
	if len(problems) != 1 || !strings.Contains(problems[0], "the subject uses internal labels") || !strings.Contains(problems[0], "1a2b3c4d5e6f") {
		t.Fatalf("problems = %v", problems)
	}
}

func TestPayloadCoverProblemsReadsThePublishedCover(t *testing.T) {
	f := coldFacts()
	if problems := PayloadCoverProblems(f.Payload()); len(problems) != 0 {
		t.Fatalf("a cold-readable payload was flagged: %v", problems)
	}
	f.Options[1].Label = "Run S-14 anyway"
	if problems := PayloadCoverProblems(f.Payload()); len(problems) != 1 || !strings.Contains(problems[0], "label of option run") {
		t.Fatalf("problems = %v", problems)
	}
}

func TestQuoteListNamesThreeThenCounts(t *testing.T) {
	if got := QuoteList([]string{"a", "b", "c", "d", "e"}); got != `"a", "b", "c", and 2 more` {
		t.Fatalf("QuoteList = %s", got)
	}
}
