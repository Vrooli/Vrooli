package runannotations

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Benchmark measures a label against gold rows using only stored probability
// vectors: it never calls a model. It is on the server, beside the store, so the
// vectors are read in one query instead of one request per run, and the same
// arithmetic serves the gate measurement and the calibration report.

// Benchmark sources: which table the scored observations come from.
const (
	SourceProduction = "production"
	SourcePinned     = "pinned"
)

// GoldRow is one labelled subject: a run, or for a message-scope label one
// message of a run (ScopeID is its event id). Positive is the desired
// behaviour: the friction event for a friction label, the statement being true
// for a noul label.
type GoldRow struct {
	RunID    string `json:"run_id"`
	ScopeID  string `json:"scope_id,omitempty"`
	Positive bool   `json:"positive"`
}

// subject is the id the label's observations are stored under: the message id
// for a message-scope label, the run id otherwise.
func (r GoldRow) subject(scope Scope) string {
	if scope == ScopeMessage {
		return strings.TrimSpace(r.ScopeID)
	}
	return strings.TrimSpace(r.RunID)
}

// BenchmarkRequest asks for one measurement.
type BenchmarkRequest struct {
	Label string `json:"label"`
	// Split is "tuning" (fit and evaluate on Rows) or "holdout" (evaluate Rows
	// with the calibration fitted on FitRows, without refitting). Empty is tuning.
	Split string    `json:"split"`
	Rows  []GoldRow `json:"rows"`
	// FitRows are the tuning rows a holdout measurement's temperature and
	// threshold are fitted on. They must share no run with Rows.
	FitRows  []GoldRow `json:"fit_rows,omitempty"`
	GoldFile string    `json:"gold_file,omitempty"`
	// Threshold overrides the threshold chosen from the calibrated curve.
	Threshold *float64 `json:"threshold,omitempty"`
	// Model and ResourceRole restrict the scored observations to one candidate.
	// Both empty reads whichever model answered last, which is refused when the
	// rows span more than one.
	Model        string `json:"model,omitempty"`
	ResourceRole string `json:"resource_role,omitempty"`
	// Source is "production" (default) or "pinned" (a bake-off's trial rows).
	Source string `json:"source,omitempty"`
}

// BenchmarkGroup is the observations scored under one (model, resource role).
type BenchmarkGroup struct {
	Model        string `json:"model"`
	ResourceRole string `json:"resource_role"`
	Rows         int    `json:"rows"`
}

// BenchmarkResult is one measurement and the evidence behind it.
type BenchmarkResult struct {
	Label          string `json:"label"`
	DefinitionHash string `json:"definition_hash"`
	Split          string `json:"split"`
	Source         string `json:"source"`
	Model          string `json:"model,omitempty"`
	ResourceRole   string `json:"resource_role,omitempty"`
	// Labelled is every gold row that named the label; Gold is the rows actually
	// scored. Only Gold may satisfy gate.min_gold.
	Labelled  int `json:"labelled"`
	Gold      int `json:"gold"`
	Positives int `json:"positives"`
	// Unobserved are labelled subjects with no stored answer under the current
	// definition for this candidate (they are accepted, not dropped: annotate
	// them, or re-annotate them pinned, and measure again). Unavailable are
	// subjects whose stored answer carries nothing to score.
	Unobserved         int      `json:"unobserved"`
	UnobservedSubjects []string `json:"unobserved_subjects,omitempty"`
	Unavailable        int      `json:"unavailable"`
	FitRows            int      `json:"fit_rows,omitempty"`
	TP                 int      `json:"true_positive"`
	FP                 int      `json:"false_positive"`
	FN                 int      `json:"false_negative"`
	TN                 int      `json:"true_negative"`

	Precision   float64 `json:"precision"`
	Recall      float64 `json:"recall"`
	Threshold   float64 `json:"threshold"`
	Temperature float64 `json:"temperature"`
	// Fitted is false when the data could not constrain a temperature (one
	// outcome class, or only fixed answers), so it stayed 1.
	Fitted bool `json:"fitted"`
	// ECE and Brier are the calibrated values on the scored rows, and the Raw
	// pair is the stored vectors read at temperature 1.
	ECE      float64 `json:"ece"`
	Brier    float64 `json:"brier"`
	RawECE   float64 `json:"raw_ece"`
	RawBrier float64 `json:"raw_brier"`
	// Reliability is the calibrated table and RawReliability the stored one.
	Reliability    []ReliabilityBin `json:"reliability"`
	RawReliability []ReliabilityBin `json:"raw_reliability"`
	// StoredTemperature is the temperature the pack currently applies to this
	// model's observations; the fitted one only takes effect once its
	// measurement is recorded in the pack.
	StoredTemperature float64 `json:"stored_temperature"`

	// Measurement is the entry to record under the label's gate.measurements.
	// It is absent when nothing was scored or the request was refused.
	Measurement *Measurement `json:"measurement,omitempty"`
	// Groups lists every (model, resource role) among the stored observations.
	Groups []BenchmarkGroup `json:"groups,omitempty"`
	// Refusal explains why no measurement was made.
	Refusal string `json:"refusal,omitempty"`
	Note    string `json:"note,omitempty"`
}

// ErrInvalidRequest marks a benchmark or series request that cannot be answered
// as asked; the surface reports it as a bad request.
var ErrInvalidRequest = errors.New("invalid benchmark request")

// Benchmark scores gold rows against the label's stored vectors.
func (s *Service) Benchmark(ctx context.Context, request BenchmarkRequest) (BenchmarkResult, error) {
	compiled, ok := s.pack.Label(request.Label)
	if !ok {
		return BenchmarkResult{}, fmt.Errorf("%w: unknown label %q", ErrInvalidRequest, request.Label)
	}
	label := compiled.label
	split := strings.TrimSpace(request.Split)
	if split == "" {
		split = SplitTuning
	}
	if split != SplitTuning && split != SplitHoldout {
		return BenchmarkResult{}, fmt.Errorf("%w: split %q must be tuning or holdout", ErrInvalidRequest, split)
	}
	if split == SplitTuning && len(request.FitRows) > 0 {
		return BenchmarkResult{}, fmt.Errorf("%w: fit rows belong to a hold-out measurement; a tuning measurement fits on its own rows", ErrInvalidRequest)
	}
	source := strings.TrimSpace(request.Source)
	if source == "" {
		source = SourceProduction
	}
	if source != SourceProduction && source != SourcePinned {
		return BenchmarkResult{}, fmt.Errorf("%w: source %q must be production or pinned", ErrInvalidRequest, source)
	}
	if compiled.derived() {
		return BenchmarkResult{}, fmt.Errorf("%w: %s is derived from other labels and has no gold of its own; measure its inputs", ErrInvalidRequest, label.ID)
	}
	for _, row := range append(append([]GoldRow(nil), request.Rows...), request.FitRows...) {
		if row.subject(label.Scope) == "" {
			return BenchmarkResult{}, fmt.Errorf("%w: a gold row for %s label %s names no %s", ErrInvalidRequest, label.Scope, label.ID, subjectField(label.Scope))
		}
	}
	if split == SplitHoldout {
		if err := disjoint(label.Scope, request.Rows, request.FitRows); err != nil {
			return BenchmarkResult{}, err
		}
	}
	store := s.store
	if source == SourcePinned {
		store = s.store.Trials()
	}
	result := BenchmarkResult{
		Label: label.ID, DefinitionHash: compiled.hash, Split: split, Source: source,
		Labelled: len(dedupeGold(label.Scope, request.Rows)), Temperature: 1, StoredTemperature: 1,
	}
	rows, fitRows := dedupeGold(label.Scope, request.Rows), dedupeGold(label.Scope, request.FitRows)
	ids := make([]string, 0, len(rows)+len(fitRows))
	for _, row := range rows {
		ids = append(ids, row.subject(label.Scope))
	}
	for _, row := range fitRows {
		ids = append(ids, row.subject(label.Scope))
	}
	observations, err := store.LatestObservations(ctx, label.Scope, ids, label.ID, compiled.hash, request.Model, request.ResourceRole)
	if err != nil {
		return BenchmarkResult{}, err
	}
	// One measurement proves one (definition, model, resource role). Rows from
	// two candidates would blend two accuracies into a number that describes
	// neither, so they are refused with the groups named.
	groups := map[ModelKey]int{}
	for _, observation := range observations {
		if _, usable := sampleFor(label, observation); !usable {
			continue
		}
		model, role := ProofKey(observation)
		groups[ModelKey{Model: model, ResourceRole: role}]++
	}
	for key, count := range groups {
		result.Groups = append(result.Groups, BenchmarkGroup{Model: key.Model, ResourceRole: key.ResourceRole, Rows: count})
	}
	sort.Slice(result.Groups, func(i, j int) bool {
		if result.Groups[i].Model != result.Groups[j].Model {
			return result.Groups[i].Model < result.Groups[j].Model
		}
		return result.Groups[i].ResourceRole < result.Groups[j].ResourceRole
	})
	if len(groups) > 1 {
		result.Refusal = "the gold rows' stored answers span more than one (model, resource role); one measurement proves one. " +
			"Pass --model and --resource-role to score a single candidate"
		return result, nil
	}
	var key ModelKey
	for candidate := range groups {
		key = candidate
	}
	result.Model, result.ResourceRole = key.Model, key.ResourceRole

	evalSamples, missing := samplesFor(label, rows, observations)
	result.Unavailable, result.Unobserved = missing.unusable, len(missing.unobserved)
	result.UnobservedSubjects = boundList(missing.unobserved, maxUnobservedListed)
	result.Gold = len(evalSamples)
	for _, sample := range evalSamples {
		if sample.Positive {
			result.Positives++
		}
	}
	if result.Gold == 0 {
		result.Note = "no gold row has a usable stored answer for the current definition" + unobservedAdvice(result.Unobserved)
		return result, nil
	}
	result.StoredTemperature, _ = label.Gate.Calibration(compiled.hash, key.Model, key.ResourceRole)
	fitSamples, _ := samplesFor(label, fitRows, observations)
	result.FitRows = len(fitSamples)

	fixed := true
	for _, sample := range evalSamples {
		fixed = fixed && sample.Fixed
	}
	temperature, threshold := 1.0, 0.0
	switch {
	case fixed:
		// A rule has no distribution to calibrate: its threshold is the declared
		// one and its temperature stays 1.
		threshold = declaredOr(label, 0.5)
	case split == SplitTuning:
		temperature, result.Fitted = FitTemperature(evalSamples)
		threshold = s.thresholdFor(label, evalSamples, temperature)
	default:
		if len(fitSamples) == 0 {
			result.Refusal = "a hold-out measurement evaluates a calibration fitted on other rows; pass the tuning rows with --fit-from"
			return result, nil
		}
		temperature, result.Fitted = FitTemperature(fitSamples)
		threshold = s.thresholdFor(label, fitSamples, temperature)
	}
	if request.Threshold != nil {
		if *request.Threshold <= 0 || *request.Threshold > 1 {
			return BenchmarkResult{}, fmt.Errorf("%w: threshold must be in (0,1]", ErrInvalidRequest)
		}
		threshold = *request.Threshold
	}
	confusion := Evaluate(evalSamples, temperature, threshold)
	result.Temperature, result.Threshold = temperature, threshold
	result.TP, result.FP, result.FN, result.TN = confusion.TP, confusion.FP, confusion.FN, confusion.TN
	result.Precision, result.Recall = confusion.Precision(), confusion.Recall()
	result.ECE, result.Brier = ECE(evalSamples, temperature), Brier(evalSamples, temperature)
	result.RawECE, result.RawBrier = ECE(evalSamples, 1), Brier(evalSamples, 1)
	result.Reliability, result.RawReliability = Reliability(evalSamples, temperature), Reliability(evalSamples, 1)
	if result.Unavailable > 0 || result.Unobserved > 0 {
		result.Note = fmt.Sprintf("%d of %d labelled row(s) are excluded from the gold count: %d with no stored answer under the current definition, %d whose stored answer carries nothing to score",
			result.Unavailable+result.Unobserved, result.Labelled, result.Unobserved, result.Unavailable) + unobservedAdvice(result.Unobserved)
	}
	result.Measurement = &Measurement{
		DefinitionHash: compiled.hash, Model: key.Model, ResourceRole: key.ResourceRole,
		Precision: round4(result.Precision), Recall: round4(result.Recall), Gold: result.Gold, Positives: result.Positives,
		Threshold: round6(threshold), Temperature: round6(temperature), ECE: round4(result.ECE), Brier: round4(result.Brier),
		Split: split, GoldFile: request.GoldFile, MeasuredAt: s.now().UTC().Format(time.RFC3339),
	}
	return result, nil
}

// thresholdFor chooses the operating point from the calibrated curve of the
// fitting rows, falling back to the label's declared threshold when the rows
// cannot separate two classes.
func (s *Service) thresholdFor(label Label, samples []Sample, temperature float64) float64 {
	if chosen, ok := ChooseThreshold(samples, temperature, label.Gate.Precision, label.Gate.Recall); ok {
		return chosen
	}
	return declaredOr(label, 0.5)
}

func declaredOr(label Label, fallback float64) float64 {
	if threshold := probabilityThreshold(label); threshold != nil && *threshold > 0 {
		return *threshold
	}
	return fallback
}

// maxUnobservedListed bounds the subjects a result names.
const maxUnobservedListed = 50

func subjectField(scope Scope) string {
	if scope == ScopeMessage {
		return "scope_id (the message event id)"
	}
	return "run_id"
}

func unobservedAdvice(count int) string {
	if count == 0 {
		return ""
	}
	return "; annotate them (annotations evaluate --runs-from <gold file>, with --pin-resource-role for a bake-off candidate) and measure again"
}

func boundList(items []string, limit int) []string {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

// missingGold are the gold rows a measurement could not score: subjects with no
// stored answer, and subjects whose answer carries nothing to score.
type missingGold struct {
	unobserved []string
	unusable   int
}

// samplesFor pairs gold rows with their stored observations. A row with no
// usable observation is reported, never counted as a negative.
func samplesFor(label Label, rows []GoldRow, observations map[string]Annotation) ([]Sample, missingGold) {
	var (
		samples []Sample
		missing missingGold
	)
	for _, row := range rows {
		subject := row.subject(label.Scope)
		observation, found := observations[subject]
		if !found {
			missing.unobserved = append(missing.unobserved, subject)
			continue
		}
		sample, usable := sampleFor(label, observation)
		if !usable {
			missing.unusable++
			continue
		}
		sample.Positive = row.Positive
		samples = append(samples, sample)
	}
	return samples, missing
}

// disjoint refuses a hold-out that shares a run with the tuning rows. Message
// labels have multiple subjects per run, but their messages still share the
// same conversation context and cannot be split across calibration and proof.
func disjoint(scope Scope, rows, fit []GoldRow) error {
	group := func(row GoldRow) string {
		if scope == ScopeMessage {
			return strings.TrimSpace(row.RunID)
		}
		return row.subject(scope)
	}
	tuned := make(map[string]struct{}, len(fit))
	for _, row := range fit {
		runID := group(row)
		if scope == ScopeMessage && runID == "" {
			return fmt.Errorf("%w: message-scoped hold-out rows must name their run_id", ErrInvalidRequest)
		}
		tuned[runID] = struct{}{}
	}
	sharedSet := make(map[string]struct{})
	for _, row := range rows {
		runID := group(row)
		if scope == ScopeMessage && runID == "" {
			return fmt.Errorf("%w: message-scoped hold-out rows must name their run_id", ErrInvalidRequest)
		}
		if _, ok := tuned[runID]; ok {
			sharedSet[runID] = struct{}{}
		}
	}
	shared := make([]string, 0, len(sharedSet))
	for runID := range sharedSet {
		shared = append(shared, runID)
	}
	if len(shared) > 0 {
		sort.Strings(shared)
		return fmt.Errorf("%w: %d run(s) appear in both the tuning rows and the evaluated rows (first: %s); a hold-out must be disjoint", ErrInvalidRequest, len(shared), shared[0])
	}
	return nil
}

// dedupeGold keeps the first row for each subject: a run (or message) is one
// gold subject.
func dedupeGold(scope Scope, rows []GoldRow) []GoldRow {
	seen := make(map[string]bool, len(rows))
	out := make([]GoldRow, 0, len(rows))
	for _, row := range rows {
		subject := row.subject(scope)
		if subject == "" || seen[subject] {
			continue
		}
		seen[subject] = true
		out = append(out, row)
	}
	return out
}

func round4(value float64) float64 { return float64(int64(value*1e4+0.5)) / 1e4 }

// round6 keeps a threshold or temperature precise enough that rounding cannot
// move a score across the threshold it was chosen between.
func round6(value float64) float64 { return float64(int64(value*1e6+0.5)) / 1e6 }
