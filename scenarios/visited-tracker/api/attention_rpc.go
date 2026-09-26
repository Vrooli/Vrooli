package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	attentionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/visited-tracker/v1/attention"
	"github.com/vrooli/vrooli/packages/proto/gen/go/visited-tracker/v1/attention/attentionv1connect"
)

type attentionRPC struct {
	attentionv1connect.UnimplementedAttentionServiceHandler
}

func attentionHandler() (string, http.Handler) {
	return attentionv1connect.NewAttentionServiceHandler(&attentionRPC{})
}

func attentionError(err error) error {
	code := connect.CodeInternal
	switch {
	case errors.Is(err, context.Canceled):
		code = connect.CodeCanceled
	case errors.Is(err, context.DeadlineExceeded):
		code = connect.CodeDeadlineExceeded
	case errors.Is(err, ErrAttentionInput):
		code = connect.CodeInvalidArgument
	case errors.Is(err, ErrCampaignConflict), errors.Is(err, ErrClaimConflict):
		code = connect.CodeAborted
	case errors.Is(err, ErrClaimExpired):
		code = connect.CodeFailedPrecondition
	case errors.Is(err, ErrClaimCapacity):
		code = connect.CodeResourceExhausted
	case errors.Is(err, ErrCampaignNotFound):
		code = connect.CodeNotFound
	}
	return connect.NewError(code, err)
}

func campaignUUID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("campaign_id must be a nonzero UUID"))
	}
	return id, nil
}

func (*attentionRPC) EnsureCampaign(ctx context.Context, req *connect.Request[attentionv1.EnsureCampaignRequest]) (*connect.Response[attentionv1.CampaignResponse], error) {
	in := req.Msg
	if strings.TrimSpace(in.Location) == "" || !validIdentity(in.Tag) || len(in.Patterns) == 0 || len(in.Patterns) > 32 {
		return nil, attentionError(fmt.Errorf("%w: require location, tag and 1..32 patterns", ErrAttentionInput))
	}
	limit := int(in.MaxFiles)
	if limit == 0 {
		limit = defaultMaxFiles
	}
	if limit < 1 || limit > 10000 {
		return nil, attentionError(fmt.Errorf("%w: max_files must be 1..10000", ErrAttentionInput))
	}
	location, err := filepath.Abs(in.Location)
	if err != nil {
		return nil, attentionError(err)
	}
	location, err = filepath.EvalSymlinks(location)
	if err != nil {
		return nil, attentionError(fmt.Errorf("%w: location is unavailable", ErrAttentionInput))
	}
	patterns := append([]string(nil), in.Patterns...)
	slices.Sort(patterns)
	patterns = slices.Compact(patterns)
	for _, pattern := range patterns {
		if len(pattern) == 0 || len(pattern) > 512 {
			return nil, attentionError(ErrAttentionInput)
		}
	}
	release, err := lockCampaignCatalog(ctx)
	if err != nil {
		return nil, attentionError(err)
	}
	defer release()
	campaigns, err := loadAllCampaigns()
	if err != nil {
		return nil, attentionError(err)
	}
	for _, c := range campaigns {
		if c.Location == nil || c.Tag == nil {
			continue
		}
		oldLocation, _ := filepath.Abs(*c.Location)
		if real, e := filepath.EvalSymlinks(oldLocation); e == nil {
			oldLocation = real
		}
		if oldLocation == location && *c.Tag == in.Tag {
			oldPatterns := append([]string(nil), c.Patterns...)
			slices.Sort(oldPatterns)
			oldPatterns = slices.Compact(oldPatterns)
			if !slices.Equal(oldPatterns, patterns) || c.MaxFiles != limit {
				return nil, attentionError(ErrClaimConflict)
			}
			return connect.NewResponse(&attentionv1.CampaignResponse{CampaignId: c.ID.String(), Revision: c.Revision}), nil
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	tag := in.Tag
	c := Campaign{
		ID: uuid.New(), Name: location + "-" + tag, Location: &location, Tag: &tag, Patterns: patterns,
		MaxFiles: limit, ExcludePatterns: defaultExcludePatterns, CreatedAt: now, Status: "active", Metadata: map[string]interface{}{},
	}
	if _, err := syncCampaignFiles(&c, c.Patterns); err != nil {
		return nil, attentionError(err)
	}
	if err := saveCampaign(ctx, &c); err != nil {
		return nil, attentionError(err)
	}
	return connect.NewResponse(&attentionv1.CampaignResponse{CampaignId: c.ID.String(), Revision: c.Revision}), nil
}

func (*attentionRPC) Preview(ctx context.Context, req *connect.Request[attentionv1.PreviewRequest]) (*connect.Response[attentionv1.PreviewResponse], error) {
	id, err := campaignUUID(req.Msg.CampaignId)
	if err != nil {
		return nil, err
	}
	limit := int(req.Msg.Limit)
	if limit == 0 {
		limit = 10
	}
	if limit < 1 || limit > 100 {
		return nil, attentionError(ErrAttentionInput)
	}
	c, err := loadCampaign(id)
	if err != nil {
		return nil, attentionError(err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Preview refreshes an isolated value; it never consumes a claim or writes.
	if _, err := syncCampaignFiles(c, c.Patterns); err != nil {
		return nil, attentionError(err)
	}
	now := time.Now()
	eligible := selectAttention(c, 0, now)
	response := &attentionv1.PreviewResponse{
		CampaignRevision: c.Revision, ObservedAt: now.UTC().Format(time.RFC3339Nano),
		EligibleCount: uint32(len(eligible)), ActiveClaimCount: uint32(activeClaimCount(c, now)),
		OldestEligibleAgeSeconds:    uint64(oldestEligibleAge(eligible, now).Seconds()),
		OldestActiveClaimAgeSeconds: uint64(oldestActiveClaimAge(c, now).Seconds()),
		Integrity:                   observeClaimIntegrity(c, now),
		StorageWrites:               campaignWriteObserver.snapshot(now),
	}
	for _, file := range eligible {
		if len(response.Candidates) >= limit {
			break
		}
		response.Candidates = append(response.Candidates, &attentionv1.Candidate{
			FileId: file.ID.String(), Path: file.FilePath,
			Revision: *file.ContentHash, Score: file.AttentionScore, Priority: priority(file), Reviewed: file.LastReviewed != nil && file.ReviewedRevision == *file.ContentHash,
		})
	}
	return connect.NewResponse(response), nil
}

func claimMessage(c *ReviewClaim) *attentionv1.Claim {
	if c == nil {
		return nil
	}
	result := &attentionv1.Claim{
		Id: c.ID, RequestId: c.RequestID, Worker: c.Worker, FileId: c.FileID.String(), Path: c.FilePath,
		Revision: c.Revision, ExpiresAt: c.ExpiresAt.Format(time.RFC3339Nano), Outcome: c.Outcome, Evidence: c.Evidence,
	}
	if c.CompletedAt != nil {
		result.CompletedAt = c.CompletedAt.Format(time.RFC3339Nano)
	}
	return result
}

func (*attentionRPC) Claim(ctx context.Context, req *connect.Request[attentionv1.ClaimRequest]) (*connect.Response[attentionv1.ClaimResponse], error) {
	id, err := campaignUUID(req.Msg.CampaignId)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ttl := int(req.Msg.TtlSeconds)
	if ttl == 0 {
		ttl = 600
	}
	claim, err := claimAttention(ctx, id, req.Msg.RequestId, req.Msg.Worker, ttl, time.Now().UTC())
	if err != nil {
		return nil, attentionError(err)
	}
	return connect.NewResponse(&attentionv1.ClaimResponse{Claim: claimMessage(claim), NoWork: claim == nil}), nil
}

func (*attentionRPC) Complete(ctx context.Context, req *connect.Request[attentionv1.CompleteRequest]) (*connect.Response[attentionv1.ClaimResponse], error) {
	id, err := campaignUUID(req.Msg.CampaignId)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	claim, err := completeAttention(ctx, id, req.Msg.ClaimId, req.Msg.Worker, req.Msg.Outcome, req.Msg.Evidence, time.Now().UTC())
	if err != nil {
		return nil, attentionError(err)
	}
	return connect.NewResponse(&attentionv1.ClaimResponse{Claim: claimMessage(claim)}), nil
}
