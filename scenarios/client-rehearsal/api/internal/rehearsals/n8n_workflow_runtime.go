package rehearsals

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxN8NRuntimeCommandOutput = 32 << 10

type N8NTaskRuntimeLease struct {
	SandboxID string
	RuntimeID string
}

type N8NWorkflowExecutionEvidence struct {
	OperationID          string
	TriggerKind          string
	DispatchMode         string
	Status               string
	OutputSHA            string
	OutputBytes          int
	ResponseSHA256       string
	ResponseBytes        int
	NodeCount            int
	RunCount             int
	ItemCount            int
	FailedRuns           int
	NodeOutputItemCounts []int
	HTTPStatus           int
	InputSHA256          string
	InputBytes           int
	PublicationState     string
	Published            bool
	CleanupRequired      bool
}

// N8NWorkflowRuntime owns the fixed control-plane calls used by one task run.
// It never takes Docker arguments, paths, node IDs, or credentials from a
// caller. The n8n resource remains the authority for runtime state and import.
type N8NWorkflowRuntime interface {
	Start(context.Context, string) (N8NTaskRuntimeLease, error)
	ImportAndExecuteManual(context.Context, N8NTaskRuntimeLease, string) (N8NWorkflowExecutionEvidence, error)
	Stop(context.Context, N8NTaskRuntimeLease) error
}

type N8NSyntheticWebhookRuntime interface {
	ImportAndDispatchWebhook(context.Context, N8NTaskRuntimeLease, string, []byte) (N8NWorkflowExecutionEvidence, error)
}

// N8NScheduledWorkflowRuntime runs one sanitized, inactive schedule-triggered
// graph through the native n8n owner CLI. It does not activate the workflow or
// wait for wall-clock scheduling.
type N8NScheduledWorkflowRuntime interface {
	ImportAndExecuteScheduled(context.Context, N8NTaskRuntimeLease, string) (N8NWorkflowExecutionEvidence, error)
}

// N8NSyntheticTriggerInputRuntime injects bounded, run-local input into the
// task copy for trigger shapes whose native CLI entrypoint emits an empty item.
// The source/public workflow remains untouched.
type N8NSyntheticTriggerInputRuntime interface {
	ImportAndExecuteManualWithInput(context.Context, N8NTaskRuntimeLease, string, map[string]any) (N8NWorkflowExecutionEvidence, error)
	ImportAndExecuteScheduledWithInput(context.Context, N8NTaskRuntimeLease, string, map[string]any) (N8NWorkflowExecutionEvidence, error)
}

type N8NWorkflowRuntimeCLI struct {
	Binary string
	Runner N8NWorkflowCommandRunner
	// seam: rehearsals.N8NWorkflowRuntimeCLI.Environment substitutes the
	// allowlisted child-process environment for deterministic tests.
	Environment func() []string
}

// seam: rehearsals.N8NWorkflowCommandRunner substitutes the owner command
// runner while retaining fixed, adapter-owned arguments.
// N8NWorkflowCommandRunner is the deterministic seam around the fixed CLI
// commands used by this owner adapter. Implementations receive only owner-built
// argv and bounded stdin; they cannot be selected by workflow data.
type N8NWorkflowCommandRunner interface {
	Run(context.Context, string, []string, []byte, ...string) ([]byte, error)
}

type execN8NWorkflowCommandRunner struct{}

func (execN8NWorkflowCommandRunner) Run(ctx context.Context, binary string, environment []string, input []byte, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = environment
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	stdout, stderr := &boundedCommandOutput{}, &boundedCommandOutput{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		// Owner stderr can contain third-party node names or workflow fields.
		return nil, n8nOwnerCommandFailureFromStreams(n8nOwnerFailurePayload(stdout.Bytes(), stderr.Bytes()), stderr.Bytes(), stderr.truncated, err)
	}
	if stdout.truncated || stderr.truncated {
		return nil, errors.New("n8n resource owner command output exceeded capture limit")
	}
	return stdout.Bytes(), nil
}

func n8nOwnerFailurePayload(stdout, stderr []byte) []byte {
	if isN8NOwnerErrorEnvelope(stdout) {
		return stdout
	}
	if isN8NOwnerErrorEnvelope(stderr) {
		return stderr
	}
	return stdout
}

func isN8NOwnerErrorEnvelope(payload []byte) bool {
	var envelope struct {
		Success *bool `json:"success"`
	}
	return json.Unmarshal(payload, &envelope) == nil && envelope.Success != nil && !*envelope.Success
}

func n8nOwnerCommandFailure(stdout []byte, cause error) error {
	var envelope struct {
		Success  *bool  `json:"success"`
		Error    string `json:"error"`
		Category string `json:"category"`
		Code     string `json:"code"`
		Hint     string `json:"hint"`
	}
	if json.Unmarshal(stdout, &envelope) == nil && envelope.Success != nil && !*envelope.Success {
		if envelope.Code == "n8n_webhook_dispatch_failed" && envelope.Category == "Runtime" && envelope.Error == "synthetic Webhook dispatch failed" {
			if failureClass, diagnosticSHA256, ok := parseN8NWebhookFailureHint(envelope.Hint); ok {
				if diagnosticSHA256 != "" {
					return fmt.Errorf("n8n resource owner webhook dispatch failed (failureClass=%s diagnosticSha256=%s): %w", failureClass, diagnosticSHA256, cause)
				}
				return fmt.Errorf("n8n resource owner webhook dispatch failed (failureClass=%s): %w", failureClass, cause)
			}
		}
		if envelope.Code == "n8n_workflow_execution_failed" && envelope.Category == "Runtime" && envelope.Error == "synthetic workflow execution failed" {
			if failureClass, diagnosticSHA256, ok := parseN8NManualExecutionFailureHint(envelope.Hint); ok {
				if diagnosticSHA256 != "" {
					return fmt.Errorf("n8n resource owner workflow execution failed (failureClass=%s diagnosticSha256=%s): %w", failureClass, diagnosticSHA256, cause)
				}
				return fmt.Errorf("n8n resource owner workflow execution failed (failureClass=%s): %w", failureClass, cause)
			}
		}
		if envelope.Code == "n8n_workflow_import_failed" && envelope.Category == "Runtime" && envelope.Error == "synthetic workflow import failed" {
			if failureClass, diagnosticSHA256, ok := parseN8NManualExecutionFailureHint(envelope.Hint); ok {
				if diagnosticSHA256 != "" {
					return fmt.Errorf("n8n resource owner workflow import failed (failureClass=%s diagnosticSha256=%s): %w", failureClass, diagnosticSHA256, cause)
				}
				return fmt.Errorf("n8n resource owner workflow import failed (failureClass=%s): %w", failureClass, cause)
			}
		}
		if envelope.Code == "n8n_manual_execution_failed" && envelope.Category == "Runtime" && envelope.Error == "synthetic Manual Trigger execution failed" {
			if failureClass, diagnosticSHA256, ok := parseN8NManualExecutionFailureHint(envelope.Hint); ok {
				if diagnosticSHA256 != "" {
					return fmt.Errorf("n8n resource owner manual execution failed (failureClass=%s diagnosticSha256=%s): %w", failureClass, diagnosticSHA256, cause)
				}
				return fmt.Errorf("n8n resource owner manual execution failed (failureClass=%s): %w", failureClass, cause)
			}
		}
		category := safeOwnerErrorToken(envelope.Category)
		code := safeOwnerErrorToken(envelope.Code)
		if category != "" && code != "" {
			return fmt.Errorf("n8n resource owner command failed (category=%s code=%s): %w", category, code, cause)
		}
	}
	return fmt.Errorf("n8n resource owner command failed: %w", cause)
}

// n8nOwnerCommandFailureFromStreams preserves structured owner classifications
// when available. For an unstructured CLI failure, it emits only a fixed
// allowlisted class and a digest of bounded stderr; command output itself never
// crosses this boundary.
func n8nOwnerCommandFailureFromStreams(payload, stderr []byte, stderrTruncated bool, cause error) error {
	if isN8NOwnerErrorEnvelope(payload) {
		failure := n8nOwnerCommandFailure(payload, cause)
		if len(stderr) > 0 && bytes.Equal(payload, stderr) {
			return n8nOwnerFailureWithStderrDigest(failure, stderr, stderrTruncated)
		}
		return failure
	}
	if len(stderr) == 0 {
		return fmt.Errorf("n8n resource owner command failed (failureClass=cli-exit-no-diagnostic): %w", cause)
	}
	return n8nOwnerFailureWithStderrDigest(fmt.Errorf("n8n resource owner command failed: %w", cause), stderr, stderrTruncated)
}

func n8nOwnerFailureWithStderrDigest(failure error, stderr []byte, stderrTruncated bool) error {
	digest := sha256.Sum256(stderr)
	return fmt.Errorf("%w (failureClass=cli-stderr-diagnostic diagnosticSha256=%s capturedDiagnosticBytes=%d diagnosticTruncated=%t)",
		failure, hex.EncodeToString(digest[:]), len(stderr), stderrTruncated)
}

func parseN8NWebhookFailureHint(hint string) (failureClass, diagnosticSHA256 string, ok bool) {
	classes := map[string]struct{}{
		"tmpfs-prepare-error": {}, "workflow-export-error": {}, "workflow-export-read-error": {},
		"workflow-export-read-limit": {}, "workflow-route-invalid": {}, "webhook-dispatch-error": {},
		"workflow-publish-error": {}, "workflow-publish-cli-unavailable": {},
		"workflow-publish-workflow-missing": {}, "workflow-publish-ownership-rejected": {},
		"workflow-publish-runtime-error": {}, "workflow-publish-output-limit": {}, "unknown": {},
	}
	parts := strings.Fields(hint)
	if len(parts) < 1 || len(parts) > 2 || !strings.HasPrefix(parts[0], "failure_class=") {
		return "", "", false
	}
	failureClass = strings.TrimPrefix(parts[0], "failure_class=")
	if _, known := classes[failureClass]; !known {
		return "", "", false
	}
	if len(parts) == 2 {
		if !strings.HasPrefix(parts[1], "diagnostic_sha256=") {
			return "", "", false
		}
		diagnosticSHA256 = strings.TrimPrefix(parts[1], "diagnostic_sha256=")
		if !isLowerN8NDiagnosticSHA256(diagnosticSHA256) {
			return "", "", false
		}
	}
	return failureClass, diagnosticSHA256, true
}

func isLowerN8NDiagnosticSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func parseN8NManualExecutionFailureHint(hint string) (failureClass, diagnosticSHA256 string, ok bool) {
	knownClasses := map[string]struct{}{
		"cli-command-unavailable": {}, "database-unavailable": {}, "engine-execution-error": {},
		"engine-exit-no-diagnostic": {}, "engine-exit-with-result": {}, "execution-permission-denied": {},
		"execution-result-shape-unsupported": {}, "execution-timeout": {}, "node-api-error": {},
		"node-execution-error": {}, "node-operation-error": {}, "node-type-unavailable": {},
		"owner-initialization": {}, "instance-owner-initialization-error": {}, "instance-owner-missing": {},
		"instance-owner-not-found": {}, "owner-user-not-found": {}, "user-management-uninitialized": {},
		"runtime-dependency-missing": {}, "task-runner-broker-port-conflict": {},
		"workflow-invalid": {}, "workflow-not-found": {}, "workflow-project-unavailable": {},
		"workflow-returned-no-data": {}, "workflow-run-error": {}, "workflow-start-node-unavailable": {},
		"workflow-state-invalid": {}, "workflow-user-error": {}, "server-cli-import-error": {},
		"import-diagnostic-exceeded": {}, "workflow-credential-invalid": {}, "unknown": {},
	}
	parts := strings.Fields(hint)
	if len(parts) < 1 || len(parts) > 2 || !strings.HasPrefix(parts[0], "failure_class=") {
		return "", "", false
	}
	failureClass = strings.TrimPrefix(parts[0], "failure_class=")
	if _, known := knownClasses[failureClass]; !known {
		return "", "", false
	}
	if len(parts) == 2 {
		if !strings.HasPrefix(parts[1], "diagnostic_sha256=") {
			return "", "", false
		}
		diagnosticSHA256 = strings.TrimPrefix(parts[1], "diagnostic_sha256=")
		if !isN8NDiagnosticSHA256(diagnosticSHA256) {
			return "", "", false
		}
	}
	return failureClass, diagnosticSHA256, true
}

func isN8NDiagnosticSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func safeOwnerErrorToken(value string) string {
	if len(value) == 0 || len(value) > 32 {
		return ""
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.') {
			return ""
		}
	}
	return value
}

func (c N8NWorkflowRuntimeCLI) Start(ctx context.Context, sandboxID string) (N8NTaskRuntimeLease, error) {
	if c.Binary == "" || !sandboxIDPattern.MatchString(sandboxID) {
		return N8NTaskRuntimeLease{}, errors.New("n8n task runtime requires a control-plane executable and Sandbox UUID")
	}
	output, err := c.run(ctx, nil, "resource", "n8n", "runtime", "start", "--sandbox-id", sandboxID, "--json")
	if err != nil {
		return N8NTaskRuntimeLease{}, err
	}
	var result struct {
		SandboxID string `json:"sandboxId"`
		RuntimeID string `json:"runtimeId"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(output, &result); err != nil || result.SandboxID != sandboxID || result.RuntimeID == "" || result.Status != "running" {
		return N8NTaskRuntimeLease{}, errors.New("n8n owner returned an invalid runtime start receipt")
	}
	return N8NTaskRuntimeLease{SandboxID: sandboxID, RuntimeID: result.RuntimeID}, nil
}

func (c N8NWorkflowRuntimeCLI) ImportAndExecuteManual(ctx context.Context, lease N8NTaskRuntimeLease, workflowJSON string) (N8NWorkflowExecutionEvidence, error) {
	return c.ImportAndExecuteManualWithInput(ctx, lease, workflowJSON, nil)
}

func (c N8NWorkflowRuntimeCLI) ImportAndExecuteManualWithInput(ctx context.Context, lease N8NTaskRuntimeLease, workflowJSON string, input map[string]any) (N8NWorkflowExecutionEvidence, error) {
	if !validTaskRuntimeLease(lease) || strings.TrimSpace(workflowJSON) == "" {
		return N8NWorkflowExecutionEvidence{}, errors.New("n8n workflow import requires a valid task runtime and workflow JSON")
	}
	if input != nil {
		var err error
		workflowJSON, err = injectSyntheticInputAfterEntry(workflowJSON, input, TriggerManual)
		if err != nil {
			return N8NWorkflowExecutionEvidence{}, err
		}
	}
	operationID, err := c.importInactiveWorkflow(ctx, lease, workflowJSON)
	if err != nil {
		return N8NWorkflowExecutionEvidence{}, err
	}
	output, err := c.run(ctx, nil, "resource", "n8n", "runtime", "execute-manual-workflow", "--sandbox-id", lease.SandboxID, "--runtime-id", lease.RuntimeID, "--operation-id", operationID, "--json")
	if err != nil {
		return N8NWorkflowExecutionEvidence{OperationID: operationID}, fmt.Errorf("n8n manual execution failed: %w", err)
	}
	var executed struct {
		SandboxID   string `json:"sandboxId"`
		RuntimeID   string `json:"runtimeId"`
		OperationID string `json:"operationId"`
		Status      string `json:"status"`
		Evidence    struct {
			OutputSHA            string `json:"outputSha256"`
			Bytes                int    `json:"outputBytes"`
			Nodes                int    `json:"nodeCount"`
			Runs                 int    `json:"executionCount"`
			Items                int    `json:"itemCount"`
			Failed               int    `json:"failedNodeExecutionCount"`
			NodeOutputItemCounts []int  `json:"nodeOutputItemCounts"`
		} `json:"evidence"`
	}
	if err := json.Unmarshal(output, &executed); err != nil || executed.SandboxID != lease.SandboxID || executed.RuntimeID != lease.RuntimeID || executed.OperationID != operationID || executed.Status != "completed" {
		return N8NWorkflowExecutionEvidence{OperationID: operationID, Status: executed.Status}, errors.New("n8n owner returned an incomplete manual execution receipt")
	}
	return N8NWorkflowExecutionEvidence{OperationID: operationID, TriggerKind: string(TriggerManual), Status: executed.Status, OutputSHA: executed.Evidence.OutputSHA, OutputBytes: executed.Evidence.Bytes, NodeCount: executed.Evidence.Nodes, RunCount: executed.Evidence.Runs, ItemCount: executed.Evidence.Items, FailedRuns: executed.Evidence.Failed, NodeOutputItemCounts: append([]int(nil), executed.Evidence.NodeOutputItemCounts...)}, nil
}

func (c N8NWorkflowRuntimeCLI) ImportAndExecuteScheduled(ctx context.Context, lease N8NTaskRuntimeLease, workflowJSON string) (N8NWorkflowExecutionEvidence, error) {
	return c.ImportAndExecuteScheduledWithInput(ctx, lease, workflowJSON, nil)
}

func (c N8NWorkflowRuntimeCLI) ImportAndExecuteScheduledWithInput(ctx context.Context, lease N8NTaskRuntimeLease, workflowJSON string, input map[string]any) (N8NWorkflowExecutionEvidence, error) {
	if !validTaskRuntimeLease(lease) || strings.TrimSpace(workflowJSON) == "" {
		return N8NWorkflowExecutionEvidence{}, errors.New("n8n scheduled execution requires a valid task runtime and workflow JSON")
	}
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil || !nativeScheduledWorkflowSupported(workflow) {
		return N8NWorkflowExecutionEvidence{}, errors.New("n8n scheduled execution accepts only one deterministic schedule trigger and bounded pure/HTTP nodes")
	}
	workflowJSON, err = rewriteScheduledWorkflowForOneShotDispatch(workflowJSON, workflow)
	if err != nil {
		return N8NWorkflowExecutionEvidence{}, err
	}
	if input != nil {
		workflowJSON, err = injectSyntheticInputAfterEntry(workflowJSON, input, TriggerManual)
		if err != nil {
			return N8NWorkflowExecutionEvidence{}, err
		}
	}
	operationID, err := c.importInactiveWorkflow(ctx, lease, workflowJSON)
	if err != nil {
		return N8NWorkflowExecutionEvidence{}, err
	}
	output, err := c.run(ctx, nil, "resource", "n8n", "runtime", "execute-workflow", "--sandbox-id", lease.SandboxID, "--runtime-id", lease.RuntimeID, "--operation-id", operationID, "--json")
	if err != nil {
		return N8NWorkflowExecutionEvidence{OperationID: operationID, TriggerKind: string(TriggerScheduled), DispatchMode: "manual-one-shot"}, fmt.Errorf("n8n scheduled graph execution failed: %w", err)
	}
	var executed struct {
		SandboxID   string `json:"sandboxId"`
		RuntimeID   string `json:"runtimeId"`
		OperationID string `json:"operationId"`
		Status      string `json:"status"`
		Evidence    struct {
			OutputSHA            string `json:"outputSha256"`
			Bytes                int    `json:"outputBytes"`
			Nodes                int    `json:"nodeCount"`
			Runs                 int    `json:"executionCount"`
			Items                int    `json:"itemCount"`
			Failed               int    `json:"failedNodeExecutionCount"`
			NodeOutputItemCounts []int  `json:"nodeOutputItemCounts"`
		} `json:"evidence"`
	}
	if err := json.Unmarshal(output, &executed); err != nil || executed.SandboxID != lease.SandboxID || executed.RuntimeID != lease.RuntimeID || executed.OperationID != operationID || executed.Status != "completed" {
		return N8NWorkflowExecutionEvidence{OperationID: operationID, TriggerKind: string(TriggerScheduled), DispatchMode: "manual-one-shot", Status: executed.Status}, errors.New("n8n owner returned an incomplete scheduled execution receipt")
	}
	return N8NWorkflowExecutionEvidence{OperationID: operationID, TriggerKind: string(TriggerScheduled), DispatchMode: "manual-one-shot", Status: executed.Status, OutputSHA: executed.Evidence.OutputSHA, OutputBytes: executed.Evidence.Bytes, NodeCount: executed.Evidence.Nodes, RunCount: executed.Evidence.Runs, ItemCount: executed.Evidence.Items, FailedRuns: executed.Evidence.Failed, NodeOutputItemCounts: append([]int(nil), executed.Evidence.NodeOutputItemCounts...)}, nil
}

// injectSyntheticInputAfterEntry inserts a native Set node into the isolated
// execution copy. It bridges CLI-trigger shapes (manual and schedule rewritten
// to manual) that otherwise begin with an empty item. Provider rewrites remain
// a separate compiler step; this only preserves the caller's bounded input.
func injectSyntheticInputAfterEntry(raw string, input map[string]any, expected TriggerKind) (string, error) {
	normalized, _, err := normalizeTriggerInput(input)
	if err != nil || normalized == nil {
		return "", errors.New("native trigger input must be a bounded synthetic JSON object")
	}
	if expected != TriggerManual {
		return "", errors.New("native synthetic input injection is reserved for CLI-startable manual triggers")
	}
	var document map[string]any
	if err := json.Unmarshal([]byte(raw), &document); err != nil || document == nil {
		return "", errors.New("native input injection requires a workflow object")
	}
	nodes, ok := document["nodes"].([]any)
	if !ok {
		return "", errors.New("native input injection requires workflow nodes")
	}
	var trigger map[string]any
	triggerCount := 0
	for _, item := range nodes {
		node, ok := item.(map[string]any)
		if !ok || node["disabled"] == true {
			continue
		}
		typeName, _ := node["type"].(string)
		if kind, isTrigger := classifyTrigger(typeName); isTrigger {
			triggerCount++
			if kind == TriggerManual {
				trigger = node
			}
		}
	}
	if trigger == nil || triggerCount != 1 {
		return "", errors.New("native input injection requires exactly one manual CLI trigger")
	}
	triggerName, _ := trigger["name"].(string)
	connections, ok := document["connections"].(map[string]any)
	if !ok {
		return "", errors.New("native input injection requires trigger connections")
	}
	connection, ok := connections[triggerName].(map[string]any)
	if !ok {
		return "", errors.New("native input injection requires a connected trigger")
	}
	main, ok := connection["main"].([]any)
	if !ok || len(main) == 0 {
		return "", errors.New("native input injection requires a main trigger output")
	}
	used := map[string]bool{}
	usedNames := map[string]bool{}
	for _, item := range nodes {
		if node, ok := item.(map[string]any); ok {
			if id, _ := node["id"].(string); id != "" {
				used[id] = true
			}
			if name, _ := node["name"].(string); name != "" {
				usedNames[name] = true
			}
		}
	}
	setID := "vrooli-synthetic-input"
	for suffix := 2; used[setID]; suffix++ {
		setID = fmt.Sprintf("vrooli-synthetic-input-%d", suffix)
	}
	triggerPosition, _ := trigger["position"].([]any)
	position := []any{0, 0}
	if len(triggerPosition) == 2 {
		if x, ok := triggerPosition[0].(float64); ok {
			position[0] = x + 200
		}
		position[1] = triggerPosition[1]
	}
	keys := make([]string, 0, len(normalized))
	for key := range normalized {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	assignments := make([]any, 0, len(keys))
	for i, key := range keys {
		value := normalized[key]
		typeName := "string"
		switch value.(type) {
		case bool:
			typeName = "boolean"
		case float64:
			typeName = "number"
		case map[string]any, []any:
			typeName = "object"
			if _, array := value.([]any); array {
				typeName = "array"
			}
			encodedValue, marshalErr := json.Marshal(value)
			if marshalErr != nil {
				return "", errors.New("encode native synthetic input assignment")
			}
			value = "=" + string(encodedValue)
		case nil:
			typeName = "object"
			value = "=null"
		}
		assignments = append(assignments, map[string]any{"id": fmt.Sprintf("synthetic-%d", i+1), "name": key, "type": typeName, "value": value})
	}
	setName := "Vrooli Synthetic Input"
	for suffix := 2; usedNames[setName]; suffix++ {
		setName = fmt.Sprintf("Vrooli Synthetic Input %d", suffix)
	}
	targets := main
	triggerConnection := make(map[string]any, len(connection))
	for key, value := range connection {
		triggerConnection[key] = value
	}
	triggerConnection["main"] = []any{[]any{map[string]any{"node": setName, "type": "main", "index": 0}}}
	connections[triggerName] = triggerConnection
	connections[setName] = map[string]any{"main": targets}
	nodes = append(nodes, map[string]any{
		"id": setID, "name": setName, "type": "n8n-nodes-base.set", "typeVersion": 3.4,
		"position": position, "parameters": map[string]any{"assignments": map[string]any{"assignments": assignments}, "options": map[string]any{}},
	})
	document["nodes"] = nodes
	encoded, err := json.Marshal(document)
	if err != nil {
		return "", errors.New("encode native synthetic input task copy")
	}
	return string(encoded), nil
}

// rewriteScheduledWorkflowForOneShotDispatch maps the pinned source schedule
// trigger to Manual Trigger only in the task-local execution copy. n8n CLI does
// not emit a wall-clock scheduler event for inactive imports; this adapter
// invokes the workflow body once while preserving the source trigger in the
// public export and reporting the dispatch mode explicitly.
func rewriteScheduledWorkflowForOneShotDispatch(raw string, workflow Workflow) (string, error) {
	if len(workflow.EntryNodeIDs) != 1 {
		return "", errors.New("scheduled one-shot dispatch requires exactly one schedule trigger")
	}
	var document map[string]any
	if err := json.Unmarshal([]byte(raw), &document); err != nil || document == nil {
		return "", errors.New("scheduled one-shot dispatch requires a workflow object")
	}
	nodes, ok := document["nodes"].([]any)
	if !ok {
		return "", errors.New("scheduled one-shot dispatch requires workflow nodes")
	}
	targetID := workflow.EntryNodeIDs[0]
	matched := false
	for _, rawNode := range nodes {
		node, ok := rawNode.(map[string]any)
		if !ok || node["id"] != targetID {
			continue
		}
		typeName, _ := node["type"].(string)
		if typeName != "n8n-nodes-base.cron" && typeName != "n8n-nodes-base.scheduleTrigger" {
			return "", errors.New("scheduled one-shot dispatch target is not a schedule trigger")
		}
		node["type"] = "n8n-nodes-base.manualTrigger"
		node["typeVersion"] = 1
		node["parameters"] = map[string]any{}
		matched = true
	}
	if !matched {
		return "", errors.New("scheduled one-shot dispatch could not locate its schedule trigger")
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		return "", errors.New("encode scheduled workflow one-shot execution copy")
	}
	return string(encoded), nil
}

func nativeScheduledWorkflowSupported(workflow Workflow) bool {
	if len(workflow.EntryNodeIDs) != 1 || len(workflow.Nodes) < 2 {
		return false
	}
	triggerID := workflow.EntryNodeIDs[0]
	triggerCount, requestCount := 0, 0
	allowed := map[string]bool{
		"n8n-nodes-base.cron": true, "n8n-nodes-base.scheduleTrigger": true,
		"n8n-nodes-base.code": true,
		"n8n-nodes-base.set":  true, "n8n-nodes-base.if": true, "n8n-nodes-base.switch": true,
		"n8n-nodes-base.merge": true, "n8n-nodes-base.noOp": true, "n8n-nodes-base.sort": true,
		"n8n-nodes-base.filter": true, "n8n-nodes-base.aggregate": true,
	}
	for _, node := range workflow.Nodes {
		if node.ID == triggerID {
			if node.Trigger != TriggerScheduled || node.Type != "n8n-nodes-base.cron" && node.Type != "n8n-nodes-base.scheduleTrigger" {
				return false
			}
			triggerCount++
			continue
		}
		if node.Type == "n8n-nodes-base.httpRequest" {
			if node.Request == nil || node.Request.Host == "" || node.Request.URL == "" {
				return false
			}
			requestCount++
			continue
		}
		if allowed[node.Type] {
			continue
		}
		// Provider nodes may reach the task companion only when the
		// description adapter resolved an operation selector. The service
		// preflight subsequently requires every such selector to match an
		// exact host/method/path profile binding before runtime start.
		if node.Request == nil || node.Request.Resource == "" || node.Request.Action == "" {
			return false
		}
		requestCount++
	}
	return triggerCount == 1 && requestCount > 0
}

func (c N8NWorkflowRuntimeCLI) ImportAndDispatchWebhook(ctx context.Context, lease N8NTaskRuntimeLease, workflowJSON string, eventJSON []byte) (N8NWorkflowExecutionEvidence, error) {
	if !validTaskRuntimeLease(lease) || strings.TrimSpace(workflowJSON) == "" {
		return N8NWorkflowExecutionEvidence{}, errors.New("n8n webhook rehearsal requires a valid task runtime and workflow JSON")
	}
	var event map[string]any
	if err := json.Unmarshal(eventJSON, &event); err != nil || event == nil {
		return N8NWorkflowExecutionEvidence{}, errors.New("n8n webhook rehearsal input must be a JSON object")
	}
	normalized, _, err := normalizeTriggerInput(event)
	if err != nil {
		return N8NWorkflowExecutionEvidence{}, fmt.Errorf("n8n webhook rehearsal input is invalid: %w", err)
	}
	eventJSON, err = json.Marshal(normalized)
	if err != nil {
		return N8NWorkflowExecutionEvidence{}, errors.New("n8n webhook rehearsal input could not be normalized")
	}
	operationID, err := c.importInactiveWorkflow(ctx, lease, workflowJSON)
	if err != nil {
		return N8NWorkflowExecutionEvidence{}, err
	}
	output, err := c.run(ctx, eventJSON, "resource", "n8n", "runtime", "dispatch-webhook-event", "--sandbox-id", lease.SandboxID, "--runtime-id", lease.RuntimeID, "--operation-id", operationID, "--event-stdin", "--json")
	if err != nil {
		return N8NWorkflowExecutionEvidence{OperationID: operationID, TriggerKind: string(TriggerWebhook)}, fmt.Errorf("n8n synthetic webhook dispatch failed: %w", err)
	}
	var receipt struct {
		SandboxID        string `json:"sandboxId"`
		RuntimeID        string `json:"runtimeId"`
		OperationID      string `json:"operationId"`
		Status           string `json:"status"`
		InputSHA256      string `json:"inputSha256"`
		InputBytes       int    `json:"inputBytes"`
		HTTPStatus       int    `json:"httpStatus"`
		ResponseSHA256   string `json:"responseSha256"`
		ResponseBytes    int    `json:"responseBytes"`
		PublicationState string `json:"publicationState"`
		Published        bool   `json:"published"`
		CleanupRequired  bool   `json:"cleanupRequired"`
	}
	inputHash := sha256.Sum256(eventJSON)
	unmarshalErr := json.Unmarshal(output, &receipt)
	responseDigest, digestErr := hex.DecodeString(receipt.ResponseSHA256)
	if unmarshalErr != nil || receipt.SandboxID != lease.SandboxID || receipt.RuntimeID != lease.RuntimeID || receipt.OperationID != operationID || receipt.Status != "completed" || receipt.PublicationState != "unpublish-requested" || receipt.CleanupRequired || receipt.HTTPStatus < 200 || receipt.HTTPStatus >= 300 || receipt.InputBytes != len(eventJSON) || receipt.InputSHA256 != hex.EncodeToString(inputHash[:]) || digestErr != nil || len(responseDigest) != sha256.Size || receipt.ResponseBytes < 0 || receipt.ResponseBytes > 1<<20 {
		return N8NWorkflowExecutionEvidence{OperationID: operationID, TriggerKind: string(TriggerWebhook), Status: receipt.Status, HTTPStatus: receipt.HTTPStatus, InputSHA256: receipt.InputSHA256, InputBytes: receipt.InputBytes, ResponseSHA256: receipt.ResponseSHA256, ResponseBytes: receipt.ResponseBytes, PublicationState: receipt.PublicationState, Published: receipt.Published, CleanupRequired: receipt.CleanupRequired}, errors.New("n8n owner returned an incomplete synthetic webhook receipt")
	}
	return N8NWorkflowExecutionEvidence{OperationID: operationID, TriggerKind: string(TriggerWebhook), Status: receipt.Status, HTTPStatus: receipt.HTTPStatus, InputSHA256: receipt.InputSHA256, InputBytes: receipt.InputBytes, ResponseSHA256: receipt.ResponseSHA256, ResponseBytes: receipt.ResponseBytes, PublicationState: receipt.PublicationState, Published: receipt.Published, CleanupRequired: receipt.CleanupRequired}, nil
}

func (c N8NWorkflowRuntimeCLI) importInactiveWorkflow(ctx context.Context, lease N8NTaskRuntimeLease, workflowJSON string) (string, error) {
	operationID := uuid.NewString()
	args := []string{"resource", "n8n", "runtime", "import-workflow", "--sandbox-id", lease.SandboxID, "--runtime-id", lease.RuntimeID, "--operation-id", operationID, "--json"}
	output, err := c.run(ctx, []byte(workflowJSON), args...)
	if err != nil {
		return "", fmt.Errorf("n8n workflow import failed: %w", err)
	}
	var imported struct {
		SandboxID   string `json:"sandboxId"`
		RuntimeID   string `json:"runtimeId"`
		OperationID string `json:"operationId"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal(output, &imported); err != nil || imported.SandboxID != lease.SandboxID || imported.RuntimeID != lease.RuntimeID || imported.OperationID != operationID || imported.Status != "imported-inactive" {
		return "", errors.New("n8n owner returned an invalid inactive workflow import receipt")
	}
	return operationID, nil
}

func (c N8NWorkflowRuntimeCLI) Stop(ctx context.Context, lease N8NTaskRuntimeLease) error {
	if !validTaskRuntimeLease(lease) {
		return errors.New("n8n task runtime stop requires a valid owner-issued Sandbox/runtime identity")
	}
	_, err := c.run(ctx, nil, "resource", "n8n", "runtime", "stop", "--sandbox-id", lease.SandboxID, "--runtime-id", lease.RuntimeID, "--json")
	return err
}

func validTaskRuntimeLease(lease N8NTaskRuntimeLease) bool {
	return sandboxIDPattern.MatchString(lease.SandboxID) && strings.HasPrefix(lease.RuntimeID, "n8n:") && len(lease.RuntimeID) <= 128
}

func (c N8NWorkflowRuntimeCLI) run(ctx context.Context, input []byte, args ...string) ([]byte, error) {
	if c.Binary == "" {
		return nil, errors.New("vrooli control-plane executable is not configured")
	}
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	runner := c.Runner
	if runner == nil {
		runner = execN8NWorkflowCommandRunner{}
	}
	environment := c.Environment
	if environment == nil {
		return nil, errors.New("n8n workflow command environment is not configured")
	}
	output, err := runner.Run(callCtx, c.Binary, environment(), input, args...)
	if err != nil {
		return nil, err
	}
	if len(output) > maxN8NRuntimeCommandOutput {
		return nil, fmt.Errorf("owner command output exceeds %d bytes", maxN8NRuntimeCommandOutput)
	}
	return output, nil
}

type boundedCommandOutput struct {
	bytes.Buffer
	truncated bool
}

func (w *boundedCommandOutput) Write(data []byte) (int, error) {
	remaining := maxN8NRuntimeCommandOutput - w.Len()
	if remaining < len(data) {
		w.truncated = true
		if remaining > 0 {
			_, _ = w.Buffer.Write(data[:remaining])
		}
		// Keep draining the child process while retaining only a bounded prefix.
		return len(data), nil
	}
	return w.Buffer.Write(data)
}

var _ io.Writer = (*boundedCommandOutput)(nil)
