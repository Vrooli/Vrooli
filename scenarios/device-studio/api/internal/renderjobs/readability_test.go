package renderjobs

import (
	"testing"

	"device-studio/internal/jobs"
	"device-studio/internal/shots"
	"github.com/stretchr/testify/require"
)

func TestReadabilityRequiresSucceededJobAndExactImmutableShotRevision(t *testing.T) {
	input := jobs.Input{ProjectID: "project", ShotRevisionID: shotRevisionID("shot", 4), ShotRevision: 4, Bindings: []jobs.ArtifactBinding{{Role: "shot_snapshot", ArtifactID: "shot:shot:revision:4", RevisionID: "revision:4", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}}
	job := jobs.Job{Input: input, State: jobs.Succeeded}
	require.True(t, readabilityJobMatches(job, "project", "shot", 4))
	require.False(t, readabilityJobMatches(job, "project", "shot", 3))
	require.False(t, readabilityJobMatches(job, "other", "shot", 4))
	job.State = jobs.Running
	require.False(t, readabilityJobMatches(job, "project", "shot", 4))
}

func TestReadabilityStaleRevisionIsCallerInvalidAndMissingJobIsNotFound(t *testing.T) {
	c, ctx, _ := integrationCoordinator(t, &integrationOps{}, false)
	job := submitIntegrationJob(t, c, ctx)
	if _, err := c.Advance(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
	completed, err := c.Reconcile(ctx, job.ID)
	require.NoError(t, err)
	require.Equal(t, jobs.Succeeded, completed.State)
	sha := completed.Attempts[0].Completion.Outputs[0].SHA256

	_, err = c.VerifyReadabilityOutput(ctx, "project", "shot", 2, completed.ID, sha)
	var invalid shots.ErrInvalidShot
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "readability_review.binding", invalid.Field)

	_, err = c.VerifyReadabilityOutput(ctx, "project", "shot", 1, "missing-render", sha)
	require.ErrorIs(t, err, shots.ErrReadabilityRenderNotFound)
}

func TestReadabilityAcceptsOnlyFinalStillAndVideoProfiles(t *testing.T) {
	require.True(t, readabilityFinalStillProfile(OutputProfilePNG))
	require.False(t, readabilityFinalStillProfile(OutputProfilePhysicalPreviewPNG))
	require.False(t, readabilityFinalStillProfile(OutputProfileMP4))
	require.True(t, isFinalVideoProfile(OutputProfileMP4))
	require.True(t, isFinalVideoProfile(OutputProfilePhysicalMotionMP4))
	require.True(t, isFinalVideoProfile(OutputProfilePhysicalMotionMP4V2))
	require.True(t, isFinalVideoProfile(OutputProfilePhysicalRecordingMP4))
	require.False(t, isFinalVideoProfile(OutputProfilePhysicalPreviewPNG))
	require.False(t, isFinalVideoProfile("video-sample-png-v1"))
}
