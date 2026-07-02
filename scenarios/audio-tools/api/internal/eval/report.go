package eval

import (
	"fmt"
	"math"

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
	Model          string
	SlopePerSecond float64
	Intercept      float64
	RSquared       float64
	SampleCount    int
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
	return report
}

func chooseWinner(rows []StrategyReport, latencyMeasured bool) int {
	return chooseWinnerWithScaling(rows, latencyMeasured, true)
}

func chooseAggregateWinner(rows []StrategyReport, latencyMeasured bool) int {
	return chooseWinnerWithScaling(rows, latencyMeasured, false)
}

func chooseWinnerWithScaling(rows []StrategyReport, latencyMeasured bool, scalingAware bool) int {
	best := 0
	for i := 1; i < len(rows); i++ {
		if strategyLess(rows[i], rows[best], latencyMeasured, scalingAware) {
			best = i
		}
	}
	return best
}

func strategyLess(a, b StrategyReport, latencyMeasured bool, scalingAware bool) bool {
	if math.Abs(a.WER-b.WER) > reportWERTieThreshold {
		return a.WER < b.WER
	}
	if scalingAware {
		aRisk, bRisk := longFormScalingRisk(a.Scaling), longFormScalingRisk(b.Scaling)
		if aRisk != bRisk {
			return aRisk < bRisk
		}
	}
	if latencyMeasured && math.Abs(a.FinalizationLatencyP95Ms-b.FinalizationLatencyP95Ms) > 25 {
		return a.FinalizationLatencyP95Ms < b.FinalizationLatencyP95Ms
	}
	if a.WhisperCalls != b.WhisperCalls {
		return a.WhisperCalls < b.WhisperCalls
	}
	return string(a.Strategy) < string(b.Strategy)
}

func corpusConfidence(clips int, audioSeconds float64, latencyMeasured bool) (string, []string) {
	notes := []string{}
	if clips < 10 {
		notes = append(notes, "Fewer than 10 clips makes per-strategy differences easy to overfit.")
	}
	if audioSeconds > 0 && audioSeconds < 120 {
		notes = append(notes, "Less than 120 seconds of audio makes latency and WER less stable.")
	}
	if !latencyMeasured {
		notes = append(notes, "Latency was not measured; the recommendation is quality/cost-only.")
	}
	if len(notes) > 0 {
		return "low", notes
	}
	if clips >= 25 && audioSeconds >= 300 && latencyMeasured {
		return "high", []string{"Corpus size and latency repeats are sufficient for a stronger local recommendation."}
	}
	return "medium", []string{"Recommendation is suitable for local tuning; confirm with a broader corpus before promotion."}
}

func explainStrategy(row, winner StrategyReport, isWinner bool, latencyMeasured bool) (string, []string, []ReportWarning) {
	scalingWarnings := append([]ReportWarning(nil), row.Scaling.Warnings...)
	scalingRisk := longFormScalingRisk(row.Scaling)
	if isWinner {
		reasons := []string{"Lowest WER after deterministic normalization."}
		if latencyMeasured {
			reasons = append(reasons, "Best tie-break balance of p95 finalization latency and compute cost.")
		} else {
			reasons = append(reasons, "Latency was not part of this run.")
		}
		if scalingRisk == 0 {
			reasons = append(reasons, "Long-form scaling is acceptable over the measured sweep.")
		}
		return "winner", reasons, scalingWarnings
	}
	reasons := []string{}
	warnings := scalingWarnings
	verdict := "competitive"
	if row.WERDeltaVsWinner > 0.005 {
		verdict = "loser"
		reasons = append(reasons, fmt.Sprintf("WER is %.1f percentage points worse than the winner.", row.WERDeltaVsWinner*100))
		warnings = append(warnings, ReportWarning{Code: "higher_wer", Severity: "warning", Message: "WER is materially higher than the winning strategy."})
	}
	if row.CallMultiplierVsWinner > 1.5 {
		if verdict == "competitive" {
			verdict = "tradeoff"
		}
		reasons = append(reasons, fmt.Sprintf("Uses %.1fx as many Whisper calls as the winner.", row.CallMultiplierVsWinner))
		warnings = append(warnings, ReportWarning{Code: "higher_compute", Severity: "warning", Message: "Compute cost is materially higher than the winning strategy."})
	}
	if latencyMeasured && row.P95DeltaMsVsWinner > 250 {
		if verdict == "competitive" {
			verdict = "tradeoff"
		}
		reasons = append(reasons, fmt.Sprintf("p95 finalization is %.0f ms slower than the winner.", row.P95DeltaMsVsWinner))
		warnings = append(warnings, ReportWarning{Code: "higher_p95", Severity: "warning", Message: "Finalization latency is materially slower than the winning strategy."})
	}
	if row.PartialRevisions > winner.PartialRevisions+3 {
		if verdict == "competitive" {
			verdict = "tradeoff"
		}
		reasons = append(reasons, "Shows more partial transcript revisions, so live text is less stable.")
		warnings = append(warnings, ReportWarning{Code: "higher_revisions", Severity: "warning", Message: "Partial transcript revisions are higher than the winning strategy."})
	}
	if scalingRisk > longFormScalingRisk(winner.Scaling) {
		if verdict == "competitive" {
			verdict = "tradeoff"
		}
		reasons = append(reasons, scalingRiskReason(row.Scaling))
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "Close to the winner on measured metrics, but loses the deterministic tie-breaks.")
	}
	return verdict, reasons, warnings
}

func winnerReasons(winner StrategyReport, rows []StrategyReport, latencyMeasured bool, aggregateWinnerIndex int) []string {
	reasons := []string{fmt.Sprintf("%s has %.1f%% WER on this corpus.", displayLabel(winner), winner.WER*100)}
	if latencyMeasured {
		reasons = append(reasons, fmt.Sprintf("p95 finalization is %.0f ms with %d Whisper calls.", winner.FinalizationLatencyP95Ms, winner.WhisperCalls))
	} else {
		reasons = append(reasons, fmt.Sprintf("Uses %d Whisper calls over %.1f seconds of audio.", winner.WhisperCalls, winner.WhisperAudioSeconds))
	}
	if aggregateWinnerIndex >= 0 && aggregateWinnerIndex < len(rows) && rows[aggregateWinnerIndex].Strategy != winner.Strategy {
		reasons = append(reasons, fmt.Sprintf("Short-form aggregate tie-breaks favored %s, but %s is safer for long dictation over the measured sweep.", displayLabel(rows[aggregateWinnerIndex]), displayLabel(winner)))
	}
	if longFormScalingRisk(winner.Scaling) == 0 {
		reasons = append(reasons, fmt.Sprintf("Measured scaling is latency=%s and compute=%s.", winner.Scaling.LatencyClassification, winner.Scaling.ComputeClassification))
	}
	for _, row := range rows {
		if row.Strategy == "overlap_agree" && row.Strategy != winner.Strategy && row.WhisperCalls > winner.WhisperCalls {
			reasons = append(reasons, "Overlap-agree is not worth its extra calls on this corpus unless a later run shows better WER or stability.")
			break
		}
	}
	return reasons
}

func longFormScalingRisk(s ScalingAnalysis) int {
	if distinctPointDurations(s.Points) < minScalingDistinctDurations {
		return 1
	}
	if s.LatencyClassification == "superlinear" || s.ComputeClassification == "superlinear" {
		return 2
	}
	if s.LatencyClassification == "inconclusive" || s.ComputeClassification == "inconclusive" {
		return 1
	}
	return 0
}

func scalingRiskReason(s ScalingAnalysis) string {
	if s.LatencyClassification == "superlinear" && s.ComputeClassification == "superlinear" {
		return "Shows superlinear latency and compute growth over the measured duration sweep."
	}
	if s.LatencyClassification == "superlinear" {
		return "Shows superlinear finalization-latency growth over the measured duration sweep."
	}
	if s.ComputeClassification == "superlinear" {
		return "Shows superlinear compute growth over the measured duration sweep."
	}
	return "Scaling evidence is inconclusive for long-form dictation."
}

func callMultiplier(calls, winnerCalls int) float64 {
	if winnerCalls <= 0 {
		if calls <= 0 {
			return 1
		}
		return 0
	}
	return float64(calls) / float64(winnerCalls)
}

func displayLabel(row StrategyReport) string {
	if row.Label != "" {
		return row.Label
	}
	return string(row.Strategy)
}
