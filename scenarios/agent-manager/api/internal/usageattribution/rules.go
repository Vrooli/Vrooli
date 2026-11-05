package usageattribution

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/opportunities"
	"agent-manager/internal/pricing"
	"agent-manager/internal/toolfriction"
	"agent-manager/internal/usageforecast"
)

// Quota lever families (SPEC §11.2 Levers, §12.3 P4-W5). Each is an
// owner-level aggregate across the window's sessions; sessions are drill-down
// evidence. R-context alone also names still-running sessions it can help now.
const (
	RuleModel    = "R-model"
	RuleModelMix = "R-model-mix"
	RuleContext  = "R-context"
	RuleLifetime = "R-lifetime"
	RuleFloor    = "R-floor"
	RuleIdle     = "R-idle"
	RuleImage    = "R-image"
	RuleSkill    = "R-skill"
)

// Rules lists every lever family in report order.
var Rules = []string{RuleModel, RuleModelMix, RuleContext, RuleLifetime, RuleFloor, RuleIdle, RuleImage, RuleSkill}

// Lever assumptions. Thresholds that are not measured from the window are
// stated here with the reason; everything else (prompt floors, growth,
// compactions, re-reads, request rates) is read from the data.
const (
	// highPrompt is the request size whose quota share R-context reports:
	// Anthropic prices prompts above 200k at its long-context rate, and the
	// 2026-10-30 audit found 70% of Claude's week above it.
	highPrompt = 200_000
	// compactionDrop marks a compaction (or a restart inside one run): a
	// request whose prompt is under half the one before it. Ordinary turns
	// never shrink context; compaction cuts it to a summary.
	compactionDrop = 0.5
	// minCap is the shortest life a cap may impose: under 20 requests a worker
	// spends a large share of each life re-orienting, so shorter caps are not a
	// realistic instruction.
	minCap = 20
	// orientRequests is the assumed re-orientation after a restart: three
	// requests at the run's mean prompt (reading the brief, the plan, the files).
	orientRequests = 3
	// trimTokens is the starting-context cut R-floor prices: a unit an operator
	// can act on (one skill, one section of AGENTS.md).
	trimTokens = 10_000
	// imageScale is the share of an image's tokens a downscale saves: halving
	// both sides quarters the tokens; half is claimed to leave room for the
	// detail a task needs.
	imageScale = 0.5
	// modelMinRuns and modelSuccessSlack gate R-model: a cheaper model needs
	// three finished runs of the workload and a success rate within 10 points.
	modelMinRuns      = 3
	modelSuccessSlack = 0.10
	idleMaxOutput     = 600 // output tokens per request under which a wait turn is idle
	maxDrill          = 10  // drill-down sessions and evidence runs kept per action
	// minActiveGap, activeGapFactor and defaultSessionHours define active:
	// now less the last request within max(minActiveGap, activeGapFactor × the
	// session's median request gap); a session's expected remaining life
	// decays over the median session length, defaultSessionHours when unknown.
	minActiveGap        = 15 * time.Minute
	activeGapFactor     = 3
	defaultSessionHours = 4.0
)

// RuleStatus says whether a lever family fired, and why not from the data.
type RuleStatus struct {
	Rule       string
	Fired      bool
	Rows       int
	Candidates int
	Reason     string
	// caveat is what the rule could not see (token counts unknown for N
	// loads); it is stated in Reason whether the rule fired or not.
	caveat string
}

// WindowKey names one provider window.
type WindowKey struct{ Provider, Pool, Window string }

// Inputs are the facts the rules read. Results are the current windows'
// attributions; each provider's longest window is the one priced, or only the
// window named by Only. Tools, Opportunities and Interventions are the
// non-quota sources of the merged list (nil when unavailable).
type Inputs struct {
	Results  []Result
	Runs     map[string]Run
	Outcomes []Outcome
	Prices   Prices
	Only     WindowKey
	// Now is when the rules run; zero reads the newest reading.
	Now time.Time
	Sources
}

// prepare selects the evaluated windows, copies them (projection settings
// belong to this evaluation, not the caller's results) and gives each its
// sessions and projection context at in.Now.
func prepare(in Inputs) ([]*Result, map[*Result][]*sessionStats) {
	primary := selectResults(in.Results, in.Only)
	sessions := map[*Result][]*sessionStats{}
	for i, result := range primary {
		own := *result
		primary[i] = &own
		stats := sessionStatsFor(&own, in.Runs)
		sessions[&own] = stats
		own.proj = newProjCtx(&own, in.Now, stats)
	}
	return primary, sessions
}

// selectResults keeps the window named by only, or else the longest window of
// each provider pool (usageforecast.Longest, the forecast's primary window).
func selectResults(results []Result, only WindowKey) []*Result {
	var out []*Result
	longest := usageforecast.Longest(len(results), func(i int) (string, string, int64) {
		w := results[i].Window
		return w.Provider, w.Pool, w.Minutes
	})
	for i := range results {
		w := results[i].Window
		if only == (WindowKey{}) && longest[i] || only != (WindowKey{}) && w.Provider == only.Provider && w.Pool == only.Pool && w.Name == only.Window {
			out = append(out, &results[i])
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].Window, out[j].Window
		if a.Provider != b.Provider {
			return a.Provider < b.Provider
		}
		return a.Pool < b.Pool
	})
	return out
}

// horizonHours is the length savings and projections are stated over: the
// window's own length, a week when unknown.
func horizonHours(result *Result) float64 {
	if result.Window.Minutes > 0 {
		return float64(result.Window.Minutes) / 60
	}
	return 7 * 24
}

// remainingHours is the time left in the window at now (the newest reading
// when now is unknown).
func remainingHours(result *Result, now time.Time) float64 {
	if now.IsZero() {
		now = result.To
	}
	horizon := horizonHours(result)
	switch {
	case !result.Window.ResetAt.IsZero():
		return max(0, min(horizon, result.Window.ResetAt.Sub(now).Hours()))
	case !result.Window.Start.IsZero():
		return max(0, min(horizon, horizon-now.Sub(result.Window.Start).Hours()))
	}
	return horizon
}

// projCtx is what a projection needs beyond a consumer's own points: when the
// rules run, how often requests come, how long sessions last and how much of
// the window's headroom the active consumers may still use.
type projCtx struct {
	now time.Time
	tau float64
	// keep is the fraction of every projected future use that fits in the
	// window's headroom (100 less used percent); nil is all of it.
	keep *float64
}

func median[T int64 | float64](values []T) T {
	var zero T
	if len(values) == 0 {
		return zero
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values[len(values)/2]
}

// newProjCtx reads now (the newest attributed use when unset) and the median
// session length, then scales the active sessions' future use to the
// window's headroom, pro rata.
func newProjCtx(result *Result, now time.Time, stats []*sessionStats) projCtx {
	if now.IsZero() {
		for _, s := range stats {
			if s.last.After(now) {
				now = s.last
			}
		}
	}
	c := projCtx{now: now, tau: defaultSessionHours}
	var lives []float64
	for _, s := range stats {
		if hours := s.last.Sub(s.first).Hours(); len(s.times) > 1 && hours > 0 {
			lives = append(lives, hours)
		}
	}
	if len(lives) > 0 {
		c.tau = median(lives)
	}
	var future float64
	for _, s := range stats {
		future += s.projection(result, c).future()
	}
	if headroom := max(0, 100-result.Window.UsedPercent); future > headroom {
		keep := headroom / future
		c.keep = &keep
	}
	return c
}

// projection is a consumer's projected use over the window's horizon. A
// finished consumer (no request within max(15 min, 3 × its median request gap)
// of now) projects to its points. An active one adds its rate over its own
// span, decaying with the median session length τ over the time R left in the
// window: r·τ·(1 − e^(−R/τ)), scaled to the window's headroom.
type projection struct {
	points, weekly float64
	active         bool
}

func (p projection) future() float64 { return p.weekly - p.points }

func projectSpan(result *Result, c projCtx, points float64, first, last time.Time, gap time.Duration) projection {
	now := c.now
	if now.IsZero() {
		now = result.To
	}
	p := projection{points: points, weekly: points}
	if last.IsZero() || now.Sub(last) > max(minActiveGap, activeGapFactor*gap) {
		return p
	}
	keep := 1.0
	if c.keep != nil {
		keep = *c.keep
	}
	hours := max(last.Sub(first).Hours(), 1)
	p.active = true
	p.weekly = points + keep*points/hours*c.tau*(1-math.Exp(-remainingHours(result, now)/c.tau))
	return p
}

// saving states a saving over the attributed period at the consumer's
// projected rate, capped at that projection: a finished consumer's saving is
// what it would have saved, an active one's what it is projected to save.
func (p projection) saving(saved float64) float64 {
	if p.points <= 0 || saved <= 0 {
		return 0
	}
	return min(saved*p.weekly/p.points, p.weekly)
}

// recoverable is the part of a saving still to be had before the window
// resets: only an active consumer's future share.
func (p projection) recoverable(saved float64) float64 {
	if !p.active || p.weekly <= 0 {
		return 0
	}
	return p.saving(saved) * p.future() / p.weekly
}

// point is one turn of a run: when, the request size it read, how many
// requests, and their quota.
type point struct {
	at               time.Time
	prompt           float64
	reports, images  int
	points, weighted float64
}

// runStat is one run's requests in the window.
type runStat struct {
	id               string
	child            bool
	workload         string
	runner, model    string
	series           []point
	modelPoints      map[string]float64
	points, weighted float64
	reports          int
	prompt           int64
}

// pointsPerToken converts the run's cached-input-equivalent tokens to points.
func (rs *runStat) pointsPerToken() float64 {
	if rs.weighted <= 0 {
		return 0
	}
	return rs.points / rs.weighted
}

// segment is one stretch of a run between compactions.
type segment struct {
	requests    int
	first, last float64
}

// segments splits the run's requests at compactions (see compactionDrop).
func (rs *runStat) segments() []segment {
	var out []segment
	for i, p := range rs.series {
		if i == 0 || p.prompt < compactionDrop*rs.series[i-1].prompt {
			out = append(out, segment{first: p.prompt})
		}
		seg := &out[len(out)-1]
		seg.requests += p.reports
		seg.last = p.prompt
	}
	return out
}

// firstPrompt is the run's first request size: its first turn's per-request
// average, or its first one-request turn's prompt when that is smaller (an
// average over many requests overstates the first).
func (rs *runStat) firstPrompt() float64 {
	if len(rs.series) == 0 {
		return 0
	}
	first := rs.series[0].prompt
	for _, p := range rs.series {
		if p.reports == 1 {
			return min(first, p.prompt)
		}
	}
	return first
}

// netAtCap is the growth model of R-context and R-lifetime at one cap. Within
// one segment a prompt grows by g = (last − first)/n a request; capping its
// life at k requests (a restart or an earlier compaction) saves g·n(n − k)/2
// cached-equivalent tokens and costs (n/k − 1) rebuilds. A run that compacts
// is priced on its own observed regrowth after each compaction, never as a
// session that stops growing. The result is the net over the run's segments.
func netAtCap(segs []segment, rebuild float64, k int) float64 {
	var net float64
	for _, s := range segs {
		n := float64(s.requests)
		if s.requests <= k {
			continue
		}
		growth := max(0, s.last-s.first) / n
		net += growth*n*(n-float64(k))/2 - (n/float64(k)-1)*rebuild
	}
	return net
}

// capSaving returns the cap that nets the most for one run and that saving in
// tokens.
func capSaving(segs []segment, rebuild float64) (bestCap int, best float64) {
	longest := 0
	for _, s := range segs {
		longest = max(longest, s.requests)
	}
	for k := minCap; k < longest; k++ {
		if net := netAtCap(segs, rebuild, k); net > best {
			bestCap, best = k, net
		}
	}
	return bestCap, best
}

type sessionStats struct {
	key, label       string
	points, weighted float64
	runs             map[string]*runStat
	idleReports      int
	idlePoints       float64
	idleStretches    int
	lastIdle         bool
	first, last      time.Time
	times            []time.Time
	images           int
}

func sessionKey(run Run, id string) string {
	if run.Session != "" {
		return run.Session
	}
	return id
}

func sessionStatsFor(result *Result, runs map[string]Run) []*sessionStats {
	byKey := map[string]*sessionStats{}
	order := make([]int, len(result.Turns))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return result.Turns[order[a]].At.Before(result.Turns[order[b]].At) })
	for _, i := range order {
		turn := result.Turns[i]
		run := runs[turn.RunID]
		key := sessionKey(run, turn.RunID)
		s := byKey[key]
		if s == nil {
			s = &sessionStats{key: key, label: shortLabel(run, key), runs: map[string]*runStat{}}
			byKey[key] = s
		}
		points, weighted := result.Points[i], result.Weighted[i]
		s.points += points
		s.weighted += weighted
		rs := s.runs[turn.RunID]
		if rs == nil {
			rs = &runStat{id: turn.RunID, child: run.Child, workload: run.Workload, runner: turn.Runner, model: turn.Model, modelPoints: map[string]float64{}}
			s.runs[turn.RunID] = rs
		}
		rs.points += points
		rs.weighted += weighted
		rs.modelPoints[turn.Model] += points
		rs.reports += turn.Reports
		rs.prompt += turn.Prompt()
		rs.series = append(rs.series, point{at: turn.At, prompt: float64(turn.Prompt()) / float64(max(turn.Reports, 1)), reports: turn.Reports, images: turn.Facts.ImagesIn, points: points, weighted: weighted})
		idle := idleTurn(turn)
		if idle {
			s.idleReports += turn.Reports
			s.idlePoints += points
			if !s.lastIdle {
				s.idleStretches++
			}
		}
		s.lastIdle = idle
		s.images += turn.Facts.ImagesIn
		s.times = append(s.times, turn.At)
		if s.first.IsZero() {
			s.first = turn.At
		}
		s.last = turn.At
	}
	// Requests after the newest reading are in no points but show their
	// session is still going.
	for id, at := range result.pendingAt {
		if s := byKey[sessionKey(runs[id], id)]; s != nil && at.After(s.last) {
			s.last = at
			s.times = append(s.times, at)
		}
	}
	out := make([]*sessionStats, 0, len(byKey))
	for _, s := range byKey {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].points != out[j].points {
			return out[i].points > out[j].points
		}
		return out[i].key < out[j].key
	})
	return out
}

// projection is the session's projected use: active by its own median request
// gap and rate over its own span.
func (s *sessionStats) projection(result *Result, c projCtx) projection {
	var gaps []int64
	for i := 1; i < len(s.times); i++ {
		gaps = append(gaps, int64(s.times[i].Sub(s.times[i-1])))
	}
	return projectSpan(result, c, s.points, s.first, s.last, time.Duration(median(gaps)))
}

func (s *sessionStats) project(result *Result) projection { return s.projection(result, result.proj) }

// sortedRuns is the session's runs, largest first, for stable evidence.
func (s *sessionStats) sortedRuns() []*runStat {
	out := make([]*runStat, 0, len(s.runs))
	for _, rs := range s.runs {
		out = append(out, rs)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].points != out[j].points {
			return out[i].points > out[j].points
		}
		return out[i].id < out[j].id
	})
	return out
}

type floorKey struct {
	runner, model string
	child         bool
}

// measuredFloors is the median first-request prompt of each runner, model and
// role (session or sub-agent) over the window's runs: the smallest context a
// run of that kind starts with, which no restart can go below.
func measuredFloors(stats []*sessionStats) map[floorKey]float64 {
	values := map[floorKey][]float64{}
	for _, s := range stats {
		for _, rs := range s.runs {
			if first := rs.firstPrompt(); first > 0 {
				k := floorKey{rs.runner, rs.model, rs.child}
				values[k] = append(values[k], first)
			}
		}
	}
	out := make(map[floorKey]float64, len(values))
	for k, v := range values {
		out[k] = median(v)
	}
	return out
}

// creationWeight is a model's cache-creation weight relative to a cached-input
// token in this window.
func creationWeight(result *Result, runner, model string) float64 {
	for _, w := range result.Calibration.Weights {
		if w.Runner == runner && w.Model == model && w.CreationWeight > 0 {
			return w.CreationWeight
		}
	}
	return fallbackPrice.CacheCreation / fallbackPrice.CacheRead
}

// idleTurn is a turn whose tool calls are mostly waits (sleep, wait, monitor)
// with little output. Spawning a sub-agent is work, not a wait.
func idleTurn(turn Turn) bool {
	if turn.Reports == 0 || turn.Tokens.OutputTokens/turn.Reports >= idleMaxOutput {
		return false
	}
	waits := turn.Facts.ToolKinds["wait"]
	return waits > 0 && 2*waits >= turn.Facts.ToolCalls
}

// rebuildCost is what restarting a run costs in cached-input-equivalent
// tokens: the floor written to cache plus orientRequests at the run's mean
// prompt.
func rebuildCost(floor, creation, meanPrompt float64) float64 {
	return floor*creation + orientRequests*meanPrompt
}

// role names a run's role for owner and action text.
func role(child bool) string {
	if child {
		return "sub-agent"
	}
	return "session"
}

// ruleGrowth is R-context (sessions) and R-lifetime (sub-agents): one growth
// model, one owner per runner. A run grows from its own first request; capping
// its segments at the best cap saves what capSaving says, priced at the run's
// own points per token. Finished sessions contribute what they would have
// saved, active ones what they are projected to save. R-context also names
// each still-running session that has never compacted and would gain from a
// restart now: restarting at prompt L to the floor F makes each of its R
// expected remaining requests F − L cheaper (both regrow alike), less one
// rebuild. A session that compacts is never told to compact.
func ruleGrowth(in Inputs, primary []*Result, sessions map[*Result][]*sessionStats, child bool) ([]Action, RuleStatus) {
	rule := RuleContext
	if child {
		rule = RuleLifetime
	}
	status := RuleStatus{Rule: rule}
	var out []Action
	// One cap per runner: a provider's windows (a 5-hour and a weekly pool)
	// see the same sessions, and fitting the growth model in each would tell
	// one runner two caps. Its longest window governs.
	governing := map[string]*Result{}
	for _, result := range primary {
		if g := governing[result.Window.Provider]; g == nil || result.Window.Minutes > g.Window.Minutes {
			governing[result.Window.Provider] = result
		}
	}
	for _, result := range primary {
		if governing[result.Window.Provider] != result {
			continue
		}
		floors := measuredFloors(sessions[result])
		// cand is a run that would save at its own best cap, kept until the
		// aggregate's one cap is chosen and every member is re-priced at it.
		type cand struct {
			rs      *runStat
			s       *sessionStats
			use     projection
			segs    []segment
			rebuild float64
			best    int
		}
		type agg struct {
			points, high float64
			caps         []float64
			compactions  int
			members      []*runStat
			cands        []cand
		}
		type restart struct {
			row     Action
			session string
		}
		byRunner := map[string]*agg{}
		var runners []string
		var restarts []restart
		for _, s := range sessions[result] {
			use := s.project(result)
			for _, rs := range s.sortedRuns() {
				if rs.child != child || rs.reports == 0 || rs.weighted <= 0 {
					continue
				}
				a := byRunner[rs.runner]
				if a == nil {
					a = &agg{}
					byRunner[rs.runner] = a
					runners = append(runners, rs.runner)
				}
				status.Candidates++
				a.members = append(a.members, rs)
				segs := rs.segments()
				a.compactions += len(segs) - 1
				a.points += rs.points
				for _, p := range rs.series {
					if p.prompt > highPrompt {
						a.high += p.points
					}
				}
				floor := floors[floorKey{rs.runner, rs.model, rs.child}]
				mean := float64(rs.prompt) / float64(rs.reports)
				rebuild := rebuildCost(floor, creationWeight(result, rs.runner, rs.model), mean)
				k, tokens := capSaving(segs, rebuild)
				if use.saving(min(tokens*rs.pointsPerToken(), rs.points)) <= 0 {
					continue
				}
				a.caps = append(a.caps, float64(k))
				a.cands = append(a.cands, cand{rs, s, use, segs, rebuild, k})
			}
			if !child && use.active {
				if row, ok := restartNow(result, s, use, floors); ok {
					restarts = append(restarts, restart{row, s.key})
				}
			}
		}
		inAggregate := map[string]bool{}
		for _, runner := range runners {
			a := byRunner[runner]
			if len(a.caps) == 0 {
				continue
			}
			// One cap is told to the operator, so every member is priced at that
			// cap, not at its own best; a member that nets nothing at it drops out.
			k := int(median(a.caps))
			var saved, recoverable float64
			var recovers []recovery
			var drill []Drill
			var evidence []Ref
			var priced []string
			var prompts []float64
			for _, c := range a.cands {
				tokens := netAtCap(c.segs, c.rebuild, k)
				value := c.use.saving(min(tokens*c.rs.pointsPerToken(), c.rs.points))
				if tokens <= 0 || value <= 0 {
					continue
				}
				priced = append(priced, c.rs.id)
				prompts = append(prompts, promptAtRequest(c.rs, k))
				saved += value
				points := c.use.recoverable(tokens * c.rs.pointsPerToken())
				recoverable += points
				recovers = append(recovers, recovery{run: c.rs.id, session: c.s.key, points: points, future: c.use.future()})
				inAggregate[c.s.key] = true
				drill = append(drill, Drill{Kind: "session", ID: c.s.key, Label: c.s.label, Value: value, StillRunning: c.use.active, n: len(c.segs) - 1, unit: "compacted %d×"})
				if len(evidence) < maxDrill {
					evidence = append(evidence, Ref{Kind: "run", ID: c.rs.id, Label: c.s.label})
				}
			}
			if len(priced) == 0 {
				continue
			}
			var after float64
			for _, rs := range a.members {
				after += pointsAfter(rs, k)
			}
			row := leverAction(result, rule, Effect{Value: saved, Unit: UnitPoints, Basis: basisOf(drill)})
			row.Recoverable, row.recovers, row.priced = recoverable, recovers, priced
			row.OwnerKind, row.Owner = OwnerHarness, runner+" "+role(child)+" context"
			row.Confidence = "medium"
			highShare := share(a.high, a.points)
			if child {
				row.Title = fmt.Sprintf("Restart %s sub-agents after ~%d requests", runner, k)
				row.Text = fmt.Sprintf("Restart %s sub-agents after about %d requests instead of letting their context grow", runner, k)
				row.Rationale = fmt.Sprintf("%d of %d sub-agents still save at that cap; %.0f%% of sub-agent quota was requests after #%d and %.0f%% at prompts above %dk. The cap is the median of each worker's best cap under the growth model; the saving re-prices every worker at that one cap, net of the floor rebuild and %d re-orientation requests.", len(priced), len(a.members), 100*share(after, a.points), k, 100*highShare, highPrompt/1000, orientRequests)
			} else {
				threshold, knob := ContextThreshold(runner, prompts)
				row.Title = fmt.Sprintf("Compact %s sessions at about %dk tokens", runner, threshold/1000)
				row.Text = fmt.Sprintf("Compact %s sessions when their context reaches about %dk tokens (roughly request %d of growth) instead of letting it grow: %s", runner, threshold/1000, k, knob)
				row.Rationale = fmt.Sprintf("%.0f%% of %s session quota was requests above %dk tokens. %d sessions still save at a cap of %d requests, which is a context of about %dk tokens; %d compactions already happened and each session is priced on its own regrowth after them. The cap is the median best cap under the growth model; the saving re-prices every session at that one cap, net of rebuilding the floor and %d re-orientation requests.", 100*highShare, runner, highPrompt/1000, len(priced), k, threshold/1000, a.compactions, orientRequests)
			}
			row.Evidence, row.Drilldown = evidence, topDrill(drill, "")
			row.Verify = compareVerify(result, runnerFilter(runner), evidence, len(priced))
			out = append(out, row)
		}
		// A live session already priced in an aggregate is its drill-down, not
		// a second action: both would count the same quota.
		for _, r := range restarts {
			if !inAggregate[r.session] {
				out = append(out, r.row)
			}
		}
	}
	if len(out) == 0 {
		status.Reason = fmt.Sprintf("no %s grew past its own first request by more than a restart or compaction costs (caps of %d+ requests) in the current window", role(child), minCap)
	}
	return out, status
}

// promptAtRequest is the request size, in tokens, the run reached at its k-th
// request (the last turn's mean request size when it made fewer).
func promptAtRequest(rs *runStat, k int) float64 {
	var seen int
	var prompt float64
	for _, p := range rs.series {
		prompt = p.prompt
		seen += p.reports
		if seen >= k {
			break
		}
	}
	return prompt
}

// minContextThreshold is the smallest threshold worth setting: below it a
// compaction costs about as much as the context it removes.
const minContextThreshold = 40_000

// claudeMinContextThreshold is the smallest auto-compact window Claude Code
// honours; a lower request is raised to it.
const claudeMinContextThreshold = 100_000

// ContextThreshold turns a request cap into the token threshold a harness is
// told: the median request size at the cap, rounded to 10k and never below
// minContextThreshold. Claude honours nothing under 100k, so its threshold is
// raised to that and the sentence says so. The knob names the per-harness
// setting an operator can use without persisting anything.
func ContextThreshold(runner string, prompts []float64) (int64, string) {
	tokens := int64(math.Round(median(append([]float64(nil), prompts...))/10_000) * 10_000)
	tokens = max(tokens, minContextThreshold)
	switch runner {
	case "claude", "claude-code":
		note := ""
		if tokens < claudeMinContextThreshold {
			tokens = claudeMinContextThreshold
			note = " (Claude ignores limits under 100k, so this is raised to 100k)"
		}
		return tokens, fmt.Sprintf("set CLAUDE_CODE_AUTO_COMPACT_WINDOW=%d for the harness%s, or use a usage mode's context limit", tokens, note)
	case "codex":
		return tokens, fmt.Sprintf("set model_auto_compact_token_limit=%d for the harness, or use a usage mode's context limit", tokens)
	}
	return tokens, "lower the harness's auto-compact threshold to that size"
}

// pointsAfter is the run's points in requests after its first k.
func pointsAfter(rs *runStat, k int) float64 {
	var seen int
	var after float64
	for _, p := range rs.series {
		if seen >= k {
			after += p.points
		}
		seen += p.reports
	}
	return after
}

// restartNow is R-context for one still-running session that has never
// compacted: what restarting it now saves over its expected remaining
// requests.
func restartNow(result *Result, s *sessionStats, use projection, floors map[floorKey]float64) (Action, bool) {
	var main *runStat
	for _, rs := range s.sortedRuns() {
		if !rs.child {
			main = rs
			break
		}
	}
	if main == nil || len(main.series) == 0 || len(main.segments()) > 1 || use.points <= 0 {
		return Action{}, false
	}
	floor := floors[floorKey{main.runner, main.model, false}]
	last := main.series[len(main.series)-1].prompt
	remaining := float64(main.reports) * use.future() / use.points
	mean := float64(main.prompt) / float64(main.reports)
	tokens := (last-floor)*remaining - rebuildCost(floor, creationWeight(result, main.runner, main.model), mean)
	saved := min(tokens*main.pointsPerToken(), use.future())
	if saved < toolfriction.MaterialScore*result.Observed {
		return Action{}, false
	}
	row := leverAction(result, RuleContext, Effect{Value: saved, Unit: UnitPoints, Basis: BasisProjected})
	row.Recoverable, row.recovers, row.priced = saved, []recovery{{run: main.id, session: s.key, points: saved, future: use.future()}}, []string{main.id}
	row.OwnerKind, row.Owner = OwnerSession, s.key
	row.Title = "Restart " + s.label
	row.Text = fmt.Sprintf("Restart or compact %s now: each request re-reads ≈%dk tokens and it is still running", s.label, int(last/1000))
	row.Rationale = fmt.Sprintf("It has not compacted. A restart drops each later request from ≈%dk to the ≈%dk floor; over the ≈%.0f requests it is projected to make that saves ≈%.1f pts, net of one rebuild. It regrows at the same rate either way.", int(last/1000), int(floor/1000), remaining, saved)
	row.Confidence = "medium"
	row.Evidence = []Ref{{Kind: "session", ID: s.key, Label: s.label}, {Kind: "run", ID: main.id}}
	row.Verify = fmt.Sprintf("`agent-manager usage burners --by session --run %s` should show its points per request fall after the restart", main.id)
	return row, true
}

// topDrill merges each session's drill-down parts, keeps the largest, and
// states each: what it would have saved (finished) or is projected to save
// (still running), or the phrase given ("up to" for a range).
func topDrill(drill []Drill, phrase string) []Drill {
	byID := map[string]int{}
	var out []Drill
	for _, d := range drill {
		if i, ok := byID[d.ID]; ok {
			out[i].Value += d.Value
			out[i].n += d.n
			out[i].StillRunning = out[i].StillRunning || d.StillRunning
			continue
		}
		byID[d.ID] = len(out)
		out = append(out, d)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Value > out[j].Value })
	out = out[:min(len(out), maxDrill)]
	for i := range out {
		d := &out[i]
		verb, state := "would have saved", "finished"
		if d.StillRunning {
			verb, state = "projected to save", "still running"
		}
		if phrase != "" {
			verb = phrase
		}
		d.Detail = fmt.Sprintf("%s ≈%.1f pts (%s)", verb, d.Value, state)
		if d.n > 0 {
			d.Detail += "; " + fmt.Sprintf(d.unit, d.n)
		}
	}
	return out
}

// basisOf is an aggregate's basis: projected when any part is still running.
func basisOf(drill []Drill) string {
	for _, d := range drill {
		if d.StillRunning {
			return BasisProjected
		}
	}
	return BasisWouldHaveSaved
}

func share(part, whole float64) float64 {
	if whole <= 0 {
		return 0
	}
	return part / whole
}

// compareVerify names the cohort check for a quota lever: every evidence run
// of the lever (saying how many of the runs it prices they are) before the
// change, against the runs of the same filter since the change, which cannot
// hold the left side's runs.
func compareVerify(result *Result, filter string, evidence []Ref, priced int) string {
	var ids []string
	for _, ref := range evidence {
		if ref.Kind == "run" {
			ids = append(ids, ref.ID)
		}
	}
	left := fmt.Sprintf("its %d evidence runs", len(ids))
	if priced > len(ids) {
		left = fmt.Sprintf("its %d evidence runs (of the %d runs it prices)", len(ids), priced)
	}
	return fmt.Sprintf("after the change: `agent-manager usage compare --provider %s --left-run-ids %s --right-filter-json '{%s,\"window\":{\"custom\":{\"from\":\"<time of the change>\",\"to\":\"<now>\"}}}'` should read improved: %s against the runs since the change", result.Window.Provider, strings.Join(ids, ","), filter, left)
}

// runnerFilter and modelFilter are the InvocationFilter fields of a verify
// cohort.
func runnerFilter(runner string) string { return fmt.Sprintf("\"runner_type\":%q", runner) }

func modelFilter(model string) string { return fmt.Sprintf("\"model\":%q", model) }

// ratioOf is the tier-prior price ratio of a model's cheaper tier.
func ratioOf(model string) float64 {
	_, ratio, _ := pricing.CheaperTier(model)
	return ratio
}

// ruleModelMix aggregates work on a dearer tier per runner, role and model,
// interactive sessions included. Whether the cheaper tier does the work as
// well is untested, so the effect is a range from nothing to the price gap and
// the confidence is "unproven"; the ratio is a tier prior.
func ruleModelMix(in Inputs, primary []*Result, sessions map[*Result][]*sessionStats) ([]Action, RuleStatus) {
	status := RuleStatus{Rule: RuleModelMix}
	var out []Action
	for _, result := range primary {
		type key struct {
			runner, model string
			child         bool
		}
		type agg struct {
			points, weekly float64
			drill          []Drill
			evidence       []Ref
			priced         []string
		}
		groups := map[key]*agg{}
		var keys []key
		for _, s := range sessions[result] {
			use := s.project(result)
			for _, rs := range s.sortedRuns() {
				if rs.workload != "" { // workload-tagged runs are R-model's
					continue
				}
				for model, points := range rs.modelPoints {
					if _, _, ok := pricing.CheaperTier(model); !ok || points <= 0 {
						continue
					}
					k := key{rs.runner, model, rs.child}
					a := groups[k]
					if a == nil {
						a = &agg{}
						groups[k] = a
						keys = append(keys, k)
					}
					weekly := use.saving(points)
					a.points += points
					a.weekly += weekly
					a.drill = append(a.drill, Drill{Kind: "session", ID: s.key, Label: s.label, Value: weekly * (1 - ratioOf(model)), StillRunning: use.active})
					a.priced = append(a.priced, rs.id)
					if len(a.evidence) < maxDrill {
						a.evidence = append(a.evidence, Ref{Kind: "run", ID: rs.id, Label: s.label})
					}
				}
			}
		}
		for _, k := range keys {
			status.Candidates++
			a := groups[k]
			marker, ratio, _ := pricing.CheaperTier(k.model)
			row := leverAction(result, RuleModelMix, Effect{Value: 0, Upper: a.weekly * (1 - ratio), Unit: UnitPoints, Basis: BasisRange})
			row.OwnerKind, row.Owner = OwnerHarness, fmt.Sprintf("%s %s model", k.runner, role(k.child))
			row.Confidence = "unproven"
			row.Title = fmt.Sprintf("Try %s for %s %ss", marker, k.runner, role(k.child))
			row.Text = fmt.Sprintf("Run lighter %s %s work on %s instead of %s", k.runner, role(k.child), marker, k.model)
			row.Rationale = fmt.Sprintf("%s %ss used %s (%.0f%% of the window). %s costs %.0f%% of %s per token by a tier prior, not a fitted ratio, and nothing here shows it does this work as well: the saving is anywhere from 0 to %s.", k.model, role(k.child), pts(a.points), 100*share(a.points, result.Observed), marker, ratio*100, k.model, pts(row.Effect.Upper))
			row.Evidence, row.Drilldown, row.priced = a.evidence, topDrill(a.drill, "up to"), a.priced
			row.Verify = compareVerify(result, runnerFilter(k.runner), a.evidence, len(a.priced))
			out = append(out, row)
		}
	}
	if len(out) == 0 {
		status.Reason = "no session or sub-agent outside a workload used a model with a cheaper tier (opus, sol, terra) in the current window"
	}
	return out, status
}

// ruleFloor prices the starting context per runner, model and role: every
// request re-reads the floor, so each trimTokens cut from it saves that many
// cached tokens a request. How much is trimmable is unknown; the effect is
// stated per trimTokens.
func ruleFloor(in Inputs, primary []*Result, sessions map[*Result][]*sessionStats) ([]Action, RuleStatus) {
	status := RuleStatus{Rule: RuleFloor}
	var out []Action
	for _, result := range primary {
		floors := measuredFloors(sessions[result])
		type agg struct {
			perTrim, floorPoints, points float64
			requests                     int
			drill                        []Drill
			evidence                     []Ref
			priced                       []string
		}
		groups := map[floorKey]*agg{}
		var keys []floorKey
		for _, s := range sessions[result] {
			use := s.project(result)
			for _, rs := range s.sortedRuns() {
				k := floorKey{rs.runner, rs.model, rs.child}
				floor := floors[k]
				if floor <= 0 || rs.reports == 0 {
					continue
				}
				a := groups[k]
				if a == nil {
					a = &agg{}
					groups[k] = a
					keys = append(keys, k)
				}
				perTrim := use.saving(float64(rs.reports) * trimTokens * rs.pointsPerToken())
				a.perTrim += perTrim
				a.floorPoints += float64(rs.reports) * min(floor, float64(rs.prompt)/float64(rs.reports)) * rs.pointsPerToken()
				a.points += rs.points
				a.requests += rs.reports
				a.priced = append(a.priced, rs.id)
				a.drill = append(a.drill, Drill{Kind: "session", ID: s.key, Label: s.label, Value: perTrim, StillRunning: use.active, n: rs.reports, unit: "%d requests"})
				if len(a.evidence) < maxDrill {
					a.evidence = append(a.evidence, Ref{Kind: "run", ID: rs.id, Label: s.label})
				}
			}
		}
		for _, k := range keys {
			status.Candidates++
			a := groups[k]
			if len(a.priced) < opportunities.MinSessions {
				continue // one run's start is not an owner's pattern
			}
			row := leverAction(result, RuleFloor, Effect{Value: a.perTrim, Unit: UnitPoints, Basis: BasisPerTrim})
			row.OwnerKind, row.Owner = OwnerHarness, fmt.Sprintf("%s %s starting context", k.runner, role(k.child))
			row.Confidence = "low"
			row.Title = fmt.Sprintf("Trim the %dk %s %s starting context", int(floors[k]/1000), k.runner, role(k.child))
			row.Text = fmt.Sprintf("Trim what every %s %s (%s) starts with: AGENTS.md/CLAUDE.md, preloaded skills, tool schemas", k.runner, role(k.child), k.model)
			row.Rationale = fmt.Sprintf("The measured first request is ≈%dk tokens and %d requests re-read it: ≈%.0f%% of these runs' quota (%s). Each %dk tokens cut saves ≈%.2f pts this window; how much is trimmable is unknown.", int(floors[k]/1000), a.requests, 100*share(a.floorPoints, a.points), pts(a.floorPoints), trimTokens/1000, a.perTrim)
			row.Evidence, row.Drilldown, row.priced = a.evidence, topDrill(a.drill, "per 10k trimmed"), a.priced
			row.Verify = compareVerify(result, runnerFilter(k.runner), a.evidence, len(a.priced))
			out = append(out, row)
		}
	}
	if len(out) == 0 {
		status.Reason = fmt.Sprintf("no runner, model and role started %d+ runs in the current window", opportunities.MinSessions)
	}
	return out, status
}

// sessionLever sums one per-session saving (points over the attributed
// period, and a count for the drill-down) into an aggregate lever per window,
// with sessions as drill-down evidence. describe fills the owner and text from
// the summed count.
func sessionLever(primary []*Result, sessions map[*Result][]*sessionStats, status *RuleStatus, unit string,
	saving func(*Result, *sessionStats) (float64, int), describe func(*Action, int),
) []Action {
	var out []Action
	for _, result := range primary {
		var saved float64
		var count int
		var drill []Drill
		var evidence []Ref
		var priced []string
		for _, s := range sessions[result] {
			points, n := saving(result, s)
			if n == 0 {
				continue
			}
			status.Candidates++
			use := s.project(result)
			if v := use.saving(points); v > 0 {
				saved, count = saved+v, count+n
				drill = append(drill, Drill{Kind: "session", ID: s.key, Label: s.label, Value: v, StillRunning: use.active, n: n, unit: unit})
				evidence = append(evidence, Ref{Kind: "session", ID: s.key, Label: s.label})
				for _, rs := range s.sortedRuns() {
					priced = append(priced, rs.id)
				}
			}
		}
		if saved <= 0 {
			continue
		}
		row := leverAction(result, status.Rule, Effect{Value: saved, Unit: UnitPoints, Basis: basisOf(drill)})
		row.Confidence, row.OwnerKind = "low", OwnerHarness
		row.Evidence, row.Drilldown, row.priced = evidence[:min(len(evidence), maxDrill)], topDrill(drill, ""), priced
		row.Verify = "after the change, " + status.Rule + " in `agent-manager usage recommendations` should shrink"
		describe(&row, count)
		out = append(out, row)
	}
	return out
}

// ruleIdle prices turns that only wait: each stretch of idle turns could be one
// blocking wait, so the saving is the idle points less one request per stretch.
func ruleIdle(_ Inputs, primary []*Result, sessions map[*Result][]*sessionStats) ([]Action, RuleStatus) {
	status := RuleStatus{Rule: RuleIdle}
	out := sessionLever(primary, sessions, &status, "%d idle requests", func(_ *Result, s *sessionStats) (float64, int) {
		if s.idleReports == 0 {
			return 0, 0
		}
		return s.idlePoints * (1 - float64(s.idleStretches)/float64(s.idleReports)), s.idleReports
	}, func(row *Action, requests int) {
		row.Owner, row.Title = "wait tools", "Replace sleep/monitor polling with blocking waits"
		row.Text = "Replace sleep and monitor polling with one blocking wait (test-genie runs wait, agent-manager await) or park the run"
		row.Rationale = fmt.Sprintf("%d requests made mostly waits with little output; each re-reads the whole context. One blocking wait per stretch would keep one request of each.", requests)
	})
	if len(out) == 0 {
		status.Reason = fmt.Sprintf("no session made requests whose tool calls were mostly waits with under %d output tokens", idleMaxOutput)
	}
	return out, status
}

// ruleImage prices viewed images: each is written to the cache once and
// re-read by every later request of its segment (measured per run); a
// downscale saves imageScale of both.
func ruleImage(_ Inputs, primary []*Result, sessions map[*Result][]*sessionStats) ([]Action, RuleStatus) {
	status := RuleStatus{Rule: RuleImage}
	out := sessionLever(primary, sessions, &status, "%d images", func(result *Result, s *sessionStats) (float64, int) {
		var points float64
		for _, rs := range s.runs {
			creation := creationWeight(result, rs.runner, rs.model)
			for i, p := range rs.series {
				rereads := 0
				for _, later := range rs.series[i+1:] {
					if p.images == 0 || later.prompt < compactionDrop*p.prompt {
						break
					}
					rereads += later.reports
				}
				points += float64(p.images*estimatedImageTokens) * (creation + float64(rereads)) * imageScale * rs.pointsPerToken()
			}
		}
		return points, s.images
	}, func(row *Action, images int) {
		row.Owner, row.Title = "screenshot capture", "Downscale screenshots before viewing them"
		row.Text = "Downscale screenshots before an agent views them (capture at half size unless detail is needed)"
		row.Rationale = fmt.Sprintf("%d viewed images (≈%d tokens each, estimated) stay in context and are re-read by later requests until a compaction. Halving both sides quarters the tokens; half is claimed.", images, estimatedImageTokens)
	})
	if len(out) == 0 {
		status.Reason = "no session viewed images in the current window"
	}
	return out, status
}

// ruleModel compares one workload's finished runs across models: a cheaper
// model that succeeds as often is a move with a measured success rate.
func ruleModel(in Inputs, primary []*Result, _ map[*Result][]*sessionStats) ([]Action, RuleStatus) {
	status := RuleStatus{Rule: RuleModel}
	window := map[string]*Result{}
	scale := map[string]float64{}
	for _, result := range primary {
		if result.Window.Minutes > 0 && result.Window.Minutes < weekMinutes {
			continue // outcomes are stated per week
		}
		runner := RunnerFor(result.Window.Provider)
		for _, weight := range result.Calibration.Weights {
			if weight.CachedPerPoint > 0 {
				window[runner] = result
				scale[runner] = 1 / (weight.CachedPerPoint * in.Prices(weight.Runner, weight.Model).normalized().CacheRead)
				break
			}
		}
	}
	type key struct{ workload, model string }
	tallies := map[key]*tally{}
	byWorkload := map[string][]string{}
	var workloads []string
	runnerOf := map[string]string{}
	for _, outcome := range in.Outcomes {
		if outcome.Workload == "" || outcome.Model == "" || outcome.Model == "unknown" {
			continue
		}
		k := key{outcome.Workload, outcome.Model}
		t := tallies[k]
		if t == nil {
			t = &tally{}
			tallies[k] = t
			if byWorkload[outcome.Workload] == nil {
				workloads = append(workloads, outcome.Workload)
			}
			byWorkload[outcome.Workload] = append(byWorkload[outcome.Workload], outcome.Model)
		}
		t.runs++
		if outcome.Accepted {
			t.accepted++
		}
		t.cost += in.Prices(outcome.Runner, outcome.Model).normalized().cost(outcome.Tokens)
		t.runIDs = append(t.runIDs, outcome.RunID)
		runnerOf[outcome.Model] = outcome.Runner
	}
	price := func(model string) Price { return in.Prices(runnerOf[model], model).normalized() }
	weeks := Lookback.Hours() / (7 * 24)
	var out []Action
	for _, workload := range workloads {
		models := byWorkload[workload]
		if len(models) < 2 {
			continue
		}
		status.Candidates++
		sort.Strings(models)
		for _, expensive := range models {
			high := tallies[key{workload, expensive}]
			result, priced := window[runnerOf[expensive]]
			if !priced || high.cost <= 0 {
				continue
			}
			var best string
			var bestRatio float64
			for _, cheap := range models {
				low := tallies[key{workload, cheap}]
				ratio := price(cheap).CacheRead / price(expensive).CacheRead
				if cheap == expensive || ratio >= 1 || low.runs < modelMinRuns || rate(low) < rate(high)-modelSuccessSlack {
					continue
				}
				if best == "" || ratio < bestRatio {
					best, bestRatio = cheap, ratio
				}
			}
			if best == "" {
				continue
			}
			low := tallies[key{workload, best}]
			weekly := high.cost * scale[runnerOf[expensive]] / weeks
			prior := price(expensive).Prior || price(best).Prior
			row := leverAction(result, RuleModel, Effect{Value: weekly * (1 - bestRatio), Unit: UnitPoints, Basis: BasisProjected})
			row.OwnerKind, row.Owner = OwnerWorkload, workload
			row.Confidence = "low"
			if high.runs >= 5 && low.runs >= 5 && !prior {
				row.Confidence = "medium"
			}
			row.Title = shorten(fmt.Sprintf("Move %s to %s", workloadLabel(workload, workload), best), titleRunes)
			row.Text = fmt.Sprintf("Move %s from %s to %s (accepted estimate %.0f%% on %d runs vs %.0f%% on %d)", workload, expensive, best, rate(low)*100, low.runs, rate(high)*100, high.runs)
			row.Rationale = fmt.Sprintf("Over the last %d days the workload's outcome estimate was accepted on %s at least as often (within %.0f points) as on %s, at %.0f%% of its price per token; %s's runs cost ≈%.1f points a week at this window's calibration.", int(Lookback.Hours()/24), best, modelSuccessSlack*100, expensive, bestRatio*100, expensive, weekly)
			if prior {
				row.Rationale += " The price ratio is a prior (a model has no price-book entry), so the saving is an upper estimate."
			}
			for _, id := range high.runIDs[:min(len(high.runIDs), maxDrill)] {
				row.Evidence = append(row.Evidence, Ref{Kind: "run", ID: id, Label: expensive})
			}
			row.priced = high.runIDs
			row.Verify = compareVerify(result, modelFilter(best), row.Evidence, len(high.runIDs))
			out = append(out, row)
		}
	}
	if len(out) == 0 {
		status.Reason = fmt.Sprintf("no workload in the last %d days ran on a cheaper model with %d+ runs and an accepted-estimate rate within %.0f points of its pricier model (%d workloads ran on 2+ models)", int(Lookback.Hours()/24), modelMinRuns, modelSuccessSlack*100, status.Candidates)
	}
	return out, status
}

type tally struct {
	runs, accepted int
	cost           float64
	runIDs         []string
}

func rate(t *tally) float64 {
	if t.runs == 0 {
		return 0
	}
	return float64(t.accepted) / float64(t.runs)
}

// Lookback is how far back R-model reads run outcomes.
const Lookback = 14 * 24 * time.Hour

const weekMinutes = 7 * 24 * 60
