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

	"agent-manager/internal/decisionpage"
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
			Text       string `json:"text"`
			Severity   string `json:"severity"`
			Evidence   string `json:"evidence"`
			TargetPath string `json:"targetPath"`
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
	looked := operatordecision.Count(len(run.SourceRunIDs), "agent run", "agent runs")
	if len(run.SourceRunIDs) == 0 {
		looked = "earlier agent work"
	}
	changes := operatordecision.Count(len(recs), "change", "changes")
	f := operatordecision.Facts{
		Subject:  fmt.Sprintf("Make the %s an investigation suggests?", changes),
		Question: fmt.Sprintf("An agent looked into %s and suggests %s. Start an agent to make all of them?", looked, changes),
		Options: []operatordecision.Option{
			{Key: investigationApplyAllAnswer, Label: fmt.Sprintf("Start an agent to make all %d", len(recs))},
			{Key: investigationRejectAnswer, Label: "Make none of them"},
		},
		Recommended:          investigationApplyAllAnswer,
		RecommendationReason: "The suggestions come with evidence; to pick only some, choose them in Agent Manager instead.",
		Reversible:           false,
		Urgency:              "normal",
		ContextURL:           reviewURLForBase(base, run),
		Correlation:          map[string]string{investigationDecisionRunKey: run.ID.String()},
	}
	if len(recs) == 0 {
		f.Subject = "Close an investigation that found nothing to change?"
		f.Question = fmt.Sprintf("An agent looked into %s and suggests no changes. Close the investigation?", looked)
		f.Options[0].Label = "Close it, changing nothing"
		f.Options[1].Label = "Reject its findings"
		f.Recommended, f.RecommendationReason = investigationRejectAnswer, ""
	}
	summary := operatordecision.Clip(out.Summary, 1500)
	big := "The investigation finished and is waiting for you."
	if summary != "" {
		big = "What it found: " + summary
	}
	f.BigPicture = big + "\n\nStarting the agent makes all the suggested changes. To apply only some of them, choose them in Agent Manager instead. Making none closes the investigation. If you do not answer, nothing changes."

	var details strings.Builder
	fmt.Fprintf(&details, "Investigation run %s.\n", run.ID)
	for i, text := range recs {
		if i == investigationDecisionListMax {
			fmt.Fprintf(&details, "… and %d more\n", len(recs)-i)
			break
		}
		fmt.Fprintf(&details, "%d. %s\n", i+1, operatordecision.Clip(text, 400))
	}
	f.TechnicalDetails = operatordecision.Clip(details.String(), operatordecision.MaxTechnicalDetailsCodePoints-1)

	page := operatordecision.NewPage(f.Subject).Raw(decisionpage.Style).
		Lead(fmt.Sprintf("An agent looked into %s to find what went wrong and how to prevent it.", looked))
	if summary != "" {
		page.Section("What it found", "").Raw(decisionpage.Quote(summary))
	}
	rows := [][]string{}
	for _, category := range out.Categories {
		for _, rec := range category.Recommendations {
			if strings.TrimSpace(rec.Text) == "" {
				continue
			}
			severity := strings.TrimSpace(rec.Severity)
			if severity == "" {
				severity = "Gap"
			}
			rows = append(rows, []string{operatordecision.Clip(rec.Text, 500), category.Name, severity, operatordecision.Clip(rec.Evidence, 400), rec.TargetPath})
		}
	}
	page.Section("What it suggests", "").Table([]string{"Change", "Area", "How serious", "Evidence", "Where"}, rows)
	effects := map[string]string{
		investigationApplyAllAnswer: "Starts one agent run that makes every change above.",
		investigationRejectAnswer:   "Closes the investigation. Nothing is changed.",
	}
	if len(recs) == 0 {
		effects[investigationApplyAllAnswer] = "Closes the investigation as finished, with nothing to change."
		effects[investigationRejectAnswer] = "Closes it and records that you do not accept its findings."
	}
	page.Options(f, effects)
	f.Explainer = page.IfNoAnswer("Nothing changes. The investigation waits in Agent Manager.").HTML()
	return f
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
