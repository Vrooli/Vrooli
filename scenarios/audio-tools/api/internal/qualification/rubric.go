// Package trustfloor defines the provider-neutral evidence required before an
// STT engine can be presented as stable. Functional coverage is intentionally
// independent from machine-specific latency bands.
package trustfloor

import "sort"

type DurationProfile struct {
	Name    string
	Seconds int
}

var DurationLadder = []DurationProfile{
	{Name: "30_seconds", Seconds: 30},
	{Name: "1_minute", Seconds: 60},
	{Name: "5_minutes", Seconds: 5 * 60},
	{Name: "15_minutes", Seconds: 15 * 60},
	{Name: "60_minutes", Seconds: 60 * 60},
}

var RequiredFaults = []string{
	"provider_busy", "delayed_ready", "slow_consumer", "missing_acknowledgement",
	"dropped_connection", "close_before_done", "backend_restart", "page_interruption",
	"retention_quota", "verifier_outage", "extractor_outage",
}

// Evidence is the persisted, provider-neutral input to a verdict. It contains
// no transcript or audio payload; reports carry those only under explicit
// evidence retention policy.
type Evidence struct {
	EngineID                   string
	AllIntervalsAccounted      bool
	DuplicateCommittedSegments int
	SilentTerminalOutcomes     int
	BoundedRecovery            bool
	DurationProfiles           map[string]bool
	FaultProfiles              map[string]bool
	WER                        float64
	DroppedSpanRate            float64
	HasBrowserProductPath      bool
	HasDeviceEvidence          bool
}

type Thresholds struct {
	MaxWER             float64
	MaxDroppedSpanRate float64
}

var DefaultThresholds = Thresholds{MaxWER: 0.25, MaxDroppedSpanRate: 0.0}

type Verdict struct {
	Stable  bool
	Reasons []string
}

// ReplayMeasurement is the transcript-free portion of a persisted provider
// replay report that can contribute to a trust-floor verdict. In particular,
// a duration profile is earned only by a fully real-time lane; deterministic
// measurements remain useful for WER but cannot establish long-turn survival.
type ReplayMeasurement struct {
	EngineID        string
	ModelID         string
	Strategy        string
	PolicyProfile   string
	WER             float64
	ReplayLane      string
	ClipDurationsMS []int64
	// SafetyObserved distinguishes legacy/partial reports from an observed
	// safety result. A failed observed gate must flow into the promotion
	// verdict; otherwise a long-form dropped span could disappear when reports
	// are aggregated across persisted experiments.
	SafetyObserved bool
	SafetyPassed   bool
}

const (
	QualificationIntervalAccounting = "interval_accounting"
	QualificationBoundedRecovery    = "bounded_recovery"
	QualificationFault              = "fault"
	QualificationBrowserProductPath = "browser_product_path"
	QualificationDevice             = "device"
)

// QualificationMeasurement is transcript-free evidence from a dedicated
// qualification harness. It is deliberately separate from replay metrics:
// browser, fault, recovery, and device gates cannot be inferred from WER or a
// successful duration run.
type QualificationMeasurement struct {
	EngineID                   string
	ModelID                    string
	Strategy                   string
	PolicyProfile              string
	Kind                       string
	FaultProfile               string
	Passed                     bool
	AllIntervalsAccounted      bool
	DuplicateCommittedSegments int
	SilentTerminalOutcomes     int
}

// EngineVerdict ties a trust-floor verdict to the provider whose persisted
// measurements supplied it.
type EngineVerdict struct {
	EngineID      string
	ModelID       string
	Strategy      string
	PolicyProfile string
	Verdict       Verdict
}

// EvaluateReplayMeasurements combines compatible persisted measurements for
// each engine. It deliberately does not invent recovery, fault, browser, or
// device evidence: those gates stay false until their dedicated artifacts are
// available. Callers are responsible for selecting only comparable-machine
// experiment reports before passing measurements here.
func EvaluateReplayMeasurements(measurements []ReplayMeasurement, thresholds Thresholds) []EngineVerdict {
	return EvaluatePromotionMeasurements(measurements, nil, thresholds)
}

// EvaluatePromotionMeasurements combines full-real-time replay metrics with
// dedicated, persisted qualification artifacts for the same provider cell.
// Evidence from a different strategy or policy must never satisfy a cell's
// trust floor.
func EvaluatePromotionMeasurements(measurements []ReplayMeasurement, qualification []QualificationMeasurement, thresholds Thresholds) []EngineVerdict {
	type aggregate struct {
		engineID               string
		modelID                string
		strategy               string
		policyProfile          string
		wer                    float64
		hasWER                 bool
		durationProfiles       map[string]bool
		safetyObserved         bool
		safetyPassed           bool
		allIntervalsAccounted  bool
		duplicateCommits       int
		silentTerminalOutcomes int
		boundedRecovery        bool
		faultProfiles          map[string]bool
		hasBrowserProductPath  bool
		hasDeviceEvidence      bool
		evidenceFailures       []string
	}
	aggregates := map[string]*aggregate{}
	for _, measurement := range measurements {
		if measurement.EngineID == "" {
			continue
		}
		key := measurement.EngineID + "\x00" + measurement.ModelID + "\x00" + measurement.Strategy + "\x00" + measurement.PolicyProfile
		agg := aggregates[key]
		if agg == nil {
			agg = &aggregate{
				engineID:         measurement.EngineID,
				modelID:          measurement.ModelID,
				strategy:         measurement.Strategy,
				policyProfile:    measurement.PolicyProfile,
				durationProfiles: make(map[string]bool),
				safetyPassed:     true,
				faultProfiles:    make(map[string]bool),
			}
			aggregates[key] = agg
		}
		if !agg.hasWER || measurement.WER < agg.wer {
			agg.wer, agg.hasWER = measurement.WER, true
		}
		if measurement.SafetyObserved {
			agg.safetyObserved = true
			agg.safetyPassed = agg.safetyPassed && measurement.SafetyPassed
		}
		if measurement.ReplayLane != "realtime" {
			continue
		}
		for _, durationMS := range measurement.ClipDurationsMS {
			for _, profile := range DurationLadder {
				if durationMS >= int64(profile.Seconds)*1000 {
					agg.durationProfiles[profile.Name] = true
				}
			}
		}
	}
	for _, item := range qualification {
		key := item.EngineID + "\x00" + item.ModelID + "\x00" + item.Strategy + "\x00" + item.PolicyProfile
		agg := aggregates[key]
		if agg == nil {
			continue
		}
		// Interval accounting is the explicit proof of all three delivery
		// invariants. Do not infer zero duplicates or silent terminals merely
		// because a replay report exists.
		if item.Kind == QualificationIntervalAccounting {
			agg.allIntervalsAccounted = agg.allIntervalsAccounted || (item.Passed && item.AllIntervalsAccounted)
			agg.duplicateCommits += item.DuplicateCommittedSegments
			agg.silentTerminalOutcomes += item.SilentTerminalOutcomes
		}
		if !item.Passed {
			name := item.Kind
			if item.Kind == QualificationFault && item.FaultProfile != "" {
				name += ":" + item.FaultProfile
			}
			agg.evidenceFailures = append(agg.evidenceFailures, "qualification evidence failed: "+name)
			continue
		}
		switch item.Kind {
		case QualificationBoundedRecovery:
			agg.boundedRecovery = true
		case QualificationFault:
			if item.FaultProfile != "" {
				agg.faultProfiles[item.FaultProfile] = true
			}
		case QualificationBrowserProductPath:
			agg.hasBrowserProductPath = true
		case QualificationDevice:
			agg.hasDeviceEvidence = true
		}
	}

	keys := make([]string, 0, len(aggregates))
	for key := range aggregates {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	verdicts := make([]EngineVerdict, 0, len(keys))
	for _, key := range keys {
		agg := aggregates[key]
		droppedSpanRate := 0.0
		if agg.safetyObserved && !agg.safetyPassed {
			// The trust floor is zero tolerance for threshold-sized dropped
			// spans and committed-text retractions. The report does not
			// preserve a population denominator, so express an observed
			// failure as the failing rate rather than inventing one.
			droppedSpanRate = 1
		}
		verdict := Evaluate(Evidence{
			EngineID:                   agg.engineID,
			AllIntervalsAccounted:      agg.allIntervalsAccounted,
			DuplicateCommittedSegments: agg.duplicateCommits,
			SilentTerminalOutcomes:     agg.silentTerminalOutcomes,
			BoundedRecovery:            agg.boundedRecovery,
			DurationProfiles:           agg.durationProfiles,
			FaultProfiles:              agg.faultProfiles,
			WER:                        agg.wer,
			DroppedSpanRate:            droppedSpanRate,
			HasBrowserProductPath:      agg.hasBrowserProductPath,
			HasDeviceEvidence:          agg.hasDeviceEvidence,
		}, thresholds)
		verdict.Reasons = append(verdict.Reasons, agg.evidenceFailures...)
		verdict.Stable = len(verdict.Reasons) == 0
		verdicts = append(verdicts, EngineVerdict{
			EngineID:      agg.engineID,
			ModelID:       agg.modelID,
			Strategy:      agg.strategy,
			PolicyProfile: agg.policyProfile,
			Verdict:       verdict,
		})
	}
	return verdicts
}

func Evaluate(e Evidence, thresholds Thresholds) Verdict {
	v := Verdict{}
	if e.EngineID == "" {
		v.Reasons = append(v.Reasons, "engine identity is missing")
	}
	if !e.AllIntervalsAccounted {
		v.Reasons = append(v.Reasons, "audio interval accounting is incomplete")
	}
	if e.DuplicateCommittedSegments != 0 {
		v.Reasons = append(v.Reasons, "duplicate committed segments were observed")
	}
	if e.SilentTerminalOutcomes != 0 {
		v.Reasons = append(v.Reasons, "silent terminal outcomes were observed")
	}
	if !e.BoundedRecovery {
		v.Reasons = append(v.Reasons, "bounded recovery was not demonstrated")
	}
	for _, profile := range DurationLadder {
		if !e.DurationProfiles[profile.Name] {
			v.Reasons = append(v.Reasons, "missing duration profile: "+profile.Name)
		}
	}
	for _, fault := range RequiredFaults {
		if !e.FaultProfiles[fault] {
			v.Reasons = append(v.Reasons, "missing fault profile: "+fault)
		}
	}
	if e.WER > thresholds.MaxWER {
		v.Reasons = append(v.Reasons, "WER exceeds trust threshold")
	}
	if e.DroppedSpanRate > thresholds.MaxDroppedSpanRate {
		v.Reasons = append(v.Reasons, "dropped-span rate exceeds trust threshold")
	}
	if !e.HasBrowserProductPath {
		v.Reasons = append(v.Reasons, "browser product-path evidence is missing")
	}
	if !e.HasDeviceEvidence {
		v.Reasons = append(v.Reasons, "manual device evidence is missing")
	}
	v.Stable = len(v.Reasons) == 0
	return v
}
