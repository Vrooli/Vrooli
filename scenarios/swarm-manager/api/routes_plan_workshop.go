package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/planclient"
	"swarm-manager/internal/planworkshop"
	"swarm-manager/internal/review"
	"swarm-manager/internal/transitions"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func (s *Server) registerPlanWorkshopRoutes(dataRoot string) {
	if s.backlogHandler == nil {
		return
	}
	if _, err := planworkshop.MigrateLegacyHistory(dataRoot, time.Now()); err != nil {
		// The migration is additive. Refusing to start the new operator surface
		// because historical metadata is malformed would preserve the retired
		// workflow as the only escape hatch, so retain the error for operators
		// while still serving new workshops.
		log.Printf("plan workshop legacy history migration: %v", err)
	}
	plans := planclient.NewConnectClient(nil, nil)
	service := planworkshop.NewService(planworkshop.NewStore(dataRoot), func(subject planworkshop.Subject) (string, string, string, error) {
		switch subject.Kind {
		case planworkshop.SubjectBacklog:
			parts := strings.SplitN(subject.Ref, "/", 2)
			if len(parts) != 2 {
				return "", "", "", fmt.Errorf("backlog subject ref must be kind/name")
			}
			kind, err := backlog.ParseBacklogKind(parts[0])
			if err != nil {
				return "", "", "", err
			}
			item, err := s.backlogHandler.Store().LoadItem(kind, parts[1])
			if err != nil {
				return "", "", "", err
			}
			id := planID(item.PlanRef)
			hash, err := currentPlanHash(context.Background(), plans, id)
			if err != nil {
				return "", "", "", err
			}
			return workshopSubjectVersion(item), id, hash, nil
		default:
			return "", "", "", fmt.Errorf("unsupported workshop subject")
		}
	})
	// Reconciliation is an internal continuation of one accepted response. The
	// workflow receives an immutable session snapshot and has no route back to
	// live Swarm or Plan Manager mutation APIs.
	if registry, err := transitions.LoadDir(filepath.Join(pathutil.ResolveScenarioRoot("swarm-manager"), ".vrooli", "swarm-transitions")); err == nil {
		workflow := agentmanager.NewWorkflowService()
		if s.agentActivitySvc != nil {
			workflow.SetWorkflowActivityRecorder(s.agentActivitySvc)
		}
		s.planWorkshopWorkflow = workflow
		service.SetReconciliationStarter(func(ctx context.Context, session planworkshop.Session, response planworkshop.Response) (planworkshop.WorkflowProvenance, error) {
			locator, err := registry.ResolveWorkflow("plan.workshop.reconcile")
			if err != nil {
				return planworkshop.WorkflowProvenance{}, err
			}
			parts := strings.SplitN(session.Subject.Ref, "/", 2)
			if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
				return planworkshop.WorkflowProvenance{}, fmt.Errorf("backlog workshop subject ref must be kind/name")
			}
			accepted, err := planWorkshopAcceptedProposalPayloads(s.agentSessionStore, response.Accepted)
			if err != nil {
				return planworkshop.WorkflowProvenance{}, err
			}
			planSnapshot, err := planWorkshopCanonicalPlanSnapshot(ctx, plans, session.PlanID)
			if err != nil {
				return planworkshop.WorkflowProvenance{}, err
			}
			input, err := structpb.NewValue(map[string]any{
				"subject": session.Subject,
				"snapshot": map[string]any{
					"workshopId": session.ID, "subjectVersion": session.SubjectVersion,
					"planId": session.PlanID, "planContentHash": session.PlanContentHash, "plan": planSnapshot,
					"packet": session.Packet,
				},
				"response":           response,
				"accepted_proposals": accepted,
			})
			if err != nil {
				return planworkshop.WorkflowProvenance{}, fmt.Errorf("encode reconciliation snapshot: %w", err)
			}
			started, err := workflow.StartWorkflow(ctx, agentmanager.Invocation{
				Owner: locator.Owner, WorkflowKey: locator.Key, Input: input,
				IdempotencyKey: "plan-workshop/" + session.ID + "/" + response.ID,
				FirstRunNodeID: "reconcile",
				Activity:       &agentmanager.WorkflowActivity{OwnerType: "backlog", OwnerKind: parts[0], OwnerName: parts[1], Purpose: "workshop"},
			})
			if err != nil {
				return planworkshop.WorkflowProvenance{}, err
			}
			return planworkshop.WorkflowProvenance{Transition: "plan.workshop.reconcile", ExecutionID: started.ExecutionID, DefinitionDigest: started.DefinitionDigest, RunID: started.RunID, StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}, nil
		})
		service.SetReviewStarter(func(ctx context.Context, session planworkshop.Session) (planworkshop.WorkflowProvenance, error) {
			locator, err := registry.ResolveWorkflow("plan.workshop.review")
			if err != nil {
				return planworkshop.WorkflowProvenance{}, err
			}
			planSnapshot, err := planWorkshopCanonicalPlanSnapshot(ctx, plans, session.PlanID)
			if err != nil {
				return planworkshop.WorkflowProvenance{}, err
			}
			input, err := structpb.NewValue(map[string]any{
				"subject": session.Subject,
				"snapshot": map[string]any{
					"workshopId": session.ID, "subjectVersion": session.SubjectVersion,
					"planId": session.PlanID, "planContentHash": session.PlanContentHash, "plan": planSnapshot,
				},
			})
			if err != nil {
				return planworkshop.WorkflowProvenance{}, fmt.Errorf("encode review snapshot: %w", err)
			}
			parts := strings.SplitN(session.Subject.Ref, "/", 2)
			if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
				return planworkshop.WorkflowProvenance{}, fmt.Errorf("backlog workshop subject ref must be kind/name")
			}
			started, err := workflow.StartWorkflow(ctx, agentmanager.Invocation{
				Owner: locator.Owner, WorkflowKey: locator.Key, Input: input,
				IdempotencyKey: "plan-workshop/review/" + session.ID + "/" + session.SubjectVersion, FirstRunNodeID: "review",
				Activity: &agentmanager.WorkflowActivity{OwnerType: "backlog", OwnerKind: parts[0], OwnerName: parts[1], Purpose: "workshop"},
			})
			if err != nil {
				return planworkshop.WorkflowProvenance{}, err
			}
			return planworkshop.WorkflowProvenance{Transition: "plan.workshop.review", ExecutionID: started.ExecutionID, DefinitionDigest: started.DefinitionDigest, RunID: started.RunID, StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}, nil
		})
		service.SetReviewCollector(func(ctx context.Context, session planworkshop.Session, provenance planworkshop.WorkflowProvenance) (planworkshop.ReviewResult, error) {
			completion, err := workflow.CollectWorkflow(ctx, provenance.ExecutionID)
			if err != nil {
				return planworkshop.ReviewResult{}, err
			}
			if completion.ExecutionID != provenance.ExecutionID || completion.DefinitionDigest != provenance.DefinitionDigest || completion.Status != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
				return planworkshop.ReviewResult{}, fmt.Errorf("review workflow terminal snapshot is not applicable")
			}
			if !planWorkshopReviewInputMatches(completion.Input, session) {
				return planworkshop.ReviewResult{}, fmt.Errorf("review workflow snapshot does not match the Plan Workshop")
			}
			if completion.Output == nil {
				return planworkshop.ReviewResult{}, fmt.Errorf("review workflow output is missing")
			}
			output, ok := completion.Output.AsInterface().(map[string]any)
			if !ok {
				return planworkshop.ReviewResult{}, fmt.Errorf("review workflow output is invalid")
			}
			raw, ok := output["result"]
			if !ok {
				return planworkshop.ReviewResult{}, fmt.Errorf("review workflow result is missing")
			}
			data, err := json.Marshal(raw)
			if err != nil {
				return planworkshop.ReviewResult{}, err
			}
			var result planworkshop.ReviewResult
			if err := json.Unmarshal(data, &result); err != nil {
				return planworkshop.ReviewResult{}, fmt.Errorf("decode typed review result: %w", err)
			}
			return result, nil
		})
		service.SetReconciliationCollector(func(ctx context.Context, session planworkshop.Session, response planworkshop.Response, provenance planworkshop.WorkflowProvenance) (planworkshop.ReconciliationResult, error) {
			completion, err := workflow.CollectWorkflow(ctx, provenance.ExecutionID)
			if err != nil {
				return planworkshop.ReconciliationResult{}, err
			}
			if completion.ExecutionID != provenance.ExecutionID || completion.DefinitionDigest != provenance.DefinitionDigest || completion.Status != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED || completion.Output == nil {
				return planworkshop.ReconciliationResult{}, fmt.Errorf("reconciliation workflow terminal snapshot is not applicable")
			}
			if !planWorkshopReconciliationInputMatches(completion.Input, session, response) {
				return planworkshop.ReconciliationResult{}, fmt.Errorf("reconciliation workflow snapshot does not match the Plan Workshop response")
			}
			output, ok := completion.Output.AsInterface().(map[string]any)
			if !ok {
				return planworkshop.ReconciliationResult{}, fmt.Errorf("reconciliation workflow output is invalid")
			}
			raw, ok := output["result"]
			if !ok {
				return planworkshop.ReconciliationResult{}, fmt.Errorf("reconciliation workflow result is missing")
			}
			data, err := json.Marshal(raw)
			if err != nil {
				return planworkshop.ReconciliationResult{}, err
			}
			var result planworkshop.ReconciliationResult
			if err := json.Unmarshal(data, &result); err != nil {
				return planworkshop.ReconciliationResult{}, fmt.Errorf("decode typed reconciliation result: %w", err)
			}
			return result, nil
		})
		service.SetCandidateCreator(func(ctx context.Context, workshop planworkshop.Session, response planworkshop.Response, provenance planworkshop.WorkflowProvenance, raw json.RawMessage) (planworkshop.CandidateReference, error) {
			candidatePlan := &sharedv1.Plan{}
			if err := protojson.Unmarshal(raw, candidatePlan); err != nil {
				return planworkshop.CandidateReference{}, fmt.Errorf("decode whole-plan reconciliation candidate: %w", err)
			}
			candidatePlan.Id = ""
			candidate, err := plans.CreateCandidateRevision(ctx, planclient.CandidateRevisionInput{PlanID: workshop.PlanID, ExpectedBaseContentHash: workshop.PlanContentHash, ProposalProvenance: "swarm-manager:plan-workshop/" + workshop.ID + "/" + response.ID + "/" + provenance.ExecutionID, CandidatePlan: candidatePlan})
			if err != nil {
				return planworkshop.CandidateReference{}, err
			}
			if candidate == nil || strings.TrimSpace(candidate.GetId()) == "" {
				return planworkshop.CandidateReference{}, fmt.Errorf("plan manager omitted the reconciliation candidate id")
			}
			preview, err := plans.PreviewCandidateRevision(ctx, candidate.GetId())
			if err != nil {
				return planworkshop.CandidateReference{}, err
			}
			findings := make([]string, 0, len(preview.GetDiagnostics()))
			diagnostics := make([]planworkshop.CandidateDiagnostic, 0, len(preview.GetDiagnostics()))
			for _, diagnostic := range preview.GetDiagnostics() {
				if diagnostic == nil {
					continue
				}
				diagnostics = append(diagnostics, planworkshop.CandidateDiagnostic{Severity: diagnostic.GetSeverity(), Code: diagnostic.GetCode(), Location: diagnostic.GetLocation(), Message: diagnostic.GetMessage(), Guidance: diagnostic.GetGuidance()})
				if strings.TrimSpace(diagnostic.GetMessage()) != "" {
					findings = append(findings, diagnostic.GetMessage())
				}
			}
			changes := make([]planworkshop.CandidateFieldChange, 0, len(preview.GetDiff().GetChanges()))
			for _, change := range preview.GetDiff().GetChanges() {
				if change != nil {
					changes = append(changes, planworkshop.CandidateFieldChange{Field: change.GetField(), BeforeJSON: change.GetBeforeJson(), AfterJSON: change.GetAfterJson()})
				}
			}
			impact := preview.GetImpact()
			return planworkshop.CandidateReference{ID: candidate.GetId(), PlanID: workshop.PlanID, ExpectedBaseContentHash: workshop.PlanContentHash, QualityStatus: preview.GetQualityStatus(), QualityFindings: findings, Diff: changes, Diagnostics: diagnostics, Impact: planworkshop.CandidateImpact{BeforeGrade: impact.GetBeforeGrade(), AfterGrade: impact.GetAfterGrade(), AddedIssueCodes: impact.GetAddedIssueCodes(), ClearedIssueCodes: impact.GetClearedIssueCodes(), ExecutionGradeRegression: impact.GetExecutionGradeRegression()}}, nil
		})
		service.SetCandidateApplier(func(ctx context.Context, workshop planworkshop.Session, candidate planworkshop.CandidateReference, acknowledgeQualityImpact bool) error {
			result, err := plans.ApplyCandidateRevision(ctx, candidate.ID, candidate.ExpectedBaseContentHash, acknowledgeQualityImpact)
			if err != nil {
				return err
			}
			if result.GetPlan() == nil || result.GetPlan().GetId() != workshop.PlanID {
				return fmt.Errorf("candidate does not apply to this workshop's canonical plan")
			}
			if workshop.Subject.Kind != planworkshop.SubjectBacklog {
				return nil
			}
			parts := strings.SplitN(workshop.Subject.Ref, "/", 2)
			if len(parts) != 2 {
				return fmt.Errorf("backlog workshop subject ref must be kind/name")
			}
			kind, err := backlog.ParseBacklogKind(parts[0])
			if err != nil {
				return err
			}
			item, err := s.backlogHandler.Store().LoadItem(kind, parts[1])
			if err != nil {
				return err
			}
			if item.PlanRef == nil || item.PlanRef.PlanID != workshop.PlanID {
				return fmt.Errorf("backlog item no longer owns this canonical plan")
			}
			item.PlanAcceptance = nil
			item.Updated = time.Now().UTC().Format(time.RFC3339)
			return s.backlogHandler.Store().SaveItem(item)
		})
		service.SetCandidateDiscarder(func(ctx context.Context, _ planworkshop.Session, candidate planworkshop.CandidateReference, reason string) error {
			if strings.TrimSpace(reason) == "" {
				reason = "ignored by operator in Plan Workshop"
			}
			_, err := plans.DiscardCandidateRevision(ctx, candidate.ID, reason)
			return err
		})
		service.SetProposalRecorder(func(ctx context.Context, workshop planworkshop.Session, provenance planworkshop.WorkflowProvenance, drafts []planworkshop.ProposalDraft) (string, []planworkshop.ProposalRef, error) {
			if s.agentSessionStore == nil {
				return "", nil, fmt.Errorf("Agent Session proposal store is not configured")
			}
			sessionID := planWorkshopAgentSessionID(workshop.ID, provenance.ExecutionID)
			target := planWorkshopProposalTarget(workshop.Subject)
			_, err := s.agentSessionStore.LoadSession(sessionID)
			if err != nil {
				if !errors.Is(err, agentsessions.ErrNotFound) {
					return "", nil, err
				}
				now := time.Now().UTC().Format(time.RFC3339Nano)
				stored := agentsessions.Session{ID: sessionID, Title: "Plan Workshop review for " + workshop.Subject.Ref, Kind: agentsessions.KindSwarmOperations, Status: agentsessions.StatusProposalReady, SkillID: agentsessions.SkillProposals, RunID: provenance.RunID, CreatedAt: now, UpdatedAt: now, ProposalTarget: &target}
				if err := s.agentSessionStore.CreateSession(stored); err != nil {
					return "", nil, err
				}
			}
			refs := make([]planworkshop.ProposalRef, 0, len(drafts))
			createdAt := time.Now().UTC().Format(time.RFC3339Nano)
			for index, draft := range drafts {
				proposalID := planWorkshopProposalID(provenance.ExecutionID, index)
				proposal := agentsessions.Proposal{
					ID: proposalID, Kind: agentsessions.ProposalMutationList, Status: agentsessions.ProposalStatusReady,
					Summary: strings.TrimSpace(draft.Summary), PayloadJSON: string(draft.Payload), Target: &target,
					CreatedAt: createdAt, UpdatedAt: createdAt,
					Attribution: &agentsessions.Attribution{Type: agentsessions.AttributionAgent, RunID: provenance.RunID, SessionID: sessionID, SessionKind: agentsessions.KindSwarmOperations, Source: "plan-workshop/" + workshop.ID},
				}
				if err := s.agentSessionStore.SaveProposal(sessionID, proposal); err != nil {
					return "", nil, err
				}
				refs = append(refs, planworkshop.ProposalRef{SessionID: sessionID, ProposalID: proposalID, ApplyMode: draft.ApplyMode})
			}
			return sessionID, refs, nil
		})
		service.SetDirectProposalApplier(func(ctx context.Context, ref planworkshop.ProposalRef, actor string) error {
			if s.agentSessionSvc == nil || s.agentSessionStore == nil {
				return fmt.Errorf("Agent Session mutation proposal service is not configured")
			}
			session, err := s.agentSessionStore.LoadSession(ref.SessionID)
			if err != nil {
				return err
			}
			for _, proposal := range session.Proposals {
				if proposal.ID != ref.ProposalID {
					continue
				}
				if proposal.Status == agentsessions.ProposalStatusApplied {
					return nil
				}
				if followUp, ok := decodeFollowUpProposal(proposal.PayloadJSON); ok {
					if s.executionSvc == nil {
						return fmt.Errorf("execution service is not configured for follow-up proposal")
					}
					if err := followUp.Validate(); err != nil {
						return err
					}
					followUpType := "followup"
					if followUp.Route == "work.correct" {
						followUpType = "fixup"
					}
					if _, err := s.executionSvc.FollowUp(ctx, execution.FollowUpRequest{
						ExecutionID:      followUp.SourceExecutionID,
						FollowUpType:     followUpType,
						Context:          followUp.Rationale,
						SourceProposalID: ref.ProposalID,
						SourceReviewRef:  followUp.SourceReviewRef,
					}); err != nil {
						return err
					}
					proposal.Status = agentsessions.ProposalStatusApplied
					proposal.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
					proposal.Decisions = append(proposal.Decisions, agentsessions.ProposalDecision{Kind: "apply", Note: "accepted in Plan Workshop by " + strings.TrimSpace(actor), DecidedAt: proposal.UpdatedAt})
					return s.agentSessionStore.SaveProposal(ref.SessionID, proposal)
				}
				if proposal.Kind != agentsessions.ProposalMutationList {
					return fmt.Errorf("direct Plan Workshop proposal is not a mutation list")
				}
				_, err := s.agentSessionSvc.DecideMutationListProposal(ctx, ref.SessionID, ref.ProposalID, nil, "accepted in Plan Workshop by "+strings.TrimSpace(actor))
				return err
			}
			return fmt.Errorf("direct Plan Workshop proposal is missing from its Agent Session")
		})
	}
	attachReviewFinding := func(ctx context.Context, kind, name string, round review.Round) {
		subjectKind := planworkshop.SubjectBacklog
		subjectRef := kind + "/" + name
		summary := strings.TrimSpace(round.AgentAssessment)
		if summary == "" {
			summary = "Review result requires operator attention"
		}
		disposition := planWorkshopDisposition(round)
		severity := "info"
		if disposition.Kind == "attention" {
			severity = "attention"
		}
		finding := planworkshop.Finding{
			ID: "review/" + kind + "/" + name + "/" + fmt.Sprint(round.RoundNum), Severity: severity, Summary: summary,
			Evidence: "review/" + kind + "/" + name + "/round/" + fmt.Sprint(round.RoundNum), Disposition: &disposition,
		}
		if _, err := service.AttachFinding(planworkshop.Subject{Kind: subjectKind, Ref: subjectRef}, finding); err != nil {
			log.Printf("plan workshop review evidence projection %s/%s round %d: %v", kind, name, round.RoundNum, err)
			return
		}
		if err := s.attachFollowUpProposal(ctx, service, planworkshop.Subject{Kind: subjectKind, Ref: subjectRef}, finding.Evidence, round.ExecutionID, disposition); err != nil {
			log.Printf("plan workshop review follow-up proposal %s/%s round %d: %v", kind, name, round.RoundNum, err)
		}
	}
	if s.reviewSvc != nil {
		s.reviewSvc.SetRoundTerminalObserver(attachReviewFinding)
	}
	planworkshop.NewHandler(service).RegisterRoutes(s.router)
}

func planWorkshopDisposition(round review.Round) planworkshop.Disposition {
	if round.Disposition != nil {
		return planworkshop.Disposition{Kind: round.Disposition.Kind, Rationale: round.Disposition.Rationale, Confidence: round.Disposition.Confidence, Scope: round.Disposition.Scope}
	}
	switch round.Classification {
	case "ready", "ready_with_notes", "delivered":
		return planworkshop.Disposition{Kind: "archive", Rationale: "Review evidence supports the completed result.", Confidence: "medium"}
	case "needs_work", "partial":
		return planworkshop.Disposition{Kind: "follow_up", Rationale: "Review evidence identifies remaining work.", Confidence: "medium"}
	default:
		return planworkshop.Disposition{Kind: "attention", Rationale: "Review did not produce a safe terminal recommendation.", Confidence: "low"}
	}
}

type followUpProposalEnvelope struct {
	Form     string                        `json:"form"`
	FollowUp planworkshop.FollowUpProposal `json:"follow_up"`
	Policy   string                        `json:"policy"`
}

func decodeFollowUpProposal(payload string) (planworkshop.FollowUpProposal, bool) {
	var envelope followUpProposalEnvelope
	if json.Unmarshal([]byte(payload), &envelope) != nil || envelope.Form != "follow_up" {
		return planworkshop.FollowUpProposal{}, false
	}
	return envelope.FollowUp, true
}

// attachFollowUpProposal persists a typed, attributable current-work proposal
// in the one Agent Session proposal store and projects its reference into the
// Plan Workshop. It never creates a run merely because a review finished.
// An explicit, opt-in bounded policy may create only high-confidence current
// work; otherwise the proposal remains operator-authorized.
func (s *Server) attachFollowUpProposal(ctx context.Context, workshop *planworkshop.Service, subject planworkshop.Subject, sourceRef, executionID string, disposition planworkshop.Disposition) error {
	if disposition.Kind != "follow_up" || strings.TrimSpace(executionID) == "" || s.agentSessionStore == nil {
		return nil
	}
	route := "work.follow_up"
	if strings.Contains(strings.ToLower(disposition.Scope), "correct") {
		route = "work.correct"
	}
	proposal := planworkshop.FollowUpProposal{
		Route: route, Target: "current_work", SourceReviewRef: sourceRef, SourceExecutionID: executionID,
		Rationale: disposition.Rationale, Confidence: disposition.Confidence, Scope: disposition.Scope,
	}
	if err := proposal.Validate(); err != nil {
		return err
	}
	policy := "operator_required"
	if automaticFollowUpAllowed(proposal) {
		policy = "automatic_allowed"
	}
	session, err := workshop.OpenOrGet(subject, nil)
	if err != nil {
		return err
	}
	agentSessionID := planWorkshopAgentSessionID(session.ID, sourceRef)
	target := planWorkshopProposalTarget(subject)
	if _, err := s.agentSessionStore.LoadSession(agentSessionID); err != nil {
		if !errors.Is(err, agentsessions.ErrNotFound) {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		if err := s.agentSessionStore.CreateSession(agentsessions.Session{ID: agentSessionID, Title: "Follow-up proposal for " + subject.Ref, Kind: agentsessions.KindSwarmOperations, Status: agentsessions.StatusProposalReady, SkillID: agentsessions.SkillProposals, CreatedAt: now, UpdatedAt: now, ProposalTarget: &target}); err != nil {
			return err
		}
	}
	payload, err := json.Marshal(followUpProposalEnvelope{Form: "follow_up", FollowUp: proposal, Policy: policy})
	if err != nil {
		return err
	}
	proposalID := planWorkshopProposalID(sourceRef, 0)
	alreadyStored := false
	if saved, err := s.agentSessionStore.LoadSession(agentSessionID); err == nil {
		for _, existing := range saved.Proposals {
			if existing.ID == proposalID {
				alreadyStored = true
				break
			}
		}
	} else {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	stored := agentsessions.Proposal{ID: proposalID, Kind: agentsessions.ProposalMutationList, Status: agentsessions.ProposalStatusReady, Summary: "Follow up on " + subject.Ref, PayloadJSON: string(payload), Target: &target, CreatedAt: now, UpdatedAt: now, Attribution: &agentsessions.Attribution{Type: agentsessions.AttributionAgent, SessionID: agentSessionID, SessionKind: agentsessions.KindSwarmOperations, Source: sourceRef}}
	if !alreadyStored {
		if err := s.agentSessionStore.SaveProposal(agentSessionID, stored); err != nil {
			return err
		}
		if s.emitter != nil {
			s.emitter.EmitAgentSessionProposalCreated(agentSessionID, map[string]any{"proposal_id": proposalID, "source_review_ref": sourceRef, "route": proposal.Route, "policy": policy, "confidence": proposal.Confidence, "scope": proposal.Scope})
		}
	}
	if _, err := workshop.AttachProposal(subject, planworkshop.ProposalRef{SessionID: agentSessionID, ProposalID: proposalID, ApplyMode: "direct"}); err != nil {
		return err
	}
	if alreadyStored || policy != "automatic_allowed" || s.executionSvc == nil {
		return nil
	}
	followUpType := "followup"
	if proposal.Route == "work.correct" {
		followUpType = "fixup"
	}
	if _, err := s.executionSvc.FollowUp(ctx, execution.FollowUpRequest{ExecutionID: proposal.SourceExecutionID, FollowUpType: followUpType, Context: proposal.Rationale, SourceProposalID: proposalID, SourceReviewRef: proposal.SourceReviewRef}); err != nil {
		return err
	}
	stored.Status = agentsessions.ProposalStatusApplied
	stored.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	stored.Decisions = append(stored.Decisions, agentsessions.ProposalDecision{Kind: "automatic_apply", Note: "bounded follow-up policy", DecidedAt: stored.UpdatedAt})
	if err := s.agentSessionStore.SaveProposal(agentSessionID, stored); err != nil {
		return err
	}
	if s.emitter != nil {
		s.emitter.EmitAgentSessionProposalApplied(agentSessionID, map[string]any{"proposal_id": proposalID, "source_review_ref": sourceRef, "policy": policy})
	}
	return nil
}

// automaticFollowUpAllowed is intentionally narrow and disabled by default.
// The opt-in environment policy is bounded to one high-confidence correction
// of the current execution; FollowUp's source-proposal deduplication provides
// the durable once-only guard and the proposal/event records provide audit.
func automaticFollowUpAllowed(proposal planworkshop.FollowUpProposal) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SWARM_MANAGER_AUTO_FOLLOW_UP"))) {
	case "1", "true", "yes", "on":
	default:
		return false
	}
	return proposal.Target == "current_work" && proposal.Confidence == "high"
}

func planWorkshopReviewInputMatches(input *structpb.Value, session planworkshop.Session) bool {
	if input == nil {
		return false
	}
	payload, ok := input.AsInterface().(map[string]any)
	if !ok {
		return false
	}
	subject, ok := payload["subject"].(map[string]any)
	if !ok || subject["kind"] != string(session.Subject.Kind) || subject["ref"] != session.Subject.Ref {
		return false
	}
	snapshot, ok := payload["snapshot"].(map[string]any)
	return ok && snapshot["workshopId"] == session.ID && snapshot["subjectVersion"] == session.SubjectVersion && snapshot["planContentHash"] == session.PlanContentHash
}

func planWorkshopReconciliationInputMatches(input *structpb.Value, session planworkshop.Session, response planworkshop.Response) bool {
	if !planWorkshopReviewInputMatches(input, session) {
		return false
	}
	payload, _ := input.AsInterface().(map[string]any)
	stored, ok := payload["response"].(map[string]any)
	return ok && stored["id"] == response.ID && stored["subject_version"] == response.SubjectVersion && stored["idempotency_key"] == response.IdempotencyKey
}

func planWorkshopAcceptedProposalPayloads(store agentsessions.Store, refs []planworkshop.ProposalRef) ([]map[string]any, error) {
	if store == nil {
		return nil, fmt.Errorf("Agent Session proposal store is not configured")
	}
	accepted := make([]map[string]any, 0, len(refs))
	for _, ref := range refs {
		session, err := store.LoadSession(ref.SessionID)
		if err != nil {
			return nil, err
		}
		found := false
		for _, proposal := range session.Proposals {
			if proposal.ID != ref.ProposalID {
				continue
			}
			var payload any
			if err := json.Unmarshal([]byte(proposal.PayloadJSON), &payload); err != nil {
				return nil, fmt.Errorf("decode accepted proposal %q payload: %w", ref.ProposalID, err)
			}
			accepted = append(accepted, map[string]any{"session_id": ref.SessionID, "proposal_id": ref.ProposalID, "apply_mode": ref.ApplyMode, "summary": proposal.Summary, "payload": payload})
			found = true
			break
		}
		if !found {
			return nil, fmt.Errorf("accepted proposal %q is missing from Agent Session %q", ref.ProposalID, ref.SessionID)
		}
	}
	return accepted, nil
}

func planWorkshopCanonicalPlanSnapshot(ctx context.Context, plans planclient.PlanReader, planID string) (any, error) {
	if strings.TrimSpace(planID) == "" {
		return nil, nil
	}
	plan, err := plans.GetPlan(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("load canonical plan review snapshot: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("canonical plan review snapshot is missing")
	}
	raw, err := protojson.Marshal(plan)
	if err != nil {
		return nil, fmt.Errorf("encode canonical plan review snapshot: %w", err)
	}
	var snapshot any
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, fmt.Errorf("decode canonical plan review snapshot: %w", err)
	}
	return snapshot, nil
}

func planWorkshopAgentSessionID(workshopID, executionID string) string {
	sum := sha256.Sum256([]byte(workshopID + "\x00" + executionID))
	return "sess_pw_" + hex.EncodeToString(sum[:8])
}

func planWorkshopProposalID(executionID string, index int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d", executionID, index)))
	return "prop_pw_" + hex.EncodeToString(sum[:8])
}

func planWorkshopProposalTarget(subject planworkshop.Subject) agentsessions.ProposalTarget {
	return agentsessions.ProposalTarget{Type: agentsessions.ContextBacklogItem, Ref: subject.Ref, Name: subject.Ref}
}

func currentPlanHash(ctx context.Context, plans planclient.PlanReader, id string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", nil
	}
	if plans == nil {
		return "", fmt.Errorf("plan-manager client is not configured")
	}
	plan, err := plans.GetPlan(ctx, id)
	if err != nil {
		return "", fmt.Errorf("load canonical plan: %w", err)
	}
	if plan == nil || strings.TrimSpace(plan.GetContentHash()) == "" {
		return "", fmt.Errorf("canonical plan has no content hash")
	}
	return strings.TrimSpace(plan.GetContentHash()), nil
}

func workshopSubjectVersion(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func planID(ref *backlog.PlanRef) string {
	if ref == nil {
		return ""
	}
	return ref.PlanID
}
