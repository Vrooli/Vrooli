// Investigation-findings responsibility: derive idempotent recurrence records
// from a terminal investigation workflow's validated structured result.
package orchestration

import (
	"context"
	"encoding/json"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/findings"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/promptmanager"

	"github.com/google/uuid"
)

type investigationStructuredOutput struct {
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

// persistInvestigationFindings derives durable recurrence rows only from a
// successfully validated workflow result. SQLite uniqueness makes recovery
// and repeated terminal notifications idempotent.
func (o *Orchestrator) persistInvestigationFindings(ctx context.Context, execution *domain.WorkflowExecution) {
	if o.findings == nil || execution == nil || execution.Owner != investigationWorkflowOwner || execution.WorkflowKey != investigationWorkflowKey || o.workflowExecutions == nil {
		return
	}
	attempts, err := o.workflowExecutions.ListAttempts(ctx, execution.ID)
	if err != nil {
		return
	}
	for _, attempt := range attempts {
		if attempt.NodeID != investigationInvestigateNodeID || attempt.RunID == nil {
			continue
		}
		investigationRun, err := o.GetRun(ctx, *attempt.RunID)
		if err != nil || investigationRun == nil || investigationRun.Result == nil || investigationRun.Result.Structured == nil || investigationRun.Result.Structured.Status != domain.StructuredResultSuccess {
			return
		}
		var output investigationStructuredOutput
		if json.Unmarshal(investigationRun.Result.Structured.Value, &output) != nil {
			return
		}
		sourceRunIDs := investigationRun.SourceRunIDs
		if len(sourceRunIDs) == 0 {
			sourceRunIDs = investigationSourceRunIDs(execution.Input)
		}
		if len(sourceRunIDs) == 0 {
			return
		}
		for _, category := range output.Categories {
			for _, recommendation := range category.Recommendations {
				if strings.TrimSpace(recommendation.Text) == "" {
					continue
				}
				severity := strings.TrimSpace(recommendation.Severity)
				if severity == "" {
					severity = "Gap"
				}
				for _, sourceRunID := range sourceRunIDs {
					_ = o.findings.Create(ctx, &findings.Finding{RunID: sourceRunID, InvestigationRunID: investigationRun.ID, Category: category.Name, Severity: severity, Recommendation: recommendation.Text, Evidence: recommendation.Evidence, TargetPath: recommendation.TargetPath})
				}
			}
		}
		o.routeInvestigationFindings(ctx, investigationRun.ID)
		o.measureAppliedFindings(ctx, execution, investigationRun.ID)
		return
	}
}

// investigationSourceRunIDs reads the immutable workflow input snapshot. A
// workflow run node is a fresh child run and therefore does not inherit the
// request's SourceRunIDs field; the input is the durable linkage between that
// child result and the corpus it investigated.
func investigationSourceRunIDs(input json.RawMessage) []uuid.UUID {
	var payload struct {
		RunIDs []string `json:"runIds"`
	}
	if json.Unmarshal(input, &payload) != nil {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(payload.RunIDs))
	for _, raw := range payload.RunIDs {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err == nil && id != uuid.Nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// routeInvestigationFindings sends each new durable finding through the
// meta-optimization intake. A stored topic is the idempotency marker: replayed
// terminal notifications never create another inbox entry.
func (o *Orchestrator) routeInvestigationFindings(ctx context.Context, investigationRunID uuid.UUID) {
	publisher, ok := o.promptClient.(promptmanager.FrictionIntakeClient)
	if !ok || publisher == nil || o.findings == nil {
		return
	}
	items, err := o.findings.List(ctx, findings.Filter{Limit: 1000})
	if err != nil {
		return
	}
	for _, item := range items {
		if item.InvestigationRunID != investigationRunID || strings.TrimSpace(item.FrictionTopic) != "" {
			continue
		}
		topic, err := publisher.PublishFriction(ctx, promptmanager.FrictionReport{InvestigationRunID: investigationRunID.String(), Fingerprint: item.Fingerprint, Category: item.Category, Severity: item.Severity, Occurrences: item.Occurrences, Recommendation: item.Recommendation, Evidence: item.Evidence, TargetPath: item.TargetPath})
		if err != nil {
			continue
		}
		_ = o.findings.SetEffectiveness(ctx, item.ID, item.BeforeValue, item.AfterValue, item.Effectiveness, topic)
	}
}

// measureAppliedFindings records the after value only once the declared apply
// node completes. Failed, rejected, and merely launched applies keep their
// durable finding explicitly not-yet-measurable.
func (o *Orchestrator) measureAppliedFindings(ctx context.Context, execution *domain.WorkflowExecution, investigationRunID uuid.UUID) {
	if o.findings == nil || o.invocationReadModel == nil || execution == nil {
		return
	}
	attempts, err := o.workflowExecutions.ListAttempts(ctx, execution.ID)
	if err != nil {
		return
	}
	applyComplete := false
	for _, attempt := range attempts {
		if attempt.NodeID == investigationApplyNodeID && attempt.Status == domain.WorkflowAttemptCompleted && attempt.RunID != nil {
			applyComplete = true
			break
		}
	}
	if !applyComplete {
		return
	}
	metrics, err := o.invocationReadModel.FindingMetrics(ctx, invocationreadmodel.Filter{})
	if err != nil || metrics.TotalFindings == 0 {
		return
	}
	after := metrics.RecurrenceRate
	items, err := o.findings.List(ctx, findings.Filter{Limit: 1000, Decision: "completed"})
	if err != nil {
		return
	}
	for _, item := range items {
		if item.InvestigationRunID != investigationRunID || item.BeforeValue == nil || item.AfterValue != nil {
			continue
		}
		if err := o.findings.SetEffectiveness(ctx, item.ID, item.BeforeValue, &after, findings.Effectiveness(item.BeforeValue, &after), item.FrictionTopic); err != nil {
			continue
		}
	}
}
