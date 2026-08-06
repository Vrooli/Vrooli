package evidence

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"scenario-to-desktop-api/procmetrics"
	"scenario-to-desktop-api/smoketest"
)

const PerformanceBaselineSchemaVersion = 1

type MeasurementClass string

const (
	MeasurementCold MeasurementClass = "cold"
	MeasurementWarm MeasurementClass = "warm"
)

// PerformanceSample is the bounded, identity-bearing input to a baseline.
// Raw traces remain producer artifacts; a baseline stores only their stable
// phase and role projections.
type PerformanceSample struct {
	RunID           string                    `json:"run_id"`
	Class           MeasurementClass          `json:"class"`
	HostFingerprint string                    `json:"host_fingerprint"`
	ArtifactDigest  string                    `json:"artifact_digest"`
	Display         string                    `json:"display"`
	DeploymentMode  string                    `json:"deployment_mode"`
	ProfilerMode    string                    `json:"profiler_mode"`
	Phases          []PhaseDuration           `json:"phases"`
	Roles           []procmetrics.RoleSummary `json:"roles,omitempty"`
}

type PerformanceBaseline struct {
	SchemaVersion int                     `json:"schema_version"`
	Name          string                  `json:"name"`
	CreatedAt     time.Time               `json:"created_at"`
	Samples       []PerformanceSample     `json:"samples"`
	Cold          map[string]Distribution `json:"cold"`
	Warm          map[string]Distribution `json:"warm"`
}

type PerformanceComparison struct {
	Status         string                  `json:"status"`
	Reason         string                  `json:"reason,omitempty"`
	Baseline       map[string]Distribution `json:"baseline,omitempty"`
	Current        map[string]Distribution `json:"current,omitempty"`
	SlowestSegment string                  `json:"slowest_segment,omitempty"`
	SlowestRole    string                  `json:"slowest_role,omitempty"`
	Regressions    []string                `json:"regressions,omitempty"`
}

const (
	ComparisonPass          = "pass"
	ComparisonRegression    = "regression"
	ComparisonNonComparable = "non-comparable"
)

// BuildPerformanceBaseline requires enough repeated samples to make a
// cold/warm distribution meaningful. Identity checks are deliberately strict;
// a host or artifact change must not silently become a comparison pass.
func BuildPerformanceBaseline(name string, samples []PerformanceSample) (PerformanceBaseline, error) {
	if strings.TrimSpace(name) == "" {
		return PerformanceBaseline{}, fmt.Errorf("baseline name is required")
	}
	if len(samples) < 10 {
		return PerformanceBaseline{}, fmt.Errorf("at least five cold and five warm samples are required")
	}
	var cold, warm int
	for _, sample := range samples {
		if strings.TrimSpace(sample.RunID) == "" || strings.TrimSpace(sample.HostFingerprint) == "" || strings.TrimSpace(sample.ArtifactDigest) == "" || strings.TrimSpace(sample.Display) == "" {
			return PerformanceBaseline{}, fmt.Errorf("sample %q is missing identity", sample.RunID)
		}
		switch sample.Class {
		case MeasurementCold:
			cold++
		case MeasurementWarm:
			warm++
		default:
			return PerformanceBaseline{}, fmt.Errorf("sample %q has unsupported class %q", sample.RunID, sample.Class)
		}
	}
	if cold < 5 || warm < 5 {
		return PerformanceBaseline{}, fmt.Errorf("at least five cold and five warm samples are required (cold=%d warm=%d)", cold, warm)
	}
	baseline := PerformanceBaseline{SchemaVersion: PerformanceBaselineSchemaVersion, Name: name, CreatedAt: time.Now().UTC(), Samples: append([]PerformanceSample(nil), samples...), Cold: map[string]Distribution{}, Warm: map[string]Distribution{}}
	for _, class := range []MeasurementClass{MeasurementCold, MeasurementWarm} {
		distributions, err := summarizeSamples(samples, class)
		if err != nil {
			return PerformanceBaseline{}, err
		}
		if class == MeasurementCold {
			baseline.Cold = distributions
		} else {
			baseline.Warm = distributions
		}
	}
	return baseline, nil
}

func summarizeSamples(samples []PerformanceSample, class MeasurementClass) (map[string]Distribution, error) {
	values := map[string][]time.Duration{}
	for _, sample := range samples {
		if sample.Class != class {
			continue
		}
		for _, phase := range sample.Phases {
			if phase.Available {
				values[phase.Name] = append(values[phase.Name], time.Duration(phase.DurationMs)*time.Millisecond)
			}
		}
	}
	result := make(map[string]Distribution, len(values))
	for name, durations := range values {
		distribution, err := SummarizeDurations(durations)
		if err != nil {
			return nil, fmt.Errorf("summarize %s %s: %w", class, name, err)
		}
		result[name] = distribution
	}
	return result, nil
}

// ComparePerformanceBaseline is fail-closed: missing phase data or identity
// drift produces non-comparable, never a passing result.
func ComparePerformanceBaseline(baseline PerformanceBaseline, current []PerformanceSample, class MeasurementClass, regressionTolerance float64) PerformanceComparison {
	if baseline.SchemaVersion != PerformanceBaselineSchemaVersion {
		return PerformanceComparison{Status: ComparisonNonComparable, Reason: "unsupported baseline schema"}
	}
	if regressionTolerance < 0 {
		return PerformanceComparison{Status: ComparisonNonComparable, Reason: "negative regression tolerance"}
	}
	if len(current) == 0 {
		return PerformanceComparison{Status: ComparisonNonComparable, Reason: "current sample set is empty"}
	}
	baseSamples := baseline.Samples
	for _, sample := range current {
		for _, reference := range baseSamples {
			if reference.Class == class && (reference.HostFingerprint != sample.HostFingerprint || reference.ArtifactDigest != sample.ArtifactDigest || reference.Display != sample.Display || reference.DeploymentMode != sample.DeploymentMode) {
				return PerformanceComparison{Status: ComparisonNonComparable, Reason: "host, artifact, display, or deployment identity changed"}
			}
		}
	}
	base := baseline.Cold
	if class == MeasurementWarm {
		base = baseline.Warm
	}
	currentSummary, err := summarizeSamples(current, class)
	if err != nil || len(base) == 0 || len(currentSummary) == 0 {
		reason := "no comparable phase data"
		if err != nil {
			reason = err.Error()
		}
		return PerformanceComparison{Status: ComparisonNonComparable, Reason: reason}
	}
	comparison := PerformanceComparison{Status: ComparisonPass, Baseline: base, Current: currentSummary}
	var slowestValue int64
	for name, currentDistribution := range currentSummary {
		reference, ok := base[name]
		if !ok {
			continue
		}
		if currentDistribution.P95Ms > slowestValue {
			slowestValue = currentDistribution.P95Ms
			comparison.SlowestSegment = name
		}
		limit := float64(reference.P95Ms) * (1 + regressionTolerance)
		if float64(currentDistribution.P95Ms) > limit {
			comparison.Regressions = append(comparison.Regressions, name)
		}
	}
	if len(comparison.Regressions) > 0 {
		sort.Strings(comparison.Regressions)
		comparison.Status = ComparisonRegression
		comparison.Reason = "current p95 exceeded the comparable baseline"
	}
	comparison.SlowestRole = slowestRole(current)
	return comparison
}

func slowestRole(samples []PerformanceSample) string {
	var name string
	var peak float64
	for _, sample := range samples {
		for _, role := range sample.Roles {
			if !role.Available || role.Unsupported {
				continue
			}
			value := role.PeakCPU
			if value == 0 {
				value = float64(role.PeakRSSBytes)
			}
			if value > peak {
				peak, name = value, string(role.Role)
			}
		}
	}
	return name
}

// PhaseDuration is a bounded, review-friendly projection of a launch trace.
// The complete event stream remains available through the trace artifact.
type PhaseDuration struct {
	Name       string `json:"name"`
	Available  bool   `json:"available"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// LaunchPhaseDurations derives named segments without inferring missing
// events as zero. This is the stable input to later cold/warm aggregation.
func LaunchPhaseDurations(trace smoketest.LaunchTrace) []PhaseDuration {
	segments := []struct {
		name       string
		start, end smoketest.LaunchEventName
	}{
		{"process_to_splash_first_paint", smoketest.EventDemoSpawn, smoketest.EventSplashFirstPaint},
		{"splash_to_app_ready", smoketest.EventSplashFirstPaint, smoketest.EventAppReady},
		{"runtime_spawn_to_ready", smoketest.EventRuntimeSpawned, smoketest.EventRuntimeReady},
		{"server_to_app_ready", smoketest.EventServerReady, smoketest.EventAppReady},
	}
	result := make([]PhaseDuration, 0, len(segments))
	for _, segment := range segments {
		duration, err := trace.Segment(segment.start, segment.end)
		if err != nil {
			result = append(result, PhaseDuration{Name: segment.name, Reason: err.Error()})
			continue
		}
		result = append(result, PhaseDuration{Name: segment.name, Available: true, DurationMs: duration.Milliseconds()})
	}
	return result
}

// Distribution is the deterministic summary used by baseline comparisons.
type Distribution struct {
	Count  int     `json:"count"`
	P50Ms  int64   `json:"p50_ms,omitempty"`
	P95Ms  int64   `json:"p95_ms,omitempty"`
	MinMs  int64   `json:"min_ms,omitempty"`
	MaxMs  int64   `json:"max_ms,omitempty"`
	Spread int64   `json:"spread_ms,omitempty"`
	MeanMs float64 `json:"mean_ms,omitempty"`
}

func SummarizeDurations(values []time.Duration) (Distribution, error) {
	if len(values) == 0 {
		return Distribution{}, fmt.Errorf("at least one duration is required")
	}
	ms := make([]int64, 0, len(values))
	var total int64
	for _, value := range values {
		if value < 0 {
			return Distribution{}, fmt.Errorf("duration cannot be negative")
		}
		valueMs := value.Milliseconds()
		ms = append(ms, valueMs)
		total += valueMs
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i] < ms[j] })
	percentile := func(percent int) int64 {
		index := (len(ms)*percent + 99) / 100
		if index < 1 {
			index = 1
		}
		return ms[index-1]
	}
	return Distribution{
		Count: len(ms), P50Ms: percentile(50), P95Ms: percentile(95),
		MinMs: ms[0], MaxMs: ms[len(ms)-1], Spread: ms[len(ms)-1] - ms[0],
		MeanMs: float64(total) / float64(len(ms)),
	}, nil
}
