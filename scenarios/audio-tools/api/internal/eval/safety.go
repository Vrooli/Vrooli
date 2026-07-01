package eval

import (
	"fmt"

	sttchain "audio-tools/internal/ai/sttchain"
)

// EvaluateSafety computes the phase-7 hard gates for one clip. WER remains
// the graded metric; these gates call out catastrophic streaming failures
// that averages can hide.
func EvaluateSafety(ops []EditOperation, timeline []CommitState, opts SafetyOptions) SafetyGateReport {
	threshold := opts.DroppedSpanThresholdWords
	if threshold <= 0 {
		threshold = DefaultDroppedSpanThresholdWords
	}
	out := SafetyGateReport{
		RetractionFree:            true,
		DroppedSpanFree:           true,
		DroppedSpanThresholdWords: threshold,
	}
	out.MaxDroppedSpanWords = maxContiguousDeletedWords(ops)
	out.RetractionEvents = detectRetractions(timeline)
	if len(out.RetractionEvents) > 0 {
		out.RetractionFree = false
		out.Reasons = append(out.Reasons, fmt.Sprintf("%d committed-text retraction(s) detected", len(out.RetractionEvents)))
	}
	if out.MaxDroppedSpanWords >= threshold {
		out.DroppedSpanFree = false
		out.Reasons = append(out.Reasons, fmt.Sprintf("max contiguous dropped span is %d words (threshold %d)", out.MaxDroppedSpanWords, threshold))
	}
	out.Passed = out.RetractionFree && out.DroppedSpanFree
	if out.Passed {
		out.Reasons = append(out.Reasons, "no committed-text retractions or threshold-sized dropped spans detected")
	}
	return out
}

func detectRetractions(timeline []CommitState) []RetractionEvent {
	var out []RetractionEvent
	if len(timeline) < 2 {
		return out
	}
	norm := DefaultNormalizeOptions()
	prevText := timeline[0].Text
	prev := Tokenize(prevText, norm)
	for _, state := range timeline[1:] {
		current := Tokenize(state.Text, norm)
		if !tokensPrefix(prev, current) {
			out = append(out, RetractionEvent{
				PreviousText: prevText,
				CurrentText:  state.Text,
				AtMs:         state.AtMs,
			})
		}
		prevText = state.Text
		prev = current
	}
	return out
}

func tokensPrefix(prefix, full []string) bool {
	if len(prefix) > len(full) {
		return false
	}
	for i := range prefix {
		if prefix[i] != full[i] {
			return false
		}
	}
	return true
}

func maxContiguousDeletedWords(ops []EditOperation) int {
	maxRun, run := 0, 0
	for _, op := range ops {
		if op.Kind == "deletion" {
			run++
			if run > maxRun {
				maxRun = run
			}
			continue
		}
		run = 0
	}
	return maxRun
}

func aggregateSafety(clips []ClipResult) SafetyGateReport {
	out := SafetyGateReport{
		Passed:                    true,
		RetractionFree:            true,
		DroppedSpanFree:           true,
		DroppedSpanThresholdWords: DefaultDroppedSpanThresholdWords,
	}
	for _, clip := range clips {
		if clip.Safety.DroppedSpanThresholdWords > 0 {
			out.DroppedSpanThresholdWords = clip.Safety.DroppedSpanThresholdWords
		}
		if !clip.Safety.Passed {
			out.Passed = false
		}
		if !clip.Safety.RetractionFree {
			out.RetractionFree = false
		}
		if !clip.Safety.DroppedSpanFree {
			out.DroppedSpanFree = false
		}
		out.RetractionEvents = append(out.RetractionEvents, clip.Safety.RetractionEvents...)
		if clip.Safety.MaxDroppedSpanWords > out.MaxDroppedSpanWords {
			out.MaxDroppedSpanWords = clip.Safety.MaxDroppedSpanWords
		}
	}
	if !out.RetractionFree {
		out.Reasons = append(out.Reasons, fmt.Sprintf("%d committed-text retraction(s) detected", len(out.RetractionEvents)))
	}
	if !out.DroppedSpanFree {
		out.Reasons = append(out.Reasons, fmt.Sprintf("max contiguous dropped span is %d words (threshold %d)", out.MaxDroppedSpanWords, out.DroppedSpanThresholdWords))
	}
	if out.Passed {
		out.Reasons = append(out.Reasons, "all clips pass retraction and dropped-span gates")
	}
	return out
}

func attributeStages(row StrategyReport) StageAttribution {
	out := StageAttribution{
		StrategyLostWords:  row.EditCounts.Substitutions + row.EditCounts.Deletions,
		EgressRejectEvents: row.SpeakerRejectionCount,
	}
	if row.SpeakerRejectionCount > 0 {
		out.EgressLostWords = row.EditCounts.Deletions
		out.Notes = append(out.Notes, "speaker rejection events occurred; deletion attribution is conservative because rejected segment text is not emitted")
	}
	if row.SpeakerRejectionCount == 0 {
		out.Notes = append(out.Notes, "no speaker rejection events were observed; lost recognized words are attributed to the strategy/input path")
	}
	return out
}

// AttributeIngressByAblation fills the ingress (target-speaker extraction)
// stage attribution that a single row cannot observe on its own. Per-row, the
// extraction stage is invisible: there is no way to know how many words the
// extractor dropped without a no-extraction baseline. The ablation grid
// provides exactly that, so for every extraction-on row we find its
// extraction-off sibling (same base strategy, same verification state, same
// augmentation group) and attribute the extra deletions to ingress. The
// surplus is moved out of StrategyLostWords so the decomposition does not
// double-count it. Rows without ablation identity are left untouched.
func AttributeIngressByAblation(report *EvalReport) {
	if report == nil {
		return
	}
	type key struct {
		base   sttchain.StrategyKind
		verify bool
		group  string
	}
	offDeletions := make(map[key]int, len(report.PerStrategy))
	for _, row := range report.PerStrategy {
		if row.BaseStrategy == "" || row.ExtractionEnabled {
			continue
		}
		offDeletions[key{row.BaseStrategy, row.VerificationEnabled, row.ConditionGroup}] = row.EditCounts.Deletions
	}
	for i := range report.PerStrategy {
		row := &report.PerStrategy[i]
		if row.BaseStrategy == "" || !row.ExtractionEnabled {
			continue
		}
		baseline, ok := offDeletions[key{row.BaseStrategy, row.VerificationEnabled, row.ConditionGroup}]
		if !ok {
			continue
		}
		surplus := row.EditCounts.Deletions - baseline
		if surplus < 0 {
			surplus = 0
		}
		row.StageAttribution.IngressLostWords = surplus
		if surplus > 0 && row.StageAttribution.StrategyLostWords >= surplus {
			row.StageAttribution.StrategyLostWords -= surplus
		}
		row.StageAttribution.Notes = append(row.StageAttribution.Notes,
			fmt.Sprintf("ingress attribution from ablation: %d extra deleted word(s) vs the extraction-off sibling are attributed to target-speaker extraction", surplus))
	}
}

type lengthBucket struct {
	label string
	min   int64
	max   int64
}

var lengthBuckets = []lengthBucket{
	{label: "<=10s", min: 0, max: 10_000},
	{label: "<=30s", min: 10_001, max: 30_000},
	{label: "<=1m", min: 30_001, max: 60_000},
	{label: "<=3m", min: 60_001, max: 180_000},
	{label: "<=5m", min: 180_001, max: 300_000},
	{label: ">5m", min: 300_001, max: 0},
}

func buildLengthCurves(clips []ClipResult) []LengthBucketCurve {
	out := make([]LengthBucketCurve, 0, len(lengthBuckets))
	for _, bucket := range lengthBuckets {
		var curve LengthBucketCurve
		curve.Bucket = bucket.label
		curve.MinDurationMs = bucket.min
		curve.MaxDurationMs = bucket.max
		var counts EditCounts
		refWords := 0
		var latency []float64
		var firstCommit []float64
		for _, clip := range clips {
			if !durationInBucket(clip.AudioDurationMs, bucket) {
				continue
			}
			curve.ClipCount++
			counts.Substitutions += clip.WER.Substitutions
			counts.Insertions += clip.WER.Insertions
			counts.Deletions += clip.WER.Deletions
			refWords += clip.WER.RefWords
			latency = append(latency, clip.LatencySamplesMs...)
			if clip.TimeToFirstCommitMs > 0 {
				firstCommit = append(firstCommit, clip.TimeToFirstCommitMs)
			}
			if clip.Safety.MaxDroppedSpanWords > curve.MaxDroppedSpanWords {
				curve.MaxDroppedSpanWords = clip.Safety.MaxDroppedSpanWords
			}
		}
		if curve.ClipCount == 0 {
			continue
		}
		if refWords > 0 {
			curve.WER = float64(counts.Total()) / float64(refWords)
		}
		curve.FinalizationLatencyP95Ms = P95(latency)
		curve.MeanTimeToFirstCommitMs = Mean(firstCommit)
		out = append(out, curve)
	}
	return out
}

func durationInBucket(durationMs int64, bucket lengthBucket) bool {
	if bucket.max == 0 {
		return durationMs >= bucket.min
	}
	return durationMs >= bucket.min && durationMs <= bucket.max
}
