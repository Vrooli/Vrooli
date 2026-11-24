//go:build audio_lab

package eval

import (
	"fmt"
	"math"
)

// TimedWord is one normalized-by-the-scorer reference or recognized word,
// with audio-relative boundaries in seconds. Reference words must come from
// a versioned human-checked alignment; hypotheses may come from a provider.
type TimedWord struct {
	Word  string  `json:"word"`
	Start float64 `json:"start_seconds"`
	End   float64 `json:"end_seconds"`
}

// TimestampDriftResult summarizes timing only for exact lexical matches in
// the deterministic minimum-edit alignment. It is descriptive evidence, not
// a model qualification verdict.
type TimestampDriftResult struct {
	ReferenceWords int     `json:"reference_words"`
	MatchedWords   int     `json:"matched_words"`
	Coverage       float64 `json:"coverage"`
	StartAbsP50Ms  float64 `json:"start_abs_p50_ms"`
	StartAbsP95Ms  float64 `json:"start_abs_p95_ms"`
	EndAbsP50Ms    float64 `json:"end_abs_p50_ms"`
	EndAbsP95Ms    float64 `json:"end_abs_p95_ms"`
	StartBiasMs    float64 `json:"start_bias_ms"`
	EndBiasMs      float64 `json:"end_bias_ms"`
	MaxStartAbsMs  float64 `json:"max_start_abs_ms"`
	MaxEndAbsMs    float64 `json:"max_end_abs_ms"`
	// Drift rates are the least-squares change in signed boundary error over
	// recording time. Unlike absolute-error percentiles, they expose gradual
	// clock/offset creep. They are observed only with at least two distinct
	// reference times and remain descriptive, not a qualification threshold.
	DriftRateObserved   bool    `json:"drift_rate_observed"`
	StartDriftMsPerHour float64 `json:"start_drift_ms_per_hour"`
	EndDriftMsPerHour   float64 `json:"end_drift_ms_per_hour"`
}

// The corpus and replay path admit at most this many timed words per clip.
// Keep the scorer's public boundary aligned with that contract, including the
// identical-stream fast path which otherwise bypasses the alignment-cell cap.
const maxTimestampWords = 20_000

// Timestamp drift is an offline evaluation path, but it must not allocate the
// EditCounts matrix used by the UI-oriented WER diff (24 bytes per cell). Two
// bits per cell retain deterministic traceback while rolling cost rows keep
// long-form scoring within a bounded memory budget.
// A full 20,000-word reference/hypothesis pair needs 400,040,001 cells.
// Two-bit traceback therefore needs about 100 MB, plus rolling cost rows.
// Keep the explicit ceiling above the admitted corpus size while bounding
// pathological larger requests to roughly 128 MiB of traceback storage.
const maxTimestampAlignmentCells = 512 * 1024 * 1024

// ScoreTimestampDrift aligns recognized words to timed reference words using
// this package's v1 lexical normalization and reports boundary error for
// exact word matches. Unmatched words remain visible through MatchedWords and
// Coverage; they are never treated as zero-drift matches.
func ScoreTimestampDrift(reference, hypothesis []TimedWord) (TimestampDriftResult, error) {
	result := TimestampDriftResult{ReferenceWords: len(reference)}
	if len(reference) == 0 {
		return result, fmt.Errorf("timestamp reference is empty")
	}
	if len(hypothesis) == 0 {
		return result, fmt.Errorf("timestamp hypothesis is empty")
	}
	if len(reference) > maxTimestampWords || len(hypothesis) > maxTimestampWords {
		return result, fmt.Errorf("timestamp comparison exceeds %d words per side", maxTimestampWords)
	}
	if err := validateTimedWords("reference", reference); err != nil {
		return result, err
	}
	if err := validateTimedWords("hypothesis", hypothesis); err != nil {
		return result, err
	}
	refTokens, hypTokens := make([]string, len(reference)), make([]string, len(hypothesis))
	for i := range reference {
		refTokens[i] = Normalize(reference[i].Word, DefaultNormalizeOptions())
	}
	for i := range hypothesis {
		hypTokens[i] = Normalize(hypothesis[i].Word, DefaultNormalizeOptions())
	}
	var starts, ends, referenceTimes []float64
	pairs, err := timestampMatches(refTokens, hypTokens)
	if err != nil {
		return result, err
	}
	for _, pair := range pairs {
		i, j := pair[0], pair[1]
		starts = append(starts, (hypothesis[j].Start-reference[i].Start)*1000)
		ends = append(ends, (hypothesis[j].End-reference[i].End)*1000)
		referenceTimes = append(referenceTimes, reference[i].Start)
	}
	result.MatchedWords = len(starts)
	result.Coverage = float64(result.MatchedWords) / float64(result.ReferenceWords)
	if len(starts) == 0 {
		// Coverage is measurable, but boundary error is not. Returning an
		// otherwise zero-valued result would make absent timing evidence look
		// like perfect timing in downstream reports.
		return result, fmt.Errorf("timestamp comparison has no exact lexical matches")
	}
	startAbs, endAbs := make([]float64, len(starts)), make([]float64, len(ends))
	for i := range starts {
		startAbs[i], endAbs[i] = math.Abs(starts[i]), math.Abs(ends[i])
		result.StartBiasMs += starts[i]
		result.EndBiasMs += ends[i]
		if endAbs[i] > result.MaxEndAbsMs {
			result.MaxEndAbsMs = endAbs[i]
		}
		if startAbs[i] > result.MaxStartAbsMs {
			result.MaxStartAbsMs = startAbs[i]
		}
	}
	denom := float64(len(starts))
	result.StartBiasMs /= denom
	result.EndBiasMs /= denom
	result.StartAbsP50Ms, result.StartAbsP95Ms = P50(startAbs), P95(startAbs)
	result.EndAbsP50Ms, result.EndAbsP95Ms = P50(endAbs), P95(endAbs)
	result.StartDriftMsPerHour, result.DriftRateObserved = driftRatePerHour(referenceTimes, starts)
	result.EndDriftMsPerHour, _ = driftRatePerHour(referenceTimes, ends)
	for _, value := range []float64{
		result.Coverage, result.StartAbsP50Ms, result.StartAbsP95Ms,
		result.EndAbsP50Ms, result.EndAbsP95Ms, result.StartBiasMs,
		result.EndBiasMs, result.MaxStartAbsMs, result.MaxEndAbsMs,
		result.StartDriftMsPerHour, result.EndDriftMsPerHour,
	} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return result, fmt.Errorf("timestamp comparison produced non-finite derived metrics")
		}
	}
	return result, nil
}

// driftRatePerHour fits signed boundary error against reference audio time.
// Centered sums reduce numerical error for long recordings whose timestamps
// are large relative to their drift. A zero rate is meaningful only when the
// observations span at least two distinct reference positions.
func driftRatePerHour(times, errorsMs []float64) (float64, bool) {
	if len(times) < 2 || len(times) != len(errorsMs) {
		return 0, false
	}
	var meanTime, meanError float64
	for i := range times {
		meanTime += times[i]
		meanError += errorsMs[i]
	}
	meanTime /= float64(len(times))
	meanError /= float64(len(errorsMs))
	var covariance, variance float64
	for i := range times {
		dt := times[i] - meanTime
		covariance += dt * (errorsMs[i] - meanError)
		variance += dt * dt
	}
	if variance == 0 {
		return 0, false
	}
	return covariance / variance * 3600, true
}

// timestampMatches returns exact lexical matches from a minimum-edit path.
// It uses the same substitution > deletion > insertion tie order as
// AlignOperations, with exact matches taking the diagonal at zero cost.
func timestampMatches(reference, hypothesis []string) ([][2]int, error) {
	n, m := len(reference), len(hypothesis)
	if n > maxTimestampWords || m > maxTimestampWords {
		return nil, fmt.Errorf("timestamp comparison exceeds %d words per side", maxTimestampWords)
	}
	// Trim a shared prefix only while every consumed token occurs exactly once
	// in each complete stream. Those unique zero-cost diagonals cannot be
	// displaced by repeated words, so this preserves the canonical path while
	// avoiding quadratic work on unchanged long prefixes. Repeated leading
	// tokens remain in the DP window because the canonical path may align them
	// later.
	prefix := uniqueSharedPrefix(reference, hypothesis)
	// Exact lexical matches at the end are unconditional zero-cost diagonals
	// in the canonical traceback path. Consume them before allocating traceback
	// so long transcripts whose differences are near the end cost O(changed
	// window²) instead of O(n*m).
	suffix := 0
	for suffix < n-prefix && suffix < m-prefix && reference[n-1-suffix] == hypothesis[m-1-suffix] {
		suffix++
	}
	if prefix > 0 || suffix > 0 {
		middle, err := timestampMatchesWindow(reference[prefix:n-suffix], hypothesis[prefix:m-suffix])
		if err != nil {
			return nil, err
		}
		matches := make([][2]int, 0, prefix+len(middle)+suffix)
		for i := 0; i < prefix; i++ {
			matches = append(matches, [2]int{i, i})
		}
		for _, pair := range middle {
			matches = append(matches, [2]int{pair[0] + prefix, pair[1] + prefix})
		}
		for i := 0; i < suffix; i++ {
			matches = append(matches, [2]int{n - suffix + i, m - suffix + i})
		}
		return matches, nil
	}
	return timestampMatchesWindow(reference, hypothesis)
}

// uniqueSharedPrefix returns the longest common leading sequence whose tokens
// each appear exactly once in both complete inputs. A token frequency of two
// across the concatenated streams proves one occurrence per side. This O(n+m)
// guard lets the quadratic traceback skip long uniquely anchored prefixes
// without changing repeated-word tie resolution.
func uniqueSharedPrefix(reference, hypothesis []string) int {
	limit := len(reference)
	if len(hypothesis) < limit {
		limit = len(hypothesis)
	}
	if limit == 0 || reference[0] != hypothesis[0] {
		return 0
	}
	frequencies := make(map[string]uint32, len(reference)+len(hypothesis))
	for _, word := range reference {
		frequencies[word]++
	}
	for _, word := range hypothesis {
		frequencies[word]++
	}
	prefix := 0
	for prefix < limit && reference[prefix] == hypothesis[prefix] && frequencies[reference[prefix]] == 2 {
		prefix++
	}
	return prefix
}

func timestampMatchesWindow(reference, hypothesis []string) ([][2]int, error) {
	n, m := len(reference), len(hypothesis)
	width := m + 1
	if n+1 > int(^uint(0)>>1)/width {
		return nil, fmt.Errorf("timestamp alignment dimensions overflow")
	}
	cells := (n + 1) * width
	if cells > maxTimestampAlignmentCells {
		return nil, fmt.Errorf("timestamp alignment requires %d cells; limit is %d", cells, maxTimestampAlignmentCells)
	}
	directions := make([]byte, (cells+3)/4)
	setDirection := func(cell, direction int) {
		shift := uint((cell & 3) * 2)
		mask := byte(3 << shift)
		directions[cell>>2] = (directions[cell>>2] &^ mask) | byte(direction<<shift)
	}
	getDirection := func(cell int) int {
		return int((directions[cell>>2] >> uint((cell&3)*2)) & 3)
	}
	const (
		diagonalMatch = iota
		substitution
		deletion
		insertion
	)
	for j := 1; j <= m; j++ {
		setDirection(j, insertion)
	}
	previous, current := make([]uint32, width), make([]uint32, width)
	for j := 1; j <= m; j++ {
		previous[j] = uint32(j)
	}
	for i := 1; i <= n; i++ {
		current[0] = uint32(i)
		setDirection(i*width, deletion)
		for j := 1; j <= m; j++ {
			cell := i*width + j
			if reference[i-1] == hypothesis[j-1] {
				current[j] = previous[j-1]
				setDirection(cell, diagonalMatch)
				continue
			}
			best, direction := previous[j-1]+1, substitution
			if cost := previous[j] + 1; cost < best {
				best, direction = cost, deletion
			}
			if cost := current[j-1] + 1; cost < best {
				best, direction = cost, insertion
			}
			current[j] = best
			setDirection(cell, direction)
		}
		previous, current = current, previous
	}
	reversed := make([][2]int, 0)
	i, j := n, m
	for i > 0 || j > 0 {
		direction := getDirection(i*width + j)
		switch direction {
		case diagonalMatch:
			reversed = append(reversed, [2]int{i - 1, j - 1})
			i, j = i-1, j-1
		case substitution:
			i, j = i-1, j-1
		case deletion:
			i--
		case insertion:
			j--
		default:
			return nil, fmt.Errorf("timestamp alignment traceback failed at (%d,%d)", i, j)
		}
	}
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	return reversed, nil
}

func validateTimedWords(label string, words []TimedWord) error {
	var previousStart, previousEnd float64
	for i, word := range words {
		if Normalize(word.Word, DefaultNormalizeOptions()) == "" {
			return fmt.Errorf("%s word %d has empty normalized text", label, i)
		}
		if math.IsNaN(word.Start) || math.IsInf(word.Start, 0) || math.IsNaN(word.End) || math.IsInf(word.End, 0) || word.Start < 0 || word.End < word.Start {
			return fmt.Errorf("%s word %d has invalid interval [%v,%v]", label, i, word.Start, word.End)
		}
		if i > 0 && (word.Start < previousStart || word.End < previousEnd) {
			return fmt.Errorf("%s word %d is not monotonically ordered", label, i)
		}
		previousStart, previousEnd = word.Start, word.End
	}
	return nil
}
