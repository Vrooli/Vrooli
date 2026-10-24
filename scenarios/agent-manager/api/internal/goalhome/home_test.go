package goalhome

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// fixtureFiles copies the selected home fixture and applies text edits in memory.
// An empty edit removes a file, allowing missing-file cases without another fake FS.
func fixtureFiles(t *testing.T, name string, edits map[string]func(string) string) fstest.MapFS {
	t.Helper()
	files := fstest.MapFS{}
	root := os.DirFS(filepath.Join("testdata", name))
	if err := fs.WalkDir(root, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(root, name)
		files[name] = &fstest.MapFile{Data: data}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for name, edit := range edits {
		var before string
		if f := files[name]; f != nil {
			before = string(f.Data)
		}
		if after := edit(before); after != "" {
			files[name] = &fstest.MapFile{Data: []byte(after)}
		} else {
			delete(files, name)
		}
	}
	return files
}

func basHome(t *testing.T, edits map[string]func(string) string) *Home {
	t.Helper()
	home, err := LoadLegacy(fixtureFiles(t, "bas", edits))
	if err != nil {
		t.Fatal(err)
	}
	return home
}

func blockingCodes(report *LintReport) string {
	var codes []string
	for _, finding := range report.Findings {
		if finding.Blocking {
			codes = append(codes, finding.Code)
		}
	}
	return strings.Join(codes, ",")
}

func TestRealBASGoalHomeParses(t *testing.T) {
	home := basHome(t, nil)
	queue := home.Queue
	if len(queue.Handoffs) != 1 || queue.Handoffs[0].Title != "Handoff" || len(queue.NeedsOperator) != 0 || len(queue.Next) != 5 || len(queue.ReadyNext()) != 0 || len(queue.WaitingOnOthers) != 2 {
		t.Fatalf("queue sections misread: %+v", queue)
	}
	var feedback []string
	for _, entry := range home.Feedback.Entries {
		feedback = append(feedback, fmt.Sprintf("%s:%t:%t", entry.ID, entry.Open(), entry.Steering()))
	}
	if strings.Join(feedback, " ") != "BAS-FB-064:true:true BAS-FB-063:true:true BAS-SUP-AUDIT-003:true:true BAS-FB-036:false:true" {
		t.Fatalf("feedback entries = %v", feedback)
	}
	open := 0
	for _, entry := range home.Workarounds.Entries {
		if entry.Open {
			open++
		}
	}
	if len(home.Workarounds.Entries) != 5 || open != 3 {
		t.Fatalf("workarounds = %d entries, %d open; want 5 and 3", len(home.Workarounds.Entries), open)
	}
	if len(home.Epochs) != 4 || home.Epochs[0].Name != "E23.md" || !home.Epochs[2].Epoch.IsAccepted() || home.Epochs[3].Epoch.IsAccepted() {
		t.Fatalf("epochs misread: %+v", home.Epochs)
	}
	if home.Goal.Destination != nil {
		t.Fatalf("the BAS GOAL.md states its destination in prose; got %+v", home.Goal.Destination)
	}
}

func TestLintBASGoalHome(t *testing.T) {
	duplicate := func(s string) string {
		return strings.Replace(s, "## Waiting on others", "### Orchestrator handoff — 2026-10-06T16:43Z\n\nNothing changed; parking for 12 hours.\n\n## Waiting on others", 1)
	}
	for _, tc := range []struct {
		name  string
		edits map[string]func(string) string
		want  string
	}{
		// Live BAS on 2026-10-07: no slice marked [ready], the Needs operator
		// placeholder is empty, and FB-063 and the SUP-AUDIT-003 supervisor
		// audit are open but the handoff names only FB-064.
		{name: "as recorded", want: "idle-queue,feedback-unacknowledged,feedback-unacknowledged"},
		{name: "duplicate handoff is named", edits: map[string]func(string) string{QueueFile: duplicate}, want: "handoff-count,idle-queue,feedback-unacknowledged,feedback-unacknowledged"},
		{name: "a [ready] slice clears the idle queue", edits: map[string]func(string) string{QueueFile: func(s string) string {
			return strings.Replace(s, "3. **RD-EXP", "3. [ready] **RD-EXP", 1)
		}}, want: "feedback-unacknowledged,feedback-unacknowledged"},
		{name: "a Needs operator item clears the idle queue", edits: map[string]func(string) string{QueueFile: func(s string) string {
			return strings.Replace(s, "(none; the operator answered BAS-FB-063 on 2026-10-06)", "- Retire Record mode? Options: keep, retire. Recommendation: keep.", 1)
		}}, want: "feedback-unacknowledged,feedback-unacknowledged"},
		{name: "naming the operator feedback leaves the supervisor audit open", edits: map[string]func(string) string{QueueFile: func(s string) string {
			return strings.Replace(s, "- The census notes", "- FB-063 is translated into the RD slices.\n- The census notes", 1)
		}}, want: "idle-queue,feedback-unacknowledged"},
		{name: "naming every open entry acknowledges them", edits: map[string]func(string) string{QueueFile: func(s string) string {
			return strings.Replace(s, "- The census notes", "- FB-063 is translated into the RD slices; SUP-AUDIT-003 is tracked under Waiting on others.\n- The census notes", 1)
		}}, want: "idle-queue"},
		{name: "a supervisor-audit source is checked", edits: map[string]func(string) string{
			QueueFile: func(s string) string {
				return strings.Replace(s, "- The census notes", "- FB-063 is translated into the RD slices.\n- The census notes", 1)
			},
			FeedbackFile: func(s string) string {
				return strings.Replace(s, "(2026-10-06, supervisor)", "(2026-10-06, supervisor-audit)", 1)
			},
		}, want: "idle-queue,feedback-unacknowledged"},
		{name: "another source is not checked", edits: map[string]func(string) string{
			QueueFile: func(s string) string {
				return strings.Replace(s, "- The census notes", "- FB-063 is translated into the RD slices.\n- The census notes", 1)
			},
			FeedbackFile: func(s string) string { return strings.Replace(s, "(2026-10-06, supervisor)", "(2026-10-06, qa)", 1) },
		}, want: "idle-queue"},
		{name: "a missing handoff is named", edits: map[string]func(string) string{QueueFile: func(s string) string {
			start, end := strings.Index(s, "## Handoff"), strings.Index(s, "## Forecast")
			return s[:start] + s[end:]
		}}, want: "handoff-count,idle-queue,feedback-unacknowledged,feedback-unacknowledged,feedback-unacknowledged"},
		{name: "an oversized handoff", edits: map[string]func(string) string{QueueFile: func(s string) string {
			return strings.Replace(s, "- The census notes", "- "+strings.Repeat("census prose ", 400)+"\n- The census notes", 1)
		}}, want: "handoff-size,idle-queue,feedback-unacknowledged,feedback-unacknowledged"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := Lint(basHome(t, tc.edits))
			if got := blockingCodes(report); got != tc.want {
				t.Fatalf("blocking findings = %s, want %s: %+v", got, tc.want, report.Findings)
			}
			if first := report.Findings[0]; first.Code == "handoff-count" && tc.name == "duplicate handoff is named" && (!strings.Contains(first.Detail, `"## Handoff" (line 10)`) || !strings.Contains(first.Detail, `"### Orchestrator handoff — 2026-10-06T16:43Z"`)) {
				t.Fatalf("duplicate handoff finding must name both sections: %s", first.Detail)
			}
			if !report.Blocking {
				t.Fatal("blocking findings must make the report blocking")
			}
		})
	}
}

func TestLintResumeBudgetCountsOnlyOpenEntries(t *testing.T) {
	padding := strings.Repeat("x", ResumeBudgetBytes)
	for _, tc := range []struct {
		name   string
		status string
		want   bool
	}{
		{name: "open entry counts", status: "open.", want: true},
		{name: "resolved entry does not", status: "resolved 2026-10-07.", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := Lint(basHome(t, map[string]func(string) string{WorkaroundsFile: func(s string) string {
				return s + "- 2026-10-07 · " + padding + " · fallback · retry · " + tc.status + "\n"
			}}))
			if got := strings.Contains(blockingCodes(report), "resume-budget"); got != tc.want {
				t.Fatalf("resume-budget = %t, want %t (open workarounds %d bytes)", got, tc.want, report.ResumeSet.OpenWorkarounds)
			}
		})
	}
	report := Lint(basHome(t, map[string]func(string) string{FeedbackFile: func(s string) string { return s + "| BAS-FB-001 | " + padding + " |\n" }}))
	if strings.Contains(blockingCodes(report), "resume-budget") {
		t.Fatalf("resolved feedback must not count: %+v", report.ResumeSet)
	}
}

func TestLintAdvisesWhenLoadedEpochExceedsBoardDefaultWithoutChangingResumeBudget(t *testing.T) {
	baseline := Lint(basHome(t, nil))
	epoch := "# E99\n\n- Status: accepted\n\n## Directives\n- D1 2026-10-20T03:00:00Z retain the acceptance gate\n\n## Slice log\n2026-10-20T03:01:00Z | verified typed state | exit metric=gates pass | net runtime lines=0 | net test lines=0 | ack=D1\n\n## Notes\n" + strings.Repeat("x\n", DefaultBoardFileBytes/2+2)
	home := basHome(t, map[string]func(string) string{
		"epochs/E99.md": func(string) string { return epoch },
	})

	if got := home.FileBytes["epochs/E99.md"]; got != len(epoch) {
		t.Fatalf("loaded epoch byte count = %d, want %d", got, len(epoch))
	}
	if home.FileBytes[GoalFile] != home.Goal.Bytes || home.FileBytes[QueueFile] != home.Queue.Bytes {
		t.Fatalf("root file sizes were not retained from the same load: %+v", home.FileBytes)
	}
	var parsed *Epoch
	for _, epochFile := range home.Epochs {
		if epochFile.Name == "E99.md" {
			parsed = epochFile.Epoch
			break
		}
	}
	if parsed == nil || len(parsed.Directives) != 1 || len(parsed.Log) != 1 || len(parsed.Log[0].Acks) != 1 || parsed.Log[0].Acks[0] != "D1" {
		t.Fatalf("oversized advisory must retain the parsed epoch's typed state: %+v", parsed)
	}
	report := Lint(home)
	var advisory *Finding
	for i := range report.Findings {
		if report.Findings[i].Code == "board-file-size-advisory" && report.Findings[i].File == "epochs/E99.md" {
			advisory = &report.Findings[i]
			break
		}
	}
	if advisory == nil || advisory.Blocking || !strings.Contains(advisory.Detail, "configured board limits are authoritative") {
		t.Fatalf("oversized epoch should have a nonblocking default-baseline advisory: %+v", report.Findings)
	}
	if report.ResumeSet != baseline.ResumeSet {
		t.Fatalf("epoch advisory must not change the distinct resume context budget: got %+v, baseline %+v", report.ResumeSet, baseline.ResumeSet)
	}
}

func TestSetHandoff(t *testing.T) {
	queue := readFixture(t, "bas", QueueFile)
	body := "- 2026-10-07T15:00Z: E26 accepted. Next: admit JX-2 as E27."
	for _, tc := range []struct {
		name    string
		queue   string
		body    string
		changed bool
		code    string
		check   func(t *testing.T, out string)
	}{
		{name: "replaces the single handoff", queue: queue, body: body, changed: true, check: func(t *testing.T, out string) {
			if !strings.Contains(out, "## Handoff\n\n"+body+"\n\n## Forecast") || strings.Contains(out, "act on BAS-FB-064") || strings.Count(out, "## Handoff") != 1 {
				t.Fatalf("handoff not replaced in place:\n%s", out)
			}
		}},
		{name: "accepts a leading heading in the text", queue: queue, body: "## Handoff\n\n" + body, changed: true},
		{name: "creates the section after Needs operator", queue: strings.Replace(queue, queue[strings.Index(queue, "## Handoff"):strings.Index(queue, "## Forecast")], "", 1), body: body, changed: true, check: func(t *testing.T, out string) {
			if !strings.Contains(out, "2026-10-06)\n\n## Handoff\n\n"+body+"\n\n## Forecast") {
				t.Fatalf("handoff not created after Needs operator:\n%s", out)
			}
		}},
		{name: "unchanged text writes nothing", queue: strings.Replace(queue, queue[strings.Index(queue, "## Handoff"):strings.Index(queue, "## Forecast")], "## Handoff\n\n"+body+"\n\n", 1), body: body + "\n\n", changed: false},
		{name: "refuses over 4 KB", queue: queue, body: strings.Repeat("x", HandoffMaxBytes+1), code: "handoff-size"},
		{name: "refuses a duplicate handoff", queue: strings.Replace(queue, "## Waiting on others", "### Orchestrator handoff — 2026-10-06T16:43Z\n\nparked\n\n## Waiting on others", 1), body: body, code: "handoff-count"},
		{name: "refuses a section heading in the text", queue: queue, body: body + "\n\n## Next\n\n1. [ready] JX-2", code: "handoff-content"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, changed, err := SetHandoff([]byte(tc.queue), tc.body)
			var refused *HandoffError
			if tc.code != "" {
				if !errors.As(err, &refused) || refused.Code != tc.code {
					t.Fatalf("want refusal %s, got %v", tc.code, err)
				}
				return
			}
			if err != nil || changed != tc.changed {
				t.Fatalf("changed=%t err=%v, want changed=%t", changed, err, tc.changed)
			}
			if !changed && string(out) != tc.queue {
				t.Fatal("an unchanged handoff must return the queue byte for byte")
			}
			if tc.check != nil {
				tc.check(t, string(out))
			}
			if parsed := ParseQueue(out); len(parsed.Handoffs) != 1 || parsed.Handoffs[0].Body != body {
				t.Fatalf("result must hold exactly the one new handoff: %+v", parsed.Handoffs)
			}
		})
	}
}

func TestWriteIfUnchangedRefusesAConcurrentChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), QueueFile)
	if err := os.WriteFile(path, []byte("written by another session\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := WriteIfUnchanged(path, []byte("what we read\n"), []byte("our update\n")); !errors.Is(err, ErrConcurrentChange) {
		t.Fatalf("want ErrConcurrentChange, got %v", err)
	}
	if data, _ := os.ReadFile(path); string(data) != "written by another session\n" {
		t.Fatalf("a refused write must leave the file untouched: %q", data)
	}
	if err := WriteIfUnchanged(path, []byte("written by another session\n"), []byte("our update\n")); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if data, _ := os.ReadFile(path); string(data) != "our update\n" || info.Mode().Perm() != 0o640 {
		t.Fatalf("write did not replace the file keeping its mode: %q %v", data, info.Mode())
	}
	if entries, _ := os.ReadDir(filepath.Dir(path)); len(entries) != 1 {
		t.Fatalf("temporary files left behind: %v", entries)
	}
}

// forecastHome builds a goal home whose epochs E1..En were accepted an hour
// apart with the given "estimate/yield" pairs (empty means absent).
func forecastHome(t *testing.T, goal, queue string, epochs ...string) *Home {
	t.Helper()
	files := fstest.MapFS{GoalFile: {Data: []byte(goal)}, QueueFile: {Data: []byte(queue)}}
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	for i, pair := range epochs {
		estimate, yield, _ := strings.Cut(pair, "/")
		header := ""
		if estimate != "" {
			header = "- Yield estimate: " + estimate + "\n"
		}
		accepted := fmt.Sprintf("ACCEPTED %s", start.Add(time.Duration(i)*time.Hour).Format(time.RFC3339))
		if yield != "" {
			accepted += " | yield=" + yield
		}
		files[fmt.Sprintf("epochs/E%d.md", i+1)] = &fstest.MapFile{Data: []byte(fmt.Sprintf("# E%d\n\n- Kind: refactor\n%s\n## Slice log\n%s | accepted\n", i+1, header, accepted))}
	}
	home, err := LoadLegacy(files)
	if err != nil {
		t.Fatal(err)
	}
	return home
}

func TestDiminishingReturns(t *testing.T) {
	goal := "# Goal\n\n- Destination: runtime lines <= 205000\n- Current: 253452 @ 2026-10-07\n"
	queue := "# Queue\n\n## Needs operator\n\n## Handoff\n\nnone\n"
	censuses := func(values ...string) string {
		text := queue + "\n## Censuses\n\n"
		for i, value := range values {
			text += fmt.Sprintf("- 2026-10-0%dT00:00:00Z | workflow owners | admissible=%s\n", i+1, value)
		}
		return text
	}
	for _, tc := range []struct {
		name     string
		goal     string
		queue    string
		epochs   []string
		fired    string
		blocking bool
	}{
		// BAS E23–E26 history: no parseable estimates, yet at about 380 lines an
		// epoch the 48k gap needs ~127 epochs, so only the gap horizon catches O3.
		{name: "gap horizon catches the BAS history", goal: goal, queue: queue, epochs: []string{"/-328 runtime lines", "/-66 runtime lines", "/-332 runtime lines", "/-801 runtime lines"}, fired: "gap-horizon", blocking: true},
		{name: "three low-yield epochs in a row", goal: "# Goal\n", queue: queue, epochs: []string{"-600 runtime lines/-100 runtime lines", "−600 runtime lines/-140 runtime lines", "-1,000 runtime lines/+20 runtime lines"}, fired: "low-yield", blocking: true},
		{name: "one epoch at a quarter of estimate breaks the run", goal: "# Goal\n", queue: queue, epochs: []string{"-600 runtime lines/-100 runtime lines", "-600 runtime lines/-150 runtime lines", "-600 runtime lines/-100 runtime lines"}},
		{name: "two empty censuses", goal: "# Goal\n", queue: censuses("RD-WF", "none", "none"), fired: "empty-censuses", blocking: true},
		{name: "an admissible census in between", goal: "# Goal\n", queue: censuses("none", "RD-WF")},
		{name: "a destination reset starts a new window", goal: goal + "- Destination set: 2026-10-02T00:00:00Z\n", queue: queue, epochs: []string{"/-328 runtime lines", "/-66 runtime lines"}},
		{name: "a [re-aim] decision acknowledges the finding", goal: goal, queue: strings.Replace(queue, "## Needs operator\n", "## Needs operator\n\n- [re-aim] Close the line-count phase? Options: re-aim at journeys, close.\n", 1), epochs: []string{"/-328 runtime lines"}, fired: "gap-horizon"},
		{name: "a reachable destination", goal: goal, queue: queue, epochs: []string{"/-5000 runtime lines", "/-4000 runtime lines"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			forecast := ForecastOf(forecastHome(t, tc.goal, tc.queue, tc.epochs...))
			var fired []string
			for _, rule := range forecast.Fired() {
				fired = append(fired, rule.Name)
			}
			if strings.Join(fired, ",") != tc.fired {
				t.Fatalf("fired %v, want %q: %+v", fired, tc.fired, forecast.Rules)
			}
			finding := forecast.Finding()
			if (finding != nil && finding.Blocking) != tc.blocking {
				t.Fatalf("finding = %+v, want blocking=%t", finding, tc.blocking)
			}
		})
	}
}

func TestDecidePark(t *testing.T) {
	decision := []Item{{Text: "Retire Record mode? Options: keep, retire.", Line: 7}}
	readyAndDecision := Queue{
		NeedsOperator: decision,
		Next:          []Item{{Text: "[ready] Independent slice", Ready: true, Line: 12}},
	}
	for _, tc := range []struct {
		name      string
		children  []string
		queue     Queue
		requested time.Duration
		allowed   bool
		rule      string
		timeout   time.Duration
	}{
		{name: "live child gets the backstop", children: []string{"worker"}, rule: ParkRuleLiveChild, allowed: true, timeout: time.Hour},
		{name: "live child refuses a 12h park", children: []string{"worker"}, requested: 12 * time.Hour, rule: ParkRuleLiveChild},
		{name: "a decision allows 72h when no slice is ready", queue: Queue{NeedsOperator: decision}, rule: ParkRuleNeedsOperator, allowed: true, timeout: 72 * time.Hour},
		{name: "a shorter wait is kept when no slice is ready", queue: Queue{NeedsOperator: decision}, requested: 2 * time.Hour, rule: ParkRuleNeedsOperator, allowed: true, timeout: 2 * time.Hour},
		{name: "ready independent work takes precedence over an operator hold", queue: readyAndDecision, requested: 12 * time.Hour, rule: ParkRuleAdmissible},
		{name: "live child backstop still applies with ready work and an operator hold", children: []string{"worker"}, queue: readyAndDecision, rule: ParkRuleLiveChild, allowed: true, timeout: time.Hour},
		// DL-1 Gherkin: no live child and nothing under Needs operator.
		{name: "neither refuses with the admissible-slice rule", requested: 12 * time.Hour, rule: ParkRuleAdmissible},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := DecidePark(tc.children, tc.queue, tc.requested)
			if got.Allowed != tc.allowed || got.Rule != tc.rule || (tc.allowed && got.Timeout != tc.timeout) {
				t.Fatalf("decision = %+v", got)
			}
			if !got.Allowed && tc.rule == ParkRuleAdmissible && !strings.Contains(got.Reason, "Admissible-slice rule") {
				t.Fatalf("refusal must state the rule: %s", got.Reason)
			}
		})
	}
}
