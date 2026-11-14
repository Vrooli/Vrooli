package shots

import (
	"testing"

	"github.com/stretchr/testify/require"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

func TestVideoReadabilityIntervalFitsSavedTrimDuration(t *testing.T) {
	timing := &shotsv1.VideoTiming{TrimStartPts: 24, TrimEndPts: 264, TimeBaseNum: 1, TimeBaseDen: 24, PlaybackRateNum: 1, PlaybackRateDen: 1}
	duration, err := videoProofDurationMillis(timing, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(10000), duration)
	proof := &shotsv1.ReadabilityProof{RegionName: "title", RequiredContent: "Station", Criteria: "Legible at final size", RegionX: .1, RegionY: .1, RegionWidth: .8, RegionHeight: .2, MinimumOutputWidthPixels: 1, MinimumOutputHeightPixels: 1, MinimumRegionWidthPixels: 1, MinimumRegionHeightPixels: 1, ProofStartMs: 9000, ProofEndMs: 10001}
	require.NoError(t, validateReadabilityProof(proof, true))
	require.ErrorContains(t, validateVideoProofInterval(proof, timing, nil), "exceeds the saved video output duration")
	proof.ProofEndMs = duration
	require.NoError(t, validateVideoProofInterval(proof, timing, nil))
}

func TestPhysicalMotionReadabilityUsesFinalFrameScheduleForStillSourceMotion(t *testing.T) {
	output := &shotsv1.PhysicalMotionOutputBinding{FrameCount: 48, OutputRateNumerator: 24, OutputRateDenominator: 1}
	proof := &shotsv1.ReadabilityProof{RegionName: "screen title", RequiredContent: "Status ready", Criteria: "The title is crisp at native output pixels", RegionX: .1, RegionY: .1, RegionWidth: .8, RegionHeight: .2, MinimumOutputWidthPixels: 1000, MinimumOutputHeightPixels: 500, MinimumRegionWidthPixels: 800, MinimumRegionHeightPixels: 100, ProofStartMs: 500, ProofEndMs: 1500}
	require.True(t, snapshotHasVideoOutput(&shotsv1.ShotSnapshot{PhysicalBinding: &shotsv1.PhysicalShotBinding{MotionOutput: output}}))
	require.NoError(t, validateReadabilityProof(proof, true))
	duration, err := videoProofDurationMillis(nil, output)
	require.NoError(t, err)
	require.Equal(t, uint64(2000), duration)
	require.NoError(t, validateVideoProofInterval(proof, nil, output))
	proof.ProofEndMs = 2001
	require.ErrorContains(t, validateVideoProofInterval(proof, nil, output), "exceeds the saved video output duration")
}

func TestRecordingSnapshotWithPhysicalMotionUsesFinalPhysicalFrameSchedule(t *testing.T) {
	output := &shotsv1.PhysicalMotionOutputBinding{FrameCount: 120, OutputRateNumerator: 30, OutputRateDenominator: 1}
	timing := &shotsv1.VideoTiming{TrimStartPts: 0, TrimEndPts: 10, TimeBaseNum: 1, TimeBaseDen: 1, PlaybackRateNum: 1, PlaybackRateDen: 1}
	snapshot := &shotsv1.ShotSnapshot{VideoTiming: timing, PhysicalBinding: &shotsv1.PhysicalShotBinding{MotionOutput: output}}
	require.True(t, snapshotHasVideoOutput(snapshot))
	proof := &shotsv1.ReadabilityProof{RegionName: "status", RequiredContent: "Ready", Criteria: "Every character is distinct at native output pixels", RegionX: .1, RegionY: .1, RegionWidth: .8, RegionHeight: .2, MinimumOutputWidthPixels: 1000, MinimumOutputHeightPixels: 500, MinimumRegionWidthPixels: 800, MinimumRegionHeightPixels: 100, ProofStartMs: 3000, ProofEndMs: 3900}
	require.NoError(t, validateVideoProofInterval(proof, timing, output))
	proof.ProofEndMs = 4001
	require.ErrorContains(t, validateVideoProofInterval(proof, timing, output), "exceeds the saved video output duration")
}
