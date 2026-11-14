package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestClientShapedGmailSheetsCRMLeadHandoffRunsInOneSitting(t *testing.T) {
	fixture := readFixture(t, "client-shapes/gmail-sheets-crm-lead-handoff.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(fixture))); got != "78167448d0ba60eeae148d9cdf594087ba0193fbf119562145453a4c3ecd15b2" {
		t.Fatalf("client-shaped fixture changed without updating its evidence: %s", got)
	}
	workflow, _, _, err := normalizeWorkflow(fixture)
	if err != nil {
		t.Fatalf("normalize client-shaped workflow: %v", err)
	}
	triggerID := ""
	for _, node := range workflow.Nodes {
		if node.Trigger == TriggerManual {
			triggerID = node.ID
		}
	}
	if triggerID == "" {
		t.Fatal("client-shaped workflow has no Manual Trigger")
	}
	profile := readFixture(t, "native_service_catalog.json")
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Date(2026, 11, 11, 1, 0, 0, 0, time.UTC) })
	run, err := service.CreateAndRun(context.Background(), CreateInput{
		Title:               "Client-shaped lead intake with CRM, audit sheet, and Gmail handoff",
		WorkflowJSON:        fixture,
		TriggerNodeID:       triggerID,
		SimulatorProfile:    "native-service-catalog",
		SimulatorSourceJSON: profile,
		RequiredOutputPaths: []string{"status"},
	})
	if err != nil {
		t.Fatalf("run client-shaped workflow: %v", err)
	}
	for _, operation := range []string{"hubspot:contact.upsert", "googleSheets:sheet.append", "gmail:message.send"} {
		if !traceHasOperation(run.Trace, operation) {
			t.Errorf("client-shaped handoff omitted synthetic operation %q: %#v", operation, run.Trace)
		}
	}
	if run.State != StateNeedsReview || run.CredentialReplacements != 3 || !strings.Contains(run.HandoffMarkdown, "Human review is required") || !traceHasOperation(run.Trace, "assert output path exists") {
		t.Fatalf("client-shaped workflow did not produce a complete handoff for human review: %#v", run)
	}
	if strings.Contains(run.HandoffMarkdown, "synthetic.invalid") || strings.Contains(run.HandoffMarkdown, "client data") {
		t.Fatalf("handoff leaked synthetic profile details: %s", run.HandoffMarkdown)
	}
}
