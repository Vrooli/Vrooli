package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"git-control-tower/internal/policygate"
	"git-control-tower/internal/proposals"
	"git-control-tower/internal/trailers"

	"github.com/vrooli/cli-core/cliutil"
	proposalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/proposals"
)

// AuditOpApplyProposal records a proposal apply attempt in the repository
// audit log.
const AuditOpApplyProposal AuditOperation = "apply_proposal"

// Commit write budget. The pre-commit check may run twice inside one commit
// (the explicit run and a GCT-managed git hook), so the budget covers two
// configured check timeouts plus git work, within the server write timeout.
const (
	commitBaseTimeout = 30 * time.Second
	commitMaxTimeout  = 30 * time.Minute
)

// commitWriteTimeout returns the write deadline for a commit that may run the
// configured pre-commit check. A fixed 30 seconds failed any check slower
// than that.
func (s *Server) commitWriteTimeout(ctx context.Context, repoPath string, skipPrecommit bool) time.Duration {
	if skipPrecommit || s.precommit == nil {
		return commitBaseTimeout
	}
	cfg, err := s.precommit.Get(ctx, repoPath)
	if err != nil || !cfg.Enabled || !cfg.RunBeforeCommit {
		return commitBaseTimeout
	}
	check := time.Duration(normalizePrecommitConfig(repoPath, cfg).TimeoutSeconds) * time.Second
	return min(commitBaseTimeout+2*check, commitMaxTimeout)
}

// initProposalService opens the proposal store. Without it the proposal
// procedures report unavailable; commits through the normal path still work.
func (s *Server) initProposalService() {
	store, err := proposals.NewStore(s.db)
	if err != nil {
		log.Printf("proposal store unavailable: %v", err)
		return
	}
	s.proposalService = proposals.NewService(store)
}

// proposalConnectServer is the ProposalService transport adapter. Domain
// rules live in internal/proposals; this file resolves the repository, the
// caller and, for apply, the human-control intent and repository lock.
type proposalConnectServer struct{ server *Server }

func (s *Server) proposalRepo(ctx context.Context, repositoryID string) (*RepoRecord, proposals.Repo, error) {
	if s.proposalService == nil {
		return nil, proposals.Repo{}, connect.NewError(connect.CodeUnavailable, errors.New("proposal store is unavailable"))
	}
	record, err := s.mutationRepo(ctx, repositoryID)
	if err != nil {
		return nil, proposals.Repo{}, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return record, proposals.Repo{ID: repositoryIDFor(record), Git: proposalGitFor(s.git, record.Path), Sandbox: s.sandboxPendingFor(record.Path)}, nil
}

// proposalRepoByID resolves the repository a stored proposal belongs to.
func (s *Server) proposalRepoByID(ctx context.Context, id string) (*RepoRecord, proposals.Repo, error) {
	if s.proposalService == nil {
		return nil, proposals.Repo{}, connect.NewError(connect.CodeUnavailable, errors.New("proposal store is unavailable"))
	}
	repositoryID, err := s.proposalService.RepositoryOf(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, proposals.Repo{}, proposalError(err)
	}
	return s.proposalRepo(ctx, repositoryID)
}

// sandboxPendingFor lists Workspace Sandbox runs whose applied changes are
// not committed yet, per repository-relative path.
func (s *Server) sandboxPendingFor(repoPath string) proposals.SandboxPending {
	if s.sandbox == nil || s.capabilities == nil {
		return nil
	}
	return func(ctx context.Context) (map[string][]string, error) {
		if !s.capabilities.IsAvailable(ctx, "workspace-sandbox") {
			return nil, errors.New("workspace sandbox unavailable")
		}
		response, err := s.sandbox.GetProvenanceByRun(ctx, repoPath)
		if err != nil || response == nil {
			return nil, err
		}
		pending := map[string][]string{}
		for _, group := range response.RunGroups {
			for _, file := range group.Files {
				if file.CommittedAt != "" || file.CommitHash != "" || file.RelativePath == "" {
					continue
				}
				pending[file.RelativePath] = append(pending[file.RelativePath], group.RunID)
			}
		}
		return pending, nil
	}
}

// proposalSubjectEvidence is the proposal part of a repo.apply_proposal intent
// digest.
func (s *Server) proposalSubjectEvidence(ctx context.Context, record *RepoRecord, subjectContext string) ([]byte, error) {
	if s.proposalService == nil {
		return nil, errors.New("proposal store is unavailable")
	}
	return s.proposalService.SubjectEvidence(ctx, proposals.Repo{ID: repositoryIDFor(record), Git: proposalGitFor(s.git, record.Path)}, subjectContext)
}

// proposalActor records who wrote a draft. A caller header that declares an
// agent downgrades a human principal to that agent kind, so agent writes get
// agent validation even in a session personal-local authentication maps to
// the OS user; a header never upgrades a caller to human.
func proposalActor(ctx context.Context, header http.Header, assertedRun string) proposals.Actor {
	actor := proposals.Actor{Kind: cliutil.CallerKindUnknown.String(), RunID: strings.ToLower(strings.TrimSpace(assertedRun))}
	if principal, ok := policygate.PrincipalFromContext(ctx); ok {
		actor.Subject, actor.Kind, actor.Verified = principal.Subject, principal.Kind.String(), principal.Verified
	} else {
		actor.Subject = "unverified"
	}
	if actor.Kind == cliutil.CallerKindHuman.String() && callerHeaderClaimsAgent(header) {
		actor.Kind = strings.ToLower(strings.TrimSpace(header.Get(cliutil.HeaderCaller)))
	}
	return actor
}

func proposalError(err error) error {
	var validation *proposals.ValidationError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &validation), errors.Is(err, proposals.ErrInvalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, proposals.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, proposals.ErrConflict):
		return connect.NewError(connect.CodeAborted, err)
	case errors.Is(err, proposals.ErrNotOpen):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, proposals.ErrForbidden):
		return connect.NewError(connect.CodePermissionDenied, err)
	default:
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			return err
		}
		return connect.NewError(connect.CodeInternal, err)
	}
}

func (h proposalConnectServer) AnchorScope(ctx context.Context, req *connect.Request[proposalsv1.AnchorScopeRequest]) (*connect.Response[proposalsv1.AnchorScopeResponse], error) {
	_, repo, err := h.server.proposalRepo(ctx, req.Msg.GetRepositoryId())
	if err != nil {
		return nil, err
	}
	anchor, err := h.server.proposalService.Anchor(ctx, repo, proposals.AnchorRequest{
		EffortRef: req.Msg.GetEffortRef(), Epoch: req.Msg.GetEpoch(), Scopes: req.Msg.GetScopes(),
		Actor: proposalActor(ctx, req.Header(), req.Msg.GetCreatorRunId()),
	})
	if err != nil {
		return nil, proposalError(err)
	}
	return connect.NewResponse(&proposalsv1.AnchorScopeResponse{Anchor: anchorProto(anchor)}), nil
}

func (h proposalConnectServer) CreateProposal(ctx context.Context, req *connect.Request[proposalsv1.CreateProposalRequest]) (*connect.Response[proposalsv1.CreateProposalResponse], error) {
	_, repo, err := h.server.proposalRepo(ctx, req.Msg.GetRepositoryId())
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	result, err := h.server.proposalService.Create(ctx, repo, proposals.CreateRequest{
		AnchorID: msg.GetAnchorId(), Work: workFromProto(msg.GetWork()), Subject: msg.GetSubject(), Body: msg.GetBody(),
		Trailers: entriesFromProto(msg.GetTrailers()), Paths: msg.GetPaths(), Evidence: evidenceFromProto(msg.GetEvidence()),
		Actor: proposalActor(ctx, req.Header(), msg.GetCreatorRunId()), ValidateOnly: msg.GetValidateOnly(),
	})
	if err != nil {
		return nil, proposalError(err)
	}
	return connect.NewResponse(&proposalsv1.CreateProposalResponse{
		Proposal: h.server.proposalProto(ctx, repo, result.View), Issues: issuesProto(result.Issues),
		Stored: result.Stored, SupersededId: result.SupersededID,
	}), nil
}

func (h proposalConnectServer) ListProposals(ctx context.Context, req *connect.Request[proposalsv1.ListProposalsRequest]) (*connect.Response[proposalsv1.ListProposalsResponse], error) {
	_, repo, err := h.server.proposalRepo(ctx, req.Msg.GetRepositoryId())
	if err != nil {
		return nil, err
	}
	filter := proposals.ListFilter{EffortRef: strings.TrimSpace(req.Msg.GetEffortRef()), Limit: int(req.Msg.GetLimit())}
	for _, state := range req.Msg.GetStates() {
		if converted := stateFromProto(state); converted != "" {
			filter.States = append(filter.States, converted)
		}
	}
	views, open, err := h.server.proposalService.List(ctx, repo, filter)
	if err != nil {
		return nil, proposalError(err)
	}
	response := &proposalsv1.ListProposalsResponse{OpenCount: int32(open)}
	for _, view := range views {
		response.Proposals = append(response.Proposals, h.server.proposalProto(ctx, repo, view))
	}
	return connect.NewResponse(response), nil
}

func (h proposalConnectServer) GetProposal(ctx context.Context, req *connect.Request[proposalsv1.GetProposalRequest]) (*connect.Response[proposalsv1.GetProposalResponse], error) {
	_, repo, err := h.server.proposalRepoByID(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	view, err := h.server.proposalService.Get(ctx, repo, req.Msg.GetId())
	if err != nil {
		return nil, proposalError(err)
	}
	return connect.NewResponse(&proposalsv1.GetProposalResponse{Proposal: h.server.proposalProto(ctx, repo, view)}), nil
}

func (h proposalConnectServer) EditProposal(ctx context.Context, req *connect.Request[proposalsv1.EditProposalRequest]) (*connect.Response[proposalsv1.EditProposalResponse], error) {
	_, repo, err := h.server.proposalRepoByID(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	edit := proposals.EditRequest{
		ID: msg.GetId(), ExpectedRevision: int(msg.GetExpectedRevision()), Trailers: entriesFromProto(msg.GetTrailers()),
		ReplaceTrailers: msg.GetReplaceTrailers(), RemovePaths: msg.GetRemovePaths(), Actor: proposalActor(ctx, req.Header(), ""),
	}
	if msg.Subject != nil {
		edit.Subject = msg.Subject
	}
	if msg.Body != nil {
		edit.Body = msg.Body
	}
	view, issues, err := h.server.proposalService.Edit(ctx, repo, edit)
	if err != nil {
		return nil, proposalError(err)
	}
	return connect.NewResponse(&proposalsv1.EditProposalResponse{Proposal: h.server.proposalProto(ctx, repo, view), Issues: issuesProto(issues)}), nil
}

func (h proposalConnectServer) WithdrawProposal(ctx context.Context, req *connect.Request[proposalsv1.WithdrawProposalRequest]) (*connect.Response[proposalsv1.WithdrawProposalResponse], error) {
	_, repo, err := h.server.proposalRepoByID(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	view, err := h.server.proposalService.Withdraw(ctx, repo, req.Msg.GetId(), req.Msg.GetReason(), proposalActor(ctx, req.Header(), ""))
	if err != nil {
		return nil, proposalError(err)
	}
	return connect.NewResponse(&proposalsv1.WithdrawProposalResponse{Proposal: h.server.proposalProto(ctx, repo, view)}), nil
}

func (h proposalConnectServer) RefreshProposal(ctx context.Context, req *connect.Request[proposalsv1.RefreshProposalRequest]) (*connect.Response[proposalsv1.RefreshProposalResponse], error) {
	_, repo, err := h.server.proposalRepoByID(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	view, dropped, err := h.server.proposalService.Refresh(ctx, repo, req.Msg.GetId(), int(req.Msg.GetExpectedRevision()), proposalActor(ctx, req.Header(), ""))
	if err != nil {
		return nil, proposalError(err)
	}
	return connect.NewResponse(&proposalsv1.RefreshProposalResponse{Proposal: h.server.proposalProto(ctx, repo, view), DroppedPaths: dropped}), nil
}

// ApplyProposal is the human-only commit path for a proposal. It follows the
// commit path exactly: verified human principal, repository lock, a fresh
// preview, single-use intent consumption, then the domain writer.
func (h proposalConnectServer) ApplyProposal(ctx context.Context, req *connect.Request[proposalsv1.ApplyProposalRequest]) (*connect.Response[proposalsv1.ApplyProposalResponse], error) {
	s := h.server
	principal, ok := policygate.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authenticate through the configured provider before applying a proposal"))
	}
	if principal.Kind != cliutil.CallerKindHuman {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("agent callers cannot apply proposals; the operator approves them"))
	}
	if callerHeaderClaimsAgent(req.Header()) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("this caller identifies as an agent; only the operator approves proposals"))
	}
	msg := req.Msg
	if strings.TrimSpace(msg.GetIntentId()) == "" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("review the exact proposal and confirm it before applying"))
	}
	if strings.TrimSpace(msg.GetId()) == "" || msg.GetRevision() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("proposal id and approved revision are required"))
	}
	record, repo, err := s.proposalRepo(ctx, msg.GetRepositoryId())
	if err != nil {
		return nil, err
	}
	writeCtx, cancel := context.WithTimeout(ctx, s.commitWriteTimeout(ctx, record.Path, msg.GetSkipPrecommitOnce()))
	defer cancel()
	cleanStaleLock(record.Path)
	unlock, err := s.repoLock.Acquire(writeCtx, record.Path)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("repository is busy, please retry"))
	}
	defer func() {
		unlock()
		s.repos.InvalidateStatus(record.Path)
	}()

	subjectContext := proposals.SubjectContext(msg.GetId(), int(msg.GetRevision()))
	preview, err := s.prepareMutationWithContext(writeCtx, repositoryIDFor(record), mutationOperationApplyProposal, subjectContext)
	if err != nil {
		return nil, proposalError(err)
	}
	consumed, err := s.intentService.Consume(writeCtx, principal, msg.GetIntentId(), preview.RepositoryID, mutationOperationApplyProposal, preview.ExpectedRevision, preview.SubjectDigest)
	if err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	writeCtx = policygate.WithIntent(writeCtx, consumed.AsHumanIntent())

	commit := func(commitCtx context.Context, message string) (proposals.CommitOutcome, error) {
		if err := requireHumanMutation(commitCtx, "apply proposal"); err != nil {
			return proposals.CommitOutcome{}, err
		}
		result, err := createAuthorizedCommit(commitCtx, CommitDeps{Git: s.git, RepoDir: record.Path, Precommit: s.precommit, Checks: s.commitChecks}, CommitRequest{
			IntentID: msg.GetIntentId(), Message: message, SkipPrecommitOnce: msg.GetSkipPrecommitOnce(),
		})
		if err != nil {
			return proposals.CommitOutcome{}, err
		}
		outcome := proposals.CommitOutcome{OID: result.Hash, Precommit: precommitOutcome(result.Precommit)}
		if !result.Success {
			outcome.Failure = commitAuditError(result)
		}
		return outcome, nil
	}
	actor := proposals.Actor{Subject: principal.Subject, Kind: principal.Kind.String(), Verified: principal.Verified}
	result, err := s.proposalService.Apply(writeCtx, repo, proposals.ApplyRequest{ID: msg.GetId(), Revision: int(msg.GetRevision()), Actor: actor}, commit)
	s.logProposalAudit(record.Path, msg, consumed.ID, result, err)
	if err != nil {
		return nil, proposalError(err)
	}
	if result.Success {
		s.notifyCommitToSandbox(record.Path, result.View.Paths(), &CommitResponse{Success: true, Hash: result.CommitOID, Message: result.View.Message.Rendered})
	}
	response := &proposalsv1.ApplyProposalResponse{
		Success: result.Success, CommitOid: result.CommitOID, Proposal: s.proposalProto(ctx, repo, result.View),
		Error: result.Error, CommitVerified: result.Verified, VerificationNotes: result.Notes,
	}
	if result.Refusal != nil {
		response.Refusal = &proposalsv1.ApplyRefusal{Code: result.Refusal.Code, Paths: result.Refusal.Paths, Detail: result.Refusal.Detail}
	}
	if result.Precommit != nil {
		p := result.Precommit
		response.Precommit = &proposalsv1.ApplyPrecommit{Status: p.Status, Command: p.Command, ExitCode: int32(p.ExitCode), Summary: p.Summary, Stdout: p.Stdout, Stderr: p.Stderr, DurationMs: p.DurationMs}
	}
	return connect.NewResponse(response), nil
}

// callerHeaderClaimsAgent reports whether the attribution header names an
// agent. Headers never grant authority, but a claim of agency only removes
// it: an agent driving a session that personal-local authentication verifies
// as the OS user still cannot approve a proposal.
func callerHeaderClaimsAgent(header http.Header) bool {
	switch strings.ToLower(strings.TrimSpace(header.Get(cliutil.HeaderCaller))) {
	case cliutil.CallerKindVrooliAgent.String(), cliutil.CallerKindExternalAgent.String(), cliutil.CallerKindOverride.String():
		return true
	default:
		return false
	}
}

func precommitOutcome(result *PrecommitRunResult) *proposals.PrecommitOutcome {
	if result == nil {
		return nil
	}
	return &proposals.PrecommitOutcome{Status: result.Status, Command: result.Command, ExitCode: result.ExitCode, Summary: result.Summary, Stdout: result.Stdout, Stderr: result.Stderr, DurationMs: result.DurationMs}
}

func (s *Server) logProposalAudit(repoPath string, req *proposalsv1.ApplyProposalRequest, intentID string, result proposals.ApplyResult, err error) {
	entry := AuditEntry{
		Operation: AuditOpApplyProposal, RepoDir: repoPath, Branch: result.View.Branch, Paths: result.View.Paths(),
		CommitHash: result.CommitOID, Success: err == nil && result.Success, Timestamp: time.Now().UTC(),
		Metadata: map[string]interface{}{"proposal_id": req.GetId(), "proposal_revision": req.GetRevision(), "proposal_digest": result.View.Digest, "intent_id": intentID, "commit_verified": result.Verified},
	}
	if result.Success {
		entry.CommitMessage = result.View.Message.Rendered
	}
	switch {
	case err != nil:
		entry.Error = err.Error()
	case result.Refusal != nil:
		entry.Error = "refused: " + result.Refusal.Code
		entry.Metadata["refusal_paths"] = result.Refusal.Paths
	case result.Error != "":
		entry.Error = result.Error
	}
	if len(result.Notes) > 0 {
		entry.Metadata["verification_notes"] = result.Notes
	}
	go func() {
		logCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.audit.Log(logCtx, entry)
	}()
}

// --- conversion ---

func (s *Server) proposalProto(ctx context.Context, repo proposals.Repo, view proposals.View) *proposalsv1.Proposal {
	if view.ID == "" {
		return nil
	}
	refs := trailers.ResolveEntries(ctx, view.Message.Trailers, s.proposalService.Resolver(repo))
	message := &proposalsv1.ProposalMessage{Subject: view.Message.Subject, Body: view.Message.Body, Rendered: view.Message.Rendered, OperatorEdited: view.Message.OperatorEdited}
	refIndex := 0
	for _, entry := range view.Message.Trailers {
		if entry.Key == "" {
			continue
		}
		trailer := &proposalsv1.ProposalTrailer{Key: entry.Key, Value: entry.Value, Kind: trailers.KindOf(entry.Key, entry.Value), Known: entry.Supported}
		if refIndex < len(refs) {
			trailer.Resolution, trailer.Detail = resolutionProto(refs[refIndex].Status), refs[refIndex].Detail
			refIndex++
		}
		message.Trailers = append(message.Trailers, trailer)
	}
	proposal := &proposalsv1.Proposal{
		Id: view.ID, RepositoryId: view.RepositoryID, State: stateProto(view.State), Revision: int32(view.Revision),
		CreatedBy: actorProto(view.CreatedBy), BaseHead: view.BaseHead, Branch: view.Branch,
		Work:    &proposalsv1.ProposalWork{EffortRef: view.Work.EffortRef, EffortRevision: view.Work.EffortRevision, Epoch: view.Work.Epoch, RunIds: view.Work.RunIDs, Plans: view.Work.Plans},
		Message: message, ProposalDigest: view.Digest, CommitOid: view.CommitOID, SupersededBy: view.SupersededBy,
		Evidence: &proposalsv1.ProposalEvidence{AnchorId: view.Evidence.AnchorID, EpochFile: view.Evidence.EpochFile, AcceptedLineSha256: view.Evidence.AcceptedLineSHA256, GateReceipts: view.Evidence.GateReceipts, MetricBefore: view.Evidence.MetricBefore, MetricAfter: view.Evidence.MetricAfter},
		Issues:   issuesProto(view.Issues), CreatedAt: timestamppb.New(view.CreatedAt), UpdatedAt: timestamppb.New(view.UpdatedAt),
		Freshness: &proposalsv1.ProposalFreshness{
			State: freshnessProto(view.Freshness.State), DriftedPaths: view.Freshness.Drifted, BaseChangedPaths: view.Freshness.BaseChanged,
			ForeignStagedPaths: view.Freshness.ForeignStaged, CurrentHead: view.Freshness.CurrentHead, Detail: view.Freshness.Detail,
			CheckedAt: timestamppb.New(view.Freshness.CheckedAt),
		},
	}
	if view.CommittedAt != nil {
		proposal.CommittedAt = timestamppb.New(*view.CommittedAt)
	}
	for _, file := range view.Files {
		converted := &proposalsv1.ProposalFile{Path: file.Path, Kind: kindProto(file.Kind), Sha256: file.SHA256, BlobId: file.BlobID, Deleted: file.Deleted, Source: file.Source}
		for _, flag := range file.Flags {
			converted.Flags = append(converted.Flags, &proposalsv1.ProposalFlag{Code: flag.Code, Detail: flag.Detail})
		}
		if current, ok := view.Freshness.CurrentBlobs[file.Path]; ok {
			converted.CurrentBlobId, converted.Drifted = current, current != file.ContentID()
		}
		proposal.Files = append(proposal.Files, converted)
	}
	for _, exclusion := range view.Excluded {
		proposal.Excluded = append(proposal.Excluded, &proposalsv1.ProposalExclusion{Path: exclusion.Path, Reason: exclusion.Reason, Detail: exclusion.Detail})
	}
	for _, event := range view.Events {
		proposal.Events = append(proposal.Events, &proposalsv1.ProposalEvent{Action: event.Action, Actor: actorProto(event.Actor), Revision: int32(event.Revision), Detail: event.Detail, Timestamp: timestamppb.New(event.Timestamp)})
	}
	return proposal
}

func anchorProto(anchor proposals.Anchor) *proposalsv1.Anchor {
	converted := &proposalsv1.Anchor{Id: anchor.ID, RepositoryId: anchor.RepositoryID, EffortRef: anchor.EffortRef, Epoch: anchor.Epoch, Scopes: anchor.Scopes, Head: anchor.Head, CreatedBy: actorProto(anchor.CreatedBy), CreatedAt: timestamppb.New(anchor.CreatedAt)}
	for _, file := range anchor.Files {
		converted.Files = append(converted.Files, &proposalsv1.AnchorFile{Path: file.Path, Sha256: file.SHA256, BlobId: file.BlobID, Deleted: file.Deleted})
	}
	return converted
}

func actorProto(actor proposals.Actor) *proposalsv1.Actor {
	return &proposalsv1.Actor{Subject: actor.Subject, Kind: actor.Kind, Verified: actor.Verified, RunId: actor.RunID}
}

func workFromProto(work *proposalsv1.ProposalWork) proposals.Work {
	if work == nil {
		return proposals.Work{}
	}
	return proposals.Work{EffortRef: work.GetEffortRef(), EffortRevision: work.GetEffortRevision(), Epoch: work.GetEpoch(), RunIDs: work.GetRunIds(), Plans: work.GetPlans()}
}

func evidenceFromProto(evidence *proposalsv1.ProposalEvidence) proposals.Evidence {
	if evidence == nil {
		return proposals.Evidence{}
	}
	return proposals.Evidence{EpochFile: evidence.GetEpochFile(), AcceptedLineSHA256: evidence.GetAcceptedLineSha256(), GateReceipts: evidence.GetGateReceipts(), MetricBefore: evidence.GetMetricBefore(), MetricAfter: evidence.GetMetricAfter()}
}

func entriesFromProto(values []*proposalsv1.ProposalTrailer) []trailers.Entry {
	entries := make([]trailers.Entry, 0, len(values))
	for _, value := range values {
		entries = append(entries, trailers.Entry{Key: value.GetKey(), Value: value.GetValue()})
	}
	return entries
}

func issuesProto(issues []trailers.Issue) []*proposalsv1.TrailerIssue {
	converted := make([]*proposalsv1.TrailerIssue, 0, len(issues))
	for _, issue := range issues {
		severity := proposalsv1.IssueSeverity_ISSUE_SEVERITY_WARNING
		if issue.Severity == trailers.SeverityError {
			severity = proposalsv1.IssueSeverity_ISSUE_SEVERITY_ERROR
		}
		converted = append(converted, &proposalsv1.TrailerIssue{Index: int32(issue.Index), Key: issue.Key, Code: issue.Code, Message: issue.Message, Severity: severity})
	}
	return converted
}

func stateProto(state proposals.State) proposalsv1.ProposalState {
	switch state {
	case proposals.StateOpen:
		return proposalsv1.ProposalState_PROPOSAL_STATE_OPEN
	case proposals.StateCommitted:
		return proposalsv1.ProposalState_PROPOSAL_STATE_COMMITTED
	case proposals.StateWithdrawn:
		return proposalsv1.ProposalState_PROPOSAL_STATE_WITHDRAWN
	case proposals.StateSuperseded:
		return proposalsv1.ProposalState_PROPOSAL_STATE_SUPERSEDED
	default:
		return proposalsv1.ProposalState_PROPOSAL_STATE_UNSPECIFIED
	}
}

func stateFromProto(state proposalsv1.ProposalState) proposals.State {
	switch state {
	case proposalsv1.ProposalState_PROPOSAL_STATE_OPEN:
		return proposals.StateOpen
	case proposalsv1.ProposalState_PROPOSAL_STATE_COMMITTED:
		return proposals.StateCommitted
	case proposalsv1.ProposalState_PROPOSAL_STATE_WITHDRAWN:
		return proposals.StateWithdrawn
	case proposalsv1.ProposalState_PROPOSAL_STATE_SUPERSEDED:
		return proposals.StateSuperseded
	default:
		return ""
	}
}

func kindProto(kind proposals.ChangeKind) proposalsv1.FileChangeKind {
	switch kind {
	case proposals.KindAdded:
		return proposalsv1.FileChangeKind_FILE_CHANGE_KIND_ADDED
	case proposals.KindModified:
		return proposalsv1.FileChangeKind_FILE_CHANGE_KIND_MODIFIED
	case proposals.KindDeleted:
		return proposalsv1.FileChangeKind_FILE_CHANGE_KIND_DELETED
	default:
		return proposalsv1.FileChangeKind_FILE_CHANGE_KIND_UNSPECIFIED
	}
}

func freshnessProto(state string) proposalsv1.FreshnessState {
	switch state {
	case proposals.FreshnessFresh:
		return proposalsv1.FreshnessState_FRESHNESS_STATE_FRESH
	case proposals.FreshnessDrifted:
		return proposalsv1.FreshnessState_FRESHNESS_STATE_DRIFTED
	case proposals.FreshnessBaseMoved:
		return proposalsv1.FreshnessState_FRESHNESS_STATE_BASE_MOVED
	case proposals.FreshnessUnknown:
		return proposalsv1.FreshnessState_FRESHNESS_STATE_UNKNOWN
	case proposals.FreshnessNotApplicable:
		return proposalsv1.FreshnessState_FRESHNESS_STATE_NOT_APPLICABLE
	default:
		return proposalsv1.FreshnessState_FRESHNESS_STATE_UNSPECIFIED
	}
}

func resolutionProto(status trailers.ResolutionStatus) proposalsv1.TrailerResolution {
	switch status {
	case trailers.Resolved:
		return proposalsv1.TrailerResolution_TRAILER_RESOLUTION_RESOLVED
	case trailers.Ambiguous:
		return proposalsv1.TrailerResolution_TRAILER_RESOLUTION_AMBIGUOUS
	case trailers.Inaccessible:
		return proposalsv1.TrailerResolution_TRAILER_RESOLUTION_INACCESSIBLE
	case trailers.Deleted:
		return proposalsv1.TrailerResolution_TRAILER_RESOLUTION_DELETED
	case trailers.Legacy:
		return proposalsv1.TrailerResolution_TRAILER_RESOLUTION_LEGACY
	case trailers.NotApplicable:
		return proposalsv1.TrailerResolution_TRAILER_RESOLUTION_NOT_APPLICABLE
	default:
		return proposalsv1.TrailerResolution_TRAILER_RESOLUTION_UNRESOLVED
	}
}
