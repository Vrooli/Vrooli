//go:build audio_lab

package eval

import (
	"fmt"

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
	// Provider-reported per-word timings captured during replay. These are
	// observed hypothesis data, not reference alignments. Times are relative
	// to each provider request; providers that omit timings produce no entries.
	HypothesisWordTimestamps          []sttchain.TimedWord `json:"hypothesis_word_timestamps,omitempty"`
	HypothesisWordTimestampsTruncated bool                 `json:"hypothesis_word_timestamps_truncated,omitempty"`

	WER                   WERResult
	NormalizedReference   string
	NormalizedHypothesis  string
	EditOperations        []EditOperation
	WhisperCalls          int
	WhisperAudioSeconds   float64
	ProviderLatencyMs     float64
	RTF                   float64
	SegmentCount          int
	PartialRevisions      int
	CommitTimeline        []CommitState
	TimeToFirstCommitMs   float64
	CommitCount           int
	SpeakerRejectionCount int
	AudioDurationMs       int64
	Safety                SafetyGateReport
	// TimestampDrift is populated only when a run supplies a timed, versioned
	// reference and timed provider output. It is descriptive, never a promotion
	// gate by itself.
	TimestampDrift           *TimestampDriftResult      `json:"timestamp_drift,omitempty"`
	TimestampDriftSkipReason string                     `json:"timestamp_drift_skip_reason,omitempty"`
	TimestampReference       *TimingReferenceProvenance `json:"timestamp_reference,omitempty"`

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

	// Provider-neutral cell identity is persisted with each report row. These
	// fields deliberately name the executed route and evidence lane instead of
	// deriving them from labels or legacy Whisper-specific strategy names.
	EngineID      string
	ModelID       string
	PolicyProfile string
	ReplayLane    string
	FaultProfile  string

	WER        float64
	EditCounts EditCounts
	RefWords   int

	WhisperCalls        int
	WhisperAudioSeconds float64
	RTF                 float64

	FinalizationLatencyP50Ms float64
	FinalizationLatencyP95Ms float64
	PartialRevisions         int
	CommitCount              int
	SpeakerRejectionCount    int
	Safety                   SafetyGateReport
	StageAttribution         StageAttribution
	LengthCurves             []LengthBucketCurve
	Scaling                  ScalingAnalysis

	// Ablation identity. These let AttributeIngressByAblation pair an
	// extraction-on row with its extraction-off sibling so word loss can be
	// attributed to the ingress (target-speaker extraction) stage. They are
	// set by the experiment assembler when it suffixes condition rows; bare
	// eval rows leave them zero/empty.
	BaseStrategy        sttchain.StrategyKind
	ExtractionEnabled   bool
	VerificationEnabled bool
	ConditionGroup      string

	WERDeltaVsWinner       float64
	P95DeltaMsVsWinner     float64
	CallMultiplierVsWinner float64
	Verdict                string
	Reasons                []string
	Warnings               []ReportWarning

	PerClip []ClipResult
}

type EvalReportSummary struct {
	WinnerStrategy  string
	WinnerLabel     string
	Recommendation  string
	Confidence      string
	Reasons         []string
	ConfidenceNotes []string
}

type ReportWarning struct {
	Code     string
	Message  string
	Severity string
}

type NormalizationPolicy struct {
	WERPolicy              string
	OverlapAgreementPolicy string
}

const (
	DefaultDroppedSpanThresholdWords = 4
	reportWERTieThreshold            = 0.0005
)

// CommitState is one committed-text snapshot emitted by the strategy or
// Segmenter. A valid streaming strategy must only append stable text over
// time; changing/removing an earlier committed token fails the retraction
// gate.
type CommitState struct {
	Text       string
	AtMs       int64
	AudioEndMs int64
}

type SafetyOptions struct {
	DroppedSpanThresholdWords int
}

type RetractionEvent struct {
	PreviousText string
	CurrentText  string
	AtMs         int64
}

type SafetyGateReport struct {
	Passed                    bool
	RetractionFree            bool
	DroppedSpanFree           bool
	RetractionEvents          []RetractionEvent
	MaxDroppedSpanWords       int
	DroppedSpanThresholdWords int
	Reasons                   []string
}

type StageAttribution struct {
	IngressLostWords   int
	StrategyLostWords  int
	EgressLostWords    int
	EgressRejectEvents int
	Notes              []string
}

type LengthBucketCurve struct {
	Bucket                   string
	MinDurationMs            int64
	MaxDurationMs            int64
	ClipCount                int
	WER                      float64
	FinalizationLatencyP95Ms float64
	MeanTimeToFirstCommitMs  float64
	MaxDroppedSpanWords      int
}

type ScalingPoint struct {
	ClipID                         string
	TargetDurationMs               int64
	RealizedDurationMs             int64
	WER                            float64
	FinalizationLatencyP50Ms       float64
	FinalizationLatencyP95Ms       float64
	FinalizationLatencySampleCount int
	TimeToFirstCommitMs            float64
	CommitCount                    int
	PartialRevisions               int
	MaxDroppedSpanWords            int
	WhisperCalls                   int
	WhisperAudioSeconds            float64
	ProviderLatencyMs              float64
	RTF                            float64
}

type ScalingModelFit struct {
	Metric         string
	Unit           string
	Model          string
	SlopePerSecond float64
	Intercept      float64
	RSquared       float64
	SampleCount    int
	Exponent       float64
	ExponentR2     float64
	Reason         string
}

type ScalingAnalysis struct {
	Points                []ScalingPoint
	LatencyClassification string
	ComputeClassification string
	Confidence            string
	Reasons               []string
	Warnings              []ReportWarning
	LatencyFit            ScalingModelFit
	ComputeFit            ScalingModelFit
	MetricFits            []ScalingModelFit
}

// EvalReport is the top-level comparison report: one StrategyReport row
// per (strategy, config). Mirrors the AI-search SuiteReport shape.
type EvalReport struct {
	PerStrategy []StrategyReport
	// Mode notes which measurement passes ran, so a consumer can tell
	// whether latency numbers are present/meaningful.
	QualityMeasured     bool
	LatencyMeasured     bool
	Summary             EvalReportSummary
	Warnings            []ReportWarning
	NormalizationPolicy NormalizationPolicy
	PromotionVerdicts   []PromotionVerdict
}

// PromotionVerdict is separate from strategy ranking: a locally preferred
// strategy is not necessarily eligible for stable provider promotion.
type PromotionVerdict struct {
	EngineID      string
	ModelID       string
	Strategy      string
	PolicyProfile string
	Stable        bool
	Reasons       []string
}

// aggregateStrategy folds per-clip results (and their pooled latency
// samples) into one StrategyReport.
func aggregateStrategy(kind sttchain.StrategyKind, label string, clips []ClipResult) StrategyReport {
	r := StrategyReport{Strategy: kind, Label: label, PerClip: clips}
	var latency []float64
	var totalAudio, totalRTFWeighted float64
	r.Safety = aggregateSafety(clips)
	for _, c := range clips {
		r.EditCounts.Substitutions += c.WER.Substitutions
		r.EditCounts.Insertions += c.WER.Insertions
		r.EditCounts.Deletions += c.WER.Deletions
		r.RefWords += c.WER.RefWords
		r.WhisperCalls += c.WhisperCalls
		r.WhisperAudioSeconds += c.WhisperAudioSeconds
		r.PartialRevisions += c.PartialRevisions
		r.CommitCount += c.CommitCount
		r.SpeakerRejectionCount += c.SpeakerRejectionCount
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
	r.StageAttribution = attributeStages(r)
	r.LengthCurves = buildLengthCurves(clips)
	r.Scaling = buildScalingAnalysis(clips)
	return r
}

func explainReport(report EvalReport) EvalReport {
	report.NormalizationPolicy = NormalizationPolicy{
		WERPolicy:              "WER lowercases text, removes Unicode punctuation and symbols, collapses whitespace, then compares whitespace-delimited tokens.",
		OverlapAgreementPolicy: "Overlap-agree compares whitespace tokens after lowercasing and stripping Unicode punctuation/symbols for agreement only; committed text remains verbatim from the first agreeing hypothesis.",
	}
	if len(report.PerStrategy) == 0 {
		report.Summary = EvalReportSummary{Confidence: "low", Recommendation: "No strategies were evaluated."}
		return report
	}

	aggregateWinnerIndex := chooseAggregateWinner(report.PerStrategy, report.LatencyMeasured)
	winnerIndex := chooseWinner(report.PerStrategy, report.LatencyMeasured)
	winner := report.PerStrategy[winnerIndex]
	totalClips := len(winner.PerClip)
	totalAudio := winner.WhisperAudioSeconds
	confidence, notes := corpusConfidence(totalClips, totalAudio, report.LatencyMeasured)
	if totalClips < 10 {
		report.Warnings = append(report.Warnings, ReportWarning{
			Code:     "tiny_corpus",
			Severity: "warning",
			Message:  fmt.Sprintf("Only %d clips were evaluated; use the recommendation as a direction, not a promotion decision.", totalClips),
		})
	}
	if totalAudio > 0 && totalAudio < 120 {
		report.Warnings = append(report.Warnings, ReportWarning{
			Code:     "short_audio",
			Severity: "warning",
			Message:  fmt.Sprintf("The evaluated audio totals %.1f seconds; at least 120 seconds gives a more stable comparison.", totalAudio),
		})
	}
	if !report.LatencyMeasured {
		report.Warnings = append(report.Warnings, ReportWarning{
			Code:     "latency_not_measured",
			Severity: "info",
			Message:  "Latency columns were not measured because real-time repeats were disabled.",
		})
	}

	for i := range report.PerStrategy {
		row := &report.PerStrategy[i]
		row.WERDeltaVsWinner = row.WER - winner.WER
		row.P95DeltaMsVsWinner = row.FinalizationLatencyP95Ms - winner.FinalizationLatencyP95Ms
		row.CallMultiplierVsWinner = callMultiplier(row.WhisperCalls, winner.WhisperCalls)
		row.Verdict, row.Reasons, row.Warnings = explainStrategy(*row, winner, i == winnerIndex, report.LatencyMeasured)
	}

	report.Summary = EvalReportSummary{
		WinnerStrategy:  string(winner.Strategy),
		WinnerLabel:     displayLabel(winner),
		Recommendation:  fmt.Sprintf("Prefer %s for this corpus.", displayLabel(winner)),
		Confidence:      confidence,
		Reasons:         winnerReasons(winner, report.PerStrategy, report.LatencyMeasured, aggregateWinnerIndex),
		ConfidenceNotes: notes,
	}
	if !strategySafetyPassed(winner) {
		report.Summary.Recommendation = fmt.Sprintf("Prefer %s for this corpus, recommended despite safety failure.", displayLabel(winner))
		report.Summary.Confidence = "low"
		report.Summary.ConfidenceNotes = append(report.Summary.ConfidenceNotes, "Every evaluated strategy failed the safety gate; treat this only as the least-bad diagnostic result.")
	}
	return report
}
