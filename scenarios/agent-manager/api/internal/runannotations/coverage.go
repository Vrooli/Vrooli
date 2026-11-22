package runannotations

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

func (s *Service) Labels(ctx context.Context, window string) ([]LabelView, error) {
	since, until, _, err := ParseWindowRange(window, s.now())
	if err != nil {
		return nil, err
	}
	accumulated, err := s.accumulateStored(ctx, since, until)
	if err != nil {
		return nil, err
	}
	out := make([]LabelView, 0, len(s.pack.Pack.Labels))
	for _, declaration := range s.pack.Pack.Labels {
		acc := accumulated[declaration.ID]
		hash := s.pack.LabelHashes[declaration.ID]
		compiled, _ := s.pack.Label(declaration.ID)
		best := s.pack.LabelStatusFor(compiled, acc.keys())
		out = append(out, LabelView{
			ID: declaration.ID, Group: declaration.Group, Description: declaration.Description,
			Scope: declaration.Scope, Method: declaration.Method, Optional: declaration.Optional,
			Role: s.labelRole(declaration), EscalateRole: s.labelEscalateRole(declaration),
			DefinitionHash: hash, Gate: declaration.Gate,
			GateState: best.State, Proven: best.State == GateProven, GateStatus: best,
			Search: declaration.Search, Friction: declaration.Friction,
			Compatibility: declaration.Compatibility, Coverage: acc.counts, HasData: acc.counts.Total() > 0,
		})
	}
	return out, nil
}

// labelAccumulator is one label's coverage while the latest rows are read:
// state counts for the current definition, and per (model, resource role) the
// observations and friction positives, so a proof is applied to exactly the
// observations it measured.
type labelAccumulator struct {
	counts StateCounts
	models map[ModelKey]*modelAccumulator
}

type modelAccumulator struct {
	observed  int
	positives int
	// proven and provenPositives are the observations whose own proof clears the
	// gate. They are counted one by one: a composite over labels is proven
	// per observation, by the inputs that observation read.
	proven          int
	provenPositives int
}

func (a *labelAccumulator) keys() []ModelKey {
	if a == nil {
		return nil
	}
	keys := make([]ModelKey, 0, len(a.models))
	for key := range a.models {
		keys = append(keys, key)
	}
	sortModelKeys(keys)
	return keys
}

// accumulate folds one latest row into its label's coverage. A row from another
// definition revision is stale evidence of earlier work, never an observation of
// the current label.
func (s *Service) accumulate(acc map[string]*labelAccumulator, row Annotation) {
	compiled, ok := s.pack.Label(row.LabelID)
	if !ok {
		return
	}
	entry := acc[row.LabelID]
	if entry == nil {
		entry = &labelAccumulator{models: map[ModelKey]*modelAccumulator{}}
		acc[row.LabelID] = entry
	}
	if row.DefinitionHash != compiled.hash {
		entry.counts.Stale++
		return
	}
	addCount(&entry.counts, row.State)
	if row.State != StateObserved && row.State != StateCompatible {
		return
	}
	model, role := ProofKey(row)
	key := ModelKey{Model: model, ResourceRole: role}
	group := entry.models[key]
	if group == nil {
		group = &modelAccumulator{}
		entry.models[key] = group
	}
	group.observed++
	proven := s.pack.StatusFor(compiled, row).State == GateProven
	if proven {
		group.proven++
	}
	if compiled.label.Friction.Enabled && s.frictionPositive(compiled.label, row) {
		group.positives++
		if proven {
			group.provenPositives++
		}
	}
}

// accumulateStored reads the latest row per (scope, label) in the window once and
// folds it into every label's coverage. Only labels that report friction, and
// composites over labels, have their stored values decoded.
func (s *Service) accumulateStored(ctx context.Context, since, until time.Time) (map[string]*labelAccumulator, error) {
	acc := map[string]*labelAccumulator{}
	want := func(labelID string) bool {
		compiled, ok := s.pack.Label(labelID)
		// A friction label's vector is scored; a derived composite's input
		// proof keys decide whether each observation is proven.
		return ok && (compiled.label.Friction.Enabled || compiled.derived())
	}
	err := s.store.ForEachLatest(ctx, since, until, want, func(row Annotation) error {
		s.accumulate(acc, row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, declaration := range s.pack.Pack.Labels {
		if acc[declaration.ID] == nil {
			acc[declaration.ID] = &labelAccumulator{models: map[ModelKey]*modelAccumulator{}}
		}
	}
	return acc, nil
}

// frictionPositive reports whether an observation is a friction event under the
// calibration of the model that produced it: the stored vector is tempered with
// that model's temperature and compared with that model's measured threshold.
// An observation with nothing to score is neither positive nor negative here.
func (s *Service) frictionPositive(label Label, row Annotation) bool {
	if _, ok := s.pack.Label(label.ID); !ok || !label.Friction.Enabled {
		return false
	}
	temperature, measured := calibrationOf(label, row)
	return isFrictionPositive(label, row, temperature, measured)
}

// isFrictionPositive is the one definition of a friction positive: the
// calibrated score reaches the threshold. The threshold is the one measured for
// the model when there is one, else the label's declared one.
func isFrictionPositive(label Label, row Annotation, temperature, measuredThreshold float64) bool {
	if !label.Friction.Enabled {
		return false
	}
	threshold := measuredThreshold
	if threshold <= 0 {
		declared := probabilityThreshold(label)
		if declared == nil {
			return false
		}
		threshold = *declared
	}
	sample, ok := sampleFor(label, row)
	if !ok {
		return false
	}
	return sample.Score(temperature) >= threshold
}

// Coverage returns the contract coverage surface. An empty store is a 200 with
// explicit not_observed label states and coverage 0, never an error. A rate is
// reported only over observations whose own model's hold-out proof clears the
// gate; baseline/ratio stay omitted until a random sample baseline is measured.
func (s *Service) Coverage(ctx context.Context, window string) (CoverageResponse, error) {
	since, until, _, err := ParseWindowRange(window, s.now())
	if err != nil {
		return CoverageResponse{}, err
	}
	accumulated, err := s.accumulateStored(ctx, since, until)
	if err != nil {
		return CoverageResponse{}, err
	}
	eligibleRuns, observedRuns, err := s.store.RunCoverageFor(ctx, since, until, s.pack.LabelHashes)
	if err != nil {
		return CoverageResponse{}, err
	}
	provenance, err := s.store.ProvenanceSummary(ctx, since, until, s.pack.Pack.Runtime.Provenance.Hash())
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
	if provenance.RunsAnnotated > 0 {
		response.Provenance = &provenance
	}
	if eligibleRuns > 0 {
		response.Coverage = float64(observedRuns) / float64(eligibleRuns)
	}
	for _, declaration := range s.pack.Pack.Labels {
		acc := accumulated[declaration.ID]
		row := s.coverageLabel(declaration, acc)
		if declaration.Population == PopulationHumanOperator && response.Provenance != nil {
			applyPopulation(&row, declaration, provenance)
		}
		response.Labels = append(response.Labels, row)
		response.Summary.Observed += acc.counts.Observed
		response.Summary.Compatible += acc.counts.Compatible
		response.Summary.NotObserved += acc.counts.NotObserved
		response.Summary.Unavailable += acc.counts.Unavailable
		response.Summary.Stale += acc.counts.Stale
	}
	if response.Summary.Total() == 0 {
		response.Labels = []CoverageLabel{}
		response.Notes = append(response.Notes, "no annotations recorded in this window: coverage is unknown, not zero friction")
	}
	return response, nil
}

// applyPopulation reports the label's coverage against the people it is about:
// the runs (or messages) a person wrote, not every annotated run. A mood label's
// denominator is human text; workflow briefs are not a population it describes.
func applyPopulation(row *CoverageLabel, label Label, provenance ProvenanceSummary) {
	population := &Population{Basis: PopulationHumanOperator, Unit: "run", Eligible: provenance.RunsHumanText}
	if label.Scope == ScopeMessage {
		population.Unit, population.Eligible = "message", provenance.HumanMessages
	}
	row.Population = population
	row.Eligible = population.Eligible
}

// coverageFromViews aggregates per-label rows for an evaluate response,
// including dry runs that never wrote to the store.
func (s *Service) coverageFromViews(views []RunView) []CoverageLabel {
	acc := map[string]*labelAccumulator{}
	for _, view := range views {
		for _, annotation := range view.Labels {
			s.accumulate(acc, s.annotationFromView(annotation, ScopeRun))
		}
		for _, message := range view.Messages {
			for _, annotation := range message.Labels {
				s.accumulate(acc, s.annotationFromView(annotation, ScopeMessage))
			}
		}
	}
	out := make([]CoverageLabel, 0, len(s.pack.Pack.Labels))
	for _, declaration := range s.pack.Pack.Labels {
		entry := acc[declaration.ID]
		if entry == nil {
			entry = &labelAccumulator{models: map[ModelKey]*modelAccumulator{}}
		}
		out = append(out, s.coverageLabel(declaration, entry))
	}
	return out
}

// coverageLabel renders one label's coverage. Every proof is per model: the
// rate is the friction rate over observations whose own (model, resource role)
// has a hold-out measurement clearing the gate, so an unproven model's answers
// never enter a proven rate and a proven model's never vouch for another.
func (s *Service) coverageLabel(declaration Label, acc *labelAccumulator) CoverageLabel {
	hash := s.pack.LabelHashes[declaration.ID]
	observed := acc.counts.Observed + acc.counts.Compatible
	row := CoverageLabel{
		ID: declaration.ID, Scope: declaration.Scope, Group: declaration.Group,
		Observed: observed, Eligible: acc.counts.Total(), Evaluated: acc.counts.Total(),
		SearchFacet: declaration.Search.Facet, MinProbability: declaration.Search.MinProbability,
		Friction: declaration.Friction, BaselineState: "not_measured",
	}
	compiled, ok := s.pack.Label(declaration.ID)
	if !ok {
		compiled = &compiledLabel{label: declaration, hash: hash}
	}
	keys := acc.keys()
	for _, key := range keys {
		group := acc.models[key]
		status := declaration.Gate.Status(hash, key.Model, key.ResourceRole)
		if compiled.derived() {
			status = s.pack.LabelStatusFor(compiled, nil)
		}
		row.Models = append(row.Models, ModelCoverage{
			Model: key.Model, ResourceRole: key.ResourceRole, Observed: group.observed,
			Positives: group.positives, Gate: status, Proven: group.observed > 0 && group.proven == group.observed,
		})
		row.Positives += group.positives
		row.ProvenObserved += group.proven
		row.ProvenPositives += group.provenPositives
	}
	row.Gate = s.pack.LabelStatusFor(compiled, keys)
	row.State = labelCoverageState(acc.counts, row.ProvenObserved, observed)
	// A rate is positives among the proven observed population. An observation
	// below its threshold is observed, not a negative of anything else.
	if declaration.Friction.Enabled && row.ProvenObserved > 0 {
		rate := float64(row.ProvenPositives) / float64(row.ProvenObserved)
		row.Rate = &rate
	}
	return row
}

func labelCoverageState(counts StateCounts, provenObserved, observed int) string {
	switch {
	case counts.Total() == 0:
		return "not_observed"
	case observed > 0 && provenObserved == observed:
		return "proven"
	case observed > 0 && provenObserved > 0:
		return "partial"
	case observed > 0:
		return "unproven"
	case counts.Unavailable > 0 && counts.NotObserved == 0:
		return "unavailable"
	default:
		return "not_observed"
	}
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

// GoldDraftRun is one subject to judge: a run, or for a message-scope label one
// message of it (ScopeID is its event id and the digest is that message).
type GoldDraftRun struct {
	RunID   string `json:"run_id"`
	ScopeID string `json:"scope_id,omitempty"`
	Digest  string `json:"digest"`
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
	scopes, err := s.store.ObservedScopes(ctx, compiled.label.Scope, since, until, labelID, compiled.hash, limit)
	if err != nil {
		return GoldDraft{}, err
	}
	draft := GoldDraft{
		Label: labelID, DefinitionHash: compiled.hash, Question: compiled.label.Question,
		Window: windowLabel, MinGold: compiled.label.Gate.MinGold, Limit: limit,
		Runs: make([]GoldDraftRun, 0, len(scopes)),
	}
	read := map[string]*RunFacts{}
	for _, scope := range scopes {
		facts, ok := read[scope.RunID]
		if !ok {
			loaded, err := s.features.Facts(ctx, ScopeRun, scope.RunID)
			if err == nil {
				if loaded.RunID == "" {
					loaded.RunID = scope.RunID
				}
				s.pack.Pack.Runtime.Provenance.Apply(&loaded)
				facts = &loaded
			}
			read[scope.RunID] = facts
		}
		if facts == nil {
			// A deleted or unreadable run cannot be labelled; skip it rather
			// than emit a packet without evidence.
			continue
		}
		// The reviewer sees a wider bounded state than the model. A label can
		// only be measured when a person can actually judge it from the packet.
		var digest string
		if compiled.label.Scope == ScopeMessage {
			digest = s.messageReviewDigest(*facts, scope.ScopeID)
		} else {
			digest, err = BuildReviewDigest(*facts, DefaultReviewDigestLimit)
			if err != nil {
				digest = ""
			}
		}
		if digest == "" {
			continue
		}
		run := GoldDraftRun{RunID: scope.RunID, Digest: digest}
		if compiled.label.Scope == ScopeMessage {
			run.ScopeID = scope.ScopeID
		}
		draft.Runs = append(draft.Runs, run)
	}
	return draft, nil
}
