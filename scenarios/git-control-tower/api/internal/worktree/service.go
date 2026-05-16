package worktree

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
)

// Service is the worktree domain's business logic layer. Connect-RPC
// handlers translate proto requests into Service calls and translate
// Service errors back into typed connect.Error values via
// ErrorCodeFor.
//
// Service has no transport imports apart from connect.Code for the
// error-mapping helper, which lives in this package so handlers don't
// have to repeat the mapping.
type Service struct {
	inspector Inspector
	mutator   Mutator
}

// NewService wires the seams. Either may be nil for read-only or
// write-only test setups; Service guards every dereference with a
// validation error.
func NewService(insp Inspector, mut Mutator) *Service {
	return &Service{inspector: insp, mutator: mut}
}

// List returns every worktree of the repo containing repoPath.
func (s *Service) List(ctx context.Context, repoPath string) ([]Worktree, error) {
	if err := requireInspector(s); err != nil {
		return nil, err
	}
	if strings.TrimSpace(repoPath) == "" {
		return nil, fmt.Errorf("%w: repo_path required", ErrInvalid)
	}
	return s.inspector.List(ctx, repoPath)
}

// Get returns the single worktree matching worktreePath.
func (s *Service) Get(ctx context.Context, repoPath, worktreePath string) (Worktree, error) {
	if err := requireInspector(s); err != nil {
		return Worktree{}, err
	}
	if strings.TrimSpace(repoPath) == "" {
		return Worktree{}, fmt.Errorf("%w: repo_path required", ErrInvalid)
	}
	if strings.TrimSpace(worktreePath) == "" {
		return Worktree{}, fmt.Errorf("%w: worktree_path required", ErrInvalid)
	}
	wts, err := s.inspector.List(ctx, repoPath)
	if err != nil {
		return Worktree{}, err
	}
	target := filepath.Clean(worktreePath)
	for _, w := range wts {
		if filepath.Clean(w.Path) == target {
			return w, nil
		}
	}
	return Worktree{}, fmt.Errorf("%w: %s", ErrNotFound, worktreePath)
}

// Identify returns the Tier-1 passive-awareness facts for repoPath.
func (s *Service) Identify(ctx context.Context, repoPath string) (Identity, error) {
	if err := requireInspector(s); err != nil {
		return Identity{}, err
	}
	if strings.TrimSpace(repoPath) == "" {
		return Identity{}, fmt.Errorf("%w: repo_path required", ErrInvalid)
	}
	return s.inspector.IdentifyPath(ctx, repoPath)
}

// ClaimedBranches returns the map used to enrich the branch list with
// `checked_out_in_worktree`.
func (s *Service) ClaimedBranches(ctx context.Context, repoPath string) (map[string]string, error) {
	if err := requireInspector(s); err != nil {
		return nil, err
	}
	if strings.TrimSpace(repoPath) == "" {
		return nil, fmt.Errorf("%w: repo_path required", ErrInvalid)
	}
	return s.inspector.ClaimedBranches(ctx, repoPath)
}

// Create runs CreateWorktree validation and dispatches to Mutator.Add.
// Service does not honor dry-run; the Connect handler short-circuits
// before calling Service when X-Dry-Run is set.
func (s *Service) Create(ctx context.Context, input CreateInput) (Worktree, error) {
	if err := requireMutator(s); err != nil {
		return Worktree{}, err
	}
	if err := validateCreateInput(input); err != nil {
		return Worktree{}, err
	}
	return s.mutator.Add(ctx, input)
}

// Remove deletes a worktree. Service refuses to remove the main
// worktree even with force; that mirrors git's own behavior and gives
// callers a stable InvalidArgument vs Internal classification.
func (s *Service) Remove(ctx context.Context, repoPath, worktreePath string, force bool) error {
	if err := requireMutator(s); err != nil {
		return err
	}
	if strings.TrimSpace(repoPath) == "" {
		return fmt.Errorf("%w: repo_path required", ErrInvalid)
	}
	if strings.TrimSpace(worktreePath) == "" {
		return fmt.Errorf("%w: worktree_path required", ErrInvalid)
	}
	// Use the inspector to refuse main-worktree removal early so the
	// caller never relies on git's exit code.
	if s.inspector != nil {
		got, err := s.Get(ctx, repoPath, worktreePath)
		if err == nil && got.IsMain {
			return fmt.Errorf("%w: cannot remove main worktree %s", ErrInvalid, worktreePath)
		}
		// Get-side NotFound is a real signal — propagate it.
		if err != nil && errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return s.mutator.Remove(ctx, repoPath, worktreePath, force)
}

// Lock applies a lock.
func (s *Service) Lock(ctx context.Context, input LockInput) (Worktree, error) {
	if err := requireMutator(s); err != nil {
		return Worktree{}, err
	}
	if err := validateLockInput(input); err != nil {
		return Worktree{}, err
	}
	return s.mutator.Lock(ctx, input)
}

// Unlock clears a lock.
func (s *Service) Unlock(ctx context.Context, repoPath, worktreePath string) (Worktree, error) {
	if err := requireMutator(s); err != nil {
		return Worktree{}, err
	}
	if strings.TrimSpace(repoPath) == "" {
		return Worktree{}, fmt.Errorf("%w: repo_path required", ErrInvalid)
	}
	if strings.TrimSpace(worktreePath) == "" {
		return Worktree{}, fmt.Errorf("%w: worktree_path required", ErrInvalid)
	}
	return s.mutator.Unlock(ctx, repoPath, worktreePath)
}

// Move relocates a worktree.
func (s *Service) Move(ctx context.Context, input MoveInput) (Worktree, error) {
	if err := requireMutator(s); err != nil {
		return Worktree{}, err
	}
	if err := validateMoveInput(input); err != nil {
		return Worktree{}, err
	}
	return s.mutator.Move(ctx, input)
}

// Prune runs git worktree prune.
func (s *Service) Prune(ctx context.Context, input PruneInput) (PruneResult, error) {
	if err := requireMutator(s); err != nil {
		return PruneResult{}, err
	}
	if strings.TrimSpace(input.RepoPath) == "" {
		return PruneResult{}, fmt.Errorf("%w: repo_path required", ErrInvalid)
	}
	return s.mutator.Prune(ctx, input)
}

// ErrorCodeFor maps a domain error to its connect.Code. Used by the
// handler package so the mapping has exactly one source of truth.
func ErrorCodeFor(err error) connect.Code {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrInvalid):
		return connect.CodeInvalidArgument
	case errors.Is(err, ErrNotFound):
		return connect.CodeNotFound
	case errors.Is(err, ErrLocked), errors.Is(err, ErrDirty):
		return connect.CodeFailedPrecondition
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

func validateCreateInput(in CreateInput) error {
	if strings.TrimSpace(in.RepoPath) == "" {
		return fmt.Errorf("%w: repo_path required", ErrInvalid)
	}
	if strings.TrimSpace(in.NewWorktreePath) == "" {
		return fmt.Errorf("%w: new_worktree_path required", ErrInvalid)
	}
	switch in.Mode() {
	case CreateModeUnspecified:
		return fmt.Errorf("%w: exactly one of existing_branch, new_branch, or commit is required", ErrInvalid)
	case CreateModeNewBranch:
		if strings.TrimSpace(in.NewBranchName) == "" {
			return fmt.Errorf("%w: new_branch.name required", ErrInvalid)
		}
	}
	if in.Track && in.Mode() != CreateModeNewBranch {
		return fmt.Errorf("%w: --track only applies when creating a new branch", ErrInvalid)
	}
	return nil
}

func validateLockInput(in LockInput) error {
	if strings.TrimSpace(in.RepoPath) == "" {
		return fmt.Errorf("%w: repo_path required", ErrInvalid)
	}
	if strings.TrimSpace(in.WorktreePath) == "" {
		return fmt.Errorf("%w: worktree_path required", ErrInvalid)
	}
	return nil
}

func validateMoveInput(in MoveInput) error {
	if strings.TrimSpace(in.RepoPath) == "" {
		return fmt.Errorf("%w: repo_path required", ErrInvalid)
	}
	if strings.TrimSpace(in.WorktreePath) == "" {
		return fmt.Errorf("%w: worktree_path required", ErrInvalid)
	}
	if strings.TrimSpace(in.NewWorktreePath) == "" {
		return fmt.Errorf("%w: new_worktree_path required", ErrInvalid)
	}
	if filepath.Clean(in.WorktreePath) == filepath.Clean(in.NewWorktreePath) {
		return fmt.Errorf("%w: new_worktree_path must differ from worktree_path", ErrInvalid)
	}
	return nil
}

func requireInspector(s *Service) error {
	if s.inspector == nil {
		return fmt.Errorf("%w: inspector seam not wired", ErrInvalid)
	}
	return nil
}

func requireMutator(s *Service) error {
	if s.mutator == nil {
		return fmt.Errorf("%w: mutator seam not wired", ErrInvalid)
	}
	return nil
}
