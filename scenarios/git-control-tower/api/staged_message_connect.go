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
	s.stagedMessages = stagedmessages.New(store, stagedmessages.GitSnapshotter{}, stagedRetainedReader{s}, newStagedMessageProvider(s))
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
	if errors.Is(e, stagedmessages.ErrNotFound) || errors.Is(e, sql.ErrNoRows) {
		code = connect.CodeNotFound
	} else if errors.Is(e, stagedmessages.ErrConflict) {
		code = connect.CodeAborted
	} else if errors.Is(e, context.Canceled) {
		code = connect.CodeCanceled
	} else if errors.Is(e, context.DeadlineExceeded) {
		code = connect.CodeDeadlineExceeded
	}
	return connect.NewError(code, e)
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

func (h advisoryConnectServer) ReadStagedMessageEvidence(ctx context.Context, req *connect.Request[v1.ReadStagedMessageEvidenceRequest]) (*connect.Response[v1.ReadStagedMessageEvidenceResponse], error) {
	if _, err := authn.RequireCapability(ctx, "git-control-tower:read"); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("verified staged-evidence read capability required"))
	}
	actor, err := authn.RequireHuman(ctx)
	execution := ""
	if err != nil {
		p := provenance.FromContext(ctx)
		if !p.IsVerifiedAgent() || p.WorkflowExecutionID == "" {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("evidence requires the owning verified workflow child"))
		}
		execution = p.WorkflowExecutionID
	} else if !actor.IsVerified() {
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
