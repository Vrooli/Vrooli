package eval

import (
	"audio-tools/internal/ai/sttchain"
)

// ClipResult is the per-clip evaluation outcome for one strategy. Quality
// fields (WER, Whisper-calls, audio-seconds, RTF) come from the
// deterministic pass and are reproducible; latency fields come from the
// real-time-paced pass(es) and are reported as a distribution.
type ClipResult struct {
	ClipID     string
	Reference  string
	Hypothesis string

	WER                 WERResult
	WhisperCalls        int
	WhisperAudioSeconds float64
	RTF                 float64
	SegmentCount        int
	PartialRevisions    int

	// LatencySamplesMs are the finalization-latency samples (last-chunk →
	// terminal Done) gathered over the real-time repeats. Empty when the
	// real-time pass was skipped.
	LatencySamplesMs []float64

	Err error
}

// FinalizationLatencyP50Ms / P95Ms summarize the per-clip latency samples.
func (c ClipResult) FinalizationLatencyP50Ms() float64 { return P50(c.LatencySamplesMs) }
func (c ClipResult) FinalizationLatencyP95Ms() float64 { return P95(c.LatencySamplesMs) }

// StrategyReport aggregates a strategy's results across the whole corpus —
// one row of the comparison table. WER is the corpus micro-average
// (Σ edits / Σ reference words), not the mean of per-clip rates, so long
// clips weigh proportionally. Latency percentiles pool every sample across
// every clip and repeat.
type StrategyReport struct {
	Strategy sttchain.StrategyKind
	Label    string

	WER        float64
	EditCounts EditCounts
	RefWords   int

	WhisperCalls        int
	WhisperAudioSeconds float64
	RTF                 float64

	FinalizationLatencyP50Ms float64
	FinalizationLatencyP95Ms float64
	PartialRevisions         int

	PerClip []ClipResult
}

// EvalReport is the top-level comparison report: one StrategyReport row
// per (strategy, config). Mirrors the AI-search SuiteReport shape.
type EvalReport struct {
	PerStrategy []StrategyReport
	// Mode notes which measurement passes ran, so a consumer can tell
	// whether latency numbers are present/meaningful.
	QualityMeasured bool
	LatencyMeasured bool
}

// aggregateStrategy folds per-clip results (and their pooled latency
// samples) into one StrategyReport.
func aggregateStrategy(kind sttchain.StrategyKind, label string, clips []ClipResult) StrategyReport {
	r := StrategyReport{Strategy: kind, Label: label, PerClip: clips}
	var latency []float64
	var totalAudio, totalRTFWeighted float64
	for _, c := range clips {
		r.EditCounts.Substitutions += c.WER.Substitutions
		r.EditCounts.Insertions += c.WER.Insertions
		r.EditCounts.Deletions += c.WER.Deletions
		r.RefWords += c.WER.RefWords
		r.WhisperCalls += c.WhisperCalls
		r.WhisperAudioSeconds += c.WhisperAudioSeconds
		r.PartialRevisions += c.PartialRevisions
		latency = append(latency, c.LatencySamplesMs...)
		// RTF aggregate is audio-weighted: Σ(rtf_i * audio_i) / Σ audio_i,
		// which equals Σ provider-time / Σ audio across clips.
		totalAudio += c.WhisperAudioSeconds
		totalRTFWeighted += c.RTF * c.WhisperAudioSeconds
	}
	if r.RefWords > 0 {
		r.WER = float64(r.EditCounts.Total()) / float64(r.RefWords)
	}
	if totalAudio > 0 {
		r.RTF = totalRTFWeighted / totalAudio
	}
	r.FinalizationLatencyP50Ms = P50(latency)
	r.FinalizationLatencyP95Ms = P95(latency)
	return r
}
