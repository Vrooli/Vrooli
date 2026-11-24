//go:build audio_lab

package eval

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// These synthetic report rows test only the gate logic. They are not model
// measurements and cannot qualify either Whisper tier.
func TestEvaluateTimestampDriftCell_RequiresEveryReviewedLongFormCell(t *testing.T) {
	for _, modelID := range []string{"whisper-small", "whisper-large-v3-turbo"} {
		t.Run(modelID, func(t *testing.T) {
			criteria := TimestampDriftCriteria{
				EngineID: "whisper-local", ModelID: modelID,
				MinimumDistinctRecordings: 3, MinimumClipDurationMs: 10 * 60 * 1000,
				MinimumCoverage: .9, MaximumStartP95Ms: 250,
				MaximumEndP95Ms: 250, MaximumStartErrorMs: 1000,
				MaximumEndErrorMs: 1000, MaximumAbsDriftMsPerHour: 250,
			}
			verdict := EvaluateTimestampDriftCell(timestampCell(modelID, 3), criteria)
			require.True(t, verdict.Passed, "%v", verdict.Reasons)
			require.Empty(t, verdict.Reasons)

			short := timestampCell(modelID, 2)
			short.PerClip[1].TimestampDriftSkipReason = "timed_reference_not_human_reviewed"
			short.PerClip[1].TimestampDrift = nil
			verdict = EvaluateTimestampDriftCell(short, criteria)
			require.False(t, verdict.Passed)
			require.Contains(t, verdict.Reasons, `clip "clip-1" has no scored timestamp drift`)
			require.Contains(t, verdict.Reasons, "report covers 1 distinct reviewed recordings; requires 3")
		})
	}
}

func TestEvaluateTimestampDriftCell_FailsClosedOnModelAndMetricMismatch(t *testing.T) {
	criteria := TimestampDriftCriteria{
		EngineID: "whisper-local", ModelID: "whisper-small",
		MinimumDistinctRecordings: 3, MinimumClipDurationMs: 600_000,
		MinimumCoverage: .9, MaximumStartP95Ms: 250,
		MaximumEndP95Ms: 250, MaximumStartErrorMs: 1000,
		MaximumEndErrorMs: 1000, MaximumAbsDriftMsPerHour: 250,
	}
	cell := timestampCell("whisper-large-v3-turbo", 3)
	cell.PerClip[0].TimestampDrift.StartDriftMsPerHour = 251
	verdict := EvaluateTimestampDriftCell(cell, criteria)
	require.False(t, verdict.Passed)
	require.Contains(t, verdict.Reasons, "strategy cell identity does not match requested engine/model")
	require.Contains(t, verdict.Reasons, `clip "clip-0" exceeds maximum absolute drift rate or lacks observations`)

	// A serialized/stale coverage value must agree with the scorer's integer
	// match counts; otherwise a low lexical match rate could be represented as
	// a passing percentage.
	cell = timestampCell(criteria.ModelID, 3)
	cell.PerClip[0].TimestampDrift.MatchedWords = 100
	verdict = EvaluateTimestampDriftCell(cell, criteria)
	require.False(t, verdict.Passed)
	require.Contains(t, verdict.Reasons, `clip "clip-0" has inconsistent matched-word coverage`)

	criteria.MaximumEndP95Ms = -1
	verdict = EvaluateTimestampDriftCell(timestampCell(criteria.ModelID, 3), criteria)
	require.False(t, verdict.Passed)
	require.Contains(t, verdict.Reasons, "timestamp drift thresholds must be finite and non-negative, with coverage in [0,1]")
}

func TestEvaluateTimestampDriftCell_RejectsNegativeAbsolutePercentiles(t *testing.T) {
	criteria := TimestampDriftCriteria{
		EngineID: "whisper-local", ModelID: "whisper-small",
		MinimumDistinctRecordings: 3, MinimumClipDurationMs: 600_000,
		MinimumCoverage: .9, MaximumStartP95Ms: 250,
		MaximumEndP95Ms: 250, MaximumStartErrorMs: 1000,
		MaximumEndErrorMs: 1000, MaximumAbsDriftMsPerHour: 250,
	}
	cell := timestampCell(criteria.ModelID, 3)
	cell.PerClip[0].TimestampDrift.StartAbsP95Ms = -1
	cell.PerClip[1].TimestampDrift.EndAbsP95Ms = -1

	verdict := EvaluateTimestampDriftCell(cell, criteria)
	require.False(t, verdict.Passed)
	require.Contains(t, verdict.Reasons, `clip "clip-0" exceeds maximum start-boundary P95`)
	require.Contains(t, verdict.Reasons, `clip "clip-1" exceeds maximum end-boundary P95`)
}

func TestEvaluateTimestampDriftCell_DoesNotCountWhitespaceVariantsAsDistinctSources(t *testing.T) {
	criteria := TimestampDriftCriteria{
		EngineID: "whisper-local", ModelID: "whisper-small",
		MinimumDistinctRecordings: 3, MinimumClipDurationMs: 600_000,
		MinimumCoverage: .9, MaximumStartP95Ms: 250,
		MaximumEndP95Ms: 250, MaximumStartErrorMs: 1000,
		MaximumEndErrorMs: 1000, MaximumAbsDriftMsPerHour: 250,
	}
	cell := timestampCell(criteria.ModelID, 3)
	cell.PerClip[1].TimestampReference.SourceReference = " ami-meeting-0 "
	cell.PerClip[2].TimestampReference.SourceReference = "ami-meeting-1"

	verdict := EvaluateTimestampDriftCell(cell, criteria)
	require.False(t, verdict.Passed)
	require.Contains(t, verdict.Reasons, "report covers 2 distinct reviewed recordings; requires 3")
	require.False(t, (&TimingReferenceProvenance{Status: "human_reviewed", SourceReference: "   ", Reviewer: "r", ReviewMethod: "m", ReviewedAt: "2026-11-22T12:00:00Z"}).ReviewReady())
}

func TestEvaluateTimestampDriftCell_RejectsIsolatedLongFormBoundarySpike(t *testing.T) {
	criteria := TimestampDriftCriteria{
		EngineID: "whisper-local", ModelID: "whisper-small",
		MinimumDistinctRecordings: 1, MinimumClipDurationMs: 600_000,
		MinimumCoverage: .9, MaximumStartP95Ms: 250, MaximumEndP95Ms: 250,
		MaximumStartErrorMs: 500, MaximumEndErrorMs: 500, MaximumAbsDriftMsPerHour: 250,
	}
	cell := timestampCell(criteria.ModelID, 1)
	// A single severe error in thousands of words may barely move P95 or the
	// fitted drift rate. The maximum-error guard must still catch it.
	cell.PerClip[0].TimestampDrift.MaxStartAbsMs = 2000
	cell.PerClip[0].TimestampDrift.MaxEndAbsMs = 2000
	verdict := EvaluateTimestampDriftCell(cell, criteria)
	require.False(t, verdict.Passed)
	require.Contains(t, verdict.Reasons, `clip "clip-0" exceeds maximum start-boundary error`)
	require.Contains(t, verdict.Reasons, `clip "clip-0" exceeds maximum end-boundary error`)
}

func timestampCell(modelID string, clips int) StrategyReport {
	cell := StrategyReport{EngineID: "whisper-local", ModelID: modelID}
	for i := 0; i < clips; i++ {
		cell.PerClip = append(cell.PerClip, ClipResult{
			ClipID:          fmt.Sprintf("clip-%d", i),
			AudioDurationMs: 10 * 60 * 1000,
			TimestampReference: &TimingReferenceProvenance{
				Status: "human_reviewed", SourceReference: fmt.Sprintf("ami-meeting-%d", i),
				Reviewer: "reviewer", ReviewMethod: "manual-listen-and-correct", ReviewedAt: "2026-11-22T12:00:00Z",
			},
			TimestampDrift: &TimestampDriftResult{
				ReferenceWords: 1000, MatchedWords: 980, Coverage: .98,
				StartAbsP95Ms: 180, EndAbsP95Ms: 190, MaxStartAbsMs: 220, MaxEndAbsMs: 230,
				DriftRateObserved: true, StartDriftMsPerHour: 40, EndDriftMsPerHour: -35,
			},
		})
	}
	return cell
}
