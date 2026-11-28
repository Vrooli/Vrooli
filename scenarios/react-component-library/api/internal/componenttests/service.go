package componenttests

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

import "react-component-library/internal/components"

type Service struct {
	runner  Runner
	reports Repository
}

// ExecutorFreshness is an optional stable identity for the execution policy
// that produced browser evidence. Changing it separates reports made under
// different executor behavior while preserving the source and closure key.
type ExecutorFreshness interface {
	ReportFreshnessKey() string
}

func NewService(runner Runner, reports Repository) *Service {
	return &Service{runner: runner, reports: reports}
}

func (s *Service) Run(ctx context.Context, request Request) (Report, error) {
	report, _, err := s.RunWithReuse(ctx, request)
	return report, err
}

func (s *Service) RunWithReuse(ctx context.Context, request Request) (Report, bool, error) {
	if expected, ok := s.expectedReport(ctx, request); ok {
		if report, err := s.reports.Get(ctx, expected); err == nil {
			// Only a report explicitly blocked by an unavailable executor is
			// transient. Other blocked reports (for example incomplete BAS
			// evidence) remain reusable for the unchanged revision.
			if !reportHasUnavailableExecutor(report) {
				if err := s.verifyDurableReport(ctx, report); err != nil {
					return Report{}, false, err
				}
				return report, true, nil
			}
		}
	}
	report, err := s.runner.Run(ctx, request)
	if err != nil {
		return Report{}, false, err
	}
	if key := executorFreshnessKey(s.runner.Executor); key != "" {
		report.ID = withExecutorFreshness(report.ID, key)
	}
	if err := s.reports.Save(ctx, report); err != nil {
		return Report{}, false, err
	}
	if err := s.verifyDurableReport(ctx, report); err != nil {
		return Report{}, false, err
	}
	return report, false, nil
}

// verifyDurableReport prevents a successful run response from claiming a
// report that the configured repository cannot return under its exact ID.
func (s *Service) verifyDurableReport(ctx context.Context, expected Report) error {
	actual, err := s.reports.Get(ctx, expected.ID)
	if err != nil {
		// Deliberately do not unwrap the repository error: sql.ErrNoRows here
		// means persistence failed its postcondition and must map to Internal,
		// not to the public Get endpoint's ordinary NotFound response.
		return fmt.Errorf("component test report %q could not be read back after save or reuse: %v", expected.ID, err)
	}
	if actual.ID != expected.ID || actual.RootLibraryID != expected.RootLibraryID || actual.RootVersion != expected.RootVersion || actual.IncludeClosure != expected.IncludeClosure || actual.SourceRevision != expected.SourceRevision || actual.Verdict != expected.Verdict {
		return fmt.Errorf("component test report %q readback identity mismatch", expected.ID)
	}
	return nil
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
	expected, ok := s.expectedReport(ctx, request)
	if !ok {
		return Report{}, false, nil
	}
	report, err := s.reports.Get(ctx, expected)
	if err != nil {
		return Report{}, false, nil
	}
	return report, true, nil
}

func (s *Service) expectedReport(ctx context.Context, request Request) (string, bool) {
	expected, ok := s.runner.ExpectedReport(ctx, request)
	if !ok {
		return "", false
	}
	if key := executorFreshnessKey(s.runner.Executor); key != "" {
		expected = withExecutorFreshness(expected, key)
	}
	return expected, true
}

func executorFreshnessKey(executor StoryExecutor) string {
	keyed, ok := executor.(ExecutorFreshness)
	if !ok {
		return ""
	}
	return strings.TrimSpace(keyed.ReportFreshnessKey())
}

func withExecutorFreshness(reportID, freshness string) string {
	sum := sha256.Sum256([]byte(freshness))
	return reportID + "_" + hex.EncodeToString(sum[:12])
}

// ExpectedReport returns the deterministic report identity when the runner
// has a revision authority. A missing authority deliberately disables reuse;
// it is safer to execute than to claim freshness without a folded revision.
func (r Runner) ExpectedReport(ctx context.Context, request Request) (string, bool) {
	// A page uses live application inputs. A source digest alone cannot prove
	// unchanged API state, so a prior page capture is never reused as a new run.
	if strings.HasPrefix(request.ComponentID, components.PageStoryPrefix) {
		return "", false
	}
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
	if key := executorFreshnessKey(s.runner.Executor); key != "" {
		report.ID = withExecutorFreshness(report.ID, key)
	}
	if err := s.reports.Save(ctx, report); err != nil {
		return Report{}, err
	}
	return report, nil
}

func (s *Service) List(ctx context.Context, componentID, version string, limit int) ([]Report, error) {
	return s.reports.List(ctx, componentID, version, limit)
}
