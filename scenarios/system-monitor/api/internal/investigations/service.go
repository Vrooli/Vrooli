package investigations

import (
	"context"
	"time"
)

type Service struct {
	repo          Repository
	retentionDays int
}

func NewService(repo Repository, retentionDays int) *Service {
	if retentionDays <= 0 {
		retentionDays = 7
	}
	return &Service{repo: repo, retentionDays: retentionDays}
}
func (s *Service) SaveRun(ctx context.Context, run Run) error { return s.repo.SaveRun(ctx, run) }
func (s *Service) ListRuns(ctx context.Context, entryID string, since time.Time, limit int) ([]Run, error) {
	return s.repo.ListRuns(ctx, entryID, since, limit)
}
func (s *Service) GetRun(ctx context.Context, id string) (Run, error) { return s.repo.GetRun(ctx, id) }
func (s *Service) Prune(ctx context.Context, now time.Time, dryRun bool) (int64, error) {
	cutoff := now.UTC().AddDate(0, 0, -s.retentionDays)
	if dryRun {
		return s.repo.CountRunsBefore(ctx, cutoff)
	}
	return s.repo.PruneRunsBefore(ctx, cutoff)
}
