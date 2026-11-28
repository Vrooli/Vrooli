package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPublic6338ManualReminderUsesGenericSyntheticSheetsAndMail(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/6338.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON))); got != "6d960557d679630dd8ec561ac3ec552c0563990607678fb9027e83220cb08f7b" {
		t.Fatalf("sanitized fixture hash changed without provenance update: %s", got)
	}
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("import sanitized public reminder export: %v", err)
	}
	if len(workflow.Nodes) != 4 || len(workflow.EntryNodeIDs) != 1 || workflow.TriggerKinds[0] != TriggerManual {
		t.Fatalf("unexpected public manual reminder graph shape: nodes=%d entries=%v triggers=%v", len(workflow.Nodes), workflow.EntryNodeIDs, workflow.TriggerKinds)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-6338-profile.json"), false)
	if err != nil {
		t.Fatalf("load synthetic Sheets and Gmail profile: %v", err)
	}
	run, err := executeDetailedWithInput(context.Background(), workflow, workflow.EntryNodeIDs[0], simulator, false, time.Date(2026, 11, 12, 8, 30, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatalf("run sanitized public reminder graph: %v trace=%#v", err, run.Trace)
	}
	for _, operation := range []string{"googleSheets:sheet.read", "gmail:message.send"} {
		if !traceHasSuccessfulOperation(run.Trace, operation) {
			t.Errorf("expected synthetic operation %q in trace: %#v", operation, run.Trace)
		}
	}
	if strings.Contains(strings.Join(traceSummaries(run.Trace), " "), "gmail.com") {
		t.Fatal("trace retained a non-synthetic mail destination")
	}
}
