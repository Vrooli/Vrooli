package toolfriction

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"agent-manager/internal/runsignal"
	"agent-manager/internal/tokenaccounting"
	"agent-manager/internal/usagelabel"
)

// Finding kinds: what an owner can fix.
const (
	KindFailure      = "failure"
	KindDegraded     = "degraded"
	KindShapeGuess   = "shape-guessing"
	KindOutputWeight = "output-weight"
	KindPolling      = "polling"
	KindSlow         = "slow"
	KindBypass       = "bypass"
	// HarmWallClock is not a finding kind but a kind of harm a degradation
	// holds beside its calls: the time agents spent in the degraded calls
	// (`waiting inline instead` of parking).
	HarmWallClock = "wall-clock"
)

// Units a finding's cost is stated in: the unit of its harm.
const (
	UnitCalls  = "calls"
	UnitMS     = "ms"
	UnitTokens = "tokens"
)

// Thresholds below which a pattern is not yet a finding.
const (
	minClusterCalls = 3
	minShapeGuesses = 3
	minPolledCalls  = 9
	minWeightCalls  = 3
	heavyP90Tokens  = 8_000
	heavyMaxTokens  = 40_000
	// heavyTaxShare: a command whose results hold this share of the window's
	// context tax is heavy however small each call is (a skill read of 4k
	// tokens, thousands of times) or however hard the harness cut it (Codex
	// shows the model about 10k tokens of a 1 MiB output).
	// A small window has no such share to speak of: it also needs heavyTaxFloor
	// re-read tokens.
	heavyTaxShare       = 0.01
	heavyTaxFloor       = 5_000_000
	maxEvidenceRuns     = 5
	maxEvidenceCommands = 3
)

// Cluster is every failure and degradation notice of Vrooli CLIs that
// normalises to one error line, across command paths. Lines that share a root
// cause (see rootCause) are one cluster whose Subs list each line.
type Cluster struct {
	Signature, Kind string
	// Root is the shared root cause the cluster was rolled up by, "" when it
	// is a single line; it, not the dominant line, identifies the cluster.
	Root               string
	Owner, OwnerKind   string
	CommandPaths       []string
	Calls, Runs        int
	DurationMS         int64
	Line               string
	Subs               []SubSignature
	Examples, RunIDs   []string
	commands, runCalls map[string]int
	// kindRuns is each kind's calls per run: a roll-up's failures and
	// notices are shares of different totals. waitRuns is the time of each
	// run's degraded calls that were not blocking waits; heldMS is the time
	// of those that were: a wait lasts as long as its target whatever the
	// degradation, so that time is held inline, not lost.
	kindRuns map[string]map[string]int
	waitRuns map[string]int64
	heldMS   int64
	subs     map[string]*subCluster
	seen     map[string]bool
}

// SubSignature is one error or notice line inside a rolled-up cluster.
type SubSignature struct {
	Kind, Signature, Line string
	Calls                 int
}

type subCluster struct {
	SubSignature
	owners map[string]int
}

// Finding is one owner-level problem: who can fix it, what it is in one plain
// sentence, how much it cost in the unit of its harm, and the evidence.
type Finding struct {
	Fingerprint, Kind, Owner, OwnerKind, CommandPath string
	Description                                      string
	// Calls and Runs count everything the finding saw (every call of a
	// profiled command path), HarmRuns the runs that show its harm, the
	// breadth its score weighs. Cost is the harm in Unit with each run's
	// contribution capped (see capped), and TopRunShare the largest single
	// run's share of the uncapped cost: near 1 means the finding is one run,
	// not a pattern.
	Calls, Runs, HarmRuns int
	Unit                  string
	Cost                  int64
	TopRunShare           float64
	// Shares state the cost of each kind of harm the finding holds against that
	// kind's total (Totals.Base): one for most findings, one per kind for a
	// root-cause roll-up of failures and notices, largest first. CostShare is
	// the largest: findings in different units rank by how much of their own
	// harm each holds.
	Shares    []Share
	CostShare float64
	// Score ranks the finding and decides whether it is listed (Score,
	// MaterialScore): its largest share weighed by the share of the window's
	// runs that show it (HarmRuns).
	Score     float64
	ErrorLine string
	// RootCause names the dependency whose broker or refresh a rolled-up
	// finding's Subs share; it, not ErrorLine, identifies the finding.
	RootCause string
	Subs      []SubSignature
	Guesses   []Guess
	// Shape guessing: calls that failed for a wrong command, flag or argument,
	// and help lookups counted as guesses.
	WrongShape, HelpLookups int
	// Polling: the commands declared blocking beside the polled one that stood
	// in for at least one polled call (runsignal.BlockingSiblings, steering),
	// and Steered the polled calls they stand in for, each run capped as Cost
	// is, in SteeredRuns: agents could call them instead there, and only there.
	Blocking    []string
	Steered     int64
	SteeredRuns int
	// Output weight: the largest call printed PrintedTokens (estimated from
	// bytes) of which the harness showed the model ReadTokens.
	PrintedTokens, ReadTokens int64
	Examples, RunIDs          []string
}

// Share is a finding's cost of one kind of harm over that kind's total, with
// the total named (Totals.Base): "12.4% of the window's 812 Vrooli CLI failed
// calls". Cost is capped per run (Capped) and Uncapped what it was before.
type Share struct {
	Kind, Unit     string
	Cost, Uncapped int64
	Share          float64
	Base           string
}

// Sentence states a share with its base, and the cap when it cut the cost:
// "failure 91 of 95 calls (each run at most 10x the median run's) = 13.6% of
// the window's 668 Vrooli CLI failed calls". The friction list, the actions
// and the bug report all say it this way.
func (s Share) Sentence() string {
	cost := figure(s.Cost, s.Unit) + unitWord(s.Unit)
	if s.Uncapped > s.Cost {
		cost = fmt.Sprintf("%s of %s%s (each run at most %dx the median run's)", figure(s.Cost, s.Unit), figure(s.Uncapped, s.Unit), unitWord(s.Unit), RunCapFactor)
	}
	return fmt.Sprintf("%s %s = %.1f%% of %s", s.Kind, cost, 100*s.Share, s.Base)
}

// Guess is a command, flag or subcommand agents tried that does not exist,
// and what the CLI suggested instead: the alias to add or the help to fix.
type Guess struct {
	Token, Suggestion string
	Calls             int
}

// rootCause is the dependency whose broker or refresh a line says failed, ""
// when it names none: the word before "broker" or "refresh" in "identity
// broker did not answer" or "identity_refresh_unavailable". Failures and
// notices that share it have one cause, so they are one cluster.
func rootCause(line string) string {
	if m := rootWord.FindStringSubmatch(strings.ToLower(line)); m != nil {
		return m[1]
	}
	return ""
}

var rootWord = regexp.MustCompile(`\b([a-z]+)[ _](?:broker|refresh)(?:\b|_)`)

// silent is the signature of a failure that printed no error line, only its
// exit status: it says nothing of the cause, so failures of different
// commands that share it are not one problem.
func silent(signature string) bool {
	return signature == noErrorMessage || strings.HasPrefix(signature, "exit code ")
}

// clusters groups the failures and notices of Vrooli CLI calls by normalised
// line, and rolls lines with a common root cause into one cluster. A line that
// starts with a Vrooli CLI's name belongs to that CLI; a line seen across
// several CLIs comes from what they share, the control plane. A silent failure
// (no error line) clusters per command path: the command is the offender.
func clusters(calls []Call) []Cluster {
	byKey := map[string]*Cluster{}
	add := func(c Call, kind, signature, line string) {
		if silent(signature) {
			signature = c.CommandPath + ": " + signature
		}
		key, root := kind+"\x00"+signature, rootCause(signature+" "+line)
		if root != "" {
			key = "root\x00" + root
		}
		cl := byKey[key]
		if cl == nil {
			cl = &Cluster{Root: root, commands: map[string]int{}, runCalls: map[string]int{}, kindRuns: map[string]map[string]int{}, waitRuns: map[string]int64{}, subs: map[string]*subCluster{}, seen: map[string]bool{}}
			byKey[key] = cl
		}
		sub := cl.subs[kind+"\x00"+signature]
		if sub == nil {
			sub = &subCluster{SubSignature: SubSignature{Kind: kind, Signature: signature, Line: line}, owners: map[string]int{}}
			cl.subs[kind+"\x00"+signature] = sub
		}
		// Every count is of calls: a line printed twice by one call (two
		// segments of one command path) is one call of that line, and a call
		// that both failed and reported a notice of one root cause is one
		// affected call, and one of each kind.
		callKey := c.RunID + "\x00" + c.CallEventID + "\x00" + c.CommandPath
		if subKey := callKey + "\x00" + kind + "\x00" + signature; !cl.seen[subKey] {
			cl.seen[subKey] = true
			sub.Calls++
			sub.owners[c.Owner]++
		}
		if kindKey := callKey + "\x00" + kind; !cl.seen[kindKey] {
			cl.seen[kindKey] = true
			if cl.kindRuns[kind] == nil {
				cl.kindRuns[kind] = map[string]int{}
			}
			cl.kindRuns[kind][c.RunID]++
			switch {
			case kind != KindDegraded || !c.HasDuration:
			case blocksUntilDone(c.CommandPath, strings.Fields(c.Command)):
				cl.heldMS += c.DurationMS
			default:
				cl.waitRuns[c.RunID] += c.DurationMS
			}
		}
		if !cl.seen[callKey] {
			cl.seen[callKey] = true
			cl.Calls++
			cl.commands[c.CommandPath]++
			cl.runCalls[c.RunID]++
			cl.Examples = addExample(cl.Examples, c)
			cl.DurationMS += c.DurationMS
		}
	}
	for _, c := range calls {
		if c.Owner == "" || c.CommandPath == "" {
			continue
		}
		// A wrong-shape failure belongs to its command's shape-guessing finding.
		if c.Signature != "" && !c.ShapeGuess {
			add(c, KindFailure, c.Signature, c.ErrorLine)
		}
		if c.Notice != "" {
			add(c, KindDegraded, c.Notice, c.NoticeLine)
		}
	}
	out := make([]Cluster, 0, len(byKey))
	for _, cl := range byKey {
		cl.Runs = len(cl.runCalls)
		cl.CommandPaths = topKeys(cl.commands, len(cl.commands))
		cl.RunIDs = topKeys(cl.runCalls, maxEvidenceRuns)
		var dominant *subCluster
		for _, sub := range cl.subs {
			cl.Subs = append(cl.Subs, sub.SubSignature)
			if sub.Kind == KindFailure {
				cl.Kind = KindFailure
			}
			if dominant == nil || sub.Calls > dominant.Calls || sub.Calls == dominant.Calls && sub.Signature < dominant.Signature {
				dominant = sub
			}
		}
		sort.Slice(cl.Subs, func(i, j int) bool {
			if cl.Subs[i].Calls != cl.Subs[j].Calls {
				return cl.Subs[i].Calls > cl.Subs[j].Calls
			}
			return cl.Subs[i].Signature < cl.Subs[j].Signature
		})
		if cl.Kind == "" {
			cl.Kind = KindDegraded
		}
		cl.Signature, cl.Line = dominant.Signature, dominant.Line
		cl.Owner = dominant.owner()
		if cl.Root != "" {
			if named := cl.namedRootOwner(); named != "" {
				cl.Owner = named
			}
		}
		cl.OwnerKind = ownerKind(cl.Owner)
		out = append(out, *cl)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Calls != out[j].Calls {
			return out[i].Calls > out[j].Calls
		}
		return out[i].Signature < out[j].Signature
	})
	return out
}

// namedRootOwner is the scenario a root-cause roll-up's lines name (by most
// calls), "" when none does. The lines that name the broker say whose it is:
// "the Agent Manager identity broker did not answer" and "agent-manager park
// unavailable (… identity_refresh_unavailable)" put the identity roll-up with
// agent-manager, whose broker (identity.sock) refuses the refresh, even though
// its bare code is printed by a dozen CLIs through the shared client.
func (cl *Cluster) namedRootOwner() string {
	calls := map[string]int{}
	best := ""
	for _, sub := range cl.subs {
		if named := namedOwner(sub.Line); named != "" {
			calls[named] += sub.Calls
			if best == "" || calls[named] > calls[best] || calls[named] == calls[best] && named < best {
				best = named
			}
		}
	}
	return best
}

// owner names who can fix a cluster: the CLI its line names, else the CLI
// that printed most of it (more than half of its calls: a few misattributed
// calls of a compound command do not move it), else the control plane that the
// CLIs share: a line printed by a dozen CLIs through a shared client with none
// above half and no scenario named.
func (sub *subCluster) owner() string {
	if named := namedOwner(sub.Line); named != "" {
		return named
	}
	for owner, calls := range sub.owners {
		if 2*calls > sub.Calls {
			return owner
		}
	}
	return controlPlane
}

// namedOwner is the Vrooli scenario CLI a line names as its subject ("test-genie
// admission is saturated", "agent-manager park unavailable"), "" when it names
// none. Quoted text (a query, a title) is not the subject, and the control
// plane's name is not either: it prints its own name in usage hints.
func namedOwner(line string) string {
	line = quoted.ReplaceAllString(line, " ")
	// A scenario's display name ("Agent Manager") names its CLI.
	for _, name := range displayName.FindAllString(line, -1) {
		cli := strings.ToLower(strings.Join(strings.Fields(name), "-"))
		if resolution := runsignal.ResolveCatalog(cli); resolution.State == "resolved" && resolution.Owner == cli {
			return cli
		}
	}
	for _, word := range strings.Fields(line) {
		word = strings.Trim(word, ":;,.()[]")
		if word == "" || word == controlPlane || !scenarioWord.MatchString(word) {
			continue
		}
		if resolution := runsignal.ResolveCatalog(word); resolution.State == "resolved" && resolution.Owner == word {
			return word
		}
	}
	return ""
}

// displayName is a run of two or more capitalised words, shaped like a
// scenario's display name.
var displayName = regexp.MustCompile(`\b[A-Z][a-z]+(?: [A-Z][a-z]+)+\b`)

// scenarioWord is a word shaped like a scenario CLI's name.
var scenarioWord = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)+$|^[a-z]{4,}$`)

// MaterialScore is the materiality floor: a finding or action whose score
// (Score) is under 1% is counted, not listed. The friction list and the
// recommendations apply this one rule to the score that ranks them.
const MaterialScore = 0.01

// Score is the one rule that ranks findings and actions and decides which are
// listed: the geometric mean √(share × breadth) of the share of its kind of
// harm it holds and the share of the window's runs that show it. A defect that
// touches many runs outranks one run's loop, and a small share seen in many
// runs (a park refusal in a tenth of all runs) is not lost under a share-only
// floor. Without a run count (windowRuns or runs zero) the score is the share.
func Score(share float64, runs, windowRuns int) float64 {
	if runs <= 0 || windowRuns <= 0 {
		return share
	}
	return math.Sqrt(share * min(1, float64(runs)/float64(windowRuns)))
}

// Material reports whether the finding is listed: its score reaches
// MaterialScore.
func (f Finding) Material() bool { return f.Score >= MaterialScore }

// findings turns profiles, clusters and bypasses into owner-level problems,
// ranked by Score.
func findings(report Report, calls []Call, polled []bool) []Finding {
	t := report.Totals
	var out []Finding
	// add completes a finding: its capped cost, a share per kind of harm
	// against that kind's total (byKind; nil is f.Kind's perRun alone), its
	// score, and a fingerprint of what the finding is, not of how much
	// evidence it has or of who it is attributed to: kind and the normalised
	// signature (the root cause of a roll-up, the command path for findings
	// about one command, the scenario and method of a bypass). The owner is an
	// attribute: a majority that flips between weeks must not reset a
	// finding's status or file it again.
	add := func(f Finding, perRun map[string]int64, byKind map[string]map[string]int64, signature string) {
		f.Cost, f.TopRunShare = Capped(perRun)
		if byKind == nil {
			byKind = map[string]map[string]int64{f.Kind: perRun}
		}
		for kind, runs := range byKind {
			cost, _ := Capped(runs)
			if cost == 0 {
				continue
			}
			total, base := t.Base(kind)
			f.Shares = append(f.Shares, Share{Kind: kind, Unit: UnitOf(kind), Cost: cost, Uncapped: sumInt64(runs), Share: shareInt64(cost, total), Base: base})
		}
		sort.Slice(f.Shares, func(i, j int) bool {
			if f.Shares[i].Share != f.Shares[j].Share {
				return f.Shares[i].Share > f.Shares[j].Share
			}
			return f.Shares[i].Kind < f.Shares[j].Kind
		})
		f.CostShare = f.Shares[0].Share
		for _, v := range perRun {
			if v > 0 {
				f.HarmRuns++
			}
		}
		f.Score = Score(f.CostShare, f.HarmRuns, t.Runs)
		f.OwnerKind = ownerKind(f.Owner)
		f.Fingerprint = fingerprint(f.Kind, signature)
		if uncapped := sumInt64(perRun); f.Cost < uncapped {
			// The amount in the sentence and the cost differ: say which is which.
			f.Description += fmt.Sprintf("; its cost counts %s of those %s%s, each run at most %dx the median run's", figure(f.Cost, f.Unit), figure(uncapped, f.Unit), unitWord(f.Unit), RunCapFactor)
			if f.TopRunShare >= ConcentratedShare && f.Runs > 1 {
				f.Description += fmt.Sprintf(" (%.0f%% of them in one run)", 100*f.TopRunShare)
			}
		} else if f.TopRunShare >= ConcentratedShare && f.Runs > 1 {
			f.Description += fmt.Sprintf("; %.0f%% of it in one run, so its cost counts that run at most %dx the median run", 100*f.TopRunShare, RunCapFactor)
		}
		if stale := report.Coverage.Stale.Paths[f.CommandPath]; stale > 0 && f.CommandPath != "" {
			f.Description += fmt.Sprintf("; %d more calls of it await re-derivation at the current fact version and are not counted", stale)
		}
		out = append(out, f)
	}
	for _, cl := range report.Clusters {
		if cl.Calls < minClusterCalls {
			continue
		}
		f := Finding{Kind: cl.Kind, Owner: cl.Owner, Calls: cl.Calls, Runs: cl.Runs, ErrorLine: cl.Signature, RootCause: cl.Root, Subs: cl.Subs, Examples: cl.Examples, RunIDs: cl.RunIDs, Unit: UnitCalls}
		where := commandList(cl.CommandPaths)
		switch {
		case len(cl.Subs) > 1:
			f.Description = fmt.Sprintf("%s hit the %s broker or refresh failing in %d calls in %d runs (a call that failed and warned is under both lines): %s", where, cl.Root, cl.Calls, cl.Runs, subList(cl.Subs))
		case cl.Kind == KindFailure && silent(cl.Line):
			f.Description = fmt.Sprintf("%s failed %d times in %d runs without an error line (%s)", where, cl.Calls, cl.Runs, cl.Line)
		case cl.Kind == KindFailure:
			f.Description = fmt.Sprintf("%s failed %d times in %d runs with: %s", where, cl.Calls, cl.Runs, cl.Line)
		case UnprintedNotes(cl.Line):
			// The summary promises notes naming the degraded providers that the
			// output never printed: the defect is the missing notes.
			f.Description = fmt.Sprintf("%s printed %q %d times in %d runs, but the output read never printed the notes naming the degraded providers: print them (provider and reason) or drop the promise", where, strings.TrimSuffix(cl.Line, " ["+unnamedProviders+"]"), cl.Calls, cl.Runs)
		default:
			f.Description = fmt.Sprintf("%s reported %q %d times in %d runs", where, cl.Line, cl.Calls, cl.Runs)
		}
		if len(cl.CommandPaths) == 1 {
			f.CommandPath = cl.CommandPaths[0]
		}
		signature := cl.Signature
		if cl.Root != "" {
			signature = "root:" + cl.Root
		}
		byKind := map[string]map[string]int64{}
		for kind, runs := range cl.kindRuns {
			byKind[kind] = mapInt64(runs)
		}
		if wait, _ := Capped(cl.waitRuns); wait > 0 {
			// Wall-clock belongs to whoever made the call wait: the time spent in
			// the degraded calls (`waiting inline instead`) is part of the cost.
			byKind[HarmWallClock] = cl.waitRuns
			f.Description += fmt.Sprintf("; %s spent in those calls", usagelabel.Duration(wait))
		}
		if cl.heldMS > 0 {
			// Parking would free the session during the wait, not shorten it:
			// the time is stated, never priced as a saving.
			f.Description += fmt.Sprintf("; %s held inline by those blocking waits (the wait itself, not a saving)", usagelabel.Duration(cl.heldMS))
		}
		add(f, mapInt64(cl.runCalls), byKind, signature)
	}
	examples := examplesByPath(calls, polled)
	for _, p := range report.Profiles {
		if p.Owner == "" {
			continue
		}
		base := Finding{Owner: p.Owner, CommandPath: p.CommandPath, Calls: p.Calls, Runs: p.Runs, RunIDs: topKeys(p.runs, maxEvidenceRuns)}
		if p.WrongShape >= minShapeGuesses {
			f := base
			f.Kind, f.Unit = KindShapeGuess, UnitCalls
			f.Examples, f.Guesses = examples[p.CommandPath].shape, examples[p.CommandPath].guesses()
			f.WrongShape, f.HelpLookups = p.WrongShape, p.ShapeGuesses-p.WrongShape
			f.Description = fmt.Sprintf("Agents guess how to call `%s`: of %d calls, %d failed for an unknown command, flag or missing argument and %d were help lookups around a failure or retry (of %d help lookups in all, %.0f%% of its calls)", p.CommandPath, p.Calls, f.WrongShape, f.HelpLookups, p.HelpLookups, 100*share(p.HelpLookups, p.Calls))
			if len(f.Guesses) > 0 {
				f.Description += "; named guesses " + guessList(f.Guesses) + ": add the alias or fix the help"
			}
			add(f, p.shapeRuns, nil, p.CommandPath)
		}
		if p.Calls >= minWeightCalls && (p.P90Tokens >= heavyP90Tokens || p.MaxTokens >= heavyMaxTokens || tokenaccounting.EstimateBytes(p.MaxOutputBytes) >= heavyMaxTokens || p.ContextTax >= heavyTaxFloor && shareInt64(p.ContextTax, t.ContextTax) >= heavyTaxShare) {
			f := base
			f.Kind, f.Unit = KindOutputWeight, UnitTokens
			f.Examples = examples[p.CommandPath].heavy
			f.Description = fmt.Sprintf("`%s` returns heavy output: median %s, p90 %s, max %s result tokens per call, %s tokens re-read in later requests; %.0f%% of calls cut it with head, grep, jq or similar", p.CommandPath, usagelabel.Tokens(p.MedianTokens), usagelabel.Tokens(p.P90Tokens), usagelabel.Tokens(p.MaxTokens), usagelabel.Tokens(p.ContextTax), 100*p.FilteredShare)
			f.PrintedTokens, f.ReadTokens = tokenaccounting.EstimateBytes(p.MaxOutputBytes), p.MaxTokens
			if printed := f.PrintedTokens; printed > 2*p.MaxTokens {
				f.Description += fmt.Sprintf("; the largest call printed about %s tokens, of which the harness showed the model %s", usagelabel.Tokens(printed), usagelabel.Tokens(p.MaxTokens))
			}
			if len(p.JSONPaths) > 0 {
				f.Description += "; after --json agents read " + strings.Join(p.JSONPaths, ", ")
			}
			add(f, p.taxRuns, nil, p.CommandPath)
		}
		if p.Polled >= minPolledCalls {
			f := base
			f.Kind, f.Unit = KindPolling, UnitCalls
			f.Examples = examples[p.CommandPath].polled
			f.Blocking = p.Applied
			f.Steered, _ = Capped(p.steerRuns)
			f.SteeredRuns = len(p.steerRuns)
			f.Description = fmt.Sprintf("Agents poll `%s`: %d calls ran three or more times within ten minutes of each other (%d inside sleep or until loops)", p.CommandPath, p.Polled, p.Wrapped)
			if p.Steered == p.Polled {
				f.Description = fmt.Sprintf("Agents poll `%s` although %s blocks: %d calls ran three or more times within ten minutes of each other (%d inside sleep or until loops)", p.CommandPath, quoteList(f.Blocking, " or "), p.Polled, p.Wrapped)
			} else {
				f.Description += steeringScope(p)
			}
			add(f, p.polledRuns, nil, p.CommandPath)
		}
		if interactiveVerbs[lastWord(p.CommandPath)] && p.Timed >= minSlowCalls && p.P90MS > slowBudgetMS && shareInt64(p.SlowExcessMS, t.VrooliDurationMS) >= slowMinShare {
			f := base
			f.Kind, f.Unit = KindSlow, UnitMS
			f.Description = fmt.Sprintf("`%s` should answer in under %s but its p90 is %s over %d timed calls: %s beyond the budget", p.CommandPath, usagelabel.Duration(slowBudgetMS), usagelabel.Duration(p.P90MS), p.Timed, usagelabel.Duration(p.SlowExcessMS))
			add(f, p.slowRuns, nil, p.CommandPath)
		}
	}
	for _, g := range bypassGroups(calls) {
		add(g.finding, g.perRun, nil, g.finding.Owner+" "+g.finding.ErrorLine)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Fingerprint < out[j].Fingerprint
	})
	return out
}

// Capped is a finding's or opportunity's cost: the sum of each run's
// contribution, each capped at RunCapFactor times the median run's, so one
// looping run (603 of a bypass's 968 calls) is not mistaken for 968 calls'
// worth of pattern. It is the one per-run cap rule of every cost. The
// median is over the runs that contributed: runs that hold the command but
// none of its harm (skill reads that put no tokens in context) would pull it
// to zero and cap every real run at almost nothing. A finding of one run keeps
// its cost (it is its own median); its TopRunShare of 1 says it is one run.
// TopRunShare is the largest run's share of the uncapped total.
func Capped(perRun map[string]int64) (cost int64, topShare float64) {
	values := make([]int64, 0, len(perRun))
	var total, top int64
	for _, v := range perRun {
		if v <= 0 {
			continue
		}
		values = append(values, v)
		total += v
		top = max(top, v)
	}
	if len(values) == 0 {
		return 0, 0
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	limit := RunCapFactor * max(values[(len(values)-1)/2], 1)
	for _, v := range values {
		cost += min(v, limit)
	}
	return cost, shareInt64(top, total)
}

// figure states a cost in its unit the way the descriptions do, and
// unitWord names the unit after it ("" for a duration, which names its own).
func figure(v int64, unit string) string {
	switch unit {
	case UnitTokens:
		return usagelabel.Tokens(v)
	case UnitMS:
		return usagelabel.Duration(v)
	}
	return strconv.FormatInt(v, 10)
}

func unitWord(unit string) string {
	if unit == UnitMS {
		return ""
	}
	return " " + unit
}

func sumInt64(values map[string]int64) int64 {
	var total int64
	for _, v := range values {
		total += v
	}
	return total
}

func mapInt64(in map[string]int) map[string]int64 {
	out := make(map[string]int64, len(in))
	for k, v := range in {
		out[k] = int64(v)
	}
	return out
}

// ConcentratedShare is the top-run share above which a finding says so.
const ConcentratedShare = 0.5

// subList names the lines of a rolled-up cluster, the most frequent first.
func subList(subs []SubSignature) string {
	parts := make([]string, 0, 4)
	for _, sub := range subs[:min(len(subs), 4)] {
		parts = append(parts, fmt.Sprintf("%s %q in %d calls", sub.Kind, bound(sub.Signature), sub.Calls))
	}
	if extra := len(subs) - len(parts); extra > 0 {
		parts = append(parts, fmt.Sprintf("%d more lines", extra))
	}
	return strings.Join(parts, "; ")
}

// steeringScope states which of a path's polls a blocking sibling stands in
// for when not all of them: a sibling declared to block only after another
// command acted on the target is claimed for the polls that followed one on
// the same target, and the rest have no blocking sibling.
func steeringScope(p Profile) string {
	var out string
	for _, b := range p.Blocking {
		if len(b.After) == 0 {
			continue
		}
		followed := fmt.Sprintf("which %d of them followed", p.Steered)
		if p.Steered == 0 {
			followed = "which none of them followed"
		}
		out += fmt.Sprintf("; `%s` blocks only after %s of the same target, %s", b.Command(), quoteList(b.After, " or "), followed)
		if p.Unnamed > 0 {
			out += fmt.Sprintf(" (%d more followed one that names no target, which is not claimed)", p.Unnamed)
		}
		break
	}
	return out + fmt.Sprintf("; %d polled calls have no blocking sibling", p.Polled-p.Steered-p.Unnamed)
}

// quoteList quotes commands and joins them with sep.
func quoteList(commands []string, sep string) string {
	quoted := make([]string, len(commands))
	for i, c := range commands {
		quoted[i] = "`" + c + "`"
	}
	return strings.Join(quoted, sep)
}

// guessList names the guessed commands and what the CLI suggested.
func guessList(guesses []Guess) string {
	parts := make([]string, 0, len(guesses))
	for _, g := range guesses {
		part := fmt.Sprintf("`%s` x%d", g.Token, g.Calls)
		if g.Suggestion != "" {
			part += fmt.Sprintf(" (suggested `%s`)", g.Suggestion)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, ", ")
}

// OwnWorkCount is the direct reads of one scenario's internals by one method
// that runs working on that scenario made (Call.OwnWork), and those runs.
type OwnWorkCount struct{ Calls, Runs int }

// Note states the reads set aside, "" when there are none.
func (o OwnWorkCount) Note(owner string) string {
	if o.Calls == 0 {
		return ""
	}
	return fmt.Sprintf(" (%d more in %d runs working on %s itself are not counted: developing or reviewing a scenario includes reading its internals)", o.Calls, o.Runs, owner)
}

// OwnWorkByTarget counts, per scenario and method, the direct reads made by
// runs working on that scenario.
func OwnWorkByTarget(calls []Call) map[[2]string]OwnWorkCount {
	out := map[[2]string]OwnWorkCount{}
	runs := map[[2]string]map[string]bool{}
	for _, c := range calls {
		if !c.OwnWork {
			continue
		}
		k := [2]string{c.BypassOwner, c.BypassMethod}
		if runs[k] == nil {
			runs[k] = map[string]bool{}
		}
		runs[k][c.RunID] = true
		out[k] = OwnWorkCount{Calls: out[k].Calls + 1, Runs: len(runs[k])}
	}
	return out
}

type bypassGroup struct {
	finding Finding
	perRun  map[string]int64
}

// MinPatternRuns is how many distinct runs must show a bypass (here and in
// the opportunities) before it is reported: one agent's habit is not a
// capability gap.
const MinPatternRuns = 3

// bypassGroups groups calls that reach a scenario's internals by scenario and
// method, for those seen in MinPatternRuns runs. Reads by runs working on that
// scenario (Call.OwnWork) are not bypasses; each group states how many it set
// aside (OwnWorkNote).
func bypassGroups(calls []Call) []bypassGroup {
	type key struct{ owner, method string }
	groups := map[key]*bypassGroup{}
	runs := map[key]map[string]int{}
	own := OwnWorkByTarget(calls)
	for _, c := range calls {
		if !c.Bypasses() {
			continue
		}
		k := key{c.BypassOwner, c.BypassMethod}
		g := groups[k]
		if g == nil {
			g = &bypassGroup{finding: Finding{Kind: KindBypass, Owner: c.BypassOwner, Unit: UnitCalls, ErrorLine: c.BypassMethod}, perRun: map[string]int64{}}
			groups[k], runs[k] = g, map[string]int{}
		}
		g.finding.Calls++
		runs[k][c.RunID]++
		g.perRun[c.RunID]++
		g.finding.Examples = addExample(g.finding.Examples, c)
	}
	var out []bypassGroup
	for k, g := range groups {
		if len(runs[k]) < MinPatternRuns {
			continue
		}
		g.finding.Runs, g.finding.RunIDs = len(runs[k]), topKeys(runs[k], maxEvidenceRuns)
		g.finding.Description = fmt.Sprintf("Agents reach around %s's CLI: %d %s in %d runs", k.owner, g.finding.Calls, k.method, g.finding.Runs) + own[[2]string{k.owner, k.method}].Note(k.owner)
		out = append(out, *g)
	}
	return out
}

type pathExamples struct {
	shape, heavy, polled []string
	guessed              map[string]*Guess
}

// guesses are the commands agents guessed at this path, most frequent first.
func (e *pathExamples) guesses() []Guess {
	out := make([]Guess, 0, len(e.guessed))
	for _, g := range e.guessed {
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Calls != out[j].Calls {
			return out[i].Calls > out[j].Calls
		}
		return out[i].Token < out[j].Token
	})
	n := 0
	for n < len(out) && n < maxGuesses && out[n].Calls >= minGuessCalls {
		n++
	}
	return out[:n]
}

// maxGuesses bounds the guesses a finding names; minGuessCalls is the
// evidence a guess needs before the advice names it ("add the alias", "fix
// the help"): one agent's typo is not the CLI's to fix.
const (
	maxGuesses    = 3
	minGuessCalls = 3
)

// examplesByPath keeps a few redacted example commands per command path for
// each finding kind: shape guesses (with the commands guessed), the heaviest
// call, and polled calls.
func examplesByPath(calls []Call, polled []bool) map[string]*pathExamples {
	out := map[string]*pathExamples{}
	heaviest := map[string]int64{}
	for i, c := range calls {
		e := out[c.CommandPath]
		if e == nil {
			e = &pathExamples{}
			out[c.CommandPath] = e
		}
		if c.Command == "" {
			continue
		}
		if c.Failed() && c.ShapeGuess || c.Help {
			e.shape = addExample(e.shape, c)
		}
		if c.Failed() && c.ShapeGuess {
			if token, suggestion := guessOf(c.ErrorLine); token != "" {
				if e.guessed == nil {
					e.guessed = map[string]*Guess{}
				}
				g := e.guessed[token]
				if g == nil {
					g = &Guess{Token: token}
					e.guessed[token] = g
				}
				g.Calls++
				if g.Suggestion == "" {
					g.Suggestion = suggestion
				}
			}
		}
		if c.ResultTokens > heaviest[c.CommandPath] {
			heaviest[c.CommandPath] = c.ResultTokens
			e.heavy = []string{c.Command}
		}
		if polled[i] {
			e.polled = addExample(e.polled, c)
		}
	}
	return out
}

// addExample adds a call's redacted command to a bounded, distinct list.
func addExample(examples []string, c Call) []string {
	if len(examples) >= maxEvidenceCommands || c.Command == "" {
		return examples
	}
	if example := c.Command; !contains(examples, example) {
		examples = append(examples, example)
	}
	return examples
}

// commandList names the command paths a cluster spans, most frequent first.
func commandList(paths []string) string {
	quoted := make([]string, 0, 3)
	for _, path := range paths[:min(len(paths), 3)] {
		quoted = append(quoted, "`"+path+"`")
	}
	text := strings.Join(quoted, ", ")
	if extra := len(paths) - len(quoted); extra > 0 {
		text += fmt.Sprintf(" and %d other commands", extra)
	}
	return text
}

func fingerprint(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:12])
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
