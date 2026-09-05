package eval

import (
	"sort"
	"strings"
	"time"
	"unicode"
)

// EditCounts is the substitution/insertion/deletion breakdown of one
// alignment. Deletions are reference words the hypothesis omitted —
// surfacing the "dropped sections" failure mode the streaming strategies
// suffer; insertions are spurious hypothesis words (hallucinations,
// prompt regurgitation); substitutions are mis-recognized words.
type EditCounts struct {
	Substitutions int
	Insertions    int
	Deletions     int
}

// EditOperation is one step in the aligned reference/hypothesis token path.
// It is intended for UI diffs and debugging, not for recomputing metrics.
type EditOperation struct {
	Kind            string
	ReferenceToken  string
	HypothesisToken string
	ReferenceIndex  int
	HypothesisIndex int
}

// Total is the raw edit distance S+I+D — the WER numerator.
func (e EditCounts) Total() int { return e.Substitutions + e.Insertions + e.Deletions }

// WERResult is a word-error-rate computation plus the breakdown and
// sequence lengths needed to interpret it.
type WERResult struct {
	EditCounts
	RefWords int
	HypWords int
}

// PresentationMetrics records the two display properties that can otherwise
// disappear when an engine comparison reports only normalized WER. Rates are
// calculated over whitespace-delimited output words: punctuation rate is the
// share of words containing at least one Unicode punctuation mark, and
// capitalisation rate is the share whose first letter is uppercase.
type PresentationMetrics struct {
	WordCount          int
	PunctuationRate    float64
	CapitalisationRate float64
}

func MeasurePresentation(text string) PresentationMetrics {
	words := strings.Fields(text)
	metrics := PresentationMetrics{WordCount: len(words)}
	if len(words) == 0 {
		return metrics
	}
	for _, word := range words {
		punctuated := false
		capitalised := false
		seenLetter := false
		for _, r := range word {
			if unicode.IsPunct(r) {
				punctuated = true
			}
			if !seenLetter && unicode.IsLetter(r) {
				seenLetter = true
				capitalised = unicode.IsUpper(r)
			}
		}
		if punctuated {
			metrics.PunctuationRate++
		}
		if capitalised {
			metrics.CapitalisationRate++
		}
	}
	denom := float64(len(words))
	metrics.PunctuationRate /= denom
	metrics.CapitalisationRate /= denom
	return metrics
}

// Rate is the word error rate (S+I+D)/N_ref. By convention an empty
// reference yields 0 when the hypothesis is also empty (perfect), and 1
// when the hypothesis has content (everything is an insertion error —
// reported as a rate of 1.0 so an empty-reference clip can't silently dilute
// an aggregate below its true error).
func (w WERResult) Rate() float64 {
	if w.RefWords == 0 {
		if w.HypWords == 0 {
			return 0
		}
		return 1
	}
	return float64(w.Total()) / float64(w.RefWords)
}

// Align computes the minimum-edit alignment between two already-tokenized
// sequences via Wagner-Fischer dynamic programming, returning the S/I/D
// breakdown of a minimum-cost path. Ties (equal total cost) are broken
// deterministically in substitution > deletion > insertion order, so the
// breakdown is reproducible run-to-run.
func Align(ref, hyp []string) EditCounts {
	return alignPath(ref, hyp, false).counts
}

// AlignOperations returns the same deterministic minimum-edit alignment as
// Align, but includes the operation path needed by report drilldowns.
func AlignOperations(ref, hyp []string) (EditCounts, []EditOperation) {
	path := alignPath(ref, hyp, true)
	return path.counts, path.ops
}

type alignmentPath struct {
	counts EditCounts
	ops    []EditOperation
}

func alignPath(ref, hyp []string, keepOps bool) alignmentPath {
	n, m := len(ref), len(hyp)
	// d[i][j] = best edit counts aligning ref[:i] to hyp[:j].
	d := make([][]EditCounts, n+1)
	for i := range d {
		d[i] = make([]EditCounts, m+1)
	}
	for i := 1; i <= n; i++ {
		d[i][0] = EditCounts{Deletions: i}
	}
	for j := 1; j <= m; j++ {
		d[0][j] = EditCounts{Insertions: j}
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if ref[i-1] == hyp[j-1] {
				d[i][j] = d[i-1][j-1]
				continue
			}
			sub := d[i-1][j-1]
			sub.Substitutions++
			del := d[i-1][j]
			del.Deletions++
			ins := d[i][j-1]
			ins.Insertions++
			d[i][j] = minCounts(sub, del, ins)
		}
	}
	out := alignmentPath{counts: d[n][m]}
	if !keepOps {
		return out
	}
	i, j := n, m
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && ref[i-1] == hyp[j-1] && d[i][j] == d[i-1][j-1]:
			out.ops = append(out.ops, EditOperation{
				Kind:            "match",
				ReferenceToken:  ref[i-1],
				HypothesisToken: hyp[j-1],
				ReferenceIndex:  i - 1,
				HypothesisIndex: j - 1,
			})
			i--
			j--
		case i > 0 && j > 0 && editCountsEqual(d[i][j], incSub(d[i-1][j-1])):
			out.ops = append(out.ops, EditOperation{
				Kind:            "substitution",
				ReferenceToken:  ref[i-1],
				HypothesisToken: hyp[j-1],
				ReferenceIndex:  i - 1,
				HypothesisIndex: j - 1,
			})
			i--
			j--
		case i > 0 && editCountsEqual(d[i][j], incDel(d[i-1][j])):
			out.ops = append(out.ops, EditOperation{
				Kind:            "deletion",
				ReferenceToken:  ref[i-1],
				ReferenceIndex:  i - 1,
				HypothesisIndex: -1,
			})
			i--
		default:
			out.ops = append(out.ops, EditOperation{
				Kind:            "insertion",
				HypothesisToken: hyp[j-1],
				ReferenceIndex:  -1,
				HypothesisIndex: j - 1,
			})
			j--
		}
	}
	for left, right := 0, len(out.ops)-1; left < right; left, right = left+1, right-1 {
		out.ops[left], out.ops[right] = out.ops[right], out.ops[left]
	}
	return out
}

// minCounts returns the lowest-Total EditCounts, breaking ties in
// sub > del > ins priority order (argument order encodes the priority).
func minCounts(a, b, c EditCounts) EditCounts {
	best := a
	if b.Total() < best.Total() {
		best = b
	}
	if c.Total() < best.Total() {
		best = c
	}
	return best
}

func incSub(e EditCounts) EditCounts {
	e.Substitutions++
	return e
}

func incDel(e EditCounts) EditCounts {
	e.Deletions++
	return e
}

func editCountsEqual(a, b EditCounts) bool {
	return a.Substitutions == b.Substitutions && a.Insertions == b.Insertions && a.Deletions == b.Deletions
}

// WER aligns ref against hyp (both pre-tokenized via Tokenize) and returns
// the rate plus breakdown and lengths.
func WER(ref, hyp []string) WERResult {
	return WERResult{
		EditCounts: Align(ref, hyp),
		RefWords:   len(ref),
		HypWords:   len(hyp),
	}
}

// CER is the character-error-rate analogue: it aligns the reference and
// hypothesis as rune sequences. Useful as a secondary metric when word
// boundaries are themselves contested (the punctuation/spacing the v1
// word normalizer folds away). The caller passes already-normalized
// strings.
func CER(ref, hyp string) WERResult {
	rr := []rune(ref)
	hr := []rune(hyp)
	refTok := make([]string, len(rr))
	for i, r := range rr {
		refTok[i] = string(r)
	}
	hypTok := make([]string, len(hr))
	for i, r := range hr {
		hypTok[i] = string(r)
	}
	return WERResult{
		EditCounts: Align(refTok, hypTok),
		RefWords:   len(refTok),
		HypWords:   len(hypTok),
	}
}

// RTF (real-time factor) is processing wall-time divided by audio
// duration: <1 means faster-than-real-time, >1 slower. Returns 0 when
// audio duration is non-positive (undefined).
func RTF(processing, audio time.Duration) float64 {
	if audio <= 0 {
		return 0
	}
	return float64(processing) / float64(audio)
}

// Percentile returns the p-th percentile (p in [0,100]) of samples using
// linear interpolation between closest ranks (the R-7 / Excel
// PERCENTILE.INC method). It sorts a copy, so the caller's slice is
// untouched. Empty input returns 0.
func Percentile(samples []float64, p float64) float64 {
	n := len(samples)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return samples[0]
	}
	if p <= 0 {
		return minFloat(samples)
	}
	if p >= 100 {
		return maxFloat(samples)
	}
	sorted := make([]float64, n)
	copy(sorted, samples)
	sort.Float64s(sorted)
	rank := p / 100 * float64(n-1)
	lo := int(rank)
	frac := rank - float64(lo)
	if lo+1 >= n {
		return sorted[n-1]
	}
	return sorted[lo] + frac*(sorted[lo+1]-sorted[lo])
}

// P50 / P95 are the latency percentiles the report surfaces.
func P50(samples []float64) float64 { return Percentile(samples, 50) }
func P95(samples []float64) float64 { return Percentile(samples, 95) }

// Mean returns the arithmetic mean of samples (0 for empty).
func Mean(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, v := range samples {
		sum += v
	}
	return sum / float64(len(samples))
}

// PartialRevisions counts how many times the live partial text CHANGED
// across a stream — a stability/jitter metric. A lower count means the
// user saw fewer rewrites of the in-progress transcript before it
// committed. Consecutive identical partials count once (the user saw no
// change); each distinct successor is one revision.
func PartialRevisions(partials []string) int {
	revisions := 0
	prev := ""
	first := true
	for _, p := range partials {
		if first {
			prev = p
			first = false
			revisions++ // the first partial is itself the first revision shown
			continue
		}
		if p != prev {
			revisions++
			prev = p
		}
	}
	return revisions
}

func minFloat(xs []float64) float64 {
	m := xs[0]
	for _, v := range xs[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxFloat(xs []float64) float64 {
	m := xs[0]
	for _, v := range xs[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
