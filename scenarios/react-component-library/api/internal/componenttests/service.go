package componenttests

import (
	"context"
	"strings"
)

import "react-component-library/internal/components"

type Service struct {
	runner  Runner
	reports Repository
}

func NewService(runner Runner, reports Repository) *Service {
	return &Service{runner: runner, reports: reports}
}

func (s *Service) Run(ctx context.Context, request Request) (Report, error) {
	report, _, err := s.RunWithReuse(ctx, request)
	return report, err
}

func (s *Service) RunWithReuse(ctx context.Context, request Request) (Report, bool, error) {
	if expected, ok := s.runner.ExpectedReport(ctx, request); ok {
		if report, err := s.reports.Get(ctx, expected); err == nil {
			// Only a report explicitly blocked by an unavailable executor is
			// transient. Other blocked reports (for example incomplete BAS
			// evidence) remain reusable for the unchanged revision.
			if !reportHasUnavailableExecutor(report) {
				return report, true, nil
			}
		}
	}
	report, err := s.runner.Run(ctx, request)
	if err != nil {
		return Report{}, false, err
	}
	if err := s.reports.Save(ctx, report); err != nil {
		return Report{}, false, err
	}
	return report, false, nil
}

func reportHasUnavailableExecutor(report Report) bool {
	for _, result := range report.Results {
		if result.Verdict == VerdictBlocked && strings.HasPrefix(result.Message, "story executor is unavailable") {
			return true
		}
	}
	return false
}

// Reusable returns the report for the current folded revision without ever
// dispatching a browser run. A missing or stale report is reported as found=false.
func (s *Service) Reusable(ctx context.Context, request Request) (Report, bool, error) {
	expected, ok := s.runner.ExpectedReport(ctx, request)
	if !ok {
		return Report{}, false, nil
	}
	report, err := s.reports.Get(ctx, expected)
	if err != nil {
		return Report{}, false, nil
	}
	return report, true, nil
}

// ExpectedReport returns the deterministic report identity when the runner
// has a revision authority. A missing authority deliberately disables reuse;
// it is safer to execute than to claim freshness without a folded revision.
func (r Runner) ExpectedReport(ctx context.Context, request Request) (string, bool) {
	if r.Revision == nil || r.Assets == nil || request.Version == "" {
		return "", false
	}
	closure, err := components.ResolveDependencyClosure(ctx, r.Assets, request.ComponentID, request.Version)
	if err != nil || len(closure) == 0 {
		return "", false
	}
	root := closure[len(closure)-1].Asset
	revision, err := r.Revision(ctx, root.CatalogID, request.Version)
	if err != nil {
		return "", false
	}
	return reportID(Report{RootLibraryID: root.LibraryID, RootVersion: request.Version, IncludeClosure: request.IncludeClosure, SourceRevision: revision}), true
}
func (s *Service) Get(ctx context.Context, id string) (Report, error) { return s.reports.Get(ctx, id) }
func (s *Service) Rerun(ctx context.Context, id string) (Report, error) {
	previous, err := s.reports.Get(ctx, id)
	if err != nil {
		return Report{}, err
	}
	request := Request{ComponentID: previous.RootComponentID, Version: previous.RootVersion, IncludeClosure: previous.IncludeClosure}
	report, err := s.runner.Run(ctx, request)
	if err != nil {
		return Report{}, err
	}
	if err := s.reports.Save(ctx, report); err != nil {
		return Report{}, err
	}
	return report, nil
}

func (s *Service) List(ctx context.Context, componentID, version string, limit int) ([]Report, error) {
	return s.reports.List(ctx, componentID, version, limit)
}
