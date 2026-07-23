package main

import (
	"context"
	"os/exec"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/connectx"

	baselineH "git-control-tower/handlers/baseline"
	evidenceH "git-control-tower/handlers/evidence"
	repoH "git-control-tower/handlers/repo"
	worktreeH "git-control-tower/handlers/worktree"
	"git-control-tower/internal/policygate"
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

	// Agent-access policy gate. Connect interceptor enforces the
	// configured AgentAccess policy (default: confirm — agents must
	// pass X-Vrooli-Authorized: true). See policygate.Decide for the
	// matrix and `.vrooli/config.json` for the operator surface.
	policyOpt := connect.WithInterceptors(policygate.NewInterceptor(s.policy.Policy, policygate.StdAuditLogger()))

	wtPath, wtHandler := worktreeH.NewHandler(worktreeH.Deps{Service: worktreeSvc}, policyOpt)
	repoPath, repoHandler := repoH.NewHandler(repoH.Deps{Service: repoSvc}, policyOpt)

	// Baselines: cross-surface review baseline substrate. The handler resolves
	// the active repo and delegates capture/diff to baseline.Service.
	baselinePath, baselineHandler := baselineH.NewHandler(baselineH.Deps{
		Service: s.baselineService,
		Repos:   baselineRepoResolver{repos: s.repos},
	}, policyOpt)

	// Evidence: the shared descriptor-aware run and artifact surface consumed by
	// Tests, Screenshots, Workflows, Overview, Baselines, and Agent. Selection is
	// based on captured metadata and open artifact kinds, never phase keys.
	evidencePath, evidenceHandler := evidenceH.NewHandler(evidenceH.Deps{
		Runs: newEvidenceRunsClient(30 * time.Second),
	}, policyOpt)

	connectx.RegisterServices(s.router,
		connectx.ServiceMount{Path: wtPath, Handler: wtHandler},
		connectx.ServiceMount{Path: repoPath, Handler: repoHandler},
		connectx.ServiceMount{Path: baselinePath, Handler: baselineHandler},
		connectx.ServiceMount{Path: evidencePath, Handler: evidenceHandler},
	)
}
