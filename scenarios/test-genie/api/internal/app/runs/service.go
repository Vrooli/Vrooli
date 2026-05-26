// Package runs hosts the Connect-RPC RunsService handler that exposes
// test-genie's append-only run index (coverage/runs.index.json) to external
// callers (the test-genie CLI and git-control-tower baseline adapters). It is a
// thin wrapper over internal/shared/runs.
package runs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"

	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// Service implements runs_v1connect.RunsServiceHandler.
type Service struct {
	scenariosRoot string
}

// NewService returns a Service. scenariosRoot resolves each request's scenario
// slug to its physical directory so the run index can be addressed.
func NewService(scenariosRoot string) *Service {
	return &Service{scenariosRoot: scenariosRoot}
}

func (s *Service) scenarioDir(scenario string) (string, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	return filepath.Join(s.scenariosRoot, scenario), nil
}

// ListRuns enumerates runs for a scenario, newest-first.
func (s *Service) ListRuns(ctx context.Context, req *connect.Request[runspb.ListRunsRequest]) (*connect.Response[runspb.ListRunsResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	records, err := sharedruns.NewIndex(dir).List()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	statusFilter := strings.TrimSpace(req.Msg.GetStatus())
	limit := int(req.Msg.GetLimit())
	out := make([]*runspb.RunInfo, 0, len(records))
	for _, r := range records {
		if statusFilter != "" && r.Status != statusFilter {
			continue
		}
		out = append(out, toRunInfo(r))
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return connect.NewResponse(&runspb.ListRunsResponse{Runs: out}), nil
}

// GetRun returns a single run record.
func (s *Service) GetRun(ctx context.Context, req *connect.Request[runspb.GetRunRequest]) (*connect.Response[runspb.GetRunResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	rec, err := sharedruns.NewIndex(dir).Find(strings.TrimSpace(req.Msg.GetRunId()))
	if err != nil {
		return nil, mapRunError(err)
	}
	return connect.NewResponse(&runspb.GetRunResponse{Run: toRunInfo(rec)}), nil
}

// DeleteRun removes a run's artifacts and index entry.
func (s *Service) DeleteRun(ctx context.Context, req *connect.Request[runspb.DeleteRunRequest]) (*connect.Response[runspb.DeleteRunResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	if err := sharedruns.DeleteRun(dir, strings.TrimSpace(req.Msg.GetRunId()), req.Msg.GetForce()); err != nil {
		return nil, mapRunError(err)
	}
	return connect.NewResponse(&runspb.DeleteRunResponse{Deleted: true}), nil
}

// PinRun protects a run from retention GC.
func (s *Service) PinRun(ctx context.Context, req *connect.Request[runspb.PinRunRequest]) (*connect.Response[runspb.PinRunResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	pinnedBy := strings.TrimSpace(req.Msg.GetPinnedBy())
	if pinnedBy == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("pinned_by is required"))
	}
	idx := sharedruns.NewIndex(dir)
	runID := strings.TrimSpace(req.Msg.GetRunId())
	if err := idx.Pin(runID, sharedruns.PinRecord{
		PinnedBy: pinnedBy,
		PinnedAt: time.Now().UTC(),
		Reason:   strings.TrimSpace(req.Msg.GetReason()),
	}); err != nil {
		return nil, mapRunError(err)
	}
	rec, err := idx.Find(runID)
	if err != nil {
		return nil, mapRunError(err)
	}
	return connect.NewResponse(&runspb.PinRunResponse{Run: toRunInfo(rec)}), nil
}

// UnpinRun removes a consumer's pin.
func (s *Service) UnpinRun(ctx context.Context, req *connect.Request[runspb.UnpinRunRequest]) (*connect.Response[runspb.UnpinRunResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	idx := sharedruns.NewIndex(dir)
	runID := strings.TrimSpace(req.Msg.GetRunId())
	if err := idx.Unpin(runID, strings.TrimSpace(req.Msg.GetPinnedBy())); err != nil {
		return nil, mapRunError(err)
	}
	rec, err := idx.Find(runID)
	if err != nil {
		return nil, mapRunError(err)
	}
	return connect.NewResponse(&runspb.UnpinRunResponse{Run: toRunInfo(rec)}), nil
}

// CompareRuns classifies per-phase differences between two runs.
func (s *Service) CompareRuns(ctx context.Context, req *connect.Request[runspb.CompareRunsRequest]) (*connect.Response[runspb.CompareRunsResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	idx := sharedruns.NewIndex(dir)
	recA, err := idx.Find(strings.TrimSpace(req.Msg.GetRunIdA()))
	if err != nil {
		return nil, mapRunError(err)
	}
	recB, err := idx.Find(strings.TrimSpace(req.Msg.GetRunIdB()))
	if err != nil {
		return nil, mapRunError(err)
	}
	resp := comparePhases(recA, recB, strings.TrimSpace(req.Msg.GetPhase()))
	return connect.NewResponse(resp), nil
}

// GetPhaseArtifact returns the raw phase-results JSON for a run+phase.
func (s *Service) GetPhaseArtifact(ctx context.Context, req *connect.Request[runspb.GetPhaseArtifactRequest]) (*connect.Response[runspb.GetPhaseArtifactResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	phase := strings.TrimSpace(req.Msg.GetPhase())
	if phase == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("phase is required"))
	}
	path := sharedartifacts.PhaseResultsPath(dir, strings.TrimSpace(req.Msg.GetRunId()), phase+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("phase artifact not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&runspb.GetPhaseArtifactResponse{
		Content:     string(data),
		ContentType: "application/json",
	}), nil
}

func mapRunError(err error) error {
	switch {
	case errors.Is(err, sharedruns.ErrRunNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, sharedruns.ErrRunPinned):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
