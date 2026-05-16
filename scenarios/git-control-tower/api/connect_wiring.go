package main

import (
	"context"
	"os/exec"

	"github.com/vrooli/api-core/connectx"

	repoH "git-control-tower/handlers/repo"
	worktreeH "git-control-tower/handlers/worktree"
	repoD "git-control-tower/internal/repo"
	worktreeD "git-control-tower/internal/worktree"
)

// gctGitRunner adapts os/exec to the narrow worktree.GitRunner seam.
// The existing flat-package ExecGitRunner has a similar shape but a
// different (broader) interface; we declare a tiny adapter here so the
// new domain package does not have to import or expose the legacy
// interface. Future incremental migration may consolidate these.
type gctGitRunner struct{ gitPath string }

func (g gctGitRunner) Run(ctx context.Context, repoDir string, args ...string) ([]byte, error) {
	bin := g.gitPath
	if bin == "" {
		bin = "git"
	}
	full := append([]string{"-C", repoDir}, args...)
	return exec.CommandContext(ctx, bin, full...).CombinedOutput()
}

// newWorktreeInspector builds a fresh worktree.Inspector wired to the
// production GitRunner adapter. Used both by mountConnectHandlers and
// by REST handlers that need worktree-aware enrichment (branch list).
func newWorktreeInspector() worktreeD.Inspector {
	return worktreeD.NewGitInspector(gctGitRunner{gitPath: "git"})
}

// mountConnectHandlers registers WorktreeService and RepoService Connect
// handlers on the existing mux router under their generated procedure
// paths. The existing flat-package REST routes are untouched.
//
// Worktree is the first proto+Connect domain in GCT; future domains
// follow the same wiring shape.
func (s *Server) mountConnectHandlers() {
	runner := gctGitRunner{gitPath: "git"}
	inspector := worktreeD.NewGitInspector(runner)
	mutator := worktreeD.NewGitMutator(runner)

	worktreeSvc := worktreeD.NewService(inspector, mutator)
	repoSvc := repoD.NewService(inspector)

	wtPath, wtHandler := worktreeH.NewHandler(worktreeH.Deps{Service: worktreeSvc})
	repoPath, repoHandler := repoH.NewHandler(repoH.Deps{Service: repoSvc})

	connectx.RegisterServices(s.router,
		connectx.ServiceMount{Path: wtPath, Handler: wtHandler},
		connectx.ServiceMount{Path: repoPath, Handler: repoHandler},
	)
}
