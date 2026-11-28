package usageattribution

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"agent-manager/internal/interventions"
	"agent-manager/internal/opportunities"
	"agent-manager/internal/toolfriction"
	"agent-manager/internal/usageforecast"
	"agent-manager/internal/usagelabel"
)

// Sources of an action.
const (
	SourceLever        = "lever"        // a quota lever over the current windows
	SourceTool         = "tool"         // a tool friction finding
	SourceOpportunity  = "opportunity"  // a repeated sequence or a bypass
	SourceIntervention = "intervention" // operator corrections the accuracy gate admits
)

// Owner kinds: who can make the change.
const (
	OwnerSkill    = "skill"
	OwnerScenario = "scenario"
	OwnerCommand  = "command"
	OwnerProgram  = "program"
	OwnerHarness  = "harness"
	OwnerSession  = "session"
	OwnerWorkload = "workload"
)

// Effect units: each action is stated in the unit of its harm.
const (
	UnitPoints        = "points"
	UnitCalls         = toolfriction.UnitCalls
	UnitMS            = toolfriction.UnitMS
	UnitTokens        = toolfriction.UnitTokens
	UnitInterventions = "interventions"
)

// Effect bases.
const (
	BasisWouldHaveSaved = "would have saved" // every part has finished: a post-mortem rate
	BasisProjected      = "projected"        // a still-running part continues at its rate
	BasisRange          = "range"            // unproven: anywhere from Value to Upper
	BasisPerTrim        = "per 10k tokens trimmed"
	BasisObserved       = "observed" // what happened in the window; removing the cause removes it
)

// titleRunes bounds a title: a label fits one line of the list.
const titleRunes = 60

// Effect is what an action changes, in its own unit. Upper is set for a range.
type Effect struct {
	Value, Upper float64
	Unit, Basis  string
}

// Drill is one session's part of an aggregate action: drill-down evidence,
// never a recommendation of its own.
type Drill struct {
	Kind, ID, Label, Detail string
	Value                   float64
	StillRunning            bool
	// n counts what unit (a format with one %d) names, shown when non-zero.
	n    int
	unit string
}

// Ref names one piece of evidence.
type Ref struct{ Kind, ID, Label string }

// Action is one owner-level change: who can make it, what to do, the
// evidence, its effect in its own unit and a score on the common scale.
type Action struct {
	Rank                   int
	Source, Kind           string
	OwnerKind, Owner       string
	Title, Text, Rationale string
	Effect                 Effect
	// Score ranks the action (0–1) and decides whether it is listed
	// (toolfriction.Score, toolfriction.MaterialScore). A quota lever is scored
	// by the share of the provider window's observed points it removes. Any
	// other action removes a share of its kind of harm (failed calls, shape
	// guesses, polled calls, call wall-clock, context tax, hand-run Vrooli CLI
	// calls, bypass calls) and touches a breadth of the window's runs; its score
	// is the geometric mean √(share × breadth), so one run's loop does not
	// outrank a defect that blocks many. Operator corrections have no run
	// count: their share alone.
	Score float64
	// ScoreBasis names the kind of harm and its total.
	ScoreBasis string
	// Recoverable is the part of a quota lever's saving still to be had before
	// its window resets; recovers is it per run, so levers on the same quota
	// combine without double counting (CombinedRecoverable).
	Recoverable float64
	recovers    []recovery
	// priced are every run a quota lever prices (Evidence names at most
	// maxDrill of them): levers of one window that price the same runs price
	// the same quota (noteOverlaps).
	priced     []string
	Confidence string
	Evidence   []Ref
	Drilldown  []Drill
	Verify     string
	// Fingerprints are the tool findings the action states (several once a
	// bypass and a tool finding merge): report or dismiss them with `usage
	// friction`.
	Fingerprints           []string
	Provider, Pool, Window string
	// TopRunShare is the largest single run's share of a tool or bypass
	// action's uncapped cost: near 1 means it is one run's loop, which the
	// effect counts at most toolfriction.RunCapFactor times the median run.
	TopRunShare float64
	// RootCause and Subs: a root-cause roll-up is one action naming the root
	// and listing each line under it. Guesses are what agents tried against a
	// hard-to-call command and what the CLI suggested.
	RootCause string
	Subs      []toolfriction.SubSignature
	Guesses   []toolfriction.Guess
	// harm is the total of the action's kind of harm, harmName its name; also
	// are the shares of further kinds a root-cause roll-up holds, stated
	// beside its score.
	harm     float64
	harmName string
	also     []toolfriction.Share
	// runs of windowRuns distinct runs in the window show the action's harm;
	// zero is unknown.
	runs, windowRuns int
	// scenario and words name the target of a tool or opportunity action, so
	// signals about the same target of the same owner merge into one action.
	scenario string
	words    []string
	merged   bool
}

// recovery is the part of a lever's recoverable points one run gives: run
// within session, whose projected future points bound what any lever can
// still recover from it.
type recovery struct {
	run, session   string
	points, future float64
}

// CombinedRecoverable is what the quota levers of window key can still
// recover before its reset, together, and how many levers add to it. Levers
// on the same quota are not added blindly: a run that several levers act on
// counts once, at the largest of their points for it, and a session gives at
// most its projected future points.
func CombinedRecoverable(actions []Action, key WindowKey) (float64, int) {
	byRun := map[string]recovery{}
	levers := 0
	for _, a := range actions {
		if a.Source != SourceLever || a.Provider != key.Provider || a.Pool != key.Pool || a.Window != key.Window || a.Recoverable <= 0 {
			continue
		}
		levers++
		for _, r := range a.recovers {
			if r.points > byRun[r.run].points {
				byRun[r.run] = r
			}
		}
	}
	bySession, future := map[string]float64{}, map[string]float64{}
	for _, r := range byRun {
		bySession[r.session] += r.points
		future[r.session] = r.future
	}
	var total float64
	for session, points := range bySession {
		total += min(points, future[session])
	}
	return total, levers
}

// affecting records how many of the window's runs show the action's harm.
func (a Action) affecting(runs, window int) Action {
	a.runs, a.windowRuns = runs, window
	return a
}

// against sets the kind of harm an action's effect is a share of.
func (a Action) against(total float64, name string) Action {
	a.harm, a.harmName = total, name
	return a
}

// Names reports whether the action's evidence or drill-down names id.
func (a Action) Names(id string) bool {
	for _, ref := range a.Evidence {
		if ref.ID == id {
			return true
		}
	}
	for _, d := range a.Drilldown {
		if d.ID == id {
			return true
		}
	}
	return false
}

// Scoring states how actions of different kinds share one ranking.
var Scoring = []string{
	"score: two scales, never interleaved. Quota levers come first, ranked on the quota scale: the share of their provider window's observed points they remove. Friction actions follow, ranked on the friction scale: √(share × breadth), share the part of their own kind of harm they remove (failed calls, shape guesses, polled calls, call wall-clock, context tax, hand-run Vrooli CLI calls or bypass calls), breadth the part of the window's runs that show it. The geometric mean keeps the scale of a share while a defect that touches many runs outranks one run's loop; operator corrections have no run count and score their share of operator messages alone. Every score names its scale and the absolute cost (calls, tokens, time or points) it is a share of",
	"assumption: kinds of harm weigh equally. No measured exchange rate between quota, failed calls, wall-clock and operator attention exists, so none is invented; each effect is shown in its own unit",
	fmt.Sprintf("an action scoring under %.0f%% (a range: whose top does) is counted, not listed; the friction list applies the same floor to the same score; a live session's own advice ranks after every aggregate, because it is one run's and ends with it", 100*toolfriction.MaterialScore),
	"a bypass and a tool finding about the same target of the same owner (a bypass and a shape-guessing finding of one scenario) are one action that lists both; it takes the larger score, as the signals overlap",
	"quota levers are not additive: levers of one window that price the same runs (a sub-agent restart, its starting context and its model) price the same quota there, so each lever names the higher-ranked levers it shares runs with; the window's outcome takes each run once, at its largest lever",
}

// Sources are the non-quota inputs of the merged list. Calls nil leaves the
// tool, skill and opportunity sources out; Library nil leaves opportunities
// out; Interventions nil leaves operator corrections out. Calls are at the
// current fact version; Stale counts what the window holds at older ones.
type Sources struct {
	Calls    []toolfriction.Call
	Stale    toolfriction.Stale
	Statuses map[string]ProblemStatus
	Library  *opportunities.Library
	// Outputs reads result text for the opportunities' dependency check.
	Outputs       opportunities.Outputs
	Interventions *interventions.Summary
	// Coverage are sentences the gatherer adds (a source it could not read).
	Coverage []string
}

// Report is one ranked, owner-level list of what to change.
type Report struct {
	Actions []Action
	// Below counts actions scoring under toolfriction.MaterialScore: counted,
	// not listed.
	Below int
	Rules []RuleStatus
	// Coverage says what each source could see, so a short list is bounded
	// silence, never "nothing to change".
	Coverage []string
	// Outcome is the named window's projection with its levers applied.
	Outcome *WindowOutcome
}

// WindowOutcome is a window's forecast with what its quota levers recover
// together taken out, and how many levers add to it.
type WindowOutcome struct {
	usageforecast.Outcome
	Levers int
}

// Recommend evaluates the quota levers and merges them with tool friction,
// opportunities and gated operator interventions into one list ranked by
// score.
func Recommend(in Inputs) Report {
	primary, sessions := prepare(in)
	report := Report{Coverage: append([]string(nil), in.Coverage...)}
	for _, result := range primary {
		report.Coverage = append(report.Coverage, fmt.Sprintf("%s %s window: %s attributed of %s observed over %d turns", result.Window.Provider, result.Window.Name, pts(result.Attributed()), pts(result.Observed), len(result.Turns)))
	}
	if len(primary) == 0 {
		report.Coverage = append(report.Coverage, "no provider window has used-percent readings: quota levers cannot be evaluated")
	}
	var actions []Action
	statuses := map[string]*RuleStatus{}
	levers := []func(Inputs, []*Result, map[*Result][]*sessionStats) ([]Action, RuleStatus){
		ruleModel, ruleModelMix,
		func(in Inputs, p []*Result, s map[*Result][]*sessionStats) ([]Action, RuleStatus) {
			return ruleGrowth(in, p, s, false)
		},
		func(in Inputs, p []*Result, s map[*Result][]*sessionStats) ([]Action, RuleStatus) {
			return ruleGrowth(in, p, s, true)
		},
		ruleFloor, ruleIdle, ruleImage,
	}
	for _, rule := range levers {
		rows, status := rule(in, primary, sessions)
		actions = append(actions, rows...)
		report.Rules = append(report.Rules, status)
	}
	tools := toolfriction.Analyze(toolfriction.Window{Calls: in.Calls, Stale: in.Stale})
	windowRuns := tools.Totals.Runs
	rows, status := skillWeights(in.Calls, tools.Totals)
	actions = append(actions, rows...)
	report.Rules = append(report.Rules, status)
	for i := range report.Rules {
		statuses[report.Rules[i].Rule] = &report.Rules[i]
	}
	if in.Calls != nil {
		report.Coverage = append(report.Coverage, "tools: "+tools.Coverage.Sentence())
		actions = append(actions, toolActions(tools, in.Statuses, skillPaths(in.Calls), windowRuns)...)
		if in.Library != nil {
			// Output shapes and skill reloads are stated by the output-weight
			// finding and R-skill: only sequences and bypasses are mined here.
			found := opportunities.Analyze(opportunities.Input{Calls: in.Calls, Stale: in.Stale, Library: *in.Library, Outputs: in.Outputs, Kinds: []string{opportunities.KindSequence, opportunities.KindBypass}})
			report.Coverage = append(report.Coverage, "opportunities: "+found.Coverage.Sentence())
			actions = append(actions, opportunityActions(found, tools.Totals, windowRuns)...)
		}
	}
	if in.Interventions != nil {
		rows, sentence := interventionActions(*in.Interventions)
		actions = append(actions, rows...)
		report.Coverage = append(report.Coverage, sentence)
	}
	report.Actions, report.Below = rank(actions, statuses)
	return report
}

// rank scores every action, counts those whose score is under the
// materiality floor, merges signals about one target, and orders the rest in
// groups (Action.group), each by score (a range, scored at its bottom, after
// every point estimate of the same score by its top): the quota levers, whose
// score is a share of a window's points, then the friction actions, whose
// score is √(share × breadth), so two scales are never interleaved as if
// comparable; each score names its scale. A live session's own advice ranks
// last: it is one run's, it ends with the run, and ranking it above the
// aggregates would turn the list into a queue of whichever session ran last.
func rank(actions []Action, rules map[string]*RuleStatus) ([]Action, int) {
	var kept []Action
	below := 0
	for _, a := range actions {
		a.Score = 0
		var top float64
		if a.harm > 0 {
			value := min(1, a.Effect.Value/a.harm)
			top = min(1, a.Effect.Upper/a.harm)
			a.Score = toolfriction.Score(value, a.runs, a.windowRuns)
			// Every score states the absolute cost it is a share of, in its unit.
			a.ScoreBasis = fmt.Sprintf("%s: %.1f%% (%s) of %s", a.scale(), 100*value, amount(a.Effect.Value, a.Effect.Unit), a.harmName)
			switch {
			case a.Effect.Basis == BasisRange:
				a.ScoreBasis = fmt.Sprintf("%s: 0–%.1f%% (0 to %s) of %s", a.scale(), 100*top, amount(a.Effect.Upper, a.Effect.Unit), a.harmName)
			case a.runs > 0 && a.windowRuns > 0:
				breadth := min(1, float64(a.runs)/float64(a.windowRuns))
				a.ScoreBasis = fmt.Sprintf("%s: %.1f%% = √(share %.1f%% (%s) of %s × breadth %.1f%%: %d of %d runs)", a.scale(), 100*a.Score, 100*value, amount(a.Effect.Value, a.Effect.Unit), a.harmName, 100*breadth, a.runs, a.windowRuns)
			}
		}
		for _, other := range a.also {
			a.ScoreBasis += "; also " + other.Sentence()
		}
		if a.TopRunShare >= toolfriction.ConcentratedShare && a.runs > 1 && a.harm > 0 {
			a.ScoreBasis += fmt.Sprintf("; %.0f%% of it in one run, so the effect counts each run at most %dx the median run", 100*a.TopRunShare, toolfriction.RunCapFactor)
		}
		a.Title = shorten(a.Title, titleRunes)
		if max(a.Score, top) < toolfriction.MaterialScore {
			below++
			continue
		}
		kept = append(kept, a)
	}
	sort.SliceStable(kept, func(i, j int) bool {
		a, b := kept[i], kept[j]
		if a.group() != b.group() {
			return a.group() < b.group()
		}
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Effect.Upper != b.Effect.Upper {
			return a.Effect.Upper > b.Effect.Upper
		}
		if a.Effect.Value != b.Effect.Value {
			return a.Effect.Value > b.Effect.Value
		}
		return a.Title < b.Title
	})
	kept = mergeSameTarget(kept)
	for i := range kept {
		kept[i].Rank = i + 1
		if status := rules[kept[i].Kind]; status != nil {
			status.Rows++
			status.Fired = true
		}
	}
	noteOverlaps(kept)
	for _, status := range rules {
		if !status.Fired && status.Reason == "" {
			status.Reason = fmt.Sprintf("it fired only under the materiality floor (score %.0f%%)", 100*toolfriction.MaterialScore)
		}
		if status.caveat != "" {
			status.Reason = strings.TrimPrefix(status.Reason+"; "+status.caveat, "; ")
		}
	}
	return kept, below
}

// noteOverlaps states on each quota lever the higher-ranked levers of the
// same window that price some of the same runs: their savings there are one
// quota, so the levers' figures do not add up (CombinedRecoverable counts
// each run once for the window's outcome).
func noteOverlaps(actions []Action) {
	for i := range actions {
		a := &actions[i]
		if a.Source != SourceLever || len(a.priced) == 0 {
			continue
		}
		mine := map[string]bool{}
		for _, id := range a.priced {
			mine[id] = true
		}
		var shared []string
		for _, b := range actions[:i] {
			if b.Source != SourceLever || b.Provider != a.Provider || b.Pool != a.Pool || b.Window != a.Window {
				continue
			}
			n := 0
			for _, id := range b.priced {
				if mine[id] {
					n++
				}
			}
			if n > 0 {
				shared = append(shared, fmt.Sprintf("#%d on %d", b.Rank, n))
			}
		}
		if len(shared) > 0 {
			a.Rationale = strings.TrimSpace(a.Rationale + fmt.Sprintf(" Not additive: it shares runs with %s of its %d runs, whose quota those levers price too; take the larger saving per run, not the sum (the window's outcome counts each run once).", strings.Join(shared, ", "), len(a.priced)))
		}
	}
}

// Score scales: what an action's score is a share of.
const (
	ScaleQuota    = "quota scale"    // share of a provider window's observed points
	ScaleFriction = "friction scale" // √(share of a kind of harm × share of runs touched)
)

// scale names the scale of the action's score.
func (a Action) scale() string {
	if a.Source == SourceLever {
		return ScaleQuota
	}
	return ScaleFriction
}

// group orders the list: aggregate quota levers, then friction actions, then
// a live session's own advice.
func (a Action) group() int {
	switch {
	case a.OwnerKind == OwnerSession:
		return 2
	case a.Source == SourceLever:
		return 0
	}
	return 1
}

// amount states an effect in its unit: "321 calls", "13.0 pts", "96.8M
// tokens", "8m 31s".
func amount(value float64, unit string) string {
	switch unit {
	case UnitPoints:
		return fmt.Sprintf("%.1f pts", value)
	case UnitTokens:
		return usagelabel.Tokens(int64(value)) + " tokens"
	case UnitMS:
		return usagelabel.Duration(int64(value))
	}
	return fmt.Sprintf("%.0f %s", value, unit)
}

var (
	wordSplit = regexp.MustCompile(`[^a-z]+`)
	// genericWords name no target: transport, verbs and views every scenario has.
	genericWords = map[string]bool{
		"http": true, "https": true, "curl": true, "localhost": true, "state": true, "table": true, "api": true, "run": true, "list": true,
		"show": true, "get": true, "status": true, "wait": true, "scenario": true, "health": true, "with": true, "from": true, "json": true,
		"data": true, "output": true, "that": true, "this": true, "path": true, "text": true, "type": true,
	}
)

// targetWords are the words of text that could name a target: lower-case,
// singular, four letters or more, not generic and not the owner's own name.
func targetWords(text, scenario string) []string {
	var out []string
	own := map[string]bool{}
	for _, w := range wordSplit.Split(strings.ToLower(scenario), -1) {
		own[strings.TrimSuffix(w, "s")] = true
	}
	for _, w := range wordSplit.Split(strings.ToLower(text), -1) {
		w = strings.TrimSuffix(w, "s")
		if len(w) >= 4 && !genericWords[w] && !own[w] && !slices.Contains(out, w) {
			out = append(out, w)
		}
	}
	return out
}

// mergeSameTarget folds an action into an earlier one of the same owner and
// target when it is the other side of a bypass (agents reach around the CLI)
// and a tool finding (the CLI is hard to call): a bypass of test-genie's
// artifacts and a finding that `test-genie runs artifacts` is guessed at are
// one change, to how artifacts are retrieved. The merged action keeps the
// larger score (the signals overlap, so they do not add) and lists both.
func mergeSameTarget(actions []Action) []Action {
	var out []Action
	for _, a := range actions {
		merged := false
		for i := range out {
			b := &out[i]
			// Only a bypass and a tool finding merge: two tool findings that share a
			// command group (effort board, effort signals) are different changes.
			if b.scenario == "" || b.scenario != a.scenario || !(b.Kind == opportunities.KindBypass && a.Source == SourceTool || a.Kind == opportunities.KindBypass && b.Source == SourceTool) {
				continue
			}
			var shared []string
			for _, w := range a.words {
				if slices.Contains(b.words, w) {
					shared = append(shared, w)
				}
			}
			if len(shared) == 0 {
				continue
			}
			if !b.merged {
				b.Rationale = fmt.Sprintf("Signals: [%s %s, %s] %s", b.Source, b.Kind, b.ScoreBasis, b.Rationale)
				b.merged = true
			}
			b.Title = shorten(fmt.Sprintf("Add or repair %s access in %s", shared[0], b.scenario), titleRunes)
			b.Text += fmt.Sprintf(" Also: %s", a.Text)
			b.Rationale += fmt.Sprintf(" | [%s %s, %s] %s", a.Source, a.Kind, a.ScoreBasis, a.Rationale)
			b.Evidence = appendRefs(b.Evidence, a.Evidence)
			b.Fingerprints = append(b.Fingerprints, a.Fingerprints...)
			b.TopRunShare = max(b.TopRunShare, a.TopRunShare)
			if len(b.Guesses) == 0 {
				b.Guesses = a.Guesses
			}
			if len(b.Subs) == 0 {
				b.RootCause, b.Subs = a.RootCause, a.Subs
			}
			b.Verify += " · " + a.Verify
			merged = true
			break
		}
		if !merged {
			out = append(out, a)
		}
	}
	return out
}

func appendRefs(have, more []Ref) []Ref {
	for _, r := range more {
		if !slices.Contains(have, r) {
			have = append(have, r)
		}
	}
	return have
}

// leverAction starts a quota lever stated in result's window: a share of the
// window's observed points.
func leverAction(result *Result, rule string, effect Effect) Action {
	a := Action{Source: SourceLever, Kind: rule, Effect: effect, Provider: result.Window.Provider, Pool: result.Window.Pool, Window: result.Window.Name}
	return a.against(result.Observed, fmt.Sprintf("the %s %s window's %.1f observed points", result.Window.Provider, result.Window.Name, result.Observed))
}

// skillPaths are the command paths that load skills: their output weight is
// R-skill's, per skill.
func skillPaths(calls []toolfriction.Call) map[string]bool {
	out := map[string]bool{}
	for _, c := range calls {
		if c.LoadsSkill() {
			out[c.CommandPath] = true
		}
	}
	return out
}

// toolActions turns the tool findings into actions. Bypasses are stated by the
// opportunities source, which names what agents reach, and skill loads' output
// weight by R-skill; dismissed findings are left out. Each action's effect is
// its finding's largest share, over that kind's base (toolfriction.Totals.Base);
// a root-cause roll-up states its other kinds' shares beside the score.
func toolActions(report toolfriction.Report, statuses map[string]ProblemStatus, skills map[string]bool, windowRuns int) []Action {
	t := report.Totals
	var out []Action
	for _, f := range report.Findings {
		status := statuses[f.Fingerprint]
		if f.Kind == toolfriction.KindBypass || status.Status == StatusDismissed || f.Kind == toolfriction.KindOutputWeight && skills[f.CommandPath] {
			continue
		}
		primary := f.Shares[0]
		total, base := t.Base(primary.Kind)
		a := Action{
			Source: SourceTool, Kind: f.Kind, Fingerprints: []string{f.Fingerprint}, OwnerKind: OwnerScenario, Owner: f.Owner, Rationale: f.Description + ".", Confidence: "high",
			Effect: Effect{Value: float64(primary.Cost), Unit: primary.Unit, Basis: BasisObserved}, TopRunShare: f.TopRunShare, RootCause: f.RootCause, Subs: f.Subs, Guesses: f.Guesses,
		}
		a = a.against(float64(total), base).affecting(f.HarmRuns, windowRuns)
		a.also = f.Shares[1:]
		a.scenario, a.words = f.Owner, targetWords(f.CommandPath, f.Owner)
		subject := f.Owner
		// A finding about one command of its owner's CLI is that command's; one
		// a line attributes to another scenario (a park refusal printed inside
		// `test-genie runs wait`) stays with that scenario.
		if f.CommandPath != "" && strings.HasPrefix(f.CommandPath+" ", f.Owner+" ") {
			a.OwnerKind, a.Owner, subject = OwnerCommand, f.CommandPath, "`"+f.CommandPath+"`"
		}
		switch f.Kind {
		case toolfriction.KindFailure:
			a.Title, a.Text = "Fix "+subject+": "+f.ErrorLine, fmt.Sprintf("Fix %s: it failed %d times in %d runs with %q", subject, f.Calls, f.Runs, f.ErrorLine)
		case toolfriction.KindDegraded:
			where := subject
			if f.CommandPath != "" {
				where = "`" + f.CommandPath + "`"
			}
			a.Title, a.Text = "Fix "+subject+" degraded: "+f.ErrorLine, fmt.Sprintf("Fix what makes %s report %q (%d calls in %d runs)", where, f.ErrorLine, f.Calls, f.Runs)
			if toolfriction.UnprintedNotes(f.ErrorLine) {
				// The promised notes are missing: printing them is the change.
				a.Title = "Make " + f.Owner + " print the degraded-provider notes it promises"
				a.Text = fmt.Sprintf("Make %s print the notes its summary promises (\"some providers degraded — see notes below\"): the output read never named the degraded providers (%d calls in %d runs)", f.Owner, f.Calls, f.Runs)
			}
		case toolfriction.KindShapeGuess:
			fixes := guessFixes(f.Guesses)
			switch {
			case f.HelpLookups > f.WrongShape:
				// Mostly help lookups: the command works when called, but agents
				// do not remember how; where they are told to run it should say.
				a.Title = "Make " + subject + " usage memorable"
				a.Text = fmt.Sprintf("%s usage is not memorable at the call site (AGENTS.md/help): agents looked up its help %d times and called it with a wrong shape %d times; state its usual call where agents are told to run it and lead its help with it", subject, f.HelpLookups, f.WrongShape)
				if fixes != "" {
					a.Text += "; also " + fixes
				}
			case fixes != "":
				a.Title, a.Text = "Make "+subject+" easier to call", fmt.Sprintf("Make %s easier to call (%d wrong-shape calls, %d help lookups): %s", subject, f.WrongShape, f.HelpLookups, fixes)
			default:
				a.Title, a.Text = "Make "+subject+" easier to call", fmt.Sprintf("Make %s easier to call: name the right command or flag in its errors and help (%d wrong-shape calls, %d help lookups)", subject, f.WrongShape, f.HelpLookups)
			}
		case toolfriction.KindOutputWeight:
			a.Confidence = "medium"
			a.Title, a.Text = "Bound "+subject+" output", fmt.Sprintf("Bound %s output: a short summary by default, details behind flags", subject)
			if f.PrintedTokens > 2*f.ReadTokens {
				a.Text += fmt.Sprintf("; its largest call printed about %s tokens, of which the harness showed the model %s", usagelabel.Tokens(f.PrintedTokens), usagelabel.Tokens(f.ReadTokens))
			}
		case toolfriction.KindPolling:
			a.Confidence = "medium"
			a.Title, a.Text = "Give "+subject+" a blocking wait", fmt.Sprintf("Give %s a blocking wait (or event) so agents stop polling it (%d polled calls)", subject, primary.Cost)
			if f.Steered > 0 {
				// The wait exists: what is missing is agents knowing it, so the
				// change belongs to whatever sends them to the polled command. It
				// claims only the polls the wait stands in for.
				wait := "`" + f.Blocking[0] + "`"
				a.OwnerKind, a.Owner = OwnerSkill, "skills/docs and "+subject+" help"
				a.Title = "Steer agents from " + subject + " to " + wait
				a.Effect.Value = float64(f.Steered)
				a = a.affecting(f.SteeredRuns, windowRuns)
				a.Text = fmt.Sprintf("Agents poll %s where %s blocks: name it in the skills and docs that send agents to %s, and in its help and not-done output (%d of %d polled calls)", subject, wait, subject, f.Steered, primary.Cost)
				if f.Steered < primary.Cost {
					a.Text += fmt.Sprintf("; no blocking sibling is shown to apply to the other %d", primary.Cost-f.Steered)
				}
			}
		case toolfriction.KindSlow:
			a.Title, a.Text = "Make "+subject+" faster", fmt.Sprintf("Make %s faster or give it a cheaper status/summary: %s beyond its budget over %d calls", subject, usagelabel.Duration(primary.Cost), f.Calls)
		default:
			continue
		}
		if len(f.Subs) > 1 && f.RootCause != "" {
			// One root cause, one action: it names the cause and lists its lines.
			a.Title = "Fix the " + f.RootCause + " root cause"
			a.Text = fmt.Sprintf("Fix the %s broker or refresh that %d lines trace to (%d calls in %d runs): %s", f.RootCause, len(f.Subs), f.Calls, f.Runs, subSummary(f.Subs))
		}
		if status.Status == StatusReported {
			a.Rationale += " Reported as " + status.BugID + "."
		}
		for _, id := range f.RunIDs {
			a.Evidence = append(a.Evidence, Ref{Kind: "run", ID: id})
		}
		a.Verify = fmt.Sprintf("after the fix, `agent-manager usage friction --since <date of the fix>` should no longer list finding %s", f.Fingerprint)
		out = append(out, a)
	}
	return out
}

// guessFixes states the alias to add or the help to fix for each guess.
func guessFixes(guesses []toolfriction.Guess) string {
	parts := make([]string, 0, len(guesses))
	for _, g := range guesses {
		if g.Suggestion != "" {
			parts = append(parts, fmt.Sprintf("add alias `%s`→`%s` (x%d)", g.Token, g.Suggestion, g.Calls))
		} else {
			parts = append(parts, fmt.Sprintf("fix help for `%s` (x%d)", g.Token, g.Calls))
		}
	}
	return strings.Join(parts, "; ")
}

// subSummary lists the lines of a root-cause roll-up, the most frequent first.
func subSummary(subs []toolfriction.SubSignature) string {
	parts := make([]string, 0, len(subs))
	for _, s := range subs {
		parts = append(parts, fmt.Sprintf("%s %q in %d calls", s.Kind, shorten(s.Signature, titleRunes), s.Calls))
	}
	return strings.Join(parts, "; ")
}

// opportunityActions states repeated sequences and bypasses. Output shapes
// and skill reloads are folded elsewhere (the output-weight finding names the
// JSON paths; R-skill counts reloads), so one behaviour is listed once.
func opportunityActions(report opportunities.Report, t toolfriction.Totals, windowRuns int) []Action {
	var out []Action
	for _, o := range report.Opportunities {
		a := Action{
			Source: SourceOpportunity, Kind: o.Kind, OwnerKind: OwnerScenario, Owner: o.Owner, Text: o.Action, Verify: o.Verify, Confidence: "medium",
			Effect: Effect{Value: float64(o.CallsSaved), Unit: UnitCalls, Basis: BasisObserved},
		}
		total, harm := int64(report.Coverage.VrooliCalls), fmt.Sprintf("the window's %d Vrooli CLI calls", report.Coverage.VrooliCalls)
		switch {
		case o.Kind == opportunities.KindBypass:
			total, harm = t.Base(toolfriction.KindBypass)
		case o.Mandated:
			// The ritual's work stays; only its round-trips go.
			harm = fmt.Sprintf("the window's %d Vrooli CLI round-trips (the mandated ritual's work stays)", report.Coverage.VrooliCalls)
		}
		a = a.against(float64(total), harm).affecting(o.Sessions, windowRuns)
		a.Rationale = fmt.Sprintf("%d occurrences in %d sessions; %s = %s", o.Occurrences, o.Sessions, o.SavedLabel, o.SavedBasis)
		if o.Kind == opportunities.KindSequence {
			a.Rationale = "program-library verdict: " + programVerdict(o) + "; " + a.Rationale
			if o.TokensSaved > 0 {
				a.Rationale += fmt.Sprintf("; up to %s result tokens in context", usagelabel.Tokens(o.TokensSaved))
			}
			if o.TokensUnknown > 0 {
				a.Rationale += fmt.Sprintf(" (token counts unknown for %d calls)", o.TokensUnknown)
			}
		}
		if len(o.Details) > 0 {
			a.Rationale += "; reached: " + strings.Join(o.Details, ", ")
		}
		if len(o.Examples) > 0 {
			a.Rationale += "; e.g. `" + strings.Join(o.Examples, "`, `") + "`"
		}
		switch {
		case o.Kind == opportunities.KindBypass:
			a.Title = "Add CLI/RPC for " + o.Subject
			a.TopRunShare = o.TopRunShare
			a.scenario, a.words = o.Owner, targetWords(strings.Join(o.Details, " ")+" "+strings.Join(o.Examples, " "), o.Owner)
		case o.Kind != opportunities.KindSequence:
			continue
		case o.Verdict == opportunities.VerdictIndependent || o.Verdict == opportunities.VerdictUnmeasured || o.Verdict == opportunities.VerdictUndetermined:
			continue // a program would save nothing it can be shown to save
		case o.Verdict == opportunities.VerdictAdopt:
			a.OwnerKind, a.Owner, a.Title = OwnerProgram, o.Program.Name, "Adopt program "+o.Program.Name
		case o.Verdict == opportunities.VerdictInvestigate:
			a.Confidence, a.Title = "low", "Check whether "+o.Program.Name+" fits a repeated sequence"
		case o.Verdict == opportunities.VerdictUnknown:
			a.Confidence, a.Title = "low", "Check the library for "+o.Subject
		default:
			a.Title = "Write a program for " + o.Subject
		}
		for _, id := range o.RunIDs {
			a.Evidence = append(a.Evidence, Ref{Kind: "run", ID: id})
		}
		out = append(out, a)
	}
	return out
}

// programVerdict is the program-library verdict of a sequence in words: adopt
// an existing program, investigate a related one, write one, or unknown.
func programVerdict(o opportunities.Opportunity) string {
	p := o.Program
	switch o.Verdict {
	case opportunities.VerdictAdopt:
		return fmt.Sprintf("adopt `%s` (calls %d of %d steps; %d program runs against %d by hand)", p.Name, p.Covered, p.Steps, p.Runs, o.Occurrences)
	case opportunities.VerdictInvestigate:
		return fmt.Sprintf("investigate `%s` (%s match, calls %d of %d steps; %d program runs against %d by hand)", p.Name, p.Basis, p.Covered, p.Steps, p.Runs, o.Occurrences)
	case opportunities.VerdictUnknown:
		return "unknown (the program library could not be read)"
	}
	return "write a program (none in the library covers it)"
}

// skillWeights is R-skill: per skill, the context its loads put in front of
// later requests (result tokens × requests that re-read them before a
// compaction), for skills loaded in at least opportunities.MinSessions runs.
// A read whose output went through a filter loaded nothing
// (toolfriction.Call.LoadsSkill).
// Its loads repeated while still in context are counted only in runs whose
// segments are known (Call.SegmentsKnown); elsewhere residency is unknown.
func skillWeights(calls []toolfriction.Call, t toolfriction.Totals) ([]Action, RuleStatus) {
	status := RuleStatus{Rule: RuleSkill}
	type agg struct {
		loads, reloads, unknown int
		unknownResidency        int
		tokens, tax             int64
		runs                    []string
	}
	unknown := 0
	skills := map[string]*agg{}
	var names []string
	seen := map[[3]string]bool{}
	for _, c := range calls {
		if !c.LoadsSkill() {
			continue
		}
		a := skills[c.SkillID]
		if a == nil {
			a = &agg{}
			skills[c.SkillID] = a
			names = append(names, c.SkillID)
		}
		key := [3]string{c.RunID, c.SkillID, fmt.Sprint(c.ResidencySegment)}
		if seen[key] && c.SegmentsKnown() {
			a.reloads++
		} else if seen[key] {
			a.unknownResidency++
		}
		seen[key] = true
		if !slices.Contains(a.runs, c.RunID) {
			a.runs = append(a.runs, c.RunID)
		}
		a.loads++
		if !c.TokensKnown() {
			a.unknown++
			unknown++
			continue
		}
		a.tokens += c.ResultTokens
		a.tax += c.ResultTokens * c.Residency
	}
	if unknown > 0 {
		// Unknown loads count no context tax: a skill under the floor may be
		// there because its loads' tokens were not recorded, not because they
		// are light.
		status.caveat = fmt.Sprintf("token counts unknown for %d loads, which count no context tax", unknown)
	}
	sort.Strings(names)
	var out []Action
	for _, name := range names {
		status.Candidates++
		a := skills[name]
		if len(a.runs) < opportunities.MinSessions {
			continue
		}
		row := Action{
			Source: SourceTool, Kind: RuleSkill, OwnerKind: OwnerSkill, Owner: name, Confidence: "medium",
			Effect: Effect{Value: float64(a.tax), Unit: UnitTokens, Basis: BasisObserved},
		}
		row = row.against(float64(t.ContextTax), "the window's context tax ("+usagelabel.Tokens(t.ContextTax)+" re-read tokens)").affecting(len(a.runs), t.Runs)
		row.Title = "Shrink skill " + name
		row.Text = fmt.Sprintf("Shrink skill `%s` or split it, with a short digest for re-reads", name)
		perLoad := "tokens a load unknown"
		if known := a.loads - a.unknown; known > 0 {
			perLoad = fmt.Sprintf("≈%d tokens a load", a.tokens/int64(known))
		}
		reloads := fmt.Sprintf("%d loads repeated it while it was still in context", a.reloads)
		switch {
		case a.unknownResidency > 0 && a.reloads == 0:
			reloads = fmt.Sprintf("%d loads repeated it in runs whose facts record no compactions (residency unknown)", a.unknownResidency)
		case a.unknownResidency > 0:
			reloads += fmt.Sprintf(", and %d more repeated it in runs whose facts record no compactions (residency unknown)", a.unknownResidency)
		}
		row.Rationale = fmt.Sprintf("Loaded %d times in %d sessions, %s; later requests re-read ≈%d tokens of it before a compaction; %s.", a.loads, len(a.runs), perLoad, a.tax, reloads)
		if a.unknown > 0 {
			row.Rationale += fmt.Sprintf(" Token counts unknown for %d loads.", a.unknown)
		}
		for _, id := range a.runs[:min(len(a.runs), maxDrill)] {
			row.Evidence = append(row.Evidence, Ref{Kind: "run", ID: id})
		}
		row.Verify = fmt.Sprintf("after the change, R-skill for `%s` in `agent-manager usage recommendations` should shrink", name)
		out = append(out, row)
	}
	if len(out) == 0 {
		status.Reason = fmt.Sprintf("no skill was loaded in %d+ sessions in the tool-friction window", opportunities.MinSessions)
	}
	return out, status
}

// interventionActions states the operator corrections the accuracy gate
// admits, by owning skill or doc; with none admitted it says so with the
// classification coverage (cannot tell, not none).
func interventionActions(s interventions.Summary) ([]Action, string) {
	var out []Action
	add := func(kind, owner, confidence string, shown int64, work interventions.Work) {
		a := Action{
			Source: SourceIntervention, Kind: kind, OwnerKind: OwnerSkill, Owner: owner, Confidence: confidence,
			Effect: Effect{Value: float64(shown), Unit: UnitInterventions, Basis: BasisObserved},
		}
		a = a.against(float64(s.Classified+s.Unclassified), fmt.Sprintf("the window's %d operator messages", s.Classified+s.Unclassified))
		a.Title = fmt.Sprintf("Fewer %s corrections", strings.ReplaceAll(kind, "_", " "))
		a.Text = fmt.Sprintf("Update %s so agents need fewer %s corrections (%d this window)", owner, strings.ReplaceAll(kind, "_", " "), shown)
		a.Rationale = fmt.Sprintf("%d operator messages corrected agents after %d requests and %d tool calls of work.", shown, work.Requests, work.ToolCalls)
		a.Verify = "`agent-manager measures get usage.interventions` per agent-hour should fall for the category"
		out = append(out, a)
	}
	for _, c := range s.Categories {
		if c.Shown > 0 {
			add(string(c.Category), c.Category.Owner(), "medium", c.Shown, c.Work)
		}
	}
	if s.Pooled.Shown > 0 {
		add(string(interventions.PooledFriction), "agent behaviour (category unproven)", "low", s.Pooled.Shown, s.Pooled.Work)
	}
	coverage := "unknown"
	if value := s.Coverage(); value != nil {
		coverage = fmt.Sprintf("%.0f%%", 100**value)
	}
	if len(out) == 0 {
		return nil, fmt.Sprintf("operator interventions: %d messages classified (%s coverage); no category has passed the accuracy gate, so none is recommended: cannot tell, not none", s.Classified, coverage)
	}
	return out, fmt.Sprintf("operator interventions: %d messages classified (%s coverage); only gate-proven categories are listed", s.Classified, coverage)
}

// shortLabel is a session's label for a list: its workload in words, else its
// title. A label longer than a title (titleRunes) is a raw first prompt, never
// shown: the session is named by its runner and short id instead.
func shortLabel(run Run, key string) string {
	if run.Workload != "" {
		return shorten(workloadLabel(run.Workload, run.Workload), titleRunes)
	}
	label := strings.Join(strings.Fields(shortenUUIDs(run.SessionLabel)), " ")
	if label != "" && len([]rune(label)) <= titleRunes {
		return label
	}
	name := "session "
	if run.Runner != "" {
		name = run.Runner + " session "
	}
	return name + ShortID(strings.TrimPrefix(key, "web-console:"))
}

// shorten collapses whitespace and cuts text to n runes with an ellipsis.
func shorten(text string, n int) string {
	text = strings.Join(strings.Fields(text), " ")
	if runes := []rune(text); len(runes) > n {
		return string(runes[:n-1]) + "…"
	}
	return text
}
