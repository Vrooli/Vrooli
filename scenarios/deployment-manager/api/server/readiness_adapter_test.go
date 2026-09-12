package server

import (
	"context"
	"errors"
	"testing"

	"deployment-manager/readiness"
	"deployment-manager/releases"
)

type predecessorReleaseReader struct{ releases []*releases.Release }

func (r predecessorReleaseReader) ListByProfile(context.Context, string, int) ([]*releases.Release, error) {
	return r.releases, nil
}

type predecessorReviewReader struct{ reviews map[string]*readiness.Review }

func (r predecessorReviewReader) Get(_ context.Context, key string) (*readiness.Review, error) {
	if review := r.reviews[key]; review != nil {
		return review, nil
	}
	return nil, errors.New("missing")
}

func TestPredecessorUsesLatestPublishedReleaseForExactTargetsAndChannel(t *testing.T) {
	items := []*releases.Release{
		{ID: "wrong-target", Status: releases.StatusPublished, Channel: "stable", GitCommitHash: "newer", ArtifactDigest: "sha256:newer", Platforms: []releases.ReleasePlatform{{Platform: "win-x64"}}},
		{ID: "right", Status: releases.StatusPublished, Channel: "stable", GitCommitHash: "old", ArtifactDigest: "sha256:old", ReadinessReviewKey: "rr-old", Platforms: []releases.ReleasePlatform{{Platform: "linux-x64"}}},
	}
	resolver := readinessPredecessorResolver{releases: predecessorReleaseReader{releases: items}, reviews: predecessorReviewReader{reviews: map[string]*readiness.Review{"rr-old": {Identity: readiness.ReviewIdentity{PolicyVersion: 2}}}}}
	got, err := resolver.LatestDeployed(context.Background(), readiness.ReviewIdentity{ProfileID: "p1", CandidateCommit: "candidate", Channel: "stable", Targets: []string{"linux-x64"}})
	if err != nil || got == nil || got.ReleaseID != "right" || got.PolicyVersion != 2 {
		t.Fatalf("predecessor=%+v err=%v", got, err)
	}
}
