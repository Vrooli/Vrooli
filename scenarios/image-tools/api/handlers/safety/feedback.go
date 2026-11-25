package safety

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	internaljobs "image-tools/internal/jobs"
	internalquarantine "image-tools/internal/quarantine"
	internalsafety "image-tools/internal/safety"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety"
)

// JobSource is the narrow job lookup the feedback surface needs. The jobs
// Manager satisfies it; declaring it here keeps the handler testable with a fake.
type JobSource interface {
	Get(ctx context.Context, id string) (internaljobs.Job, error)
}

// Deps wires the safety domain's runtime dependencies.
type Deps struct {
	// Policy is the RESOLVED Responsible-Use policy (preset + overrides).
	Policy internalsafety.Policy
	// Feedback persists the labelled refusal/flag corpus. Nil leaves the
	// feedback verbs unimplemented.
	Feedback *internalsafety.FeedbackStore
	// Jobs reads the reviewed job whose evidence a judgement snapshots.
	Jobs JobSource
	// Quarantine retains refused outputs for bounded admin review. Nil leaves
	// the review queue empty.
	Quarantine *internalquarantine.Store
	// RequireAdmin gates the quarantine surface on a public/cloud deployment.
	// Local deployments leave it false.
	RequireAdmin bool
	// AdminToken is the explicit public-deployment review token. When admin
	// review is required and this is empty, the surface is unavailable rather
	// than open.
	AdminToken string
	// Health reports the reviewer's live state for the policy summary.
	Health func() internalsafety.ReviewHealth
	Logger *log.Logger
}

// RecordSafetyFeedback records one operator judgement against a reviewed output.
func (h *connectHandler) RecordSafetyFeedback(ctx context.Context, req *connect.Request[safetyv1.RecordSafetyFeedbackRequest]) (*connect.Response[safetyv1.SafetyFeedbackCase], error) {
	if h.feedback == nil || h.jobs == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("safety feedback is not configured"))
	}
	label, err := feedbackLabelFromProto(req.Msg.GetLabel())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	job, err := h.jobs.Get(ctx, strings.TrimSpace(req.Msg.GetJobId()))
	if err != nil {
		return nil, h.jobError(err, "RecordSafetyFeedback")
	}
	observation, err := feedbackObservation(job, req.Msg.GetOutputRef())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	item, err := internalsafety.BuildFeedbackCase(observation, label, req.Msg.GetNote(), req.Msg.GetActor())
	if err != nil {
		return nil, h.feedbackError(err, "RecordSafetyFeedback")
	}
	stored, err := h.feedback.Record(ctx, item)
	if err != nil {
		return nil, h.feedbackError(err, "RecordSafetyFeedback")
	}
	return connect.NewResponse(feedbackCaseToProto(stored)), nil
}

// ClearSafetyFeedback removes one judgement.
func (h *connectHandler) ClearSafetyFeedback(ctx context.Context, req *connect.Request[safetyv1.ClearSafetyFeedbackRequest]) (*connect.Response[safetyv1.ClearSafetyFeedbackResponse], error) {
	if h.feedback == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("safety feedback is not configured"))
	}
	cleared, err := h.feedback.Clear(ctx, req.Msg.GetJobId(), req.Msg.GetOutputRef())
	if err != nil {
		return nil, h.feedbackError(err, "ClearSafetyFeedback")
	}
	return connect.NewResponse(&safetyv1.ClearSafetyFeedbackResponse{Cleared: cleared}), nil
}

// ListSafetyFeedback returns recorded judgements newest first.
func (h *connectHandler) ListSafetyFeedback(ctx context.Context, req *connect.Request[safetyv1.ListSafetyFeedbackRequest]) (*connect.Response[safetyv1.ListSafetyFeedbackResponse], error) {
	if h.feedback == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("safety feedback is not configured"))
	}
	filters := internalsafety.FeedbackFilters{
		ReasonCode: strings.TrimSpace(req.Msg.GetReasonCode()),
		ModelID:    strings.TrimSpace(req.Msg.GetModelId()),
		Operation:  strings.TrimSpace(req.Msg.GetOperation()),
		JobID:      strings.TrimSpace(req.Msg.GetJobId()),
	}
	if req.Msg.GetLabel() != safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_UNSPECIFIED {
		label, err := feedbackLabelFromProto(req.Msg.GetLabel())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		filters.Label = label
	}
	cases, total, err := h.feedback.List(ctx, filters, int(req.Msg.GetLimit()), int(req.Msg.GetOffset()))
	if err != nil {
		return nil, h.feedbackError(err, "ListSafetyFeedback")
	}
	resp := &safetyv1.ListSafetyFeedbackResponse{Total: int32(total)}
	for _, item := range cases {
		resp.Cases = append(resp.Cases, feedbackCaseToProto(item))
	}
	return connect.NewResponse(resp), nil
}

// GetSafetyFeedbackStats aggregates the labelled corpus.
func (h *connectHandler) GetSafetyFeedbackStats(ctx context.Context, req *connect.Request[safetyv1.GetSafetyFeedbackStatsRequest]) (*connect.Response[safetyv1.SafetyFeedbackStats], error) {
	if h.feedback == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("safety feedback is not configured"))
	}
	var since time.Time
	if ts := req.Msg.GetSince(); ts != nil {
		since = ts.AsTime()
	}
	stats, err := h.feedback.Stats(ctx, since)
	if err != nil {
		return nil, h.feedbackError(err, "GetSafetyFeedbackStats")
	}
	return connect.NewResponse(feedbackStatsToProto(stats)), nil
}

// ExportSafetyCorpus renders the labelled corpus as JSON Lines.
func (h *connectHandler) ExportSafetyCorpus(ctx context.Context, req *connect.Request[safetyv1.ExportSafetyCorpusRequest]) (*connect.Response[safetyv1.SafetyCorpus], error) {
	if h.feedback == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("safety feedback is not configured"))
	}
	filters := internalsafety.FeedbackFilters{}
	if req.Msg.GetLabel() != safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_UNSPECIFIED {
		label, err := feedbackLabelFromProto(req.Msg.GetLabel())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		filters.Label = label
	}
	if req.Msg.GetSince() != nil {
		filters.Since = req.Msg.GetSince().AsTime()
	}
	jsonl, count, err := h.feedback.Export(ctx, filters, int(req.Msg.GetLimit()))
	if err != nil {
		return nil, h.feedbackError(err, "ExportSafetyCorpus")
	}
	return connect.NewResponse(&safetyv1.SafetyCorpus{Jsonl: jsonl, Cases: int32(count)}), nil
}

// feedbackObservation reads the engine's safety_* result meta from a job.
func feedbackObservation(job internaljobs.Job, outputRef string) (internalsafety.FeedbackObservation, error) {
	meta := job.Meta
	review := strings.TrimSpace(meta[internaljobs.MetaSafetyReview])
	action := strings.TrimSpace(meta[internaljobs.MetaSafetyAction])
	evidence := strings.TrimSpace(meta[internaljobs.MetaSafetyEvidence])
	if review == "" && action == "" {
		return internalsafety.FeedbackObservation{}, internalsafety.ErrNoSafetyVerdict
	}
	if evidence == "" {
		return internalsafety.FeedbackObservation{}, internalsafety.ErrFeedbackNoEvidence
	}
	score, _ := strconv.ParseFloat(strings.TrimSpace(meta[internaljobs.MetaNSFWScore]), 64)
	observation := internalsafety.FeedbackObservation{
		JobID:         job.ID,
		OutputRef:     strings.TrimSpace(outputRef),
		Operation:     job.Operation,
		ModelID:       payloadModelID(job.Payload),
		Action:        action,
		ReasonCode:    strings.TrimSpace(meta[internaljobs.MetaSafetyReason]),
		SafetyReview:  review,
		Score:         score,
		Evidence:      evidence,
		ReviewedRefs:  splitRefList(meta[internaljobs.MetaSafetyReviewedRefs]),
		RefusedRefs:   splitRefList(meta[internaljobs.MetaSafetyRefusedRefs]),
		ShadowRefs:    splitRefList(meta[internaljobs.MetaSafetyShadowRefs]),
		SensitiveRefs: splitRefList(meta[internaljobs.MetaSensitiveRefs]),
	}
	return observation, nil
}

// payloadModelID reads the model that produced the job from its stored request.
func payloadModelID(payload []byte) string {
	var echo struct {
		ModelID string `json:"model_id"`
	}
	if err := json.Unmarshal(payload, &echo); err != nil {
		return ""
	}
	return strings.TrimSpace(echo.ModelID)
}

func splitRefList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if ref := strings.TrimSpace(part); ref != "" {
			out = append(out, ref)
		}
	}
	return out
}

func (h *connectHandler) feedbackError(err error, op string) error {
	switch {
	case errors.Is(err, internalsafety.ErrFeedbackLabel),
		errors.Is(err, internalsafety.ErrFeedbackNote):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, internalsafety.ErrNoSafetyVerdict),
		errors.Is(err, internalsafety.ErrFeedbackNoEvidence),
		errors.Is(err, internalsafety.ErrFeedbackRefAbsent):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	h.logf("%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, err)
}

func (h *connectHandler) jobError(err error, op string) error {
	if errors.Is(err, internaljobs.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	h.logf("%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, err)
}

func (h *connectHandler) logf(format string, args ...any) {
	if h.logger != nil {
		h.logger.Printf("safety."+format, args...)
	}
}

func feedbackLabelFromProto(label safetyv1.SafetyFeedbackLabel) (internalsafety.FeedbackLabel, error) {
	switch label {
	case safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_FALSE_POSITIVE:
		return internalsafety.FeedbackFalsePositive, nil
	case safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_CONFIRMED:
		return internalsafety.FeedbackConfirmed, nil
	case safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_UNCERTAIN:
		return internalsafety.FeedbackUncertain, nil
	default:
		return "", internalsafety.ErrFeedbackLabel
	}
}

func feedbackLabelToProto(label internalsafety.FeedbackLabel) safetyv1.SafetyFeedbackLabel {
	switch label {
	case internalsafety.FeedbackFalsePositive:
		return safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_FALSE_POSITIVE
	case internalsafety.FeedbackConfirmed:
		return safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_CONFIRMED
	case internalsafety.FeedbackUncertain:
		return safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_UNCERTAIN
	default:
		return safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_UNSPECIFIED
	}
}

func feedbackCaseToProto(item internalsafety.FeedbackCase) *safetyv1.SafetyFeedbackCase {
	out := &safetyv1.SafetyFeedbackCase{
		Id:              item.ID,
		JobId:           item.JobID,
		OutputRef:       item.OutputRef,
		Label:           feedbackLabelToProto(item.Label),
		Action:          item.Action,
		ReasonCode:      item.ReasonCode,
		SafetyReview:    item.SafetyReview,
		Score:           item.Score,
		Operation:       item.Operation,
		ModelId:         item.ModelID,
		ProfileName:     item.ProfileName,
		ReviewerVersion: item.ReviewerVersion,
		Note:            item.Note,
		Actor:           item.Actor,
		EvidenceJson:    item.Evidence,
		FindingJson:     item.Finding,
	}
	if !item.RecordedAt.IsZero() {
		out.RecordedAt = timestamppb.New(item.RecordedAt)
	}
	return out
}

func feedbackStatsToProto(stats internalsafety.FeedbackStats) *safetyv1.SafetyFeedbackStats {
	out := &safetyv1.SafetyFeedbackStats{
		Total:             stats.Total,
		FalsePositives:    stats.FalsePositives,
		Confirmed:         stats.Confirmed,
		Uncertain:         stats.Uncertain,
		FalsePositiveRate: stats.FalsePositiveRate,
	}
	if !stats.Since.IsZero() {
		out.Since = timestamppb.New(stats.Since)
	}
	for _, count := range stats.ByLabel {
		out.ByLabel = append(out.ByLabel, &safetyv1.SafetyFeedbackLabelCount{
			Label: feedbackLabelToProto(count.Label),
			Cases: count.Cases,
		})
	}
	for _, bucket := range stats.ByReason {
		out.ByReason = append(out.ByReason, &safetyv1.SafetyFeedbackReasonStat{
			ReasonCode:        bucket.Key,
			Cases:             bucket.Cases,
			FalsePositives:    bucket.FalsePositives,
			Confirmed:         bucket.Confirmed,
			FalsePositiveRate: bucket.FalsePositiveRate,
		})
	}
	for _, bucket := range stats.ByProfile {
		out.ByProfile = append(out.ByProfile, &safetyv1.SafetyFeedbackProfileStat{
			ProfileName:       bucket.Key,
			Cases:             bucket.Cases,
			FalsePositives:    bucket.FalsePositives,
			Confirmed:         bucket.Confirmed,
			FalsePositiveRate: bucket.FalsePositiveRate,
		})
	}
	return out
}
