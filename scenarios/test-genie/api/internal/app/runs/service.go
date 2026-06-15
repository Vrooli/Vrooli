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

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/runmanager"
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// executionPlanner previews a run plan to derive its ETA and surface validation
// errors synchronously (bad preset/phase) before a run is started.
type executionPlanner interface {
	Preview(ctx context.Context, req orchestrator.SuiteExecutionRequest) (*execution.ExecutionPlanPreview, error)
}

// Service implements runs_v1connect.RunsServiceHandler. Beyond the read-only run
// index surface, it owns the durable run lifecycle (StartRun/FollowRun/WaitRun/
// AbortRun/GetRunStatus) by delegating to the run manager.
type Service struct {
	scenariosRoot string
	runManager    *runmanager.Manager
	planner       executionPlanner
}

// NewService returns a Service. scenariosRoot resolves each request's scenario
// slug to its physical directory so the run index can be addressed. runManager
// and planner power the durable run-lifecycle RPCs.
func NewService(scenariosRoot string, runManager *runmanager.Manager, planner executionPlanner) *Service {
	return &Service{scenariosRoot: scenariosRoot, runManager: runManager, planner: planner}
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

// ListRunVideos enumerates the recorded workflow videos for a run. The binary
// content is served by the REST artifact route; this returns the relative-path
// handles that route consumes.
func (s *Service) ListRunVideos(ctx context.Context, req *connect.Request[runspb.ListRunVideosRequest]) (*connect.Response[runspb.ListRunVideosResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	runID := strings.TrimSpace(req.Msg.GetRunId())
	if runID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("run_id is required"))
	}
	videos, err := sharedartifacts.ListRunVideos(dir, runID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*runspb.RunVideo, 0, len(videos))
	for _, v := range videos {
		out = append(out, &runspb.RunVideo{
			Workflow:  v.Workflow,
			RelPath:   v.RelPath,
			SizeBytes: v.SizeBytes,
		})
	}
	return connect.NewResponse(&runspb.ListRunVideosResponse{Videos: out}), nil
}

// ListRunVisuals enumerates the per-page UI smoke visual artifacts (screenshot +
// optional video) a run captured under the baseline capture profile. The binary
// content is served by the REST artifact route; this returns the structured
// page set + rel-path handles git-control-tower diffs at the metadata level.
func (s *Service) ListRunVisuals(ctx context.Context, req *connect.Request[runspb.ListRunVisualsRequest]) (*connect.Response[runspb.ListRunVisualsResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	runID := strings.TrimSpace(req.Msg.GetRunId())
	if runID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("run_id is required"))
	}
	visuals, err := sharedartifacts.ListRunVisuals(dir, runID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*runspb.RunVisual, 0, len(visuals))
	for _, v := range visuals {
		out = append(out, &runspb.RunVisual{
			Page:                v.Page,
			Label:               v.Label,
			ScreenshotRelPath:   v.ScreenshotRelPath,
			VideoRelPath:        v.VideoRelPath,
			ScreenshotSizeBytes: v.ScreenshotSizeBytes,
		})
	}
	return connect.NewResponse(&runspb.ListRunVisualsResponse{Visuals: out}), nil
}

// ResolveArtifact maps a (scenario, runID, run-relative path) to an absolute
// filesystem path under the run's artifact root, rejecting traversal. Used by
// the REST binary artifact route to stream videos without exposing
// scenariosRoot. Returns os.ErrNotExist when the resolved file is missing.
func (s *Service) ResolveArtifact(scenario, runID, relPath string) (string, error) {
	dir, err := s.scenarioDir(scenario)
	if err != nil {
		return "", err
	}
	abs, err := sharedartifacts.ResolveRunArtifact(dir, strings.TrimSpace(runID), relPath)
	if err != nil {
		return "", err
	}
	if _, statErr := os.Stat(abs); statErr != nil {
		return "", statErr
	}
	return abs, nil
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
