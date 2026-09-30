package eval

import (
	"fmt"
	"math"
	"sort"
)

const minScalingDistinctDurations = 3

const (
	minScalingSuperlinearDistinctDurations = 4
	scalingFlatRelativeRange               = 0.12
	scalingSuperlinearExponentMargin       = 0.10
	scalingUsefulFitR2                     = 0.70
	scalingNearLinearR2Margin              = 0.01
	scalingMinimumPositiveDuration         = 0.001
	scalingAdequateLatencySamples          = 2
)

func buildScalingAnalysis(clips []ClipResult) ScalingAnalysis {
	points := make([]ScalingPoint, 0, len(clips))
	distinctDurations := map[int64]struct{}{}
	for _, clip := range clips {
		if clip.AudioDurationMs <= 0 {
			continue
		}
		distinctDurations[clip.AudioDurationMs] = struct{}{}
		points = append(points, ScalingPoint{
			ClipID:                         clip.ClipID,
			RealizedDurationMs:             clip.AudioDurationMs,
			WER:                            clip.WER.Rate(),
			FinalizationLatencyP50Ms:       clip.FinalizationLatencyP50Ms(),
			FinalizationLatencyP95Ms:       clip.FinalizationLatencyP95Ms(),
			FinalizationLatencySampleCount: len(clip.LatencySamplesMs),
			TimeToFirstCommitMs:            clip.TimeToFirstCommitMs,
			CommitCount:                    clip.CommitCount,
			PartialRevisions:               clip.PartialRevisions,
			MaxDroppedSpanWords:            clip.Safety.MaxDroppedSpanWords,
			WhisperCalls:                   clip.WhisperCalls,
			WhisperAudioSeconds:            clip.WhisperAudioSeconds,
			ProviderLatencyMs:              clip.ProviderLatencyMs,
			RTF:                            clip.RTF,
		})
	}

	analysis := ScalingAnalysis{
		Points:                points,
		LatencyClassification: "inconclusive",
		ComputeClassification: "inconclusive",
		Confidence:            "low",
	}
	if len(points) == 0 {
		analysis.Reasons = append(analysis.Reasons, "No positive-duration clips were available for scaling analysis.")
		analysis.Warnings = append(analysis.Warnings, ReportWarning{
			Code:     "insufficient_scaling_points",
			Severity: "info",
			Message:  "Scaling analysis needs at least three distinct positive durations.",
		})
		return analysis
	}
	if len(distinctDurations) < minScalingDistinctDurations {
		analysis.Reasons = append(analysis.Reasons, fmt.Sprintf("Only %d distinct positive duration(s) were measured; at least %d are required to classify scaling.", len(distinctDurations), minScalingDistinctDurations))
		analysis.Warnings = append(analysis.Warnings, ReportWarning{
			Code:     "insufficient_scaling_points",
			Severity: "info",
			Message:  "Scaling classification is inconclusive because the sweep has too few distinct durations.",
		})
		return analysis
	}

	fits, classifications, reasons := classifyScalingMetrics(points)
	latencyFit, latencyClassification := fitForMetric(fits, classifications, metricFinalizationP95)
	computeFit, computeClassification := highestRiskFit(fits, classifications, scalingComputeMetrics)
	analysis.LatencyFit = latencyFit
	analysis.ComputeFit = computeFit
	analysis.MetricFits = fits
	analysis.LatencyClassification = latencyClassification
	analysis.ComputeClassification = computeClassification
	analysis.Confidence = scalingConfidence(len(distinctDurations), latencyClassification, computeClassification, hasSingleSampleLatency(points))
	analysis.Reasons = append(analysis.Reasons, reasons...)
	if latencyClassification == "superlinear" {
		analysis.Warnings = append(analysis.Warnings, ReportWarning{
			Code:     "superlinear_latency_growth",
			Severity: "warning",
			Message:  "Finalization latency appears to grow faster than audio duration over the measured sweep.",
		})
	}
	if computeClassification == "superlinear" {
		analysis.Warnings = append(analysis.Warnings, ReportWarning{
			Code:     "superlinear_compute_growth",
			Severity: "warning",
			Message:  "At least one compute metric appears to grow faster than audio duration over the measured sweep.",
		})
	}
	return analysis
}

type scalingMetric string

const (
	metricFinalizationP95 scalingMetric = "finalization_latency_p95_ms"
	metricTTFC            scalingMetric = "time_to_first_commit_ms"
	metricWER             scalingMetric = "wer"
	metricDroppedSpan     scalingMetric = "max_dropped_span_words"
	metricProviderLatency scalingMetric = "provider_latency_ms"
	metricWhisperCalls    scalingMetric = "whisper_calls"
	metricWhisperAudioSec scalingMetric = "whisper_audio_seconds"
	metricRTF             scalingMetric = "rtf"
)

type scalingMetricDef struct {
	name    scalingMetric
	unit    string
	compute bool
	extract func(ScalingPoint) (float64, bool)
	weight  func(ScalingPoint) float64
}

var scalingMetricRegistry = []scalingMetricDef{
	{
		name: metricFinalizationP95, unit: "ms",
		extract: func(point ScalingPoint) (float64, bool) {
			return point.FinalizationLatencyP95Ms, point.FinalizationLatencySampleCount > 0
		},
		weight: latencySampleWeight,
	},
	{
		name: metricTTFC, unit: "ms",
		extract: func(point ScalingPoint) (float64, bool) {
			return point.TimeToFirstCommitMs, point.TimeToFirstCommitMs > 0
		},
	},
	{
		name: metricWER, unit: "rate",
		extract: func(point ScalingPoint) (float64, bool) {
			return point.WER, point.WER > 0
		},
	},
	{
		name: metricDroppedSpan, unit: "words",
		extract: func(point ScalingPoint) (float64, bool) {
			return float64(point.MaxDroppedSpanWords), point.MaxDroppedSpanWords > 0
		},
	},
	{
		name: metricProviderLatency, unit: "ms", compute: true,
		extract: func(point ScalingPoint) (float64, bool) {
			return point.ProviderLatencyMs, point.ProviderLatencyMs > 0
		},
	},
	{
		name: metricWhisperCalls, unit: "calls", compute: true,
		extract: func(point ScalingPoint) (float64, bool) {
			return float64(point.WhisperCalls), point.WhisperCalls > 0
		},
	},
	{
		name: metricWhisperAudioSec, unit: "seconds", compute: true,
		extract: func(point ScalingPoint) (float64, bool) {
			return point.WhisperAudioSeconds, point.WhisperAudioSeconds > 0
		},
	},
	{
		name: metricRTF, unit: "ratio", compute: true,
		extract: func(point ScalingPoint) (float64, bool) {
			return point.RTF, point.RTF > 0
		},
	},
}

var scalingComputeMetrics = []scalingMetric{
	metricProviderLatency,
	metricWhisperCalls,
	metricWhisperAudioSec,
	metricRTF,
}

type metricSample struct {
	durationSeconds float64
	value           float64
	weight          float64
}

type candidateFit struct {
	ScalingModelFit
	transform func(float64) float64
}

func classifyScalingMetrics(points []ScalingPoint) ([]ScalingModelFit, map[scalingMetric]string, []string) {
	fits := make([]ScalingModelFit, 0, len(scalingMetricRegistry))
	classifications := make(map[scalingMetric]string, len(scalingMetricRegistry))
	reasons := make([]string, 0, len(scalingMetricRegistry))
	for _, metric := range scalingMetricRegistry {
		fit, classification, reason := classifyScaling(points, metric)
		fits = append(fits, fit)
		classifications[metric.name] = classification
		reasons = append(reasons, reason)
	}
	return fits, classifications, reasons
}

func classifyScaling(points []ScalingPoint, metric scalingMetricDef) (ScalingModelFit, string, string) {
	samples := scalingSamples(points, metric)
	distinctDurations := distinctSampleDurations(samples)
	if distinctDurations < minScalingDistinctDurations {
		return ScalingModelFit{Metric: string(metric.name), Unit: metric.unit, Model: "none", SampleCount: len(samples), Reason: "insufficient distinct positive durations"}, "inconclusive", fmt.Sprintf("%s scaling is inconclusive because fewer than %d distinct positive durations were measured.", metric.name, minScalingDistinctDurations)
	}
	if robustRelativeValueRange(samples) <= scalingFlatRelativeRange {
		fit := constantFit(samples, metric, "values stayed within the flat threshold")
		return fit, "flat", fmt.Sprintf("%s stayed effectively flat across the measured durations.", metric.name)
	}

	candidates := []candidateFit{
		fitTransformed(samples, metric, "linear", func(x float64) float64 { return x }),
		fitTransformed(samples, metric, "n_log_n", func(x float64) float64 {
			if x <= 1 {
				return x
			}
			return x * math.Log(x)
		}),
		fitTransformed(samples, metric, "quadratic", func(x float64) float64 { return x * x }),
	}
	logFit := fitLogLog(samples, metric)
	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.RSquared > best.RSquared {
			best = candidate
		}
	}
	linear := candidates[0]
	if best.RSquared < scalingUsefulFitR2 {
		best.Reason = "no model cleared the useful-fit threshold"
		best.Exponent = logFit.Exponent
		best.ExponentR2 = logFit.ExponentR2
		return best.ScalingModelFit, "inconclusive", fmt.Sprintf("%s scaling is inconclusive; the best %s fit had R^2 %.2f.", metric.name, best.Model, best.RSquared)
	}
	best.Exponent = logFit.Exponent
	best.ExponentR2 = logFit.ExponentR2
	if canClassifySuperlinear(metric, samples, distinctDurations, logFit) {
		best.Reason = "log-log exponent and model fit show faster-than-linear growth"
		return best.ScalingModelFit, "superlinear", fmt.Sprintf("%s fit %s with exponent %.2f over the measured sweep.", metric.name, best.Model, best.Exponent)
	}
	if linear.RSquared >= best.RSquared-scalingNearLinearR2Margin {
		linear.Reason = "linear fit was competitive with the best model"
		linear.Exponent = logFit.Exponent
		linear.ExponentR2 = logFit.ExponentR2
		return linear.ScalingModelFit, "linear", fmt.Sprintf("%s scaled approximately linearly over the measured sweep.", metric.name)
	}

	best.Reason = "best model was not materially superlinear"
	return best.ScalingModelFit, "linear", fmt.Sprintf("%s scaled close enough to linear for this sweep.", metric.name)
}

func fitForMetric(fits []ScalingModelFit, classifications map[scalingMetric]string, metric scalingMetric) (ScalingModelFit, string) {
	for _, fit := range fits {
		if fit.Metric == string(metric) {
			return fit, classifications[metric]
		}
	}
	return ScalingModelFit{Metric: string(metric), Model: "none", Reason: "metric was not registered"}, "inconclusive"
}

func highestRiskFit(fits []ScalingModelFit, classifications map[scalingMetric]string, metrics []scalingMetric) (ScalingModelFit, string) {
	var chosenFit ScalingModelFit
	chosenClassification := "inconclusive"
	for _, metric := range metrics {
		fit, classification := fitForMetric(fits, classifications, metric)
		if chosenFit == (ScalingModelFit{}) || scalingClassificationRisk(classification) > scalingClassificationRisk(chosenClassification) {
			chosenFit = fit
			chosenClassification = classification
		}
	}
	if chosenFit != (ScalingModelFit{}) {
		chosenFit.Reason = fmt.Sprintf("%s; selected as the highest-risk compute metric across provider latency, calls, processed audio seconds, and RTF", chosenFit.Reason)
	}
	return chosenFit, chosenClassification
}

func scalingClassificationRisk(classification string) int {
	switch classification {
	case "superlinear":
		return 3
	case "linear":
		return 2
	case "flat":
		return 1
	default:
		return 0
	}
}

func scalingSamples(points []ScalingPoint, metric scalingMetricDef) []metricSample {
	out := make([]metricSample, 0, len(points))
	for _, point := range points {
		durationSeconds := float64(point.RealizedDurationMs) / 1000
		if durationSeconds < scalingMinimumPositiveDuration {
			continue
		}
		value, ok := metric.extract(point)
		if !ok || value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		weight := 1.0
		if metric.weight != nil {
			weight = metric.weight(point)
		}
		if weight <= 0 || math.IsNaN(weight) || math.IsInf(weight, 0) {
			weight = 1
		}
		out = append(out, metricSample{durationSeconds: durationSeconds, value: value, weight: weight})
	}
	return out
}

func robustRelativeValueRange(samples []metricSample) float64 {
	if len(samples) == 0 {
		return 0
	}
	values := make([]float64, 0, len(samples))
	for _, sample := range samples {
		values = append(values, sample.value)
	}
	sort.Float64s(values)
	lowIndex := 0
	highIndex := len(values) - 1
	if len(values) >= 5 {
		lowIndex = int(math.Floor(float64(len(values)-1) * 0.10))
		highIndex = int(math.Ceil(float64(len(values)-1) * 0.90))
	}
	minValue, maxValue := values[lowIndex], values[highIndex]
	if maxValue <= 0 {
		return 0
	}
	return (maxValue - minValue) / maxValue
}

func distinctSampleDurations(samples []metricSample) int {
	durations := map[int64]struct{}{}
	for _, sample := range samples {
		if sample.durationSeconds < scalingMinimumPositiveDuration {
			continue
		}
		durations[int64(math.Round(sample.durationSeconds*1000))] = struct{}{}
	}
	return len(durations)
}

func distinctPointDurations(points []ScalingPoint) int {
	durations := map[int64]struct{}{}
	for _, point := range points {
		if point.RealizedDurationMs <= 0 {
			continue
		}
		durations[point.RealizedDurationMs] = struct{}{}
	}
	return len(durations)
}

func constantFit(samples []metricSample, metric scalingMetricDef, reason string) ScalingModelFit {
	var sum, weightSum float64
	for _, sample := range samples {
		sum += sample.value * sample.weight
		weightSum += sample.weight
	}
	mean := sum / weightSum
	return ScalingModelFit{
		Metric:      string(metric.name),
		Unit:        metric.unit,
		Model:       "constant",
		Intercept:   mean,
		RSquared:    1,
		SampleCount: len(samples),
		Reason:      reason,
	}
}

func fitTransformed(samples []metricSample, metric scalingMetricDef, model string, transform func(float64) float64) candidateFit {
	var sumWeight, sumX, sumY float64
	xs := make([]float64, 0, len(samples))
	for _, sample := range samples {
		x := transform(sample.durationSeconds)
		xs = append(xs, x)
		sumWeight += sample.weight
		sumX += x * sample.weight
		sumY += sample.value * sample.weight
	}
	meanX := sumX / sumWeight
	meanY := sumY / sumWeight
	var numerator, denominator float64
	for i, sample := range samples {
		dx := xs[i] - meanX
		numerator += sample.weight * dx * (sample.value - meanY)
		denominator += sample.weight * dx * dx
	}
	slope := 0.0
	if denominator > 0 {
		slope = numerator / denominator
	}
	intercept := meanY - slope*meanX
	var residual, total float64
	for i, sample := range samples {
		predicted := intercept + slope*xs[i]
		delta := sample.value - predicted
		residual += sample.weight * delta * delta
		centered := sample.value - meanY
		total += sample.weight * centered * centered
	}
	r2 := 1.0
	if total > 0 {
		r2 = 1 - residual/total
	}
	if r2 < 0 {
		r2 = 0
	}
	return candidateFit{
		ScalingModelFit: ScalingModelFit{
			Metric:         string(metric.name),
			Unit:           metric.unit,
			Model:          model,
			SlopePerSecond: slope,
			Intercept:      intercept,
			RSquared:       r2,
			SampleCount:    len(samples),
		},
		transform: transform,
	}
}

func fitLogLog(samples []metricSample, metric scalingMetricDef) ScalingModelFit {
	logSamples := make([]metricSample, 0, len(samples))
	for _, sample := range samples {
		if sample.durationSeconds <= 0 || sample.value <= 0 {
			continue
		}
		logSamples = append(logSamples, metricSample{
			durationSeconds: sample.durationSeconds,
			value:           math.Log(sample.value),
			weight:          sample.weight,
		})
	}
	if len(logSamples) < minScalingDistinctDurations {
		return ScalingModelFit{Metric: string(metric.name), Unit: metric.unit, Model: "log_log", Reason: "insufficient positive log-log samples"}
	}
	fit := fitTransformed(logSamples, metric, "log_log", math.Log).ScalingModelFit
	fit.Model = "log_log"
	fit.Exponent = fit.SlopePerSecond
	fit.ExponentR2 = fit.RSquared
	return fit
}

func canClassifySuperlinear(metric scalingMetricDef, samples []metricSample, distinctDurations int, logFit ScalingModelFit) bool {
	if distinctDurations < minScalingSuperlinearDistinctDurations {
		return false
	}
	if metric.name == metricFinalizationP95 && !latencySamplesAdequate(samples) {
		return false
	}
	return logFit.Exponent > 1+scalingSuperlinearExponentMargin && logFit.ExponentR2 >= scalingUsefulFitR2
}

func latencySampleWeight(point ScalingPoint) float64 {
	if point.FinalizationLatencySampleCount <= 0 {
		return 1
	}
	return float64(point.FinalizationLatencySampleCount)
}

func latencySamplesAdequate(samples []metricSample) bool {
	for _, sample := range samples {
		if sample.weight < scalingAdequateLatencySamples {
			return false
		}
	}
	return true
}

func hasSingleSampleLatency(points []ScalingPoint) bool {
	for _, point := range points {
		if point.FinalizationLatencySampleCount == 1 {
			return true
		}
	}
	return false
}

func scalingConfidence(distinctDurations int, latencyClassification string, computeClassification string, singleSampleLatency bool) string {
	if latencyClassification == "inconclusive" || computeClassification == "inconclusive" {
		return "low"
	}
	if singleSampleLatency {
		return "low"
	}
	if distinctDurations >= 5 {
		return "high"
	}
	return "medium"
}
