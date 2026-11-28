//go:build audio_lab

package eval

import (
	"fmt"
	"math"
	"strings"
)

// TimestampDriftCriteria is the explicit acceptance envelope for one named
// model cell. Criteria are supplied by the caller instead of hidden in the
// scorer, so candidate thresholds remain reviewable and model policy remains
// data rather than a special-case for Whisper.
type TimestampDriftCriteria struct {
	EngineID                  string
	ModelID                   string
	MinimumDistinctRecordings int
	MinimumClipDurationMs     int64
	MinimumCoverage           float64
	MaximumStartP95Ms         float64
	MaximumEndP95Ms           float64
	// P95 thresholds alone can hide a severe isolated boundary failure in a
	// long recording. Keep the worst observed word boundary inside the
	// caller-supplied acceptance envelope too.
	MaximumStartErrorMs       float64
	MaximumEndErrorMs         float64
	MaximumAbsDriftMsPerHour  float64
}

type TimestampDriftVerdict struct {
	EngineID string   `json:"engine_id"`
	ModelID  string   `json:"model_id"`
	Passed   bool     `json:"passed"`
	Reasons  []string `json:"reasons,omitempty"`
}

// EvaluateTimestampDriftCell is a fail-closed acceptance gate over one
// strategy/model cell. It requires distinct reviewed source recordings and a
// scored result for every clip; it does not turn unreviewed AMI alignments or
// skipped reports into evidence. The per-call criteria define the provisional
// or owner-approved envelope and must be retained with any qualification
// receipt by the caller.
func EvaluateTimestampDriftCell(cell StrategyReport, criteria TimestampDriftCriteria) TimestampDriftVerdict {
	verdict := TimestampDriftVerdict{EngineID: cell.EngineID, ModelID: cell.ModelID}
	if err := validateTimestampDriftCriteria(criteria); err != nil {
		verdict.Reasons = append(verdict.Reasons, err.Error())
	}
	if cell.EngineID != criteria.EngineID || cell.ModelID != criteria.ModelID {
		verdict.Reasons = append(verdict.Reasons, "strategy cell identity does not match requested engine/model")
	}
	if len(cell.PerClip) == 0 {
		verdict.Reasons = append(verdict.Reasons, "report has no clips")
		return verdict
	}

	seenClips := make(map[string]bool, len(cell.PerClip))
	seenSources := make(map[string]bool, len(cell.PerClip))
	for _, clip := range cell.PerClip {
		if clip.ClipID == "" || seenClips[clip.ClipID] {
			verdict.Reasons = append(verdict.Reasons, "report contains an empty or duplicate clip id")
			continue
		}
		seenClips[clip.ClipID] = true
		if clip.AudioDurationMs < criteria.MinimumClipDurationMs {
			verdict.Reasons = append(verdict.Reasons, fmt.Sprintf("clip %q is shorter than the minimum duration", clip.ClipID))
		}
		if clip.TimestampDrift == nil || clip.TimestampDriftSkipReason != "" {
			verdict.Reasons = append(verdict.Reasons, fmt.Sprintf("clip %q has no scored timestamp drift", clip.ClipID))
			continue
		}
		provenance := clip.TimestampReference
		if !provenance.ReviewReady() {
			verdict.Reasons = append(verdict.Reasons, fmt.Sprintf("clip %q lacks human-reviewed timing provenance", clip.ClipID))
		} else {
			// Source IDs are operator metadata. Canonicalize surrounding
			// whitespace so formatting differences cannot inflate independent
			// recording coverage in a qualification receipt.
			seenSources[strings.TrimSpace(provenance.SourceReference)] = true
		}
		drift := clip.TimestampDrift
		if drift.ReferenceWords <= 0 || drift.MatchedWords <= 0 || drift.MatchedWords > drift.ReferenceWords ||
			!finiteTimingValue(drift.Coverage) || drift.Coverage < criteria.MinimumCoverage || drift.Coverage > 1 {
			verdict.Reasons = append(verdict.Reasons, fmt.Sprintf("clip %q is below minimum matched-word coverage", clip.ClipID))
		} else if expectedCoverage := float64(drift.MatchedWords) / float64(drift.ReferenceWords); math.Abs(drift.Coverage-expectedCoverage) > 1e-12 {
			// Coverage is derived from the integer match counts by the scorer.
			// Reject inconsistent report rows so a stale or edited percentage
			// cannot satisfy the gate while the underlying lexical coverage fails.
			verdict.Reasons = append(verdict.Reasons, fmt.Sprintf("clip %q has inconsistent matched-word coverage", clip.ClipID))
		}
		if !finiteTimingValue(drift.StartAbsP95Ms) || drift.StartAbsP95Ms < 0 || drift.StartAbsP95Ms > criteria.MaximumStartP95Ms {
			verdict.Reasons = append(verdict.Reasons, fmt.Sprintf("clip %q exceeds maximum start-boundary P95", clip.ClipID))
		}
		if !finiteTimingValue(drift.EndAbsP95Ms) || drift.EndAbsP95Ms < 0 || drift.EndAbsP95Ms > criteria.MaximumEndP95Ms {
			verdict.Reasons = append(verdict.Reasons, fmt.Sprintf("clip %q exceeds maximum end-boundary P95", clip.ClipID))
		}
		if !finiteTimingValue(drift.MaxStartAbsMs) || drift.MaxStartAbsMs < 0 || drift.MaxStartAbsMs > criteria.MaximumStartErrorMs {
			verdict.Reasons = append(verdict.Reasons, fmt.Sprintf("clip %q exceeds maximum start-boundary error", clip.ClipID))
		}
		if !finiteTimingValue(drift.MaxEndAbsMs) || drift.MaxEndAbsMs < 0 || drift.MaxEndAbsMs > criteria.MaximumEndErrorMs {
			verdict.Reasons = append(verdict.Reasons, fmt.Sprintf("clip %q exceeds maximum end-boundary error", clip.ClipID))
		}
		if !drift.DriftRateObserved || !finiteTimingValue(drift.StartDriftMsPerHour) || !finiteTimingValue(drift.EndDriftMsPerHour) ||
			math.Abs(drift.StartDriftMsPerHour) > criteria.MaximumAbsDriftMsPerHour || math.Abs(drift.EndDriftMsPerHour) > criteria.MaximumAbsDriftMsPerHour {
			verdict.Reasons = append(verdict.Reasons, fmt.Sprintf("clip %q exceeds maximum absolute drift rate or lacks observations", clip.ClipID))
		}
	}
	if len(seenSources) < criteria.MinimumDistinctRecordings {
		verdict.Reasons = append(verdict.Reasons, fmt.Sprintf("report covers %d distinct reviewed recordings; requires %d", len(seenSources), criteria.MinimumDistinctRecordings))
	}
	verdict.Passed = len(verdict.Reasons) == 0
	return verdict
}

func validateTimestampDriftCriteria(criteria TimestampDriftCriteria) error {
	if strings.TrimSpace(criteria.EngineID) == "" || strings.TrimSpace(criteria.ModelID) == "" {
		return fmt.Errorf("timestamp drift criteria require engine and model ids")
	}
	if criteria.MinimumDistinctRecordings < 1 || criteria.MinimumClipDurationMs <= 0 {
		return fmt.Errorf("timestamp drift criteria require positive recording count and clip duration")
	}
	if !finiteTimingValue(criteria.MinimumCoverage) || criteria.MinimumCoverage < 0 || criteria.MinimumCoverage > 1 ||
		!finiteTimingValue(criteria.MaximumStartP95Ms) || criteria.MaximumStartP95Ms < 0 ||
		!finiteTimingValue(criteria.MaximumEndP95Ms) || criteria.MaximumEndP95Ms < 0 ||
		!finiteTimingValue(criteria.MaximumStartErrorMs) || criteria.MaximumStartErrorMs < 0 ||
		!finiteTimingValue(criteria.MaximumEndErrorMs) || criteria.MaximumEndErrorMs < 0 ||
		!finiteTimingValue(criteria.MaximumAbsDriftMsPerHour) || criteria.MaximumAbsDriftMsPerHour < 0 {
		return fmt.Errorf("timestamp drift thresholds must be finite and non-negative, with coverage in [0,1]")
	}
	return nil
}

func finiteTimingValue(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
