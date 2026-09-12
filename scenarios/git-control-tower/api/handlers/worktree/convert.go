package worktree

import (
	"path/filepath"

	"git-control-tower/internal/worktree"

	worktreev1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree"
)

// domainToProto converts a domain worktree.Worktree into the proto wire
// representation. Defined here (not in internal/worktree) so the domain
// package stays free of proto imports.
func domainToProto(w worktree.Worktree) *worktreev1.Worktree {
	return &worktreev1.Worktree{
		Path:           w.Path,
		Name:           w.Name,
		HeadCommit:     w.HeadCommit,
		Branch:         w.Branch,
		Detached:       w.Detached,
		Locked:         w.Locked,
		LockReason:     w.LockReason,
		Prunable:       w.Prunable,
		PrunableReason: w.PrunableReason,
		IsMain:         w.IsMain,
	}
}

// createInputFromProto converts the validated proto request into the
// domain CreateInput shape. protovalidate-style annotations on the
// proto guard min_len; the oneof source still needs runtime resolution.
func createInputFromProto(req *worktreev1.CreateWorktreeRequest) worktree.CreateInput {
	in := worktree.CreateInput{
		RepoPath:        req.GetRepoPath(),
		NewWorktreePath: req.GetNewWorktreePath(),
		Force:           req.GetForce(),
		Track:           req.GetTrack(),
	}
	switch src := req.GetSource().(type) {
	case *worktreev1.CreateWorktreeRequest_ExistingBranch:
		in.ExistingBranch = src.ExistingBranch
	case *worktreev1.CreateWorktreeRequest_NewBranch:
		in.NewBranchName = src.NewBranch.GetName()
		in.NewBranchStart = src.NewBranch.GetStartPoint()
	case *worktreev1.CreateWorktreeRequest_Commit:
		in.Commit = src.Commit
	}
	return in
}

func baseName(p string) string {
	if p == "" {
		return ""
	}
	return filepath.Base(p)
}
