package experiment

import (
	"context"
	"fmt"
	"time"
)

// Service ties experiment metadata to large report artifacts. The async runner
// owns lifecycle transitions; this service keeps artifact writes atomic enough
// for callers that need to persist a report and update the result ref.
type Service struct {
	repo  Repository
	blobs BlobBytes
}

// NewService constructs an experiment Service.
func NewService(repo Repository, blobs BlobBytes) *Service {
	return &Service{repo: repo, blobs: blobs}
}

func (s *Service) CreateExperiment(ctx context.Context, exp Experiment) (Experiment, error) {
	return s.repo.CreateExperiment(ctx, exp)
}

func (s *Service) GetExperiment(ctx context.Context, id string) (Experiment, error) {
	return s.repo.GetExperiment(ctx, id)
}

func (s *Service) ListExperiments(ctx context.Context, filter ListFilter) ([]Experiment, error) {
	return s.repo.ListExperiments(ctx, filter)
}

func (s *Service) ListRuns(ctx context.Context, experimentID string) ([]Run, error) {
	return s.repo.ListRuns(ctx, experimentID)
}

func (s *Service) GetReport(ctx context.Context, exp Experiment) ([]byte, error) {
	if exp.ResultRef == "" {
		return nil, fmt.Errorf("experiment: %q has no report", exp.ID)
	}
	return s.blobs.Get(ctx, exp.ResultRef)
}

func (s *Service) StoreReport(ctx context.Context, exp Experiment, report []byte, mime string, now time.Time) (Experiment, error) {
	if exp.ID == "" {
		return Experiment{}, fmt.Errorf("experiment: StoreReport requires experiment id")
	}
	key := fmt.Sprintf("reports/%s/%s.json", now.UTC().Format("2006-01"), exp.ID)
	if err := s.blobs.Put(ctx, key, report, mime); err != nil {
		return Experiment{}, fmt.Errorf("experiment: store report: %w", err)
	}
	exp.ResultRef = key
	if err := s.repo.UpdateExperiment(ctx, exp); err != nil {
		_ = s.blobs.Delete(ctx, key)
		return Experiment{}, err
	}
	return exp, nil
}
