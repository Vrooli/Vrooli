package runannotations

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func (s *Service) Labels(ctx context.Context, window string) ([]LabelView, error) {
	since, until, _, err := ParseWindowRange(window, s.now())
	if err != nil {
		return nil, err
	}
	counts, err := s.store.CoverageWindowFor(ctx, since, until, s.pack.LabelHashes)
	if err != nil {
		return nil, err
	}
	out := make([]LabelView, 0, len(s.pack.Pack.Labels))
	for _, declaration := range s.pack.Pack.Labels {
		coverage := counts[declaration.ID]
		out = append(out, LabelView{
			ID: declaration.ID, Group: declaration.Group, Description: declaration.Description,
			Scope: declaration.Scope, Method: declaration.Method, Optional: declaration.Optional,
			DefinitionHash: s.pack.LabelHashes[declaration.ID], Gate: declaration.Gate,
			GateState: declaration.Gate.State(), Proven: declaration.Gate.Proven(),
			Search: declaration.Search, Friction: declaration.Friction,
			Compatibility: declaration.Compatibility, Coverage: coverage, HasData: coverage.Total() > 0,
		})
	}
	return out, nil
}

// Coverage returns the contract coverage surface. An empty store is a 200 with
// explicit not_observed label states and coverage 0, never an error. A rate is
// reported only for a proven gate; baseline/ratio stay omitted until a random
// sample baseline is measured.
func (s *Service) Coverage(ctx context.Context, window string) (CoverageResponse, error) {
	since, until, _, err := ParseWindowRange(window, s.now())
	if err != nil {
		return CoverageResponse{}, err
	}
	counts, err := s.store.CoverageWindowFor(ctx, since, until, s.pack.LabelHashes)
	if err != nil {
		return CoverageResponse{}, err
	}
	eligibleRuns, observedRuns, err := s.store.RunCoverageFor(ctx, since, until, s.pack.LabelHashes)
	if err != nil {
		return CoverageResponse{}, err
	}
	basis := "annotated_population"
	if s.census != nil {
		census, err := s.census.EligibleRuns(ctx, since, until)
		if err != nil {
			return CoverageResponse{}, err
		}
		if census >= 0 {
			eligibleRuns = census
			basis = "candidate_census"
		}
	}
	response := CoverageResponse{
		Window:        windowRange(since, until),
		Pack:          PackRef{Name: s.pack.Pack.Name, Version: s.pack.Pack.Version},
		EligibleRuns:  eligibleRuns,
		ObservedRuns:  observedRuns,
		CoverageBasis: basis,
		Weighting:     s.weighting(),
	}
	if eligibleRuns > 0 {
		response.Coverage = float64(observedRuns) / float64(eligibleRuns)
	}
	for _, declaration := range s.pack.Pack.Labels {
		coverage := counts[declaration.ID]
		positives := 0
		threshold := probabilityThreshold(declaration)
		// Positives exist only for a friction label. A classification label
		// (run.type, operator.emotion) has a probability but no friction
		// event, and counting its winning answer as one would invent a rate.
		if declaration.Friction.Enabled && threshold != nil && coverage.Total() > 0 {
			bound, above, directional := frictionScoreBound(declaration)
			selector := FrictionSelector{Threshold: *threshold, LevelBound: levelBound(bound, directional), LevelAtLeast: above}
			if directional {
				selector.ScoreBound = frictionScoreLimit(declaration)
			}
			positives, err = s.store.PositivesFor(ctx, since, until, declaration.ID, s.pack.LabelHashes[declaration.ID], selector)
			if err != nil {
				return CoverageResponse{}, err
			}
		}
		response.Labels = append(response.Labels, coverageLabel(declaration, coverage, positives))
		response.Summary.Observed += coverage.Observed
		response.Summary.Compatible += coverage.Compatible
		response.Summary.NotObserved += coverage.NotObserved
		response.Summary.Unavailable += coverage.Unavailable
		response.Summary.Stale += coverage.Stale
	}
	if response.Summary.Total() == 0 {
		response.Labels = []CoverageLabel{}
		response.Notes = append(response.Notes, "no annotations recorded in this window: coverage is unknown, not zero friction")
	}
	return response, nil
}

// coverageFromViews aggregates per-label rows for an evaluate response,
// including dry runs that never wrote to the store.
func (s *Service) coverageFromViews(views []RunView) []CoverageLabel {
	type aggregate struct {
		counts    StateCounts
		positives int
	}
	byLabel := map[string]*aggregate{}
	for _, view := range views {
		for _, annotation := range view.Labels {
			entry := byLabel[annotation.ID]
			if entry == nil {
				entry = &aggregate{}
				byLabel[annotation.ID] = entry
			}
			switch annotation.State {
			case StateObserved:
				entry.counts.Observed++
			case StateCompatible:
				entry.counts.Compatible++
			case StateNotObserved:
				entry.counts.NotObserved++
			case StateUnavailable:
				entry.counts.Unavailable++
			}
			if compiled, ok := s.pack.Label(annotation.ID); ok && compiled.label.Friction.Enabled {
				if annotationIsPositive(compiled.label, annotation) {
					entry.positives++
				}
			}
		}
	}
	out := make([]CoverageLabel, 0, len(s.pack.Pack.Labels))
	for _, declaration := range s.pack.Pack.Labels {
		entry := byLabel[declaration.ID]
		if entry == nil {
			entry = &aggregate{}
		}
		out = append(out, coverageLabel(declaration, entry.counts, entry.positives))
	}
	return out
}

func coverageLabel(declaration Label, counts StateCounts, positives int) CoverageLabel {
	state := labelCoverageState(declaration, counts)
	observed := counts.Observed + counts.Compatible
	row := CoverageLabel{
		ID: declaration.ID, Group: declaration.Group, State: state,
		Gate: gateStatus(declaration.Gate), Observed: observed, Eligible: counts.Total(),
		Evaluated: counts.Total(), Positives: positives, SearchFacet: declaration.Search.Facet,
		MinProbability: declaration.Search.MinProbability, Friction: declaration.Friction,
		BaselineState: "not_measured",
	}
	// A rate is positives among the observed population, and only for a proven
	// label. A label observed below its threshold is still observed; it is not a
	// negative and it is not a rate.
	if declaration.Friction.Enabled && declaration.Gate.Proven() && observed > 0 {
		rate := float64(positives) / float64(observed)
		row.Rate = &rate
	}
	return row
}

func labelCoverageState(declaration Label, counts StateCounts) string {
	switch {
	case counts.Total() == 0:
		return "not_observed"
	case counts.Observed+counts.Compatible > 0:
		if declaration.Gate.Proven() {
			return "proven"
		}
		return "unproven"
	case counts.Unavailable > 0 && counts.NotObserved == 0:
		return "unavailable"
	default:
		return "not_observed"
	}
}

func gateStatus(gate Gate) GateStatus {
	state := gate.State()
	reason := ""
	switch state {
	case "unmeasured":
		reason = "no measured hold-out"
		if gate.staleMeasurement != "" {
			reason = gate.staleMeasurement
		}
	case "insufficient_gold":
		reason = "gold set below min_gold"
	case "below_threshold":
		state = "unproven"
		reason = "measured precision or recall below target"
	}
	status := GateStatus{
		State: state, MinGold: gate.MinGold, Reason: reason,
		TargetPrecision: gate.Precision, TargetRecall: gate.Recall,
	}
	if gate.Measured != nil {
		status.Precision = gate.Measured.Precision
		status.Recall = gate.Measured.Recall
		status.Gold = gate.Measured.Gold
	}
	return status
}

// annotationIsPositive applies the same friction selector as the store. A
// score label is judged by the probability mass in its declared friction
// region, not by the winning level's probability: on a five-level scale the
// top level is almost never above 0.6 even when the model is sure the score is
// in the friction direction. Choice and noul labels keep the top probability.
func annotationIsPositive(label Label, annotation AnnotationView) bool {
	if !label.Friction.Enabled {
		return false
	}
	threshold := probabilityThreshold(label)
	if threshold == nil {
		return false
	}
	if bound, above, ok := frictionScoreBound(label); ok {
		// The observed level must be in the declared direction; the threshold
		// then asks whether the model gives that direction enough mass.
		if annotation.Value.Score == nil || !scoreInRegion(*annotation.Value.Score, label) {
			return false
		}
		if mass, usable := scoreRegionMass(annotation.Value, bound, above); usable {
			return mass >= *threshold
		}
		// Legacy rows stored a score without its level distribution. Judge
		// them by their top probability.
		return annotation.Probability != nil && *annotation.Probability >= *threshold
	}
	if label.Friction.MinScore != nil || label.Friction.MaxScore != nil {
		// A label without level metadata keeps the score-plus-confidence rule.
		if annotation.Value.Score == nil || annotation.Probability == nil || *annotation.Probability < *threshold {
			return false
		}
		return scoreInRegion(*annotation.Value.Score, label)
	}
	if annotation.Probability == nil || *annotation.Probability < *threshold {
		return false
	}
	return true
}

// frictionScoreBound converts the declared score ceiling/floor into an
// inclusive level-index bound. Probabilities are stored by level index while
// the reported score is index+offset.
func frictionScoreBound(label Label) (bound int, atLeast, ok bool) {
	if label.Question == nil || label.Question.Type != QuestionScore {
		return 0, false, false
	}
	levels := len(label.Question.Levels)
	if levels == 0 {
		return 0, false, false
	}
	offset := label.Question.Offset
	if label.Friction.MaxScore != nil {
		bound = int(math.Floor(*label.Friction.MaxScore - offset))
		if bound < 0 {
			return -1, false, true
		}
		if bound >= levels {
			bound = levels - 1
		}
		return bound, false, true
	}
	if label.Friction.MinScore != nil {
		bound = int(math.Ceil(*label.Friction.MinScore - offset))
		if bound < 0 {
			bound = 0
		}
		if bound >= levels {
			return levels, true, true
		}
		return bound, true, true
	}
	return 0, false, false
}

// scoreRegionMass sums the stored level probabilities inside a friction
// region. It reports false when the observation carries no level distribution,
// so an uncalibrated answer stays unknown instead of counting as a negative.
func scoreRegionMass(value Value, bound int, atLeast bool) (float64, bool) {
	if len(value.Probabilities) == 0 {
		return 0, false
	}
	mass := 0.0
	found := false
	for key, probability := range value.Probabilities {
		index, err := strconv.Atoi(strings.TrimSpace(key))
		if err != nil {
			continue
		}
		found = true
		if (atLeast && index >= bound) || (!atLeast && index <= bound) {
			mass += probability
		}
	}
	return mass, found
}

// scoreInRegion reports a raw score in the declared friction direction.
func scoreInRegion(score float64, label Label) bool {
	if label.Friction.MaxScore != nil {
		return score <= *label.Friction.MaxScore
	}
	if label.Friction.MinScore != nil {
		return score >= *label.Friction.MinScore
	}
	return false
}

// frictionScoreLimit returns the raw score bound for a legacy fallback.
func frictionScoreLimit(label Label) *float64 {
	if label.Friction.MaxScore != nil {
		return label.Friction.MaxScore
	}
	return label.Friction.MinScore
}

// levelBound turns the (bound, directional) pair into the optional selector
// field. A non-directional label carries no bound.
func levelBound(bound int, directional bool) *int {
	if !directional {
		return nil
	}
	return &bound
}

func probabilityThreshold(label Label) *float64 {
	if label.Friction.MinProbability != nil {
		return label.Friction.MinProbability
	}
	return label.Search.MinProbability
}

func (s *Service) weighting() Weighting {
	weighting := Weighting{Mode: "census+sample", SampleRate: 1, ByLabel: map[string]float64{}}
	smallest := 1.0
	for _, declaration := range s.pack.Pack.Labels {
		rate := 1.0
		if declaration.Sampling.Rate != nil {
			rate = *declaration.Sampling.Rate
		}
		weighting.ByLabel[declaration.ID] = rate
		if rate < smallest {
			smallest = rate
		}
	}
	weighting.SampleRate = smallest
	return weighting
}

func windowRange(since, until time.Time) WindowRange {
	window := WindowRange{}
	if !since.IsZero() {
		window.Since = since.UTC().Format(time.RFC3339)
	}
	if !until.IsZero() {
		window.Until = until.UTC().Format(time.RFC3339)
	}
	return window
}

// ParseWindowRange resolves a window into [since, until]. Until is now.
func ParseWindowRange(window string, now time.Time) (time.Time, time.Time, string, error) {
	since, label, err := ParseWindow(window, now)
	if err != nil {
		return time.Time{}, time.Time{}, "", err
	}
	return since, now, label, nil
}

// ParseWindow accepts "", "all", a Go duration, Nd/Nw, or RFC3339.
func ParseWindow(window string, now time.Time) (time.Time, string, error) {
	window = strings.TrimSpace(window)
	if window == "" {
		window = "7d"
	}
	if window == "all" {
		return time.Time{}, window, nil
	}
	if window == "0" {
		return time.Time{}, "all", nil
	}
	if parsed, err := time.Parse(time.RFC3339, window); err == nil {
		return parsed.UTC(), window, nil
	}
	if duration, err := time.ParseDuration(window); err == nil {
		if duration < 0 {
			return time.Time{}, "", fmt.Errorf("window %q must not be negative", window)
		}
		return now.Add(-duration).UTC(), window, nil
	}
	if len(window) > 1 {
		suffix := window[len(window)-1]
		number := window[:len(window)-1]
		var unit time.Duration
		switch suffix {
		case 'd', 'D':
			unit = 24 * time.Hour
		case 'w', 'W':
			unit = 7 * 24 * time.Hour
		}
		if unit > 0 {
			var count int
			if _, err := fmt.Sscanf(number, "%d", &count); err == nil && count >= 0 {
				return now.Add(-time.Duration(count) * unit).UTC(), window, nil
			}
		}
	}
	return time.Time{}, "", fmt.Errorf("window must be all, a duration (24h), Nd, Nw or RFC3339")
}

// DefaultGoldDraft bounds a blind review packet.
const DefaultGoldDraft = 20

// GoldDraft is a blind review packet for labelling a gold set. It carries the
// bounded state a reviewer judges, never the stored answer: seeing the model's
// probability would make the measurement circular.
type GoldDraft struct {
	Label          string    `json:"label"`
	DefinitionHash string    `json:"definition_hash"`
	Question       *Question `json:"question,omitempty"`
	Window         string    `json:"window"`
	// MinGold is the label's declared gate size. Limit is the number of runs
	// actually requested; a default draft must be large enough to measure.
	MinGold int            `json:"min_gold"`
	Limit   int            `json:"limit"`
	Runs    []GoldDraftRun `json:"runs"`
}

// GoldDraftRun is one run to judge.
type GoldDraftRun struct {
	RunID  string `json:"run_id"`
	Digest string `json:"digest"`
}

// GoldDraft builds review packets for runs already observed under the current
// label definition, so every labelled row can be scored by the benchmark.
func (s *Service) GoldDraft(ctx context.Context, labelID, window string, limit int) (GoldDraft, error) {
	compiled, ok := s.pack.Label(labelID)
	if !ok {
		return GoldDraft{}, fmt.Errorf("unknown label %q", labelID)
	}
	if limit <= 0 {
		limit = DefaultGoldDraft
	}
	// A draft smaller than the label's gate can never measure it. Default and
	// undersized requests grow to min_gold so one command produces an
	// actionable set.
	if compiled.label.Gate.MinGold > limit {
		limit = compiled.label.Gate.MinGold
	}
	if limit > 100 {
		limit = 100
	}
	since, until, windowLabel, err := ParseWindowRange(window, s.now())
	if err != nil {
		return GoldDraft{}, err
	}
	scopes, err := s.store.ObservedScopes(ctx, since, until, labelID, compiled.hash, limit)
	if err != nil {
		return GoldDraft{}, err
	}
	draft := GoldDraft{
		Label: labelID, DefinitionHash: compiled.hash, Question: compiled.label.Question,
		Window: windowLabel, MinGold: compiled.label.Gate.MinGold, Limit: limit,
		Runs: make([]GoldDraftRun, 0, len(scopes)),
	}
	for _, runID := range scopes {
		facts, err := s.features.Facts(ctx, ScopeRun, runID)
		if err != nil {
			// A deleted or unreadable run cannot be labelled; skip it rather
			// than emit a packet without evidence.
			continue
		}
		if facts.RunID == "" {
			facts.RunID = runID
		}
		// The reviewer sees a wider bounded state than the model. A label can
		// only be measured when a person can actually judge it from the packet.
		digest, err := BuildReviewDigest(facts, DefaultReviewDigestLimit)
		if err != nil {
			continue
		}
		draft.Runs = append(draft.Runs, GoldDraftRun{RunID: runID, Digest: digest})
	}
	return draft, nil
}
