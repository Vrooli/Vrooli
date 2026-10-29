package usageattribution

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Rule families (SPEC T-6, R-1).
const (
	RuleModel    = "R-model"
	RuleContext  = "R-context"
	RuleFanout   = "R-fanout"
	RuleFriction = "R-friction"
	RuleIdle     = "R-idle"
	RuleImage    = "R-image"
)

// Rules lists every rule family in report order.
var Rules = []string{RuleModel, RuleContext, RuleFanout, RuleFriction, RuleIdle, RuleImage}

// Rule thresholds. They are deliberately coarse: a rule names a lever worth
// an operator's look, and its estimate carries the confidence.
const (
	contextMinReports    = 50
	contextMinPrompt     = 120_000
	contextTargetPrompt  = 60_000
	fanoutMinChildren    = 3
	fanoutMinChildPrompt = 40_000
	fanoutBriefPrompt    = 20_000
	idleMinReports       = 20
	idleMaxOutput        = 600
	imageMinHighDetail   = 10
	imageMinImages       = 30
	imageRereads         = 10
	frictionMinEpisodes  = 3
	modelMinRuns         = 3
	modelSuccessSlack    = 0.10
	maxEvidence          = 10
	// activeGrace is how close to the newest reading a consumer's last turn
	// must be for it to count as still running.
	activeGrace     = time.Hour
	maxFrictionRows = 10
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
	Fanout   *FanoutFacts
	Idle     *IdleFacts
	Image    *ImageFacts
	Friction *FrictionFacts
	Model    *ModelFacts
}

// ContextFacts back R-context.
type ContextFacts struct {
	AveragePromptTokens, CachedShare float64
	Requests                         int
}

// FanoutFacts back R-fanout.
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

// ModelFacts back R-model.
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
	primary := selectResults(in.Results, in.Only)
	var report Report
	var dominantShare float64
	sessions := map[*Result][]*sessionStats{}
	for _, result := range primary {
		stats := sessionStatsFor(result, in.Runs)
		sessions[result] = stats
		// Points of different subscriptions are not comparable; the dominant
		// consumer has the largest share of its own window.
		for _, s := range stats {
			if result.Observed > 0 && s.points/result.Observed > dominantShare {
				dominantShare, report.Dominant, report.DominantLabel = s.points/result.Observed, s.key, s.label
			}
		}
	}
	var levers []Recommendation
	rules := []func(Inputs, []*Result, map[*Result][]*sessionStats) ([]Recommendation, RuleStatus){ruleModel, ruleContext, ruleFanout, ruleFriction, ruleIdle, ruleImage}
	for _, rule := range rules {
		rows, status := rule(in, primary, sessions)
		status.Rows, status.Fired = len(rows), len(rows) > 0
		levers = append(levers, rows...)
		report.Rules = append(report.Rules, status)
	}
	report.Plans, report.Folded = buildPlans(levers)
	report.Problems = frictionProblems(in, primary)
	return report
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

// horizonFactor extrapolates use over the attributed period to the window's
// horizon; periods under a seventh of the horizon (a day for a week)
// extrapolate as that seventh.
func horizonFactor(result *Result) float64 {
	horizon := horizonHours(result)
	return horizon / max(result.To.Sub(result.From).Hours(), horizon/7)
}

// projection is a consumer's projected use over the window's horizon
// (F-ORCH-8). A consumer still running at the end of the attributed usage
// keeps its rate over the attributed period for the rest of the horizon; one
// that stopped is assumed to use nothing more, so a short session is never
// extrapolated to a week.
type projection struct {
	points, weekly, hours float64
	active                bool
}

func project(result *Result, points float64, last time.Time) projection {
	p := projection{points: points, weekly: points, hours: result.To.Sub(result.From).Hours()}
	if !last.IsZero() && attributedEnd(result).Sub(last) <= activeGrace {
		p.active, p.weekly = true, points*horizonFactor(result)
	}
	return p
}

// attributedEnd is the newest attributed turn; usage after the newest
// reading is pending and in no consumer's points.
func attributedEnd(result *Result) time.Time {
	var end time.Time
	for _, turn := range result.Turns {
		if turn.At.After(end) {
			end = turn.At
		}
	}
	return end
}

// weeklySaving scales a saving over the attributed period like the
// consumer's use and caps it at the consumer's projected use.
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

type childStats struct {
	runID   string
	points  float64
	reports int
	prompt  int64
}

type sessionStats struct {
	key, label                       string
	points, weighted, cachedWeighted float64
	mainPoints, mainWeighted         float64
	mainCached                       float64
	mainReports                      int
	mainPrompt                       int64
	children                         map[string]*childStats
	idleReports                      int
	idlePoints                       float64
	images, highDetail               int
	runPoints                        map[string]float64
	last                             time.Time
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
			s = &sessionStats{key: key, label: label, children: map[string]*childStats{}, runPoints: map[string]float64{}}
			byKey[key] = s
		}
		points, weighted := result.Points[i], result.Weighted[i]
		s.points += points
		s.weighted += weighted
		s.cachedWeighted += float64(turn.Tokens.CacheReadTokens)
		s.runPoints[turn.RunID] += points
		if run.Child {
			child := s.children[turn.RunID]
			if child == nil {
				child = &childStats{runID: turn.RunID}
				s.children[turn.RunID] = child
			}
			child.points += points
			child.reports += turn.Reports
			child.prompt += turn.Prompt()
		} else {
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
		if turn.At.After(s.last) {
			s.last = turn.At
		}
	}
	out := make([]*sessionStats, 0, len(byKey))
	for _, s := range byKey {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].points > out[j].points })
	return out
}

// idleTurn is polling: a turn whose tool calls are mostly waits (sleep,
// wait, monitor, agent waits) with little output, or any turn of a parked run.
func idleTurn(turn Turn, run Run) bool {
	if run.Parked {
		return true
	}
	if turn.Reports == 0 || turn.Tokens.OutputTokens/turn.Reports >= idleMaxOutput {
		return false
	}
	waits := turn.Facts.ToolKinds["wait"] + turn.Facts.ToolKinds["agent"]
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
// over the window's whole horizon, but only the time left until the reset can
// still be saved, and only by a target that goes on using quota. A target that
// stopped has nothing further to cut.
func recoverable(saved float64, u Use) float64 {
	if !u.StillRunning {
		return 0
	}
	return saved * u.RemainingShare
}

func ruleContext(in Inputs, primary []*Result, sessions map[*Result][]*sessionStats) ([]Recommendation, RuleStatus) {
	status := RuleStatus{Rule: RuleContext}
	var rows []Recommendation
	for _, result := range primary {
		for _, s := range sessions[result] {
			if s.mainReports == 0 {
				continue
			}
			status.Candidates++
			average := float64(s.mainPrompt) / float64(s.mainReports)
			if s.mainReports < contextMinReports || average < contextMinPrompt || s.mainWeighted <= 0 {
				continue
			}
			cachedShare := s.mainCached / s.mainWeighted
			use := project(result, s.points, s.last)
			row := newRecommendation(in, RuleContext, "context", result, use.weeklySaving(s.mainPoints*cachedShare*(1-contextTargetPrompt/average)), use)
			row.TargetKind, row.TargetID, row.TargetLabel = "session", s.key, s.label
			row.Confidence = "medium"
			if s.mainReports >= 200 && average >= 150_000 {
				row.Confidence = "high"
			}
			row.Action = fmt.Sprintf("Start a fresh session or compact %s (%dk re-read per request over %d requests)", s.label, int(average/1000), s.mainReports)
			row.Rationale = fmt.Sprintf("Each request re-reads ≈%dk tokens of context; %.0f%% of the session's quota cost is cached re-reads. Keeping the context near %dk would cut that share by %.0f%%.", int(average/1000), cachedShare*100, contextTargetPrompt/1000, (1-contextTargetPrompt/average)*100)
			row.Evidence = s.topRuns()
			row.Facts.Context = &ContextFacts{AveragePromptTokens: average, Requests: s.mainReports, CachedShare: cachedShare}
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		status.Reason = fmt.Sprintf("no session made %d+ provider requests averaging %dk+ context tokens in the current window", contextMinReports, contextMinPrompt/1000)
	}
	return rows, status
}

func ruleFanout(in Inputs, primary []*Result, sessions map[*Result][]*sessionStats) ([]Recommendation, RuleStatus) {
	status := RuleStatus{Rule: RuleFanout}
	var rows []Recommendation
	for _, result := range primary {
		for _, s := range sessions[result] {
			if len(s.children) == 0 {
				continue
			}
			status.Candidates++
			var points float64
			var reports int
			var prompt int64
			children := make([]*childStats, 0, len(s.children))
			for _, child := range s.children {
				points += child.points
				reports += child.reports
				prompt += child.prompt
				children = append(children, child)
			}
			if len(children) < fanoutMinChildren || reports == 0 {
				continue
			}
			average := float64(prompt) / float64(reports)
			if average < fanoutMinChildPrompt {
				continue
			}
			use := project(result, s.points, s.last)
			row := newRecommendation(in, RuleFanout, "fanout", result, use.weeklySaving(points*(1-fanoutBriefPrompt/average)*0.5), use)
			row.TargetKind, row.TargetID, row.TargetLabel = "session", s.key, s.label
			row.Confidence = "low"
			if len(children) >= 5 {
				row.Confidence = "medium"
			}
			row.Action = fmt.Sprintf("Cap sub-agent fan-out in %s (%d sub-agents each re-reading ≈%dk tokens); give each a short brief instead of the parent context", s.label, len(children), int(average/1000))
			row.Rationale = fmt.Sprintf("%d sub-agents used %s; their requests average %dk tokens of context. About half of what exceeds a %dk brief is assumed avoidable.", len(children), pts(points), int(average/1000), fanoutBriefPrompt/1000)
			sort.Slice(children, func(i, j int) bool { return children[i].points > children[j].points })
			row.Evidence = []Ref{{Kind: "session", ID: s.key, Label: s.label}}
			for _, child := range children {
				if len(row.Evidence) > maxEvidence {
					break
				}
				row.Evidence = append(row.Evidence, Ref{Kind: "run", ID: child.runID, Label: "sub-agent"})
			}
			row.Facts.Fanout = &FanoutFacts{SubAgents: len(children), SubAgentPoints: points, AveragePromptTokens: average}
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		status.Reason = fmt.Sprintf("no session had %d+ sub-agents averaging %dk+ context tokens in the current window", fanoutMinChildren, fanoutMinChildPrompt/1000)
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
			use := project(result, s.points, s.last)
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
			// An image is read once as input and then re-read from cache while
			// it stays in context; downscaling halves both.
			pointsPerCached := s.points / s.weighted
			imageTokens := float64(s.images * estimatedImageTokens)
			use := project(result, s.points, s.last)
			row := newRecommendation(in, RuleImage, "image", result, use.weeklySaving(imageTokens*(inputWeightOf(result)+imageRereads)*pointsPerCached*0.5), use)
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

// inputWeightOf is the window's most common input weight relative to cached
// input (≈20 for current API prices).
func inputWeightOf(result *Result) float64 {
	for _, weight := range result.Calibration.Weights {
		if weight.InputWeight > 0 {
			return weight.InputWeight
		}
	}
	return fallbackPrice.Input / fallbackPrice.CacheRead
}

func ruleFriction(in Inputs, primary []*Result, _ map[*Result][]*sessionStats) ([]Recommendation, RuleStatus) {
	status := RuleStatus{Rule: RuleFriction}
	problems := frictionProblems(in, primary)
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
		row.Rationale = "Friction episodes are durable read-model facts (re-reads, stalls, wait misuse, retries). Each is priced as its turns' share of its run's turns times the run's attributed points; half is assumed avoidable."
		row.Evidence = append([]Ref(nil), p.Evidence...)
		row.Facts.Friction = &FrictionFacts{Episodes: p.Episodes, Runs: p.Runs, EpisodeTurns: p.EpisodeTurns, EpisodePoints: p.Points}
		rows = append(rows, row)
	}
	if len(rows) > maxFrictionRows {
		rows = rows[:maxFrictionRows]
	}
	if len(rows) == 0 {
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
			row := newRecommendation(in, RuleModel, "model", result, weekly*(1-bestRatio), projection{points: weekly, weekly: weekly, hours: Lookback.Hours(), active: true})
			row.TargetKind, row.TargetID, row.TargetLabel = "workload", workload, workload
			row.Confidence = "low"
			if high.runs >= 5 && low.runs >= 5 {
				row.Confidence = "medium"
			}
			row.Action = fmt.Sprintf("Move %s from %s to %s (success %.0f%% on %d runs vs %.0f%% on %d)", workload, expensive, best, rate(low)*100, low.runs, rate(high)*100, high.runs)
			row.Rationale = fmt.Sprintf("Over the last %d days the same workload succeeded on %s at least as often (within %.0f points) as on %s, at %.0f%% of its price per token. %s's runs cost ≈%.1f points a week at this window's calibration.", int(Lookback.Hours()/24), best, modelSuccessSlack*100, expensive, bestRatio*100, expensive, weekly)
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
