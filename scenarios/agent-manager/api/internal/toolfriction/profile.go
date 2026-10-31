package toolfriction

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/runsignal"
	"agent-manager/internal/usagelabel"
)

// Owner kinds of a profile, cluster or finding.
const (
	OwnerScenario     = "scenario"
	OwnerControlPlane = "control-plane"
	OwnerExternal     = "external"
	controlPlane      = "vrooli"
)

// Polling: the same normalised command run pollMinRun or more times in one
// run with every gap under pollGap.
const (
	pollGap    = 10 * time.Minute
	pollMinRun = 3
	// reshapeGap: a repeat within it that cuts the output differently (another
	// filter, other jq paths) re-reads the result just printed in a new shape;
	// it is not a poll (see pollSeries).
	reshapeGap = 30 * time.Second
	// helpRetryWindow is how many calls of a run before and after a help
	// lookup may hold the failure or retry that makes it a shape guess.
	helpRetryWindow = 5
	// slowBudgetMS is how long a query-shaped command (status, list, get,
	// read…) is expected to take; time beyond it is the slow finding's cost.
	// A path is slow when it has minSlowCalls timed calls, its p90 is over the
	// budget and its excess is at least slowMinShare of the window's Vrooli CLI
	// time.
	slowBudgetMS = 5_000
	minSlowCalls = 5
	slowMinShare = 0.001
	// RunCapFactor: one run's contribution to a finding's cost is capped at
	// this many times the median run's contribution, so a finding made by one
	// looping run does not rank as if many runs hit it.
	RunCapFactor = 10
)

// Coverage says how much of the window the profiles can see: command calls
// and those whose outcome is known. Silence needs it: no finding over unknown
// outcomes means "cannot tell", not "no friction". OlderParser counts the
// commands of runs read by an older transcript parser: their missing outcomes
// are a parser gap, not evidence of success; NoTranscript those of runs with
// no transcript record, whose parser is not known. Stale is what the window
// holds at an older fact version, which no finding reads. UnknownTokens counts
// the commands whose result no segment sizes (see Call.TokenBasis and
// sizedResults: a segment whose output sits on its holder is not unknown). OwnWork
// counts the direct reads of a scenario's database, port or state by runs
// working on that scenario (Call.OwnWork), which are not bypasses.
type Coverage struct {
	Commands, OutcomeKnown             int
	VrooliCommands, VrooliOutcomeKnown int
	OlderParser, NoTranscript          int
	UnknownTokens                      int
	OwnWork                            int
	Stale                              Stale
}

// Stale is what a window holds at an older invocation fact version: runs whose
// facts await re-derivation by the current rules, their command calls, and
// those calls per command path. Reading them would score old features by new
// rules, so findings exclude them and say how many of their command's calls
// they left out.
type Stale struct {
	Runs, Calls int
	Paths       map[string]int
}

// Sentence is the stale part of a coverage line, "" when nothing is stale.
func (s Stale) Sentence() string {
	if s.Runs == 0 {
		return ""
	}
	return fmt.Sprintf("%d runs (%d calls) awaiting re-derivation; findings exclude them", s.Runs, s.Calls)
}

// Window is one window's calls at the current fact version and what it holds
// at older ones.
type Window struct {
	Calls []Call
	Stale Stale
}

// KnownShare is the share of command calls with a known outcome.
func (c Coverage) KnownShare() float64 { return share(c.OutcomeKnown, c.Commands) }

// Sentence is the coverage line printed beside every list, empty or not.
func (c Coverage) Sentence() string {
	stale := c.Stale.Sentence()
	if c.Commands == 0 {
		sentence := "no command calls in the window: cannot tell whether tools caused friction"
		if stale != "" {
			sentence = "no command calls at the current fact version: cannot tell whether tools caused friction; " + stale
		}
		return sentence
	}
	sentence := fmt.Sprintf("%d commands, outcome known for %.0f%% (%d Vrooli CLI calls, %.0f%% known)", c.Commands, 100*c.KnownShare(), c.VrooliCommands, 100*share(c.VrooliOutcomeKnown, c.VrooliCommands))
	if c.OlderParser > 0 {
		sentence += fmt.Sprintf("; %d commands from runs read by an older parser; outcomes there may be missing", c.OlderParser)
	}
	if c.NoTranscript > 0 {
		sentence += fmt.Sprintf("; %d commands from runs with no transcript record, whose parser is unknown", c.NoTranscript)
	}
	if c.UnknownTokens > 0 {
		sentence += fmt.Sprintf("; result tokens unknown for %d commands", c.UnknownTokens)
	}
	if c.OwnWork > 0 {
		sentence += fmt.Sprintf("; %d direct reads of a scenario's internals by runs working on that scenario are not counted as bypasses", c.OwnWork)
	}
	if stale != "" {
		sentence += "; " + stale
	}
	return sentence
}

// Profile is one command path in the window.
type Profile struct {
	CommandPath, Owner, OwnerKind string
	Calls, Runs                   int
	// Decided calls succeeded or failed; FailureRate is Failed over them, so
	// no-match, refusals, verdicts, interrupts and unknowns do not count.
	Decided, Failed, NoMatch, Refused, Unknown int
	FailureRate                                float64
	// ShapeGuesses are failures that say the shape was wrong, plus help
	// lookups around a failed or retried call (see helpGuesses). HelpLookups is
	// every bare help lookup of the path. WrongShape are the failures among
	// ShapeGuesses.
	ShapeGuesses, HelpLookups, WrongShape int
	// Output weight: result tokens per call, the share the harness truncated,
	// the context tax (result tokens × the requests that re-read them) and the
	// share of calls that piped the output through a filter.
	MedianTokens, P90Tokens, MaxTokens int64
	// MaxOutputBytes is the largest output a call printed, which the harness
	// may have cut before the model read it (MaxTokens is what it read).
	MaxOutputBytes                int64
	TruncatedShare, FilteredShare float64
	ContextTax                    int64
	JSONPaths                     []string
	// TotalMS is the time of calls with a known duration (a chain's time
	// once, on the segment holding it); P90MS is over Timed calls.
	TotalMS, P90MS int64
	// Timed calls have a duration of their own, not a chain's they hold (see
	// sharedDurations); SlowExcessMS is their time beyond slowBudgetMS,
	// counted only for query-shaped commands.
	Timed        int
	SlowExcessMS int64
	// Polled calls belong to a series of one normalised command run three or
	// more times with gaps under ten minutes; Wrapped ran inside a sleep or
	// until/while loop. Steered are the polled calls a blocking sibling
	// stands in for (see steering), Unnamed those that missed one only
	// because the start before them named no target.
	Polled, Wrapped, Steered, Unnamed int
	// Blocking are the commands its manifest declares blocking beside the
	// path (runsignal.BlockingSiblings) and Applied those that stood in for
	// at least one polled call, in declaration order.
	Blocking []runsignal.Blocking
	Applied  []string
	// CostShare is the profile's largest share of any harm in the window:
	// failed calls, wall-clock or context tax. Profiles rank by it.
	CostShare float64
	runs      map[string]int
	// Each run's share of a finding's cost, for the concentration cap.
	shapeRuns, polledRuns, steerRuns, taxRuns, slowRuns map[string]int64
}

// Totals are the window's harm in each unit, the denominators of every share,
// and the runs whose commands it read, the denominator of every breadth.
type Totals struct {
	Calls, FailedCalls  int
	Runs                int
	DurationMS          int64
	ContextTax          int64
	ShapeGuesses        int
	Polled, BypassCalls int
	FilteredShare       float64
	// Vrooli CLI calls (scenario CLIs and the control plane): those that
	// succeeded or failed, the failures among them, the shape guesses and the
	// wrong-shape failures among them, the polled calls and their known time.
	// These are the totals of the kinds only a Vrooli CLI has (see Base): a
	// failure or a slow run of grep or git is nobody's to fix here and does
	// not dilute them.
	VrooliCalls, VrooliDecided, VrooliFailed, VrooliShapeGuesses, VrooliWrongShape, VrooliPolled int
	VrooliDurationMS                                                                             int64
}

// UnitOf is the unit a kind of finding states its cost in.
func UnitOf(kind string) string {
	switch kind {
	case KindSlow, HarmWallClock:
		return UnitMS
	case KindOutputWeight:
		return UnitTokens
	}
	return UnitCalls
}

// Base is the window's total of a kind of harm and its name: the
// denominator every share of that kind is over, so the friction list, the
// recommendations and their readers state one base per kind. Failures,
// degradations, shape guesses, polls, slow time and time in degraded calls are
// over Vrooli CLI calls,
// the only calls whose owner can act on them; context tax is over every
// command, since any output re-read costs the same; bypasses over bypass
// calls.
func (t Totals) Base(kind string) (int64, string) {
	switch kind {
	case KindFailure:
		return int64(t.VrooliFailed), fmt.Sprintf("the window's %d Vrooli CLI failed calls", t.VrooliFailed)
	case KindDegraded:
		return int64(t.VrooliCalls), fmt.Sprintf("the window's %d Vrooli CLI calls", t.VrooliCalls)
	case KindShapeGuess:
		return int64(t.VrooliWrongShape), fmt.Sprintf("the window's %d Vrooli CLI wrong-shape calls", t.VrooliWrongShape)
	case KindPolling:
		return int64(t.VrooliPolled), fmt.Sprintf("the window's %d polled Vrooli CLI calls", t.VrooliPolled)
	case KindSlow, HarmWallClock:
		return t.VrooliDurationMS, "the window's Vrooli CLI time (" + usagelabel.Duration(t.VrooliDurationMS) + ")"
	case KindOutputWeight:
		return t.ContextTax, "the window's context tax (" + usagelabel.Tokens(t.ContextTax) + " re-read tokens)"
	case KindBypass:
		return int64(t.BypassCalls), fmt.Sprintf("the window's %d bypass calls", t.BypassCalls)
	}
	return int64(t.Calls), fmt.Sprintf("the window's %d command calls", t.Calls)
}

// VrooliFailureRate is the failed share of Vrooli CLI calls whose outcome
// says whether the command worked.
func (t Totals) VrooliFailureRate() float64 { return share(t.VrooliFailed, t.VrooliDecided) }

// VrooliShapeGuessRate is the share of Vrooli CLI calls that guessed a shape.
func (t Totals) VrooliShapeGuessRate() float64 { return share(t.VrooliShapeGuesses, t.VrooliCalls) }

// PolledShare is the share of command calls that belong to a poll series.
func (t Totals) PolledShare() float64 { return share(t.Polled, t.Calls) }

// Report is everything the window's calls show.
type Report struct {
	Coverage Coverage
	Totals   Totals
	Profiles []Profile
	Clusters []Cluster
	Findings []Finding
}

// Analyze profiles a window's calls (in any order) and derives its findings.
func Analyze(w Window) Report {
	calls := append([]Call(nil), w.Calls...)
	sort.SliceStable(calls, func(i, j int) bool {
		if calls[i].RunID != calls[j].RunID {
			return calls[i].RunID < calls[j].RunID
		}
		return calls[i].At.Before(calls[j].At)
	})
	polled := pollSeries(calls)
	steered, unnamed := steering(calls, polled)
	helpGuessed := helpGuesses(calls)
	shared := sharedDurations(calls)
	sized := sizedResults(calls)
	report := Report{Coverage: Coverage{Stale: w.Stale}}
	byPath := map[string]*Profile{}
	tokens := map[string][]int64{}
	durations := map[string][]int64{}
	truncated, filtered := map[string]int{}, map[string]int{}
	paths := map[string]map[string]int{}
	filteredAll := 0
	runs := map[string]bool{}
	for i, c := range calls {
		runs[c.RunID] = true
		if c.Bypasses() {
			report.Totals.BypassCalls++
		}
		if c.OwnWork {
			report.Coverage.OwnWork++
		}
		if !c.IsCommand() {
			continue // a file-read tool call: read for bypass only
		}
		report.Coverage.Commands++
		if c.OlderParser {
			report.Coverage.OlderParser++
		}
		if c.NoTranscript {
			report.Coverage.NoTranscript++
		}
		if !c.TokensKnown() && !sized[resultKey(c)] {
			report.Coverage.UnknownTokens++
		}
		known := c.Outcome != runsignal.OutcomeUnknown
		if known {
			report.Coverage.OutcomeKnown++
		}
		if c.Owner != "" {
			report.Coverage.VrooliCommands++
			if known {
				report.Coverage.VrooliOutcomeKnown++
			}
		}
		p := byPath[c.CommandPath]
		if p == nil {
			p = &Profile{CommandPath: c.CommandPath, Owner: c.Owner, OwnerKind: ownerKind(c.Owner), runs: map[string]int{}, shapeRuns: map[string]int64{}, polledRuns: map[string]int64{}, steerRuns: map[string]int64{}, taxRuns: map[string]int64{}, slowRuns: map[string]int64{}, Blocking: runsignal.BlockingSiblings(c.CommandPath)}
			byPath[c.CommandPath] = p
		}
		p.Calls++
		p.runs[c.RunID]++
		switch {
		case c.decided():
			p.Decided++
			if c.Failed() {
				p.Failed++
			}
		case c.Outcome == runsignal.OutcomeNoMatch:
			p.NoMatch++
		case c.Outcome == runsignal.OutcomeRefused:
			p.Refused++
		case c.Outcome == runsignal.OutcomeUnknown:
			p.Unknown++
		}
		if c.Failed() && c.ShapeGuess {
			// The costly part of guessing: a call that failed for its shape and
			// must be made again. Help lookups around it are cheap reads.
			p.WrongShape++
			p.shapeRuns[c.RunID]++
		}
		if (c.Failed() && c.ShapeGuess) || helpGuessed[i] {
			p.ShapeGuesses++
		}
		if c.Help {
			p.HelpLookups++
		}
		tokens[c.CommandPath] = append(tokens[c.CommandPath], c.ResultTokens)
		p.ContextTax += c.ResultTokens * c.Residency
		p.taxRuns[c.RunID] += c.ResultTokens * c.Residency
		p.MaxOutputBytes = max(p.MaxOutputBytes, c.OutputBytes)
		if c.HasDuration {
			p.TotalMS += c.DurationMS
		}
		if c.HasDuration && !shared[i] {
			durations[c.CommandPath] = append(durations[c.CommandPath], c.DurationMS)
			p.Timed++
			if interactiveVerbs[lastWord(c.CommandPath)] {
				excess := max(c.DurationMS-slowBudgetMS, 0)
				p.SlowExcessMS += excess
				p.slowRuns[c.RunID] += excess
			}
		}
		if c.Truncated {
			truncated[c.CommandPath]++
		}
		if c.Filtered {
			filtered[c.CommandPath]++
			filteredAll++
		}
		for _, path := range c.JSONPaths {
			if paths[c.CommandPath] == nil {
				paths[c.CommandPath] = map[string]int{}
			}
			paths[c.CommandPath][path]++
		}
		if polled[i] {
			p.Polled++
			p.polledRuns[c.RunID]++
			if c.Wrapped {
				p.Wrapped++
			}
			if len(steered[i]) > 0 {
				p.Steered++
				p.steerRuns[c.RunID]++
			}
			if unnamed[i] {
				p.Unnamed++
			}
			for _, b := range steered[i] {
				if !slices.Contains(p.Applied, b.Command()) {
					p.Applied = append(p.Applied, b.Command())
				}
			}
		}
	}
	t := &report.Totals
	for _, p := range byPath {
		p.Runs = len(p.runs)
		p.FailureRate = share(p.Failed, p.Decided)
		p.MedianTokens, p.P90Tokens, p.MaxTokens = percentile(tokens[p.CommandPath], 0.5), percentile(tokens[p.CommandPath], 0.9), percentile(tokens[p.CommandPath], 1)
		p.P90MS = percentile(durations[p.CommandPath], 0.9)
		p.TruncatedShare, p.FilteredShare = share(truncated[p.CommandPath], p.Calls), share(filtered[p.CommandPath], p.Calls)
		p.JSONPaths = topKeys(paths[p.CommandPath], 5)
		t.Calls += p.Calls
		t.FailedCalls += p.Failed
		t.DurationMS += p.TotalMS
		t.ContextTax += p.ContextTax
		t.ShapeGuesses += p.ShapeGuesses
		t.Polled += p.Polled
		if p.Owner != "" {
			t.VrooliCalls += p.Calls
			t.VrooliDecided += p.Decided
			t.VrooliFailed += p.Failed
			t.VrooliShapeGuesses += p.ShapeGuesses
			t.VrooliWrongShape += p.WrongShape
			t.VrooliPolled += p.Polled
			t.VrooliDurationMS += p.TotalMS
		}
	}
	t.FilteredShare = share(filteredAll, t.Calls)
	t.Runs = len(runs)
	for _, p := range byPath {
		p.CostShare = max(share(p.Failed, t.FailedCalls), shareInt64(p.TotalMS, t.DurationMS), shareInt64(p.ContextTax, t.ContextTax))
		report.Profiles = append(report.Profiles, *p)
	}
	sort.Slice(report.Profiles, func(i, j int) bool {
		a, b := report.Profiles[i], report.Profiles[j]
		if a.CostShare != b.CostShare {
			return a.CostShare > b.CostShare
		}
		return a.CommandPath < b.CommandPath
	})
	report.Clusters = clusters(calls)
	report.Findings = findings(report, calls, polled)
	return report
}

// RunScore is one run's tool friction: the share of its command calls that
// failed, guessed a command's shape or belonged to a poll series (Frictional:
// a call counts once), over the commands read and those with a known outcome.
// It is the one per-run friction figure supervision triggers read. Through is
// how far the run's events are projected at the current fact version (zero:
// not projected at it), set by the store.
type RunScore struct {
	Commands, OutcomeKnown, Failed, ShapeGuesses, Polled, Frictional int
	Score                                                            float64
	Through                                                          time.Time
}

// MinScorableKnownShare is the share of a run's commands that must have a
// known outcome before its score says anything: below it the failures it
// cannot see could be most of them, so a low score is not quiet.
const MinScorableKnownShare = 0.8

// Scorable reports whether the score can be read: the run has commands and
// enough of them have a known outcome. An unscorable run is "cannot tell",
// never "no friction".
func (s RunScore) Scorable() bool {
	return s.Commands > 0 && share(s.OutcomeKnown, s.Commands) >= MinScorableKnownShare
}

// ScoreRun scores one run's calls (in any order) with the same failure,
// shape-guess and poll rules as the window profiles.
func ScoreRun(calls []Call) RunScore {
	calls = append([]Call(nil), calls...)
	sort.SliceStable(calls, func(i, j int) bool { return calls[i].At.Before(calls[j].At) })
	polled, helpGuessed := pollSeries(calls), helpGuesses(calls)
	var s RunScore
	for i, c := range calls {
		if !c.IsCommand() {
			continue
		}
		s.Commands++
		if c.Outcome != runsignal.OutcomeUnknown {
			s.OutcomeKnown++
		}
		failed, guessed := c.Failed(), (c.Failed() && c.ShapeGuess) || helpGuessed[i]
		if failed {
			s.Failed++
		}
		if guessed {
			s.ShapeGuesses++
		}
		if polled[i] {
			s.Polled++
		}
		if failed || guessed || polled[i] {
			s.Frictional++
		}
	}
	s.Score = share(s.Frictional, s.Commands)
	return s
}

// pollSeries marks the calls of every poll series: one run, one normalised
// read command (Key is set only for those), three or more distinct calls,
// each within pollGap of the last. A wait that ran its course is not a poll
// (see Call.waited), and neither is a reshape: a repeat within reshapeGap of
// the series' last call that cuts the output in another shape (filtered or
// not, other jq paths) re-reads the result just printed, so it neither joins
// nor breaks the series. A repeat in the same shape stays a poll however soon.
func pollSeries(calls []Call) []bool {
	marked := make([]bool, len(calls))
	type key struct{ run, command string }
	last := map[key][]int{}
	flush := func(series []int) {
		distinct := map[string]bool{}
		for _, i := range series {
			distinct[calls[i].CallEventID] = true
		}
		if len(distinct) >= pollMinRun {
			for _, i := range series {
				marked[i] = true
			}
		}
	}
	for i, c := range calls {
		if c.Key == "" || c.Help || c.waited() {
			continue
		}
		k := key{c.RunID, c.Key}
		series := last[k]
		if n := len(series); n > 0 && c.reshapes(calls[series[n-1]]) {
			continue
		}
		if len(series) > 0 && c.At.Sub(calls[series[len(series)-1]].At) >= pollGap {
			flush(series)
			series = nil
		}
		last[k] = append(series, i)
	}
	for _, series := range last {
		flush(series)
	}
	return marked
}

// steering is, for each polled call, the blocking siblings that stand in for
// it (runsignal.BlockingSiblings): one with no precondition always; one
// declared to block only after another command acted on the target
// (`vrooli scenario wait` after a start or restart) only when such a command
// ran on the poll's target earlier in the run, inside the poll's stretch (its
// command's calls with gaps under pollGap) or within pollGap before it.
// unnamed marks the polls that missed one only because the matching command
// named no target (`make restart` in the scenario's directory): it cannot be
// shown to be the same target, so it is not claimed.
func steering(calls []Call, polled []bool) ([][]runsignal.Blocking, []bool) {
	steered, unnamed := make([][]runsignal.Blocking, len(calls)), make([]bool, len(calls))
	siblings := map[string][]runsignal.Blocking{}
	after := map[string]bool{}
	for i, c := range calls {
		if _, ok := siblings[c.CommandPath]; !ok && polled[i] {
			siblings[c.CommandPath] = runsignal.BlockingSiblings(c.CommandPath)
			for _, b := range siblings[c.CommandPath] {
				for _, path := range b.After {
					after[path] = true
				}
			}
		}
	}
	type stretch struct{ run, key string }
	type acted struct{ run, path, target string }
	began, last := map[stretch]time.Time{}, map[stretch]time.Time{}
	actedAt := map[acted]time.Time{}
	for i, c := range calls {
		if after[c.CommandPath] {
			actedAt[acted{c.RunID, c.CommandPath, c.target()}] = c.At
		}
		if c.Key == "" {
			continue
		}
		k := stretch{c.RunID, c.Key}
		if at, ok := last[k]; !ok || c.At.Sub(at) >= pollGap {
			began[k] = c.At
		}
		last[k] = c.At
		if !polled[i] {
			continue
		}
		since := began[k].Add(-pollGap)
		ranOn := func(b runsignal.Blocking, target string) bool {
			for _, path := range b.After {
				if at, ok := actedAt[acted{c.RunID, path, target}]; ok && !at.Before(since) {
					return true
				}
			}
			return false
		}
		target := c.target()
		for _, b := range siblings[c.CommandPath] {
			switch {
			case len(b.After) == 0 || target != "" && ranOn(b, target):
				steered[i] = append(steered[i], b)
			case ranOn(b, ""):
				unnamed[i] = true
			}
		}
		if len(steered[i]) > 0 {
			unnamed[i] = false
		}
	}
	return steered, unnamed
}

// reshapes reports whether c re-reads the result of prev, the same command
// run moments before, in another shape: within reshapeGap, with a different
// output filter or other jq paths.
func (c Call) reshapes(prev Call) bool {
	return c.At.Sub(prev.At) < reshapeGap && (c.Filtered != prev.Filtered || strings.Join(c.JSONPaths, "\n") != strings.Join(prev.JSONPaths, "\n"))
}

// helpGuesses marks the help lookups that show an agent guessing a command's
// shape: within a few calls of one run, either a call of the same program
// failed before the lookup and the program ran again after it, or the same
// command path ran both before and after it (a retry). A lookup with neither
// is reading the manual, however often agents do it: navigating a subcommand
// index is not friction until a call around it went wrong.
func helpGuesses(calls []Call) []bool {
	marked := make([]bool, len(calls))
	for i, c := range calls {
		if !c.Help {
			continue
		}
		program := firstWord(c.CommandPath)
		failedBefore, ranBefore := false, false
		for j := i - 1; j >= 0 && j >= i-helpRetryWindow && calls[j].RunID == c.RunID; j-- {
			if firstWord(calls[j].CommandPath) != program || calls[j].Help {
				continue
			}
			failedBefore = failedBefore || calls[j].Failed()
			ranBefore = ranBefore || calls[j].CommandPath == c.CommandPath
		}
		for j := i + 1; j < len(calls) && j <= i+helpRetryWindow && calls[j].RunID == c.RunID; j++ {
			if firstWord(calls[j].CommandPath) == program && !calls[j].Help && (failedBefore || ranBefore && calls[j].CommandPath == c.CommandPath) {
				marked[i] = true
				break
			}
		}
	}
	return marked
}

func ownerKind(owner string) string {
	switch owner {
	case "":
		return OwnerExternal
	case controlPlane:
		return OwnerControlPlane
	}
	return OwnerScenario
}

func share(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

func shareInt64(n, d int64) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

// percentile is the nearest-rank percentile (q in (0, 1]) of values.
func percentile(values []int64, q float64) int64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	rank := int(q*float64(len(sorted))+0.999999) - 1
	return sorted[min(max(rank, 0), len(sorted)-1)]
}

// topKeys are the n most frequent keys, ties alphabetically.
func topKeys(counts map[string]int, n int) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] != counts[keys[j]] {
			return counts[keys[i]] > counts[keys[j]]
		}
		return keys[i] < keys[j]
	})
	if len(keys) > n {
		keys = keys[:n]
	}
	return keys
}

func firstWord(path string) string {
	for i := 0; i < len(path); i++ {
		if path[i] == ' ' {
			return path[:i]
		}
	}
	return path
}
