package shots_test

import (
	"context"
	"errors"
	"testing"

	"device-studio/internal/shots"
	"github.com/stretchr/testify/require"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
)

type readabilityOutputStub struct {
	wantProject, wantShot, wantRender, wantSHA string
	wantRevision                               int64
	output                                     shots.ReadabilityOutput
	err                                        error
}

func (s *readabilityOutputStub) VerifyReadabilityOutput(_ context.Context, project, shot string, revision int64, render, sha string) (shots.ReadabilityOutput, error) {
	if project != s.wantProject || shot != s.wantShot || revision != s.wantRevision || render != s.wantRender || sha != s.wantSHA {
		return shots.ReadabilityOutput{}, errors.New("output binding mismatch")
	}
	if s.err != nil {
		return shots.ReadabilityOutput{}, s.err
	}
	return s.output, nil
}

func readabilityProof() *shotsv1.ReadabilityProof {
	return &shotsv1.ReadabilityProof{RegionName: "headline", RequiredContent: "Station name and arrival time", Criteria: "Both strings are legible at native output size", RegionX: .1, RegionY: .1, RegionWidth: .5, RegionHeight: .25, MinimumOutputWidthPixels: 1200, MinimumOutputHeightPixels: 800, MinimumRegionWidthPixels: 500, MinimumRegionHeightPixels: 200}
}

func TestReadabilityReviewRequiresAdequateExactOutputAndStaysRevisionBound(t *testing.T) {
	ctx := context.Background()
	service, projectID, _, _, _ := setupAll(t)
	snapshot := validSnapshot()
	snapshot.ReadabilityProof = readabilityProof()
	shot, first, err := service.Create(ctx, projectID, "Readable output", snapshot)
	require.NoError(t, err)
	verifier := &readabilityOutputStub{wantProject: projectID, wantShot: shot.ID, wantRevision: 1, wantRender: "render-1", wantSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", output: shots.ReadabilityOutput{Width: 800, Height: 600, SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", MediaType: "image/png"}}
	shots.SetReadabilityOutputVerifier(service, verifier)
	request := shots.ReviewReadabilityRequest{ProjectID: projectID, ShotID: shot.ID, ShotRevision: 1, RenderID: "render-1", OutputSHA256: verifier.wantSHA, Verdict: int32(shotsv1.ReadabilityVerdict_READABILITY_VERDICT_ACCEPTED), ExplicitlyInspectedFinalSize: true}
	_, err = service.ReviewOutputReadability(ctx, request)
	require.ErrorContains(t, err, "below the declared minimum")
	request.Verdict = int32(shotsv1.ReadabilityVerdict_READABILITY_VERDICT_REJECTED)
	rejected, err := service.ReviewOutputReadability(ctx, request)
	require.NoError(t, err)
	require.Equal(t, shotsv1.ReadabilityVerdict_READABILITY_VERDICT_REJECTED, shotsv1.ReadabilityVerdict(rejected.Verdict))

	verifier.output = shots.ReadabilityOutput{Width: 2560, Height: 1440, SHA256: verifier.wantSHA, MediaType: "image/png"}
	request.Verdict = int32(shotsv1.ReadabilityVerdict_READABILITY_VERDICT_ACCEPTED)
	_, err = service.ReviewOutputReadability(ctx, request)
	require.ErrorIs(t, err, shots.ErrReadabilityAlreadyReviewed)
	verifier.wantRender = "render-2"
	verifier.wantSHA = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	verifier.output.SHA256 = verifier.wantSHA
	request.RenderID = verifier.wantRender
	request.OutputSHA256 = verifier.wantSHA
	request.Verdict = int32(shotsv1.ReadabilityVerdict_READABILITY_VERDICT_ACCEPTED)
	accepted, err := service.ReviewOutputReadability(ctx, request)
	require.NoError(t, err)
	require.Equal(t, uint32(1280), accepted.RegionWidth)
	require.Equal(t, first.Revision, accepted.ShotRevision)
	readback, err := service.GetOutputReadabilityReview(ctx, projectID, shot.ID, 1, "render-2", verifier.wantSHA)
	require.NoError(t, err)
	require.Equal(t, accepted.ID, readback.ID)
	rejectedReadback, err := service.GetOutputReadabilityReview(ctx, projectID, shot.ID, 1, "render-1", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	require.NoError(t, err)
	require.Equal(t, shotsv1.ReadabilityVerdict_READABILITY_VERDICT_REJECTED, shotsv1.ReadabilityVerdict(rejectedReadback.Verdict))
	_, err = service.GetOutputReadabilityReview(ctx, projectID, shot.ID, 1, "render-2", "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
	require.ErrorIs(t, err, shots.ErrReadabilityReviewNotFound)

	changed := proto.Clone(first.Snapshot).(*shotsv1.ShotSnapshot)
	changed.ReadabilityProof.Criteria = "A later revision uses a different criterion"
	_, second, err := service.Save(ctx, shot.ID, 1, "Readable output", changed)
	require.NoError(t, err)
	_, err = service.GetOutputReadabilityReview(ctx, projectID, shot.ID, second.Revision, "render-2", verifier.wantSHA)
	require.ErrorIs(t, err, shots.ErrReadabilityReviewNotFound)
	request.ShotRevision = 2
	_, err = service.ReviewOutputReadability(ctx, request)
	require.ErrorContains(t, err, "output binding mismatch")
}

func TestReadabilityDeclarationRejectsInvalidRegionAndReviewNeedsExplicitAttestation(t *testing.T) {
	service, projectID, _, _, _ := setupAll(t)
	invalid := validSnapshot()
	invalid.ReadabilityProof = readabilityProof()
	invalid.ReadabilityProof.RegionX = .8
	invalid.ReadabilityProof.RegionWidth = .3
	_, _, err := service.Create(context.Background(), projectID, "Invalid proof", invalid)
	require.ErrorContains(t, err, "normalized output bounds")

	valid := validSnapshot()
	valid.ReadabilityProof = readabilityProof()
	shot, _, err := service.Create(context.Background(), projectID, "Valid proof", valid)
	require.NoError(t, err)
	verifier := &readabilityOutputStub{wantProject: projectID, wantShot: shot.ID, wantRevision: 1, wantRender: "render-1", wantSHA: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", output: shots.ReadabilityOutput{Width: 2560, Height: 1440, SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", MediaType: "image/png"}}
	shots.SetReadabilityOutputVerifier(service, verifier)
	_, err = service.ReviewOutputReadability(context.Background(), shots.ReviewReadabilityRequest{ProjectID: projectID, ShotID: shot.ID, ShotRevision: 1, RenderID: "render-1", OutputSHA256: verifier.wantSHA, Verdict: int32(shotsv1.ReadabilityVerdict_READABILITY_VERDICT_ACCEPTED)})
	require.ErrorContains(t, err, "explicit final-size inspection")
}

func TestStaleReadabilityOutputBindingDoesNotRecordVerdict(t *testing.T) {
	service, projectID, _, _, _ := setupAll(t)
	snapshot := validSnapshot()
	snapshot.ReadabilityProof = readabilityProof()
	shot, _, err := service.Create(context.Background(), projectID, "Stale output review", snapshot)
	require.NoError(t, err)
	sha := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	verifier := &readabilityOutputStub{
		wantProject: projectID, wantShot: shot.ID, wantRevision: 1, wantRender: "render-1", wantSHA: sha,
		err: shots.ErrInvalidShot{Field: "readability_review.binding", Reason: "render must be a succeeded output for this exact shot revision"},
	}
	shots.SetReadabilityOutputVerifier(service, verifier)
	request := shots.ReviewReadabilityRequest{ProjectID: projectID, ShotID: shot.ID, ShotRevision: 1, RenderID: "render-1", OutputSHA256: sha, Verdict: int32(shotsv1.ReadabilityVerdict_READABILITY_VERDICT_ACCEPTED), ExplicitlyInspectedFinalSize: true}
	_, err = service.ReviewOutputReadability(context.Background(), request)
	var invalid shots.ErrInvalidShot
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "readability_review.binding", invalid.Field)
	_, err = service.GetOutputReadabilityReview(context.Background(), projectID, shot.ID, 1, "render-1", sha)
	require.ErrorIs(t, err, shots.ErrReadabilityReviewNotFound)
}
