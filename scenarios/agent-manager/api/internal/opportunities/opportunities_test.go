package opportunities

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/tokenaccounting"
	"agent-manager/internal/toolfriction"
)

var epoch = time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)

// session builds one run's calls in order. Each spec is "command line" with
// its owner and path taken from the first words the way the catalog resolves
// them: a leading `~` marks a tool outside Vrooli.
type session struct {
	id    string
	calls []toolfriction.Call
}

func newSession(id string) *session { return &session{id: id} }

func (s *session) vrooli(owner, path, command string) *session {
	return s.add(toolfriction.Call{Owner: owner, CommandPath: path, Features: toolfriction.Features{Command: command}})
}

func (s *session) raw(path, command string) *session {
	return s.add(toolfriction.Call{CommandPath: path, Features: toolfriction.Features{Command: command}})
}

func (s *session) add(c toolfriction.Call) *session {
	c.RunID, c.At, c.ResultTokens = s.id, epoch.Add(time.Duration(len(s.calls))*time.Minute), 100
	c.CallEventID = fmt.Sprintf("%s-%d", s.id, len(s.calls))
	s.calls = append(s.calls, c)
	return s
}

func (s *session) skill(name string, segment int64) *session {
	return s.add(toolfriction.Call{Owner: "prompt-manager", CommandPath: "prompt-manager skill read", SkillID: name, ResidencySegment: segment, Features: toolfriction.Features{Command: "prompt-manager skill read " + name}})
}

func (s *session) bypass(owner, method, command string) *session {
	return s.add(toolfriction.Call{CommandPath: "sqlite3", Features: toolfriction.Features{Command: command, BypassOwner: owner, BypassMethod: method}})
}

func (s *session) jq(path, command string, paths ...string) *session {
	return s.add(toolfriction.Call{Owner: strings.Fields(path)[0], CommandPath: path, Features: toolfriction.Features{Command: command, JSONPaths: paths}})
}

func flatten(sessions ...*session) []toolfriction.Call {
	var out []toolfriction.Call
	for _, s := range sessions {
		out = append(out, s.calls...)
	}
	return out
}

func repeat(n int, build func(i int) *session) []*session {
	out := make([]*session, n)
	for i := range out {
		out[i] = build(i)
	}
	return out
}

const orientationSubject = "search-hub query → prompt-manager skill read → vrooli-memory journal note"

func orientation(id string) *session {
	return newSession(id).
		vrooli("search-hub", "search-hub query", "search-hub query intent").
		raw("rg", "rg TODO").
		vrooli("prompt-manager", "prompt-manager skill read", "prompt-manager skill read idea-workshop").
		vrooli("vrooli-memory", "vrooli-memory journal note", "vrooli-memory journal note --kind work-record")
}

func testLoop(id string) *session {
	return newSession(id).
		vrooli("vrooli", "vrooli scenario test", "vrooli scenario test agent-manager --phases unit").
		vrooli("vrooli", "vrooli scenario test", "vrooli scenario test agent-manager --phases unit").
		vrooli("test-genie", "test-genie runs wait", "test-genie runs wait agent-manager run-1").
		vrooli("test-genie", "test-genie runs show", "test-genie runs show run-1").
		vrooli("vrooli-memory", "vrooli-memory journal note", "vrooli-memory journal note")
}

var testGenieLibrary = Library{Programs: []Program{
	{Name: "test-genie.iterate", Scenario: "test-genie", Purpose: "Validate a change: submit test phases and return one durable wait command", Bindings: []string{"test-genie/validation/create"}},
	{Name: "agent-manager.friction-digest", Scenario: "agent-manager", Purpose: "Digest friction", Bindings: []string{"agent-manager/usage/friction"}},
}}

func find(t *testing.T, r Report, kind, subject string) Opportunity {
	t.Helper()
	for _, o := range r.Opportunities {
		if o.Kind == kind && o.Subject == subject {
			return o
		}
	}
	t.Fatalf("no %s opportunity %q in %+v", kind, subject, subjects(r))
	return Opportunity{}
}

func subjects(r Report) []string {
	out := make([]string, len(r.Opportunities))
	for i, o := range r.Opportunities {
		out[i] = o.Kind + ":" + o.Subject
	}
	return out
}

// The orientation ritual is a skill ritual owned by prompt-manager and, with
// no program in the library covering it, a program candidate. A tool outside
// Vrooli between its steps does not break it.
func TestOrientationRitualIsAProgramCandidate(t *testing.T) {
	calls := flatten(repeat(4, func(i int) *session { return orientation(fmt.Sprintf("o%d", i)) })...)
	r := Analyze(Input{Calls: calls, Library: testGenieLibrary})
	o := find(t, r, KindSequence, orientationSubject)
	if o.Verdict != VerdictCandidate || o.Owner != "prompt-manager" || o.Sessions != 4 || o.Occurrences != 4 || o.Program != nil {
		t.Fatalf("orientation = %+v", o)
	}
	if o.CallsSaved != 4*(3-1) || len(o.RunIDs) != maxEvidenceRuns || len(o.Examples) == 0 || o.Verify == "" {
		t.Fatalf("orientation impact/evidence = %+v", o)
	}
}

// The test loop (repeated test submissions collapsed) is related to
// test-genie.iterate only by affinity (its scenario and a rare word): that is
// a lead to investigate, never "adopt". The program's own runs are counted
// from `program-runtime library run` calls of it. One program call per
// occurrence replaces at most the motif's other steps: 3 of its 4 steps, even
// where a step was repeated (5 calls an occurrence).
func TestTestLoopAffinityMatchIsInvestigateNotAdopt(t *testing.T) {
	sessions := repeat(3, func(i int) *session { return testLoop(fmt.Sprintf("t%d", i)) })
	sessions = append(sessions, newSession("p0").
		vrooli("program-runtime", "program-runtime library run", "program-runtime library run test-genie.iterate --scenario x").
		vrooli("program-runtime", "program-runtime library get", "program-runtime library get test-genie.iterate").
		vrooli("prompt-manager", "prompt-manager skill read", "prompt-manager skill read test-genie.iterate"))
	r := Analyze(Input{Calls: flatten(sessions...), Library: testGenieLibrary})
	o := find(t, r, KindSequence, "vrooli scenario test → test-genie runs wait → test-genie runs show → vrooli-memory journal note")
	if o.Verdict != VerdictInvestigate || o.Program == nil || o.Program.Name != "test-genie.iterate" || o.Program.Basis != BasisAffinity || !strings.HasPrefix(o.Action, "investigate") {
		t.Fatalf("test loop = %+v program=%+v", o, o.Program)
	}
	if o.Program.Runs != 1 || o.Program.Covered != 0 || o.Program.Steps != 4 || o.Occurrences != 3 || o.CallsSaved != 3*(4-1) {
		t.Fatalf("test loop counts = %+v program=%+v", o, o.Program)
	}
}

// A program calling exactly half of a motif's steps is a binding match but
// not a majority: investigate, not adopt.
func TestHalfBindingCoverageIsInvestigate(t *testing.T) {
	library := Library{Programs: []Program{{Name: "t.half", Scenario: "test-genie", Bindings: []string{"test-genie/runs/wait", "test-genie/runs/show"}}}}
	calls := flatten(repeat(3, func(i int) *session { return testLoop(fmt.Sprintf("t%d", i)) })...)
	o := find(t, Analyze(Input{Calls: calls, Library: library}), KindSequence, "vrooli scenario test → test-genie runs wait → test-genie runs show → vrooli-memory journal note")
	if o.Verdict != VerdictInvestigate || o.Program.Basis != BasisBinding || o.Program.Covered != 2 {
		t.Fatalf("half coverage = %+v program=%+v", o, o.Program)
	}
}

// Shifted and rotated n-grams of one behaviour merge into one maximal motif:
// a run doing A B C D E once is one occurrence of one sequence, and its calls
// are counted once, not once per 3- and 4-gram inside it.
func TestOverlappingGramsMergeIntoOneMaximalMotif(t *testing.T) {
	calls := flatten(repeat(3, func(i int) *session {
		s := newSession(fmt.Sprintf("m%d", i))
		for _, w := range []string{"a", "b", "c", "d", "e"} {
			s.vrooli(w, w+" x", w+" x")
		}
		return s
	})...)
	r := Analyze(Input{Calls: calls, Library: Library{}})
	got := subjects(r)
	if len(got) != 1 || got[0] != "sequence:a x → b x → c x → d x → e x" {
		t.Fatalf("sequences = %v, want one maximal motif", got)
	}
	if o := r.Opportunities[0]; o.Occurrences != 3 || o.CallsSaved != 3*(5-1) || o.Sessions != 3 {
		t.Fatalf("motif counts = %+v", o)
	}
}

// A program that calls a command in the sequence is a binding match, the
// strongest basis, and beats a program that merely shares an owner.
func TestBindingMatchBeatsOwnerMatch(t *testing.T) {
	library := Library{Programs: []Program{
		{Name: "a.owner-only", Scenario: "search-hub", Purpose: "query the hub"},
		{Name: "z.binds", Scenario: "prompt-manager", Bindings: []string{"search-hub/query", "vrooli-memory/journal/note"}},
	}}
	calls := flatten(repeat(3, func(i int) *session { return orientation(fmt.Sprintf("o%d", i)) })...)
	o := find(t, Analyze(Input{Calls: calls, Library: library}), KindSequence, orientationSubject)
	if o.Program == nil || o.Program.Name != "z.binds" || o.Program.Basis != "binding" {
		t.Fatalf("match = %+v", o.Program)
	}
}

// A program of a step's owner that does not name the step's verb is not a
// replacement: it is only the same owner's.
func TestSameOwnerProgramWithoutTheVerbIsNotAnAdoption(t *testing.T) {
	library := Library{Programs: []Program{{Name: "test-genie.reports", Scenario: "test-genie", Purpose: "Join fleet health rows"}}}
	calls := flatten(repeat(3, func(i int) *session { return testLoop(fmt.Sprintf("t%d", i)) })...)
	o := find(t, Analyze(Input{Calls: calls, Library: library}), KindSequence, "vrooli scenario test → test-genie runs wait → test-genie runs show → vrooli-memory journal note")
	if o.Verdict != VerdictCandidate {
		t.Fatalf("verdict = %q, want candidate", o.Verdict)
	}
}

// A program that shares one step of a sequence is related, not a replacement.
func TestProgramSharingOneStepDoesNotMakeAnAdoption(t *testing.T) {
	library := Library{Programs: []Program{{Name: "prepare", Scenario: "x", Purpose: "reads", Bindings: []string{"prompt-manager/skill/read"}}}}
	calls := flatten(repeat(3, func(i int) *session { return orientation(fmt.Sprintf("o%d", i)) })...)
	o := find(t, Analyze(Input{Calls: calls, Library: library}), KindSequence, orientationSubject)
	if o.Verdict != VerdictCandidate || o.Program != nil {
		t.Fatalf("verdict = %q program = %+v, want candidate", o.Verdict, o.Program)
	}
}

// A loop of three commands also shows as its rotations; the most widespread
// rotation stands for it, and a part of a longer sequence with nearly the same
// sessions is not listed again.
func TestRotationsAndContainedPartsCollapse(t *testing.T) {
	calls := flatten(repeat(5, func(i int) *session {
		s := newSession(fmt.Sprintf("r%d", i))
		for rep := 0; rep < 2; rep++ {
			s.vrooli("a", "a x", "a x").vrooli("b", "b x", "b x").vrooli("c", "c x", "c x")
		}
		return s
	})...)
	// A sixth session ends mid-loop, so only one rotation reaches six sessions.
	calls = append(calls, newSession("r9").vrooli("a", "a x", "a x").vrooli("b", "b x", "b x").vrooli("c", "c x", "c x").calls...)
	got := subjects(Analyze(Input{Calls: calls, Library: Library{}}))
	for _, s := range got {
		if s == "sequence:b x → c x → a x" || s == "sequence:c x → a x → b x" {
			t.Fatalf("rotation of a x → b x → c x listed: %v", got)
		}
	}
	if !slices.Contains(got, "sequence:a x → b x → c x") {
		t.Fatalf("sequences = %v, want the most widespread rotation", got)
	}
}

func TestSequencesNeedThreeSessionsAndCollapseOnlyRepeats(t *testing.T) {
	calls := flatten(repeat(2, func(i int) *session { return orientation(fmt.Sprintf("o%d", i)) })...)
	if r := Analyze(Input{Calls: calls, Library: testGenieLibrary}); len(r.Opportunities) != 0 {
		t.Fatalf("two sessions reported: %v", subjects(r))
	}
	// One session repeating A B C three times is one session, not three.
	one := newSession("loop")
	for i := 0; i < 3; i++ {
		one.vrooli("a", "a x", "a x").vrooli("b", "b x", "b x").vrooli("c", "c x", "c x")
	}
	if r := Analyze(Input{Calls: one.calls, Library: testGenieLibrary}); len(r.Opportunities) != 0 {
		t.Fatalf("one session reported: %v", subjects(r))
	}
}

// A 3-gram inside a 4-gram that holds the same sessions is part of it.
func TestShorterSequenceInsideLongerWithSameSessionsIsNotReportedTwice(t *testing.T) {
	calls := flatten(repeat(3, func(i int) *session {
		return newSession(fmt.Sprintf("s%d", i)).vrooli("a", "a x", "a x").vrooli("b", "b x", "b x").vrooli("c", "c x", "c x").vrooli("d", "d x", "d x")
	})...)
	r := Analyze(Input{Calls: calls, Library: Library{}})
	got := subjects(r)
	if len(got) != 1 || got[0] != "sequence:a x → b x → c x → d x" {
		t.Fatalf("sequences = %v, want only the 4-gram", got)
	}
}

func TestUnreadableLibraryLeavesSequencesUnmatched(t *testing.T) {
	calls := flatten(repeat(3, func(i int) *session { return orientation(fmt.Sprintf("o%d", i)) })...)
	r := Analyze(Input{Calls: calls, Library: Library{Unavailable: "program-runtime unreachable"}})
	o := find(t, r, KindSequence, orientationSubject)
	if o.Verdict != VerdictUnknown || o.Program != nil {
		t.Fatalf("verdict = %q program = %+v, want unknown", o.Verdict, o.Program)
	}
	if !strings.Contains(r.Coverage.Sentence(), "program-runtime unreachable") {
		t.Fatalf("coverage = %s", r.Coverage.Sentence())
	}
}

// Bypasses cluster by owner and method and name the tables, endpoints and
// state directories reached, with ids folded.
func TestBypassClustersNameWhatAgentsReach(t *testing.T) {
	sessions := repeat(4, func(i int) *session {
		id := fmt.Sprintf("b%d", i)
		return newSession(id).
			bypass("agent-manager", toolfriction.BypassDatabase, `sqlite3 "file:scenarios/agent-manager/data/agent-manager.db?mode=ro" "select count(*) from run_events join runs on 1"`).
			bypass("agent-manager", toolfriction.BypassHTTP, fmt.Sprintf("curl localhost:17364/api/v1/runs/0d5f3c2e-1111-2222-3333-44445555666%d/events", i)).
			bypass("storage-manager", toolfriction.BypassDatabase, `sqlite3 scenarios/storage-manager/data/storage-manager.db "select * from reclaim_jobs"`)
	})
	sessions = append(sessions, newSession("lone").bypass("web-console", toolfriction.BypassState, "ls scenarios/web-console/data/state"))
	r := Analyze(Input{Calls: flatten(sessions...)})
	db := find(t, r, KindBypass, "agent-manager via "+toolfriction.BypassDatabase)
	if db.Occurrences != 4 || db.Sessions != 4 || db.Owner != "agent-manager" || len(db.Details) != 2 || db.Details[0] != "table run_events ×4" {
		t.Fatalf("db bypass = %+v", db)
	}
	if got := find(t, r, KindBypass, "agent-manager via "+toolfriction.BypassHTTP).Details; len(got) != 1 || got[0] != "/api/v1/runs ×4" {
		t.Fatalf("http targets = %v", got)
	}
	find(t, r, KindBypass, "storage-manager via "+toolfriction.BypassDatabase)
	for _, o := range r.Opportunities {
		if strings.HasPrefix(o.Subject, "web-console") {
			t.Fatalf("one-session bypass reported: %+v", o)
		}
	}
}

func TestOutputShapeNamesTheFieldsAgentsExtract(t *testing.T) {
	calls := flatten(repeat(3, func(i int) *session {
		return newSession(fmt.Sprintf("j%d", i)).
			jq("vrooli scenario status", "vrooli scenario status x --json", ".name", ".ports[].url").
			jq("vrooli scenario status", "vrooli scenario status y --json", ".name")
	})...)
	calls = append(calls, newSession("j9").jq("test-genie runs list", "test-genie runs list --json", ".runs[].id").calls...)
	r := Analyze(Input{Calls: calls})
	o := find(t, r, KindOutputShape, "vrooli scenario status")
	if o.Owner != "vrooli" || o.Occurrences != 6 || o.Sessions != 3 || len(o.Details) != 2 || o.Details[0] != ".name ×6" {
		t.Fatalf("shape = %+v", o)
	}
	for _, other := range r.Opportunities {
		if other.Subject == "test-genie runs list" {
			t.Fatalf("one-session shape reported: %+v", other)
		}
	}
}

// Loads of a skill within one compaction segment after the first are extra; a
// load after a compaction is not. A skill is reported when extras recur in
// three sessions.
func TestSkillReloadsCountOnlyLoadsWithinOneSegment(t *testing.T) {
	sessions := repeat(3, func(i int) *session {
		return newSession(fmt.Sprintf("k%d", i)).skill("adaptive-research", 0).skill("adaptive-research", 0).skill("adaptive-research", 0).skill("adaptive-research", 1).skill("report-bug", 0)
	})
	sessions = append(sessions, newSession("k9").skill("report-bug", 0).skill("report-bug", 0))
	r := Analyze(Input{Calls: flatten(sessions...)})
	o := find(t, r, KindSkillReload, "adaptive-research")
	if o.Owner != "prompt-manager" || o.Occurrences != 6 || o.CallsSaved != 6 || o.TokensSaved != 600 || o.Sessions != 3 {
		t.Fatalf("skill = %+v", o)
	}
	for _, other := range r.Opportunities {
		if other.Subject == "report-bug" {
			t.Fatalf("one-session reload reported: %+v", other)
		}
	}
}

func TestRankedByCallsSavedAndEmptyReportsCoverage(t *testing.T) {
	calls := flatten(append(
		repeat(3, func(i int) *session { return orientation(fmt.Sprintf("o%d", i)) }),
		repeat(5, func(i int) *session {
			return newSession(fmt.Sprintf("b%d", i)).bypass("agent-manager", toolfriction.BypassDatabase, "sqlite3 x.db 'select 1 from runs'")
		})...)...)
	r := Analyze(Input{Calls: calls, Library: Library{Programs: []Program{{Name: "p", Scenario: "x"}}}})
	for i := 1; i < len(r.Opportunities); i++ {
		if r.Opportunities[i-1].CallsSaved < r.Opportunities[i].CallsSaved {
			t.Fatalf("not ranked: %v", subjects(r))
		}
	}
	empty := Analyze(Input{})
	if len(empty.Opportunities) != 0 || !strings.Contains(empty.Coverage.Sentence(), "cannot tell") {
		t.Fatalf("empty = %+v / %s", empty, empty.Coverage.Sentence())
	}
	if s := r.Coverage.Sentence(); !strings.Contains(s, "1 library programs") || !strings.Contains(s, "8 runs") {
		t.Fatalf("coverage = %s", s)
	}
}

// A loop of the shape in one stretch is several occurrences: periods are
// counted, and the calls saved are the stretch's calls less one per period, so
// the figure grows with the work replaced and not with the stretch count.
func TestLoopingSequenceCountsPeriodsNotStretches(t *testing.T) {
	const periods = 5
	calls := flatten(repeat(3, func(i int) *session {
		s := newSession(fmt.Sprintf("l%d", i))
		for p := 0; p < periods; p++ {
			s.vrooli("a", "a x", "a x").vrooli("b", "b x", "b x").vrooli("c", "c x", "c x")
		}
		return s
	})...)
	o := find(t, Analyze(Input{Calls: calls, Library: Library{}}), KindSequence, "a x → b x → c x")
	if o.Occurrences != 3*periods || o.CallsSaved != 3*periods*(3-1) {
		t.Fatalf("loop occurrences %d, calls saved %d; want %d periods and %d calls saved", o.Occurrences, o.CallsSaved, 3*periods, 3*periods*2)
	}
}

// The orientation ritual is mandated: a program saves round-trips and result
// tokens, and the action says so rather than claiming the work disappears.
func TestMandatedRitualStatesRoundTripsNotWork(t *testing.T) {
	calls := flatten(repeat(3, func(i int) *session { return orientation(fmt.Sprintf("o%d", i)) })...)
	r := Analyze(Input{Calls: calls, Library: testGenieLibrary})
	o := find(t, r, KindSequence, orientationSubject)
	if !o.Mandated || !strings.Contains(o.Action, "round-trips") || !strings.Contains(o.Action, "not the work") {
		t.Fatalf("mandated ritual = %+v", o)
	}
	loop := find(t, Analyze(Input{Calls: flatten(repeat(3, func(i int) *session { return testLoop(fmt.Sprintf("t%d", i)) })...), Library: Library{}}), KindSequence, "vrooli scenario test → test-genie runs wait → test-genie runs show → vrooli-memory journal note")
	if loop.Mandated {
		t.Fatalf("a test loop is not a mandated ritual: %+v", loop)
	}
}

// A bypass made mostly by one run's loop counts that run at most
// RunCapFactor times the median run and reports the run's share.
func TestBypassCountsOneLoopingRunAtMostTheCap(t *testing.T) {
	var calls []toolfriction.Call
	for i, n := range []int{2, 2, 2, 200} {
		for j := 0; j < n; j++ {
			calls = append(calls, toolfriction.Call{RunID: fmt.Sprintf("r%d", i), CommandPath: "sqlite3", Features: toolfriction.Features{Command: "sqlite3 x.db", BypassOwner: "test-genie", BypassMethod: "database"}})
		}
	}
	got := bypasses(calls)
	if len(got) != 1 || got[0].Occurrences != 206 || got[0].CallsSaved != 3*2+toolfriction.RunCapFactor*2 || got[0].TopRunShare < 0.9 {
		t.Fatalf("bypass = %+v, want 206 occurrences counted as %d calls with the looping run's share", got, 6+toolfriction.RunCapFactor*2)
	}
}

// A motif's savings count only the calls of its own steps: when the most
// frequent shape is the ritual, other Vrooli commands that some of its merged
// stretches run in between are not credited to it, and one program call per
// occurrence saves at most the subject's other steps.
func TestRitualSavesOnlyItsOwnStepsCalls(t *testing.T) {
	plain := repeat(4, func(i int) *session {
		return newSession(fmt.Sprintf("o%d", i)).
			vrooli("search-hub", "search-hub query", "search-hub query intent").
			vrooli("prompt-manager", "prompt-manager skill read", "prompt-manager skill read idea-workshop").
			vrooli("vrooli-memory", "vrooli-memory journal note", "vrooli-memory journal note")
	})
	extended := repeat(3, func(i int) *session {
		return newSession(fmt.Sprintf("x%d", i)).
			vrooli("search-hub", "search-hub query", "search-hub query intent").
			vrooli("prompt-manager", "prompt-manager skill read", "prompt-manager skill read idea-workshop").
			vrooli("vrooli-memory", "vrooli-memory journal note", "vrooli-memory journal note").
			vrooli("vrooli", "vrooli scenario status", "vrooli scenario status x").
			vrooli("vrooli", "vrooli scenario logs", "vrooli scenario logs x")
	})
	r := Analyze(Input{Calls: flatten(append(plain, extended...)...), Library: testGenieLibrary})
	o := find(t, r, KindSequence, orientationSubject)
	if o.Occurrences != 7 || o.CallsSaved != 7*(3-1) || o.TokensSaved != 7*3*100 {
		t.Fatalf("ritual = %d occurrences, %d calls saved, %d tokens; want 7, 14 (its own 21 calls less one program call each), 2100", o.Occurrences, o.CallsSaved, o.TokensSaved)
	}
	if !o.Mandated {
		t.Errorf("the ritual of mandated commands is not marked mandated: %+v", o)
	}
}

// A loop is a turn the stretch repeats in full at least once: A B C A only
// starts over and is one occurrence of itself, A B C A B C is two of A B C.
func TestPeriodNeedsAFullRepeat(t *testing.T) {
	steps := func(paths ...string) []step {
		out := make([]step, len(paths))
		for i, p := range paths {
			out[i] = step{path: p}
		}
		return out
	}
	if got := period(steps("a", "b", "c", "a")); len(got) != 4 {
		t.Errorf("A B C A period = %d steps, want the stretch itself", len(got))
	}
	if got := period(steps("a", "b", "c", "a", "b", "c", "a")); len(got) != 3 {
		t.Errorf("A B C A B C A period = %d steps, want 3", len(got))
	}
	long := make([]string, 0, 18)
	for i := 0; i < 2; i++ {
		for j := 0; j < maxMotifStep+1; j++ {
			long = append(long, fmt.Sprintf("s%d", j))
		}
	}
	if got := period(steps(long...)); len(got) > maxMotifStep {
		t.Errorf("a turn longer than maxMotifStep (%d) is not a motif period: %d", maxMotifStep, len(got))
	}
}

// Kinds limits what is mined: the recommendations read sequences and bypasses
// only, and do not pay for output shapes and skill reloads they would drop.
func TestKindsLimitWhatIsMined(t *testing.T) {
	var sessions []*session
	for i := 0; i < 3; i++ {
		s := orientation(fmt.Sprintf("o%d", i))
		s.skill("idea-workshop", 0).skill("idea-workshop", 0)
		s.jq("vrooli scenario status", "vrooli scenario status x --json | jq .state", ".state")
		sessions = append(sessions, s)
	}
	all := Analyze(Input{Calls: flatten(sessions...), Library: testGenieLibrary})
	only := Analyze(Input{Calls: flatten(sessions...), Library: testGenieLibrary, Kinds: []string{KindSequence, KindBypass}})
	kinds := func(r Report) map[string]bool {
		out := map[string]bool{}
		for _, o := range r.Opportunities {
			out[o.Kind] = true
		}
		return out
	}
	if got := kinds(all); !got[KindSequence] || !got[KindSkillReload] || !got[KindOutputShape] {
		t.Fatalf("all kinds = %v", got)
	}
	if got := kinds(only); !got[KindSequence] || got[KindSkillReload] || got[KindOutputShape] {
		t.Errorf("sequences and bypasses only = %v", got)
	}
}

// The coverage line says what awaits re-derivation at an older fact version.
func TestCoverageStatesStaleRuns(t *testing.T) {
	r := Analyze(Input{Calls: flatten(orientation("o0")), Stale: toolfriction.Stale{Runs: 281, Calls: 165_120}})
	if got := r.Coverage.Sentence(); !strings.Contains(got, "281 runs (165120 calls) awaiting re-derivation; findings exclude them") {
		t.Errorf("coverage = %q", got)
	}
}

// Every kind counts its calls saved by the one per-run rule
// (toolfriction.Capped): one run's loop of jq extractions or skill reloads
// counts at most ten times the median run, and states its share. Every kind
// states one basis for what a saved call is, with the cap when it cut the
// count, so occurrences and calls saved never differ unexplained.
func TestEveryKindCapsOneLoopingRun(t *testing.T) {
	loop := newSession("loop")
	for i := 0; i < 60; i++ {
		loop.jq("test-genie runs show", "test-genie runs show r --json | jq .status", ".status")
		loop.skill("idea-workshop", 0)
	}
	quiet := repeat(3, func(i int) *session {
		s := newSession(fmt.Sprintf("s%d", i))
		return s.jq("test-genie runs show", "test-genie runs show r --json | jq .status", ".status").skill("idea-workshop", 0).skill("idea-workshop", 0)
	})
	report := Analyze(Input{Calls: flatten(append(quiet, loop)...), Kinds: []string{KindOutputShape, KindSkillReload}})
	for _, o := range report.Opportunities {
		// 60 (or 59 reloads) in one run against one in each of three others:
		// the loop counts at most 10x the median run's one.
		if o.CallsSaved != 13 || o.TopRunShare < 0.9 {
			t.Errorf("%s %s: %d calls saved, top run %.2f; want 10+1+1+1 = 13", o.Kind, o.Subject, o.CallsSaved, o.TopRunShare)
		}
		if want := savedBy[o.Kind].basis + fmt.Sprintf("; 13 of %d counted, each session at most 10x the median session's", o.Occurrences); o.SavedBasis != want || o.SavedLabel != savedBy[o.Kind].label {
			t.Errorf("%s %s basis = %q, want %q", o.Kind, o.SavedLabel, o.SavedBasis, want)
		}
	}
	// Uncapped, the basis is the kind's alone.
	for _, o := range Analyze(Input{Calls: flatten(quiet...), Kinds: []string{KindOutputShape, KindSkillReload}}).Opportunities {
		if o.SavedBasis != savedBy[o.Kind].basis || o.SavedBasis == "" {
			t.Errorf("%s uncapped basis = %q", o.Kind, o.SavedBasis)
		}
	}
	if len(report.Opportunities) != 2 {
		t.Fatalf("opportunities = %+v", report.Opportunities)
	}
}

// What a kind's number counts is named for what the change does: only a
// sequence removes calls; a bypass replaces them, a focused flag removes the
// jq pipeline and leaves the call.
func TestEveryKindNamesWhatItsNumberCounts(t *testing.T) {
	for kind, want := range map[string]string{KindSequence: "calls saved", KindBypass: "calls replaced", KindOutputShape: "extractions removed", KindSkillReload: "reads saved"} {
		if got := savedBy[kind]; got.label != want || got.basis == "" {
			t.Errorf("%s = %+v, want label %q and a basis", kind, got, want)
		}
	}
}

// The mandated ritual is the commands AGENTS.md makes every agent run: each
// prefix the rule reads is a command AGENTS.md names, so a change to the
// ritual there fails here instead of silently mislabelling sequences.
func TestMandatedRitualIsWhatAgentsMDNames(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, prefix := range mandatedRitual {
		if !strings.Contains(string(body), prefix) {
			t.Errorf("AGENTS.md no longer names %q: update mandatedRitual", prefix)
		}
	}
}

// A skill read twice in a run an older transcript parser read (no compaction
// events, every load in segment 0) is a repeat with unknown residency: never
// "still in context", and it saves nothing.
func TestSkillReloadsWithoutSegmentsClaimNoResidency(t *testing.T) {
	sessions := repeat(3, func(i int) *session {
		return newSession(fmt.Sprintf("s%d", i)).skill("big", 0).skill("big", 0)
	})
	calls := flatten(sessions...)
	for i := range calls {
		calls[i].OlderParser = true
	}
	got := skillReloads(calls)
	if len(got) != 1 || got[0].CallsSaved != 0 || got[0].TokensSaved != 0 || strings.Contains(got[0].Action, "still in context") || !strings.Contains(got[0].Action, "residency unknown") {
		t.Fatalf("reloads = %+v", got)
	}
}

// Merged grams need not occur together: runs that share only part of a
// motif (search → skill → library read, without the journal note ending it) are not
// its occurrences or its evidence. The motif shown is the one most runs ran in
// full, and only those runs count.
func TestAMotifCountsOnlyRunsThatRanItInFull(t *testing.T) {
	var sessions []*session
	for i := 0; i < 7; i++ {
		s := newSession(fmt.Sprintf("r%d", i))
		s.vrooli("search-hub", "search-hub query", "search-hub query x").
			vrooli("prompt-manager", "prompt-manager skill read", "prompt-manager skill read idea-workshop")
		if i < 5 {
			s.vrooli("vrooli-memory", "vrooli-memory journal note", "vrooli-memory journal note")
		}
		if i >= 4 {
			s.vrooli("program-runtime", "program-runtime library read", "program-runtime library read x")
		}
		sessions = append(sessions, s)
	}
	r := Analyze(Input{Calls: flatten(sessions...), Library: Library{}})
	o := find(t, r, KindSequence, orientationSubject)
	if o.Occurrences != 5 || o.Sessions != 5 || o.CallsSaved != 5*2 || slices.ContainsFunc(o.RunIDs, func(id string) bool { return id == "r5" || id == "r6" }) {
		t.Fatalf("motif = %+v", o)
	}
}

// An affinity match needs the program to name the commands of at least two
// steps: one shared word ("list" in "a request list") is not a link, so no
// program is offered for a sequence it calls none of.
func TestOneSharedWordIsNoAffinity(t *testing.T) {
	library := Library{Programs: []Program{{Name: "program-runtime.watch-set", Scenario: "program-runtime", Purpose: "Start up to eight delegated Agent Manager runs from a request list, collect each once, classify the collected outputs with one governed batch call, and summarize by label.", Bindings: []string{"ai-gateway/inference/run-batch"}}}}
	calls := flatten(repeat(3, func(i int) *session {
		return newSession(fmt.Sprintf("p%d", i)).
			vrooli("prompt-manager", "prompt-manager skill topology", "prompt-manager skill topology x").
			vrooli("program-runtime", "program-runtime bindings list", "program-runtime bindings list").
			vrooli("program-runtime", "program-runtime library read", "program-runtime library read x")
	})...)
	o := find(t, Analyze(Input{Calls: calls, Library: library}), KindSequence, "prompt-manager skill topology → program-runtime bindings list → program-runtime library read")
	if o.Program != nil || o.Verdict != VerdictCandidate {
		t.Fatalf("watch-set offered for a sequence it calls none of: %+v program=%+v", o, o.Program)
	}
}

// "Still in context" is claimed only over the runs whose segments are known;
// runs with only residency-unknown repeats are counted in their own clause.
func TestSkillReloadSessionsAreCountedPerClaim(t *testing.T) {
	calls := flatten(repeat(5, func(i int) *session {
		return newSession(fmt.Sprintf("s%d", i)).skill("big", 0).skill("big", 0)
	})...)
	for i := range calls {
		calls[i].OlderParser = calls[i].RunID >= "s2"
	}
	got := skillReloads(calls)
	if len(got) != 1 || !strings.Contains(got[0].Action, "read again 2 times while still in context in 2 sessions") || !strings.HasSuffix(got[0].Action, "; 3 repeats in 3 sessions whose facts record no compactions (residency unknown: a repeat may follow one)") || got[0].Sessions != 5 {
		t.Fatalf("reloads = %+v", got)
	}
}

// A skill read whose output went through a filter never put the skill in
// context: it is neither the load nor a reload. A reload's count states how
// many of its reloads the per-run cap counts as calls saved.
func TestFilteredSkillReadsAreNotLoads(t *testing.T) {
	filtered := func(s *session) *session {
		s.skill("research-coordination", 0)
		s.calls[len(s.calls)-1].Filtered = true
		s.calls[len(s.calls)-1].Command += " | rg phase"
		return s
	}
	sessions := repeat(3, func(i int) *session {
		s := newSession(fmt.Sprintf("f%d", i)).skill("research-coordination", 0)
		return filtered(s).skill("research-coordination", 0)
	})
	sessions = append(sessions, repeat(3, func(i int) *session {
		return filtered(filtered(newSession(fmt.Sprintf("g%d", i))).skill("adaptive-research", 0))
	})...)
	r := Analyze(Input{Calls: flatten(sessions...), Kinds: []string{KindSkillReload}})
	o := find(t, r, KindSkillReload, "research-coordination")
	if o.Occurrences != 3 || o.CallsSaved != 3 || o.TokensSaved != 300 || !strings.Contains(o.Action, "read again 3 times") {
		t.Fatalf("reloads = %+v, want one unfiltered reload in each of three sessions", o)
	}
	if len(r.Opportunities) != 1 {
		t.Fatalf("a filtered read counted as a load: %v", subjects(r))
	}
	loop := newSession("loop")
	for i := 0; i < 30; i++ {
		loop.skill("research-coordination", 0)
	}
	capped := find(t, Analyze(Input{Calls: flatten(append(sessions, loop)...), Kinds: []string{KindSkillReload}}), KindSkillReload, "research-coordination")
	if capped.CallsSaved != 13 || !strings.Contains(capped.Action, "read again 32 times") || !strings.Contains(capped.SavedBasis, "13 of 32 counted, each session at most 10x the median session's") {
		t.Fatalf("capped reloads = %d saved: %s", capped.CallsSaved, capped.Action)
	}
}

// A help lookup or a bare executable is not a step of the work: a motif is
// made of the commands around them.
func TestSequencesSkipHelpLookupsAndBareExecutables(t *testing.T) {
	sessions := repeat(3, func(i int) *session {
		s := newSession(fmt.Sprintf("h%d", i)).
			vrooli("test-genie", "test-genie runs", "test-genie runs --help").
			vrooli("vrooli", "vrooli scenario test", "vrooli scenario test x").
			vrooli("prompt-manager", "prompt-manager", "prompt-manager").
			vrooli("program-runtime", "program-runtime library run", "program-runtime library run p").
			vrooli("test-genie", "test-genie runs show", "test-genie runs show r")
		s.calls[0].Help = true
		return s
	})
	r := Analyze(Input{Calls: flatten(sessions...), Library: Library{}, Kinds: []string{KindSequence}})
	find(t, r, KindSequence, "vrooli scenario test → program-runtime library run → test-genie runs show")
	for _, o := range r.Opportunities {
		if strings.Contains(o.Subject, "--help") || strings.Contains(o.Subject, "test-genie runs →") || strings.Contains(o.Subject, "→ prompt-manager →") {
			t.Errorf("a help lookup or bare executable is a step: %s", o.Subject)
		}
	}
}

// Evidence runs are the runs that show a pattern most, not the first runs
// read: otherwise every pattern names the same sessions.
func TestEvidenceRunsShowThePatternMost(t *testing.T) {
	sessions := repeat(4, func(i int) *session {
		return newSession(fmt.Sprintf("e%d", i)).bypass("storage-manager", toolfriction.BypassDatabase, "sqlite3 storage.db 'select 1 from jobs'")
	})
	for i := 0; i < 4; i++ {
		sessions[3].bypass("storage-manager", toolfriction.BypassDatabase, "sqlite3 storage.db 'select 1 from jobs'")
	}
	for i := 0; i < 2; i++ {
		sessions[2].bypass("storage-manager", toolfriction.BypassDatabase, "sqlite3 storage.db 'select 1 from jobs'")
	}
	o := find(t, Analyze(Input{Calls: flatten(sessions...)}), KindBypass, "storage-manager via "+toolfriction.BypassDatabase)
	if strings.Join(o.RunIDs, ",") != "e3,e2,e0" {
		t.Fatalf("evidence runs = %v, want the runs with most calls first", o.RunIDs)
	}
}

// The opportunities and the friction profiles count the same commands for one
// window: file-read tool calls are read for bypasses only by both.
func TestCoverageCountsTheCommandsFrictionCounts(t *testing.T) {
	calls := flatten(orientation("c1"), orientation("c2"))
	calls = append(calls, toolfriction.Call{RunID: "c1", At: epoch, Features: toolfriction.Features{BypassOwner: "web-console", BypassMethod: toolfriction.BypassState}})
	got, want := Analyze(Input{Calls: calls}).Coverage.Calls, toolfriction.Analyze(toolfriction.Window{Calls: calls}).Coverage.Commands
	if got != want || got != 8 {
		t.Fatalf("opportunities count %d commands, friction %d; want both 8", got, want)
	}
}

// Output text that could not be read leaves a sequence unmeasured and says
// why, never quietly.
func TestUnreadableOutputsSayWhy(t *testing.T) {
	runs, _ := lookupRuns(6, true)
	failing := func([]string) (map[string]string, error) { return nil, fmt.Errorf("database is locked") }
	o := find(t, Analyze(Input{Calls: flatten(runs...), Library: Library{}, Outputs: failing}), KindSequence, lookupSubject)
	if o.Verdict != VerdictUnmeasured || o.CallsSaved != 0 || !strings.Contains(o.Action, "could not be read (database is locked)") {
		t.Fatalf("unreadable = %+v", o)
	}
}

// A run working on the scenario whose database it reads is not reaching
// around that scenario's CLI; the action says how many reads it set aside.
func TestBypassesLeaveOutRunsWorkingOnTheScenario(t *testing.T) {
	sessions := repeat(5, func(i int) *session {
		return newSession(fmt.Sprintf("w%d", i)).bypass("agent-manager", toolfriction.BypassDatabase, `sqlite3 data/agent-manager.db "select 1 from runs"`)
	})
	calls := flatten(sessions...)
	calls[3].OwnWork, calls[4].OwnWork = true, true
	o := find(t, Analyze(Input{Calls: calls}), KindBypass, "agent-manager via "+toolfriction.BypassDatabase)
	if o.Occurrences != 3 || o.Sessions != 3 || !strings.Contains(o.Action, "(2 more in 2 runs working on agent-manager itself are not counted") {
		t.Fatalf("bypass = %+v", o)
	}
}

// A reload whose output sits on another segment of its command (v32: `skill
// read x; plans list`) is a load with unknown tokens: counted, its tokens
// stated as unknown, never read as zero.
func TestSkillReloadsCountUnknownTokensApart(t *testing.T) {
	sessions := repeat(3, func(i int) *session {
		s := newSession(fmt.Sprintf("u%d", i)).skill("report-bug", 0).skill("report-bug", 0)
		s.calls[1].ResultTokens, s.calls[1].TokenBasis = 0, tokenaccounting.BasisUnknown
		return s
	})
	o := find(t, Analyze(Input{Calls: flatten(sessions...), Kinds: []string{KindSkillReload}}), KindSkillReload, "report-bug")
	if o.Occurrences != 3 || o.TokensUnknown != 3 || o.TokensSaved != 0 || !strings.Contains(o.Action, "(0 result tokens; tokens unknown for 3 of them)") {
		t.Fatalf("reloads = %+v", o)
	}
}
