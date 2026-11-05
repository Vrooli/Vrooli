package proposals

import (
	"context"
	"errors"

	"git-control-tower/internal/trailers"
)

// Resolver resolves the trailer kinds GCT owns: proposals in its store and
// continued commits in the repository. Effort, epoch, run, plan and backlog
// references delegate to the configured owner resolver; unavailable evidence
// remains unresolved and never blocks a proposal.
func (s *Service) Resolver(repo Repo) trailers.Resolver {
	return localResolver{service: s, repo: repo}
}

type localResolver struct {
	service *Service
	repo    Repo
}

func (r localResolver) Resolve(ctx context.Context, key, value string) (string, trailers.ResolutionStatus, string, error) {
	spec, _ := trailers.SpecFor(key)
	switch spec.Kind {
	case "proposal":
		proposal, err := r.service.store.Get(ctx, value)
		if errors.Is(err, ErrNotFound) {
			return "", trailers.Unresolved, "no proposal with this ID in Git Control Tower", nil
		}
		if err != nil {
			return "", "", "", err
		}
		if proposal.RepositoryID != r.repo.ID {
			return "", trailers.Inaccessible, "", nil
		}
		return proposal.ID, trailers.Resolved, "proposal " + string(proposal.State), nil
	case "commit":
		exists, err := r.repo.Git.ObjectExists(ctx, value)
		if err != nil {
			return "", "", "", err
		}
		if !exists {
			return "", trailers.Unresolved, "commit not found in this repository", nil
		}
		return value, trailers.Resolved, "commit exists in this repository", nil
	default:
		if r.service.ownerResolver != nil {
			return r.service.ownerResolver.Resolve(ctx, key, value)
		}
		return "", trailers.Unresolved, "owner evidence unavailable for " + trailers.KindOf(key, value), nil
	}
}
