// Package repo (handlers) implements the RepoService Connect-RPC
// service. In Tier 1, it exposes only GetRepoStatus, carrying the
// worktree-awareness fields the UI status header and CLI repo-status
// command consume.
package repo

import (
	"context"
	"log"
	"net/http"

	"connectrpc.com/connect"

	"git-control-tower/internal/repo"

	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
	repoconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo/repo_v1connect"
)

// Server implements repoconnect.RepoServiceHandler.
type Server struct {
	svc    *repo.Service
	logger *log.Logger
}

// Deps wires dependencies.
type Deps struct {
	Service *repo.Service
	Logger  *log.Logger
}

// NewServer constructs a Server.
func NewServer(d Deps) *Server {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &Server{svc: d.Service, logger: d.Logger}
}

// NewHandler returns the procedure prefix and http.Handler for mounting.
func NewHandler(d Deps) (string, http.Handler) {
	return repoconnect.NewRepoServiceHandler(NewServer(d))
}

func (s *Server) GetRepoStatus(ctx context.Context, req *connect.Request[repov1.GetRepoStatusRequest]) (*connect.Response[repov1.GetRepoStatusResponse], error) {
	status, err := s.svc.GetRepoStatus(ctx, req.Msg.GetRepoPath())
	if err != nil {
		connectErr := repo.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			s.logger.Printf("repo.GetRepoStatus: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&repov1.GetRepoStatusResponse{
		Branch:   status.Branch,
		Detached: status.Detached,
		Worktree: &repov1.WorktreeIdentity{
			IsLinkedWorktree:    status.Identity.IsLinkedWorktree,
			CommonRepoRoot:      status.Identity.CommonRepoRoot,
			WorktreeName:        status.Identity.WorktreeName,
			WorktreeHead:        status.Identity.WorktreeHead,
			LinkedWorktreeCount: int32(status.Identity.LinkedWorktreeCount),
		},
	}), nil
}
