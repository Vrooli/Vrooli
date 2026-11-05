// Investigation-approval decision: once an investigation's findings are
// recorded, its workflow waits for the operator to approve the recommendations.
// This raises that decision in the operator's inbox, as run review does, so the
// operator need not find the run first. The answer resolves through the same
// approval signal the Agent Manager UI sends (CreateInvestigationApplyRun).
package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

// InvestigationDecisionEventType is the decision an investigation raises when
// it waits for approval. notification-hub subscribes to **.decision_requested.v1.
const InvestigationDecisionEventType = "agent_manager.investigation.decision_requested.v1"

// InvestigationDecisionWithdrawalEventType closes the inbox ask once the
// investigation is decided in Agent Manager; it is not an operator answer.
const InvestigationDecisionWithdrawalEventType = "agent_manager.investigation.decision_withdrawn.v1"

const (
	investigationDecisionRunKey  = "investigation_run_id"
	investigationApplyAllAnswer  = "apply"
	investigationRejectAnswer    = "reject"
	investigationDecisionListMax = 10
)

// investigationDecisionOutput is the part of the investigate node's validated
// result the decision shows.
type investigationDecisionOutput struct {
	Summary    string `json:"summary"`
	Categories []struct {
		Name            string `json:"name"`
		Recommendations []struct {
			Text     string `json:"text"`
			Severity string `json:"severity"`
		} `json:"recommendations"`
	} `json:"categories"`
}

func (out investigationDecisionOutput) recommendations() []string {
	var texts []string
	for _, category := range out.Categories {
		for _, rec := range category.Recommendations {
			if text := strings.TrimSpace(rec.Text); text != "" {
				texts = append(texts, text)
			}
		}
	}
	return texts
}

// investigationAwaitingApproval returns the investigate node's run and result
// when the execution is a live investigation whose findings are ready.
func (o *Orchestrator) investigationAwaitingApproval(ctx context.Context, execution *domain.WorkflowExecution) (*domain.Run, investigationDecisionOutput, bool) {
	var out investigationDecisionOutput
	if execution == nil || execution.Status.Terminal() || execution.Owner != investigationWorkflowOwner || execution.WorkflowKey != investigationWorkflowKey || o.workflowExecutions == nil {
		return nil, out, false
	}
	run, err := o.workflowNodeRun(ctx, execution.ID, investigationInvestigateNodeID)
	if err != nil || run == nil || run.Result == nil || run.Result.Structured == nil || run.Result.Structured.Status != domain.StructuredResultSuccess {
		return nil, out, false
	}
	if allowed, _ := domain.CanApplyInvestigationRun(run, o.investigationTagAllowlist(ctx)); !allowed {
		return nil, out, false
	}
	if json.Unmarshal(run.Result.Structured.Value, &out) != nil {
		return nil, out, false
	}
	return run, out, true
}

// notifyInvestigationAwaitingApproval publishes the approval decision for an
// investigation whose findings are ready. The event id is deterministic, so
// the repeated nudges of one execution raise one ask. Best-effort: a failure is
// logged and retried by the next nudge.
func (o *Orchestrator) notifyInvestigationAwaitingApproval(ctx context.Context, execution *domain.WorkflowExecution) {
	if o == nil || o.approvalPublisher == nil {
		return
	}
	run, out, ok := o.investigationAwaitingApproval(ctx, execution)
	if !ok {
		return
	}
	event := investigationDecisionEvent(run, out, o.resolveWebConsoleNoticeBase(ctx))
	if err := o.approvalPublisher.PublishDomainEvent(ctx, event); err != nil {
		approvalLog().Warn("investigation approval decision publication failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
}

// investigationDecisionRef is the exact reference of the inbox ask raised for
// an execution awaiting approval, or "" when none was raised.
func (o *Orchestrator) investigationDecisionRef(ctx context.Context, execution *domain.WorkflowExecution) string {
	run, out, ok := o.investigationAwaitingApproval(ctx, execution)
	if !ok {
		return ""
	}
	return operatordecision.EventID(investigationDecisionEvent(run, out, o.resolveWebConsoleNoticeBase(ctx)))
}

// withdrawInvestigationDecision closes the inbox ask after the investigation
// was decided. An ask the operator answered stays answered at the hub, so this
// is safe whichever side decided. Best-effort: a failure leaves an ask whose
// answer is then a no-op.
func (o *Orchestrator) withdrawInvestigationDecision(ctx context.Context, runID uuid.UUID, ref string) {
	if o == nil || o.approvalPublisher == nil || ref == "" {
		return
	}
	event := eventbus.DomainEvent{
		Source:    runApprovalSource,
		EventType: InvestigationDecisionWithdrawalEventType,
		Payload: map[string]any{
			"requester_ref":     ref,
			"source_event_type": InvestigationDecisionEventType,
			"reason":            "decided_in_agent_manager",
			"correlation":       map[string]any{investigationDecisionRunKey: runID.String()},
		},
	}
	if err := o.approvalPublisher.PublishDomainEvent(ctx, event); err != nil {
		approvalLog().Warn("investigation approval withdrawal publication failed", obs.KeyRunID, runID.String(), obs.KeyError, err.Error())
	}
}

func investigationDecisionEvent(run *domain.Run, out investigationDecisionOutput, base string) eventbus.DomainEvent {
	// The investigate run's end time keeps the event id stable across nudges.
	occurred := run.CreatedAt.UTC()
	if run.EndedAt != nil {
		occurred = run.EndedAt.UTC()
	}
	return operatordecision.BuildEventTyped(runApprovalSource, InvestigationDecisionEventType, occurred, investigationDecisionFacts(run, out, base))
}

func investigationDecisionFacts(run *domain.Run, out investigationDecisionOutput, base string) operatordecision.Facts {
	recs := out.recommendations()
	subject := "Investigation " + shortRunID(run.ID.String()) + " is ready"
	question := fmt.Sprintf("Apply all %d recommendation(s) from this investigation?", len(recs))
	if len(recs) == 0 {
		question = "This investigation made no recommendations. Close it without applying anything?"
	}
	big := strings.TrimSpace(out.Summary)
	if big == "" {
		big = "The investigation finished and is waiting for your decision."
	}
	big += "\n\nApplying starts an agent run that makes the recommended changes. To apply only some of them, choose them in Agent Manager instead."
	details := strings.Builder{}
	for i, text := range recs {
		if i == investigationDecisionListMax {
			fmt.Fprintf(&details, "… and %d more\n", len(recs)-i)
			break
		}
		fmt.Fprintf(&details, "%d. %s\n", i+1, truncateRunes(text, 400))
	}
	options := []operatordecision.Option{
		{Key: investigationApplyAllAnswer, Label: fmt.Sprintf("Apply all %d", len(recs))},
		{Key: investigationRejectAnswer, Label: "Reject"},
	}
	recommended := investigationApplyAllAnswer
	if len(recs) == 0 {
		options[0].Label = "Apply nothing"
		recommended = investigationRejectAnswer
	}
	return operatordecision.Facts{
		Subject:          subject,
		Question:         question,
		Options:          options,
		Recommended:      recommended,
		Reversible:       false,
		Urgency:          "normal",
		BigPicture:       truncateRunes(big, operatordecision.MaxBigPictureCodePoints-1),
		TechnicalDetails: truncateRunes(details.String(), operatordecision.MaxTechnicalDetailsCodePoints-1),
		ContextURL:       reviewURLForBase(base, run),
		Correlation:      map[string]string{investigationDecisionRunKey: run.ID.String()},
	}
}

// InvestigationDecisionEffect applies the operator's inbox answer to the
// waiting investigation.
func (o *Orchestrator) InvestigationDecisionEffect() operatordecision.DomainEffect {
	return investigationDecisionEffect{o: o}
}

type investigationDecisionEffect struct{ o *Orchestrator }

func (investigationDecisionEffect) Handles(sourceEventType string) bool {
	return sourceEventType == InvestigationDecisionEventType
}

// Apply signals the investigation with the operator's answer. Only an explicit,
// attributed answer acts; an expiry or default leaves the investigation waiting
// for a decision in Agent Manager. An investigation already decided there is a
// benign no-op, so a redelivery never applies twice.
func (e investigationDecisionEffect) Apply(ctx context.Context, res operatordecision.DecisionResolution) (bool, error) {
	runID, err := uuid.Parse(strings.TrimSpace(res.Correlation[investigationDecisionRunKey]))
	if err != nil || e.o == nil {
		return true, nil
	}
	if res.State != operatordecision.StateAnswered || strings.TrimSpace(res.AnsweredBy) == "" {
		return true, nil
	}
	req := CreateInvestigationApplyRequest{InvestigationRunID: runID, CustomContext: res.Note}
	switch strings.TrimSpace(res.Answer) {
	case investigationApplyAllAnswer:
		executionID, err := e.o.workflowExecutions.ExecutionIDForRun(ctx, runID)
		if err != nil {
			return false, err
		}
		execution, err := e.o.workflowExecutions.Get(ctx, executionID)
		if err != nil {
			return false, err
		}
		_, out, ok := e.o.investigationAwaitingApproval(ctx, execution)
		if !ok {
			return true, nil
		}
		req.Decision, req.Selected = "completed", out.recommendations()
		if len(req.Selected) == 0 {
			req.Decision = "abstained"
		}
	case investigationRejectAnswer:
		req.Decision = "rejected"
	default:
		return true, nil
	}
	if _, err := e.o.CreateInvestigationApplyRun(ctx, req); err != nil {
		var validation *domain.ValidationError
		if errors.As(err, &validation) {
			approvalLog().Info("investigation approval answer not applied", obs.KeyRunID, runID.String(), obs.KeyError, err.Error())
			return true, nil
		}
		return false, err
	}
	return true, nil
}
