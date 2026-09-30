package main

import (
	"context"
	"os/exec"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/connectx"

	baselineH "git-control-tower/handlers/baseline"
	branchH "git-control-tower/handlers/branch"
	evidenceH "git-control-tower/handlers/evidence"
	repoH "git-control-tower/handlers/repo"
	worktreeH "git-control-tower/handlers/worktree"
	"git-control-tower/internal/policygate"
	repoD "git-control-tower/internal/repo"
	worktreeD "git-control-tower/internal/worktree"
	advisoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory/advisory_v1connect"
	auditorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/auditor/auditor_v1connect"
	authconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/auth/auth_v1connect"
	humancontrolconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/human_control/human_control_v1connect"
	reviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/review/review_v1connect"
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
// by the typed BranchService list handler for worktree-aware enrichment.
func newWorktreeInspector() worktreeD.Inspector {
	return worktreeD.NewGitInspector(gctGitRunner{gitPath: "git"})
}

// mountConnectHandlers registers WorktreeService and RepoService Connect
// handlers on the existing mux router under their generated procedure
// paths. Typed callers do not depend on the legacy REST status route.
//
// Worktree, Repo, and Branch are typed proto+Connect domains; remaining
// transport migration follows the same wiring shape.
func (s *Server) mountConnectHandlers() {
	runner := gctGitRunner{gitPath: "git"}
	inspector := worktreeD.NewGitInspector(runner)
	mutator := worktreeD.NewGitMutator(runner)

	worktreeSvc := worktreeD.NewService(inspector, mutator)
	repoSvc := repoD.NewService(inspector)

	// Connect policy gate. Caller headers remain attribution only; mutating
	// procedures require a verified principal and exact server-issued intent.
	// The legacy AgentAccess setting remains useful for diagnostics, but cannot
	// authorize a human mutation.
	policyOpt := connect.WithInterceptors(policygate.NewInterceptor(s.policy.Policy, policygate.StdAuditLogger()))

	wtPath, wtHandler := worktreeH.NewHandler(worktreeH.Deps{Service: worktreeSvc}, policyOpt)
	repoPath, repoHandler := repoH.NewHandler(repoH.Deps{
		Service:              repoSvc,
		ListRepositories:     s.listRepositoriesConnect,
		GetActiveRepository:  s.getActiveRepositoryConnect,
		SetActiveRepository:  s.setActiveRepositoryConnect,
		OpenRepository:       s.openRepositoryConnect,
		CloneRepository:      s.cloneRepositoryConnect,
		RemoveRepository:     s.removeRepositoryConnect,
		GetRepoStatus:        s.getRepoStatusConnect,
		GetRepoDiff:          s.getRepoDiffConnect,
		GetRepoGroups:        s.getRepoGroupsConnect,
		GetSyncStatus:        s.getSyncStatusConnect,
		GetRepoHistory:       s.getRepoHistoryConnect,
		GetApprovedChanges:   s.getApprovedChangesConnect,
		GetProvenance:        s.getProvenanceConnect,
		GetBlame:             s.getBlameConnect,
		SearchProvenance:     s.searchProvenanceConnect,
		GetFiles:             s.getFilesConnect,
		GetDirectoryContents: s.getDirectoryContentsConnect,
		GetRelatedFiles:      s.getRelatedFilesConnect,
		SearchContent:        s.searchContentConnect,
		DeletePath:           s.deletePathConnect,
		SaveFileContent:      s.saveFileContentConnect,
		DiscardFiles:         s.discardFilesConnect,
		IgnorePath:           s.ignorePathConnect,
		PushToRemote:         s.pushToRemoteConnect,
		InspectPushSafety:    s.inspectPushSafetyConnect,
		GetPushRecovery:      s.getPushRecoveryConnect,
		PreparePushRecovery:  s.preparePushRecoveryConnect,
		PullFromRemote:       s.pullFromRemoteConnect,
		RunUpstreamAction:    s.runUpstreamActionConnect,
		GetGroupingRules:     s.getGroupingRulesConnect,
		SaveGroupingRules:    s.saveGroupingRulesConnect,
		GetGitignoreHealth:   s.getGitignoreHealthConnect,
		MoveGitignoreEntry:   s.moveGitignoreEntryConnect,
		GetTrackedBinaries:   s.getTrackedBinariesConnect,
		UntrackBinary:        s.untrackBinaryConnect,
		GetPrecommitConfig:   s.getPrecommitConfigConnect,
		SavePrecommitConfig:  s.savePrecommitConfigConnect,
		RunPrecommit:         s.runPrecommitConnect,
		StageFiles:           s.stageFilesConnect,
		UnstageFiles:         s.unstageFilesConnect,
		CreateCommit:         s.createCommitConnect,
		ListCredentials:      s.listCredentialsConnect,
		SaveCredential:       s.saveCredentialConnect,
		DeleteCredential:     s.deleteCredentialConnect,
		TestCredential:       s.testCredentialConnect,
		UpdateRemoteURL:      s.updateRemoteURLConnect,
		ListSSHKeys:          s.listSSHKeysConnect,
		GenerateSSHKey:       s.generateSSHKeyConnect,
		GetSSHPublicKey:      s.getSSHPublicKeyConnect,
		TestSSHConnection:    s.testSSHConnectionConnect,
		DeleteSSHKey:         s.deleteSSHKeyConnect,
	}, policyOpt)
	branchPath, branchHandler := branchH.NewHandler(branchH.Deps{
		List:    s.listBranchesConnect,
		Create:  s.createBranchConnect,
		Switch:  s.switchBranchConnect,
		Publish: s.publishBranchConnect,
	}, policyOpt)

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
	humanControlPath, humanControlHandler := humancontrolconnect.NewHumanControlServiceHandler(s, policyOpt)
	advisoryPath, advisoryHandler := advisoryconnect.NewAdvisoryServiceHandler(advisoryConnectServer{}, policyOpt)
	auditorPath, auditorHandler := auditorconnect.NewAuditorServiceHandler(auditorConnectServer{server: s}, policyOpt)
	reviewPath, reviewHandler := reviewconnect.NewReviewServiceHandler(reviewConnectServer{server: s}, policyOpt)
	authPath, authHandler := authconnect.NewAuthServiceHandler(authConnectServer{}, policyOpt)

	connectx.RegisterServices(s.router,
		connectx.ServiceMount{Path: wtPath, Handler: wtHandler},
		connectx.ServiceMount{Path: repoPath, Handler: repoHandler},
		connectx.ServiceMount{Path: branchPath, Handler: branchHandler},
		connectx.ServiceMount{Path: baselinePath, Handler: baselineHandler},
		connectx.ServiceMount{Path: evidencePath, Handler: evidenceHandler},
		connectx.ServiceMount{Path: humanControlPath, Handler: humanControlHandler},
		connectx.ServiceMount{Path: advisoryPath, Handler: advisoryHandler},
		connectx.ServiceMount{Path: authPath, Handler: authHandler},
		connectx.ServiceMount{Path: auditorPath, Handler: auditorHandler},
		connectx.ServiceMount{Path: reviewPath, Handler: reviewHandler},
	)
}
