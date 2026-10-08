package server

import (
	"context"

	"deployment-manager/readiness"
	"deployment-manager/releases"
)

type readinessReleaseReader interface {
	ListByProfile(context.Context, string, int) ([]*releases.Release, error)
}

type readinessReviewReader interface {
	Get(context.Context, string) (*readiness.Review, error)
}

type readinessPredecessorResolver struct {
	releases readinessReleaseReader
	reviews  readinessReviewReader
}

func (r readinessPredecessorResolver) LatestDeployed(ctx context.Context, identity readiness.ReviewIdentity) (*readiness.Predecessor, error) {
	items, err := r.releases.ListByProfile(ctx, identity.ProfileID, 100)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item == nil || item.Status != releases.StatusPublished || item.Channel != identity.Channel || item.GitCommitHash == identity.CandidateCommit || !sameTargetSet(item.Platforms, identity.Targets) {
			continue
		}
		predecessor := &readiness.Predecessor{ReleaseID: item.ID, Commit: item.GitCommitHash, ArtifactDigest: item.ArtifactDigest}
		if item.ReadinessReviewKey != "" && r.reviews != nil {
			if review, reviewErr := r.reviews.Get(ctx, item.ReadinessReviewKey); reviewErr == nil && review != nil {
				predecessor.PolicyVersion = review.Identity.PolicyVersion
			}
		}
		return predecessor, nil
	}
	return nil, nil
}

func sameTargetSet(platforms []releases.ReleasePlatform, targets []string) bool {
	if len(platforms) != len(targets) {
		return false
	}
	counts := make(map[string]int, len(platforms))
	for _, platform := range platforms {
		counts[platform.Platform]++
	}
	for _, target := range targets {
		if counts[target] == 0 {
			return false
		}
		counts[target]--
	}
	return true
}
