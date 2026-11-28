// Package trustfloor defines the provider-neutral evidence required before an
// STT engine can be presented as stable. Functional coverage is intentionally
// independent from machine-specific latency bands.
package trustfloor

import (
	"math"
	"sort"
)

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
	QualificationIntervalAccounting  = "interval_accounting"
	QualificationBoundedRecovery     = "bounded_recovery"
	QualificationFault               = "fault"
	QualificationBrowserProductPath  = "browser_product_path"
	QualificationDevice              = "device"
	QualificationCorpusQuality       = "corpus_quality"
	QualificationAcceptanceComposite = "acceptance_composite"
	QualificationModelMemory         = "model_memory"
)

// QualificationMeasurement is transcript-free evidence from a dedicated
// qualification harness. It is deliberately separate from replay metrics:
// browser, fault, recovery, and device gates cannot be inferred from WER or a
// successful duration run.
type QualificationMeasurement struct {
	EngineID      string
	ModelID       string
	Strategy      string
	PolicyProfile string
	Kind          string
	FaultProfile  string
	// ArtifactRef binds related rows to the same observed product run. A fault
	// row without a browser-path row from that artifact is diagnostic only.
	ArtifactRef                string
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
	aggregates := make(map[string]*promotionAggregate)
	collectReplayMeasurements(aggregates, measurements)
	collectQualificationMeasurements(aggregates, qualification)
	linkFaultArtifactsToBrowserRuns(aggregates)
	return promotionVerdicts(aggregates, thresholds)
}

type promotionAggregate struct {
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
	browserArtifacts       map[string]bool
	faultArtifacts         map[string]map[string]bool
}

func collectReplayMeasurements(aggregates map[string]*promotionAggregate, measurements []ReplayMeasurement) {
	for _, measurement := range measurements {
		if measurement.EngineID == "" {
			continue
		}
		agg := promotionAggregateFor(aggregates, measurement.EngineID, measurement.ModelID, measurement.Strategy, measurement.PolicyProfile)
		if math.IsNaN(measurement.WER) || math.IsInf(measurement.WER, 0) || measurement.WER < 0 {
			agg.evidenceFailures = append(agg.evidenceFailures, "invalid WER observation")
		} else {
			// A stable verdict must satisfy the quality floor for every compatible
			// observation. Keeping only the best WER lets a good run hide a failed
			// repeat or an earlier same-cell regression, while safety evidence is
			// already aggregated fail-closed below.
			if !agg.hasWER || measurement.WER > agg.wer {
				agg.wer, agg.hasWER = measurement.WER, true
			}
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
}

func promotionAggregateFor(aggregates map[string]*promotionAggregate, engineID, modelID, strategy, policyProfile string) *promotionAggregate {
	key := promotionCellKey(engineID, modelID, strategy, policyProfile)
	agg := aggregates[key]
	if agg == nil {
		agg = &promotionAggregate{
			engineID: engineID, modelID: modelID, strategy: strategy, policyProfile: policyProfile,
			durationProfiles: make(map[string]bool), safetyPassed: true,
			faultProfiles: make(map[string]bool), browserArtifacts: make(map[string]bool),
			faultArtifacts: make(map[string]map[string]bool),
		}
		aggregates[key] = agg
	}
	return agg
}

func promotionCellKey(engineID, modelID, strategy, policyProfile string) string {
	return engineID + "\x00" + modelID + "\x00" + strategy + "\x00" + policyProfile
}

func collectQualificationMeasurements(aggregates map[string]*promotionAggregate, qualification []QualificationMeasurement) {
	for _, item := range qualification {
		agg := aggregates[promotionCellKey(item.EngineID, item.ModelID, item.Strategy, item.PolicyProfile)]
		if agg != nil {
			applyQualificationMeasurement(agg, item)
		}
	}
}

func applyQualificationMeasurement(agg *promotionAggregate, item QualificationMeasurement) {
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
		return
	}
	switch item.Kind {
	case QualificationBoundedRecovery:
		agg.boundedRecovery = true
	case QualificationFault:
		if item.FaultProfile != "" && item.ArtifactRef != "" {
			if agg.faultArtifacts[item.FaultProfile] == nil {
				agg.faultArtifacts[item.FaultProfile] = make(map[string]bool)
			}
			agg.faultArtifacts[item.FaultProfile][item.ArtifactRef] = true
		}
	case QualificationBrowserProductPath:
		if item.ArtifactRef != "" {
			agg.browserArtifacts[item.ArtifactRef] = true
			agg.hasBrowserProductPath = true
		}
	case QualificationDevice:
		agg.hasDeviceEvidence = true
	}
}

func linkFaultArtifactsToBrowserRuns(aggregates map[string]*promotionAggregate) {
	// A pass from a test injector or an older browser run cannot satisfy a
	// fault cell by itself. Require its product-path proof to refer to the same
	// persisted run artifact.
	for _, agg := range aggregates {
		for fault, artifacts := range agg.faultArtifacts {
			for artifact := range artifacts {
				if agg.browserArtifacts[artifact] {
					agg.faultProfiles[fault] = true
					break
				}
			}
		}
	}
}

func promotionVerdicts(aggregates map[string]*promotionAggregate, thresholds Thresholds) []EngineVerdict {
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
	appendThresholdReasons(&v, thresholds)
	appendDeliveryEvidenceReasons(&v, e)
	appendQualityReasons(&v, e, thresholds)
	appendProductPathReasons(&v, e)
	v.Stable = len(v.Reasons) == 0
	return v
}

func appendThresholdReasons(v *Verdict, thresholds Thresholds) {
	if math.IsNaN(thresholds.MaxWER) || math.IsInf(thresholds.MaxWER, 0) || thresholds.MaxWER < 0 || math.IsNaN(thresholds.MaxDroppedSpanRate) || math.IsInf(thresholds.MaxDroppedSpanRate, 0) || thresholds.MaxDroppedSpanRate < 0 {
		v.Reasons = append(v.Reasons, "qualification thresholds are invalid")
	}
}

func appendDeliveryEvidenceReasons(v *Verdict, e Evidence) {
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
}

func appendQualityReasons(v *Verdict, e Evidence, thresholds Thresholds) {
	if math.IsNaN(e.WER) || math.IsInf(e.WER, 0) || e.WER < 0 {
		v.Reasons = append(v.Reasons, "WER is invalid")
	} else if e.WER > thresholds.MaxWER {
		v.Reasons = append(v.Reasons, "WER exceeds trust threshold")
	}
	if math.IsNaN(e.DroppedSpanRate) || math.IsInf(e.DroppedSpanRate, 0) || e.DroppedSpanRate < 0 {
		v.Reasons = append(v.Reasons, "dropped-span rate is invalid")
	} else if e.DroppedSpanRate > thresholds.MaxDroppedSpanRate {
		v.Reasons = append(v.Reasons, "dropped-span rate exceeds trust threshold")
	}
}

func appendProductPathReasons(v *Verdict, e Evidence) {
	if !e.HasBrowserProductPath {
		v.Reasons = append(v.Reasons, "browser product-path evidence is missing")
	}
	if !e.HasDeviceEvidence {
		v.Reasons = append(v.Reasons, "automated operating-system device evidence is missing")
	}
}
