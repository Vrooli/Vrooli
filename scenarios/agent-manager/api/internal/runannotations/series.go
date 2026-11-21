package runannotations

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// SeriesPoint is one time bucket of a message-scope label.
type SeriesPoint struct {
	Start    time.Time `json:"start"`
	Messages int       `json:"messages"`
	// ProvenMessages are the messages whose own model's hold-out proof clears
	// the gate; ProvenMean averages only those.
	ProvenMessages int `json:"proven_messages"`
	// Mean is the mean calibrated value over every message in the bucket: the
	// expected score on a score label, the probability on a noul label. It is
	// evidence of mood, not a measurement, until ProvenMessages == Messages.
	Mean              *float64 `json:"mean,omitempty"`
	ProvenMean        *float64 `json:"proven_mean,omitempty"`
	FrictionPositives int      `json:"friction_positives,omitempty"`
}

// SeriesResult is a label's per-message series over human operator text.
type SeriesResult struct {
	Label          string        `json:"label"`
	DefinitionHash string        `json:"definition_hash"`
	Bucket         string        `json:"bucket"`
	Window         WindowRange   `json:"window"`
	Messages       int           `json:"messages"`
	ProvenMessages int           `json:"proven_messages"`
	Points         []SeriesPoint `json:"points"`
	Note           string        `json:"note,omitempty"`
}

// Series reads a message-scope label's observations into time buckets by when
// each message was written. Only human operator messages are subjects, so the
// series is never a measurement of workflow briefs.
func (s *Service) Series(ctx context.Context, labelID, window, bucket string) (SeriesResult, error) {
	compiled, ok := s.pack.Label(labelID)
	if !ok {
		return SeriesResult{}, fmt.Errorf("%w: unknown label %q", ErrInvalidRequest, labelID)
	}
	if compiled.label.Scope != ScopeMessage {
		return SeriesResult{}, fmt.Errorf("%w: label %s has scope %s; a series needs a message-scope label", ErrInvalidRequest, labelID, compiled.label.Scope)
	}
	step := 24 * time.Hour
	switch strings.ToLower(strings.TrimSpace(bucket)) {
	case "", "day":
		bucket = "day"
	case "hour":
		step = time.Hour
	case "week":
		step = 7 * 24 * time.Hour
	default:
		return SeriesResult{}, fmt.Errorf("%w: bucket %q must be hour, day or week", ErrInvalidRequest, bucket)
	}
	since, until, _, err := ParseWindowRange(window, s.now())
	if err != nil {
		return SeriesResult{}, err
	}
	rows, err := s.store.MessageObservations(ctx, labelID, compiled.hash, since, until)
	if err != nil {
		return SeriesResult{}, err
	}
	result := SeriesResult{Label: labelID, DefinitionHash: compiled.hash, Bucket: bucket, Window: windowRange(since, until), Points: []SeriesPoint{}}
	type accumulator struct {
		point                  SeriesPoint
		sum, provenSum         float64
		counted, provenCounted int
	}
	byStart := map[time.Time]*accumulator{}
	var order []time.Time
	for _, row := range rows {
		if row.OccurredAt.IsZero() {
			continue
		}
		start := row.OccurredAt.UTC().Truncate(step)
		entry := byStart[start]
		if entry == nil {
			entry = &accumulator{point: SeriesPoint{Start: start}}
			byStart[start] = entry
			order = append(order, start)
		}
		entry.point.Messages++
		result.Messages++
		model, role := ProofKey(row)
		proven := s.pack.StatusFor(compiled, row).State == GateProven
		if proven {
			entry.point.ProvenMessages++
			result.ProvenMessages++
		}
		temperature, measured := compiled.label.Gate.Calibration(row.DefinitionHash, model, role)
		if isFrictionPositive(compiled.label, row, temperature, measured) {
			entry.point.FrictionPositives++
		}
		if value, ok := calibratedValue(compiled.label, row, temperature); ok {
			entry.sum += value
			entry.counted++
			if proven {
				entry.provenSum += value
				entry.provenCounted++
			}
		}
	}
	for _, start := range order {
		entry := byStart[start]
		if entry.counted > 0 {
			mean := entry.sum / float64(entry.counted)
			entry.point.Mean = &mean
		}
		if entry.provenCounted > 0 {
			mean := entry.provenSum / float64(entry.provenCounted)
			entry.point.ProvenMean = &mean
		}
		result.Points = append(result.Points, entry.point)
	}
	switch {
	case result.Messages == 0:
		result.Note = "no message of human operator text has been observed in this window: the series is unknown, not neutral"
	case result.ProvenMessages < result.Messages:
		result.Note = fmt.Sprintf("%d of %d message(s) come from a model with no clearing hold-out proof for this definition; the mean is evidence of mood, not a measurement", result.Messages-result.ProvenMessages, result.Messages)
	}
	return result, nil
}

// calibratedValue is an observation's value on the label's own scale at the
// model's temperature: the expected score of a score label, the probability of a
// noul label. A choice has no scalar value.
func calibratedValue(label Label, row Annotation, temperature float64) (float64, bool) {
	// A rule, a signal or a composite states its value outright: there is no
	// distribution to temper, and its scale is whatever its rule computes (a
	// valence composite runs from -1 to 1).
	if isRuleMethod(row.Method) {
		return numericObservation(row)
	}
	sample, ok := sampleFor(label, row)
	if !ok || label.Question == nil {
		return 0, false
	}
	switch label.Question.Type {
	case QuestionScore:
		if sample.Fixed || len(sample.Probs) != len(label.Question.Levels) {
			if row.Value.Score != nil {
				return *row.Value.Score, true
			}
			return 0, false
		}
		expected := 0.0
		for index, probability := range sample.Tempered(temperature) {
			expected += (float64(index) + label.Question.Offset) * probability
		}
		return expected, !math.IsNaN(expected)
	case QuestionNoul:
		return sample.Score(temperature), true
	}
	return 0, false
}
