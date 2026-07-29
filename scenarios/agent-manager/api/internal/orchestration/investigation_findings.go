// Investigation-findings responsibility: derive idempotent recurrence records
// from a terminal investigation workflow's validated structured result.
package orchestration

import (
	"context"
	"encoding/json"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/findings"
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
		for _, category := range output.Categories {
			for _, recommendation := range category.Recommendations {
				if strings.TrimSpace(recommendation.Text) == "" {
					continue
				}
				severity := strings.TrimSpace(recommendation.Severity)
				if severity == "" {
					severity = "Gap"
				}
				for _, sourceRunID := range investigationRun.SourceRunIDs {
					_ = o.findings.Create(ctx, &findings.Finding{RunID: sourceRunID, InvestigationRunID: investigationRun.ID, Category: category.Name, Severity: severity, Recommendation: recommendation.Text, Evidence: recommendation.Evidence, TargetPath: recommendation.TargetPath})
				}
			}
		}
		return
	}
}
