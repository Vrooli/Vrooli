package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"

	"connectrpc.com/connect"
	"git-control-tower/internal/policygate"
	"git-control-tower/internal/pushsafety"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
)

func (s *Server) inspectPushSafetyConnect(ctx context.Context, req *repov1.InspectPushSafetyRequest) (*repov1.PushSafetyReport, error) {
	repo, e := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if e != nil {
		return nil, connectFileError(e)
	}
	deps := PushPullDeps{Git: s.git, RepoDir: repo.Path, CredStore: s.credStore}
	remote, branch := resolvePushTarget(ctx, deps, repo.Path, req.GetRemote(), req.GetBranch())
	r := s.git.InspectPushSafety(ctx, repo.Path, remote, branch, lookupCredential(ctx, deps, remote))
	return pushSafetyProto(r), nil
}

func pushSafetyProto(r pushsafety.Report) *repov1.PushSafetyReport {
	out := &repov1.PushSafetyReport{Complete: r.Complete, State: r.State, Reason: r.Reason, Head: r.Head, Base: r.Base, Remote: r.Remote, Branch: r.Branch, Limit: r.Limit, Commits: r.Commits, Fingerprint: r.Fingerprint, CanPrepare: r.CanPrepare, RecoveryReason: r.RecoveryReason}
	out.StagedComplete, out.StagedReason = r.StagedComplete, r.StagedReason
	for _, f := range r.StagedFiles {
		out.StagedFiles = append(out.StagedFiles, &repov1.PushSafetyFile{Oid: f.OID, Bytes: f.Bytes, Paths: f.Paths, Blocked: f.Blocked})
	}
	for _, f := range r.Files {
		out.Files = append(out.Files, &repov1.PushSafetyFile{Oid: f.OID, Bytes: f.Bytes, Paths: f.Paths, Commits: f.Commits, Blocked: f.Blocked})
	}
	return out
}

func (s *Server) preparePushRecoveryConnect(ctx context.Context, req *repov1.PreparePushRecoveryRequest) (*repov1.PushRecoveryArtifact, error) {
	if e := requireHumanPrincipal(ctx, "prepare push recovery"); e != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, e)
	}
	if req.GetFingerprint() == "" || req.GetIntentId() == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Review and confirm the exact recovery preview first."))
	}
	principal, _ := policygate.PrincipalFromContext(ctx)
	// Bind the intent to the report fingerprint (including source, destination,
	// remote tip and approved path set), rather than trusting request parameters.
	preview, e := s.prepareMutationWithContext(ctx, req.GetRepositoryId(), "repo.recovery.prepare", req.GetFingerprint())
	if e != nil {
		return nil, connectFileError(e)
	}
	intent, e := s.intentService.Consume(ctx, principal, req.GetIntentId(), preview.RepositoryID, "repo.recovery.prepare", preview.ExpectedRevision, preview.SubjectDigest)
	if e != nil {
		return nil, connect.NewError(connect.CodeAborted, e)
	}
	root, e := s.pushRecoveryRoot(ctx)
	if e != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, e)
	}
	deps := PushPullDeps{Git: s.git, RepoDir: preview.RepositoryPath, CredStore: s.credStore, RecoveryRoot: root}
	remote, branch := resolvePushTarget(ctx, deps, preview.RepositoryPath, req.GetRemote(), req.GetBranch())
	cred := lookupCredential(ctx, deps, remote)
	writeCtx, cancel := context.WithTimeout(policygate.WithIntent(context.WithoutCancel(ctx), intent.AsHumanIntent()), 10*time.Minute)
	defer cancel()
	if s.fileRoots != nil {
		s.fileRoots.RecordWrite(writeCtx)
	}
	a, e := PreparePushRecovery(writeCtx, deps, remote, branch, req.GetFingerprint(), cred)
	s.auditLogAsync(AuditEntry{Operation: AuditOperation("push_recovery_prepare"), RepoDir: preview.RepositoryPath, Success: e == nil && a.State == "prepared", Timestamp: time.Now().UTC()})
	if e != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, e)
	}
	return recoveryArtifactProto(a), nil
}

func recoveryArtifactProto(a pushsafety.Artifact) *repov1.PushRecoveryArtifact {
	out := &repov1.PushRecoveryArtifact{State: a.State, Message: a.Message, Fingerprint: a.Fingerprint, Head: a.Head, Base: a.Base, Candidate: a.Candidate, OriginalBundle: a.OriginalBundle, RepairedBundle: a.RepairedBundle, Paths: a.Paths, SignaturesRemoved: a.SignaturesRemoved}
	for _, m := range a.Mappings {
		out.Mappings = append(out.Mappings, &repov1.RecoveryCommitMapping{Original: m.Original, Replacement: m.Replacement})
	}
	return out
}

func PreparePushRecovery(ctx context.Context, deps PushPullDeps, remote, branch, fingerprint string, cred *StoredCredential) (pushsafety.Artifact, error) {
	if e := requireHumanMutation(ctx, "prepare push recovery"); e != nil {
		return pushsafety.Artifact{}, e
	}
	r := deps.Git.InspectPushSafety(ctx, deps.RepoDir, remote, branch, cred)
	if !r.Complete || !r.CanPrepare || r.Fingerprint != fingerprint {
		return pushsafety.Artifact{}, errors.New("Recovery preview is stale or unavailable. Refresh; no source files or history were changed.")
	}
	a, e := deps.Git.PreparePushRecovery(ctx, deps.RepoDir, deps.RecoveryRoot, r, cred)
	if e == nil && a.State == "prepared" && r.Head != "" {
		current, readErr := deps.Git.RevParse(ctx, deps.RepoDir, "HEAD")
		if readErr != nil || strings.TrimSpace(string(current)) != r.Head {
			a.State = "stale"
			a.Message = "The source commit changed during preparation. Bundles are retained for the reviewed snapshot; refresh before applying."
		}
	}
	return a, e
}

func (s *Server) getPushRecoveryConnect(ctx context.Context, req *repov1.GetPushRecoveryRequest) (*repov1.PushRecoveryArtifact, error) {
	repo, e := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if e != nil {
		return nil, connectFileError(e)
	}
	root, e := s.pushRecoveryRoot(ctx)
	if e != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, e)
	}
	a, e := s.git.GetPushRecovery(ctx, repo.Path, root, req.GetFingerprint())
	if e != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, e)
	}
	if a.State == "prepared" {
		deps := PushPullDeps{Git: s.git, RepoDir: repo.Path, CredStore: s.credStore}
		current := s.git.InspectPushSafety(ctx, repo.Path, a.Remote, a.Branch, lookupCredential(ctx, deps, a.Remote))
		a = pushsafety.CheckCurrent(a, current)
	}
	return recoveryArtifactProto(a), nil
}

func (s *Server) pushRecoveryRoot(ctx context.Context) (string, error) {
	if database.IsTestMode(ctx) && s.fileRoots == nil {
		return "", errors.New("recovery tests require a leased file-storage root")
	}
	if s.fileRoots != nil {
		root, e := s.fileRoots.PickRequired(ctx, storage.ClassData)
		if e != nil {
			return "", e
		}
		return filepath.Join(root, "push-recovery"), nil
	}
	if s.storageResolver == nil {
		return "", errors.New("recovery storage is not configured")
	}
	return s.storageResolver.Path(storage.Options{ScenarioID: "git-control-tower"}, storage.ClassData, "push-recovery")
}
