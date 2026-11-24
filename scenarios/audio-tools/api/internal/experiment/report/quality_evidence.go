//go:build audio_lab

package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"audio-tools/internal/acceptance"
	intcorpus "audio-tools/internal/corpus"
	inteval "audio-tools/internal/eval"
	intexp "audio-tools/internal/experiment"
	"audio-tools/internal/modelcatalog"
)

const (
	maxQualityCleanWER       = 0.10
	maxQualityNoisyWER       = 0.20
	qualityMeasurementSchema = "corpus-quality-subgroups-v1"
)

var requiredQualityAccents = []string{"uk", "us", "south_african"}

type qualityCounts struct{ edits, words int }

// QualityEvidenceForReport turns a completed report into one persisted
// transcript-free quality receipt per provider cell. It is deliberately
// fail-closed: the producer emits a failed receipt when the corpus or report
// is incomplete, and never substitutes a bundled smoke fixture for a cohort.
func QualityEvidenceForReport(clips []intcorpus.Clip, report inteval.EvalReport, build string) []intexp.QualificationEvidence {
	// The corpus service is global, but a report is scoped to the recipe's
	// selected clips. Assess the realized cohort represented by the report;
	// otherwise unrelated legacy/dev rows can poison a held-out receipt even
	// though they were neither requested nor measured. Per-cell completeness
	// below still fails a provider cell that omits any clip in this union.
	cohortClips := measuredCohortClips(clips, report)
	cohort := intcorpus.AssessQualityCohort(cohortClips, acceptance.Cohort)
	out := make([]intexp.QualificationEvidence, 0, len(report.PerStrategy))
	for _, cell := range report.PerStrategy {
		passed, metrics, reasons := assessQualityCell(cohortClips, cell, report.QualityMeasured, cohort, build)
		capability := "unknown"
		canonicalModelID := cell.ModelID
		if model, ok := modelcatalog.Default().ByEngineModelID(cell.EngineID, cell.ModelID); !ok {
			passed = false
			reasons = append(reasons, "model id is not declared in the model catalog")
		} else {
			canonicalModelID = model.ID
			capability = string(model.Capability)
			if model.Capability != modelcatalog.CapabilitySTT {
				passed = false
				reasons = append(reasons, "corpus quality receipt requires an STT model")
			}
		}
		// Explicit provider-cell reports use an internal CellID to keep
		// otherwise-colliding rows distinct. The acceptance envelope, however,
		// joins on the canonical strategy declared by the cell. Do not leak the
		// internal lane-qualified identifier into the cross-run receipt key.
		strategy := string(cell.Strategy)
		if strings.TrimSpace(string(cell.BaseStrategy)) != "" {
			strategy = string(cell.BaseStrategy)
		}
		claims := map[string]any{
			"corpus-quality": map[string]any{
				"in_band":                  passed,
				"negative_controls_passed": passed,
			},
		}
		machine, err := acceptance.MachineJSONForCell(build, qualityDeviceProfile(cell.ReplayLane), acceptance.Cohort, acceptance.Policy, cell.EngineID, canonicalModelID, capability, strategy, claims)
		if err != nil {
			machine, _ = json.Marshal(map[string]any{
				"target_revision": acceptance.TargetRevision,
				"method":          acceptance.Method,
				"receipt_error":   err.Error(),
				"acceptance":      claims,
			})
			passed = false
			reasons = append(reasons, err.Error())
		}
		payload := map[string]any{
			"schema_version": qualityMeasurementSchema,
			"metrics":        metrics,
			"provider_cell": map[string]string{
				"engine_id": cell.EngineID, "model_id": canonicalModelID, "capability": capability,
				"strategy": strategy, "policy_profile": cell.PolicyProfile,
			},
			"corpus": map[string]any{
				"revision": cohort.Revision, "eligible": cohort.Eligible,
				"clip_count": cohort.ClipCount, "reference_words": cohort.ReferenceWords,
				"speaker_count": cohort.SpeakerCount, "clean_count": cohort.CleanCount,
				"noisy_20db_count": cohort.Noisy20dBCount, "noisy_10db_count": cohort.Noisy10dBCount,
			},
			"quality": metrics, "passed": passed, "reasons": reasons,
		}
		// Preserve the metrics alongside the exact target-aware envelope so the
		// receipt remains auditable without retaining transcript text.
		var envelope map[string]any
		if json.Unmarshal(machine, &envelope) == nil {
			envelope["quality_measurement"] = payload
			machine, _ = json.Marshal(envelope)
		}
		out = append(out, intexp.QualificationEvidence{
			EngineID: cell.EngineID, ModelID: canonicalModelID, Strategy: strategy,
			PolicyProfile: cell.PolicyProfile, Kind: "corpus_quality", Passed: passed,
			ArtifactRef: fmt.Sprintf("experiment-quality:%s:%s:%s", safePart(cell.EngineID), safePart(canonicalModelID), safePart(string(cell.Strategy))),
			Notes:       "server-owned held-out corpus quality qualification; pass requires provenance, cohort coverage, and subgroup WER thresholds",
			MachineJSON: machine,
		})
	}
	return out
}

func measuredCohortClips(clips []intcorpus.Clip, report inteval.EvalReport) []intcorpus.Clip {
	byID := make(map[string]intcorpus.Clip, len(clips))
	for _, clip := range clips {
		byID[clip.ID] = clip
	}
	ids := make(map[string]struct{})
	for _, cell := range report.PerStrategy {
		for _, measured := range cell.PerClip {
			ids[measured.ClipID] = struct{}{}
		}
	}
	out := make([]intcorpus.Clip, 0, len(ids))
	for _, clip := range clips {
		if _, ok := ids[clip.ID]; ok {
			out = append(out, clip)
		}
	}
	return out
}

type qualityMetric struct {
	Name      string   `json:"name"`
	Condition string   `json:"condition"`
	Accent    string   `json:"accent,omitempty"`
	Available bool     `json:"available"`
	SNRDB     float64  `json:"snr_db"`
	WER       *float64 `json:"wer,omitempty"`
	RefWords  int      `json:"ref_words"`
	MaxWER    float64  `json:"max_wer"`
}

func assessQualityCell(clips []intcorpus.Clip, cell inteval.StrategyReport, measured bool, cohort intcorpus.QualityCohortReport, build string) (bool, []qualityMetric, []string) {
	reasons := append([]string(nil), cohort.Reasons...)
	reasons = qualityCellPrerequisiteReasons(reasons, cell, measured, build)
	byID := make(map[string]intcorpus.Clip, len(clips))
	for _, clip := range clips {
		byID[clip.ID] = clip
	}
	seen := make(map[string]bool, len(cell.PerClip))
	groups := map[string]qualityCounts{}
	for _, clip := range cell.PerClip {
		meta, ok := byID[clip.ClipID]
		if !ok {
			reasons = append(reasons, fmt.Sprintf("report contains unknown clip %q", clip.ClipID))
			continue
		}
		if seen[clip.ClipID] {
			reasons = append(reasons, fmt.Sprintf("report contains duplicate clip %q", clip.ClipID))
			continue
		}
		seen[clip.ClipID] = true
		condition := qualityConditionGroup(meta)
		g := groups[condition]
		g.edits += clip.WER.EditCounts.Total()
		g.words += clip.WER.RefWords
		groups[condition] = g
		accent := qualityAccent(meta.Tags)
		if accent == "" {
			reasons = append(reasons, fmt.Sprintf("clip %q has no recognized accent group", meta.ID))
			continue
		}
		key := "accent_noise:" + accent + ":" + strings.TrimPrefix(condition, "noisy_")
		g = groups[key]
		g.edits += clip.WER.EditCounts.Total()
		g.words += clip.WER.RefWords
		groups[key] = g
	}
	if len(seen) != len(clips) {
		reasons = append(reasons, fmt.Sprintf("report covers %d of %d corpus clips", len(seen), len(clips)))
	}
	metrics := make([]qualityMetric, 0, 12)
	for _, spec := range []struct {
		name string
		max  float64
	}{
		{"clean", maxQualityCleanWER}, {"noisy_20db", maxQualityNoisyWER}, {"noisy_10db", maxQualityNoisyWER},
	} {
		appendQualityMetric(&metrics, &reasons, groups, spec.name, "", spec.name, spec.name, spec.max)
	}
	for _, accent := range requiredQualityAccents {
		for _, spec := range []struct {
			condition string
			max       float64
		}{{"clean", maxQualityCleanWER}, {"noisy_20db", maxQualityNoisyWER}, {"noisy_10db", maxQualityNoisyWER}} {
			name := "accent_noise:" + accent + ":" + strings.TrimPrefix(spec.condition, "noisy_")
			appendQualityMetric(&metrics, &reasons, groups, name, accent, name, spec.condition, spec.max)
		}
	}
	return len(reasons) == 0, metrics, reasons
}

func appendQualityMetric(metrics *[]qualityMetric, reasons *[]string, groups map[string]qualityCounts, name, accent, key, condition string, max float64) {
	g, ok := groups[key]
	metric := qualityMetric{Name: name, Condition: condition, Accent: accent, MaxWER: max, RefWords: g.words, Available: ok && g.words > 0}
	if strings.HasSuffix(condition, "20db") {
		metric.SNRDB = 20
	} else if strings.HasSuffix(condition, "10db") {
		metric.SNRDB = 10
	}
	if !metric.Available {
		*reasons = append(*reasons, "report has no measured "+name+" subgroup")
		*metrics = append(*metrics, metric)
		return
	}
	wer := float64(g.edits) / float64(g.words)
	metric.WER = &wer
	*metrics = append(*metrics, metric)
	if wer > max {
		*reasons = append(*reasons, fmt.Sprintf("%s WER %.4f exceeds %.4f", name, wer, max))
	}
}

func qualityAccent(tags []string) string {
	accent := ""
	for _, tag := range tags {
		if !strings.HasPrefix(tag, "accent_") {
			continue
		}
		candidate := strings.TrimPrefix(tag, "accent_")
		if candidate != "uk" && candidate != "us" && candidate != "south_african" {
			return ""
		}
		if accent != "" && accent != candidate {
			return ""
		}
		accent = candidate
	}
	return accent
}

func qualityCellPrerequisiteReasons(reasons []string, cell inteval.StrategyReport, measured bool, build string) []string {
	if !measured {
		reasons = append(reasons, "report did not measure quality")
	}
	if strings.TrimSpace(build) == "" || strings.EqualFold(strings.TrimSpace(build), "unknown") {
		reasons = append(reasons, "server build identity is unavailable")
	}
	if strings.TrimSpace(cell.EngineID) == "" || strings.TrimSpace(cell.ModelID) == "" || strings.TrimSpace(string(cell.Strategy)) == "" {
		reasons = append(reasons, "provider cell identity is incomplete")
	}
	if !qualitySafetyPassed(cell.Safety) {
		reasons = append(reasons, "report safety gate failed")
		reasons = append(reasons, cell.Safety.Reasons...)
	}
	return reasons
}

func qualityConditionGroup(meta intcorpus.Clip) string {
	group := string(meta.NoiseCondition)
	if meta.NoiseCondition == intcorpus.NoiseNoisy {
		if meta.SNRDB == 20 {
			return "noisy_20db"
		}
		if meta.SNRDB == 10 {
			return "noisy_10db"
		}
	}
	return group
}

// qualitySafetyPassed requires an explicit evaluated safety result. A quality
// receipt must not promote a report whose evaluated clips contain a retraction
// or threshold-sized dropped span, even when subgroup WER is in band; reports
// that omitted safety observation are also not eligible for promotion.
func qualitySafetyPassed(safety inteval.SafetyGateReport) bool {
	return safety.Passed
}

func qualityDeviceProfile(lane string) string {
	if strings.EqualFold(strings.TrimSpace(lane), "realtime") {
		return "D1"
	}
	// Replay lane is not a resource-profile declaration. In particular, a
	// deterministic/batch report does not prove the constrained D2 cell; that
	// profile requires its own owner-backed execution receipt. Keep the
	// provenance explicitly unqualified so a future quality pass cannot be
	// joined as native D2 evidence by inference.
	return "UNQUALIFIED"
}

func safePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return strings.NewReplacer("/", "_", ":", "_", " ", "_").Replace(value)
}
