// Package workflowreplay implements GCT's workflow-evidence lens over Test
// Genie. Selection is by stable artifact kind, never by producer phase.
package workflowreplay

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	wrpb "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/workflowreplay"
	"github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/workflowreplay/workflowreplay_v1connect"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

const (
	workflowVideoKind = "workflow.video"
	defaultLimit      = 10
)

type RunsClient interface {
	ListRuns(ctx context.Context, scenario string, limit int) ([]*runspb.RunInfo, error)
	GetRun(ctx context.Context, scenario, runID string) (*runspb.RunInfo, error)
	ListRunArtifacts(ctx context.Context, scenario, runID string, kinds []string) ([]*runspb.ArtifactRef, error)
}

type Server struct{ runs RunsClient }
type Deps struct{ Runs RunsClient }

func NewServer(d Deps) *Server { return &Server{runs: d.Runs} }

func NewHandler(d Deps, opts ...connect.HandlerOption) (string, http.Handler) {
	return workflowreplay_v1connect.NewWorkflowReplayServiceHandler(NewServer(d), opts...)
}

func toSummary(run *runspb.RunInfo) *wrpb.RunSummary {
	return &wrpb.RunSummary{
		RunId: run.GetRunId(), Status: run.GetStatus(), StartedAt: run.GetStartedAt(),
		CompletedAt: run.GetCompletedAt(), GitSha: run.GetGitSha(), GitBranch: run.GetGitBranch(), GitDirty: run.GetGitDirty(),
	}
}

func (s *Server) ListRecentRuns(ctx context.Context, req *connect.Request[wrpb.ListRecentRunsRequest]) (*connect.Response[wrpb.ListRecentRunsResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = defaultLimit
	}
	runs, err := s.runs.ListRuns(ctx, scenario, limit*3)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	out := make([]*wrpb.RunSummary, 0, limit)
	for _, run := range runs {
		artifacts, err := s.runs.ListRunArtifacts(ctx, scenario, run.GetRunId(), []string{workflowVideoKind})
		if err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
		if len(artifacts) == 0 {
			continue
		}
		out = append(out, toSummary(run))
		if len(out) >= limit {
			break
		}
	}
	return connect.NewResponse(&wrpb.ListRecentRunsResponse{Runs: out}), nil
}

func (s *Server) GetRunDetail(ctx context.Context, req *connect.Request[wrpb.GetRunDetailRequest]) (*connect.Response[wrpb.GetRunDetailResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	runID := strings.TrimSpace(req.Msg.GetRunId())
	if scenario == "" || runID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario and run_id are required"))
	}
	run, err := s.runs.GetRun(ctx, scenario, runID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	artifacts, err := s.runs.ListRunArtifacts(ctx, scenario, runID, []string{workflowVideoKind})
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(&wrpb.GetRunDetailResponse{Run: toSummary(run), Artifacts: artifacts}), nil
}
