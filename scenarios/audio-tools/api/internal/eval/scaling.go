package eval

import (
	"fmt"
	"math"
	"strings"
)

const minScalingDistinctDurations = 3

const (
	scalingFlatRelativeRange       = 0.12
	scalingSuperlinearR2Margin     = 0.02
	scalingUsefulFitR2             = 0.70
	scalingNearLinearR2Margin      = 0.01
	scalingMinimumPositiveDuration = 0.001
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

	latencyFit, latencyClassification, latencyReason := classifyScaling(points, metricFinalizationP95)
	computeFit, computeClassification, computeReason := classifyComputeScaling(points)
	analysis.LatencyFit = latencyFit
	analysis.ComputeFit = computeFit
	analysis.LatencyClassification = latencyClassification
	analysis.ComputeClassification = computeClassification
	analysis.Confidence = scalingConfidence(len(distinctDurations), latencyClassification, computeClassification)
	analysis.Reasons = append(analysis.Reasons, latencyReason, computeReason)
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
	metricProviderLatency scalingMetric = "provider_latency_ms"
	metricWhisperCalls    scalingMetric = "whisper_calls"
	metricWhisperAudioSec scalingMetric = "whisper_audio_seconds"
	metricRTF             scalingMetric = "rtf"
)

type metricSample struct {
	durationSeconds float64
	value           float64
}

type candidateFit struct {
	ScalingModelFit
	transform func(float64) float64
}

func classifyScaling(points []ScalingPoint, metric scalingMetric) (ScalingModelFit, string, string) {
	samples := scalingSamples(points, metric)
	distinctDurations := distinctSampleDurations(samples)
	if distinctDurations < minScalingDistinctDurations {
		return ScalingModelFit{Metric: string(metric), Model: "none", SampleCount: len(samples), Reason: "insufficient distinct positive durations"}, "inconclusive", fmt.Sprintf("%s scaling is inconclusive because fewer than %d distinct positive durations were measured.", metric, minScalingDistinctDurations)
	}
	if relativeValueRange(samples) <= scalingFlatRelativeRange {
		fit := constantFit(samples, string(metric), "values stayed within the flat threshold")
		return fit, "flat", fmt.Sprintf("%s stayed effectively flat across the measured durations.", metric)
	}

	candidates := []candidateFit{
		fitTransformed(samples, string(metric), "linear", func(x float64) float64 { return x }),
		fitTransformed(samples, string(metric), "n_log_n", func(x float64) float64 {
			if x <= 1 {
				return x
			}
			return x * math.Log(x)
		}),
		fitTransformed(samples, string(metric), "quadratic", func(x float64) float64 { return x * x }),
	}
	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.RSquared > best.RSquared {
			best = candidate
		}
	}
	linear := candidates[0]
	if best.RSquared < scalingUsefulFitR2 {
		best.Reason = "no model cleared the useful-fit threshold"
		return best.ScalingModelFit, "inconclusive", fmt.Sprintf("%s scaling is inconclusive; the best %s fit had R^2 %.2f.", metric, best.Model, best.RSquared)
	}
	if (best.Model == "quadratic" || best.Model == "n_log_n") && best.RSquared-linear.RSquared >= scalingSuperlinearR2Margin {
		best.Reason = "superlinear model fit materially better than the linear model"
		return best.ScalingModelFit, "superlinear", fmt.Sprintf("%s fit %s better than linear over the measured sweep.", metric, best.Model)
	}
	if linear.RSquared >= best.RSquared-scalingNearLinearR2Margin {
		linear.Reason = "linear fit was competitive with the best model"
		return linear.ScalingModelFit, "linear", fmt.Sprintf("%s scaled approximately linearly over the measured sweep.", metric)
	}

	best.Reason = "best model was not materially superlinear"
	return best.ScalingModelFit, "linear", fmt.Sprintf("%s scaled close enough to linear for this sweep.", metric)
}

func classifyComputeScaling(points []ScalingPoint) (ScalingModelFit, string, string) {
	metrics := []scalingMetric{
		metricProviderLatency,
		metricWhisperCalls,
		metricWhisperAudioSec,
		metricRTF,
	}
	var chosenFit ScalingModelFit
	chosenClassification := "inconclusive"
	reasons := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		fit, classification, reason := classifyScaling(points, metric)
		reasons = append(reasons, reason)
		if chosenFit == (ScalingModelFit{}) || scalingClassificationRisk(classification) > scalingClassificationRisk(chosenClassification) {
			chosenFit = fit
			chosenClassification = classification
		}
	}
	if chosenFit != (ScalingModelFit{}) {
		chosenFit.Reason = fmt.Sprintf("%s; selected as the highest-risk compute metric across provider latency, calls, processed audio seconds, and RTF", chosenFit.Reason)
	}
	return chosenFit, chosenClassification, strings.Join(reasons, " ")
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

func scalingSamples(points []ScalingPoint, metric scalingMetric) []metricSample {
	out := make([]metricSample, 0, len(points))
	for _, point := range points {
		durationSeconds := float64(point.RealizedDurationMs) / 1000
		if durationSeconds < scalingMinimumPositiveDuration {
			continue
		}
		var value float64
		switch metric {
		case metricFinalizationP95:
			if point.FinalizationLatencySampleCount == 0 {
				continue
			}
			value = point.FinalizationLatencyP95Ms
		case metricProviderLatency:
			value = point.ProviderLatencyMs
		case metricWhisperCalls:
			value = float64(point.WhisperCalls)
		case metricWhisperAudioSec:
			value = point.WhisperAudioSeconds
		case metricRTF:
			value = point.RTF
		}
		if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		out = append(out, metricSample{durationSeconds: durationSeconds, value: value})
	}
	return out
}

func relativeValueRange(samples []metricSample) float64 {
	if len(samples) == 0 {
		return 0
	}
	minValue, maxValue := samples[0].value, samples[0].value
	for _, sample := range samples[1:] {
		minValue = math.Min(minValue, sample.value)
		maxValue = math.Max(maxValue, sample.value)
	}
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

func constantFit(samples []metricSample, metric string, reason string) ScalingModelFit {
	var sum float64
	for _, sample := range samples {
		sum += sample.value
	}
	mean := sum / float64(len(samples))
	return ScalingModelFit{
		Metric:      metric,
		Model:       "constant",
		Intercept:   mean,
		RSquared:    1,
		SampleCount: len(samples),
		Reason:      reason,
	}
}

func fitTransformed(samples []metricSample, metric string, model string, transform func(float64) float64) candidateFit {
	var sumX, sumY float64
	xs := make([]float64, 0, len(samples))
	for _, sample := range samples {
		x := transform(sample.durationSeconds)
		xs = append(xs, x)
		sumX += x
		sumY += sample.value
	}
	meanX := sumX / float64(len(samples))
	meanY := sumY / float64(len(samples))
	var numerator, denominator float64
	for i, sample := range samples {
		dx := xs[i] - meanX
		numerator += dx * (sample.value - meanY)
		denominator += dx * dx
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
		residual += delta * delta
		centered := sample.value - meanY
		total += centered * centered
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
			Metric:         metric,
			Model:          model,
			SlopePerSecond: slope,
			Intercept:      intercept,
			RSquared:       r2,
			SampleCount:    len(samples),
		},
		transform: transform,
	}
}

func scalingConfidence(distinctDurations int, latencyClassification string, computeClassification string) string {
	if latencyClassification == "inconclusive" || computeClassification == "inconclusive" {
		return "low"
	}
	if distinctDurations >= 5 {
		return "high"
	}
	return "medium"
}
