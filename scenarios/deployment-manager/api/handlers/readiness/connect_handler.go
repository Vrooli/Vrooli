package readinesshandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"deployment-manager/readiness"

	"connectrpc.com/connect"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness/readinessv1connect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type goalReader interface {
	GoalClosed(context.Context, string) (bool, error)
}

type ConnectHandler struct {
	readinessconnect.UnimplementedReadinessServiceHandler
	preparer *readiness.Preparer
	repo     readiness.ReviewRepository
	goals    goalReader
	policy   readiness.Checklist
	now      func() time.Time
}

func NewConnectHandler(preparer *readiness.Preparer, repo readiness.ReviewRepository, goals goalReader) *ConnectHandler {
	return &ConnectHandler{preparer: preparer, repo: repo, goals: goals, policy: readiness.DefaultChecklist(), now: time.Now}
}

func (h *ConnectHandler) ReportEvidence(ctx context.Context, req *connect.Request[readinessv1.ReportEvidenceRequest]) (*connect.Response[readinessv1.ReportEvidenceResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.ObservedAt == nil || !req.Msg.ObservedAt.IsValid() {
		return nil, invalid(errors.New("identity, criterion, producer binding, status, observation time, and evidence reference are required"))
	}
	identity, err := (readiness.ReviewIdentity{Scenario: req.Msg.Scenario, ProfileID: req.Msg.ProfileId, CandidateCommit: req.Msg.CandidateCommit, ArtifactDigest: req.Msg.ArtifactDigest, Targets: req.Msg.Targets, Channel: req.Msg.Channel, PolicyVersion: int(req.Msg.PolicyVersion)}).Canonical()
	if err != nil {
		return nil, invalid(err)
	}
	criterion, ok := policyItem(h.policy, req.Msg.CriterionId)
	if !ok || criterion.ProducerBinding() == "" || criterion.ProducerBinding() != req.Msg.ProducerBinding {
		return nil, failed(errors.New("criterion and producer binding do not match the active readiness policy"))
	}
	key, err := identity.Key()
	if err != nil {
		return nil, invalid(err)
	}
	observation := readiness.EvidenceObservation{
		Identity: identity, CriterionID: criterion.ID, ProducerBinding: req.Msg.ProducerBinding,
		Evidence: readiness.EvidenceItem{
			CriterionID: criterion.ID, Status: readiness.SignalStatus(req.Msg.Status),
			Producer: criterion.Owner, ProducerVersion: req.Msg.ProducerVersion,
			CandidateCommit: identity.CandidateCommit, ArtifactDigest: identity.ArtifactDigest,
			Target: strings.Join(identity.Targets, ","), Environment: identity.Channel,
			PolicyVersion: identity.PolicyVersion, ObservedAt: req.Msg.ObservedAt.AsTime().UTC(),
			Reference: req.Msg.EvidenceReference, Detail: req.Msg.Detail,
		},
	}
	if err := h.repo.SaveObservation(ctx, observation); err != nil {
		return nil, failed(err)
	}
	return connect.NewResponse(&readinessv1.ReportEvidenceResponse{IdentityKey: key, CriterionId: criterion.ID, ProducerBinding: req.Msg.ProducerBinding, Accepted: true}), nil
}

func (h *ConnectHandler) PrepareReview(ctx context.Context, req *connect.Request[readinessv1.PrepareReviewRequest]) (*connect.Response[readinessv1.ReviewResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, invalid(errors.New("request is required"))
	}
	decision, err := h.preparer.Prepare(ctx, readiness.PrepareRequest{
		Identity: readiness.ReviewIdentity{Scenario: req.Msg.Scenario, ProfileID: req.Msg.ProfileId, CandidateCommit: req.Msg.CandidateCommit, ArtifactDigest: req.Msg.ArtifactDigest, Targets: req.Msg.Targets, Channel: req.Msg.Channel, PolicyVersion: int(req.Msg.PolicyVersion)}, Facts: req.Msg.Facts, Deliverable: req.Msg.Deliverable, Trigger: req.Msg.Trigger,
	})
	if err != nil {
		return nil, failed(err)
	}
	evidence, findings, err := h.repo.ListEvaluation(ctx, decision.Review.Key)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(toResponse(&decision.Review, evidence, findings, decision.Next, decision.Deduped)), nil
}

func (h *ConnectHandler) GetReview(ctx context.Context, req *connect.Request[readinessv1.GetReviewRequest]) (*connect.Response[readinessv1.ReviewResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.ReviewKey == "" {
		return nil, invalid(errors.New("review_key is required"))
	}
	return h.response(ctx, req.Msg.ReviewKey)
}

func (h *ConnectHandler) ListReviews(ctx context.Context, req *connect.Request[readinessv1.ListReviewsRequest]) (*connect.Response[readinessv1.ListReviewsResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, invalid(errors.New("request is required"))
	}
	reviews, err := h.repo.ListReviews(ctx, readiness.ReviewStatus(req.Msg.Status), int(req.Msg.PageSize))
	if err != nil {
		return nil, internal(err)
	}
	result := &readinessv1.ListReviewsResponse{Count: int32(len(reviews))}
	for index := range reviews {
		evidence, findings, err := h.repo.ListEvaluation(ctx, reviews[index].Key)
		if err != nil {
			return nil, internal(err)
		}
		result.Reviews = append(result.Reviews, toResponse(&reviews[index], evidence, findings, nil, false))
	}
	return connect.NewResponse(result), nil
}

func (h *ConnectHandler) ListReviewWaivers(ctx context.Context, req *connect.Request[readinessv1.ListReviewWaiversRequest]) (*connect.Response[readinessv1.ListReviewWaiversResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, invalid(errors.New("request is required"))
	}
	waivers, err := h.repo.ListWaivers(ctx, req.Msg.ReviewKey, int(req.Msg.PageSize))
	if err != nil {
		return nil, internal(err)
	}
	result := &readinessv1.ListReviewWaiversResponse{Count: int32(len(waivers))}
	for _, waiver := range waivers {
		result.Waivers = append(result.Waivers, &readinessv1.ReviewWaiver{ReviewKey: waiver.ReviewKey, CriterionId: waiver.CriterionID, Actor: waiver.Actor, Reason: waiver.Reason, ExpiresAt: timestamppb.New(waiver.ExpiresAt), InvalidationTrigger: waiver.Trigger, CreatedAt: timestamppb.New(waiver.CreatedAt)})
	}
	return connect.NewResponse(result), nil
}

func (h *ConnectHandler) SynchronizeGoalClosure(ctx context.Context, req *connect.Request[readinessv1.SynchronizeGoalClosureRequest]) (*connect.Response[readinessv1.ReviewResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.ReviewKey == "" {
		return nil, invalid(errors.New("review_key is required"))
	}
	review, err := h.repo.Get(ctx, req.Msg.ReviewKey)
	if err != nil {
		return nil, internal(err)
	}
	if review.GoalRef == "" || h.goals == nil {
		return nil, failed(errors.New("review has no configured independent goal"))
	}
	closed, err := h.goals.GoalClosed(ctx, review.GoalRef)
	if err != nil {
		return nil, unavailable(err)
	}
	if !closed {
		return nil, failed(errors.New("independent readiness goal is not closed"))
	}
	if review.GoalClosedAt == nil {
		if err := h.repo.RecordGoalClosure(ctx, review.Key, h.now().UTC()); err != nil {
			return nil, internal(err)
		}
	}
	return h.response(ctx, review.Key)
}

func (h *ConnectHandler) ApproveReview(ctx context.Context, req *connect.Request[readinessv1.ApproveReviewRequest]) (*connect.Response[readinessv1.ReviewResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.Identity == nil || req.Msg.ReviewKey == "" || req.Msg.Actor == "" {
		return nil, invalid(errors.New("review_key, identity, and actor are required"))
	}
	review, err := h.repo.Get(ctx, req.Msg.ReviewKey)
	if err != nil {
		return nil, internal(err)
	}
	identity := toDomainIdentity(req.Msg.Identity)
	canonical, err := identity.Canonical()
	canonicalKey, keyErr := canonical.Key()
	if err != nil || keyErr != nil || canonicalKey != review.Key {
		return nil, failed(errors.New("approval identity does not exactly match the stored review"))
	}
	if err := h.revalidate(ctx, review); err != nil {
		return nil, failed(err)
	}
	if err := h.repo.Approve(ctx, review.Key, canonical, req.Msg.Actor, h.now().UTC()); err != nil {
		return nil, failed(err)
	}
	return h.response(ctx, review.Key)
}

func (h *ConnectHandler) CreateWaiver(ctx context.Context, req *connect.Request[readinessv1.CreateWaiverRequest]) (*connect.Response[readinessv1.ReviewResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.ExpiresAt == nil || !req.Msg.ExpiresAt.IsValid() {
		return nil, invalid(errors.New("review, criterion, actor, reason, and valid expiry are required"))
	}
	criterion, ok := policyItem(h.policy, req.Msg.CriterionId)
	if !ok || !criterion.Waiver.Eligible {
		return nil, failed(errors.New("criterion is not waiver eligible"))
	}
	now := h.now().UTC()
	expires := req.Msg.ExpiresAt.AsTime().UTC()
	if criterion.Waiver.MaxAgeSeconds > 0 && expires.After(now.Add(time.Duration(criterion.Waiver.MaxAgeSeconds)*time.Second)) {
		return nil, failed(errors.New("waiver expiry exceeds the criterion policy maximum"))
	}
	if err := h.repo.SaveWaiver(ctx, readiness.ReviewWaiver{ReviewKey: req.Msg.ReviewKey, CriterionID: req.Msg.CriterionId, Actor: req.Msg.Actor, Reason: req.Msg.Reason, ExpiresAt: expires, Trigger: req.Msg.InvalidationTrigger, CreatedAt: now}); err != nil {
		return nil, failed(err)
	}
	return h.response(ctx, req.Msg.ReviewKey)
}

func (h *ConnectHandler) RecordHumanCheck(ctx context.Context, req *connect.Request[readinessv1.RecordHumanCheckRequest]) (*connect.Response[readinessv1.ReviewResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.ReviewedAt == nil || !req.Msg.ReviewedAt.IsValid() {
		return nil, invalid(errors.New("review, human criterion, verdict, actor, evidence reference, and review time are required"))
	}
	criterion, ok := policyItem(h.policy, req.Msg.CriterionId)
	if !ok || criterion.HumanReview == nil {
		return nil, failed(errors.New("criterion is not an independent human review criterion"))
	}
	if err := h.repo.SaveHumanCheck(ctx, readiness.HumanCheck{ReviewKey: req.Msg.ReviewKey, CriterionID: req.Msg.CriterionId, Verdict: req.Msg.Verdict, Actor: req.Msg.Actor, EvidenceReference: req.Msg.EvidenceReference, ReviewedAt: req.Msg.ReviewedAt.AsTime().UTC()}); err != nil {
		return nil, failed(err)
	}
	return h.response(ctx, req.Msg.ReviewKey)
}

func (h *ConnectHandler) CheckPolicyProjection(_ context.Context, req *connect.Request[readinessv1.CheckPolicyProjectionRequest]) (*connect.Response[readinessv1.CheckPolicyProjectionResponse], error) {
	candidate := readiness.BuiltInPolicyJSON()
	if req != nil && req.Msg != nil && len(req.Msg.PolicyJson) > 0 {
		candidate = req.Msg.PolicyJson
	}
	if err := readiness.CheckProjection(candidate); err != nil {
		return nil, failed(err)
	}
	return connect.NewResponse(&readinessv1.CheckPolicyProjectionResponse{PolicyVersion: int32(h.policy.Version), CriterionCount: int32(len(h.policy.Items)), Matches: true}), nil
}

func (h *ConnectHandler) revalidate(ctx context.Context, review *readiness.Review) error {
	if review.GoalClosedAt == nil {
		return errors.New("independent goal closure has not been synchronized")
	}
	evidence, _, err := h.repo.ListEvaluation(ctx, review.Key)
	if err != nil {
		return err
	}
	if len(evidence) != len(h.policy.Items) {
		return errors.New("required evidence set is incomplete")
	}
	items := make(map[string]readiness.Item, len(h.policy.Items))
	for _, item := range h.policy.Items {
		items[item.ID] = item
	}
	now := h.now().UTC()
	activeWaivers, err := h.repo.ListActiveWaivers(ctx, review.Key, now)
	if err != nil {
		return err
	}
	waived := make(map[string]struct{}, len(activeWaivers))
	for _, waiver := range activeWaivers {
		waived[waiver.CriterionID] = struct{}{}
	}
	humanChecks, err := h.repo.ListHumanChecks(ctx, review.Key)
	if err != nil {
		return err
	}
	humanPassed := make(map[string]bool, len(humanChecks))
	for _, check := range humanChecks {
		humanPassed[check.CriterionID] = check.Verdict == "passed"
	}
	for _, item := range evidence {
		criterion, ok := items[item.CriterionID]
		if !ok || item.CandidateCommit != review.Identity.CandidateCommit || item.ArtifactDigest != review.Identity.ArtifactDigest || item.PolicyVersion != review.Identity.PolicyVersion {
			return fmt.Errorf("evidence %q no longer matches the review identity", item.CriterionID)
		}
		if criterion.Freshness.Basis == "max_age" && now.Sub(item.ObservedAt) > time.Duration(criterion.Freshness.MaxAgeSeconds)*time.Second {
			return fmt.Errorf("evidence %q is stale", item.CriterionID)
		}
		switch item.Status {
		case readiness.SignalPassed, readiness.SignalNotApplicable:
		case readiness.SignalWaived:
			if _, ok := waived[item.CriterionID]; !ok {
				return fmt.Errorf("evidence %q no longer has an active waiver", item.CriterionID)
			}
		case readiness.SignalUnknown:
			if criterion.HumanReview == nil || !humanPassed[item.CriterionID] {
				return fmt.Errorf("evidence %q has no passed independent human check", item.CriterionID)
			}
		default:
			return fmt.Errorf("evidence %q has blocking disposition %q", item.CriterionID, item.Status)
		}
	}
	return nil
}

func (h *ConnectHandler) response(ctx context.Context, key string) (*connect.Response[readinessv1.ReviewResponse], error) {
	review, err := h.repo.Get(ctx, key)
	if err != nil {
		return nil, internal(err)
	}
	evidence, findings, err := h.repo.ListEvaluation(ctx, key)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(toResponse(review, evidence, findings, nil, false)), nil
}

func toDomainIdentity(value *readinessv1.ReviewIdentity) readiness.ReviewIdentity {
	return readiness.ReviewIdentity{Scenario: value.Scenario, ProfileID: value.ProfileId, CandidateCommit: value.CandidateCommit, ArtifactDigest: value.ArtifactDigest, Targets: value.Targets, Channel: value.Channel, PolicyVersion: int(value.PolicyVersion)}
}

func toProtoIdentity(value readiness.ReviewIdentity) *readinessv1.ReviewIdentity {
	return &readinessv1.ReviewIdentity{Scenario: value.Scenario, ProfileId: value.ProfileID, CandidateCommit: value.CandidateCommit, ArtifactDigest: value.ArtifactDigest, Targets: value.Targets, Channel: value.Channel, PolicyVersion: int32(value.PolicyVersion)}
}

func toResponse(review *readiness.Review, evidence []readiness.EvidenceItem, findings []readiness.ReviewFinding, next []string, deduped bool) *readinessv1.ReviewResponse {
	result := &readinessv1.ReviewResponse{ReviewKey: review.Key, Status: string(review.Status), Identity: toProtoIdentity(review.Identity), ComparisonMode: string(review.ComparisonMode), PredecessorReleaseId: review.PredecessorReleaseID, PredecessorCommit: review.PredecessorCommit, PredecessorArtifactDigest: review.PredecessorArtifactDigest, GoalRef: review.GoalRef, ApprovedBy: review.ApprovedBy, NextActions: next, Deduped: deduped}
	if review.GoalClosedAt != nil {
		result.GoalClosedAt = timestamppb.New(*review.GoalClosedAt)
	}
	if review.ApprovedAt != nil {
		result.ApprovedAt = timestamppb.New(*review.ApprovedAt)
	}
	for _, value := range findings {
		result.Findings = append(result.Findings, asStruct(value))
	}
	for _, value := range evidence {
		result.Evidence = append(result.Evidence, asStruct(value))
	}
	return result
}

func asStruct(value any) *structpb.Struct {
	data, _ := json.Marshal(value)
	var mapped map[string]any
	_ = json.Unmarshal(data, &mapped)
	result, _ := structpb.NewStruct(mapped)
	return result
}

func policyItem(policy readiness.Checklist, id string) (readiness.Item, bool) {
	for _, item := range policy.Items {
		if item.ID == id {
			return item, true
		}
	}
	return readiness.Item{}, false
}

func invalid(err error) error     { return connect.NewError(connect.CodeInvalidArgument, err) }
func failed(err error) error      { return connect.NewError(connect.CodeFailedPrecondition, err) }
func unavailable(err error) error { return connect.NewError(connect.CodeUnavailable, err) }
func internal(err error) error    { return connect.NewError(connect.CodeInternal, err) }
