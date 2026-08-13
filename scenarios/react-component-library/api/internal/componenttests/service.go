package componenttests

import "context"

type Service struct {
	runner  Runner
	reports Repository
}

func NewService(runner Runner, reports Repository) *Service {
	return &Service{runner: runner, reports: reports}
}

func (s *Service) Run(ctx context.Context, request Request) (Report, error) {
	report, err := s.runner.Run(ctx, request)
	if err != nil {
		return Report{}, err
	}
	if err := s.reports.Save(ctx, report); err != nil {
		return Report{}, err
	}
	return report, nil
}
func (s *Service) Get(ctx context.Context, id string) (Report, error) { return s.reports.Get(ctx, id) }
func (s *Service) Rerun(ctx context.Context, id string) (Report, error) {
	previous, err := s.reports.Get(ctx, id)
	if err != nil {
		return Report{}, err
	}
	return s.Run(ctx, Request{ComponentID: previous.RootComponentID, Version: previous.RootVersion, IncludeClosure: previous.IncludeClosure})
}

func (s *Service) List(ctx context.Context, componentID, version string, limit int) ([]Report, error) {
	return s.reports.List(ctx, componentID, version, limit)
}
