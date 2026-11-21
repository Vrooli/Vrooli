// Package opportunities turns what agents repeatedly do by hand into records
// their owners can act on: command sequences that a program of the library
// already does or could do, direct reads of a scenario's internals that a CLI
// or RPC should serve, JSON fields agents cut out of a command's output that a
// focused flag should return, and skills loaded again while still in context.
// It is a pure aggregation over the stored tool calls and the program library:
// no I/O, no clock, no event payloads. Opportunities are records for the
// owning scenario, never built automatically.
package opportunities

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/toolfriction"
)

// Kinds of opportunity.
const (
	KindSequence    = "sequence"     // a repeated command sequence: adopt a program or write one
	KindBypass      = "bypass"       // agents reach around a CLI to a scenario's internals
	KindOutputShape = "output-shape" // agents cut the same fields out of a command's JSON
	KindSkillReload = "skill-reload" // a skill is read again while still in context
)

// Sequence verdicts: what the program library says about a repeated sequence.
const (
	VerdictAdopt       = "adopt"       // a program calls the commands of most of its steps
	VerdictInvestigate = "investigate" // a program is related (affinity or a minority of steps): check before adopting
	VerdictCandidate   = "candidate"   // none is
	VerdictUnknown     = "unknown"     // the library could not be read
	// VerdictIndependent: agents run the steps together but choose each later
	// step themselves, so a program cannot; VerdictUnmeasured: the output text
	// that shows whether they do was not available. Neither claims savings.
	VerdictIndependent = "independent"
	VerdictUnmeasured  = "unmeasured"
	// VerdictUndetermined: the measured rate's 95% interval spans the
	// threshold, so the sample cannot say. No savings are claimed.
	VerdictUndetermined = "undetermined"
)

const (
	// MinSessions is how many distinct sessions (runs) must show a pattern
	// before it is reported: one agent's habit is not a capability gap. It is
	// the tool friction's bypass rule too.
	MinSessions = toolfriction.MinPatternRuns
	// MaxOpportunities bounds a report after ranking.
	MaxOpportunities = 100
	maxEvidenceRuns  = 3
	maxExamples      = 2
	maxDetails       = 6
)

// Input is one window's tool calls at the current fact version (in run and
// time order, as the read model returns them), what the window holds at older
// versions, and the program library to match sequences against. Kinds names
// the kinds to mine; nil mines all four.
type Input struct {
	Calls    []toolfriction.Call
	Stale    toolfriction.Stale
	Library  Library
	From, To time.Time
	Kinds    []string
	// Outputs reads result text to measure whether a sequence's later steps
	// used earlier output; nil leaves that unmeasurable (VerdictUnmeasured).
	Outputs Outputs
}

// Opportunity is one actionable record for an owner.
type Opportunity struct {
	Kind string
	// Owner is the scenario that can act: the program's or CLI's owner, and
	// prompt-manager for skill rituals and skill loads.
	Owner string
	// Subject identifies the pattern: the sequence, "owner via method", the
	// command path, or the skill.
	Subject string
	// Action is the plain recommendation.
	Action string
	// Verdict and Program are set for sequences only.
	Verdict string
	Program *ProgramMatch
	// MeasuredSteps are the sampled later steps of a sequence that have
	// arguments chosen per occurrence and readable earlier output, and
	// DependentSteps those that took an argument from it.
	MeasuredSteps, DependentSteps int
	// Sessions are the distinct runs showing it, Occurrences how often.
	Sessions, Occurrences int
	// CallsSaved are the calls the change removes or replaces by a supported
	// one; TokensSaved the result tokens those calls put in context (an upper
	// bound for sequences; zero where the output is not what costs), and
	// TokensUnknown how many of those calls have unknown result tokens, which
	// TokensSaved leaves out.
	CallsSaved    int
	TokensSaved   int64
	TokensUnknown int
	// SavedLabel names what CallsSaved counts for the kind and SavedBasis says
	// what one of them is (savedBy) and, when the per-run cap cut the count,
	// how many of how many were counted.
	SavedLabel, SavedBasis string
	// TopRunShare is the largest single run's share of the uncapped calls
	// saved; near 1 means it is one run's loop. CallsSaved counts each run at
	// most toolfriction.RunCapFactor times the median run (toolfriction.Capped),
	// for every kind.
	TopRunShare float64
	// Mandated marks a sequence made only of commands AGENTS.md requires: a
	// program saves its round-trips and tokens, not the work.
	Mandated bool
	// Details are what the pattern touches: tables, endpoints, JSON paths.
	Details []string
	// RunIDs and Examples are drill-down evidence; examples are redacted
	// command lines.
	RunIDs   []string
	Examples []string
	// Verify names the check that shows the change worked.
	Verify string
}

// Coverage is what the analysis could see, so an empty list reads as bounded
// silence rather than absence. Calls are command calls (file-read tool calls
// are read for bypasses only), the count the friction coverage states.
type Coverage struct {
	Calls, VrooliCalls, Runs int
	Stale                    toolfriction.Stale
	From, To                 time.Time
	Library                  Library
}

// Sentence says what was read and what could not be.
func (c Coverage) Sentence() string {
	stale := ""
	if sentence := c.Stale.Sentence(); sentence != "" {
		stale = "; " + sentence
	}
	if c.Calls == 0 {
		return "no command calls at the current fact version in the window: cannot tell" + stale
	}
	library := fmt.Sprintf("%d library programs", len(c.Library.Programs))
	if c.Library.Unavailable != "" {
		library = "program library unavailable (" + c.Library.Unavailable + "), sequences cannot be matched"
	}
	return fmt.Sprintf("%d command calls (%d Vrooli CLI) in %d runs at the current fact version; a pattern needs %d runs; %s%s", c.Calls, c.VrooliCalls, c.Runs, MinSessions, library, stale)
}

// Report is the ranked opportunities and what they were drawn from.
type Report struct {
	Opportunities []Opportunity
	Coverage      Coverage
}

// Analyze mines the kinds in.Kinds names (all four when nil) from one
// window, ranked by calls saved, then tokens, then subject.
func Analyze(in Input) Report {
	runs := groupByRun(in.Calls)
	report := Report{Coverage: Coverage{Runs: len(runs), Stale: in.Stale, From: in.From, To: in.To, Library: in.Library}}
	for _, c := range in.Calls {
		if !c.IsCommand() {
			continue // a file-read tool call: read for bypasses only, as the friction profiles do
		}
		report.Coverage.Calls++
		if c.Owner != "" {
			report.Coverage.VrooliCalls++
		}
	}
	mine := func(kind string) bool { return in.Kinds == nil || slices.Contains(in.Kinds, kind) }
	if mine(KindSequence) {
		report.Opportunities = append(report.Opportunities, sequences(runs, in)...)
	}
	if mine(KindBypass) {
		report.Opportunities = append(report.Opportunities, bypasses(in.Calls)...)
	}
	if mine(KindOutputShape) {
		report.Opportunities = append(report.Opportunities, outputShapes(in.Calls)...)
	}
	if mine(KindSkillReload) {
		report.Opportunities = append(report.Opportunities, skillReloads(in.Calls)...)
	}
	sort.SliceStable(report.Opportunities, func(i, j int) bool {
		a, b := report.Opportunities[i], report.Opportunities[j]
		if a.CallsSaved != b.CallsSaved {
			return a.CallsSaved > b.CallsSaved
		}
		if a.TokensSaved != b.TokensSaved {
			return a.TokensSaved > b.TokensSaved
		}
		return a.Kind+a.Subject < b.Kind+b.Subject
	})
	if len(report.Opportunities) > MaxOpportunities {
		report.Opportunities = report.Opportunities[:MaxOpportunities]
	}
	return report
}

// run is one session's calls in time order.
type run struct {
	id    string
	calls []toolfriction.Call
}

// groupByRun splits calls by run, keeping the read order (stable by time).
func groupByRun(calls []toolfriction.Call) []run {
	index := map[string]int{}
	var runs []run
	for _, c := range calls {
		i, ok := index[c.RunID]
		if !ok {
			i = len(runs)
			index[c.RunID] = i
			runs = append(runs, run{id: c.RunID})
		}
		runs[i].calls = append(runs[i].calls, c)
	}
	for _, r := range runs {
		sort.SliceStable(r.calls, func(i, j int) bool { return r.calls[i].At.Before(r.calls[j].At) })
	}
	return runs
}

// evidence collects distinct run ids (with how often each showed the
// pattern) and example commands in first-seen order, and each run's calls
// saved.
type evidence struct {
	runs     []string
	hits     map[string]int
	examples []string
	saved    map[string]int64
}

// save adds calls a run would save.
func (e *evidence) save(id string, calls int) {
	if e.saved == nil {
		e.saved = map[string]int64{}
	}
	e.saved[id] += int64(calls)
}

// run records that run id showed the pattern n more times.
func (e *evidence) run(id string, n int) {
	if e.hits == nil {
		e.hits = map[string]int{}
	}
	if _, ok := e.hits[id]; !ok {
		e.runs = append(e.runs, id)
	}
	e.hits[id] += n
}

func (e *evidence) example(command string) {
	if command == "" || len(e.examples) >= maxExamples {
		return
	}
	for _, have := range e.examples {
		if have == command {
			return
		}
	}
	e.examples = append(e.examples, command)
}

func (e *evidence) sessions() int { return len(e.runs) }

// saved is what CallsSaved counts for a kind: its name and its one basis. A
// bypass and a jq extraction are not removed calls, so they are not called so.
type saved struct{ label, basis string }

var savedBy = map[string]saved{
	KindSequence:    {"calls saved", "each full occurrence's steps less the one program call that would run them"},
	KindBypass:      {"calls replaced", "direct reads of the scenario's internals, each by one supported command or RPC call"},
	KindOutputShape: {"extractions removed", "jq pipelines a focused flag makes unnecessary, one per call; the command call stays"},
	KindSkillReload: {"reads saved", "reads of a skill still in context, one each, in runs whose compactions are known"},
}

// fill sets the opportunity's sessions, evidence and calls saved, each run's
// capped by the one per-run rule (toolfriction.Capped), and states the basis
// with the cap when it cut the count. The evidence runs are those that showed
// the pattern most (first seen on a tie), not the first runs read, which
// would name the same sessions for every pattern.
func (e *evidence) fill(o *Opportunity) {
	capped, top := toolfriction.Capped(e.saved)
	o.CallsSaved, o.TopRunShare = int(capped), top
	o.SavedLabel, o.SavedBasis = savedBy[o.Kind].label, savedBy[o.Kind].basis
	var uncapped int64
	for _, n := range e.saved {
		uncapped += n
	}
	if capped < uncapped {
		o.SavedBasis += fmt.Sprintf("; %d of %d counted, each session at most %dx the median session's", capped, uncapped, toolfriction.RunCapFactor)
	}
	o.Sessions = e.sessions()
	runs := append([]string(nil), e.runs...)
	sort.SliceStable(runs, func(i, j int) bool { return e.hits[runs[i]] > e.hits[runs[j]] })
	o.RunIDs = runs[:min(len(runs), maxEvidenceRuns)]
	o.Examples = e.examples
}

// verifyAfter is the sentence that names the check: the same view over a
// window that starts after the change, where the pattern should shrink.
func verifyAfter(what string, sessions int) string {
	return fmt.Sprintf("after the change, `agent-manager usage opportunities --since <date of the change>` should no longer list %s (now in %d sessions)", what, sessions)
}

// topCounts is the n most frequent keys with their counts, ties by key.
func topCounts(counts map[string]int, n int) []string {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] != counts[keys[j]] {
			return counts[keys[i]] > counts[keys[j]]
		}
		return keys[i] < keys[j]
	})
	out := make([]string, 0, min(n, len(keys)))
	for _, k := range keys[:min(n, len(keys))] {
		out = append(out, fmt.Sprintf("%s ×%d", k, counts[k]))
	}
	return out
}

func joinSteps(steps []string) string { return strings.Join(steps, " → ") }
