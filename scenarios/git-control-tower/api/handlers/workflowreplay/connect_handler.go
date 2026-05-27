// Package workflowreplay implements the WorkflowReplayService Connect-RPC, a
// thin proxy over test-genie's RunsService (Plan B Decision 3). It filters runs
// to the playbooks phase and maps test-genie's run records to the compact shape
// the GCT Workflows tab consumes, so the UI never calls test-genie directly.
//
// Binary video bytes are NOT served here; a GCT REST route streams them.
//
// Testing rule: handler tests inject a fake RunsClient; no real test-genie or
// network is touched.
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

// playbooksPhase is the test-genie phase whose runs the Workflows tab surfaces.
const playbooksPhase = "playbooks"

// defaultLimit caps ListRecentRuns when the caller passes 0.
const defaultLimit = 10

// RunsClient is the narrow seam over test-genie's RunsService that this proxy
// needs. The concrete implementation lives in the flat main package and is
// wired in connect_wiring.go.
type RunsClient interface {
	ListRuns(ctx context.Context, scenario string, limit int) ([]*runspb.RunInfo, error)
	GetRun(ctx context.Context, scenario, runID string) (*runspb.RunInfo, error)
	ListRunVideos(ctx context.Context, scenario, runID string) ([]*runspb.RunVideo, error)
}

// Server implements workflowreplay_v1connect.WorkflowReplayServiceHandler.
type Server struct {
	runs RunsClient
}

// Deps wires the Connect server.
type Deps struct {
	Runs RunsClient
}

// NewServer builds a Server.
func NewServer(d Deps) *Server { return &Server{runs: d.Runs} }

// NewHandler returns the (procedure-prefix, http.Handler) pair the router mounts.
func NewHandler(d Deps, opts ...connect.HandlerOption) (string, http.Handler) {
	return workflowreplay_v1connect.NewWorkflowReplayServiceHandler(NewServer(d), opts...)
}

func playbooksInfo(run *runspb.RunInfo) (*runspb.PhaseInfo, bool) {
	for _, p := range run.GetPhases() {
		if p.GetName() == playbooksPhase {
			return p, true
		}
	}
	return nil, false
}

func toSummary(run *runspb.RunInfo) *wrpb.RunSummary {
	s := &wrpb.RunSummary{
		RunId:       run.GetRunId(),
		Status:      run.GetStatus(),
		StartedAt:   run.GetStartedAt(),
		CompletedAt: run.GetCompletedAt(),
		GitSha:      run.GetGitSha(),
		GitBranch:   run.GetGitBranch(),
		GitDirty:    run.GetGitDirty(),
	}
	if p, ok := playbooksInfo(run); ok {
		s.PlaybooksStatus = p.GetStatus()
		s.PlaybooksDurationSeconds = p.GetDurationSeconds()
	}
	return s
}

// ListRecentRuns returns recent runs that include a playbooks phase, newest-first.
func (s *Server) ListRecentRuns(ctx context.Context, req *connect.Request[wrpb.ListRecentRunsRequest]) (*connect.Response[wrpb.ListRecentRunsResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = defaultLimit
	}

	// Over-fetch then filter: not every run includes playbooks, so ask for more
	// than `limit` to fill the page after filtering.
	runs, err := s.runs.ListRuns(ctx, scenario, limit*3)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	out := make([]*wrpb.RunSummary, 0, limit)
	for _, r := range runs {
		if _, ok := playbooksInfo(r); !ok {
			continue
		}
		out = append(out, toSummary(r))
		if len(out) >= limit {
			break
		}
	}
	return connect.NewResponse(&wrpb.ListRecentRunsResponse{Runs: out}), nil
}

// GetRunDetail returns one run plus its recorded workflow videos.
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
	videos, err := s.runs.ListRunVideos(ctx, scenario, runID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	out := &wrpb.GetRunDetailResponse{Run: toSummary(run)}
	for _, v := range videos {
		out.Videos = append(out.Videos, &wrpb.WorkflowVideo{
			Workflow:  v.GetWorkflow(),
			RelPath:   v.GetRelPath(),
			SizeBytes: v.GetSizeBytes(),
		})
	}
	return connect.NewResponse(out), nil
}
