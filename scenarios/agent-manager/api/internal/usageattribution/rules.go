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

// Recommendation is one deterministic rule finding.
type Recommendation struct {
	Rank        int
	Rule, Lever string
	TargetKind  string
	TargetID    string
	TargetLabel string
	Provider    string
	Pool        string
	Window      string
	WeeklySaved float64
	Confidence  string
	Action      string
	Rationale   string
	// ConsumerPoints is the target's attributed points in the window.
	ConsumerPoints float64
	Evidence       []Ref
	Metrics        []Metric
}

// Ref names one piece of evidence.
type Ref struct{ Kind, ID, Label string }

// Metric is one number behind a finding.
type Metric struct {
	Name  string
	Value float64
	Unit  string
}

// RuleStatus says whether a rule fired, and why not from the data.
type RuleStatus struct {
	Rule       string
	Fired      bool
	Rows       int
	Candidates int
	Reason     string
}

// Inputs are the facts the rules read. Results are the current windows'
// attributions; each provider's longest window is the one priced.
type Inputs struct {
	Results  []Result
	Runs     map[string]Run
	Episodes []Episode
	Outcomes []Outcome
	Prices   Prices
}

// Report is the ranked findings.
type Report struct {
	Recommendations []Recommendation
	Rules           []RuleStatus
	// Dominant is the session with the most attributed points.
	Dominant      string
	DominantLabel string
}

// Recommend evaluates every rule family and ranks the findings by weekly
// saving, largest first.
func Recommend(in Inputs) Report {
	primary := primaryResults(in.Results)
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
	rules := []func(Inputs, []*Result, map[*Result][]*sessionStats) ([]Recommendation, RuleStatus){ruleModel, ruleContext, ruleFanout, ruleFriction, ruleIdle, ruleImage}
	for _, rule := range rules {
		rows, status := rule(in, primary, sessions)
		status.Rows, status.Fired = len(rows), len(rows) > 0
		report.Recommendations = append(report.Recommendations, rows...)
		report.Rules = append(report.Rules, status)
	}
	sort.SliceStable(report.Recommendations, func(i, j int) bool {
		return report.Recommendations[i].WeeklySaved > report.Recommendations[j].WeeklySaved
	})
	for i := range report.Recommendations {
		report.Recommendations[i].Rank = i + 1
	}
	return report
}

// primaryResults keeps each provider's longest window: weekly savings are
// stated in the weekly window.
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

// weeklyFactor extrapolates use over the attributed period to a week;
// periods under a day extrapolate as a day.
func weeklyFactor(result *Result) float64 {
	hours := result.To.Sub(result.From).Hours()
	return (7 * 24) / max(hours, 24)
}

// projection is a consumer's projected weekly use (F-ORCH-8). A consumer
// still running at the end of the attributed usage keeps its rate over the
// attributed period for the rest of the week; one that stopped is assumed to
// use nothing more, so a short session is never extrapolated to a week.
type projection struct {
	points, weekly, hours float64
	active                bool
}

func project(result *Result, points float64, last time.Time) projection {
	p := projection{points: points, weekly: points, hours: result.To.Sub(result.From).Hours()}
	if !last.IsZero() && attributedEnd(result).Sub(last) <= activeGrace {
		p.active, p.weekly = true, points*weeklyFactor(result)
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
// consumer's use and caps it at the consumer's projected weekly use.
func (p projection) weeklySaving(saved float64) float64 {
	if p.points <= 0 || saved <= 0 {
		return 0
	}
	return min(saved*p.weekly/p.points, p.weekly)
}

func (p projection) merge(other projection) projection {
	return projection{points: p.points + other.points, weekly: p.weekly + other.weekly, hours: max(p.hours, other.hours), active: p.active || other.active}
}

// basis states how the weekly estimate was derived.
func (p projection) basis() string {
	if p.active {
		return fmt.Sprintf(" Basis: %s over the %.0fh attributed so far; still running, so projected at that rate to %s this week. The estimate is capped at that projection.", pts(p.points), p.hours, pts(p.weekly))
	}
	return fmt.Sprintf(" Basis: %s over the %.0fh attributed so far; it has stopped, so no further use is assumed this week and the estimate is what the lever would have saved of that use.", pts(p.points), p.hours)
}

func (p projection) metrics() []Metric {
	active := 0.0
	if p.active {
		active = 1
	}
	return []Metric{{"projected_weekly_points", p.weekly, "points"}, {"basis_hours", p.hours, "hours"}, {"still_running", active, "bool"}}
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

func newRecommendation(rule, lever string, result *Result, saved float64) Recommendation {
	return Recommendation{Rule: rule, Lever: lever, Provider: result.Window.Provider, Pool: result.Window.Pool, Window: result.Window.Name, WeeklySaved: saved}
}

func ruleContext(_ Inputs, primary []*Result, sessions map[*Result][]*sessionStats) ([]Recommendation, RuleStatus) {
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
			row := newRecommendation(RuleContext, "context", result, use.weeklySaving(s.mainPoints*cachedShare*(1-contextTargetPrompt/average)))
			row.TargetKind, row.TargetID, row.TargetLabel, row.ConsumerPoints = "session", s.key, s.label, s.points
			row.Confidence = "medium"
			if s.mainReports >= 200 && average >= 150_000 {
				row.Confidence = "high"
			}
			row.Action = fmt.Sprintf("Start a fresh session or compact %s (%dk re-read per request over %d requests)", s.label, int(average/1000), s.mainReports)
			row.Rationale = fmt.Sprintf("Each request re-reads ≈%dk tokens of context; %.0f%% of the session's quota cost is cached re-reads. Keeping the context near %dk would cut that share by %.0f%%.", int(average/1000), cachedShare*100, contextTargetPrompt/1000, (1-contextTargetPrompt/average)*100) + use.basis()
			row.Evidence = s.topRuns()
			row.Metrics = append([]Metric{{"average_prompt_tokens", average, "tokens"}, {"requests", float64(s.mainReports), "requests"}, {"cached_share", cachedShare, "ratio"}, {"window_points", s.points, "points"}}, use.metrics()...)
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		status.Reason = fmt.Sprintf("no session made %d+ provider requests averaging %dk+ context tokens in the current window", contextMinReports, contextMinPrompt/1000)
	}
	return rows, status
}

func ruleFanout(_ Inputs, primary []*Result, sessions map[*Result][]*sessionStats) ([]Recommendation, RuleStatus) {
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
			row := newRecommendation(RuleFanout, "fanout", result, use.weeklySaving(points*(1-fanoutBriefPrompt/average)*0.5))
			row.TargetKind, row.TargetID, row.TargetLabel, row.ConsumerPoints = "session", s.key, s.label, s.points
			row.Confidence = "low"
			if len(children) >= 5 {
				row.Confidence = "medium"
			}
			row.Action = fmt.Sprintf("Cap sub-agent fan-out in %s (%d sub-agents each re-reading ≈%dk tokens); give each a short brief instead of the parent context", s.label, len(children), int(average/1000))
			row.Rationale = fmt.Sprintf("%d sub-agents used %s; their requests average %dk tokens of context. About half of what exceeds a %dk brief is assumed avoidable.", len(children), pts(points), int(average/1000), fanoutBriefPrompt/1000) + use.basis()
			sort.Slice(children, func(i, j int) bool { return children[i].points > children[j].points })
			row.Evidence = []Ref{{Kind: "session", ID: s.key, Label: s.label}}
			for _, child := range children {
				if len(row.Evidence) > maxEvidence {
					break
				}
				row.Evidence = append(row.Evidence, Ref{Kind: "run", ID: child.runID, Label: "sub-agent"})
			}
			row.Metrics = append([]Metric{{"sub_agents", float64(len(children)), "runs"}, {"sub_agent_points", points, "points"}, {"average_prompt_tokens", average, "tokens"}}, use.metrics()...)
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
			row := newRecommendation(RuleIdle, "idle", result, use.weeklySaving(s.idlePoints*0.7))
			row.TargetKind, row.TargetID, row.TargetLabel, row.ConsumerPoints = "session", s.key, s.label, s.points
			row.Confidence = "low"
			if s.idleReports >= 100 {
				row.Confidence = "medium"
			}
			row.Action = fmt.Sprintf("Replace polling in %s with an event wait or park it (%d idle requests used %s)", s.label, s.idleReports, pts(s.idlePoints))
			row.Rationale = "Idle requests are provider requests whose tool calls are mostly waits with little output, or requests of a parked run. Each re-reads the whole context; most of that is avoidable with a blocking wait." + use.basis()
			row.Evidence = s.topRuns()
			row.Metrics = append([]Metric{{"idle_requests", float64(s.idleReports), "requests"}, {"idle_points", s.idlePoints, "points"}}, use.metrics()...)
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		status.Reason = fmt.Sprintf("no session had %d+ idle provider requests (mostly waits, <%d output tokens) or a parked run consuming in the current window", idleMinReports, idleMaxOutput)
	}
	return rows, status
}

func ruleImage(_ Inputs, primary []*Result, sessions map[*Result][]*sessionStats) ([]Recommendation, RuleStatus) {
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
			row := newRecommendation(RuleImage, "image", result, use.weeklySaving(imageTokens*(inputWeightOf(result)+imageRereads)*pointsPerCached*0.5))
			row.TargetKind, row.TargetID, row.TargetLabel, row.ConsumerPoints = "session", s.key, s.label, s.points
			row.Confidence = "low"
			row.Action = fmt.Sprintf("Downscale screenshots before viewing in %s (%d images, %d high-detail, ≈%dk tokens)", s.label, s.images, s.highDetail, int(imageTokens/1000))
			row.Rationale = fmt.Sprintf("Viewed images are input tokens (≈%d each, estimated) and stay in context, so each is re-read on later turns. High-detail or original-size screenshots cost the most.", estimatedImageTokens) + use.basis()
			row.Evidence = s.topRuns()
			row.Metrics = append([]Metric{{"images", float64(s.images), "images"}, {"high_detail_images", float64(s.highDetail), "images"}, {"estimated_image_tokens", imageTokens, "tokens"}}, use.metrics()...)
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
	// An episode is priced as its turns' share of its run's turns in the
	// window, times the run's attributed points: a run's friction never
	// costs more than the run consumed.
	type runUsage struct {
		result  *Result
		points  float64
		reports int
		last    time.Time
	}
	runs := map[string]*runUsage{}
	for _, result := range primary {
		for i, turn := range result.Turns {
			usage := runs[turn.RunID]
			if usage == nil {
				usage = &runUsage{result: result}
				runs[turn.RunID] = usage
			}
			usage.points += result.Points[i]
			usage.reports += turn.Reports
			if turn.At.After(usage.last) {
				usage.last = turn.At
			}
		}
	}
	type group struct {
		pattern, element, owner string
		episodes                []Episode
		runTurns                map[string]int
	}
	groups := map[string]*group{}
	runFriction := map[string]int{}
	for _, episode := range in.Episodes {
		usage := runs[episode.RunID]
		if usage == nil || usage.points <= 0 || usage.reports <= 0 || episode.Turns <= 0 {
			continue
		}
		runFriction[episode.RunID] += episode.Turns
		key := episode.Fingerprint
		if key == "" {
			key = episode.Pattern + "|" + episode.RepeatedElement
		}
		g := groups[key]
		if g == nil {
			g = &group{pattern: episode.Pattern, element: episode.RepeatedElement, owner: episode.Owner, runTurns: map[string]int{}}
			groups[key] = g
		}
		g.episodes = append(g.episodes, episode)
		g.runTurns[episode.RunID] += episode.Turns
	}
	var rows []Recommendation
	for key, g := range groups {
		status.Candidates++
		if len(g.episodes) < frictionMinEpisodes {
			continue
		}
		var points, saved float64
		var use projection
		var turns int
		var largest *runUsage // the row is stated in its largest run's window
		for id, episodeTurns := range g.runTurns {
			usage := runs[id]
			// Episodes overlap; a run's friction is at most all of its turns,
			// shared among its fingerprints by their turns.
			share := min(1, float64(runFriction[id])/float64(usage.reports)) * float64(episodeTurns) / float64(runFriction[id])
			points += share * usage.points
			runUse := project(usage.result, share*usage.points, usage.last)
			saved += runUse.weeklySaving(share * usage.points * 0.5)
			use = use.merge(runUse)
			turns += episodeTurns
			if largest == nil || usage.points > largest.points {
				largest = usage
			}
		}
		row := newRecommendation(RuleFriction, "friction", largest.result, saved)
		label := g.pattern
		if g.element != "" {
			label += ": " + truncate(g.element, 80)
		}
		row.TargetKind, row.TargetID, row.TargetLabel, row.ConsumerPoints = "episode", key, label, points
		row.Confidence = "low"
		if len(g.episodes) >= 10 {
			row.Confidence = "medium"
		}
		row.Action = fmt.Sprintf("Fix the recurring %s friction (%d episodes across %d runs, %s)", label, len(g.episodes), len(g.runTurns), pts(points))
		if g.owner != "" {
			row.Action += "; suspected owner " + g.owner
		}
		row.Rationale = "Friction episodes are durable read-model facts (re-reads, stalls, wait misuse, retries). Each is priced as its turns' share of its run's turns times the run's attributed points; half is assumed avoidable." + use.basis()
		sort.Slice(g.episodes, func(i, j int) bool { return g.episodes[i].Turns > g.episodes[j].Turns })
		for _, id := range topRunsByTurns(g.runTurns, maxEvidence/2) {
			row.Evidence = append(row.Evidence, Ref{Kind: "run", ID: id})
		}
		for _, episode := range g.episodes {
			if len(row.Evidence) >= maxEvidence {
				break
			}
			row.Evidence = append(row.Evidence, Ref{Kind: "episode", ID: episode.EpisodeID, Label: "run " + episode.RunID})
		}
		row.Metrics = append([]Metric{{"episodes", float64(len(g.episodes)), "calls"}, {"runs", float64(len(g.runTurns)), "runs"}, {"episode_turns", float64(turns), "turns"}, {"episode_points", points, "points"}}, use.metrics()...)
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].WeeklySaved > rows[j].WeeklySaved })
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
			row := newRecommendation(RuleModel, "model", result, weekly*(1-bestRatio))
			row.TargetKind, row.TargetID, row.TargetLabel, row.ConsumerPoints = "workload", workload, workload, weekly
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
			row.Metrics = []Metric{{"expensive_runs", float64(high.runs), "runs"}, {"cheaper_runs", float64(low.runs), "runs"}, {"expensive_success_rate", rate(high), "ratio"}, {"cheaper_success_rate", rate(low), "ratio"}, {"price_ratio", bestRatio, "ratio"}, {"expensive_weekly_points", weekly, "points"}}
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
