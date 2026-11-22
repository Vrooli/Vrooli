package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/proposals"
	"git-control-tower/internal/reviewjobs"
	"git-control-tower/internal/stagedmessages"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/provenance"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

func (s *Server) initStagedMessages() {
	store, e := reviewjobs.New(s.db)
	if e != nil {
		log.Printf("staged message durable owner unavailable: %v", e)
		return
	}
	s.stagedMessages = stagedmessages.New(store, stagedmessages.GitSnapshotter{Now: time.Now}, stagedRetainedReader{s}, newStagedMessageProvider(s))
	s.stagedMessages.SetContinuations(stagedmessages.GitContinuations{})
}

type stagedRetainedReader struct{ server *Server }

func (r stagedRetainedReader) Messages(ctx context.Context, repoID string, snapshot *v1.StagedMessageSnapshot) ([]stagedmessages.RetainedMessage, []*v1.Omission, error) {
	if r.server.proposalService == nil {
		return nil, []*v1.Omission{{Reason: "retained proposal capture unavailable"}}, nil
	}
	candidates, e := r.server.proposalService.RetainedCandidates(ctx, repoID, 100)
	if e != nil {
		return nil, nil, e
	}
	selected := map[string]*v1.StagedMessageFile{}
	for _, f := range snapshot.Files {
		selected[f.Path] = f
	}
	var out []stagedmessages.RetainedMessage
	var omissions []*v1.Omission
	total := 0
	readCount := 0
	repo := proposals.Repo{ID: repoID}
	for _, p := range candidates {
		m := stagedmessages.RetainedMessage{ID: "proposal:" + p.ID + "@" + strconv.Itoa(p.Revision), Revision: strconv.Itoa(p.Revision), Title: p.Message.Subject, Body: p.Message.Body, Lifecycle: string(p.State)}
		unavailable := 0
		for _, f := range p.Files {
			rf := stagedmessages.RetainedFile{Path: f.Path, BaseBlobID: f.BaseBlobID, BlobID: f.BlobID, OldMode: f.BaseMode, NewMode: f.Mode, Captured: f.Captured}
			if f.Deleted {
				rf.BlobID = ""
				rf.NewMode = "000000"
			}
			sf := selected[f.Path]
			if sf == nil {
				rf.Captured = false
			} else if !f.Captured {
				unavailable++
			} else if stagedmessages.ExactRetainedFile(sf, rf) {
				// Owner-retained immutable identities suffice for whole-file
				// correspondence even when explanatory bytes were omitted.
			} else if sf.OmissionReason != "" || readCount >= 64 || total >= 4<<20 {
				unavailable++
			} else {
				readCount++
				_, before, after, err := r.server.proposalService.ReviewFile(ctx, repo, p.ID, p.Revision, f.Path)
				if err != nil || len(before)+len(after) > 256<<10 || total+len(before)+len(after) > 4<<20 {
					unavailable++
				} else {
					rf.Before = before
					rf.After = after
					rf.BytesAvailable = true
					total += len(before) + len(after)
				}
			}
			m.Files = append(m.Files, rf)
		}
		if unavailable > 0 {
			omissions = append(omissions, &v1.Omission{Reason: fmt.Sprintf("%s: %d retained files lack fragment bytes (missing capture, unavailable content or bounded 64-file/4 MiB detail budget)", m.ID, unavailable)})
		}
		out = append(out, m)
	}
	if len(candidates) == 100 {
		omissions = append(omissions, &v1.Omission{Reason: "latest 100 retained proposal candidate limit reached; older revisions not searched"})
	}
	return out, omissions, nil
}

func (h advisoryConnectServer) stagedRepo(ctx context.Context, repoID string) (*RepoRecord, error) {
	if h.server == nil || h.server.stagedMessages == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("staged message owner unavailable"))
	}
	if repoID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("repository id is required"))
	}
	record, e := h.server.mutationRepo(ctx, repoID)
	if e != nil {
		return nil, connect.NewError(connect.CodeNotFound, e)
	}
	return record, nil
}

func stagedMessageError(e error) error {
	if e == nil {
		return nil
	}
	var typed *connect.Error
	if errors.As(e, &typed) {
		return e
	}
	code := connect.CodeFailedPrecondition
	message := e
	if errors.Is(e, reviewjobs.ErrStorage) {
		code = connect.CodeInternal
		message = errors.New("staged message storage is temporarily unavailable")
	} else if errors.Is(e, stagedmessages.ErrNotFound) || errors.Is(e, sql.ErrNoRows) {
		code = connect.CodeNotFound
	} else if errors.Is(e, reviewjobs.ErrQueuePathConflict) {
		code = connect.CodeAlreadyExists
	} else if errors.Is(e, stagedmessages.ErrConflict) || errors.Is(e, reviewjobs.ErrQueueConflict) {
		code = connect.CodeAborted
	} else if errors.Is(e, context.Canceled) {
		code = connect.CodeCanceled
	} else if errors.Is(e, context.DeadlineExceeded) {
		code = connect.CodeDeadlineExceeded
	}
	return connect.NewError(code, message)
}

func (h advisoryConnectServer) CheckStagedMessage(ctx context.Context, req *connect.Request[v1.CheckStagedMessageRequest]) (*connect.Response[v1.CheckStagedMessageResponse], error) {
	record, e := h.stagedRepo(ctx, req.Msg.GetRepositoryId())
	if e != nil {
		return nil, e
	}
	out, e := h.server.stagedMessages.Check(ctx, repositoryIDFor(record), record.Path, req.Msg.GetSnapshotDigest())
	if e != nil {
		return nil, stagedMessageError(e)
	}
	return connect.NewResponse(out), nil
}

func (h advisoryConnectServer) CollectStagedMessage(ctx context.Context, req *connect.Request[v1.CollectStagedMessageRequest]) (*connect.Response[v1.StagedMessageCollection], error) {
	record, e := h.stagedRepo(ctx, req.Msg.GetRepositoryId())
	if e != nil {
		return nil, e
	}
	out, e := h.server.stagedMessages.Collect(ctx, repositoryIDFor(record), record.Path)
	if e != nil {
		return nil, stagedMessageError(e)
	}
	return connect.NewResponse(out), nil
}

func (h advisoryConnectServer) StartStagedMessage(ctx context.Context, req *connect.Request[v1.StartStagedMessageRequest]) (*connect.Response[v1.StagedMessageJob], error) {
	record, e := h.stagedRepo(ctx, req.Msg.GetRepositoryId())
	if e != nil {
		return nil, e
	}
	out, e := h.server.stagedMessages.Start(ctx, repositoryIDFor(record), record.Path, req.Msg)
	if e != nil {
		return nil, stagedMessageError(e)
	}
	return connect.NewResponse(out), nil
}

func (h advisoryConnectServer) RetryStagedMessage(ctx context.Context, req *connect.Request[v1.RetryStagedMessageRequest]) (*connect.Response[v1.StagedMessageJob], error) {
	record, e := h.stagedRepo(ctx, req.Msg.GetRepositoryId())
	if e != nil {
		return nil, e
	}
	out, e := h.server.stagedMessages.Retry(ctx, repositoryIDFor(record), record.Path, req.Msg)
	if e != nil {
		return nil, stagedMessageError(e)
	}
	return connect.NewResponse(out), nil
}

func (h advisoryConnectServer) GetStagedMessageProviderReadiness(ctx context.Context, _ *connect.Request[v1.StagedMessageProviderReadinessRequest]) (*connect.Response[v1.StagedMessageProviderReadiness], error) {
	if h.server == nil || h.server.stagedMessages == nil {
		return connect.NewResponse(&v1.StagedMessageProviderReadiness{Provider: "workspace-sandbox", State: "unknown", Message: "Workspace Sandbox readiness could not be confirmed."}), nil
	}
	readiness := h.server.stagedMessages.ProviderReadiness(ctx)
	out := &v1.StagedMessageProviderReadiness{Provider: readiness.Provider, State: readiness.State, Known: readiness.Known, Message: readiness.Message}
	if readiness.ObservedAt != nil && !readiness.ObservedAt.IsZero() {
		out.ObservedAt = readiness.ObservedAt.UTC().Format(time.RFC3339Nano)
	}
	return connect.NewResponse(out), nil
}

func (h advisoryConnectServer) StartWorktreeStagedMessage(ctx context.Context, req *connect.Request[v1.StartWorktreeStagedMessageRequest]) (*connect.Response[v1.StagedMessageJob], error) {
	record, e := h.stagedRepo(ctx, req.Msg.GetRepositoryId())
	if e != nil {
		return nil, e
	}
	out, e := h.server.stagedMessages.StartWorktree(ctx, repositoryIDFor(record), record.Path, req.Msg)
	if e != nil {
		return nil, stagedMessageError(e)
	}
	return connect.NewResponse(out), nil
}

func (h advisoryConnectServer) QueueWorktreeSelection(ctx context.Context, req *connect.Request[v1.QueueWorktreeSelectionRequest]) (*connect.Response[v1.QueuedWorktreeSelection], error) {
	record, e := h.stagedRepo(ctx, req.Msg.GetRepositoryId())
	if e != nil {
		return nil, e
	}
	out, e := h.server.stagedMessages.QueueWorktreeSelection(ctx, repositoryIDFor(record), record.Path, req.Msg)
	if e != nil {
		return nil, stagedMessageError(e)
	}
	return connect.NewResponse(out), nil
}

func (h advisoryConnectServer) FinalizeQueuedSelection(ctx context.Context, req *connect.Request[v1.FinalizeQueuedSelectionRequest]) (*connect.Response[v1.QueuedWorktreeSelection], error) {
	record, e := h.stagedRepo(ctx, req.Msg.GetRepositoryId())
	if e != nil {
		return nil, e
	}
	out, e := h.server.stagedMessages.FinalizeQueuedSelection(ctx, repositoryIDFor(record), record.Path, req.Msg)
	if e != nil {
		return nil, stagedMessageError(e)
	}
	return connect.NewResponse(out), nil
}

func (h advisoryConnectServer) StartQueuedSelection(ctx context.Context, req *connect.Request[v1.StartQueuedSelectionRequest]) (*connect.Response[v1.StagedMessageJob], error) {
	record, e := h.stagedRepo(ctx, req.Msg.GetRepositoryId())
	if e != nil {
		return nil, e
	}
	out, e := h.server.stagedMessages.StartQueuedSelection(ctx, repositoryIDFor(record), record.Path, req.Msg)
	if e != nil {
		return nil, stagedMessageError(e)
	}
	return connect.NewResponse(out), nil
}

func (h advisoryConnectServer) ListWorktreeSuggestionQueue(ctx context.Context, req *connect.Request[v1.ListWorktreeSuggestionQueueRequest]) (*connect.Response[v1.WorktreeSuggestionQueue], error) {
	record, e := h.stagedRepo(ctx, req.Msg.GetRepositoryId())
	if e != nil {
		return nil, e
	}
	out, e := h.server.stagedMessages.ListWorktreeQueue(ctx, repositoryIDFor(record))
	if e != nil {
		return nil, stagedMessageError(e)
	}
	return connect.NewResponse(out), nil
}

func (h advisoryConnectServer) EnqueueWorktreeSuggestion(ctx context.Context, req *connect.Request[v1.EnqueueWorktreeSuggestionRequest]) (*connect.Response[v1.WorktreeSuggestionQueueEntry], error) {
	record, e := h.stagedRepo(ctx, req.Msg.GetRepositoryId())
	if e != nil {
		return nil, e
	}
	out, e := h.server.stagedMessages.EnqueueWorktreeQueue(ctx, repositoryIDFor(record), req.Msg)
	if e != nil {
		return nil, stagedMessageError(e)
	}
	return connect.NewResponse(out), nil
}

func (h advisoryConnectServer) UpdateWorktreeSuggestionQueueEntry(ctx context.Context, req *connect.Request[v1.UpdateWorktreeSuggestionQueueEntryRequest]) (*connect.Response[v1.WorktreeSuggestionQueueMutationResponse], error) {
	record, e := h.stagedRepo(ctx, req.Msg.GetRepositoryId())
	if e != nil {
		return nil, e
	}
	writeCtx, cancel, unlock, e := h.lockQueueMutation(ctx, record.Path)
	if e != nil {
		return nil, e
	}
	defer cancel()
	defer unlock()
	out, e := h.server.stagedMessages.UpdateWorktreeQueue(writeCtx, repositoryIDFor(record), record.Path, req.Msg)
	if e != nil {
		return nil, stagedMessageError(e)
	}
	return connect.NewResponse(out), nil
}

func (h advisoryConnectServer) RetireWorktreeSuggestionQueueHead(ctx context.Context, req *connect.Request[v1.RetireWorktreeSuggestionQueueHeadRequest]) (*connect.Response[v1.WorktreeSuggestionQueueMutationResponse], error) {
	record, e := h.stagedRepo(ctx, req.Msg.GetRepositoryId())
	if e != nil {
		return nil, e
	}
	writeCtx, cancel, unlock, e := h.lockQueueMutation(ctx, record.Path)
	if e != nil {
		return nil, e
	}
	defer cancel()
	defer unlock()
	out, e := h.server.stagedMessages.RetireWorktreeQueueHead(writeCtx, repositoryIDFor(record), record.Path, req.Msg)
	if e != nil {
		return nil, stagedMessageError(e)
	}
	return connect.NewResponse(out), nil
}

func (h advisoryConnectServer) RemoveWorktreeSuggestionQueueEntry(ctx context.Context, req *connect.Request[v1.RemoveWorktreeSuggestionQueueEntryRequest]) (*connect.Response[v1.WorktreeSuggestionQueueMutationResponse], error) {
	record, e := h.stagedRepo(ctx, req.Msg.GetRepositoryId())
	if e != nil {
		return nil, e
	}
	writeCtx, cancel, unlock, e := h.lockQueueMutation(ctx, record.Path)
	if e != nil {
		return nil, e
	}
	defer cancel()
	defer unlock()
	out, e := h.server.stagedMessages.RemoveWorktreeQueueEntry(writeCtx, repositoryIDFor(record), req.Msg)
	if e != nil {
		return nil, stagedMessageError(e)
	}
	return connect.NewResponse(out), nil
}

func (h advisoryConnectServer) lockQueueMutation(ctx context.Context, repoPath string) (context.Context, context.CancelFunc, func(), error) {
	writeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	unlock, err := h.server.repoLock.Acquire(writeCtx, repoPath)
	if err != nil {
		cancel()
		return nil, nil, nil, connect.NewError(connect.CodeUnavailable, errors.New("repository is busy, please retry"))
	}
	return writeCtx, cancel, unlock, nil
}

func (h advisoryConnectServer) ReadStagedMessageEvidence(ctx context.Context, req *connect.Request[v1.ReadStagedMessageEvidenceRequest]) (*connect.Response[v1.ReadStagedMessageEvidenceResponse], error) {
	principal, err := authn.RequireCapability(ctx, "git-control-tower:read")
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("verified staged-evidence read capability required"))
	}
	proof := provenance.FromContext(ctx)
	execution := ""
	if proof.IsVerifiedAgent() {
		// Attribute the read to the verified child when it carries a workflow
		// binding. Its own scope is sufficient. Some local auth providers cannot
		// forward the human credential to Agent Manager, though, so an explicitly
		// verified owner may authorize the read when the child token names that
		// same owner. Never combine one owner's authority with another owner's run.
		if proof.WorkflowExecutionID == "" {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("evidence requires a verified workflow-bound child identity"))
		}
		ownerAuthorized := principal.IsVerified() && principal.IsHuman() && principal.Subject != "" && principal.Subject == proof.Subject
		if proof.RequireScope("git-control-tower:read") != nil && !ownerAuthorized {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("evidence requires the workflow child or its verified owner to hold git-control-tower:read"))
		}
		execution = proof.WorkflowExecutionID
	} else if !principal.IsVerified() || !principal.IsHuman() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("verified evidence reader required"))
	}
	record, err := h.stagedRepo(ctx, req.Msg.RepositoryId)
	if err != nil {
		return nil, err
	}
	page, err := h.server.stagedMessages.ReadEvidence(ctx, repositoryIDFor(record), record.Path, req.Msg, execution)
	if err != nil {
		return nil, stagedMessageError(err)
	}
	return connect.NewResponse(page), nil
}

func (h advisoryConnectServer) stagedJob(ctx context.Context, req *connect.Request[v1.StagedMessageJobRequest], action string) (*connect.Response[v1.StagedMessageJob], error) {
	record, e := h.stagedRepo(ctx, req.Msg.GetRepositoryId())
	if e != nil {
		return nil, e
	}
	timeout := time.Duration(req.Msg.GetTimeoutSeconds()) * time.Second
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	if timeout > 55*time.Second {
		timeout = 55 * time.Second
	}
	out, e := h.server.stagedMessages.Job(ctx, repositoryIDFor(record), record.Path, req.Msg.GetJobId(), req.Msg.GetRequestKey(), action, timeout)
	if e != nil {
		return nil, stagedMessageError(e)
	}
	return connect.NewResponse(out), nil
}

func (h advisoryConnectServer) GetStagedMessageJob(ctx context.Context, r *connect.Request[v1.StagedMessageJobRequest]) (*connect.Response[v1.StagedMessageJob], error) {
	return h.stagedJob(ctx, r, "get")
}

func (h advisoryConnectServer) WaitStagedMessageJob(ctx context.Context, r *connect.Request[v1.StagedMessageJobRequest]) (*connect.Response[v1.StagedMessageJob], error) {
	return h.stagedJob(ctx, r, "wait")
}

func (h advisoryConnectServer) CancelStagedMessageJob(ctx context.Context, r *connect.Request[v1.StagedMessageJobRequest]) (*connect.Response[v1.StagedMessageJob], error) {
	return h.stagedJob(ctx, r, "cancel")
}
