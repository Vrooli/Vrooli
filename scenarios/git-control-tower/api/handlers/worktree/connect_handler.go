// Package worktree (handlers) implements the WorktreeService Connect-RPC
// service. It translates proto messages to/from the domain shapes in
// internal/worktree and delegates business logic to worktree.Service.
//
// Testing rule: handler tests construct *Server with FakeInspector /
// FakeMutator fakes from internal/worktree/mocks. NO real git is ever
// invoked at any layer.
package worktree

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"git-control-tower/internal/worktree"

	worktreev1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree"
	worktreeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree/worktree_v1connect"
)

// DryRunHeader is the standard X-Dry-Run header cli-core sets when
// --dry-run is passed. Handlers short-circuit mutating methods when
// this header is "true".
const DryRunHeader = "X-Dry-Run"

// Server implements worktreeconnect.WorktreeServiceHandler.
type Server struct {
	svc    *worktree.Service
	logger interface{ Printf(string, ...any) }
	audit  func(MutationAuditEvent)
}

// MutationAuditEvent is the bounded, domain-neutral record passed to GCT's
// audit owner after a worktree mutation succeeds. Dry-run attempts are
// recorded with DryRun=true but never imply that Git changed state.
type MutationAuditEvent struct {
	Operation string
	RepoPath  string
	Paths     []string
	Branch    string
	Mode      string
	Reason    string
	Force     bool
	DryRun    bool
}

const (
	AuditWorktreeCreated  = "worktree_created"
	AuditWorktreeRemoved  = "worktree_removed"
	AuditWorktreeLocked   = "worktree_locked"
	AuditWorktreeUnlocked = "worktree_unlocked"
	AuditWorktreeMoved    = "worktree_moved"
	AuditWorktreePruned   = "worktree_pruned"
)

// Deps wires the dependencies the Connect server needs.
type Deps struct {
	Service *worktree.Service
	Logger  interface{ Printf(string, ...any) }
	Audit   func(MutationAuditEvent)
}

// NewServer returns a fully-constructed Server.
func NewServer(d Deps) *Server {
	if d.Logger == nil {
		d.Logger = quietLogger{}
	}
	return &Server{svc: d.Service, logger: d.Logger, audit: d.Audit}
}

type quietLogger struct{}

func (quietLogger) Printf(string, ...any) {}

// NewHandler returns the (procedure-prefix, http.Handler) pair the
// router mounts. Splits NewServer + NewHandler so tests can build a
// Server in-process without instantiating an HTTP transport. Extra
// connect.HandlerOption values (e.g. interceptor wiring from main.go)
// are passed through to NewWorktreeServiceHandler.
func NewHandler(d Deps, opts ...connect.HandlerOption) (string, http.Handler) {
	return worktreeconnect.NewWorktreeServiceHandler(NewServer(d), opts...)
}

// isDryRun returns true when the X-Dry-Run header is set to "true" on
// the inbound request.
func isDryRun(header http.Header) bool {
	return header.Get(DryRunHeader) == "true"
}

func (s *Server) recordMutation(event MutationAuditEvent) {
	if s.audit != nil {
		s.audit(event)
	}
}

func createModeLabel(mode worktree.CreateMode) string {
	switch mode {
	case worktree.CreateModeExistingBranch:
		return "existing_branch"
	case worktree.CreateModeNewBranch:
		return "new_branch"
	case worktree.CreateModeDetachedCommit:
		return "detached_commit"
	default:
		return "unspecified"
	}
}

// --- generated WorktreeServiceHandler methods ---

func (s *Server) ListWorktrees(ctx context.Context, req *connect.Request[worktreev1.ListWorktreesRequest]) (*connect.Response[worktreev1.ListWorktreesResponse], error) {
	wts, err := s.svc.List(ctx, req.Msg.GetRepoPath())
	if err != nil {
		return nil, s.wrap("ListWorktrees", err)
	}
	out := &worktreev1.ListWorktreesResponse{Worktrees: make([]*worktreev1.Worktree, 0, len(wts))}
	for _, w := range wts {
		out.Worktrees = append(out.Worktrees, domainToProto(w))
	}
	return connect.NewResponse(out), nil
}

func (s *Server) GetWorktree(ctx context.Context, req *connect.Request[worktreev1.GetWorktreeRequest]) (*connect.Response[worktreev1.GetWorktreeResponse], error) {
	w, err := s.svc.Get(ctx, req.Msg.GetRepoPath(), req.Msg.GetWorktreePath())
	if err != nil {
		return nil, s.wrap("GetWorktree", err)
	}
	return connect.NewResponse(&worktreev1.GetWorktreeResponse{Worktree: domainToProto(w)}), nil
}

func (s *Server) CreateWorktree(ctx context.Context, req *connect.Request[worktreev1.CreateWorktreeRequest]) (*connect.Response[worktreev1.CreateWorktreeResponse], error) {
	in := createInputFromProto(req.Msg)
	if isDryRun(req.Header()) {
		// Run validation but skip the Mutator. Synthesize a realistic
		// proto Worktree so consumers can verify shape end-to-end.
		if err := previewCreate(in); err != nil {
			return nil, s.wrap("CreateWorktree (dry-run)", err)
		}
		s.recordMutation(MutationAuditEvent{Operation: AuditWorktreeCreated, RepoPath: in.RepoPath, Paths: []string{in.NewWorktreePath}, Branch: in.ExistingBranch + in.NewBranchName, Mode: createModeLabel(in.Mode()), DryRun: true})
		return connect.NewResponse(&worktreev1.CreateWorktreeResponse{
			Worktree: domainToProto(worktree.Worktree{
				Path:       in.NewWorktreePath,
				Name:       baseName(in.NewWorktreePath),
				Branch:     in.ExistingBranch + in.NewBranchName,
				Detached:   in.Mode() == worktree.CreateModeDetachedCommit,
				HeadCommit: in.Commit,
			}),
			DryRun: true,
		}), nil
	}
	w, err := s.svc.Create(ctx, in)
	if err != nil {
		return nil, s.wrap("CreateWorktree", err)
	}
	s.recordMutation(MutationAuditEvent{Operation: AuditWorktreeCreated, RepoPath: in.RepoPath, Paths: []string{w.Path}, Branch: w.Branch, Mode: createModeLabel(in.Mode())})
	return connect.NewResponse(&worktreev1.CreateWorktreeResponse{
		Worktree: domainToProto(w),
		DryRun:   false,
	}), nil
}

func (s *Server) RemoveWorktree(ctx context.Context, req *connect.Request[worktreev1.RemoveWorktreeRequest]) (*connect.Response[worktreev1.RemoveWorktreeResponse], error) {
	if isDryRun(req.Header()) {
		if err := previewRemove(req.Msg.GetRepoPath(), req.Msg.GetWorktreePath(), req.Msg.GetForce()); err != nil {
			return nil, s.wrap("RemoveWorktree (dry-run)", err)
		}
		s.recordMutation(MutationAuditEvent{Operation: AuditWorktreeRemoved, RepoPath: req.Msg.GetRepoPath(), Paths: []string{req.Msg.GetWorktreePath()}, Force: req.Msg.GetForce(), DryRun: true})
		return connect.NewResponse(&worktreev1.RemoveWorktreeResponse{DryRun: true}), nil
	}
	if err := s.svc.Remove(ctx, req.Msg.GetRepoPath(), req.Msg.GetWorktreePath(), req.Msg.GetForce()); err != nil {
		return nil, s.wrap("RemoveWorktree", err)
	}
	s.recordMutation(MutationAuditEvent{Operation: AuditWorktreeRemoved, RepoPath: req.Msg.GetRepoPath(), Paths: []string{req.Msg.GetWorktreePath()}, Force: req.Msg.GetForce()})
	return connect.NewResponse(&worktreev1.RemoveWorktreeResponse{DryRun: false}), nil
}

func (s *Server) LockWorktree(ctx context.Context, req *connect.Request[worktreev1.LockWorktreeRequest]) (*connect.Response[worktreev1.LockWorktreeResponse], error) {
	in := worktree.LockInput{
		RepoPath:     req.Msg.GetRepoPath(),
		WorktreePath: req.Msg.GetWorktreePath(),
		Reason:       req.Msg.GetReason(),
	}
	if isDryRun(req.Header()) {
		if err := previewLock(in); err != nil {
			return nil, s.wrap("LockWorktree (dry-run)", err)
		}
		s.recordMutation(MutationAuditEvent{Operation: AuditWorktreeLocked, RepoPath: in.RepoPath, Paths: []string{in.WorktreePath}, Reason: in.Reason, DryRun: true})
		return connect.NewResponse(&worktreev1.LockWorktreeResponse{
			Worktree: domainToProto(worktree.Worktree{
				Path:       in.WorktreePath,
				Name:       baseName(in.WorktreePath),
				Locked:     true,
				LockReason: in.Reason,
			}),
			DryRun: true,
		}), nil
	}
	w, err := s.svc.Lock(ctx, in)
	if err != nil {
		return nil, s.wrap("LockWorktree", err)
	}
	s.recordMutation(MutationAuditEvent{Operation: AuditWorktreeLocked, RepoPath: in.RepoPath, Paths: []string{w.Path}, Reason: in.Reason})
	return connect.NewResponse(&worktreev1.LockWorktreeResponse{Worktree: domainToProto(w)}), nil
}

func (s *Server) UnlockWorktree(ctx context.Context, req *connect.Request[worktreev1.UnlockWorktreeRequest]) (*connect.Response[worktreev1.UnlockWorktreeResponse], error) {
	if isDryRun(req.Header()) {
		if err := previewUnlock(req.Msg.GetRepoPath(), req.Msg.GetWorktreePath()); err != nil {
			return nil, s.wrap("UnlockWorktree (dry-run)", err)
		}
		s.recordMutation(MutationAuditEvent{Operation: AuditWorktreeUnlocked, RepoPath: req.Msg.GetRepoPath(), Paths: []string{req.Msg.GetWorktreePath()}, DryRun: true})
		return connect.NewResponse(&worktreev1.UnlockWorktreeResponse{
			Worktree: domainToProto(worktree.Worktree{
				Path: req.Msg.GetWorktreePath(),
				Name: baseName(req.Msg.GetWorktreePath()),
			}),
			DryRun: true,
		}), nil
	}
	w, err := s.svc.Unlock(ctx, req.Msg.GetRepoPath(), req.Msg.GetWorktreePath())
	if err != nil {
		return nil, s.wrap("UnlockWorktree", err)
	}
	s.recordMutation(MutationAuditEvent{Operation: AuditWorktreeUnlocked, RepoPath: req.Msg.GetRepoPath(), Paths: []string{w.Path}})
	return connect.NewResponse(&worktreev1.UnlockWorktreeResponse{Worktree: domainToProto(w)}), nil
}

func (s *Server) MoveWorktree(ctx context.Context, req *connect.Request[worktreev1.MoveWorktreeRequest]) (*connect.Response[worktreev1.MoveWorktreeResponse], error) {
	in := worktree.MoveInput{
		RepoPath:        req.Msg.GetRepoPath(),
		WorktreePath:    req.Msg.GetWorktreePath(),
		NewWorktreePath: req.Msg.GetNewWorktreePath(),
	}
	if isDryRun(req.Header()) {
		if err := previewMove(in); err != nil {
			return nil, s.wrap("MoveWorktree (dry-run)", err)
		}
		s.recordMutation(MutationAuditEvent{Operation: AuditWorktreeMoved, RepoPath: in.RepoPath, Paths: []string{in.WorktreePath, in.NewWorktreePath}, DryRun: true})
		return connect.NewResponse(&worktreev1.MoveWorktreeResponse{
			Worktree: domainToProto(worktree.Worktree{Path: in.NewWorktreePath, Name: baseName(in.NewWorktreePath)}),
			DryRun:   true,
		}), nil
	}
	w, err := s.svc.Move(ctx, in)
	if err != nil {
		return nil, s.wrap("MoveWorktree", err)
	}
	s.recordMutation(MutationAuditEvent{Operation: AuditWorktreeMoved, RepoPath: in.RepoPath, Paths: []string{in.WorktreePath, w.Path}, Branch: w.Branch})
	return connect.NewResponse(&worktreev1.MoveWorktreeResponse{Worktree: domainToProto(w)}), nil
}

func (s *Server) PruneWorktrees(ctx context.Context, req *connect.Request[worktreev1.PruneWorktreesRequest]) (*connect.Response[worktreev1.PruneWorktreesResponse], error) {
	in := worktree.PruneInput{
		RepoPath:   req.Msg.GetRepoPath(),
		Reason:     req.Msg.GetReason(),
		ReportOnly: req.Msg.GetReportOnly(),
	}
	if isDryRun(req.Header()) {
		if err := previewPrune(in); err != nil {
			return nil, s.wrap("PruneWorktrees (dry-run)", err)
		}
		// Translate the transport-level dry-run into the in-process
		// ReportOnly flag so the response is faithful: pruned_paths
		// remains empty (we did nothing), DryRun is true.
		s.recordMutation(MutationAuditEvent{Operation: AuditWorktreePruned, RepoPath: in.RepoPath, Reason: in.Reason, DryRun: true})
		return connect.NewResponse(&worktreev1.PruneWorktreesResponse{DryRun: true}), nil
	}
	res, err := s.svc.Prune(ctx, in)
	if err != nil {
		return nil, s.wrap("PruneWorktrees", err)
	}
	s.recordMutation(MutationAuditEvent{Operation: AuditWorktreePruned, RepoPath: in.RepoPath, Paths: append([]string(nil), res.PrunedPaths...), Reason: in.Reason, DryRun: in.ReportOnly})
	return connect.NewResponse(&worktreev1.PruneWorktreesResponse{
		PrunedPaths: res.PrunedPaths,
		DryRun:      in.ReportOnly,
	}), nil
}

func (s *Server) wrap(op string, err error) error {
	connectErr := worktree.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		s.logger.Printf("worktree.%s: %v", op, err)
	}
	return connectErr
}

// previewCreate runs the same validation Service.Create does, but never
// calls the Mutator. Reused by the dry-run code path.
func previewCreate(in worktree.CreateInput) error {
	tmp := worktree.NewService(nil, &noopMutator{})
	_, err := tmp.Create(context.Background(), in)
	// Only validation failures matter here; tmp.Create succeeds through
	// the noopMutator otherwise.
	if errors.Is(err, worktree.ErrInvalid) {
		return err
	}
	return nil
}

func previewRemove(repoPath, worktreePath string, force bool) error {
	return worktree.NewService(nil, &noopMutator{}).Remove(context.Background(), repoPath, worktreePath, force)
}

func previewLock(in worktree.LockInput) error {
	_, err := worktree.NewService(nil, &noopMutator{}).Lock(context.Background(), in)
	return err
}

func previewUnlock(repoPath, worktreePath string) error {
	_, err := worktree.NewService(nil, &noopMutator{}).Unlock(context.Background(), repoPath, worktreePath)
	return err
}

func previewMove(in worktree.MoveInput) error {
	_, err := worktree.NewService(nil, &noopMutator{}).Move(context.Background(), in)
	return err
}

func previewPrune(in worktree.PruneInput) error {
	in.ReportOnly = true
	_, err := worktree.NewService(nil, &noopMutator{}).Prune(context.Background(), in)
	return err
}

type noopMutator struct{}

func (noopMutator) Add(context.Context, worktree.CreateInput) (worktree.Worktree, error) {
	return worktree.Worktree{}, nil
}
func (noopMutator) Remove(context.Context, string, string, bool) error { return nil }
func (noopMutator) Lock(context.Context, worktree.LockInput) (worktree.Worktree, error) {
	return worktree.Worktree{}, nil
}

func (noopMutator) Unlock(context.Context, string, string) (worktree.Worktree, error) {
	return worktree.Worktree{}, nil
}

func (noopMutator) Move(context.Context, worktree.MoveInput) (worktree.Worktree, error) {
	return worktree.Worktree{}, nil
}

func (noopMutator) Prune(context.Context, worktree.PruneInput) (worktree.PruneResult, error) {
	return worktree.PruneResult{}, nil
}

var _ worktree.Mutator = noopMutator{}
