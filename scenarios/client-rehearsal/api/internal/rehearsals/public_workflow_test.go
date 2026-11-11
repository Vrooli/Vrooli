package rehearsals

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPublicSupabaseWorkflowRunsSelectedWebhookBranchesInSyntheticSimulator(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/8258.json")
	profileJSON := readFixture(t, "public-workflow-profiles.json")
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize sanitized public workflow: %v", err)
	}
	simulator, err := NewDescriptionSimulator(profileJSON, false)
	if err != nil {
		t.Fatalf("load synthetic Supabase profile: %v", err)
	}
	if hosts := simulator.Hosts(); len(hosts) != 4 || hosts[0] != "api.synthetic.invalid" || hosts[1] != "crawl.synthetic.invalid" || hosts[2] != "postgres.synthetic.invalid" || hosts[3] != "supabase.synthetic.invalid" {
		t.Fatalf("simulator must expose only the declared synthetic hosts, got %v", hosts)
	}
	triggerIDs := map[string]string{}
	for _, node := range workflow.Nodes {
		if node.Trigger == TriggerWebhook {
			triggerIDs[node.Name] = node.ID
		}
	}

	serviceOperations := map[string]string{
		"Webhook":  "supabase:row.get",
		"Webhook1": "supabase:row.create",
		"Webhook3": "supabase:row.update",
		"Webhook4": "supabase:row.delete",
	}
	for triggerName, expectedOperation := range serviceOperations {
		t.Run(triggerName, func(t *testing.T) {
			triggerID := triggerIDs[triggerName]
			if triggerID == "" {
				t.Fatalf("fixture is missing trigger %q", triggerName)
			}
			repo := newMemoryRepository()
			service := NewService(repo, SyntheticEngine{}, func() time.Time { return time.Unix(42, 0).UTC() })
			run, runErr := service.CreateAndRun(context.Background(), CreateInput{
				Title:               "Sanitized public Supabase fixture " + triggerName,
				WorkflowJSON:        workflowJSON,
				TriggerNodeID:       triggerID,
				SimulatorProfile:    "public-workflow-supabase-catalog",
				SimulatorSourceJSON: profileJSON,
			})
			if runErr != nil {
				t.Fatalf("run sanitized public workflow: %v", runErr)
			}
			if run.State != StateNeedsReview || !strings.Contains(run.HandoffMarkdown, "Human review is required") {
				t.Fatalf("expected a completed synthetic handoff awaiting review, got %#v", run)
			}
			if run.CredentialReplacements == 0 {
				t.Fatal("expected credential references to be replaced before simulation")
			}
			if !traceHasOperation(run.Trace, expectedOperation) {
				t.Fatalf("selected branch did not execute %q: %#v", expectedOperation, run.Trace)
			}
			if !traceHasStatus(run.Trace, "succeeded") {
				t.Fatalf("expected successful synthetic service outcome: %#v", run.Trace)
			}
			for _, private := range []string{"synthetic-contacts", "supabaseApi", "phone-shaped"} {
				if strings.Contains(run.HandoffMarkdown, private) {
					t.Fatalf("handoff leaked fixture detail %q", private)
				}
			}
		})
	}
}

func TestPublicPostgresWorkflowRunsSyntheticReadOnlyListBranch(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/16081.json")
	profileJSON := readFixture(t, "public-workflow-profiles.json")
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize sanitized public workflow: %v", err)
	}
	triggerID := ""
	for _, node := range workflow.Nodes {
		if node.Name == "Webhook" && node.Trigger == TriggerWebhook {
			triggerID = node.ID
		}
	}
	if triggerID == "" {
		t.Fatal("fixture is missing its webhook entry")
	}
	encoded, err := json.Marshal(workflow)
	if err != nil {
		t.Fatal(err)
	}
	for _, sourceSQL := range []string{"SELECT * FROM jobs", "INSERT INTO jobs", "UPDATE jobs", "DELETE FROM jobs"} {
		if strings.Contains(strings.ToLower(string(encoded)), strings.ToLower(sourceSQL)) {
			t.Fatalf("normalized workflow retained source SQL %q", sourceSQL)
		}
	}

	repo := newMemoryRepository()
	service := NewService(repo, SyntheticEngine{}, func() time.Time { return time.Unix(43, 0).UTC() })
	run, runErr := service.CreateAndRun(context.Background(), CreateInput{
		Title:               "Sanitized public Postgres webhook list",
		WorkflowJSON:        workflowJSON,
		TriggerNodeID:       triggerID,
		SimulatorProfile:    "public-workflow-postgres-catalog",
		SimulatorSourceJSON: profileJSON,
	})
	if runErr != nil {
		t.Fatalf("run sanitized public workflow: %v", runErr)
	}
	if run.State != StateNeedsReview || !traceHasOperation(run.Trace, "executeQuery.select") {
		t.Fatalf("expected only the synthetic list query branch to run: state=%s trace=%#v", run.State, run.Trace)
	}
	if !traceHasNode(run.Trace, "Get All Jobs") || run.CredentialReplacements == 0 {
		t.Fatalf("expected the selected list branch and replaced credentials: %#v", run)
	}
	for _, mutation := range []string{"executeQuery.insert", "executeQuery.update", "executeQuery.delete"} {
		if traceHasOperation(run.Trace, mutation) {
			t.Fatalf("read-only webhook list rehearsal executed mutation %q: %#v", mutation, run.Trace)
		}
	}
	var listResponse string
	for _, event := range run.Trace {
		if strings.Contains(event.Operation, "executeQuery.select") {
			listResponse = event.ResponseSummary
			break
		}
	}
	if !strings.Contains(listResponse, `"rows"`) || !strings.Contains(listResponse, `"synthetic-id"`) {
		t.Fatalf("selected branch did not produce the known synthetic list response: %q", listResponse)
	}
	if strings.Contains(run.HandoffMarkdown, "SELECT") || !strings.Contains(run.HandoffMarkdown, "Human review is required") {
		t.Fatalf("handoff leaked SQL or omitted review requirement: %s", run.HandoffMarkdown)
	}

	withFault, faultErr := service.CreateAndRun(context.Background(), CreateInput{
		Title:               "Sanitized public Postgres webhook list with transient failure",
		WorkflowJSON:        workflowJSON,
		TriggerNodeID:       triggerID,
		SimulatorProfile:    "public-workflow-postgres-catalog",
		SimulatorSourceJSON: profileJSON,
		InjectSeededFault:   true,
	})
	if faultErr != nil {
		t.Fatalf("recover sanitized public workflow from injected service fault: %v", faultErr)
	}
	if withFault.State != StateNeedsReview || withFault.FaultSummary != "Seeded transient synthetic service HTTP 503." || !strings.Contains(withFault.RepairSummary, "retried the failed synthetic service operation once") || !traceHasStatus(withFault.Trace, "failed") || !traceHasStatus(withFault.Trace, "succeeded") {
		t.Fatalf("expected a recovered faulted public workflow and review handoff: %#v", withFault)
	}
}

func TestPublicInvoiceWorkflowUsesGenericHTTPAndSplitOutSimulation(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/22567.json")
	profileJSON := readFixture(t, "public-workflow-profiles.json")
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize sanitized public workflow: %v", err)
	}
	if len(workflow.Nodes) != 5 || len(workflow.EntryNodeIDs) != 1 || workflow.Nodes[1].SetValues["username"] != "synthetic-user" {
		t.Fatalf("expected static synthetic Set values and one manual entry: %#v", workflow)
	}
	encoded, err := json.Marshal(workflow)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "c09StarterDataUrl") || workflow.Nodes[1].SetValues["participantKey"] != "synthetic-key" || !strings.Contains(string(encoded), "api.synthetic.invalid") {
		t.Fatalf("normalized graph retained source URL program or lost the synthetic stand-in host/key: %s", encoded)
	}

	repo := newMemoryRepository()
	service := NewService(repo, SyntheticEngine{}, func() time.Time { return time.Unix(44, 0).UTC() })
	run, runErr := service.CreateAndRun(context.Background(), CreateInput{
		Title:               "Sanitized public invoice and payment export",
		WorkflowJSON:        workflowJSON,
		TriggerNodeID:       workflow.EntryNodeIDs[0],
		SimulatorProfile:    "public-invoice-and-payment-catalog",
		SimulatorSourceJSON: profileJSON,
	})
	if runErr != nil {
		t.Fatalf("run sanitized public workflow: %v", runErr)
	}
	if run.State != StateNeedsReview || !traceHasOperation(run.Trace, "GET /v1/invoices-and-payments") {
		t.Fatalf("expected synthetic API request and human-review handoff: state=%s trace=%#v", run.State, run.Trace)
	}
	for node, want := range map[string]string{"Split invoices": "split into 2 synthetic items", "Split payments": "split into 1 synthetic items"} {
		found := false
		for _, event := range run.Trace {
			if event.RequestSummary == node {
				found = strings.Contains(event.ResponseSummary, want) && event.Status == "succeeded"
			}
		}
		if !found {
			t.Fatalf("expected %q to emit %q: %#v", node, want, run.Trace)
		}
	}
}

func TestPublicSubworkflowCrawlerUsesSyntheticDescriptionAndSafeSetProjection(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/2713.json")
	profileJSON := readFixture(t, "public-workflow-profiles.json")
	workflow, _, replacements, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize sanitized public subworkflow export: %v", err)
	}
	child := workflow.Subworkflows["2713-export"]
	if replacements != 1 || len(workflow.EntryNodeIDs) != 1 || child == nil || child.SubworkflowEntryID == "" {
		t.Fatalf("expected one sanitized credential and an imported subworkflow entry: replacements=%d workflow=%#v", replacements, workflow)
	}
	if child.Nodes[2].SetProjections["response"] != "data.markdown" || child.Nodes[1].Request.Host != "crawl.synthetic.invalid" {
		t.Fatalf("expected safe output projection and synthetic endpoint: %#v", child.Nodes)
	}

	simulator, err := NewDescriptionSimulator(profileJSON, false)
	if err != nil {
		t.Fatal(err)
	}
	var trace []TraceEvent
	output, err := executeGraph(context.Background(), workflow, workflow.EntryNodeIDs[0], map[string]any{"query": map[string]any{"url": "https://site.synthetic.invalid/page"}}, simulator, false, time.Unix(45, 0), &trace, map[string]bool{}, 0)
	if err != nil {
		t.Fatalf("execute real sanitized export through generic simulator: %v", err)
	}
	if output["response"] != "synthetic-value" || !traceHasOperation(trace, "POST /v1/scrape") {
		t.Fatalf("expected synthetic crawl output and generic HTTP route, output=%#v trace=%#v", output, trace)
	}

	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Unix(46, 0).UTC() })
	run, err := service.CreateAndRun(context.Background(), CreateInput{
		Title:               "Sanitized public subworkflow crawler export",
		WorkflowJSON:        workflowJSON,
		TriggerNodeID:       workflow.EntryNodeIDs[0],
		SimulatorProfile:    "public-workflow-catalog",
		SimulatorSourceJSON: profileJSON,
	})
	if err != nil {
		t.Fatalf("create public workflow review handoff: %v", err)
	}
	if run.State != StateNeedsReview || !traceHasOperation(run.Trace, "POST /v1/scrape") {
		t.Fatalf("expected completed synthetic run with human review required: %#v", run)
	}
}

func TestPublicSubworkflowOutputDefectCanBeRepairedAgainstFailedRun(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/2713.json")
	profileJSON := readFixture(t, "public-workflow-profiles.json")
	const correctProjection = `={{ $json.data.markdown }}`
	const defectiveProjection = `={{ $json.data.html }}`
	if !strings.Contains(workflowJSON, correctProjection) {
		t.Fatal("expected fixture projection to be present")
	}
	defective := strings.Replace(workflowJSON, correctProjection, defectiveProjection, 1)
	repo := newMemoryRepository()
	service := NewService(repo, SyntheticEngine{}, func() time.Time { return time.Unix(47, 0).UTC() })
	checks := []string{"response"}
	failed, err := service.CreateAndRun(context.Background(), CreateInput{
		Title:               "Seeded public crawler output defect",
		WorkflowJSON:        defective,
		SimulatorProfile:    "public-workflow-catalog",
		SimulatorSourceJSON: profileJSON,
		RequiredOutputPaths: checks,
	})
	if err != nil {
		t.Fatalf("run seeded output defect: %v", err)
	}
	if failed.State != StateFailed || failed.FaultSummary != "Required output paths missing: response" || !traceHasStatus(failed.Trace, "failed") {
		t.Fatalf("expected detectable missing output and failed trace assertion: %#v", failed)
	}
	if strings.Contains(failed.HandoffMarkdown, "synthetic-value") {
		t.Fatal("handoff must not contain simulated response values")
	}

	repaired, err := service.CreateAndRun(context.Background(), CreateInput{
		Title:               "Repaired public crawler output mapping",
		WorkflowJSON:        workflowJSON,
		SimulatorProfile:    "public-workflow-catalog",
		SimulatorSourceJSON: profileJSON,
		RequiredOutputPaths: checks,
		RepairOfID:          failed.ID,
	})
	if err != nil {
		t.Fatalf("run corrected revision against failed evidence: %v", err)
	}
	if repaired.State != StateNeedsReview || repaired.SourceSHA256 == failed.SourceSHA256 || !traceHasOperation(repaired.Trace, "repair lineage") || !strings.Contains(repaired.RepairSummary, failed.ID) {
		t.Fatalf("expected changed passing revision linked to failed run: failed=%#v repaired=%#v", failed, repaired)
	}
	stored, err := service.Get(context.Background(), repaired.ID)
	if err != nil || stored.RepairSummary != repaired.RepairSummary || !traceHasOperation(stored.Trace, "repair lineage") {
		t.Fatalf("expected repository readback to retain repair lineage: stored=%#v err=%v", stored, err)
	}
}
