package toolfriction

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/runsignal"
)

// A finding is what it is, not who a week's majority attributes it to: when
// the CLI that printed most of a line changes, its fingerprint (and so its
// reported or dismissed status, and the bug filed for it) stays.
func TestFingerprintIsKindAndSignatureNotTheOwnerVote(t *testing.T) {
	window := func(searchHub, promptManager int) Finding {
		s := newSession()
		for i := 0; i < searchHub; i++ {
			s.shell("search-hub query x", 1, "Error: index locked by another writer")
		}
		for i := 0; i < promptManager; i++ {
			s.shell("prompt-manager skill read x", 1, "Error: index locked by another writer")
		}
		for _, f := range s.report().Findings {
			if f.Kind == KindFailure {
				return f
			}
		}
		t.Fatal("no failure finding")
		return Finding{}
	}
	before, after := window(3, 1), window(1, 3)
	if before.Owner == after.Owner {
		t.Fatalf("the majority should flip: %s, %s", before.Owner, after.Owner)
	}
	if before.Fingerprint != after.Fingerprint {
		t.Errorf("fingerprint changed with the owner vote: %s (%s) → %s (%s)", before.Fingerprint, before.Owner, after.Fingerprint, after.Owner)
	}
}

// The materiality floor reads the score that ranks findings, √(share ×
// breadth): a park refusal in a third of the runs is material though it is
// under 1% of the Vrooli CLI calls. Its inline time is the blocking wait
// itself, which parking would not shorten: stated as held, never priced. A
// degraded call that is not a wait carries its time as a share.
func TestAWidespreadDegradationIsMaterialByScoreAndStatesItsWaitAsHeld(t *testing.T) {
	s := newSession()
	for run := 0; run < 40; run++ {
		s.run = fmt.Sprintf("run-%02d", run)
		for i := 0; i < 50; i++ {
			s.shell("agent-manager run list", 0, "[]")
		}
		if run%3 == 0 {
			s.shell("test-genie runs wait r1", 0, "agent-manager park unavailable (park request: api error (409): cannot park Run in running state: runner type is unavailable; cannot safely end the parked turn) — waiting inline instead")
			s.shell("search-hub query x", 0, "search-hub degraded: embeddings unavailable, falling back to lexical search")
		}
	}
	r := s.report()
	f := findingOf(t, r, KindDegraded, "agent-manager")
	if f.Calls != 14 || f.Runs != 14 {
		t.Fatalf("park finding = %+v", f)
	}
	if f.CostShare >= MaterialScore || !f.Material() || f.Score != Score(f.CostShare, 14, 40) {
		t.Errorf("share %.4f is under the floor but the score %.4f (√(share × 14/40 runs)) is not: material=%v", f.CostShare, f.Score, f.Material())
	}
	for _, share := range f.Shares {
		if share.Kind == HarmWallClock {
			t.Errorf("a blocking wait's time is not a saving: shares %+v", f.Shares)
		}
	}
	if !strings.Contains(f.Description, "21.0 s held inline by those blocking waits (the wait itself, not a saving)") || strings.Contains(f.Description, "spent in those calls") {
		t.Errorf("description = %s", f.Description)
	}
	g := findingOf(t, r, KindDegraded, "search-hub")
	if len(g.Shares) != 2 || !strings.Contains(g.Description, "21.0 s spent in those calls") {
		t.Errorf("a degraded query carries its time: %+v", g)
	}
}

// A failure that printed no error line names nothing but its command: such
// failures of different commands are each command's finding, never one
// "exit code" line pooled onto the control plane.
func TestASilentFailureIsItsCommandsFinding(t *testing.T) {
	s := newSession()
	for i := 0; i < 3; i++ {
		s.shell("vrooli scenario restart agent-manager", 1, "")
		s.shell("unit-health validate scenario agent-manager", 1, "")
	}
	r := s.report()
	for _, owner := range []string{"vrooli", "unit-health"} {
		f := findingOf(t, r, KindFailure, owner)
		if f.CommandPath == "" || f.Calls != 3 || !strings.Contains(f.Description, "without an error line (exit code 1)") {
			t.Errorf("%s: %+v", owner, f)
		}
	}
	failures := 0
	for _, f := range r.Findings {
		if f.Kind == KindFailure {
			failures++
		}
	}
	if failures != 2 {
		t.Errorf("silent failures pooled across commands: %d failure findings", failures)
	}
}

// A test command's non-zero exit is the suite's verdict, declared by its
// manifest (`semantics: verify`, on a command or a durable-run exception), not
// a tool failure. A test command called with the wrong shape, or stopped by an
// error of the tool (admission saturated), never reached a verdict: that is a
// failure.
func TestATestVerdictIsNotAToolFailure(t *testing.T) {
	s := newSession()
	for i := 0; i < 3; i++ {
		s.shell("vrooli scenario test agent-manager --phases unit", 1, "Error: suite execution completed with failures")
		s.shell("test-genie runs wait agent-manager 20261030-1", 1, "run 20261030-1 failed: 3 phases failed")
		s.shell("test-genie execute agent-manager", 1, "Error: suite execution completed with failures")
	}
	for i := 0; i < 3; i++ {
		s.shell("vrooli scenario test agent-manager --phase unit", 1, "Error: unknown flag: --phase")
		s.shell("test-genie execute agent-manager", 1, "Error: start run: resource_exhausted: test-genie admission is saturated")
	}
	r := s.report()
	if p := profileOf(t, r, "test-genie runs wait"); p.Failed != 0 {
		t.Errorf("a failed suite is a verdict, not a failure: %+v", p)
	}
	if p := profileOf(t, r, "vrooli scenario test"); p.Failed != 3 || p.WrongShape != 3 {
		t.Errorf("only the wrong-shape calls failed: %+v", p)
	}
	if p := profileOf(t, r, "test-genie execute"); p.Failed != 3 {
		t.Errorf("only the saturated starts failed, which reached no verdict: %+v", p)
	}
	for _, f := range r.Findings {
		if f.Kind == KindFailure && !strings.Contains(f.ErrorLine, "saturated") {
			t.Errorf("a verdict became a failure finding: %+v", f)
		}
	}
}

// A summary that some providers were degraded names them from the lines below
// it: each degraded provider is its own finding, and one whose notes the output
// read does not reach says so.
func TestADegradedSummaryNamesTheDegradedProvider(t *testing.T) {
	s := newSession()
	summary := "Summary:\n  Searched 10 corpora; 74 total hit(s) (some providers degraded — see notes below).\nResults by provider:\n"
	for i := 0; i < 3; i++ {
		s.run = fmt.Sprintf("run-%d", i)
		s.shell("search-hub query a", 0, summary+"  ▸ agent-manager.runs — degraded: provider agent-manager.runs circuit unavailable\n  ▸ knowledge-observatory.docs (10)")
		s.shell("search-hub query b", 0, summary+"  1. Some result")
	}
	var named, unnamed int
	for _, f := range s.report().Findings {
		if f.Kind != KindDegraded {
			continue
		}
		if f.Owner != "search-hub" || f.Calls != 3 {
			t.Errorf("degraded finding = %+v", f)
		}
		switch {
		case strings.Contains(f.ErrorLine, "[degraded: agent-manager.runs]"):
			named++
		case strings.Contains(f.ErrorLine, "degraded provider not named in the output read"):
			unnamed++
		}
	}
	if named != 1 || unnamed != 1 {
		t.Errorf("want one finding naming agent-manager.runs and one saying the provider was not named: %d, %d", named, unnamed)
	}
}

// When a finding's cost counts fewer calls than its sentence names (one run's
// loop capped), the description says which number is which.
func TestAFindingStatesWhatItsCostCounts(t *testing.T) {
	s := newSession()
	s.run = "loop"
	for i := 0; i < 40; i++ {
		s.shell("search-hub query x", 1, "Error: index locked by another writer")
	}
	for _, run := range []string{"a", "b", "c"} {
		s.run = run
		s.shell("search-hub query x", 1, "Error: index locked by another writer")
	}
	f := findingOf(t, s.report(), KindFailure, "search-hub")
	if f.Calls != 43 || f.Cost != 13 || !strings.Contains(f.Description, "its cost counts 13 of those 43 calls, each run at most 10x the median run's") {
		t.Errorf("finding = %+v", f)
	}
	// Tokens too: the re-read total in the sentence is uncapped, the cost is not.
	w := newSession()
	w.run = "loop"
	for i := 0; i < 40; i++ {
		w.tool("Bash", map[string]any{"command": "vrooli locks"}, 0, "locks", 10_000)
	}
	for _, run := range []string{"a", "b", "c"} {
		w.run = run
		w.tool("Bash", map[string]any{"command": "vrooli locks"}, 0, "locks", 10_000)
	}
	g := findingOf(t, w.report(), KindOutputWeight, "vrooli")
	if !strings.Contains(g.Description, "4.3M tokens re-read in later requests") || !strings.Contains(g.Description, "its cost counts 1.3M of those 4.3M tokens, each run at most 10x the median run's") {
		t.Errorf("finding = %s", g.Description)
	}
}

// A command's finding says how many more of its calls the window holds at an
// older fact version: they are not counted, and a reader comparing it with a
// count over every version can tell why.
func TestAFindingSaysHowManyOfItsCallsAwaitReDerivation(t *testing.T) {
	s := newSession()
	for i := 0; i < 3; i++ {
		s.shell("agent-manager runs list", 1, `Error: Unknown command: runs (did you mean "run"?)`)
	}
	r := Analyze(Window{Calls: s.calls(), Stale: Stale{Runs: 29, Calls: 300, Paths: map[string]int{"agent-manager": 271}}})
	if f := findingOf(t, r, KindShapeGuess, "agent-manager"); !strings.Contains(f.Description, "271 more calls of it await re-derivation") {
		t.Errorf("finding = %s", f.Description)
	}
}

// Reading one finished result again in another shape is not polling: run
// 0bbbdd88 read `test-genie runs show <id> --json` and, seconds later, piped
// the same result through two different jq filters. The same read repeated in
// the same shape stays a poll however soon, and so does a reshape that comes
// after the reshape gap, when the state could have changed.
func TestReshapingAResultJustReadIsNotPolling(t *testing.T) {
	soon := func(s *session) *session { s.at = s.at.Add(-50 * time.Second); return s } // the next call 10 s later
	show := "test-genie runs show 20261029-051055-7f2f41f5 --scenario device-control --json"
	s := newSession()
	s.shell(show, 0, `{"run":{"status":"failed"}}`)
	soon(s).shell(show+" | jq '.run.phases[] | .name, .findings[].severity'", 0, "unit")
	soon(s).shell(show+" | jq '.run.phases[].name'", 0, "unit")
	if p := profileOf(t, s.report(), "test-genie runs show"); p.Polled != 0 {
		t.Errorf("reshaping a result just read was counted as %d polled calls", p.Polled)
	}
	same := newSession()
	same.shell(show, 0, "{}")
	soon(same).shell(show, 0, "{}")
	soon(same).shell(show, 0, "{}")
	if p := profileOf(t, same.report(), "test-genie runs show"); p.Polled != 3 {
		t.Errorf("the same read repeated in the same shape is a poll: %d polled", p.Polled)
	}
	later := newSession()
	later.shell(show, 0, "{}")
	later.shell(show+" | jq '.run.status'", 0, "running")
	later.shell(show+" | jq '.run.phases[].name'", 0, "unit")
	if p := profileOf(t, later.report(), "test-genie runs show"); p.Polled != 3 {
		t.Errorf("reshapes a minute apart can see a new state: %d polled", p.Polled)
	}
}

// A run working on the scenario it reads (developing or reviewing it) does not
// reach around that scenario's CLI: its reads are counted apart and the
// bypass finding says how many it set aside.
func TestReadsByRunsWorkingOnTheScenarioAreNotBypasses(t *testing.T) {
	s := newSession()
	for run := 0; run < 5; run++ {
		s.run = fmt.Sprintf("run-%d", run)
		s.shell(`sqlite3 "file:scenarios/agent-manager/data/agent-manager.db?mode=ro" "select count(*) from runs"`, 0, "3")
		s.shell(`sqlite3 "file:scenarios/agent-manager/data/agent-manager.db?mode=ro" "select count(*) from run_events"`, 0, "9")
	}
	calls := s.calls()
	for i := range calls {
		calls[i].OwnWork = calls[i].RunID >= "run-3" // runs 3 and 4 develop agent-manager
	}
	r := Analyze(Window{Calls: calls})
	f := findingOf(t, r, KindBypass, "agent-manager")
	if f.Calls != 6 || f.Runs != 3 || r.Totals.BypassCalls != 6 {
		t.Errorf("bypass = %d calls in %d runs (window %d), want the 6 calls of the 3 runs not working on agent-manager", f.Calls, f.Runs, r.Totals.BypassCalls)
	}
	if !strings.Contains(f.Description, "(4 more in 2 runs working on agent-manager itself are not counted") {
		t.Errorf("description = %s", f.Description)
	}
	if r.Coverage.OwnWork != 4 || !strings.Contains(r.Coverage.Sentence(), "4 direct reads of a scenario's internals by runs working on that scenario are not counted as bypasses") {
		t.Errorf("coverage = %s", r.Coverage.Sentence())
	}
	for i := range calls {
		calls[i].OwnWork = true
	}
	for _, f := range Analyze(Window{Calls: calls}).Findings {
		if f.Kind == KindBypass {
			t.Errorf("own work only, yet a bypass: %s", f.Description)
		}
	}
	if !WorksOn(5, 10) || WorksOn(4, 10) || WorksOn(0, 0) {
		t.Errorf("a run works on a scenario when half or more of its tool calls naming a scenario tree name that one")
	}
}

// A verify command's report line is its verdict, whatever words the checked
// thing has: a passing vitest line naming "unavailable paging reasons" in a
// failing suite is not a tool error, so the exit is a verdict, neither a
// failure nor a cluster. A real tool error before the verdict stays a failure.
func TestAVerifyCommandsReportLineIsItsVerdict(t *testing.T) {
	s := newSession()
	for i := 0; i < 3; i++ {
		s.shell("vrooli scenario test agent-manager --phases unit", 1, "✓ src/features/memory/MemoryDetailView.test.tsx > preserves unavailable paging reasons without reporting a healthy pressure level 12ms\nFAIL src/features/usage/UsagePage.test.tsx")
		s.shell("vrooli scenario test agent-manager --phases unit", 1, "Error: start run: resource_exhausted: test-genie admission is saturated (caller preview capacity)")
	}
	r := s.report()
	p := profileOf(t, r, "vrooli scenario test")
	if p.Failed != 3 || p.Decided != 3 {
		t.Errorf("profile = %d failed of %d decided, want the 3 admission errors only (verdicts are neither)", p.Failed, p.Decided)
	}
	for _, cl := range r.Clusters {
		if strings.Contains(cl.Signature, "paging") {
			t.Errorf("a report line clustered as a failure: %+v", cl)
		}
	}
	// Facts projected before the rule stored the report line as the error:
	// the read applies the current rule to it.
	code := 1
	stored := Observe(Fact{InvocationFact: runsignal.InvocationFact{CommandPath: "vrooli scenario test", Executable: "vrooli", Ownership: runsignal.OwnershipResolved, Outcome: runsignal.OutcomeFailure, SemanticsKind: "verify", ExitCode: &code}, RunID: "r"},
		Features{ErrorLine: "✓ src<path> > MemoryDetailView > preserves unavailable paging reasons", Signature: "✓ src<path> > memorydetailview > preserves unavailable paging reasons"})
	if stored.Failed() || !stored.Verdict || stored.Signature != "" || len(clusters([]Call{stored, stored, stored})) != 0 {
		t.Errorf("a stored report line still fails or clusters: %+v", stored)
	}
}

// search-hub's summary promises notes on the degraded providers; when the
// output never prints them, that missing note is the defect to state.
func TestUnprintedProviderNotesAreStatedAsTheDefect(t *testing.T) {
	s := newSession()
	for run := 0; run < 3; run++ {
		s.run = fmt.Sprintf("run-%d", run)
		s.shell("search-hub query x --type skill", 0, "Summary:\n  Searched 3 corpora; 30 total hit(s) (some providers degraded — see notes below).\nResults by provider:\n  ▸ cli-health.commands (10)\nRetrieval Hints:\n  --all")
	}
	f := findingOf(t, s.report(), KindDegraded, "search-hub")
	if !strings.Contains(f.Description, "never printed the notes naming the degraded providers") || strings.Contains(f.Description, "not named in the output read") {
		t.Errorf("description = %s", f.Description)
	}
}
