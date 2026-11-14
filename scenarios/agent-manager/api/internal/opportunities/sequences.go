package opportunities

import (
	"fmt"
	"sort"
	"strings"

	"agent-manager/internal/toolfriction"
)

// sequence lengths mined, and the overlap that makes two mined sequences one
// behaviour: sharing two steps of one run means one is the other shifted (a
// longer sequence) or rotated (a loop).
const (
	minGram      = 3
	maxGram      = 4
	minOverlap   = 2
	maxMotifStep = 8
)

// step is one command path in a session's sequence: consecutive repeats of
// the same path are one step.
type step struct {
	path, owner    string
	calls, unknown int
	tokens         int64
	command        string
	// output is the first call's result event, the text it printed.
	output string
}

// stepsOf is a session's Vrooli CLI commands in order, consecutive duplicates
// collapsed. Tools outside Vrooli (rg, sed…) have no owner and are not steps;
// nor are help lookups and bare executables (`prompt-manager` with no
// subcommand), which read a manual and do no step of the work.
func stepsOf(calls []toolfriction.Call) []step {
	var out []step
	for _, c := range calls {
		if c.Owner == "" || c.Help || !strings.Contains(c.CommandPath, " ") {
			continue
		}
		unknown := 0
		if !c.TokensKnown() {
			unknown = 1
		}
		if n := len(out); n > 0 && out[n-1].path == c.CommandPath {
			out[n-1].calls++
			out[n-1].unknown += unknown
			out[n-1].tokens += c.ResultTokens
			continue
		}
		out = append(out, step{path: c.CommandPath, owner: c.Owner, calls: 1, unknown: unknown, tokens: c.ResultTokens, command: c.Command, output: c.OutputEventID})
	}
	return out
}

// sliceBoundary is the step that closes a slice of work: the agent journals
// what it did, then starts the next task. It may end a motif; the steps of the
// following task are not part of it.
const sliceBoundary = "vrooli-memory journal note"

// span is one occurrence of a mined sequence: steps [start, end) of a run.
type span struct{ run, start, end int }

// sequences mines 3- and 4-grams of command paths, keeps those in at least
// MinSessions runs, and merges the ones whose occurrences overlap by two or
// more steps into one maximal motif, so a behaviour is counted once however
// many shifted or rotated n-grams it shows as. Each motif occurrence is a
// maximal stretch of a run its n-grams cover; a step belongs to one motif.
func sequences(runs []run, in Input) []Opportunity {
	steps := make([][]step, len(runs))
	spans := map[string][]span{}
	var keys []string
	for ri, r := range runs {
		steps[ri] = stepsOf(r.calls)
		for n := minGram; n <= maxGram; n++ {
			for i := 0; i+n <= len(steps[ri]); i++ {
				key := pathKey(steps[ri][i : i+n])
				if spans[key] == nil {
					keys = append(keys, key)
				}
				spans[key] = append(spans[key], span{ri, i, i + n})
			}
		}
	}
	var grams []string
	for _, key := range keys {
		if distinctRuns(spans[key]) >= MinSessions {
			grams = append(grams, key)
		}
	}
	groups := mergeOverlapping(grams, spans)
	programRuns := programRunCounts(in)
	claimed := make([][]bool, len(runs))
	for ri := range runs {
		claimed[ri] = make([]bool, len(steps[ri]))
	}
	var out []Opportunity
	for _, group := range groups {
		if m, ok := motifOf(group, spans, steps, runs, claimed); ok {
			out = append(out, m.opportunity(in, programRuns))
		}
	}
	return out
}

func pathKey(window []step) string {
	paths := make([]string, len(window))
	for i, s := range window {
		paths[i] = s.path
	}
	return strings.Join(paths, "\x00")
}

func distinctRuns(spans []span) int {
	seen := map[int]bool{}
	for _, s := range spans {
		seen[s.run] = true
	}
	return len(seen)
}

// mergeOverlapping unions the grams whose occurrences share minOverlap steps
// of a run, and returns the groups largest (most runs) first.
func mergeOverlapping(grams []string, spans map[string][]span) [][]string {
	parent := map[string]string{}
	var find func(string) string
	find = func(k string) string {
		if parent[k] != k {
			parent[k] = find(parent[k])
		}
		return parent[k]
	}
	type tagged struct {
		span
		gram string
	}
	byRun := map[int][]tagged{}
	for _, g := range grams {
		parent[g] = g
		for _, s := range spans[g] {
			byRun[s.run] = append(byRun[s.run], tagged{s, g})
		}
	}
	for _, list := range byRun {
		sort.Slice(list, func(i, j int) bool { return list[i].start < list[j].start })
		for i := range list {
			for j := i + 1; j < len(list) && list[j].start <= list[i].end-minOverlap; j++ {
				if min(list[i].end, list[j].end)-list[j].start >= minOverlap {
					parent[find(list[j].gram)] = find(list[i].gram)
				}
			}
		}
	}
	members := map[string][]string{}
	for _, g := range grams {
		members[find(g)] = append(members[find(g)], g)
	}
	out := make([][]string, 0, len(members))
	runsOf := map[string]int{}
	for root, group := range members {
		var all []span
		for _, g := range group {
			all = append(all, spans[g]...)
		}
		runsOf[root] = distinctRuns(all)
		out = append(out, group)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := runsOf[find(out[i][0])], runsOf[find(out[j][0])]
		if a != b {
			return a > b
		}
		return out[i][0] < out[j][0]
	})
	return out
}

// motif is one merged behaviour: its representative steps (the subject) and
// totals. occurrences are periods of the shape; calls, tokens and unknown
// count only the calls of subject steps inside its stretches, so other
// commands a merged stretch interleaves are not credited to it. In each run a
// program call per period replaces calls − occurrences round-trips however
// often the shape loops, and never more than the subject's other steps:
// (len − 1) per occurrence; runs are capped by the one per-run rule.
type motif struct {
	steps       []step
	evidence    evidence
	occurrences int
	calls       int
	unknown     int
	tokens      int64
	// places are the steps of each occurrence, for measuring result dependency.
	places [][]step
}

// saved is the round-trips one program call per occurrence replaces in a
// run with that many calls of subject steps and occurrences.
func (m motif) saved(calls, occurrences int) int {
	return max(0, min(calls-occurrences, (len(m.steps)-1)*occurrences))
}

// mandatedRitual are the command prefixes AGENTS.md makes every agent run
// (recall, skill load, journal): a sequence of only these is work the operator
// requires, so wrapping it saves round-trips and result tokens, not the work.
var mandatedRitual = []string{"vrooli-memory journal", "search-hub query", "prompt-manager skill read", "prompt-manager discover"}

func (m motif) mandated() bool {
	for _, s := range m.steps {
		ok := false
		for _, prefix := range mandatedRitual {
			ok = ok || strings.HasPrefix(s.path, prefix)
		}
		if !ok {
			return false
		}
	}
	return len(m.steps) > 0
}

// motifOf turns a group into a motif: per run, the steps its grams cover and
// no earlier motif claimed, split into maximal stretches of at least minGram
// steps. Merged grams need not all occur together, so a stretch need not hold
// every step: the motif's subject is the stretch shape (a repeated loop
// reduced to one period) found in the most runs, in the order observed, and
// only the places a stretch runs the subject in full are occurrences, evidence
// and savings; the steps of those alone are claimed.
func motifOf(group []string, spans map[string][]span, steps [][]step, runs []run, claimed [][]bool) (motif, bool) {
	covered := map[int][]bool{}
	for _, g := range group {
		for _, s := range spans[g] {
			if covered[s.run] == nil {
				covered[s.run] = make([]bool, len(steps[s.run]))
			}
			for i := s.start; i < s.end; i++ {
				covered[s.run][i] = !claimed[s.run][i]
			}
		}
	}
	var stretches []span
	shapes := map[string][]step{}
	for ri := range runs {
		marks := covered[ri]
		for i := 0; i < len(marks); {
			if !marks[i] {
				i++
				continue
			}
			j := i
			for j < len(marks) && marks[j] {
				j++
			}
			for from, k := i, i; k < j; k++ {
				if steps[ri][k].path != sliceBoundary && k < j-1 {
					continue
				}
				if k+1-from >= minGram {
					shape := period(steps[ri][from : k+1])
					shapes[pathKey(shape)] = shape
					stretches = append(stretches, span{ri, from, k + 1})
				}
				from = k + 1
			}
			i = j
		}
	}
	var m motif
	best, bestRuns := "", 0
	for key, shape := range shapes {
		runsWith := map[int]bool{}
		for _, st := range stretches {
			if len(occurrences(steps[st.run][st.start:st.end], shape)) > 0 {
				runsWith[st.run] = true
			}
		}
		if n := len(runsWith); n > bestRuns || n == bestRuns && key < best {
			best, bestRuns = key, n
		}
	}
	if bestRuns < MinSessions {
		return m, false
	}
	m.steps = shapes[best]
	for _, st := range stretches {
		found := occurrences(steps[st.run][st.start:st.end], m.steps)
		if len(found) == 0 {
			continue
		}
		id := runs[st.run].id
		m.evidence.run(id, len(found))
		m.occurrences += len(found)
		calls := 0
		for _, at := range found {
			m.places = append(m.places, steps[st.run][st.start+at:st.start+at+len(m.steps)])
			for k := st.start + at; k < st.start+at+len(m.steps); k++ {
				step := steps[st.run][k]
				calls += step.calls
				m.unknown += step.unknown
				m.tokens += step.tokens
				m.evidence.example(step.command)
				claimed[st.run][k] = true
			}
		}
		m.calls += calls
		m.evidence.save(id, m.saved(calls, len(found)))
	}
	return m, true
}

// occurrences are the offsets in a stretch where the subject's steps run in
// full and in order, without overlap.
func occurrences(stretch, subject []step) []int {
	var out []int
	for i := 0; i+len(subject) <= len(stretch); {
		if pathKey(stretch[i:i+len(subject)]) == pathKey(subject) {
			out = append(out, i)
			i += len(subject)
			continue
		}
		i++
	}
	return out
}

// period reduces a stretch that repeats a loop (A B C A B C A) to one turn of
// it: the shortest turn of at most maxMotifStep steps that the stretch repeats
// in full at least once. A stretch that only starts over (A B C A) is not a
// loop; it, like any other, is bounded at maxMotifStep steps.
func period(stretch []step) []step {
	for p := 2; p <= maxMotifStep && 2*p <= len(stretch); p++ {
		repeats := true
		for i := p; i < len(stretch) && repeats; i++ {
			repeats = stretch[i].path == stretch[i-p].path
		}
		if repeats {
			return stretch[:p]
		}
	}
	return stretch[:min(len(stretch), maxMotifStep)]
}

func (m motif) paths() []string {
	out := make([]string, len(m.steps))
	for i, s := range m.steps {
		out[i] = s.path
	}
	return out
}

// opportunity states the motif with what the library says about it. A
// program is adopted only when it calls the commands of a majority of the
// motif's steps; an affinity or minority match is a lead to investigate.
func (m motif) opportunity(in Input, programRuns map[string]int) Opportunity {
	subject := joinSteps(m.paths())
	o := Opportunity{
		Kind: KindSequence, Subject: subject, Occurrences: m.occurrences,
		TokensSaved: m.tokens, TokensUnknown: m.unknown, Owner: dominantOwner(m.steps), Mandated: m.mandated(),
		Verify: verifyAfter("`"+subject+"`", m.evidence.sessions()),
	}
	m.evidence.fill(&o)
	if in.Library.Unavailable != "" {
		o.Verdict = VerdictUnknown
		o.Action = fmt.Sprintf("check whether a program covers `%s` (%d times in %d sessions): the program library could not be read", subject, m.occurrences, o.Sessions)
		return o
	}
	d := m.measureDependency(in.Outputs)
	o.DependentSteps, o.MeasuredSteps = d.Used, d.Measured
	switch {
	case d.varying() && d.Measured == 0:
		o.Verdict, o.CallsSaved, o.TokensSaved = VerdictUnmeasured, 0, 0
		why := "was not available"
		if d.Unread != "" {
			why = "could not be read (" + d.Unread + ")"
		}
		o.Action = fmt.Sprintf("cannot tell whether a program could run `%s` (%d times in %d sessions): the output text of its earlier steps %s to check whether later steps used it, so no program savings are claimed", subject, m.occurrences, o.Sessions, why)
		return o
	case d.varying():
		low, high := d.interval()
		rate := fmt.Sprintf("%d of %d sampled later steps used an earlier step's result, %.0f%% (95%% interval %.0f–%.0f%%)", d.Used, d.Measured, 100*d.rate(), 100*low, 100*high)
		switch {
		case high < MinDependency:
			o.Verdict, o.CallsSaved, o.TokensSaved = VerdictIndependent, 0, 0
			o.Action = fmt.Sprintf("agents run `%s` together (%d times in %d sessions) but choose each step independently (%s): a program cannot pick the later step, so it would save no calls", subject, m.occurrences, o.Sessions, rate)
			return o
		case low < MinDependency:
			o.Verdict, o.CallsSaved, o.TokensSaved = VerdictUndetermined, 0, 0
			o.Action = fmt.Sprintf("cannot tell whether a program could pick the later steps of `%s` (%d times in %d sessions): %s, which does not settle the %.0f%% a program needs; no program savings are claimed", subject, m.occurrences, o.Sessions, rate, 100*MinDependency)
			return o
		}
	}
	impact := fmt.Sprintf("%d calls saved", o.CallsSaved)
	if o.Mandated {
		impact = fmt.Sprintf("a program saves %d round-trips and up to %d result tokens of context, not the work: AGENTS.md mandates this ritual", o.CallsSaved, o.TokensSaved)
	}
	if d.varying() {
		low, _ := d.interval()
		impact += fmt.Sprintf(" (%.0f%% of %d later steps used an earlier step's result, at least %.0f%% at 95%%)", 100*d.rate(), d.Measured, 100*low)
	}
	match := in.Library.best(m.steps)
	if match == nil {
		o.Verdict = VerdictCandidate
		o.Action = fmt.Sprintf("write a program for `%s`: run by hand %d times in %d sessions; %s", subject, m.occurrences, o.Sessions, impact)
		return o
	}
	match.Runs = programRuns[match.Name]
	o.Program = match
	if match.Basis == BasisBinding && 2*match.Covered > match.Steps {
		o.Verdict, o.Owner = VerdictAdopt, match.Scenario
		o.Action = fmt.Sprintf("adopt existing program `%s` (calls %d of %d steps) for `%s`: run by hand %d times in %d sessions against %d program runs; %s", match.Name, match.Covered, match.Steps, subject, m.occurrences, o.Sessions, match.Runs, impact)
		return o
	}
	o.Verdict = VerdictInvestigate
	o.Action = fmt.Sprintf("investigate whether program `%s` covers `%s` (%s match, calls %d of %d steps): run by hand %d times in %d sessions; %s; if it does not cover it, write a program for it", match.Name, subject, match.Basis, match.Covered, match.Steps, m.occurrences, o.Sessions, impact)
	return o
}

// dominantOwner is the scenario whose CLI most steps are; a sequence that
// loads a skill is a skill ritual owned by prompt-manager. Ties go to the
// owner met first.
func dominantOwner(steps []step) string {
	counts, first := map[string]int{}, []string{}
	for _, s := range steps {
		if strings.HasPrefix(s.path, "prompt-manager skill") {
			return "prompt-manager"
		}
		if counts[s.owner] == 0 {
			first = append(first, s.owner)
		}
		counts[s.owner]++
	}
	best := first[0]
	for _, owner := range first[1:] {
		if counts[owner] > counts[best] {
			best = owner
		}
	}
	return best
}

// programRunCommand is the command that runs a library program; the word
// after it names the program.
const programRunCommand = "program-runtime library run"

// programRunCounts counts the runs of each library program: calls of
// `program-runtime library run <program>`. A program named anywhere else (its
// help, `library get`, a skill read about it) was not run.
func programRunCounts(in Input) map[string]int {
	names := map[string]bool{}
	for _, p := range in.Library.Programs {
		names[p.Name] = true
	}
	counts := map[string]int{}
	for _, c := range in.Calls {
		if c.CommandPath != programRunCommand || c.Help {
			continue
		}
		words := strings.Fields(strings.TrimPrefix(c.Command, programRunCommand))
		for _, word := range words {
			if strings.HasPrefix(word, "-") {
				continue
			}
			if name := strings.Trim(word, `"'`); names[name] {
				counts[name]++
			}
			break
		}
	}
	return counts
}
