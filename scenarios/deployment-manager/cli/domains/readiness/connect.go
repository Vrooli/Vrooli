package readiness

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness/readinessv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	ConnectGroupName = "readiness-reviews"
	WaiverGroupName  = "readiness-review-waivers"
)

type connectHandlers struct {
	client readinessconnect.ReadinessServiceClient
}

func newConnectHandlers(core *cliapp.ScenarioApp) *connectHandlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &connectHandlers{client: readinessconnect.NewReadinessServiceClient(httpClient, baseURL)}
}

func (h *connectHandlers) prepare(ctx cliapp.OperationContext) (*readinessv1.ReviewResponse, error) {
	response, err := h.client.PrepareReview(context.Background(), connect.NewRequest(&readinessv1.PrepareReviewRequest{
		Scenario: ctx.Positional("scenario"), ProfileId: ctx.Positional("profile_id"), CandidateCommit: ctx.Positional("candidate_commit"), ArtifactDigest: ctx.Positional("artifact_digest"), Targets: ctx.Positionals("targets"), Channel: ctx.Positional("channel"), PolicyVersion: 2,
		Deliverable: ctx.Flag("deliverable"), Trigger: ctx.Flag("trigger"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("prepare readiness review", err, nil)
	}
	return response.Msg, nil
}

func (h *connectHandlers) reportEvidence(ctx cliapp.OperationContext) (*readinessv1.ReportEvidenceResponse, error) {
	observed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(ctx.Flag("observed-at")))
	if err != nil {
		return nil, fmt.Errorf("parse --observed-at as RFC3339: %w", err)
	}
	response, err := h.client.ReportEvidence(context.Background(), connect.NewRequest(&readinessv1.ReportEvidenceRequest{
		Scenario: ctx.Positional("scenario"), ProfileId: ctx.Positional("profile_id"), CandidateCommit: ctx.Positional("candidate_commit"), ArtifactDigest: ctx.Positional("artifact_digest"), Targets: ctx.Positionals("targets"), Channel: ctx.Positional("channel"), PolicyVersion: 2,
		CriterionId: ctx.Positional("criterion_id"), ProducerBinding: ctx.Positional("producer_binding"), ProducerVersion: ctx.Flag("producer-version"), Status: ctx.Positional("status"), ObservedAt: timestamppb.New(observed), EvidenceReference: ctx.Flag("evidence-reference"), Detail: ctx.Flag("detail"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("report readiness evidence", err, nil)
	}
	return response.Msg, nil
}

func (h *connectHandlers) get(ctx cliapp.OperationContext) (*readinessv1.ReviewResponse, error) {
	response, err := h.client.GetReview(context.Background(), connect.NewRequest(&readinessv1.GetReviewRequest{ReviewKey: ctx.Positional("review_key")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get readiness review", err, nil)
	}
	return response.Msg, nil
}

func (h *connectHandlers) list(ctx cliapp.OperationContext) (*readinessv1.ListReviewsResponse, error) {
	response, err := h.client.ListReviews(context.Background(), connect.NewRequest(&readinessv1.ListReviewsRequest{Status: ctx.Flag("status"), PageSize: parsePageSize(ctx.Flag("page-size"))}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list readiness reviews", err, nil)
	}
	return response.Msg, nil
}

func (h *connectHandlers) listWaivers(ctx cliapp.OperationContext) (*readinessv1.ListReviewWaiversResponse, error) {
	response, err := h.client.ListReviewWaivers(context.Background(), connect.NewRequest(&readinessv1.ListReviewWaiversRequest{ReviewKey: ctx.Flag("review-key"), PageSize: parsePageSize(ctx.Flag("page-size"))}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list readiness waivers", err, nil)
	}
	return response.Msg, nil
}

func (h *connectHandlers) sync(ctx cliapp.OperationContext) (*readinessv1.ReviewResponse, error) {
	response, err := h.client.SynchronizeGoalClosure(context.Background(), connect.NewRequest(&readinessv1.SynchronizeGoalClosureRequest{ReviewKey: ctx.Positional("review_key")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("synchronize readiness goal", err, nil)
	}
	return response.Msg, nil
}

func (h *connectHandlers) approve(ctx cliapp.OperationContext) (*readinessv1.ReviewResponse, error) {
	response, err := h.client.ApproveReview(context.Background(), connect.NewRequest(&readinessv1.ApproveReviewRequest{
		ReviewKey: ctx.Positional("review_key"), Actor: ctx.Positional("actor"), Identity: &readinessv1.ReviewIdentity{
			Scenario: ctx.Positional("scenario"), ProfileId: ctx.Positional("profile_id"), CandidateCommit: ctx.Positional("candidate_commit"), ArtifactDigest: ctx.Positional("artifact_digest"), Targets: ctx.Positionals("targets"), Channel: ctx.Positional("channel"), PolicyVersion: 2,
		},
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("approve readiness review", err, nil)
	}
	return response.Msg, nil
}

func (h *connectHandlers) createWaiver(ctx cliapp.OperationContext) (*readinessv1.ReviewResponse, error) {
	expires, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(ctx.Flag("expires-at")))
	if err != nil {
		return nil, fmt.Errorf("parse --expires-at as RFC3339: %w", err)
	}
	response, err := h.client.CreateWaiver(context.Background(), connect.NewRequest(&readinessv1.CreateWaiverRequest{
		ReviewKey: ctx.Positional("review_key"), CriterionId: ctx.Positional("criterion_id"), Actor: ctx.Positional("actor"), Reason: ctx.Positional("reason"), ExpiresAt: timestamppb.New(expires), InvalidationTrigger: ctx.Flag("invalidation-trigger"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create readiness waiver", err, nil)
	}
	return response.Msg, nil
}

func (h *connectHandlers) recordHumanCheck(ctx cliapp.OperationContext) (*readinessv1.ReviewResponse, error) {
	reviewed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(ctx.Flag("reviewed-at")))
	if err != nil {
		return nil, fmt.Errorf("parse --reviewed-at as RFC3339: %w", err)
	}
	response, err := h.client.RecordHumanCheck(context.Background(), connect.NewRequest(&readinessv1.RecordHumanCheckRequest{
		ReviewKey: ctx.Positional("review_key"), CriterionId: ctx.Positional("criterion_id"), Verdict: ctx.Positional("verdict"), Actor: ctx.Positional("actor"), EvidenceReference: ctx.Positional("evidence_reference"), ReviewedAt: timestamppb.New(reviewed),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("record readiness human check", err, nil)
	}
	return response.Msg, nil
}

func (h *connectHandlers) policyCheck(_ cliapp.OperationContext) (*readinessv1.CheckPolicyProjectionResponse, error) {
	response, err := h.client.CheckPolicyProjection(context.Background(), connect.NewRequest(&readinessv1.CheckPolicyProjectionRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("check readiness policy", err, nil)
	}
	return response.Msg, nil
}

func reviewMutationReport(_ cliapp.OperationContext, response *readinessv1.ReviewResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Review %s is %s.", response.GetReviewKey(), response.GetStatus())}, Changes: []string{fmt.Sprintf("Goal: %s; findings: %d; evidence: %d", response.GetGoalRef(), len(response.GetFindings()), len(response.GetEvidence()))}, NextCommand: []string{fmt.Sprintf("deployment-manager readiness get --review-key %s", response.GetReviewKey())}}
}

func evidenceMutationReport(_ cliapp.OperationContext, response *readinessv1.ReportEvidenceResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Accepted %s evidence for %s.", response.GetCriterionId(), response.GetIdentityKey())}, Changes: []string{fmt.Sprintf("Producer binding: %s", response.GetProducerBinding())}}
}

func reviewListReport(_ cliapp.OperationContext, response *readinessv1.ReviewResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Review %s is %s.", response.GetReviewKey(), response.GetStatus())}, ResultsHeading: "Readiness evidence", Results: []string{fmt.Sprintf("%d evidence rows; %d unresolved findings; comparison=%s", len(response.GetEvidence()), len(response.GetFindings()), response.GetComparisonMode())}, ResultCount: 1, ListShaped: true}
}

func policyListReport(_ cliapp.OperationContext, response *readinessv1.CheckPolicyProjectionResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Policy v%d matches=%t.", response.GetPolicyVersion(), response.GetMatches())}, ResultsHeading: "Policy", Results: []string{fmt.Sprintf("%d criteria", response.GetCriterionCount())}, ResultCount: 1, ListShaped: true}
}

func reviewsListReport(_ cliapp.OperationContext, response *readinessv1.ListReviewsResponse) cliapp.ListReport {
	results := make([]string, 0, len(response.GetReviews()))
	for _, review := range response.GetReviews() {
		results = append(results, fmt.Sprintf("%s — %s", review.GetReviewKey(), review.GetStatus()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d readiness reviews.", response.GetCount())}, ResultsHeading: "Reviews", Results: results, ResultCount: len(results), ListShaped: true}
}

func waiversListReport(_ cliapp.OperationContext, response *readinessv1.ListReviewWaiversResponse) cliapp.ListReport {
	results := make([]string, 0, len(response.GetWaivers()))
	for _, waiver := range response.GetWaivers() {
		results = append(results, fmt.Sprintf("%s/%s", waiver.GetReviewKey(), waiver.GetCriterionId()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d readiness waivers.", response.GetCount())}, ResultsHeading: "Waivers", Results: results, ResultCount: len(results), ListShaped: true}
}

func parsePageSize(raw string) int32 {
	var value int
	if _, err := fmt.Sscan(raw, &value); err != nil || value < 1 || value > 500 {
		return 100
	}
	return int32(value)
}

func RegisterConnect(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newConnectHandlers(core)
	return cliapp.LoadFromManifestPrimitives(manifest, ConnectGroupName, map[string]cliapp.PrimitiveHandler{
		"ReadinessService.ReportEvidence":         cliapp.ProtoMutation(h.reportEvidence, evidenceMutationReport),
		"ReadinessService.PrepareReview":          cliapp.ProtoMutation(h.prepare, reviewMutationReport),
		"ReadinessService.GetReview":              cliapp.ProtoList(h.get, reviewListReport),
		"ReadinessService.ListReviews":            cliapp.ProtoList(h.list, reviewsListReport),
		"ReadinessService.SynchronizeGoalClosure": cliapp.ProtoMutation(h.sync, reviewMutationReport),
		"ReadinessService.ApproveReview":          cliapp.ProtoMutation(h.approve, reviewMutationReport),
		"ReadinessService.RecordHumanCheck":       cliapp.ProtoMutation(h.recordHumanCheck, reviewMutationReport),
		"ReadinessService.CheckPolicyProjection":  cliapp.ProtoList(h.policyCheck, policyListReport),
	})
}

func RegisterWaivers(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newConnectHandlers(core)
	return cliapp.LoadFromManifestPrimitives(manifest, WaiverGroupName, map[string]cliapp.PrimitiveHandler{
		"ReadinessService.ListReviewWaivers": cliapp.ProtoList(h.listWaivers, waiversListReport),
		"ReadinessService.CreateWaiver":      cliapp.ProtoMutation(h.createWaiver, reviewMutationReport),
	})
}
