package n8n

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestValidateManualWorkflowAllowsOnlyOnePinnedManualTrigger(t *testing.T) {
	id := workflowIDFor(testSandboxID, runtimeIDFor(testSandboxID), uuid.MustParse(workflowOperationID))
	base := `[{"id":"` + id + `","active":false,"nodes":[{"type":"n8n-nodes-base.manualTrigger","disabled":false,"parameters":{}},{"type":"n8n-nodes-base.httpRequest","disabled":false,"parameters":{"url":"http://crm.synthetic.invalid:18080/v1/contacts"}}]}]`
	if err := validateExportedSyntheticManualWorkflow([]byte(base), id); err != nil {
		t.Fatalf("safe manual workflow rejected: %v", err)
	}
	splitWorkflow := strings.Replace(base, `{"type":"n8n-nodes-base.httpRequest","disabled":false,"parameters":{"url":"http://crm.synthetic.invalid:18080/v1/contacts"}}`, `{"type":"n8n-nodes-base.splitOut","disabled":false,"parameters":{"fieldToSplitOut":"assignment.invoices","include":"noOtherFields","options":{}}}`, 1)
	if err := validateExportedSyntheticManualWorkflow([]byte(splitWorkflow), id); err != nil {
		t.Fatalf("bounded local Split Out rejected: %v", err)
	}
	for name, unsafe := range map[string]string{
		"empty split path":         strings.Replace(splitWorkflow, `"assignment.invoices"`, `""`, 1),
		"malformed split path":     strings.Replace(splitWorkflow, `"assignment.invoices"`, `"assignment..invoices"`, 1),
		"prototype path":           strings.Replace(splitWorkflow, `"assignment.invoices"`, `"assignment.__proto__"`, 1),
		"unsupported split option": strings.Replace(splitWorkflow, `"options":{}`, `"options":{"binaryPropertyName":"payload"}`, 1),
	} {
		t.Run("Split Out "+name, func(t *testing.T) {
			if err := validateExportedSyntheticManualWorkflow([]byte(unsafe), id); err == nil {
				t.Fatalf("unsupported Split Out configuration was accepted: %s", unsafe)
			}
		})
	}
	for name, unsafe := range map[string]string{
		"schedule trigger":       strings.Replace(base, `"type":"n8n-nodes-base.manualTrigger"`, `"type":"n8n-nodes-base.scheduleTrigger"`, 1),
		"webhook trigger":        strings.Replace(base, `"type":"n8n-nodes-base.manualTrigger"`, `"type":"n8n-nodes-base.webhook"`, 1),
		"extra manual trigger":   strings.Replace(base, `{"type":"n8n-nodes-base.httpRequest","disabled":false,"parameters":{"url":"http://crm.synthetic.invalid:18080/v1/contacts"}}`, `{"type":"n8n-nodes-base.manualTrigger","disabled":false,"parameters":{}}`, 1),
		"code node":              strings.Replace(base, `"type":"n8n-nodes-base.httpRequest"`, `"type":"n8n-nodes-base.code"`, 1),
		"external URL":           strings.Replace(base, "crm.synthetic.invalid", "api.example.com", 1),
		"nested proxy":           strings.Replace(base, `"url":"http://crm.synthetic.invalid:18080/v1/contacts"`, `"url":"http://crm.synthetic.invalid:18080/v1/contacts","options":{"proxy":"http://proxy.example"}`, 1),
		"published workflow":     strings.Replace(base, `"active":false`, `"active":true`, 1),
		"wrong identity":         strings.Replace(base, id, "anotherworkflowid", 1),
		"duplicate JSON key":     strings.Replace(base, `"active":false`, `"active":false,"active":false`, 1),
		"error workflow":         strings.Replace(base, `"active":false`, `"active":false,"settings":{"errorWorkflow":"external-workflow"}`, 1),
		"invalid error workflow": strings.Replace(base, `"active":false`, `"active":false,"settings":{"errorWorkflow":{}}`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateExportedSyntheticManualWorkflow([]byte(unsafe), id); err == nil {
				t.Fatalf("unsafe manual workflow accepted: %s", unsafe)
			}
		})
	}
}

func TestExecuteSyntheticManualWorkflowUsesFixedCLIAndRedactedReceipt(t *testing.T) {
	runner := ownedRuntimeRunner(t, runtimeIDFor(testSandboxID))
	baseCall := runner.call
	operationID := uuid.MustParse(workflowOperationID)
	workflowID := workflowIDFor(testSandboxID, runtimeIDFor(testSandboxID), operationID)
	exported := `[{"id":"` + workflowID + `","active":false,"nodes":[{"type":"n8n-nodes-base.manualTrigger","disabled":false,"parameters":{}},{"type":"n8n-nodes-base.set","disabled":false,"parameters":{}}]}]`
	const secret = "private-result-content"
	runData := `{"data":{"resultData":{"runData":{"Set values":[{"data":{"main":[[{"json":{"secret":"` + secret + `"}}]]}}]}}}}`
	var lifecycle []string
	var executeArgs []string
	stageCleanup := false
	runner.call = func(ctx context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, " n8n export:workflow "):
			lifecycle = append(lifecycle, "export")
			return nil, nil
		case len(args) > 5 && args[0] == "exec" && args[4] == "node" && args[5] == "-e":
			return []byte(exported), nil
		case strings.Contains(joined, " n8n execute "):
			lifecycle = append(lifecycle, "execute")
			executeArgs = append([]string(nil), args...)
			return []byte(runData), nil
		case len(args) > 0 && args[0] == "exec" && containsArg(args, "rm"):
			stageCleanup = true
			return nil, nil
		default:
			return baseCall(ctx, args...)
		}
	}
	owner := &Owner{Runner: runner}
	receipt, err := owner.ExecuteSyntheticManualWorkflow(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), operationID)
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"exec", "--user", "node", "vrooli-n8n-5e33a6f0b1bf4f5d931595ba60d9e025", "n8n", "execute", "--id", workflowID, "--rawOutput"}
	if strings.Join(executeArgs, "\x00") != strings.Join(wantArgs, "\x00") {
		t.Fatalf("manual execution did not use the fixed CLI command: %#v", executeArgs)
	}
	if strings.Join(lifecycle, ",") != "export,execute" || !stageCleanup {
		t.Fatalf("manual operation lifecycle = %v, temporary export cleaned=%t", lifecycle, stageCleanup)
	}
	if receipt.Status != "completed" || receipt.Evidence.NodeCount != 1 || receipt.Evidence.ItemCount != 1 || receipt.CleanupRequired {
		t.Fatalf("unexpected receipt: %#v", receipt)
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), workflowID) || strings.Contains(string(encoded), secret) || strings.Contains(string(encoded), "Set values") {
		t.Fatalf("manual execution receipt exposed private identity or output: %s", encoded)
	}
}

func TestExecuteSyntheticManualWorkflowRejectsScheduleBeforeExecution(t *testing.T) {
	runner := ownedRuntimeRunner(t, runtimeIDFor(testSandboxID))
	baseCall := runner.call
	operationID := uuid.MustParse(workflowOperationID)
	workflowID := workflowIDFor(testSandboxID, runtimeIDFor(testSandboxID), operationID)
	exported := `[{"id":"` + workflowID + `","active":false,"nodes":[{"type":"n8n-nodes-base.scheduleTrigger","disabled":false,"parameters":{}}]}]`
	executed := false
	runner.call = func(ctx context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, " n8n export:workflow ") {
			return nil, nil
		}
		if len(args) > 5 && args[0] == "exec" && args[4] == "node" && args[5] == "-e" {
			return []byte(exported), nil
		}
		if strings.Contains(joined, " n8n execute ") {
			executed = true
			return []byte(`{}`), nil
		}
		if len(args) > 0 && args[0] == "exec" && containsArg(args, "rm") {
			return nil, nil
		}
		return baseCall(ctx, args...)
	}
	owner := &Owner{Runner: runner}
	if _, err := owner.ExecuteSyntheticManualWorkflow(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), operationID); err == nil {
		t.Fatal("manual-only operation accepted a Schedule Trigger")
	}
	if executed {
		t.Fatal("Schedule workflow reached n8n execute")
	}
}

func TestValidateManualWorkflowRejectsBadJSON(t *testing.T) {
	if err := validateExportedSyntheticManualWorkflow([]byte(`[{"id": "x"}] trailing`), "x"); err == nil {
		t.Fatal("trailing workflow export data was accepted")
	}
	if err := validateExportedSyntheticManualWorkflow([]byte(fmt.Sprintf(`[{"id":%q,"active":false,"nodes":[]}]`, "x")), "x"); err == nil {
		t.Fatal("workflow with no trigger was accepted")
	}
}
