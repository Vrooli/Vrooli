package rehearsals

import (
	"context"
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
	if hosts := simulator.Hosts(); len(hosts) != 1 || hosts[0] != "supabase.synthetic.invalid" {
		t.Fatalf("simulator must expose only the declared synthetic host, got %v", hosts)
	}
	triggerIDs := map[string]string{}
	for _, node := range workflow.Nodes {
		if node.Trigger == TriggerWebhook {
			triggerIDs[node.Name] = node.ID
		}
	}

	serviceNodeNames := map[string]string{
		"Webhook":  "Get a row",
		"Webhook1": "Create a row",
		"Webhook3": "Update a row",
		"Webhook4": "Delete a row",
	}
	for triggerName, expectedNode := range serviceNodeNames {
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
			if !traceHasOperation(run.Trace, expectedNode) {
				t.Fatalf("selected branch did not execute %q: %#v", expectedNode, run.Trace)
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
