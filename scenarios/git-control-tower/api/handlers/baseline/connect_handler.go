// Package baseline (handlers) implements the BaselinesService Connect-RPC
// service. It translates proto messages to/from the domain shapes in
// internal/baseline and delegates orchestration to baseline.Service.
//
// Testing rule: handler tests construct *Server with fake adapters + a fake
// RepoResolver. No real git, test-genie, or scenario-auditor is invoked.
package baseline

import (
	"context"
	"errors"
	"log"
	"net/http"

	"connectrpc.com/connect"

	bl "git-control-tower/internal/baseline"
	"git-control-tower/internal/git"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
	"github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines/baselines_v1connect"
)

// RepoResolver maps an optional explicit repoID (0 = active repo) to a concrete
// (repoID, repoDir) pair the baseline service operates on.
type RepoResolver interface {
	Resolve(ctx context.Context, repoID int64) (int64, string, error)
}

// Server implements baselines_v1connect.BaselinesServiceHandler.
type Server struct {
	svc    *bl.Service
	repos  RepoResolver
	logger *log.Logger
}

// Deps wires the Connect server.
type Deps struct {
	Service *bl.Service
	Repos   RepoResolver
	Logger  *log.Logger
}

// NewServer builds a Server.
func NewServer(d Deps) *Server {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &Server{svc: d.Service, repos: d.Repos, logger: d.Logger}
}

// NewHandler returns the (procedure-prefix, http.Handler) pair the router
// mounts.
func NewHandler(d Deps, opts ...connect.HandlerOption) (string, http.Handler) {
	return baselines_v1connect.NewBaselinesServiceHandler(NewServer(d), opts...)
}

// resolveTarget resolves the repo and the effective branch. When requested
// branch is empty and allBranches is false, the current branch is derived from
// the working tree.
func (s *Server) resolveTarget(ctx context.Context, repoID int64, branch string, allBranches bool) (int64, string, string, error) {
	rid, repoDir, err := s.repos.Resolve(ctx, repoID)
	if err != nil {
		return 0, "", "", err
	}
	if allBranches {
		return rid, repoDir, "", nil
	}
	if branch == "" {
		st, gerr := git.Capture(ctx, repoDir)
		if gerr != nil {
			return 0, "", "", gerr
		}
		branch = bl.ResolveStorageBranch(st)
	}
	return rid, repoDir, branch, nil
}

func (s *Server) CreateBaseline(ctx context.Context, req *connect.Request[baselinesv1.CreateBaselineRequest]) (*connect.Response[baselinesv1.CreateBaselineResponse], error) {
	m := req.Msg
	rid, repoDir, err := s.repos.Resolve(ctx, m.GetRepoId())
	if err != nil {
		return nil, s.wrap("CreateBaseline", err)
	}
	res, err := s.svc.Create(ctx, bl.CreateRequest{
		RepoID: rid, RepoDir: repoDir, Scenario: m.GetScenario(), Name: m.GetName(),
		Branch: m.GetBranch(), Include: m.GetInclude(), Fast: m.GetFast(),
		Capture: false, CreatedBy: m.GetCreatedBy(), Reason: m.GetReason(),
	})
	if err != nil {
		return nil, s.wrap("CreateBaseline", err)
	}
	return connect.NewResponse(&baselinesv1.CreateBaselineResponse{
		Baseline: manifestToProto(res.Manifest), Skipped: res.Skipped, DirtyWarning: res.DirtyWarning,
	}), nil
}

func (s *Server) SnapshotForBaseline(ctx context.Context, req *connect.Request[baselinesv1.SnapshotForBaselineRequest]) (*connect.Response[baselinesv1.SnapshotForBaselineResponse], error) {
	m := req.Msg
	rid, repoDir, err := s.repos.Resolve(ctx, m.GetRepoId())
	if err != nil {
		return nil, s.wrap("SnapshotForBaseline", err)
	}
	res, err := s.svc.Create(ctx, bl.CreateRequest{
		RepoID: rid, RepoDir: repoDir, Scenario: m.GetScenario(), Name: m.GetName(),
		Branch: m.GetBranch(), Include: m.GetInclude(), Fast: m.GetFast(),
		Capture: true, CreatedBy: m.GetCreatedBy(), Reason: m.GetReason(),
	})
	if err != nil {
		return nil, s.wrap("SnapshotForBaseline", err)
	}
	return connect.NewResponse(&baselinesv1.SnapshotForBaselineResponse{
		Baseline: manifestToProto(res.Manifest), Skipped: res.Skipped, DirtyWarning: res.DirtyWarning,
	}), nil
}

func (s *Server) GetBaseline(ctx context.Context, req *connect.Request[baselinesv1.GetBaselineRequest]) (*connect.Response[baselinesv1.GetBaselineResponse], error) {
	m := req.Msg
	rid, _, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("GetBaseline", err)
	}
	manifest, err := s.svc.Get(ctx, rid, m.GetScenario(), branch, m.GetName())
	if err != nil {
		return nil, s.wrap("GetBaseline", err)
	}
	return connect.NewResponse(&baselinesv1.GetBaselineResponse{Baseline: manifestToProto(manifest)}), nil
}

func (s *Server) ListBaselines(ctx context.Context, req *connect.Request[baselinesv1.ListBaselinesRequest]) (*connect.Response[baselinesv1.ListBaselinesResponse], error) {
	m := req.Msg
	rid, _, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), m.GetAllBranches())
	if err != nil {
		return nil, s.wrap("ListBaselines", err)
	}
	manifests, err := s.svc.List(ctx, rid, m.GetScenario(), branch)
	if err != nil {
		return nil, s.wrap("ListBaselines", err)
	}
	out := &baselinesv1.ListBaselinesResponse{Baselines: make([]*baselinesv1.BaselineManifest, 0, len(manifests))}
	for _, mf := range manifests {
		out.Baselines = append(out.Baselines, manifestToProto(mf))
	}
	return connect.NewResponse(out), nil
}

func (s *Server) DiffBaseline(ctx context.Context, req *connect.Request[baselinesv1.DiffBaselineRequest]) (*connect.Response[baselinesv1.DiffBaselineResponse], error) {
	m := req.Msg
	rid, repoDir, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("DiffBaseline", err)
	}
	res, err := s.svc.Diff(ctx, rid, repoDir, m.GetScenario(), branch, m.GetName(), m.GetSurface())
	if err != nil {
		return nil, s.wrap("DiffBaseline", err)
	}
	out := &baselinesv1.DiffBaselineResponse{
		Baseline:   manifestToProto(res.Manifest),
		CurrentGit: gitToProto(res.CurrentGit),
		Staleness: &baselinesv1.Staleness{
			CommitsSince: int32(res.Staleness.CommitsSince),
			FilesChanged: int32(res.Staleness.FilesChanged),
			LikelyStale:  res.Staleness.LikelyStale,
		},
		Verdict: string(res.Verdict),
	}
	for _, d := range res.Surfaces {
		out.Surfaces = append(out.Surfaces, surfaceDiffToProto(d))
	}
	return connect.NewResponse(out), nil
}

func (s *Server) DeleteBaseline(ctx context.Context, req *connect.Request[baselinesv1.DeleteBaselineRequest]) (*connect.Response[baselinesv1.DeleteBaselineResponse], error) {
	m := req.Msg
	rid, _, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("DeleteBaseline", err)
	}
	if err := s.svc.Delete(ctx, rid, m.GetScenario(), branch, m.GetName()); err != nil {
		return nil, s.wrap("DeleteBaseline", err)
	}
	return connect.NewResponse(&baselinesv1.DeleteBaselineResponse{Deleted: true}), nil
}

func (s *Server) EditBaseline(ctx context.Context, req *connect.Request[baselinesv1.EditBaselineRequest]) (*connect.Response[baselinesv1.EditBaselineResponse], error) {
	m := req.Msg
	rid, _, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("EditBaseline", err)
	}
	manifest, err := s.svc.Edit(ctx, bl.EditRequest{
		RepoID: rid, Scenario: m.GetScenario(), Branch: branch, Name: m.GetName(),
		Surface: m.GetSurface(), PinRunID: m.GetPinRunId(), Reason: m.GetReason(),
	})
	if err != nil {
		return nil, s.wrap("EditBaseline", err)
	}
	return connect.NewResponse(&baselinesv1.EditBaselineResponse{Baseline: manifestToProto(manifest)}), nil
}

// wrap maps domain errors to Connect codes and logs internal ones.
func (s *Server) wrap(op string, err error) error {
	switch {
	case errors.Is(err, bl.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, bl.ErrAlreadyExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	s.logger.Printf("baselines.%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, err)
}
