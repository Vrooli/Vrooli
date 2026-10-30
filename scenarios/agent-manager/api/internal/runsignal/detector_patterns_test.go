package runsignal

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// timeline builds a run the way a transcript does (calls, results with exit
// codes, messages) and derives facts and episodes through the production path.
type timeline struct {
	at     time.Time
	events []*domain.RunEvent
	n      int
}

func newTimeline() *timeline {
	return &timeline{at: time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC)}
}

func (tl *timeline) add(kind domain.RunEventType, data domain.EventPayload) *domain.RunEvent {
	tl.at = tl.at.Add(time.Second)
	event := &domain.RunEvent{ID: uuid.New(), EventType: kind, Timestamp: tl.at, Data: data}
	tl.events = append(tl.events, event)
	return event
}

func (tl *timeline) idle(d time.Duration) { tl.at = tl.at.Add(d) }

// tool adds a call and its result.
func (tl *timeline) tool(name string, input map[string]any, exit int) *domain.RunEvent {
	tl.n++
	id := fmt.Sprintf("call-%d", tl.n)
	call := tl.add(domain.EventTypeToolCall, &domain.ToolCallEventData{ToolName: name, ToolCallID: id, Input: input})
	tl.add(domain.EventTypeToolResult, &domain.ToolResultEventData{ToolName: name, ToolCallID: id, Success: exit == 0, ExitCode: &exit})
	return call
}

func (tl *timeline) sh(command string, exit int) *domain.RunEvent {
	return tl.tool("Bash", map[string]any{"command": command}, exit)
}

func (tl *timeline) say(role, content string, terminal bool) {
	tl.add(domain.EventTypeMessage, &domain.MessageEventData{Role: role, Content: content, Terminal: terminal})
}

func (tl *timeline) change(path, kind string) {
	tl.add(domain.EventTypeToolCall, &domain.ToolCallEventData{ToolName: "file_change", Input: map[string]any{"files": []map[string]string{{"path": path, "kind": kind}}}})
}

func (tl *timeline) episodes() []FrictionEpisode {
	return DeriveEpisodes(DeriveInvocationFacts(tl.events), tl.events)
}

func (tl *timeline) of(pattern string) []FrictionEpisode {
	var out []FrictionEpisode
	for _, e := range tl.episodes() {
		if e.Pattern == pattern {
			out = append(out, e)
		}
	}
	return out
}

// expect asserts how many episodes of a pattern a timeline yields.
func expect(t *testing.T, name string, tl *timeline, pattern string, want int) []FrictionEpisode {
	t.Helper()
	got := tl.of(pattern)
	if len(got) != want {
		t.Errorf("%s: %d %s episodes %+v, want %d", name, len(got), pattern, got, want)
	}
	return got
}

func TestHandoffContinuationIsInformationalAndNeverFiresInGoalOrInjectedRuns(t *testing.T) {
	build := func(next string, goal bool) *timeline {
		tl := newTimeline()
		if goal {
			tl.add(domain.EventTypeGoalStatusChanged, nil)
		}
		tl.sh("ls", 0)
		tl.say("assistant", "done", true)
		tl.sh("ls docs", 0)
		tl.sh("ls internal", 0)
		if next != "" {
			tl.say("user", next, false)
		}
		return tl
	}
	got := expect(t, "plain follow-up", build("now do the next thing", false), "handoff-continuation", 1)
	if len(got) == 1 && (got[0].Turns != 2 || got[0].Tokens != 0 || got[0].WallClockMS != 0) {
		t.Errorf("handoff is counted, not priced: %+v", got[0])
	}
	expect(t, "goal-mode run", build("now do the next thing", true), "handoff-continuation", 0)
	goalRun := build("now do the next thing", false)
	if got := DeriveEpisodes(DeriveInvocationFacts(goalRun.events), goalRun.events, WithGoalRun(true)); len(got) != 0 {
		for _, e := range got {
			if e.Pattern == "handoff-continuation" {
				t.Errorf("a run with a goal_id flagged a handoff: %+v", e)
			}
		}
	}
	expect(t, "answering a wake", build("[wake] child ended", false), "handoff-continuation", 0)
	expect(t, "answering a notification", build("<task-notification>x</task-notification>", false), "handoff-continuation", 0)
	if spec, ok := PatternByID("handoff-continuation"); !ok || !spec.Informational || spec.Cost != CostNone {
		t.Errorf("handoff-continuation must be informational and unpriced: %+v", spec)
	}
}

func TestFallbackAfterCapabilityNeedsTwoCallsOfTheSameJob(t *testing.T) {
	one := newTimeline()
	one.sh("search-hub query foo && rg foo docs", 0)
	expect(t, "one compound call", one, "fallback-after-capability", 0)

	two := newTimeline()
	two.sh("search-hub query foo", 0)
	two.sh("rg foo docs", 0)
	got := expect(t, "two calls, same capability", two, "fallback-after-capability", 1)
	if len(got) == 1 && got[0].SuspectedOwnerScenario != "search-hub" {
		t.Errorf("owner = %q, want search-hub", got[0].SuspectedOwnerScenario)
	}

	other := newTimeline()
	other.sh("search-hub query foo", 0)
	other.sh("curl https://example.com", 0)
	expect(t, "an unrelated external tool", other, "fallback-after-capability", 0)
}

func TestGuidanceRepairNeedsTheOperator(t *testing.T) {
	build := func(correction string) *timeline {
		tl := newTimeline()
		tl.sh("ls docs", 0)
		tl.say("user", correction, false)
		tl.sh("cat README.md", 0)
		return tl
	}
	expect(t, "operator correction", build("Don't use that path, use the documented one."), "guidance-repair", 1)
	expect(t, "injected message", build("<task-notification>Don't use that path, use the documented one.</task-notification>"), "guidance-repair", 0)
	expect(t, "wake", build("[wake] Don't use that path, use the documented one."), "guidance-repair", 0)
}

func TestWrongPrimitiveIsLimitedToRecallSearches(t *testing.T) {
	for name, tc := range map[string]struct {
		command string
		want    int
	}{
		"searching docs":                   {"rg retry docs/plans", 1},
		"searching memory":                 {"grep -r retry .vrooli/work", 1},
		"searching prior sessions":         {"rg retry ~/.claude/projects", 1},
		"natural language, no path":        {`rg "what did we decide about the retry policy"`, 1},
		"find under docs":                  {"find docs/ -name '*.md'", 1},
		"code search in the tree":          {"rg handleRetry internal/", 0},
		"one file":                         {"grep -n retry api/main.go", 0},
		"a regex over the working tree":    {`rg "func (a|b)\\(" `, 0},
		"find in code":                     {"find internal -name '*.go'", 0},
		"single word pattern, no path":     {"rg retry", 0},
		"search-hub already used (no row)": {"search-hub query retry", 0},
	} {
		tl := newTimeline()
		tl.sh(tc.command, 0)
		tl.sh(tc.command, 0)
		expect(t, name, tl, "wrong-primitive", tc.want)
	}
}

func TestStallIsOnlyAnAgentOwedGapAfterAToolResult(t *testing.T) {
	min := time.Minute
	for name, tc := range map[string]struct {
		build func(*timeline)
		want  int
	}{
		"result then a late call": {func(tl *timeline) { tl.sh("ls", 0); tl.idle(10 * min); tl.sh("ls", 0) }, 1},
		"after a terminal message": {func(tl *timeline) {
			tl.sh("ls", 0)
			tl.say("assistant", "done", true)
			tl.idle(10 * min)
			tl.sh("ls", 0)
		}, 0},
		"after an evidence-only message": {func(tl *timeline) {
			tl.sh("ls", 0)
			tl.add(domain.EventTypeMessage, &domain.MessageEventData{Role: "assistant", EvidenceOnly: true})
			tl.idle(10 * min)
			tl.sh("ls", 0)
		}, 0},
		"an injected message in the gap": {func(tl *timeline) {
			tl.sh("ls", 0)
			tl.say("user", "[wake] child ended", false)
			tl.idle(10 * min)
			tl.sh("ls", 0)
		}, 0},
	} {
		tl := newTimeline()
		tc.build(tl)
		got := expect(t, name, tl, "stall", tc.want)
		for _, e := range got {
			if e.WallClockMS < int64(10*min/time.Millisecond) || e.Tokens != 0 {
				t.Errorf("%s: a stall costs its idle gap and no tokens: %+v", name, e)
			}
		}
	}
	if spec, _ := PatternByID("stall"); spec.TimeKind != TimeIdle || spec.Cost != CostGap {
		t.Errorf("stall time is an idle gap: %+v", spec)
	}
}

func TestReadThenRereadSkipsVolatileSourcesCompactionAndDistance(t *testing.T) {
	twice := func(path string, between func(*timeline)) *timeline {
		tl := newTimeline()
		tl.sh("cat "+path, 0)
		if between != nil {
			between(tl)
		}
		tl.sh("cat "+path, 0)
		return tl
	}
	expect(t, "plain file", twice("docs/PLAN.md", nil), "read-then-reread", 1)
	for _, path := range []string{"/proc/meminfo", "/sys/class/net/x", "api.log", "events.jsonl"} {
		expect(t, path, twice(path, nil), "read-then-reread", 0)
	}
	follow := newTimeline()
	follow.sh("tail -f out.txt", 0)
	follow.sh("tail -f out.txt", 0)
	expect(t, "tail -f", follow, "read-then-reread", 0)
	expect(t, "compaction between", twice("docs/PLAN.md", func(tl *timeline) {
		tl.add(domain.EventTypeCompaction, &domain.CompactionEventData{Summary: "s"})
	}), "read-then-reread", 0)
	expect(t, "far apart", twice("docs/PLAN.md", func(tl *timeline) {
		for i := 0; i < 25; i++ {
			tl.sh(fmt.Sprintf("echo %d", i), 0)
		}
	}), "read-then-reread", 0)
	expect(t, "edited between", twice("docs/PLAN.md", func(tl *timeline) { tl.change("docs/PLAN.md", "modify") }), "read-then-reread", 0)
}

func TestFlagHuntingIsRetryingAFailedCommandWithAnotherArgumentShape(t *testing.T) {
	hunt := newTimeline()
	hunt.sh("search-hub query --a", 2)
	hunt.sh("search-hub query --b", 2)
	hunt.sh("search-hub query --c", 0)
	expect(t, "another flag set after a failure", hunt, "flag-hunting", 1)

	same := newTimeline()
	for i := 0; i < 3; i++ {
		same.sh("search-hub query --a", 2)
	}
	expect(t, "one flag set repeated", same, "flag-hunting", 0)

	builtin := newTimeline()
	builtin.sh("cd /tmp && search-hub query --a", 2)
	builtin.sh("cd /var && search-hub query --a", 2)
	builtin.sh("cd /usr && search-hub query --a", 2)
	expect(t, "cd in front is not a flag", builtin, "flag-hunting", 0)

	noFailure := newTimeline()
	noFailure.sh("search-hub query --a", 0)
	noFailure.sh("search-hub query --b", 0)
	noFailure.sh("search-hub query --c", 0)
	expect(t, "no failure first", noFailure, "flag-hunting", 0)
}

func TestWaitMisuseIsConsecutiveShortWaitsOnOneJob(t *testing.T) {
	wait := func(tl *timeline, cell string, yield float64) {
		tl.tool("wait", map[string]any{"cell_id": cell, "yield_time_ms": yield}, 0)
	}
	three := newTimeline()
	for i := 0; i < 3; i++ {
		wait(three, "c1", 5000)
	}
	expect(t, "three short waits on one cell", three, "wait-misuse", 1)

	broken := newTimeline()
	wait(broken, "c1", 5000)
	wait(broken, "c1", 5000)
	broken.sh("ls", 0)
	wait(broken, "c1", 5000)
	expect(t, "a call between breaks the group", broken, "wait-misuse", 0)

	jobs := newTimeline()
	wait(jobs, "c1", 5000)
	wait(jobs, "c2", 5000)
	wait(jobs, "c3", 5000)
	expect(t, "different jobs", jobs, "wait-misuse", 0)

	blocking := newTimeline()
	for i := 0; i < 3; i++ {
		wait(blocking, "c1", 90_000)
	}
	expect(t, "native wait with a long yield", blocking, "wait-misuse", 0)

	designated := newTimeline()
	for i := 0; i < 3; i++ {
		designated.sh("test-genie runs wait --json agent-manager run-1", 0)
	}
	expect(t, "designated blocking wait", designated, "wait-misuse", 0)

	javascript := newTimeline()
	for i := 0; i < 3; i++ {
		javascript.tool("exec", map[string]any{"input": "await new Promise(r => setTimeout(r, 10))"}, 0)
	}
	expect(t, "await is not a wait", javascript, "wait-misuse", 0)
}

func TestFailureDetectorsMatchCommandPathAndTreatNoMatchAsNotFailure(t *testing.T) {
	noMatch := newTimeline()
	noMatch.sh("rg needle haystack", 1)
	noMatch.sh("rg needle haystack", 1)
	for _, pattern := range []string{"command-failure", "retry-after-failure", "capability-abandoned", "abandoned-command-family", "flag-hunting"} {
		expect(t, "rg exit 1", noMatch, pattern, 0)
	}

	failTwice := newTimeline()
	failTwice.sh("mytool run", 2)
	failTwice.sh("mytool run --verbose", 2)
	expect(t, "non-project command fails", failTwice, "command-failure", 0)
	expect(t, "same command retried", failTwice, "retry-after-failure", 1)

	differentCommands := newTimeline()
	differentCommands.sh("mytool run", 2)
	differentCommands.sh("othertool build", 2)
	expect(t, "another program is not a retry", differentCommands, "retry-after-failure", 0)

	// A search that found nothing later still ran: the family is not abandoned.
	recovered := newTimeline()
	recovered.sh("mytool run", 2)
	recovered.sh("ls", 0)
	recovered.sh("mytool run", 0)
	expect(t, "later success", recovered, "abandoned-command-family", 0)
}

func TestCapabilityAbandonedIgnoresUsageProbesAndCountsALaterSuccess(t *testing.T) {
	abandoned := newTimeline()
	abandoned.sh("search-hub query x", 2)
	abandoned.sh("ls", 0)
	expect(t, "gave up", abandoned, "capability-abandoned", 1)

	recovered := newTimeline()
	recovered.sh("search-hub query x", 2)
	recovered.sh("ls", 0)
	recovered.sh("search-hub query y", 0)
	expect(t, "later success of the family", recovered, "capability-abandoned", 0)

	probe := newTimeline()
	probe.sh("search-hub query --help", 2)
	probe.sh("ls", 0)
	expect(t, "a failed usage probe", probe, "capability-abandoned", 0)
}

func TestEditRevertIgnoresACodexRewrite(t *testing.T) {
	rewrite := newTimeline()
	rewrite.change("a.txt", "delete")
	rewrite.change("a.txt", "add")
	expect(t, "delete then add in one turn", rewrite, "edit-revert", 0)

	revert := newTimeline()
	revert.change("a.txt", "add")
	revert.say("assistant", "changed my mind", false)
	revert.change("a.txt", "delete")
	expect(t, "add then delete", revert, "edit-revert", 1)

	nextTurn := newTimeline()
	nextTurn.change("a.txt", "delete")
	nextTurn.say("assistant", "again", false)
	nextTurn.change("a.txt", "add")
	expect(t, "delete then add across turns", nextTurn, "edit-revert", 1)
}

func TestOscillationExcludesReadOnlyStatusCommands(t *testing.T) {
	cycle := func(a, b string) *timeline {
		tl := newTimeline()
		for i := 0; i < 2; i++ {
			tl.sh(a, 0)
			tl.sh(b, 0)
		}
		return tl
	}
	expect(t, "two real commands", cycle("search-hub query a", "prompt-manager skill read x"), "oscillation", 1)
	expect(t, "status polling", cycle("test-genie runs status r", "search-hub runs list"), "oscillation", 0)
	expect(t, "identity and ls", cycle("agent-manager run identity", "ls"), "oscillation", 0)
}

func TestRepeatedWorkNeedsNoStateChangeBetween(t *testing.T) {
	plain := newTimeline()
	plain.sh("make build", 0)
	plain.sh("make build", 0)
	expect(t, "same command twice", plain, "repeated-work", 1)

	edited := newTimeline()
	edited.sh("make build", 0)
	edited.change("main.go", "modify")
	edited.sh("make build", 0)
	expect(t, "edit between", edited, "repeated-work", 0)

	fixed := newTimeline()
	fixed.sh("make build", 2)
	fixed.sh("make build", 0)
	expect(t, "failing then passing", fixed, "repeated-work", 0)
}

func TestVrooliOwnedFailuresRouteToTheOwningCommand(t *testing.T) {
	tl := newTimeline()
	tl.sh("search-hub query a", 2)
	tl.sh("search-hub query b", 2)
	tl.sh("ls", 0)
	for _, pattern := range []string{"command-failure", "abandoned-command-family"} {
		for _, e := range tl.of(pattern) {
			if e.SuspectedOwnerScenario != "search-hub" || e.SuspectedOwnerCommand != "search-hub query" || e.OwnerConfidence != "manifest-derived" {
				t.Errorf("%s owner = %q/%q (%s), want search-hub / search-hub query", pattern, e.SuspectedOwnerScenario, e.SuspectedOwnerCommand, e.OwnerConfidence)
			}
		}
	}
	if len(tl.of("command-failure")) != 2 {
		t.Errorf("want one command-failure per failing call: %+v", tl.episodes())
	}
}

// command-failure is scenario friction: a failing project-owned command, one
// episode per call, whatever the cause; never a non-project command, a
// no-match, a help probe or an interrupt.
func TestCommandFailureIsScenarioFrictionOfProjectCommands(t *testing.T) {
	tl := newTimeline()
	call := tl.add(domain.EventTypeToolCall, &domain.ToolCallEventData{ToolName: "Bash", ToolCallID: "x1", Input: map[string]any{"command": "search-hub query a"}})
	exit := 1
	tl.add(domain.EventTypeToolResult, &domain.ToolResultEventData{ToolName: "Bash", ToolCallID: "x1", ExitCode: &exit, Output: "\n  capacity broker unavailable: lock held\nstack trace"})
	tl.sh("ls /nonexistent", 2)
	tl.sh("rg needle .", 1)
	tl.sh("python3 broken.py", 1)
	tl.sh("search-hub query --help", 2)
	tl.sh("search-hub query b", 130)
	got := expect(t, "one project failure", tl, "command-failure", 1)
	if len(got) == 1 {
		e := got[0]
		if e.RepeatedElement != "capacity broker unavailable: lock held" || e.SuspectedOwnerScenario != "search-hub" || e.SuspectedOwnerCommand != "search-hub query" || e.StartEventID != call.ID.String() {
			t.Errorf("episode = %+v", e)
		}
		spec, _ := PatternByID("command-failure")
		if text := spec.Render(e.SuspectedOwnerCommand, e.RepeatedElement, 0); text != "`search-hub query` failed: capacity broker unavailable: lock held" {
			t.Errorf("description = %q", text)
		}
	}
}

// An episode is priced by its own redundant calls, never by the run's usage.
func TestEpisodeCostIsItsOwnRedundantCalls(t *testing.T) {
	tl := newTimeline()
	tl.add(domain.EventTypeMetric, &domain.UsageEventData{InputTokens: 9_000_000, OutputTokens: 1_000_000})
	tl.sh("make build", 0)
	tl.sh("make build", 0)
	tl.add(domain.EventTypeMetric, &domain.UsageEventData{InputTokens: 9_000_000, OutputTokens: 1_000_000})
	facts := DeriveInvocationFacts(tl.events)
	for i := range facts {
		facts[i].ArgTokens, facts[i].ResultTokens = 30, 70
		ms := int64(2500)
		facts[i].DurationMS = &ms
	}
	var got []FrictionEpisode
	for _, e := range DeriveEpisodes(facts, tl.events) {
		if e.Pattern == "repeated-work" {
			got = append(got, e)
		}
	}
	if len(got) != 1 {
		t.Fatalf("episodes = %+v", got)
	}
	// repeated-work prices the repeating (second) call only.
	if got[0].Turns != 1 || got[0].Tokens != 100 || got[0].WallClockMS != 2500 {
		t.Errorf("episode = turns %d tokens %d ms %d, want 1 / 100 / 2500", got[0].Turns, got[0].Tokens, got[0].WallClockMS)
	}
}

func TestPatternTableIsTheOnlyList(t *testing.T) {
	seen := map[string]bool{}
	for _, spec := range PatternSpecs() {
		if spec.ID == "" || spec.Version != EpisodeClassifierVersion || spec.Cause == "" || spec.TimeKind == "" || spec.Cost == "" || strings.TrimSpace(spec.Description) == "" || spec.detect == nil {
			t.Errorf("incomplete spec %+v", spec)
		}
		if seen[spec.ID] {
			t.Errorf("duplicate pattern %s", spec.ID)
		}
		seen[spec.ID] = true
	}
	if _, ok := PatternByID("time-to-first-success"); ok {
		t.Error("time-to-first-success is deleted")
	}
	if len(EpisodeDetectors()) != len(PatternSpecs()) {
		t.Error("detectors are the table")
	}
	if got := (PatternSpec{Description: "{Cmd} fails.", DescriptionRepeated: "{Cmd} fails {n} times."}).Render("tool x", "", 3); got != "`tool x` fails 3 times." {
		t.Errorf("render = %q", got)
	}
}

func TestIsCorrectionAboutPriorAction(t *testing.T) {
	for _, text := range []string{
		"That's not what I asked for", "that's wrong, use the other file", "Why did you delete that?",
		"I said use the CLI", "You should have run the tests first", "Don't do that again", "No, that is the wrong file",
		"Wrong approach, revert it", "not what I asked",
	} {
		if !isCorrection(text) {
			t.Errorf("expected a correction: %q", text)
		}
	}
	for _, text := range []string{
		"No problem, go ahead", "stop here and commit", "You need to run the build next", "Stop. Then summarise.",
		"There is a mistake budget in the plan doc", "Nice, correction applied thanks", "no worries", "Never mind, continue",
		"you should run the linter",
	} {
		if isCorrection(text) {
			t.Errorf("not a correction: %q", text)
		}
	}
}

// An operator correction belongs to the agent, never to the command that ran
// beside it, so corrections group into one problem whatever the agent ran.
func TestGuidanceRepairHasNoCommandOwner(t *testing.T) {
	spec, _ := PatternByID("guidance-repair")
	fact := InvocationFact{CallEventID: "c1", Ownership: OwnershipResolved, Executable: "vrooli", CommandPath: "vrooli scenario logs"}
	e := newEpisode(spec.ID, fact, fact, nil, nil)
	if e.SuspectedOwnerScenario != "" || e.SuspectedOwnerCommand != "" || e.Fingerprint != fingerprint(spec.ID, "", spec.Cause) {
		t.Fatalf("episode = %+v, want no command owner", e)
	}
}
