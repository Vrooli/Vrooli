package coverage

import (
	"context"
	"errors"
	"log"

	"data-backup-manager/internal/coverage"
	"data-backup-manager/internal/sources"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/coverage"
	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/sources"
)

// Deps wires the seams the Connect coverage handler needs.
type Deps struct {
	Service   coverage.Service
	Decisions *SensitiveDecisionFlow
	Logger    *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the coverage Connect-RPC handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetCoverageReport(ctx context.Context, _ *connect.Request[coveragev1.GetCoverageReportRequest]) (*connect.Response[coveragev1.GetCoverageReportResponse], error) {
	report, err := h.deps.Service.Report(ctx)
	if err != nil {
		return nil, h.translate("GetCoverageReport", err)
	}
	return connect.NewResponse(&coveragev1.GetCoverageReportResponse{Report: reportToProto(report)}), nil
}

func (h *connectHandler) AcceptDefaultTargets(ctx context.Context, req *connect.Request[coveragev1.AcceptDefaultTargetsRequest]) (*connect.Response[coveragev1.AcceptDefaultTargetsResponse], error) {
	result, err := h.deps.Service.AcceptDefaults(ctx, coverage.AcceptOptions{
		IncludeSensitive: req.Msg.IncludeSensitive,
		DryRun:           req.Msg.DryRun,
	})
	if err != nil {
		return nil, h.translate("AcceptDefaultTargets", err)
	}
	return connect.NewResponse(acceptToProto(result)), nil
}

func (h *connectHandler) RequestSensitiveReview(ctx context.Context, _ *connect.Request[coveragev1.RequestSensitiveReviewRequest]) (*connect.Response[coveragev1.RequestSensitiveReviewResponse], error) {
	if h.deps.Decisions == nil || !h.deps.Decisions.Ready() {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("verified Notification Hub callback is not configured"))
	}
	askID, count, err := h.deps.Decisions.Request(ctx)
	if err != nil {
		return nil, h.translate("RequestSensitiveReview", err)
	}
	return connect.NewResponse(&coveragev1.RequestSensitiveReviewResponse{AskId: askID, CandidateCount: int32(count), Pending: askID != ""}), nil
}

func (h *connectHandler) GetSensitivePolicy(ctx context.Context, _ *connect.Request[coveragev1.GetSensitivePolicyRequest]) (*connect.Response[coveragev1.GetSensitivePolicyResponse], error) {
	policy, err := h.deps.Decisions.Policy(ctx)
	if err != nil {
		return nil, h.policyError("GetSensitivePolicy", err)
	}
	return connect.NewResponse(&coveragev1.GetSensitivePolicyResponse{Policy: policyToProto(policy)}), nil
}

func (h *connectHandler) RevokeSensitivePolicy(ctx context.Context, req *connect.Request[coveragev1.RevokeSensitivePolicyRequest]) (*connect.Response[coveragev1.RevokeSensitivePolicyResponse], error) {
	policy, err := h.deps.Decisions.Revoke(ctx, req.Msg.RevokedBy)
	if err != nil {
		return nil, h.policyError("RevokeSensitivePolicy", err)
	}
	return connect.NewResponse(&coveragev1.RevokeSensitivePolicyResponse{Policy: policyToProto(policy)}), nil
}

func (h *connectHandler) ExcludeSensitiveTarget(ctx context.Context, req *connect.Request[coveragev1.ExcludeSensitiveTargetRequest]) (*connect.Response[coveragev1.ExcludeSensitiveTargetResponse], error) {
	policy, err := h.deps.Decisions.Exclude(ctx, req.Msg.Owner, req.Msg.Name, req.Msg.Reason, req.Msg.ExcludedBy)
	if err != nil {
		return nil, h.policyError("ExcludeSensitiveTarget", err)
	}
	return connect.NewResponse(&coveragev1.ExcludeSensitiveTargetResponse{Policy: policyToProto(policy)}), nil
}

func (h *connectHandler) AdoptSensitiveConsent(ctx context.Context, req *connect.Request[coveragev1.AdoptSensitiveConsentRequest]) (*connect.Response[coveragev1.AdoptSensitiveConsentResponse], error) {
	policy, swept, err := h.deps.Decisions.AdoptAnswerAsConsent(ctx, req.Msg.HubAskId)
	if err != nil {
		if errors.Is(err, ErrConsentNotAdoptable) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, h.policyError("AdoptSensitiveConsent", err)
	}
	return connect.NewResponse(&coveragev1.AdoptSensitiveConsentResponse{Policy: policyToProto(policy), RegisteredCount: int32(len(swept.Registered)), FailedCount: int32(swept.Failed)}), nil
}

// policyError separates a refused request (nothing to revoke, a missing field)
// from a store failure.
func (h *connectHandler) policyError(op string, err error) error {
	switch {
	case h.deps.Decisions == nil || h.deps.Decisions.policy == nil:
		return connect.NewError(connect.CodeUnavailable, err)
	case errors.Is(err, ErrPolicyRequest):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	h.deps.Logger.Printf("coverage.%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, err)
}

func policyToProto(p StandingPolicy) *coveragev1.SensitivePolicy {
	out := &coveragev1.SensitivePolicy{
		Active: p.Active, GrantAskId: p.GrantAskID, GrantedBy: p.GrantedBy,
		GrantedByIdentityKind: p.GrantedByKind, GrantedByIdentitySource: p.GrantedBySource,
		RevokedBy: p.RevokedBy, AddedTotal: int32(p.AddedTotal), GrantBasis: p.GrantBasis, GrantText: p.GrantText,
	}
	if p.GrantedAt != nil {
		out.GrantedAt = timestamppb.New(*p.GrantedAt)
	}
	if p.RevokedAt != nil {
		out.RevokedAt = timestamppb.New(*p.RevokedAt)
	}
	if p.LastNoteAt != nil {
		out.LastNoteAt = timestamppb.New(*p.LastNoteAt)
	}
	if p.LastSweepAt != nil {
		out.LastSweepAt = timestamppb.New(*p.LastSweepAt)
	}
	for _, e := range p.Excluded {
		out.Excluded = append(out.Excluded, &coveragev1.SensitivePolicyExclusion{Owner: e.Owner, Name: e.Name, Reason: e.Reason, ExcludedBy: e.By, ExcludedAt: timestamppb.New(e.At)})
	}
	for _, a := range p.Unreported {
		out.Unreported = append(out.Unreported, &coveragev1.SensitivePolicyAddition{Owner: a.Owner, Name: a.Name, Group: a.Group, ApproxBytes: a.Bytes, AddedAt: timestamppb.New(a.At)})
	}
	for _, e := range p.Ledger {
		out.History = append(out.History, &coveragev1.SensitivePolicyEvent{At: timestamppb.New(e.At), Kind: e.Kind, By: e.By, Detail: e.Detail})
	}
	return out
}

// translate maps a domain error to a Connect error, logging only internal ones.
func (h *connectHandler) translate(op string, err error) error {
	connectErr := coverage.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("coverage.%s: %v", op, err)
	}
	return connectErr
}

func reportToProto(r coverage.Report) *coveragev1.CoverageReport {
	out := &coveragev1.CoverageReport{
		Summary:             summaryToProto(r.Summary),
		RegisteredTargets:   make([]*coveragev1.RegisteredTarget, 0, len(r.Registered)),
		RecommendedTargets:  make([]*coveragev1.SuggestedTarget, 0, len(r.Recommended)),
		SensitiveTargets:    make([]*coveragev1.SuggestedTarget, 0, len(r.Sensitive)),
		ReacquirableTargets: make([]*coveragev1.SuggestedTarget, 0, len(r.Reacquirable)),
	}
	for _, t := range r.Registered {
		out.RegisteredTargets = append(out.RegisteredTargets, registeredToProto(t))
	}
	for _, s := range r.Recommended {
		out.RecommendedTargets = append(out.RecommendedTargets, suggestionToProto(s))
	}
	for _, s := range r.Sensitive {
		out.SensitiveTargets = append(out.SensitiveTargets, suggestionToProto(s))
	}
	for _, s := range r.Reacquirable {
		out.ReacquirableTargets = append(out.ReacquirableTargets, suggestionToProto(s))
	}
	return out
}

func summaryToProto(s coverage.Summary) *coveragev1.CoverageSummary {
	return &coveragev1.CoverageSummary{
		RegisteredCount:                  int32(s.RegisteredCount),
		RecommendedCount:                 int32(s.RecommendedCount),
		SensitiveCount:                   int32(s.SensitiveCount),
		PlannedCount:                     int32(s.PlannedCount),
		BackedUpCount:                    int32(s.BackedUpCount),
		VerifiedCount:                    int32(s.VerifiedCount),
		DefaultCoverageComplete:          s.DefaultCoverageComplete,
		HasSensitiveUnreviewed:           s.HasSensitiveUnreviewed,
		HasUnplannedRegisteredTargets:    s.HasUnplannedRegisteredTargets,
		HasUnverifiedTargets:             s.HasUnverifiedTargets,
		DestinationCopyCount:             int32(s.DestinationCopyCount),
		AllDestinationBackedUpCount:      int32(s.AllDestinationBackedUpCount),
		AllDestinationVerifiedCount:      int32(s.AllDestinationVerifiedCount),
		MultiDestinationTargetCount:      int32(s.MultiDestinationTargetCount),
		PhysicallyIndependentTargetCount: int32(s.PhysicallyIndependentTargetCount),
	}
}

func registeredToProto(t coverage.RegisteredTarget) *coveragev1.RegisteredTarget {
	out := &coveragev1.RegisteredTarget{
		Id:                         t.ID,
		Owner:                      t.Owner,
		Name:                       t.Name,
		SourceKind:                 kindToProto(t.SourceKind),
		Locator:                    t.Locator,
		Planned:                    t.Planned,
		Critical:                   t.Critical,
		Reacquirable:               t.Reacquirable,
		ReacquisitionCommand:       t.ReacquisitionCommand,
		ReacquisitionEvidence:      t.ReacquisitionEvidence,
		ReacquisitionNeedsOperator: t.ReacquisitionNeedsOperator,
		ReacquisitionAuthReason:    t.ReacquisitionAuthReason,
		AllDestinationsBackedUp:    t.AllDestinationsBackedUp,
		AllDestinationsVerified:    t.AllDestinationsVerified,
		PhysicalIndependenceProven: t.PhysicalIndependenceProven,
		IndependenceWarnings:       append([]string(nil), t.IndependenceWarnings...),
	}
	if !t.LastSuccessAt.IsZero() {
		out.LastSuccessAt = timestamppb.New(t.LastSuccessAt)
	}
	if !t.LastVerifiedAt.IsZero() {
		out.LastVerifiedAt = timestamppb.New(t.LastVerifiedAt)
	}
	for _, destination := range t.Destinations {
		status := &coveragev1.TargetDestinationStatus{DestinationId: destination.DestinationID}
		if !destination.LastSuccessAt.IsZero() {
			status.LastSuccessAt = timestamppb.New(destination.LastSuccessAt)
		}
		if !destination.LastVerifiedAt.IsZero() {
			status.LastVerifiedAt = timestamppb.New(destination.LastVerifiedAt)
		}
		out.Destinations = append(out.Destinations, status)
	}
	return out
}

func suggestionToProto(s coverage.Suggestion) *coveragev1.SuggestedTarget {
	return &coveragev1.SuggestedTarget{
		Id:                         s.ID,
		Owner:                      s.Owner,
		Name:                       s.Name,
		SourceKind:                 kindToProto(s.SourceKind),
		Locator:                    s.Locator,
		Rationale:                  s.Rationale,
		ApproxBytes:                s.ApproxBytes,
		Sensitive:                  s.Sensitive,
		Warning:                    s.Warning,
		Critical:                   s.Critical,
		Reacquirable:               s.Reacquirable,
		BudgetMaxBytes:             s.BudgetMaxBytes,
		BudgetMaxAge:               s.BudgetMaxAge,
		Retention:                  s.Retention,
		ReacquisitionCommand:       s.ReacquisitionCommand,
		ReacquisitionEvidence:      s.ReacquisitionEvidence,
		ReacquisitionNeedsOperator: s.ReacquisitionNeedsOperator,
		ReacquisitionAuthReason:    s.ReacquisitionAuthReason,
	}
}

func acceptToProto(r coverage.AcceptResult) *coveragev1.AcceptDefaultTargetsResponse {
	out := &coveragev1.AcceptDefaultTargetsResponse{
		Accepted:         make([]*coveragev1.AcceptedTarget, 0, len(r.Accepted)),
		SkippedSensitive: make([]*coveragev1.SuggestedTarget, 0, len(r.SkippedSensitive)),
		Failed:           make([]*coveragev1.AcceptError, 0, len(r.Failed)),
		DryRun:           r.DryRun,
	}
	for _, a := range r.Accepted {
		out.Accepted = append(out.Accepted, &coveragev1.AcceptedTarget{
			TargetId:     a.TargetID,
			SuggestionId: a.SuggestionID,
			Owner:        a.Owner,
			Name:         a.Name,
			SourceKind:   kindToProto(a.SourceKind),
			Locator:      a.Locator,
			Sensitive:    a.Sensitive,
			Critical:     a.Critical,
		})
	}
	for _, s := range r.SkippedSensitive {
		out.SkippedSensitive = append(out.SkippedSensitive, suggestionToProto(s))
	}
	for _, f := range r.Failed {
		out.Failed = append(out.Failed, &coveragev1.AcceptError{
			SuggestionId: f.SuggestionID,
			Owner:        f.Owner,
			Name:         f.Name,
			Message:      f.Message,
		})
	}
	return out
}

// kindToProto translates the domain SourceKind to the proto enum so domain code
// never imports the generated enum (mirrors the targets/discovery handlers).
func kindToProto(k sources.SourceKind) sourcesv1.SourceKind {
	switch k {
	case sources.KindWorkspace:
		return sourcesv1.SourceKind_SOURCE_KIND_WORKSPACE_CHECKPOINT
	case sources.KindFilesystem:
		return sourcesv1.SourceKind_SOURCE_KIND_FILESYSTEM
	case sources.KindSQLite:
		return sourcesv1.SourceKind_SOURCE_KIND_SQLITE
	case sources.KindPostgres:
		return sourcesv1.SourceKind_SOURCE_KIND_POSTGRES
	case sources.KindRedis:
		return sourcesv1.SourceKind_SOURCE_KIND_REDIS
	case sources.KindQdrant:
		return sourcesv1.SourceKind_SOURCE_KIND_QDRANT
	case sources.KindObjectStorage:
		return sourcesv1.SourceKind_SOURCE_KIND_OBJECT_STORAGE
	default:
		return sourcesv1.SourceKind_SOURCE_KIND_UNSPECIFIED
	}
}
