package orchestration

import (
	"context"
	"fmt"
	"os"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"agent-manager/internal/workflowcatalog"
)

func (o *Orchestrator) ValidateWorkflow(ctx context.Context, data []byte) (*WorkflowValidationResult, error) {
	parsed, err := workflowcatalog.Parse(data, nil)
	if err != nil {
		return &WorkflowValidationResult{Diagnostics: []domain.WorkflowDiagnostic{{Code: "decode", Message: err.Error()}}}, nil
	}
	result := &WorkflowValidationResult{Valid: !domain.HasBlockingDiagnostic(parsed.Diagnostics), Digest: parsed.Digest, Diagnostics: parsed.Diagnostics}
	if result.Valid {
		if diagnostics := o.validateWorkflowTargets(ctx, &parsed.Definition, nil); len(diagnostics) != 0 {
			result.Valid = false
			result.Diagnostics = diagnostics
			result.Digest = ""
		} else {
			result.Definition = &parsed.Definition
		}
	}
	return result, nil
}

// ReconcileScenarioWorkflows is a thin projection over the unified declaration
// reconcile that returns only the workflow-kind results. The RPC name keeps
// working for existing Go consumer clients while reading exclusively from the
// new .vrooli/agent-manager/ declaration block.
func (o *Orchestrator) ReconcileScenarioWorkflows(ctx context.Context, req ReconcileScenarioWorkflowsRequest) (*ReconcileScenarioWorkflowsResult, error) {
	res, err := o.ReconcileScenarioDeclarations(ctx, ReconcileScenarioDeclarationsRequest(req))
	if err != nil {
		return nil, err
	}
	result := &ReconcileScenarioWorkflowsResult{Scenario: res.Scenario, Results: res.WorkflowResults, DryRun: res.DryRun, ValidateOnly: res.ValidateOnly}
	result.recountWorkflows()
	return result, nil
}

func (o *Orchestrator) validateWorkflowTargets(ctx context.Context, d *domain.WorkflowDefinition, siblingKeys map[string]bool) []domain.WorkflowDiagnostic {
	var diagnostics []domain.WorkflowDiagnostic
	for i, node := range d.Nodes {
		path := fmt.Sprintf("nodes[%d]", i)
		if node.Run != nil && node.Run.ProfileKey != "" {
			profile, err := o.profiles.GetByKey(ctx, node.Run.ProfileKey)
			if err != nil || profile == nil {
				diagnostics = append(diagnostics, domain.WorkflowDiagnostic{Code: "profile_missing", Path: path + ".run.profileKey", Message: "profile does not resolve"})
			}
		}
		if node.Run != nil && node.Run.RoleRef != "" {
			if o.rolePolicy == nil {
				diagnostics = append(diagnostics, domain.WorkflowDiagnostic{Code: "role_catalog_unavailable", Path: path + ".run.roleRef", Message: "portable role catalog is unavailable"})
			} else if active := o.rolePolicy.Active(); active == nil {
				diagnostics = append(diagnostics, domain.WorkflowDiagnostic{Code: "role_catalog_unavailable", Path: path + ".run.roleRef", Message: "portable role catalog is unavailable"})
			} else if _, ok := active.Catalog().Roles[node.Run.RoleRef]; !ok {
				diagnostics = append(diagnostics, domain.WorkflowDiagnostic{Code: "role_missing", Path: path + ".run.roleRef", Message: "portable role does not resolve"})
			}
		}
		if node.Child != nil && !siblingKeys[node.Child.WorkflowKey] {
			if o.workflows == nil {
				diagnostics = append(diagnostics, domain.WorkflowDiagnostic{Code: "child_catalog_unavailable", Path: path + ".childWorkflow.workflowKey", Message: "workflow catalog is unavailable"})
				continue
			}
			active, err := o.workflows.GetActive(ctx, d.Owner, node.Child.WorkflowKey)
			if err != nil || active == nil {
				diagnostics = append(diagnostics, domain.WorkflowDiagnostic{Code: "child_missing", Path: path + ".childWorkflow.workflowKey", Message: "child workflow does not resolve"})
			}
		}
	}
	return diagnostics
}

func (o *Orchestrator) ListWorkflowRevisions(ctx context.Context, owner, key string, opts ListOptions) ([]*domain.WorkflowRevision, error) {
	if o.workflows == nil {
		return nil, domain.NewConfigInvalidError("workflowRepository", "workflow catalog persistence is not configured", nil)
	}
	return o.workflows.List(ctx, strings.TrimSpace(owner), strings.TrimSpace(key), repository.ListFilter{Limit: opts.Limit, Offset: opts.Offset})
}

func (o *Orchestrator) GetWorkflowRevision(ctx context.Context, owner, key, digest string) (*domain.WorkflowRevision, error) {
	if o.workflows == nil {
		return nil, domain.NewConfigInvalidError("workflowRepository", "workflow catalog persistence is not configured", nil)
	}
	if strings.TrimSpace(digest) != "" {
		return o.workflows.GetByDigest(ctx, strings.TrimSpace(digest))
	}
	return o.workflows.GetActive(ctx, strings.TrimSpace(owner), strings.TrimSpace(key))
}

func readWorkflowSource(path string) ([]byte, os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("stat workflow source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("workflow source must be a regular file")
	}
	if info.Size() <= 0 || info.Size() > workflowcatalog.MaxDefinitionBytes {
		return nil, nil, fmt.Errorf("workflow source exceeds bounded size")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read workflow source: %w", err)
	}
	return data, info, nil
}

func (r *ReconcileScenarioWorkflowsResult) recountWorkflows() {
	r.Created, r.Activated, r.Unchanged, r.Skipped, r.Failed = 0, 0, 0, 0, 0
	for _, item := range r.Results {
		switch item.Status {
		case WorkflowReconcileCreated:
			r.Created++
		case WorkflowReconcileActivated:
			r.Activated++
		case WorkflowReconcileUnchanged:
			r.Unchanged++
		case WorkflowReconcileSkipped:
			r.Skipped++
		case WorkflowReconcileFailedValidation:
			r.Failed++
		}
	}
}
