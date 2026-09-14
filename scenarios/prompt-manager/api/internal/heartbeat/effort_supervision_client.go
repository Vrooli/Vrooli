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
				return mapEffortAssessment(a), nil
			}
		}
	}
	return nil, nil
}

func mapEffortAssessment(a *ampb.EffortAssessment) *SupervisionAssessment {
	if a == nil {
		return nil
	}
	return &SupervisionAssessment{
		ID: a.GetAssessmentId(), WakeID: a.GetIdempotencyKey(), RunID: a.GetSupervisorRunId(),
		Disposition: a.GetDisposition(), TargetRevisions: a.GetTargetRevisions(),
		EvidenceRefs: append([]string(nil), a.GetEvidenceRefs()...), SourceLedgerRef: a.GetSourceLedgerRef(),
		AllowanceRef: a.GetAllowanceRef(), ObservedUsage: a.GetObservedUsage(), RepairLinks: a.GetRepairLinks(),
	}
}

func appendUniqueRef(refs []string, values ...string) []string {
	seen := make(map[string]struct{}, len(refs)+len(values))
	for _, ref := range refs {
		if strings.TrimSpace(ref) != "" {
			seen[ref] = struct{}{}
		}
	}
	for _, ref := range values {
		if strings.TrimSpace(ref) == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}
	return refs
}

func boundedRefs(refs []string, limit int) []string {
	refs = appendUniqueRef(nil, refs...)
	if len(refs) > limit {
		return refs[:limit]
	}
	return refs
}

func mapEffortBoard(board *ampb.EffortBoard) *EffortDiscovery {
	cut := &EffortDiscovery{
		Coverage:          "complete",
		NextCursor:        board.GetNextPageToken(),
		QuotaObservations: append([]*ampb.EffortQuotaObservation(nil), board.GetQuotaObservations()...),
	}
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
		if enrollment.GetAuthorizedBy() == "" || (!enrollment.GetAutonomousSupervision() && len(enrollment.GetPermittedActions()) == 0) || enrollment.GetAuthorityExpiresAt() == nil || board.GetObservedAt() == nil || !enrollment.AuthorityExpiresAt.AsTime().After(board.ObservedAt.AsTime()) {
			observationOnly = true
			reason += "; qualified supervision mandate unavailable or expired; retain observation and owner reconciliation"
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
		anchor := dispatchAuthorizationAnchor(enrollment)
		namedWaits := boundedRefs(row.GetPendingOperations(), 16)
		detailRefs := appendUniqueRef(nil, "agent-manager:GetEffortBoard:"+enrollment.GetEffortRef())
		detailRefs = appendUniqueRef(detailRefs, row.GetEvidenceRefs()...)
		detailRefs = appendUniqueRef(detailRefs, namedWaits...)
		if assessment := row.GetLastAssessment(); assessment != nil {
			detailRefs = appendUniqueRef(detailRefs, assessment.GetEvidenceRefs()...)
			detailRefs = appendUniqueRef(detailRefs, assessment.GetSourceLedgerRef(), assessment.GetAllowanceRef())
			for _, link := range assessment.GetRepairLinks() {
				if link == nil {
					continue
				}
				detailRefs = appendUniqueRef(detailRefs, link.GetWorkRef(), link.GetAssigningOwnerRef())
				detailRefs = appendUniqueRef(detailRefs, link.GetCompletionEvidenceRefs()...)
			}
		}
		changedEvidence := appendUniqueRef(nil, row.GetEvidenceRefs()...)
		if row.GetChangeIdentity() != "" {
			changedEvidence = appendUniqueRef(changedEvidence, row.GetChangeIdentity())
		}
		if anchor {
			reason = "server-owned dispatch authorization anchor; no accepted effort or supervision/sample subject"
		}
		changedEvidence = boundedRefs(changedEvidence, 16)
		detailRefs = boundedRefs(detailRefs, 32)
		assessment := row.GetLastAssessment()
		cut.Efforts = append(cut.Efforts, EffortObservation{
			Priority:                    enrollment.GetSupervisionPriority(),
			RecoveryVerificationPending: verificationPending,
			ID:                          enrollment.GetEffortRef(), TargetRevision: enrollment.GetTargetRevision(),
			EvidenceRevision:       row.GetChangeIdentity(),
			SupervisorOwnerSubject: enrollment.GetSupervisorOwnerSubject(), SupervisorScope: enrollment.GetSupervisorScope(),
			Eligible:  !enrollment.GetWithdrawn() && valid && !anchor,
			Freshness: freshness, ObservationOnly: observationOnly,
			Retired: enrollment.GetWithdrawn(), WaitRef: strings.Join(namedWaits, ", "),
			BoardRef: "agent-manager:GetEffortBoard:" + enrollment.GetEffortRef(), Reason: reason,
			PriorAssessment: mapEffortAssessment(assessment), ChangedEvidence: changedEvidence,
			Usage: row.GetUsage(), NamedWaits: namedWaits, DetailRefs: detailRefs,
			QuotaObservations: append([]*ampb.EffortQuotaObservation(nil), row.GetQuotaObservations()...),
			RepairLinks:       assessment.GetRepairLinks(),
		})
	}
	cut.Error = strings.Join(findings, "; ")
	return cut
}

func dispatchAuthorizationAnchor(enrollment *ampb.EffortEnrollment) bool {
	if enrollment == nil || enrollment.GetWorkspace() != "" || enrollment.GetDestinationRef() != "" || len(enrollment.GetPermittedActions()) != 0 {
		return false
	}
	hasOwnerAnchor := enrollment.GetAuthorizedBy() != "" && enrollment.GetSupervisorOwnerSubject() != "" && enrollment.GetSupervisorScope() != "" && enrollment.GetAuthorizedBy() == enrollment.GetSupervisorOwnerSubject()
	if enrollment.GetDispatchAuthorization() == nil && !hasOwnerAnchor {
		return false
	}
	if len(enrollment.GetSubjects()) == 0 {
		return true
	}
	for _, subject := range enrollment.GetSubjects() {
		if subject == nil || subject.GetRole() != "supervisor" {
			return false
		}
	}
	return true
}
