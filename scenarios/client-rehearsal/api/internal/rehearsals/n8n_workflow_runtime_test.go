package rehearsals

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

type scriptedN8NRunner struct {
	calls             [][]string
	input             [][]byte
	webhookHTTPStatus int
}

func (r *scriptedN8NRunner) Run(_ context.Context, binary string, environment []string, input []byte, args ...string) ([]byte, error) {
	if binary != "/opt/vrooli/bin/vrooli" || len(environment) == 0 || environment[0] != "PATH=/usr/bin:/bin" {
		return nil, errUnexpectedN8NCommand
	}
	r.calls = append(r.calls, append([]string(nil), args...))
	r.input = append(r.input, append([]byte(nil), input...))
	value := func(flag string) string {
		for i := 0; i+1 < len(args); i++ {
			if args[i] == flag {
				return args[i+1]
			}
		}
		return ""
	}
	sandboxID, runtimeID, operationID := value("--sandbox-id"), value("--runtime-id"), value("--operation-id")
	switch {
	case strings.Contains(strings.Join(args, " "), " runtime start "):
		return []byte(`{"sandboxId":"` + sandboxID + `","runtimeId":"n8n:` + sandboxID + `","status":"running"}`), nil
	case strings.Contains(strings.Join(args, " "), " import-workflow "):
		return []byte(`{"sandboxId":"` + sandboxID + `","runtimeId":"` + runtimeID + `","operationId":"` + operationID + `","status":"imported-inactive"}`), nil
	case strings.Contains(strings.Join(args, " "), " execute-manual-workflow "):
		return []byte(`{"sandboxId":"` + sandboxID + `","runtimeId":"` + runtimeID + `","operationId":"` + operationID + `","status":"completed","evidence":{"outputSha256":"synthetic-hash","outputBytes":32,"nodeCount":2,"executionCount":1,"itemCount":1,"failedNodeExecutionCount":0,"nodeOutputItemCounts":[1,1]}}`), nil
	case strings.Contains(strings.Join(args, " "), " execute-workflow "):
		return []byte(`{"sandboxId":"` + sandboxID + `","runtimeId":"` + runtimeID + `","operationId":"` + operationID + `","status":"completed","evidence":{"outputSha256":"scheduled-hash","outputBytes":48,"nodeCount":2,"executionCount":1,"itemCount":1,"failedNodeExecutionCount":0,"nodeOutputItemCounts":[1,1]}}`), nil
	case strings.Contains(strings.Join(args, " "), " dispatch-webhook-event "):
		hash := sha256.Sum256(input)
		status, responseStatus := "completed", 202
		if r.webhookHTTPStatus != 0 {
			status, responseStatus = "completed-with-http-error", r.webhookHTTPStatus
		}
		return []byte(`{"sandboxId":"` + sandboxID + `","runtimeId":"` + runtimeID + `","operationId":"` + operationID + `","status":"` + status + `","inputSha256":"` + hex.EncodeToString(hash[:]) + `","inputBytes":` + fmt.Sprint(len(input)) + `,"httpStatus":` + fmt.Sprint(responseStatus) + `,"responseSha256":"` + strings.Repeat("c", 64) + `","responseBytes":12,"publicationState":"unpublish-requested","published":true,"cleanupRequired":false}`), nil
	default:
		return []byte(`{}`), nil
	}
}

func TestN8NWorkflowRuntimeCLIImportsAndExecutesBoundedScheduleGraph(t *testing.T) {
	const sandboxID = "9c54f876-6b12-4a55-a11d-65e18af3d223"
	lease := N8NTaskRuntimeLease{SandboxID: sandboxID, RuntimeID: "n8n:" + sandboxID}
	runner := &scriptedN8NRunner{}
	owner := N8NWorkflowRuntimeCLI{Binary: "/opt/vrooli/bin/vrooli", Runner: runner, Environment: func() []string { return []string{"PATH=/usr/bin:/bin"} }}
	workflow := `{"name":"schedule flow","nodes":[{"id":"s","name":"Schedule","type":"n8n-nodes-base.scheduleTrigger","parameters":{"rule":{"interval":[{"field":"days","daysInterval":1,"triggerAtHour":8,"triggerAtMinute":0}]}}},{"id":"r","name":"Request","type":"n8n-nodes-base.httpRequest","parameters":{"method":"POST","url":"https://crm.synthetic.invalid/v1/contacts"}}],"connections":{"Schedule":{"main":[[{"node":"Request"}]]}}}`
	evidence, err := owner.ImportAndExecuteScheduledWithInput(context.Background(), lease, workflow, map[string]any{"feedUrl": "https://feeds.synthetic.invalid/approved.xml"})
	if err != nil {
		t.Fatal(err)
	}
	if evidence.TriggerKind != string(TriggerScheduled) || evidence.DispatchMode != "manual-one-shot" || evidence.Status != "completed" || evidence.NodeCount != 2 || evidence.OutputSHA != "scheduled-hash" {
		t.Fatalf("unexpected native schedule receipt: %+v", evidence)
	}
	if len(runner.calls) != 2 || !strings.Contains(strings.Join(runner.calls[0], " "), "import-workflow") || !strings.Contains(strings.Join(runner.calls[1], " "), "execute-workflow") || !strings.Contains(string(runner.input[0]), `"type":"n8n-nodes-base.manualTrigger"`) || strings.Contains(string(runner.input[0]), `"type":"n8n-nodes-base.scheduleTrigger"`) || !strings.Contains(string(runner.input[0]), `"name":"feedUrl"`) || !strings.Contains(string(runner.input[0]), "feeds.synthetic.invalid") || !strings.Contains(string(runner.input[0]), `"node":"Vrooli Synthetic Input"`) || len(runner.input[1]) != 0 {
		t.Fatalf("scheduled execution must import a manual one-shot copy through the fixed owner execution: calls=%v inputs=%q", runner.calls, runner.input)
	}
	var injected map[string]any
	if err := json.Unmarshal(runner.input[0], &injected); err != nil {
		t.Fatal(err)
	}
	connections := injected["connections"].(map[string]any)
	setConnection := connections["Vrooli Synthetic Input"].(map[string]any)
	mainOutputs := setConnection["main"].([]any)
	firstOutput := mainOutputs[0].([]any)
	if firstOutput[0].(map[string]any)["node"] != "Request" {
		t.Fatalf("synthetic input adapter must preserve the original downstream edge: %#v", connections)
	}
	unsafe := strings.Replace(workflow, `"n8n-nodes-base.scheduleTrigger"`, `"n8n-nodes-base.googleSheets"`, 1)
	runner.calls, runner.input = nil, nil
	if _, err := owner.ImportAndExecuteScheduled(context.Background(), lease, unsafe); err == nil || len(runner.calls) != 0 {
		t.Fatalf("unsupported schedule graph must fail before owner calls: err=%v calls=%v", err, runner.calls)
	}

	scheduledIntegration, _, _, err := normalizeWorkflow(`{"name":"scheduled native sheet append","nodes":[{"id":"schedule","name":"Schedule","type":"n8n-nodes-base.scheduleTrigger","parameters":{"rule":{"interval":[{"field":"days","daysInterval":1,"triggerAtHour":8,"triggerAtMinute":0}]}}},{"id":"append","name":"Append synthetic lead","type":"n8n-nodes-base.googleSheets","typeVersion":4.7,"parameters":{"resource":"sheet","operation":"append","documentId":{"__rl":true,"mode":"id","value":"synthetic-sheet-id"},"sheetName":{"__rl":true,"mode":"name","value":"Leads"}}}],"connections":{"Schedule":{"main":[[{"node":"Append synthetic lead"}]]}}}`)
	if err != nil {
		t.Fatalf("normalize scheduled native integration graph: %v", err)
	}
	if !nativeScheduledWorkflowSupported(scheduledIntegration) {
		t.Fatal("schedule graph with a typed native integration operation should pass the reusable graph-admission layer")
	}
	apiSimulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-21013-native-profile.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	apiProfile, err := apiSimulator.ExportVirtualAPIProfile()
	if err != nil || !workflowUsesExactVirtualAPIRoutes(scheduledIntegration, apiProfile) {
		t.Fatalf("scheduled native operation must be bound to an exact API-description route: profileErr=%v", err)
	}
}

func TestN8NNativeSyntheticInputAdapterSupportsManualAndRejectsWebhookShape(t *testing.T) {
	manual := `{"name":"manual flow","nodes":[{"id":"trigger","name":"Manual","type":"n8n-nodes-base.manualTrigger","typeVersion":1,"position":[100,200],"parameters":{}},{"id":"end","name":"End","type":"n8n-nodes-base.noOp","position":[300,200],"parameters":{}},{"id":"existing-input","name":"Vrooli Synthetic Input","type":"n8n-nodes-base.noOp","parameters":{}}],"connections":{"Manual":{"main":[[{"node":"End","type":"main","index":0}]],"futurePort":[[{"node":"End","type":"futurePort","index":0}]]}}}`
	adapted, err := injectSyntheticInputAfterEntry(manual, map[string]any{"record": map[string]any{"id": "synthetic-7"}, "active": true}, TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(adapted, `"name":"record"`) || !strings.Contains(adapted, `"name":"active"`) || !strings.Contains(adapted, `"node":"End"`) {
		t.Fatalf("manual adapter lost synthetic input or original graph: %s", adapted)
	}
	if !strings.Contains(adapted, `"futurePort"`) || !strings.Contains(adapted, `"name":"Vrooli Synthetic Input 2"`) {
		t.Fatal("manual adapter dropped an unrelated original connection port")
	}
	webhook := `{"name":"webhook flow","nodes":[{"id":"trigger","name":"Webhook","type":"n8n-nodes-base.webhook","typeVersion":2,"position":[0,0],"parameters":{"path":"synthetic","httpMethod":"POST"}},{"id":"end","name":"End","type":"n8n-nodes-base.noOp","position":[200,0],"parameters":{}}],"connections":{"Webhook":{"main":[[{"node":"End","type":"main","index":0}]]}}}`
	if _, err := injectSyntheticInputAfterEntry(webhook, map[string]any{"event": "synthetic"}, TriggerWebhook); err == nil {
		t.Fatal("webhook input must remain on its owner-dispatched event path")
	}
}

func TestN8NManualCLIImportsTaskCopyWithSyntheticInput(t *testing.T) {
	const sandboxID = "9c54f876-6b12-4a55-a11d-65e18af3d223"
	lease := N8NTaskRuntimeLease{SandboxID: sandboxID, RuntimeID: "n8n:" + sandboxID}
	runner := &scriptedN8NRunner{}
	owner := N8NWorkflowRuntimeCLI{Binary: "/opt/vrooli/bin/vrooli", Runner: runner, Environment: func() []string { return []string{"PATH=/usr/bin:/bin"} }}
	workflow := `{"name":"manual flow","nodes":[{"id":"t","name":"Manual","type":"n8n-nodes-base.manualTrigger","typeVersion":1,"parameters":{}},{"id":"e","name":"End","type":"n8n-nodes-base.noOp","parameters":{}}],"connections":{"Manual":{"main":[[{"node":"End","type":"main","index":0}]]}}}`
	if _, err := owner.ImportAndExecuteManualWithInput(context.Background(), lease, workflow, map[string]any{"row": 3}); err != nil {
		t.Fatal(err)
	}
	if len(runner.input) != 2 || !bytes.Contains(runner.input[0], []byte(`"name":"row"`)) || !bytes.Contains(runner.input[0], []byte(`"value":3`)) || !bytes.Contains(runner.input[0], []byte(`"node":"Vrooli Synthetic Input"`)) {
		t.Fatalf("manual CLI task copy omitted generic synthetic input adapter: %q", runner.input)
	}
	if !bytes.Contains(runner.input[0], []byte(`"type":"n8n-nodes-base.manualTrigger"`)) || bytes.Contains(runner.input[0], []byte(`"name":"row"`)) && bytes.Contains([]byte(workflow), []byte(`"name":"row"`)) {
		t.Fatal("manual task-local adapter must retain the trigger and not mutate the source JSON")
	}
}

var errUnexpectedN8NCommand = &unexpectedN8NCommandError{}

type unexpectedN8NCommandError struct{}

func (*unexpectedN8NCommandError) Error() string { return "unexpected n8n command invocation" }

func TestN8NWorkflowRuntimeCLIUsesInjectedOwnerCommandSeam(t *testing.T) {
	const sandboxID = "9c54f876-6b12-4a55-a11d-65e18af3d223"
	runner := &scriptedN8NRunner{}
	environmentCalls := 0
	owner := N8NWorkflowRuntimeCLI{Binary: "/opt/vrooli/bin/vrooli", Runner: runner, Environment: func() []string {
		environmentCalls++
		return []string{"PATH=/usr/bin:/bin", "VROOLI_ROOT=/safe/root"}
	}}
	lease, err := owner.Start(context.Background(), sandboxID)
	if err != nil {
		t.Fatal(err)
	}
	if lease.SandboxID != sandboxID || lease.RuntimeID != "n8n:"+sandboxID {
		t.Fatalf("unexpected runtime lease: %+v", lease)
	}
	workflow := `{"name":"synthetic workflow","active":false}`
	evidence, err := owner.ImportAndExecuteManual(context.Background(), lease, workflow)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Status != "completed" || evidence.NodeCount != 2 || evidence.RunCount != 1 || evidence.ItemCount != 1 || evidence.OutputSHA != "synthetic-hash" || !slices.Equal(evidence.NodeOutputItemCounts, []int{1, 1}) {
		t.Fatalf("unexpected content-free execution evidence: %+v", evidence)
	}
	if err := owner.Stop(context.Background(), lease); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 4 || environmentCalls != 4 {
		t.Fatalf("runner calls=%d environment calls=%d, want 4 each", len(runner.calls), environmentCalls)
	}
	if got := strings.Join(runner.calls[1], " "); !strings.Contains(got, "import-workflow") || !strings.Contains(got, "--operation-id "+evidence.OperationID) {
		t.Fatalf("workflow import did not use owner-issued operation ID: %s", got)
	}
	if string(runner.input[1]) != workflow || len(runner.input[0]) != 0 || len(runner.input[2]) != 0 || len(runner.input[3]) != 0 {
		t.Fatal("workflow body must be staged only on the fixed import command")
	}
	var output any
	if json.Unmarshal(runner.input[1], &output) != nil {
		t.Fatal("runner did not receive the complete bounded workflow JSON")
	}
}

func TestN8NWorkflowRuntimeRequiresInjectedCommandEnvironment(t *testing.T) {
	owner := N8NWorkflowRuntimeCLI{Binary: "/opt/vrooli/bin/vrooli", Runner: &scriptedN8NRunner{}}
	_, err := owner.Start(context.Background(), "9c54f876-6b12-4a55-a11d-65e18af3d223")
	if err == nil || !strings.Contains(err.Error(), "environment is not configured") {
		t.Fatalf("Start error = %v, want missing injected environment", err)
	}
}

func TestN8NWorkflowRuntimeCLIImportsAndDispatchesSyntheticWebhookInput(t *testing.T) {
	const sandboxID = "9c54f876-6b12-4a55-a11d-65e18af3d223"
	lease := N8NTaskRuntimeLease{SandboxID: sandboxID, RuntimeID: "n8n:" + sandboxID}
	runner := &scriptedN8NRunner{}
	owner := N8NWorkflowRuntimeCLI{Binary: "/opt/vrooli/bin/vrooli", Runner: runner, Environment: func() []string { return []string{"PATH=/usr/bin:/bin"} }}
	event := []byte(`{"event":"synthetic"}`)
	evidence, err := owner.ImportAndDispatchWebhook(context.Background(), lease, `{"name":"webhook flow"}`, event)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.TriggerKind != string(TriggerWebhook) || evidence.Status != "completed" || evidence.HTTPStatus != 202 || evidence.InputBytes != len(event) || evidence.InputSHA256 == "" || evidence.ResponseBytes != 12 || evidence.ResponseSHA256 != strings.Repeat("c", 64) {
		t.Fatalf("unexpected webhook execution evidence: %+v", evidence)
	}
	if len(runner.calls) != 2 || !strings.Contains(strings.Join(runner.calls[0], " "), "import-workflow") || !strings.Contains(strings.Join(runner.calls[1], " "), "dispatch-webhook-event") || !strings.Contains(strings.Join(runner.calls[1], " "), "--event-stdin") || string(runner.input[1]) != string(event) {
		t.Fatalf("unexpected webhook owner calls/inputs: calls=%v inputs=%q", runner.calls, runner.input)
	}
}

func TestN8NWorkflowRuntimePreservesWebhookRouteFailureState(t *testing.T) {
	const sandboxID = "9c54f876-6b12-4a55-a11d-65e18af3d223"
	lease := N8NTaskRuntimeLease{SandboxID: sandboxID, RuntimeID: "n8n:" + sandboxID}
	runner := &scriptedN8NRunner{webhookHTTPStatus: http.StatusNotFound}
	owner := N8NWorkflowRuntimeCLI{Binary: "/opt/vrooli/bin/vrooli", Runner: runner, Environment: func() []string { return []string{"PATH=/usr/bin:/bin"} }}
	evidence, err := owner.ImportAndDispatchWebhook(context.Background(), lease, `{"name":"synthetic webhook"}`, []byte(`{"event":"synthetic"}`))
	if err == nil {
		t.Fatal("404 webhook response must remain a failed dispatch")
	}
	if evidence.Status != "completed-with-http-error" || evidence.HTTPStatus != http.StatusNotFound || evidence.PublicationState != "unpublish-requested" || !evidence.Published || evidence.CleanupRequired || len(evidence.ResponseSHA256) != 64 || evidence.ResponseBytes != 12 {
		t.Fatalf("webhook route failure did not preserve content-free publication evidence: %+v", evidence)
	}
}

func TestN8NOwnerCommandFailurePreservesOnlySafeTypedFields(t *testing.T) {
	cause := errors.New("exit status 1")
	err := n8nOwnerCommandFailure([]byte(`{"success":false,"category":"Runtime","code":"untyped_cli_error","error":"workflow secret data"}`), cause)
	if err == nil || !strings.Contains(err.Error(), "category=Runtime") || !strings.Contains(err.Error(), "code=untyped_cli_error") || strings.Contains(err.Error(), "workflow secret data") {
		t.Fatalf("typed owner error = %v, want safe category/code only", err)
	}
	err = n8nOwnerCommandFailure([]byte(`{"success":false,"category":"bad category","code":"secret-value"}`), cause)
	if err == nil || strings.Contains(err.Error(), "bad category") || strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("unsafe owner tokens escaped sanitization: %v", err)
	}
	err = n8nOwnerCommandFailure([]byte("not-json"), cause)
	if err == nil || strings.Contains(err.Error(), "not-json") || strings.Contains(err.Error(), "workflow secret data") {
		t.Fatalf("raw command output escaped sanitization: %v", err)
	}
}

func TestN8NOwnerCommandFailureSummarizesStderrWithoutExposingIt(t *testing.T) {
	cause := errors.New("exit status 1")
	secret := "synthetic workflow field secret-value"
	stderr := []byte(secret)
	digest := sha256.Sum256(stderr)
	err := n8nOwnerCommandFailureFromStreams(nil, stderr, false, cause)
	if err == nil || !strings.Contains(err.Error(), "failureClass=cli-stderr-diagnostic") || !strings.Contains(err.Error(), "diagnosticSha256="+hex.EncodeToString(digest[:])) || !strings.Contains(err.Error(), "capturedDiagnosticBytes=") || strings.Contains(err.Error(), secret) {
		t.Fatalf("stderr failure summary = %v, want only fixed class, digest and size", err)
	}
}

func TestN8NOwnerCommandFailureHandlesInvalidJSONAndTruncation(t *testing.T) {
	cause := errors.New("exit status 1")
	invalidJSON := []byte(`{"error":"private workflow secret`)
	digest := sha256.Sum256(invalidJSON)
	err := n8nOwnerCommandFailureFromStreams(nil, invalidJSON, false, cause)
	if err == nil || !strings.Contains(err.Error(), "failureClass=cli-stderr-diagnostic") || !strings.Contains(err.Error(), "diagnosticSha256="+hex.EncodeToString(digest[:])) || strings.Contains(err.Error(), "private workflow secret") {
		t.Fatalf("invalid JSON stderr summary = %v, want bounded content-free digest", err)
	}

	writer := &boundedCommandOutput{}
	input := bytes.Repeat([]byte("x"), maxN8NRuntimeCommandOutput+1)
	if n, writeErr := writer.Write(input); writeErr != nil || n != len(input) || !writer.truncated || writer.Len() != maxN8NRuntimeCommandOutput {
		t.Fatalf("bounded stderr capture = n:%d bytes:%d truncated:%t err:%v", n, writer.Len(), writer.truncated, writeErr)
	}
	truncatedDigest := sha256.Sum256(writer.Bytes())
	err = n8nOwnerCommandFailureFromStreams(nil, writer.Bytes(), writer.truncated, cause)
	if err == nil || !strings.Contains(err.Error(), "diagnosticSha256="+hex.EncodeToString(truncatedDigest[:])) || !strings.Contains(err.Error(), "capturedDiagnosticBytes=32768") || !strings.Contains(err.Error(), "diagnosticTruncated=true") {
		t.Fatalf("truncated stderr summary = %v, want digest of bounded prefix and truncation marker", err)
	}
}

func TestN8NOwnerCommandFailureUsesStructuredStderrAndNoDiagnosticClass(t *testing.T) {
	cause := errors.New("exit status 1")
	structured := []byte(`{"success":false,"category":"Runtime","code":"untyped_cli_error","error":"private workflow data"}`)
	digest := sha256.Sum256(structured)
	err := n8nOwnerCommandFailureFromStreams(n8nOwnerFailurePayload(nil, structured), structured, false, cause)
	if err == nil || !strings.Contains(err.Error(), "category=Runtime") || !strings.Contains(err.Error(), "code=untyped_cli_error") || !strings.Contains(err.Error(), "failureClass=cli-stderr-diagnostic") || !strings.Contains(err.Error(), "diagnosticSha256="+hex.EncodeToString(digest[:])) || strings.Contains(err.Error(), "private workflow data") {
		t.Fatalf("structured stderr summary = %v, want safe owner fields plus stderr digest", err)
	}

	err = n8nOwnerCommandFailureFromStreams(nil, nil, false, cause)
	if err == nil || !strings.Contains(err.Error(), "failureClass=cli-exit-no-diagnostic") || strings.Contains(err.Error(), "diagnosticSha256") {
		t.Fatalf("empty stderr summary = %v, want no-diagnostic class and no digest", err)
	}
}

func TestN8NManualExecutionFailurePreservesClassAndDiagnosticDigest(t *testing.T) {
	cause := errors.New("exit status 1")
	digest := strings.Repeat("b", 64)
	payload := []byte(`{"success":false,"error":"synthetic Manual Trigger execution failed","code":"n8n_manual_execution_failed","category":"Runtime","hint":"failure_class=engine-execution-error diagnostic_sha256=` + digest + `"}`)
	err := n8nOwnerCommandFailure(payload, cause)
	if err == nil || !strings.Contains(err.Error(), "failureClass=engine-execution-error") || !strings.Contains(err.Error(), "diagnosticSha256="+digest) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("manual failure envelope lost safe classification or digest: %v", err)
	}

	for name, invalid := range map[string]string{
		"unsafe class":  `{"success":false,"error":"synthetic Manual Trigger execution failed","code":"n8n_manual_execution_failed","category":"Runtime","hint":"failure_class=secret-value"}`,
		"bad digest":    `{"success":false,"error":"synthetic Manual Trigger execution failed","code":"n8n_manual_execution_failed","category":"Runtime","hint":"failure_class=node-api-error diagnostic_sha256=not-a-hash"}`,
		"wrong message": `{"success":false,"error":"leaked workflow payload","code":"n8n_manual_execution_failed","category":"Runtime","hint":"failure_class=node-api-error diagnostic_sha256=` + digest + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			err := n8nOwnerCommandFailure([]byte(invalid), cause)
			if err == nil || strings.Contains(err.Error(), "secret-value") || strings.Contains(err.Error(), "not-a-hash") || strings.Contains(err.Error(), "leaked workflow payload") {
				t.Fatalf("unsafe manual failure fields escaped the wrapper: %v", err)
			}
		})
	}
}

func TestN8NWorkflowExecutionFailurePreservesOnlyAllowlistedClassAndDigest(t *testing.T) {
	cause := errors.New("exit status 1")
	digest := strings.Repeat("e", 64)
	payload := []byte(`{"success":false,"error":"synthetic workflow execution failed","code":"n8n_workflow_execution_failed","category":"Runtime","hint":"failure_class=engine-execution-error diagnostic_sha256=` + digest + `"}`)
	err := n8nOwnerCommandFailure(payload, cause)
	if err == nil || !strings.Contains(err.Error(), "failureClass=engine-execution-error") || !strings.Contains(err.Error(), "diagnosticSha256="+digest) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("workflow execution envelope lost safe classification or digest: %v", err)
	}
	for name, invalid := range map[string]string{
		"unknown class": `{"success":false,"error":"synthetic workflow execution failed","code":"n8n_workflow_execution_failed","category":"Runtime","hint":"failure_class=secret-value"}`,
		"bad digest":    `{"success":false,"error":"synthetic workflow execution failed","code":"n8n_workflow_execution_failed","category":"Runtime","hint":"failure_class=engine-execution-error diagnostic_sha256=bad"}`,
		"wrong message": `{"success":false,"error":"workflow payload secret","code":"n8n_workflow_execution_failed","category":"Runtime","hint":"failure_class=engine-execution-error diagnostic_sha256=` + digest + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			err := n8nOwnerCommandFailure([]byte(invalid), cause)
			if err == nil || strings.Contains(err.Error(), "secret-value") || strings.Contains(err.Error(), "bad") || strings.Contains(err.Error(), "workflow payload secret") || strings.Contains(err.Error(), digest) {
				t.Fatalf("unsafe workflow execution fields escaped wrapper: %v", err)
			}
		})
	}
}

func TestN8NWorkflowImportFailurePreservesOnlyAllowlistedClassAndDigest(t *testing.T) {
	cause := errors.New("exit status 1")
	digest := strings.Repeat("7", 64)
	payload := []byte(`{"success":false,"error":"synthetic workflow import failed","code":"n8n_workflow_import_failed","category":"Runtime","hint":"failure_class=server-cli-import-error diagnostic_sha256=` + digest + `"}`)
	err := n8nOwnerCommandFailure(payload, cause)
	if err == nil || !strings.Contains(err.Error(), "workflow import failed (failureClass=server-cli-import-error") || !strings.Contains(err.Error(), "diagnosticSha256="+digest) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("workflow import envelope lost safe classification or digest: %v", err)
	}
	invalid := []byte(`{"success":false,"error":"private import output","code":"n8n_workflow_import_failed","category":"Runtime","hint":"failure_class=private-value"}`)
	err = n8nOwnerCommandFailure(invalid, cause)
	if err == nil || strings.Contains(err.Error(), "private-value") || strings.Contains(err.Error(), "private import output") {
		t.Fatalf("unsafe workflow import fields escaped adapter: %v", err)
	}
}

func TestN8NWebhookFailurePreservesOnlyAllowlistedClassAndDigest(t *testing.T) {
	cause := errors.New("exit status 1")
	digest := strings.Repeat("c", 64)
	payload := []byte(`{"success":false,"error":"synthetic Webhook dispatch failed","code":"n8n_webhook_dispatch_failed","category":"Runtime","hint":"failure_class=webhook-dispatch-error diagnostic_sha256=` + digest + `"}`)
	err := n8nOwnerCommandFailure(payload, cause)
	if err == nil || !strings.Contains(err.Error(), "failureClass=webhook-dispatch-error") || !strings.Contains(err.Error(), "diagnosticSha256="+digest) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("webhook failure envelope lost its safe classification or digest: %v", err)
	}
	for name, invalid := range map[string]string{
		"unknown class": `{"success":false,"error":"synthetic Webhook dispatch failed","code":"n8n_webhook_dispatch_failed","category":"Runtime","hint":"failure_class=private-value"}`,
		"bad digest":    `{"success":false,"error":"synthetic Webhook dispatch failed","code":"n8n_webhook_dispatch_failed","category":"Runtime","hint":"failure_class=webhook-dispatch-error diagnostic_sha256=private-value"}`,
		"wrong message": `{"success":false,"error":"payload secret","code":"n8n_webhook_dispatch_failed","category":"Runtime","hint":"failure_class=webhook-dispatch-error diagnostic_sha256=` + digest + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			err := n8nOwnerCommandFailure([]byte(invalid), cause)
			if err == nil || strings.Contains(err.Error(), "private-value") || strings.Contains(err.Error(), "payload secret") || strings.Contains(err.Error(), digest) {
				t.Fatalf("unsafe webhook failure fields escaped the caller wrapper: %v", err)
			}
		})
	}
}

func TestN8NOwnerFailurePayloadSelectsOnlyStructuredCLIError(t *testing.T) {
	stdout := []byte(`{"success":false,"code":"untyped_cli_error"}`)
	stderr := []byte(`diagnostic includes private workflow data`)
	if got := n8nOwnerFailurePayload(stdout, stderr); string(got) != string(stdout) {
		t.Fatalf("structured stdout error should win: %q", got)
	}
	if got := n8nOwnerFailurePayload([]byte(""), stdout); string(got) != string(stdout) {
		t.Fatalf("structured stderr CLI error must be retained: %q", got)
	}
	if got := n8nOwnerFailurePayload([]byte("ordinary output"), stderr); string(got) != "ordinary output" {
		t.Fatalf("unstructured stderr must not cross the error boundary: %q", got)
	}
}
