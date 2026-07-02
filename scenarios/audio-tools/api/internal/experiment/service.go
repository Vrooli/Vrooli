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

func (s *Service) DeleteExperiment(ctx context.Context, exp Experiment) (bool, error) {
	if !exp.Status.Terminal() {
		return false, fmt.Errorf("experiment: cannot delete %q while status is %s", exp.ID, exp.Status)
	}
	if err := s.repo.DeleteExperiment(ctx, exp.ID); err != nil {
		return false, err
	}
	if exp.ResultRef == "" {
		return false, nil
	}
	if err := s.blobs.Delete(ctx, exp.ResultRef); err != nil {
		return false, fmt.Errorf("experiment: delete report %q: %w", exp.ResultRef, err)
	}
	return true, nil
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
	key, err := s.storeReportBlob(ctx, exp, report, mime, now)
	if err != nil {
		return Experiment{}, err
	}
	exp.ResultRef = key
	if err := s.repo.UpdateExperiment(ctx, exp); err != nil {
		if deleteErr := s.blobs.Delete(ctx, key); deleteErr != nil {
			return Experiment{}, fmt.Errorf("experiment: update report ref: %w; rollback delete %q: %v", err, key, deleteErr)
		}
		return Experiment{}, err
	}
	return exp, nil
}

func (s *Service) storeReportBlob(ctx context.Context, exp Experiment, report []byte, mime string, now time.Time) (string, error) {
	if exp.ID == "" {
		return "", fmt.Errorf("experiment: StoreReport requires experiment id")
	}
	key := fmt.Sprintf("reports/%s/%s.json", now.UTC().Format("2006-01"), exp.ID)
	if err := s.blobs.Put(ctx, key, report, mime); err != nil {
		return "", fmt.Errorf("experiment: store report: %w", err)
	}
	return key, nil
}
