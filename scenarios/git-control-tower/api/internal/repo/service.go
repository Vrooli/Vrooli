// Package repo owns the Connect-RPC repo identity surface. In Tier 1
// it exposes only GetRepoStatus, focused on worktree-awareness fields.
// Pre-existing REST `/api/v1/repo/status` continues to serve legacy
// consumers without modification.
//
// Testing rule: tests substitute the worktree.Inspector fake; no real
// git is invoked.
package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"git-control-tower/internal/worktree"
)

// Status is the in-process shape returned by Service.GetRepoStatus.
type Status struct {
	Branch   string
	Detached bool
	Identity worktree.Identity
}

// Service is the repo identity domain. It wraps the worktree.Inspector
// seam — repo status is fundamentally a worktree-identity question once
// REST status has been migrated. Today only the Tier-1 fields surface.
type Service struct {
	inspector worktree.Inspector
}

// NewService wires the seam.
func NewService(insp worktree.Inspector) *Service {
	return &Service{inspector: insp}
}

// GetRepoStatus returns Status for repoPath.
func (s *Service) GetRepoStatus(ctx context.Context, repoPath string) (Status, error) {
	if s.inspector == nil {
		return Status{}, errors.New("repo: inspector seam not wired")
	}
	if strings.TrimSpace(repoPath) == "" {
		return Status{}, fmt.Errorf("%w: repo_path required", ErrInvalid)
	}
	id, err := s.inspector.IdentifyPath(ctx, repoPath)
	if err != nil {
		return Status{}, err
	}
	return Status{
		Branch:   id.Branch,
		Detached: id.Detached,
		Identity: id,
	}, nil
}

// ErrInvalid is the InvalidArgument-class error for this domain.
var ErrInvalid = errors.New("repo: invalid request")

// ErrorCodeFor maps domain errors to connect codes.
func ErrorCodeFor(err error) connect.Code {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrInvalid):
		return connect.CodeInvalidArgument
	default:
		return connect.CodeInternal
	}
}

// ToConnectError converts a domain error to a typed *connect.Error.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	if connectErr := new(connect.Error); errors.As(err, &connectErr) {
		return err
	}
	return connect.NewError(ErrorCodeFor(err), err)
}
