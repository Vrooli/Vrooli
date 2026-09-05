package orchestration

import (
	"context"
	"strings"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestReconcileScenarioWorkflowsPreservesDeclarationValidation(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	result, err := o.ReconcileScenarioWorkflows(context.Background(), ReconcileScenarioWorkflowsRequest{})
	if err == nil {
		t.Fatal("expected missing scenario validation error")
	}
	if result != nil {
		t.Fatalf("result = %+v, want nil on validation error", result)
	}
}

func TestValidateWorkflowReportsParseAndTargetDiagnostics(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()

	invalid, err := o.ValidateWorkflow(ctx, []byte(`{`))
	if err != nil || invalid.Valid || len(invalid.Diagnostics) != 1 || invalid.Diagnostics[0].Code != "decode" {
		t.Fatalf("invalid workflow=%+v err=%v", invalid, err)
	}

	profile, err := parseSourceProfile([]byte(fixtureProfile))
	if err != nil {
		t.Fatal(err)
	}
	profile.ID = uuid.New()
	if err := o.profiles.Create(ctx, profile); err != nil {
		t.Fatal(err)
	}
	valid, err := o.ValidateWorkflow(ctx, []byte(fixtureWorkflow))
	if err != nil || !valid.Valid || valid.Digest == "" || valid.Definition == nil {
		t.Fatalf("valid workflow=%+v err=%v", valid, err)
	}

	missingProfile := strings.Replace(fixtureWorkflow, "fixture-scn/default", "fixture-scn/missing", 1)
	missing, err := o.ValidateWorkflow(ctx, []byte(missingProfile))
	if err != nil || missing.Valid || len(missing.Diagnostics) != 1 || missing.Diagnostics[0].Code != "profile_missing" {
		t.Fatalf("missing profile workflow=%+v err=%v", missing, err)
	}
}

func TestReconcileScenarioWorkflowsProjectsSelfDeclarationFailure(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	result, err := o.ReconcileScenarioWorkflows(context.Background(), ReconcileScenarioWorkflowsRequest{Scenario: agentManagerSelfScenario, DryRun: true})
	if err != nil {
		t.Fatalf("reconcile self-declared workflows: %v", err)
	}
	if result.Scenario != agentManagerSelfScenario || !result.DryRun || result.Created != 0 || result.Failed != 1 {
		t.Fatalf("reconciliation result=%+v", result)
	}
	if len(result.Results) != 1 || result.Results[0].Status != WorkflowReconcileFailedValidation || !strings.Contains(result.Results[0].Message, "prompt-manager source client") {
		t.Fatalf("workflow result=%+v", result.Results)
	}
}

func TestValidateWorkflowTargetsReportsEveryUnresolvableDependency(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	diagnostics := o.validateWorkflowTargets(context.Background(), &domain.WorkflowDefinition{
		Owner: "owner",
		Nodes: []domain.WorkflowNode{
			{ID: "profile", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{ProfileKey: "owner/missing"}},
			{ID: "role", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.missing"}},
			{ID: "child", Kind: domain.WorkflowNodeChild, Child: &domain.WorkflowChildNode{WorkflowKey: "owner/missing-child"}},
		},
	}, map[string]bool{})
	if len(diagnostics) != 3 {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
	got := map[string]string{}
	for _, diagnostic := range diagnostics {
		got[diagnostic.Path] = diagnostic.Code
	}
	if got["nodes[0].run.profileKey"] != "profile_missing" || got["nodes[1].run.roleRef"] != "role_catalog_unavailable" || got["nodes[2].childWorkflow.workflowKey"] != "child_missing" {
		t.Fatalf("diagnostic codes=%v", got)
	}

	// A sibling declared in the same batch is resolvable without requiring it
	// to be active before the batch activates.
	if diagnostics := o.validateWorkflowTargets(context.Background(), &domain.WorkflowDefinition{
		Owner: "owner",
		Nodes: []domain.WorkflowNode{{ID: "child", Kind: domain.WorkflowNodeChild, Child: &domain.WorkflowChildNode{WorkflowKey: "owner/sibling"}}},
	}, map[string]bool{"owner/sibling": true}); len(diagnostics) != 0 {
		t.Fatalf("sibling diagnostics=%+v", diagnostics)
	}
}

func TestReconcileScenarioWorkflowsResultRecountsEveryStatus(t *testing.T) {
	result := &ReconcileScenarioWorkflowsResult{Results: []WorkflowReconcileResult{
		{Status: WorkflowReconcileCreated},
		{Status: WorkflowReconcileActivated},
		{Status: WorkflowReconcileUnchanged},
		{Status: WorkflowReconcileSkipped},
		{Status: WorkflowReconcileFailedValidation},
		{Status: WorkflowReconcileCreated},
	}}

	result.recountWorkflows()
	if result.Created != 2 || result.Activated != 1 || result.Unchanged != 1 || result.Skipped != 1 || result.Failed != 1 {
		t.Fatalf("unexpected reconciliation counts: %+v", result)
	}
}

func TestListWorkflowRevisionsReturnsReconciledCatalogEntry(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()
	scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/default.json", ".vrooli/agent-manager/round.json"),
		".vrooli/agent-manager/default.json": fixtureProfile,
		".vrooli/agent-manager/round.json":   fixtureWorkflow,
	})
	if _, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false); err != nil {
		t.Fatalf("reconcile declarations: %v", err)
	}

	items, err := o.ListWorkflowRevisions(ctx, " fixture-scn ", " fixture-scn/round ", ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("list workflow revisions: %v", err)
	}
	if len(items) != 1 || items[0].Key != "fixture-scn/round" || !items[0].Active {
		t.Fatalf("workflow revisions = %+v, want one active reconciled revision", items)
	}

	got, err := o.GetWorkflowRevision(ctx, "fixture-scn", "fixture-scn/round", items[0].Digest)
	if err != nil {
		t.Fatalf("get workflow revision by digest: %v", err)
	}
	if got == nil || got.ID != items[0].ID {
		t.Fatalf("revision = %+v, want %s", got, items[0].ID)
	}
}
