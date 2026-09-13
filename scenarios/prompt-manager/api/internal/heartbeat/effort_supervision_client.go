package heartbeat

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	amconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// DiscoverEfforts consumes AM's canonical board. AM's owner scheduler reconciles
// discovery; this read never enrolls efforts or synthesizes a grant.
func (c *AgentManagerClient) DiscoverEfforts(ctx context.Context, limit int, cursor string) (*EffortDiscovery, error) {
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("AM effort page limit must be between 1 and 100")
	}
	base, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	client := amconnect.NewAgentManagerServiceClient(c.httpClient, base)
	response, err := client.GetEffortBoard(ctx, connect.NewRequest(&ampb.GetEffortBoardRequest{PageSize: uint32(limit), PageToken: cursor}))
	if err != nil {
		return nil, err
	}
	return mapEffortBoard(response.Msg), nil
}

// GetEffortAssessment uses the owner's exact-effort projection, not the current
// discovery page. At most MaxEffortsPerWake reads occur on terminal recovery.
func (c *AgentManagerClient) GetEffortAssessment(ctx context.Context, effortID string) (*SupervisionAssessment, error) {
	base, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	client := amconnect.NewAgentManagerServiceClient(c.httpClient, base)
	response, err := client.GetEffortBoard(ctx, connect.NewRequest(&ampb.GetEffortBoardRequest{EffortRef: effortID, PageSize: 1}))
	if err != nil {
		return nil, err
	}
	for _, row := range response.Msg.GetRows() {
		if row.GetEnrollment().GetEffortRef() != effortID {
			continue
		}
		a := row.GetLastAssessment()
		if a == nil {
			return nil, nil
		}
		for _, id := range a.GetEffortRefs() {
			if id == effortID {
				return &SupervisionAssessment{ID: a.GetAssessmentId(), WakeID: a.GetIdempotencyKey(), RunID: a.GetSupervisorRunId(), Disposition: a.GetDisposition(), TargetRevisions: a.GetTargetRevisions()}, nil
			}
		}
	}
	return nil, nil
}

func mapEffortBoard(board *ampb.EffortBoard) *EffortDiscovery {
	cut := &EffortDiscovery{Coverage: "complete", NextCursor: board.GetNextPageToken()}
	if board.GetPartial() || cut.NextCursor != "" {
		cut.Coverage = "partial"
	}
	var findings []string
	for _, finding := range board.GetDiscovery().GetFindings() {
		findings = append(findings, finding.GetCode()+": "+finding.GetReason())
	}
	cut.Error = strings.Join(findings, "; ")
	for _, row := range board.GetRows() {
		enrollment := row.GetEnrollment()
		id := enrollment.GetEffortRef()
		if strings.TrimSpace(id) == "" || len(id) > 512 {
			cut.Coverage = "partial"
			findings = append(findings, "observation excluded: missing or over-limit effort identity")
			continue
		}
		valid := strings.TrimSpace(row.GetChangeIdentity()) != ""
		reason := row.GetRationale()
		freshness := strings.ToLower(strings.TrimPrefix(row.GetFreshness().String(), "EFFORT_FRESHNESS_"))
		observationOnly := row.GetFreshness() != ampb.EffortFreshness_EFFORT_FRESHNESS_FRESH
		if observationOnly {
			reason += "; source freshness=" + freshness + "; uncertainty assessment only, no steering"
		}
		if strings.TrimSpace(enrollment.GetTargetRevision()) == "" {
			observationOnly = true
			reason += "; accepted target revision unknown; retain exact empty revision for assessment, no steering"
		}
		if enrollment.GetAuthorizedBy() == "" || len(enrollment.GetPermittedActions()) == 0 || enrollment.GetAuthorityExpiresAt() == nil || board.GetObservedAt() == nil || !enrollment.AuthorityExpiresAt.AsTime().After(board.ObservedAt.AsTime()) {
			observationOnly = true
			reason += "; qualified steering grant unavailable or expired; observation and owner repair coordination remain distinct from business mutation"
		}
		if !valid {
			reason += "; observation excluded: evidence identity unavailable"
			cut.Coverage = "partial"
			findings = append(findings, id+": observation excluded: evidence identity unavailable")
		}
		verificationPending := false
		for _, directive := range row.GetDirectives() {
			state := directive.GetRecoveryVerification().GetState()
			if directive.GetDelivery() == ampb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED &&
				directive.GetRecoveryExpectation() != nil && directive.GetSupersededBy() == "" &&
				directive.GetTargetRevision() == enrollment.GetTargetRevision() && (state == "" || state == "pending") {
				verificationPending = true
			}
		}
		cut.Efforts = append(cut.Efforts, EffortObservation{
			RecoveryVerificationPending: verificationPending,
			ID:                          enrollment.GetEffortRef(), TargetRevision: enrollment.GetTargetRevision(),
			EvidenceRevision:       row.GetChangeIdentity(),
			SupervisorOwnerSubject: enrollment.GetSupervisorOwnerSubject(), SupervisorScope: enrollment.GetSupervisorScope(),
			Eligible:  !enrollment.GetWithdrawn() && valid && row.GetNextAction() != "authorization-only",
			Freshness: freshness, ObservationOnly: observationOnly,
			Retired: enrollment.GetWithdrawn(), WaitRef: strings.Join(row.GetPendingOperations(), ", "),
			BoardRef: "agent-manager:GetEffortBoard:" + enrollment.GetEffortRef(), Reason: reason,
		})
	}
	cut.Error = strings.Join(findings, "; ")
	return cut
}
