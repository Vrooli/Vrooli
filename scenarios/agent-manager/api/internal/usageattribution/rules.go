package usageattribution

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/pricing"
)

// Rule families (SPEC T-6, R-1).
const (
	RuleModel    = "R-model"
	RuleModelMix = "R-model-mix"
	RuleContext  = "R-context"
	RuleLifetime = "R-lifetime"
	RuleFloor    = "R-floor"
	RuleFriction = "R-friction"
	RuleIdle     = "R-idle"
	RuleImage    = "R-image"
)

// Rules lists every rule family in report order.
var Rules = []string{RuleModel, RuleModelMix, RuleContext, RuleLifetime, RuleFloor, RuleFriction, RuleIdle, RuleImage}

// Rule thresholds. They are deliberately coarse: a rule names a lever worth
// an operator's look, and its estimate carries the confidence. Prompt targets
// are never constants: the measured floor (measuredFloors) is the target.
const (
	contextMinReports = 50
	contextMinPrompt  = 120_000
	// lifetimeMinRequests is the fewest requests a worker must have made to be
	// worth restarting; lifetimeMinCap is the shortest life a restart may cap
	// it at; lifetimeOrientRequests are the requests a restarted worker spends
	// re-orienting at its mean prompt size.
	lifetimeMinRequests    = 40
	lifetimeMinCap         = 20
	lifetimeOrientRequests = 3
	lifetimeMinShare       = 0.05
	floorMinShare          = 0.2
	floorMinReports        = 50
	modelMixMinSaving      = minPlanSaving
	idleMinReports         = 20
	idleMaxOutput          = 600
	imageMinHighDetail     = 10
	imageMinImages         = 30
	imageRereads           = 10
	frictionMinEpisodes    = 3
	modelMinRuns           = 3
	modelSuccessSlack      = 0.10
	maxEvidence            = 10
	// minActiveGap, activeGapFactor and defaultSessionHours define active:
	// now less the last turn within max(minActiveGap, activeGapFactor × the
	// session's median request gap); a session's expected remaining life
	// decays over the median session length, defaultSessionHours when
	// unknown.
	minActiveGap        = 15 * time.Minute
	activeGapFactor     = 3
	defaultSessionHours = 4.0
	maxFrictionRows     = 10
)

// Recommendation is one lever: a deterministic rule finding about one target.
// Levers of one target are folded into a Plan, whose saving does not count
// overlapping levers twice.
type Recommendation struct {
	Rule, Lever string
	TargetKind  string
	TargetID    string
	TargetLabel string
	Provider    string
	Pool        string
	Window      string
	// Saved is the lever's saving on its own, in points of the window over its
	// whole horizon at this rate (a week for weekly windows); Marginal is what
	// it adds after the larger levers of its group, set when the group is
	// built. Recoverable is the part of Saved still to be had before the
	// window resets.
	Saved, Marginal, Recoverable float64
	Confidence                   string
	Action                       string
	Rationale                    string
	Evidence                     []Ref
	Facts                        LeverFacts
	Use                          Use
}

// Use is the target's consumption in the stating window: ConsumerPoints
// attributed so far, projected over the window's horizon, and its share of the
// window's observed increase.
type Use struct {
	ConsumerPoints float64
	// AtThisRate is the use over the window's whole horizon if the target
	// goes on at its rate so far; it is a projection, not a forecast.
	AtThisRate float64
	// RemainingHours and RemainingShare are the time left before the window
	// resets and its share of the horizon.
	RemainingHours float64
	RemainingShare float64
	BasisHours     float64
	Share          float64
	StillRunning   bool
}

// Ref names one piece of evidence.
type Ref struct{ Kind, ID, Label string }

// LeverFacts are the numbers behind a lever; the rule sets exactly one field.
type LeverFacts struct {
	Context  *ContextFacts
	Fanout   *FanoutFacts // R-lifetime
	Idle     *IdleFacts
	Image    *ImageFacts
	Friction *FrictionFacts
	Model    *ModelFacts
}

// ContextFacts back R-context and R-floor (AveragePromptTokens is then the
// measured floor and CachedShare the floor's share of the session's points).
type ContextFacts struct {
	AveragePromptTokens, CachedShare float64
	Requests                         int
}

// FanoutFacts back R-lifetime: SubAgents counts the workers worth restarting,
// SubAgentPoints their points and AveragePromptTokens their mean prompt.
type FanoutFacts struct {
	SubAgents                           int
	SubAgentPoints, AveragePromptTokens float64
}

// IdleFacts back R-idle.
type IdleFacts struct {
	IdleRequests int
	IdlePoints   float64
}

// ImageFacts back R-image.
type ImageFacts struct {
	Images, HighDetailImages int
	EstimatedTokens          int64
}

// FrictionFacts back R-friction.
type FrictionFacts struct {
	Episodes, Runs, EpisodeTurns int
	EpisodePoints                float64
}

// ModelFacts back R-model and R-model-mix.
type ModelFacts struct {
	ExpensiveModel, CheaperModel             string
	ExpensiveRuns, CheaperRuns               int
	ExpensiveSuccessRate, CheaperSuccessRate float64
	PriceRatio, ExpensiveWeeklyPoints        float64
}

// RuleStatus says whether a rule fired, and why not from the data.
type RuleStatus struct {
	Rule       string
	Fired      bool
	Rows       int
	Candidates int
	Reason     string
}

// WindowKey names one provider window.
type WindowKey struct{ Provider, Pool, Window string }

// Inputs are the facts the rules read. Results are the current windows'
// attributions; each provider's longest window is the one priced, or only the
// window named by Only.
type Inputs struct {
	Results  []Result
	Runs     map[string]Run
	Episodes []Episode
	Outcomes []Outcome
	Prices   Prices
	Only     WindowKey
	// Now is when the rules run; zero reads the newest reading.
	Now time.Time
	// Dismissed fingerprints are friction problems the operator set aside:
	// they are listed but never recommended.
	Dismissed map[string]bool
}

// Report is the recommendations, one plan per target.
type Report struct {
	// Plans hold one entry per target, largest combined saving first.
	Plans []Plan
	// Folded are the plans whose combined saving is below minPlanSaving:
	// counted, not listed.
	Folded []Plan
	Rules  []RuleStatus
	// Problems are the recurring friction problems R-friction priced.
	Problems []Problem
	// Dominant is the session with the most attributed points.
	Dominant      string
	DominantLabel string
}

// Recommend evaluates every rule family and folds the levers by target,
// ranking the plans by combined saving, largest first.
func Recommend(in Inputs) Report {
	primary, sessions := prepare(in)
	var report Report
	var dominantShare float64
	for _, result := range primary {
		stats := sessions[result]
		// Points of different subscriptions are not comparable; the dominant
		// consumer has the largest share of its own window.
		for _, s := range stats {
			if result.Observed > 0 && s.points/result.Observed > dominantShare {
				dominantShare, report.Dominant, report.DominantLabel = s.points/result.Observed, s.key, s.label
			}
		}
	}
	var levers []Recommendation
	problems := frictionProblems(in, primary)
	friction := func(in Inputs, primary []*Result, _ map[*Result][]*sessionStats) ([]Recommendation, RuleStatus) {
		return ruleFriction(in, primary, problems)
	}
	rules := []func(Inputs, []*Result, map[*Result][]*sessionStats) ([]Recommendation, RuleStatus){ruleModel, ruleModelMix, ruleContext, ruleLifetime, ruleFloor, friction, ruleIdle, ruleImage}
	for _, rule := range rules {
		rows, status := rule(in, primary, sessions)
		status.Rows, status.Fired = len(rows), len(rows) > 0
		levers = append(levers, rows...)
		report.Rules = append(report.Rules, status)
	}
	report.Plans, report.Folded = buildPlans(levers)
	report.Problems = problems
	return report
}

// prepare selects the evaluated windows, copies them (projection settings
// belong to this evaluation, not the caller's results) and gives each its
// sessions and projection context at in.Now, so every consumer prices
// savings the same way.
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

// selectResults keeps the window named by only, or else each provider's
// longest window: weekly savings are stated in the weekly window.
func selectResults(results []Result, only WindowKey) []*Result {
	if only != (WindowKey{}) {
		var out []*Result
		for i := range results {
			w := results[i].Window
			if w.Provider == only.Provider && w.Pool == only.Pool && w.Name == only.Window {
				out = append(out, &results[i])
			}
		}
		return out
	}
	return primaryResults(results)
}

// primaryResults keeps each provider's longest window.
func primaryResults(results []Result) []*Result {
	best := map[string]int{}
	for i, result := range results {
		current, ok := best[result.Window.Provider]
		if !ok || result.Window.Minutes > results[current].Window.Minutes {
			best[result.Window.Provider] = i
		}
	}
	out := make([]*Result, 0, len(best))
	for i := range results {
		if best[results[i].Window.Provider] == i {
			out = append(out, &results[i])
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Window.Provider < out[j].Window.Provider })
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
// when now is unknown): to its reset, else the horizon less the time since it
// began.
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
	// gap is the pooled median request gap; tau the median session length
	// in hours (defaultSessionHours when unknown).
	gap time.Duration
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

// newProjCtx reads now (the newest attributed reading when unset), the median
// request gap and session length from the window's sessions, then scales the
// active sessions' future use to the window's headroom, pro rata.
func newProjCtx(result *Result, now time.Time, stats []*sessionStats) projCtx {
	if now.IsZero() { // unknown: the newest attributed use is the latest evidence of now
		for _, s := range stats {
			if s.last.After(now) {
				now = s.last
			}
		}
	}
	c := projCtx{now: now, tau: defaultSessionHours}
	var gaps []int64
	var lives []float64
	for _, s := range stats {
		for i := 1; i < len(s.times); i++ {
			gaps = append(gaps, int64(s.times[i].Sub(s.times[i-1])))
		}
		if hours := s.last.Sub(s.first).Hours(); len(s.times) > 1 && hours > 0 {
			lives = append(lives, hours)
		}
	}
	c.gap = time.Duration(median(gaps))
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

// projection is a consumer's projected use over the window's horizon
// (F-ORCH-8). A finished consumer (no turn within max(15 min, 3 × its median
// request gap) of now) projects to its points. An active one adds its rate
// over its own span, decaying with the median session length τ over the time
// R left in the window: r·τ·(1 − e^(−R/τ)), scaled to the window's headroom.
type projection struct {
	points, weekly, hours float64
	active                bool
}

func (p projection) future() float64 { return p.weekly - p.points }

func project(result *Result, points float64, first, last time.Time) projection {
	return projectSpan(result, result.proj, points, first, last, result.proj.gap)
}

func projectSpan(result *Result, c projCtx, points float64, first, last time.Time, gap time.Duration) projection {
	now := c.now
	if now.IsZero() {
		now = result.To
	}
	hours := max(last.Sub(first).Hours(), 1)
	p := projection{points: points, weekly: points, hours: hours}
	if last.IsZero() || now.Sub(last) > max(minActiveGap, activeGapFactor*gap) {
		return p
	}
	tau := c.tau
	if tau <= 0 {
		tau = defaultSessionHours
	}
	keep := 1.0
	if c.keep != nil {
		keep = *c.keep
	}
	p.active = true
	p.weekly = points + keep*points/hours*tau*(1-math.Exp(-remainingHours(result, now)/tau))
	return p
}

// weeklySaving scales a saving over the attributed period like the
// consumer's projected use and caps it at that projection.
func (p projection) weeklySaving(saved float64) float64 {
	if p.points <= 0 || saved <= 0 {
		return 0
	}
	return min(saved*p.weekly/p.points, p.weekly)
}

func (p projection) merge(other projection) projection {
	return projection{points: p.points + other.points, weekly: p.weekly + other.weekly, hours: max(p.hours, other.hours), active: p.active || other.active}
}

func (p projection) use(result *Result, now time.Time) Use {
	remaining := remainingHours(result, now)
	u := Use{ConsumerPoints: p.points, AtThisRate: p.weekly, BasisHours: p.hours, StillRunning: p.active, RemainingHours: remaining, RemainingShare: remaining / horizonHours(result)}
	if result.Observed > 0 {
		u.Share = p.points / result.Observed
	}
	return u
}

// runStat is one run's requests in the window: its first and last request
// sizes bound the growth the lifetime rule reads.
type runStat struct {
	child                   bool
	workload                string
	runner, model           string
	first, last             time.Time
	firstPrompt, lastPrompt float64
	firstSingle             bool               // firstPrompt came from a one-request turn, so it is exactly the first request
	modelPoints             map[string]float64 // points by the model of each turn (a run can switch models)
	points, weighted        float64
	reports                 int
	prompt                  int64
}

type sessionStats struct {
	key, label                       string
	points, weighted, cachedWeighted float64
	mainPoints, mainWeighted         float64
	mainCached                       float64
	mainReports                      int
	mainPrompt                       int64
	runs                             map[string]*runStat
	idleReports                      int
	idlePoints                       float64
	images, highDetail               int
	runPoints                        map[string]float64
	first, last                      time.Time
	times                            []time.Time
}

func sessionStatsFor(result *Result, runs map[string]Run) []*sessionStats {
	byKey := map[string]*sessionStats{}
	for i, turn := range result.Turns {
		run := runs[turn.RunID]
		key := run.Session
		if key == "" {
			key = turn.RunID
		}
		s := byKey[key]
		if s == nil {
			label := run.SessionLabel
			if label == "" {
				label = "session " + ShortID(key)
			}
			s = &sessionStats{key: key, label: label, runs: map[string]*runStat{}, runPoints: map[string]float64{}}
			byKey[key] = s
		}
		points, weighted := result.Points[i], result.Weighted[i]
		s.points += points
		s.weighted += weighted
		s.cachedWeighted += float64(turn.Tokens.CacheReadTokens)
		s.runPoints[turn.RunID] += points
		rs := s.runs[turn.RunID]
		if rs == nil {
			rs = &runStat{child: run.Child, workload: run.Workload, runner: turn.Runner, model: turn.Model, modelPoints: map[string]float64{}}
			s.runs[turn.RunID] = rs
		}
		rs.points += points
		rs.modelPoints[turn.Model] += points
		rs.weighted += weighted
		rs.reports += turn.Reports
		rs.prompt += turn.Prompt()
		request := float64(turn.Prompt()) / float64(max(turn.Reports, 1))
		// A turn with one request is exactly that request; a turn with many
		// reports only gives their average, so a one-request turn wins over an
		// earlier multi-request turn as the run's first-request evidence.
		single := turn.Reports == 1
		if rs.first.IsZero() || (single && !rs.firstSingle) || (single == rs.firstSingle && turn.At.Before(rs.first)) {
			rs.first, rs.firstPrompt, rs.firstSingle = turn.At, request, single
		}
		if turn.At.After(rs.last) {
			rs.last, rs.lastPrompt = turn.At, request
		}
		if !run.Child {
			s.mainPoints += points
			s.mainWeighted += weighted
			s.mainCached += float64(turn.Tokens.CacheReadTokens)
			s.mainReports += turn.Reports
			s.mainPrompt += turn.Prompt()
		}
		if idleTurn(turn, run) {
			s.idleReports += turn.Reports
			s.idlePoints += points
		}
		s.images += turn.Facts.ImagesIn
		s.highDetail += turn.Facts.ImagesInHighDetail
		s.times = append(s.times, turn.At)
		if s.first.IsZero() || turn.At.Before(s.first) {
			s.first = turn.At
		}
		if turn.At.After(s.last) {
			s.last = turn.At
		}
	}
	// Requests after the newest reading are in no points but show their
	// session is still going.
	for id, at := range result.pendingAt {
		run := runs[id]
		key := run.Session
		if key == "" {
			key = id
		}
		if s := byKey[key]; s != nil && at.After(s.last) {
			s.last = at
			s.times = append(s.times, at)
		}
	}
	out := make([]*sessionStats, 0, len(byKey))
	for _, s := range byKey {
		sort.Slice(s.times, func(i, j int) bool { return s.times[i].Before(s.times[j]) })
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].points > out[j].points })
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

// project is the session's projection under the result's settings.
func (s *sessionStats) project(result *Result) projection { return s.projection(result, result.proj) }

type floorKey struct {
	runner, model string
	child         bool
}

// measuredFloors is the median first-request prompt (the earliest one-request
// turn's prompt when the run has one, else the earliest turn's per-request
// average, an approximation that overstates the floor when that turn grew) of each runner, model and
// role (session or sub-agent) over the window's runs: the smallest context a
// run of that kind starts with, which no restart can go below.
func measuredFloors(stats []*sessionStats) map[floorKey]float64 {
	values := map[floorKey][]float64{}
	for _, s := range stats {
		for _, rs := range s.runs {
			if rs.firstPrompt > 0 {
				k := floorKey{rs.runner, rs.model, rs.child}
				values[k] = append(values[k], rs.firstPrompt)
			}
		}
	}
	out := make(map[floorKey]float64, len(values))
	for k, v := range values {
		out[k] = median(v)
	}
	return out
}

// weightsOf is a model's category weights relative to a cached-input token in
// this window: creation and input.
func weightsOf(result *Result, runner, model string) (creation, input float64) {
	for _, w := range result.Calibration.Weights {
		if w.Runner == runner && w.Model == model && w.CreationWeight > 0 {
			return w.CreationWeight, w.InputWeight
		}
	}
	return fallbackPrice.CacheCreation / fallbackPrice.CacheRead, fallbackPrice.Input / fallbackPrice.CacheRead
}

// idleTurn is polling: a turn whose tool calls are mostly waits (sleep, wait,
// monitor) with little output, or any turn of a parked run. Spawning a
// sub-agent is work, not a wait.
func idleTurn(turn Turn, run Run) bool {
	if run.Parked {
		return true
	}
	if turn.Reports == 0 || turn.Tokens.OutputTokens/turn.Reports >= idleMaxOutput {
		return false
	}
	waits := turn.Facts.ToolKinds["wait"]
	return waits > 0 && 2*waits >= turn.Facts.ToolCalls
}

func (s *sessionStats) topRuns() []Ref {
	ids := make([]string, 0, len(s.runPoints))
	for id := range s.runPoints {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return s.runPoints[ids[i]] > s.runPoints[ids[j]] })
	refs := []Ref{{Kind: "session", ID: s.key, Label: s.label}}
	for _, id := range ids {
		if len(refs) > maxEvidence {
			break
		}
		refs = append(refs, Ref{Kind: "run", ID: id})
	}
	return refs
}

func newRecommendation(in Inputs, rule, lever string, result *Result, saved float64, use projection) Recommendation {
	u := use.use(result, in.Now)
	return Recommendation{Rule: rule, Lever: lever, Provider: result.Window.Provider, Pool: result.Window.Pool, Window: result.Window.Name, Saved: saved, Recoverable: recoverable(saved, u), Use: u}
}

// recoverable is the part of a saving still to be had: the saving is stated
// over the window's whole horizon, but only its future share (the projected
// use beyond the points already spent) can still be cut, and only by a target
// that is still active. A finished target has nothing further to cut.
func recoverable(saved float64, u Use) float64 {
	if !u.StillRunning || u.AtThisRate <= 0 {
		return 0
	}
	return saved * max(0, u.AtThisRate-u.ConsumerPoints) / u.AtThisRate
}

// rebuildCost is what restarting a run costs in cached-input-equivalent
// tokens: the floor written to cache plus a few orientation requests at the
// run's mean prompt.
func rebuildCost(floor, creationWeight, meanPrompt float64) float64 {
	return floor*creationWeight + lifetimeOrientRequests*meanPrompt
}

func ruleContext(in Inputs, primary []*Result, sessions map[*Result][]*sessionStats) ([]Recommendation, RuleStatus) {
	status := RuleStatus{Rule: RuleContext}
	var rows []Recommendation
	for _, result := range primary {
		floors := measuredFloors(sessions[result])
		for _, s := range sessions[result] {
			if s.mainReports == 0 {
				continue
			}
			status.Candidates++
			average := float64(s.mainPrompt) / float64(s.mainReports)
			if s.mainReports < contextMinReports || average < contextMinPrompt || s.mainWeighted <= 0 {
				continue
			}
			runner, model := s.mainRunnerModel()
			floor := floors[floorKey{runner, model, false}]
			if floor <= 0 || floor >= average {
				continue
			}
			creation, _ := weightsOf(result, runner, model)
			cachedShare := s.mainCached / s.mainWeighted
			pointsPerToken := s.mainPoints / s.mainWeighted
			// One restart rebuilds the floor; the saving is the cached re-reads
			// above the floor, less that rebuild.
			saved := s.mainPoints*cachedShare*(1-floor/average) - rebuildCost(floor, creation, average)*pointsPerToken
			if saved <= 0 {
				continue
			}
			use := s.project(result)
			row := newRecommendation(in, RuleContext, "context", result, use.weeklySaving(saved), use)
			row.TargetKind, row.TargetID, row.TargetLabel = "session", s.key, s.label
			row.Confidence = "medium"
			if s.mainReports >= 200 && average >= 150_000 {
				row.Confidence = "high"
			}
			row.Action = fmt.Sprintf("Start a fresh session or compact %s (%dk re-read per request over %d requests)", s.label, int(average/1000), s.mainReports)
			row.Rationale = fmt.Sprintf("Each request re-reads ≈%dk tokens of context; %.0f%% of the session's quota cost is cached re-reads. A fresh session starts at the measured floor of ≈%dk, which would cut that share by %.0f%%, less the cost of rebuilding the floor.", int(average/1000), cachedShare*100, int(floor/1000), (1-floor/average)*100)
			row.Evidence = s.topRuns()
			row.Facts.Context = &ContextFacts{AveragePromptTokens: average, Requests: s.mainReports, CachedShare: cachedShare}
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		status.Reason = fmt.Sprintf("no session made %d+ provider requests averaging %dk+ context tokens, above its measured floor and by more than a rebuild costs, in the current window", contextMinReports, contextMinPrompt/1000)
	}
	return rows, status
}

// mainRunnerModel is the runner and model of the session's largest
// non-sub-agent run.
func (s *sessionStats) mainRunnerModel() (runner, model string) {
	var best float64 = -1
	for _, rs := range s.runs {
		if !rs.child && rs.points > best {
			best, runner, model = rs.points, rs.runner, rs.model
		}
	}
	return runner, model
}

// ruleLifetime prices restarting each sub-agent at a request cap. A worker's
// prompt grows by g = (L − S)/n a request from its own first request S to L
// over n requests (the measured floor F only prices the rebuild, so a worker
// that starts above F is not credited for that excess); capping its life at k requests saves g·n(n − k)/2 tokens and costs
// (n/k − 1) rebuilds. The cap k ≥ 20 that nets the most is chosen; a worker
// whose best net saving is not positive is not reported.
func ruleLifetime(in Inputs, primary []*Result, sessions map[*Result][]*sessionStats) ([]Recommendation, RuleStatus) {
	status := RuleStatus{Rule: RuleLifetime}
	var rows []Recommendation
	for _, result := range primary {
		floors := measuredFloors(sessions[result])
		for _, s := range sessions[result] {
			var workers, caps []float64
			var saved, points, prompt float64
			var ids []string
			for id, rs := range s.runs {
				if !rs.child || rs.reports < lifetimeMinRequests || rs.weighted <= 0 {
					continue
				}
				status.Candidates++
				floor := floors[floorKey{rs.runner, rs.model, true}]
				if floor <= 0 || rs.lastPrompt <= floor {
					continue
				}
				n := float64(rs.reports)
				mean := float64(rs.prompt) / n
				creation, _ := weightsOf(result, rs.runner, rs.model)
				rebuild := rebuildCost(floor, creation, mean)
				growth := (rs.lastPrompt - rs.firstPrompt) / n // credit only growth the worker made from its own start
				bestCap, best := 0.0, 0.0
				for k := float64(lifetimeMinCap); k < n; k++ {
					if net := growth*n*(n-k)/2 - (n/k-1)*rebuild; net > best {
						bestCap, best = k, net
					}
				}
				if best/rs.weighted < lifetimeMinShare {
					continue
				}
				saved += best * rs.points / rs.weighted
				points += rs.points
				prompt += mean
				caps = append(caps, bestCap)
				workers = append(workers, rs.points)
				ids = append(ids, id)
			}
			if len(workers) == 0 {
				continue
			}
			sort.Slice(ids, func(a, b int) bool { return s.runs[ids[a]].points > s.runs[ids[b]].points })
			use := s.project(result)
			row := newRecommendation(in, RuleLifetime, "lifetime", result, use.weeklySaving(min(saved, points)), use)
			row.TargetKind, row.TargetID, row.TargetLabel = "session", s.key, s.label
			row.Confidence = "low"
			if len(workers) >= 5 {
				row.Confidence = "medium"
			}
			restartAfter := int(median(caps))
			row.Action = fmt.Sprintf("Restart sub-agents of %s after about %d requests (%d sub-agents grow past their starting context)", s.label, restartAfter, len(workers))
			row.Rationale = fmt.Sprintf("%d sub-agents used %s; each starts at the measured floor and its prompt grows with every request. Restarting at the request cap saves the growth beyond it, less the cost of rebuilding the floor and re-orienting after each restart; the cap is the one that nets the most.", len(workers), pts(points))
			row.Evidence = []Ref{{Kind: "session", ID: s.key, Label: s.label}}
			for _, id := range ids {
				if len(row.Evidence) > maxEvidence {
					break
				}
				row.Evidence = append(row.Evidence, Ref{Kind: "run", ID: id, Label: "sub-agent"})
			}
			row.Facts.Fanout = &FanoutFacts{SubAgents: len(workers), SubAgentPoints: points, AveragePromptTokens: prompt / float64(len(workers))}
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		status.Reason = fmt.Sprintf("no sub-agent made %d+ requests while growing past its measured floor by more than a restart's rebuild costs in the current window", lifetimeMinRequests)
	}
	return rows, status
}

// ruleModelMix reads every session's work on a dearer tier, interactive
// sessions included. Whether the cheaper tier would do the work as well is
// untested here, so the saving is a range from nothing to the full price gap
// (points x (1 - PriceRatio), in the facts and the rationale) and the lever
// claims none (Saved is its lower end, 0) so an unproven lever never ranks a
// plan; the confidence is "unproven" and the price ratio is a tier prior.
func ruleModelMix(in Inputs, primary []*Result, sessions map[*Result][]*sessionStats) ([]Recommendation, RuleStatus) {
	status := RuleStatus{Rule: RuleModelMix}
	var rows []Recommendation
	for _, result := range primary {
		for _, s := range sessions[result] {
			type tier struct {
				points float64
				runs   int
			}
			byModel := map[string]*tier{}
			for _, rs := range s.runs {
				if rs.workload != "" { // workload-tagged runs are R-model's
					continue
				}
				for model, points := range rs.modelPoints {
					if _, _, ok := pricing.CheaperTier(model); !ok {
						continue
					}
					if byModel[model] == nil {
						byModel[model] = &tier{}
					}
					byModel[model].points += points
					byModel[model].runs++
				}
			}
			models := make([]string, 0, len(byModel))
			for model := range byModel {
				models = append(models, model)
			}
			sort.Strings(models)
			for _, model := range models {
				status.Candidates++
				marker, ratio, _ := pricing.CheaperTier(model)
				used := byModel[model].points
				upper := used * (1 - ratio)
				use := s.project(result)
				if use.weeklySaving(upper) < modelMixMinSaving {
					continue
				}
				row := newRecommendation(in, RuleModelMix, "model-mix", result, 0, use)
				row.TargetKind, row.TargetID, row.TargetLabel = "session", s.key, s.label
				row.Confidence = "unproven"
				row.Action = fmt.Sprintf("Try %s instead of %s for the lighter work in %s", marker, model, s.label)
				row.Rationale = fmt.Sprintf("%s used %s of %s's quota. %s costs %.0f%% of %s per token, but that ratio is a tier prior, not fitted on this window, and nothing here shows %s does this work as well, so the saving is anywhere from 0 to %s.", model, pts(used), s.label, marker, ratio*100, model, marker, pts(upper))
				row.Evidence = s.topRuns()
				row.Facts.Model = &ModelFacts{ExpensiveModel: model, CheaperModel: marker, ExpensiveRuns: byModel[model].runs, PriceRatio: ratio, ExpensiveWeeklyPoints: use.weeklySaving(used)}
				rows = append(rows, row)
			}
		}
	}
	if len(rows) == 0 {
		status.Reason = "no session or sub-agent outside a workload used a model with a cheaper tier (opus, sol, terra) enough to matter in the current window"
	}
	return rows, status
}

// ruleFloor reports how much of a session's points are requests that carry
// nothing but the floor. How much of the floor is trimmable is unknown, so no
// saving is claimed: the row is informational.
func ruleFloor(in Inputs, primary []*Result, sessions map[*Result][]*sessionStats) ([]Recommendation, RuleStatus) {
	status := RuleStatus{Rule: RuleFloor}
	var rows []Recommendation
	for _, result := range primary {
		floors := measuredFloors(sessions[result])
		for _, s := range sessions[result] {
			var floorWeighted, requests float64
			var runner, model string
			var top float64
			for _, rs := range s.runs {
				floor := floors[floorKey{rs.runner, rs.model, rs.child}]
				// Weighted like s.weighted (a cached-input token is 1): the
				// first request writes the floor to cache, the rest re-read it.
				creation, _ := weightsOf(result, rs.runner, rs.model)
				used := min(floor, float64(rs.prompt)/float64(max(rs.reports, 1)))
				if rs.reports > 0 {
					floorWeighted += used * (creation + float64(rs.reports-1))
				}
				requests += float64(rs.reports)
				if floor > top {
					top, runner, model = floor, rs.runner, rs.model
				}
			}
			if requests < floorMinReports || s.weighted <= 0 {
				continue
			}
			status.Candidates++
			share := floorWeighted / s.weighted
			if share < floorMinShare {
				continue
			}
			use := s.project(result)
			row := newRecommendation(in, RuleFloor, "floor", result, 0, use)
			row.TargetKind, row.TargetID, row.TargetLabel = "session", s.key, s.label
			row.Confidence = "low"
			row.Action = fmt.Sprintf("Check what %s's %dk starting context contains (%s %s)", s.label, int(top/1000), runner, model)
			row.Rationale = fmt.Sprintf("About %.0f%% of %s's points (≈%s) are the %dk-token floor every request starts with (%d requests × the measured first-request size). How much of the floor can be trimmed is unknown, so no saving is claimed.", share*100, s.label, pts(share*s.points), int(top/1000), int(requests))
			row.Evidence = s.topRuns()
			row.Facts.Context = &ContextFacts{AveragePromptTokens: top, Requests: int(requests), CachedShare: share}
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		status.Reason = fmt.Sprintf("no session made %d+ requests with the measured floor making up %.0f%%+ of its tokens in the current window", floorMinReports, floorMinShare*100)
	}
	return rows, status
}

func ruleIdle(in Inputs, primary []*Result, sessions map[*Result][]*sessionStats) ([]Recommendation, RuleStatus) {
	status := RuleStatus{Rule: RuleIdle}
	var rows []Recommendation
	for _, result := range primary {
		for _, s := range sessions[result] {
			if s.idleReports == 0 {
				continue
			}
			status.Candidates++
			if s.idleReports < idleMinReports || s.idlePoints <= 0 {
				continue
			}
			use := s.project(result)
			row := newRecommendation(in, RuleIdle, "idle", result, use.weeklySaving(s.idlePoints*0.7), use)
			row.TargetKind, row.TargetID, row.TargetLabel = "session", s.key, s.label
			row.Confidence = "low"
			if s.idleReports >= 100 {
				row.Confidence = "medium"
			}
			row.Action = fmt.Sprintf("Replace polling in %s with an event wait or park it (%d idle requests used %s)", s.label, s.idleReports, pts(s.idlePoints))
			row.Rationale = "Idle requests are provider requests whose tool calls are mostly waits with little output, or requests of a parked run. Each re-reads the whole context; most of that is avoidable with a blocking wait."
			row.Evidence = s.topRuns()
			row.Facts.Idle = &IdleFacts{IdleRequests: s.idleReports, IdlePoints: s.idlePoints}
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		status.Reason = fmt.Sprintf("no session had %d+ idle provider requests (mostly waits, <%d output tokens) or a parked run consuming in the current window", idleMinReports, idleMaxOutput)
	}
	return rows, status
}

func ruleImage(in Inputs, primary []*Result, sessions map[*Result][]*sessionStats) ([]Recommendation, RuleStatus) {
	status := RuleStatus{Rule: RuleImage}
	var rows []Recommendation
	for _, result := range primary {
		for _, s := range sessions[result] {
			if s.images == 0 {
				continue
			}
			status.Candidates++
			if (s.highDetail < imageMinHighDetail && s.images < imageMinImages) || s.weighted <= 0 {
				continue
			}
			// An image is written to the cache once and then re-read while it
			// stays in context; downscaling halves both.
			pointsPerCached := s.points / s.weighted
			imageTokens := float64(s.images * estimatedImageTokens)
			use := s.project(result)
			runner, model := s.mainRunnerModel()
			creation, _ := weightsOf(result, runner, model)
			row := newRecommendation(in, RuleImage, "image", result, use.weeklySaving(imageTokens*(creation+imageRereads)*pointsPerCached*0.5), use)
			row.TargetKind, row.TargetID, row.TargetLabel = "session", s.key, s.label
			row.Confidence = "low"
			row.Action = fmt.Sprintf("Downscale screenshots before viewing in %s (%d images, %d high-detail, ≈%dk tokens)", s.label, s.images, s.highDetail, int(imageTokens/1000))
			row.Rationale = fmt.Sprintf("Viewed images are input tokens (≈%d each, estimated) and stay in context, so each is re-read on later turns. High-detail or original-size screenshots cost the most.", estimatedImageTokens)
			row.Evidence = s.topRuns()
			row.Facts.Image = &ImageFacts{Images: s.images, HighDetailImages: s.highDetail, EstimatedTokens: int64(imageTokens)}
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		status.Reason = fmt.Sprintf("no session viewed %d+ images or %d+ high-detail images in the current window", imageMinImages, imageMinHighDetail)
	}
	return rows, status
}

func ruleFriction(in Inputs, primary []*Result, problems []Problem) ([]Recommendation, RuleStatus) {
	status := RuleStatus{Rule: RuleFriction}
	status.Candidates = len(problems)
	var rows []Recommendation
	for _, p := range problems {
		if in.Dismissed[p.Fingerprint] {
			continue
		}
		row := newRecommendation(in, RuleFriction, "friction", p.result, p.EstSaved, p.use)
		row.TargetKind, row.TargetID, row.TargetLabel = "friction", p.Fingerprint, p.Description
		row.Confidence = "low"
		if p.Episodes >= 10 {
			row.Confidence = "medium"
		}
		row.Action = fmt.Sprintf("Fix the recurring problem: %s (%d episodes across %d runs, %s; owner: %s)", strings.TrimSuffix(p.Description, "."), p.Episodes, p.Runs, pts(p.Points), p.Owner)
		row.Rationale = fmt.Sprintf("These are %s episodes, a pattern shown because it passes the precision gate on labelled real runs. Each is priced as its turns' share of its run's turns times the run's attributed points; half is assumed avoidable.", p.Pattern)
		row.Evidence = append([]Ref(nil), p.Evidence...)
		row.Facts.Friction = &FrictionFacts{Episodes: p.Episodes, Runs: p.Runs, EpisodeTurns: p.EpisodeTurns, EpisodePoints: p.Points}
		rows = append(rows, row)
	}
	if len(rows) > maxFrictionRows {
		// Agent-behaviour problems have no owner to file a bug with; their plan
		// card is their only action, so they are kept past the cap.
		kept := rows[:maxFrictionRows:maxFrictionRows]
		for _, row := range rows[maxFrictionRows:] {
			if problem, ok := problemByFingerprint(problems, row.TargetID); ok && problem.OwnerKind == OwnerAgent {
				kept = append(kept, row)
			}
		}
		rows = kept
	}
	if len(rows) == 0 && status.Candidates > 0 {
		status.Reason = fmt.Sprintf("all %d recurring friction problems are dismissed", status.Candidates)
	} else if len(rows) == 0 {
		status.Reason = fmt.Sprintf("no friction fingerprint recurred %d+ times in runs that consumed quota in the current window (%d episodes read)", frictionMinEpisodes, len(in.Episodes))
	}
	return rows, status
}

func topRunsByTurns(values map[string]int, limit int) []string {
	weights := make(map[string]float64, len(values))
	for key, value := range values {
		weights[key] = float64(value)
	}
	return topKeys(weights, limit)
}

func ruleModel(in Inputs, primary []*Result, _ map[*Result][]*sessionStats) ([]Recommendation, RuleStatus) {
	status := RuleStatus{Rule: RuleModel}
	// The scale of each runner's window prices a run's tokens in points.
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
			byWorkload[outcome.Workload] = append(byWorkload[outcome.Workload], outcome.Model)
		}
		t.runs++
		if outcome.Succeeded {
			t.succeeded++
		}
		t.cost += in.Prices(outcome.Runner, outcome.Model).normalized().cost(outcome.Tokens)
		t.runIDs = append(t.runIDs, outcome.RunID)
		runnerOf[outcome.Model] = outcome.Runner
	}
	cachedPrice := func(model string) float64 { return in.Prices(runnerOf[model], model).normalized().CacheRead }
	isPrior := func(model string) bool { return in.Prices(runnerOf[model], model).normalized().Prior }
	weeks := Lookback.Hours() / (7 * 24)
	var rows []Recommendation
	for workload, models := range byWorkload {
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
				ratio := cachedPrice(cheap) / cachedPrice(expensive)
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
			// The workload's cost is already a weekly rate over the outcome history.
			row := newRecommendation(in, RuleModel, "model", result, weekly*(1-bestRatio), projection{points: weekly * (1 - remainingHours(result, in.Now)/horizonHours(result)), weekly: weekly, hours: Lookback.Hours(), active: true})
			row.TargetKind, row.TargetID, row.TargetLabel = "workload", workload, workload
			row.Confidence = "low"
			if high.runs >= 5 && low.runs >= 5 {
				row.Confidence = "medium"
			}
			// A ratio built on a tier prior is a guess: keep the point estimate,
			// never claim more than low confidence, and word it as a ceiling.
			prior := isPrior(expensive) || isPrior(best)
			saving := ""
			if prior {
				row.Confidence = "low"
				saving = fmt.Sprintf(", saving up to ≈%.1f points a week", weekly*(1-bestRatio))
			}
			row.Action = fmt.Sprintf("Move %s from %s to %s%s (success %.0f%% on %d runs vs %.0f%% on %d)", workload, expensive, best, saving, rate(low)*100, low.runs, rate(high)*100, high.runs)
			row.Rationale = fmt.Sprintf("Over the last %d days the same workload succeeded on %s at least as often (within %.0f points) as on %s, at %.0f%% of its price per token. %s's runs cost ≈%.1f points a week at this window's calibration.", int(Lookback.Hours()/24), best, modelSuccessSlack*100, expensive, bestRatio*100, expensive, weekly)
			if prior {
				row.Rationale += " The price ratio is a prior, not measured: at least one of these models has no price-book entry, so the saving is an upper estimate."
			}
			row.Evidence = []Ref{{Kind: "workload", ID: workload, Label: workload}}
			for _, id := range high.runIDs {
				if len(row.Evidence) > maxEvidence {
					break
				}
				row.Evidence = append(row.Evidence, Ref{Kind: "run", ID: id, Label: expensive})
			}
			row.Facts.Model = &ModelFacts{ExpensiveModel: expensive, CheaperModel: best, ExpensiveRuns: high.runs, CheaperRuns: low.runs, ExpensiveSuccessRate: rate(high), CheaperSuccessRate: rate(low), PriceRatio: bestRatio, ExpensiveWeeklyPoints: weekly}
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		status.Reason = fmt.Sprintf("no workload in the last %d days ran on a cheaper model with %d+ runs and a success rate within %.0f points of its pricier model (%d workloads ran on 2+ models)", int(Lookback.Hours()/24), modelMinRuns, modelSuccessSlack*100, status.Candidates)
	}
	return rows, status
}

type tally struct {
	runs, succeeded int
	cost            float64
	runIDs          []string
}

func rate(t *tally) float64 {
	if t.runs == 0 {
		return 0
	}
	return float64(t.succeeded) / float64(t.runs)
}

func topKeys(values map[string]float64, limit int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return values[keys[i]] > values[keys[j]] })
	if len(keys) > limit {
		keys = keys[:limit]
	}
	return keys
}

func truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}

// Lookback is how far back R-model reads run outcomes.
const Lookback = 14 * 24 * time.Hour

const weekMinutes = 7 * 24 * 60

func problemByFingerprint(problems []Problem, fingerprint string) (Problem, bool) {
	for _, p := range problems {
		if p.Fingerprint == fingerprint {
			return p, true
		}
	}
	return Problem{}, false
}
