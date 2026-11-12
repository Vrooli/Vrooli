package n8n

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
)

var syntheticManualNodeAllowlist = map[string]struct{}{
	"n8n-nodes-base.manualTrigger": {},
	"n8n-nodes-base.set":           {},
	"n8n-nodes-base.splitOut":      {},
	"n8n-nodes-base.if":            {},
	"n8n-nodes-base.switch":        {},
	"n8n-nodes-base.merge":         {},
	"n8n-nodes-base.noOp":          {},
	"n8n-nodes-base.httpRequest":   {},
	"n8n-nodes-base.postgres":      {},
}

type ManualWorkflowExecutionReceipt struct {
	SandboxID       string                    `json:"sandboxId"`
	RuntimeID       string                    `json:"runtimeId"`
	OperationID     string                    `json:"operationId"`
	Status          string                    `json:"status"`
	Evidence        WorkflowExecutionEvidence `json:"evidence,omitempty"`
	CleanupRequired bool                      `json:"cleanupRequired,omitempty"`
}

type exportedSyntheticManualWorkflow struct {
	ID       string         `json:"id"`
	Active   *bool          `json:"active"`
	Settings map[string]any `json:"settings"`
	Nodes    []struct {
		Type       string         `json:"type"`
		Disabled   bool           `json:"disabled"`
		Parameters map[string]any `json:"parameters"`
	} `json:"nodes"`
}

// ExecuteSyntheticManualWorkflow runs only an imported workflow whose sole
// active trigger is the pinned built-in Manual Trigger. The workflow ID is
// derived from owner identity and operation ID, and the CLI argv is fixed.
func (o *Owner) ExecuteSyntheticManualWorkflow(ctx context.Context, sandboxID, runtimeID string, operationID uuid.UUID) (receipt ManualWorkflowExecutionReceipt, resultErr error) {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return ManualWorkflowExecutionReceipt{}, err
	}
	if runtimeID != runtimeIDFor(canonical) {
		return ManualWorkflowExecutionReceipt{}, ErrForeignObject
	}
	if operationID == uuid.Nil {
		return ManualWorkflowExecutionReceipt{}, errors.New("manual workflow operation ID must be a nonzero UUID")
	}
	canonical, containerName, err := o.ownedRunningContainer(ctx, canonical, runtimeID)
	if err != nil {
		return ManualWorkflowExecutionReceipt{}, err
	}
	runner, ok := o.runner().(quietCommandRunner)
	if !ok {
		return ManualWorkflowExecutionReceipt{}, errors.New("runtime command runner cannot guarantee manual workflow output redaction")
	}
	workflowID := workflowIDFor(canonical, runtimeID, operationID)
	stagePath := "/tmp/.vrooli-manual-export-" + workflowID + "-" + uuid.NewString() + ".json"
	receipt = ManualWorkflowExecutionReceipt{
		SandboxID: canonical, RuntimeID: runtimeID, OperationID: operationID.String(), Status: "failed",
	}
	if _, err := runner.RunQuiet(ctx, "exec", "--user", "node", containerName, "rm", "-f", "--", stagePath); err != nil && !isNotFound(err) {
		receipt.CleanupRequired = true
		return receipt, errors.New("could not prepare task tmpfs for manual workflow lookup")
	}
	defer func() {
		if cleanupErr := removeWorkflowInput(context.Background(), runner, containerName, stagePath); cleanupErr != nil {
			receipt.CleanupRequired = true
			if receipt.Status == "completed" {
				receipt.Status = "completed-cleanup-required"
			}
			if resultErr == nil {
				resultErr = errors.New("manual workflow ran but temporary export cleanup failed")
			}
		}
	}()
	if _, err := runner.RunQuiet(ctx, "exec", "--user", "node", containerName, "n8n", "export:workflow", "--id="+workflowID, "--output="+stagePath); err != nil {
		return receipt, errors.New("owner-imported manual workflow lookup failed")
	}
	exportBytes, exceeded, err := runner.RunStdoutLimitQuiet(ctx, maxSyntheticWorkflowJSON, "exec", "--user", "node", containerName, "node", "-e", readSyntheticWorkflowProgram, stagePath)
	if err != nil || exceeded || len(exportBytes) > maxSyntheticWorkflowJSON {
		return receipt, errors.New("owner-imported manual workflow lookup exceeded its bounded export")
	}
	if err := validateExportedSyntheticManualWorkflow(exportBytes, workflowID); err != nil {
		return receipt, err
	}
	execution, err := o.ExecuteSyntheticWorkflow(ctx, canonical, runtimeID, operationID)
	if err != nil {
		return receipt, errors.New("synthetic Manual Trigger execution failed")
	}
	receipt.Status = execution.Status
	receipt.Evidence = execution.Evidence
	return receipt, nil
}

func validateSyntheticSplitOutNode(parameters map[string]any) error {
	if len(parameters) == 0 {
		return errors.New("Split Out parameters are required")
	}
	field, ok := parameters["fieldToSplitOut"].(string)
	if !ok || !validSyntheticFieldPath(field) {
		return errors.New("Split Out fieldToSplitOut must be a bounded dotted JSON field path")
	}
	include, _ := parameters["include"].(string)
	switch include {
	case "", "noOtherFields", "allOtherFields":
	case "selectedOtherFields":
		fields, ok := parameters["fieldsToInclude"].(string)
		if !ok || len(fields) == 0 || len(fields) > 512 {
			return errors.New("Split Out selected fields must be a bounded field list")
		}
		for _, selected := range strings.Split(fields, ",") {
			if !validSyntheticFieldPath(strings.TrimSpace(selected)) {
				return errors.New("Split Out selected field list contains an invalid path")
			}
		}
	default:
		return errors.New("Split Out include mode is unsupported")
	}
	for key, value := range parameters {
		if key == "fieldToSplitOut" || key == "include" || key == "fieldsToInclude" {
			continue
		}
		if key == "options" {
			options, ok := value.(map[string]any)
			if !ok || len(options) != 0 {
				return errors.New("Split Out options are not modeled")
			}
			continue
		}
		return fmt.Errorf("Split Out parameter %q is outside the synthetic execution profile", key)
	}
	return nil
}

func validSyntheticFieldPath(path string) bool {
	if len(path) == 0 || len(path) > 256 || strings.TrimSpace(path) != path {
		return false
	}
	for _, segment := range strings.Split(path, ".") {
		if segment == "" || segment == "__proto__" || segment == "prototype" || segment == "constructor" {
			return false
		}
		for _, r := range segment {
			if !(r == '_' || r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
				return false
			}
		}
	}
	return true
}

func validateExportedSyntheticManualWorkflow(data []byte, expectedWorkflowID string) error {
	if len(data) == 0 || len(data) > maxSyntheticWorkflowJSON {
		return errors.New("manual workflow export is empty or oversized")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := checkUniqueJSONValue(decoder); err != nil {
		return errors.New("manual workflow export has duplicate or malformed JSON fields")
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("manual workflow export contains trailing JSON")
	}
	var exports []exportedSyntheticManualWorkflow
	if err := json.Unmarshal(data, &exports); err != nil || len(exports) != 1 {
		return errors.New("manual workflow export must contain exactly one workflow")
	}
	workflow := exports[0]
	if workflow.ID != expectedWorkflowID || workflow.Active == nil || *workflow.Active || len(workflow.Nodes) == 0 || len(workflow.Nodes) > 256 {
		return errors.New("manual workflow identity or node count is invalid")
	}
	if rawErrorWorkflow, exists := workflow.Settings["errorWorkflow"]; exists {
		errorWorkflow, ok := rawErrorWorkflow.(string)
		if !ok || strings.TrimSpace(errorWorkflow) != "" {
			return errors.New("manual workflow error-workflow setting is outside the synthetic execution profile")
		}
	}
	manualTriggerCount := 0
	for _, node := range workflow.Nodes {
		if _, allowed := syntheticManualNodeAllowlist[node.Type]; !allowed {
			return errors.New("workflow node type is outside the synthetic Manual Trigger execution profile")
		}
		if node.Disabled {
			continue
		}
		if node.Type == "n8n-nodes-base.manualTrigger" {
			manualTriggerCount++
		}
		switch node.Type {
		case "n8n-nodes-base.httpRequest":
			if err := validateSyntheticHTTPNode(node.Parameters); err != nil {
				return err
			}
		case "n8n-nodes-base.splitOut":
			if err := validateSyntheticSplitOutNode(node.Parameters); err != nil {
				return err
			}
		}
	}
	if manualTriggerCount != 1 {
		return fmt.Errorf("workflow must contain exactly one enabled built-in Manual Trigger; found %d", manualTriggerCount)
	}
	return nil
}
