package runs

import "context"

// Service is the application-level seam over Repository. It exists so
// callers (handlers, pipeline recorders, CLI) depend on a small surface
// rather than the SQLite-shaped Repository interface.
type Service struct {
	repo Repository
}

// NewService wires a Service over the provided repository.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// Record persists a single verification run and returns the inserted row
// (with id/timestamps filled in if the caller omitted them).
func (s *Service) Record(ctx context.Context, run Run) (Run, error) {
	return s.repo.Insert(ctx, run)
}

// Get returns the run with the given id or ErrNotFound.
func (s *Service) Get(ctx context.Context, id string) (Run, error) {
	return s.repo.Get(ctx, id)
}

// List returns runs filtered by q (FlowID optional, Limit defaulted to
// 50 if non-positive) newest-first.
func (s *Service) List(ctx context.Context, q ListQuery) ([]Run, error) {
	return s.repo.List(ctx, q)
}
