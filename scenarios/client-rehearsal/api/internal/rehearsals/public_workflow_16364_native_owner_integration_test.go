package rehearsals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// This opt-in run exercises the public scheduled newsletter through the
// isolated pinned n8n resource owner and exact task-private provider APIs. It
// does not use Workspace Sandbox and earns no Workspace Sandbox gate credit.
func TestLivePublic16364NativeOwnerScheduledRun(t *testing.T) {
	if os.Getenv("F7_LIVE_PUBLIC_16364_OWNER") != "1" {
		t.Skip("set F7_LIVE_PUBLIC_16364_OWNER=1 to run the isolated native schedule owner integration")
	}
	outPath := strings.TrimSpace(os.Getenv("F7_NATIVE_16364_RECEIPT_OUT"))
	if outPath == "" {
		t.Fatal("F7_NATIVE_16364_RECEIPT_OUT must name the content-free owner receipt path")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	binary := strings.TrimSpace(os.Getenv("F7_VROOLI_BIN"))
	if binary == "" {
		binary = "/home/matthalloran8/.vrooli/bin/vrooli"
	}
	owner := N8NWorkflowRuntimeCLI{Binary: binary, Environment: os.Environ}
	sandboxID := uuid.NewString()
	workflowJSON := readFixture(t, "public-workflows/16364.json")
	profileJSON := readFixture(t, "public-workflow-16364-native-profile.json")
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize sanitized public #16364: %v", err)
	}
	if !nativeScheduledWorkflowSupported(workflow) {
		t.Fatal("public #16364 no longer fits the generic bounded schedule/API native runner")
	}
	if !strings.Contains(profileJSON, `"2026-11-16"`) || !strings.Contains(profileJSON, `"2026-11-15"`) {
		t.Fatal("public schedule response no longer carries the two synthetic date fixtures")
	}
	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	profileJSON = strings.ReplaceAll(profileJSON, "2026-11-16", today)
	profileJSON = strings.ReplaceAll(profileJSON, "2026-11-15", yesterday)
	simulator, err := NewDescriptionSimulator(profileJSON, false)
	if err != nil {
		t.Fatalf("load public schedule Gmail/Sheets API descriptions: %v", err)
	}
	profile, err := simulator.ExportVirtualAPIProfile()
	if err != nil || len(profile.Hosts) != 2 || len(profile.Routes) != 3 || !workflowUsesExactVirtualAPIRoutes(workflow, profile) {
		t.Fatalf("public schedule profile does not bind its exact two Sheets reads and Gmail send: hosts=%v routes=%d err=%v", profile.Hosts, len(profile.Routes), err)
	}
	runtimeWorkflow, err := rewriteWorkflowHTTPRequestsForVirtualAPI(workflowJSON, workflow, profile)
	if err != nil {
		t.Fatalf("bind unchanged public #16364 to the exact task provider profile: %v", err)
	}
	contractRoutes := make([]struct {
		Host        string `json:"host"`
		Method      string `json:"method"`
		Path        string `json:"path"`
		Query       string `json:"query,omitempty"`
		Status      int    `json:"status"`
		ContentType string `json:"contentType"`
		Body        []byte `json:"body"`
	}, 0, len(profile.Routes))
	for _, route := range profile.Routes {
		contractRoutes = append(contractRoutes, struct {
			Host        string `json:"host"`
			Method      string `json:"method"`
			Path        string `json:"path"`
			Query       string `json:"query,omitempty"`
			Status      int    `json:"status"`
			ContentType string `json:"contentType"`
			Body        []byte `json:"body"`
		}{route.Host, route.Method, route.Path, route.Query, route.Status, route.ContentType, []byte(route.Body)})
	}
	contractJSON, err := json.Marshal(struct {
		TaskID string   `json:"taskId"`
		Hosts  []string `json:"hosts"`
		Routes any      `json:"routes"`
	}{sandboxID, profile.Hosts, contractRoutes})
	if err != nil {
		t.Fatal(err)
	}
	workflowDigest := sha256.Sum256([]byte(workflowJSON))
	profileDigest := sha256.Sum256(contractJSON)

	companionStart, err := owner.run(ctx, contractJSON, "resource", "n8n", "runtime", "virtual-api-start", "--sandbox-id", sandboxID, "--contract-stdin", "--json")
	if err != nil {
		t.Fatalf("start exact task-private schedule API companion for task %s: %v", sandboxID, err)
	}
	var companion struct {
		SandboxID          string `json:"sandboxId"`
		CompanionRuntimeID string `json:"companionRuntimeId"`
		Status             string `json:"status"`
		RouteCount         int    `json:"routeCount"`
		ProfileFingerprint string `json:"profileFingerprintSha256"`
		NoEgressVerified   bool   `json:"noEgressVerified"`
	}
	decodeErr := json.Unmarshal(companionStart, &companion)
	lease := N8NTaskRuntimeLease{SandboxID: sandboxID, RuntimeID: "n8n:" + sandboxID}
	companionID := companion.CompanionRuntimeID
	runtimeStopped, companionStopped := false, false
	var cleanupErr error
	var failedRunReceipt map[string]any
	runVariant := "prefix-diagnostic"
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cleanupCancel()
		if companionID != "" && !companionStopped {
			_, stopErr := owner.run(cleanupCtx, nil, "resource", "n8n", "runtime", "virtual-api-stop", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
			cleanupErr = errors.Join(cleanupErr, stopErr)
			companionStopped = stopErr == nil
		}
		if !runtimeStopped {
			stopErr := owner.Stop(cleanupCtx, lease)
			cleanupErr = errors.Join(cleanupErr, stopErr)
			runtimeStopped = stopErr == nil
		}
		if cleanupErr != nil {
			t.Errorf("isolated schedule owner cleanup failed: %v", cleanupErr)
		}
		if failedRunReceipt != nil {
			statusJSON, statusErr := owner.run(cleanupCtx, nil, "resource", "n8n", "runtime", "status", "--sandbox-id", sandboxID, "--json")
			var status struct {
				Status string `json:"status"`
			}
			if statusErr == nil {
				statusErr = json.Unmarshal(statusJSON, &status)
			}
			failedRunReceipt["cleanup"] = map[string]any{
				"companionStopped": companionStopped, "runtimeStopped": runtimeStopped,
				"runtimeStatus": status.Status, "statusVerified": statusErr == nil && status.Status == "stopped",
				"cleanupError": cleanupErr != nil, "statusError": statusErr != nil,
			}
			encoded, encodeErr := json.MarshalIndent(failedRunReceipt, "", "  ")
			if encodeErr == nil {
				encodeErr = os.WriteFile(outPath, append(encoded, '\n'), 0o600)
			}
			if encodeErr != nil {
				t.Errorf("write content-free failed scheduled owner receipt: %v", encodeErr)
			}
		}
	}()
	if decodeErr != nil || companion.SandboxID != sandboxID || companion.Status != "attached" || companion.RouteCount != 3 || companion.ProfileFingerprint != hex.EncodeToString(profileDigest[:]) || !companion.NoEgressVerified {
		t.Fatalf("task-private native provider API did not attest its bounded profile: receipt=%s err=%v", companionStart, decodeErr)
	}

	// Probe the unchanged public graph prefix through its first Code node and
	// provider route. This distinguishes task-local runner/owner setup failure
	// from a later graph node without retaining engine output.
	noCodeJSON, err := public16364NoCodeExecutionPrefix(workflowJSON)
	if err != nil {
		t.Fatalf("prepare #16364 trigger-to-provider probe: %v", err)
	}
	noCodeWorkflow, _, _, err := normalizeWorkflow(noCodeJSON)
	if err != nil {
		t.Fatalf("normalize #16364 trigger-to-provider probe: %v", err)
	}
	noCodeRuntimeJSON, err := rewriteWorkflowHTTPRequestsForVirtualAPI(noCodeJSON, noCodeWorkflow, profile)
	if err != nil {
		t.Fatalf("bind #16364 no-Code provider probe to exact task route: %v", err)
	}
	runVariant = "schedule-to-products-without-code"
	noCodeExecution, err := owner.ImportAndExecuteScheduled(ctx, lease, noCodeRuntimeJSON)
	if err != nil {
		failedRunReceipt = public16364FailureReceipt(ctx, owner, sandboxID, companionID, lease, workflowDigest, profileDigest, runVariant, err)
		t.Fatalf("execute #16364 trigger-to-provider diagnostic copy without Code: %v", err)
	}
	if noCodeExecution.Status != "completed" || noCodeExecution.NodeCount < 1 || noCodeExecution.FailedRuns != 0 {
		t.Fatalf("#16364 no-Code owner probe returned incomplete evidence: %+v", noCodeExecution)
	}
	prefixJSON, err := public16364ExecutionPrefix(workflowJSON)
	if err != nil {
		t.Fatalf("prepare #16364 owner-path prefix probe: %v", err)
	}
	prefixWorkflow, _, _, err := normalizeWorkflow(prefixJSON)
	if err != nil {
		t.Fatalf("normalize #16364 owner-path prefix probe: %v", err)
	}
	prefixRuntimeJSON, err := rewriteWorkflowHTTPRequestsForVirtualAPI(prefixJSON, prefixWorkflow, profile)
	if err != nil {
		t.Fatalf("bind #16364 prefix provider to exact task companion route: %v", err)
	}
	runVariant = "schedule-config-code-to-products-read"
	prefixExecution, err := owner.ImportAndExecuteScheduled(ctx, lease, prefixRuntimeJSON)
	if err != nil {
		failedRunReceipt = public16364FailureReceipt(ctx, owner, sandboxID, companionID, lease, workflowDigest, profileDigest, runVariant, err)
		t.Fatalf("execute public #16364 prefix through Code and Sheets nodes: %v", err)
	}
	if prefixExecution.Status != "completed" || prefixExecution.NodeCount < 2 || prefixExecution.FailedRuns != 0 {
		t.Fatalf("#16364 native prefix probe returned incomplete evidence: %+v", prefixExecution)
	}
	runVariant = "full-public-graph"
	execution, err := owner.ImportAndExecuteScheduled(ctx, lease, runtimeWorkflow)
	if err != nil {
		failedRunReceipt = public16364FailureReceipt(ctx, owner, sandboxID, companionID, lease, workflowDigest, profileDigest, runVariant, err)
		t.Fatalf("execute unchanged public #16364 schedule on pinned n8n: %v", err)
	}
	if execution.Status != "completed" || execution.TriggerKind != string(TriggerScheduled) || execution.DispatchMode != "manual-one-shot" || execution.NodeCount < 9 || execution.ItemCount < 1 || execution.FailedRuns != 0 || len(execution.OutputSHA) != 64 {
		t.Fatalf("native scheduled execution receipt is incomplete: %+v", execution)
	}
	hitsJSON, err := owner.run(ctx, nil, "resource", "n8n", "runtime", "virtual-api-hits", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
	if err != nil {
		t.Fatalf("read exact content-free Sheets/Gmail route hits: %v", err)
	}
	var hits struct {
		SandboxID string `json:"sandboxId"`
		Status    string `json:"status"`
		TotalHits uint64 `json:"totalHits"`
		Routes    []struct {
			Host   string `json:"host"`
			Method string `json:"method"`
			Path   string `json:"path"`
			Count  uint64 `json:"count"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(hitsJSON, &hits); err != nil || hits.SandboxID != sandboxID || hits.Status != "available" || hits.TotalHits != 6 || len(hits.Routes) != 3 {
		t.Fatalf("scheduled workflow did not read two Sheets tabs and send two synthetic newsletters: receipt=%s err=%v", hitsJSON, err)
	}
	want := map[string]uint64{
		"sheets.googleapis.com|GET|/v4/spreadsheets/synthetic-sheet-document/values/products":   3,
		"sheets.googleapis.com|GET|/v4/spreadsheets/synthetic-sheet-document/values/recipients": 1,
		"gmail.googleapis.com|POST|/gmail/v1/users/me/messages/send":                            2,
	}
	for _, hit := range hits.Routes {
		key := hit.Host + "|" + hit.Method + "|" + hit.Path
		if count, ok := want[key]; !ok || count != hit.Count {
			t.Errorf("unexpected native route hit %q count=%d", key, hit.Count)
		} else {
			delete(want, key)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing exact native API route hits: %v", want)
	}
	if _, err := owner.run(ctx, nil, "resource", "n8n", "runtime", "virtual-api-stop", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json"); err != nil {
		t.Fatalf("stop task-private VirtualAPI companion: %v", err)
	}
	companionStopped = true
	if err := owner.Stop(ctx, lease); err != nil {
		t.Fatalf("stop isolated pinned n8n runtime: %v", err)
	}
	runtimeStopped = true
	statusJSON, err := owner.run(ctx, nil, "resource", "n8n", "runtime", "status", "--sandbox-id", sandboxID, "--json")
	var status struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(statusJSON, &status); err != nil || status.Status != "stopped" {
		t.Fatalf("owner did not confirm runtime cleanup: receipt=%s err=%v", statusJSON, err)
	}
	receipt := map[string]any{
		"kind": "native-n8n-resource-owner-scheduled-run", "workspaceSandboxGateCredit": false,
		"publicWorkflowId": "16364", "sanitizedSourceSha256": hex.EncodeToString(workflowDigest[:]),
		"resourceTaskScopeId": sandboxID, "runtimeId": lease.RuntimeID,
		"runtimeImage": "n8nio/n8n@sha256:416c29f75a0f9ac65b4339761a372af48f4a7e948de9df0d837b080d2b9d47a8", "runtimeVersion": "2.48.0",
		"profileSha256": hex.EncodeToString(profileDigest[:]), "noEgressVerified": companion.NoEgressVerified,
		"execution": map[string]any{
			"operationId": execution.OperationID, "triggerKind": execution.TriggerKind, "dispatchMode": execution.DispatchMode,
			"status": execution.Status, "outputSha256": execution.OutputSHA, "outputBytes": execution.OutputBytes,
			"nodeCount": execution.NodeCount, "runCount": execution.RunCount,
			"itemCount": execution.ItemCount, "failedRuns": execution.FailedRuns,
		},
		"routeHits": map[string]any{"status": hits.Status, "totalHits": hits.TotalHits, "routes": hits.Routes},
		"cleanup":   "VirtualAPI companion stopped; n8n runtime stopped; owner status confirmed stopped",
	}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outPath, append(encoded, '\n'), 0o600); err != nil {
		t.Fatalf("write content-free scheduled owner receipt: %v", err)
	}
}

func public16364FailureReceipt(ctx context.Context, owner N8NWorkflowRuntimeCLI, sandboxID, companionID string, lease N8NTaskRuntimeLease, workflowDigest, profileDigest [32]byte, variant string, failure error) map[string]any {
	var failureClass, diagnosticSHA256 string
	for _, rawField := range strings.Fields(failure.Error()) {
		field := strings.Trim(rawField, "()")
		if strings.HasPrefix(field, "failureClass=") {
			failureClass = strings.TrimSuffix(strings.TrimPrefix(field, "failureClass="), "):")
		}
		if strings.HasPrefix(field, "diagnosticSha256=") {
			diagnosticSHA256 = strings.TrimSuffix(strings.TrimPrefix(field, "diagnosticSha256="), "):")
		}
	}
	var routeHits any
	if hitJSON, hitErr := owner.run(ctx, nil, "resource", "n8n", "runtime", "virtual-api-hits", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json"); hitErr == nil {
		var hitReceipt struct {
			Status    string `json:"status"`
			TotalHits uint64 `json:"totalHits"`
			Routes    []struct {
				Host   string `json:"host"`
				Method string `json:"method"`
				Path   string `json:"path"`
				Count  uint64 `json:"count"`
			} `json:"routes"`
		}
		if json.Unmarshal(hitJSON, &hitReceipt) == nil {
			routeHits = map[string]any{"status": hitReceipt.Status, "totalHits": hitReceipt.TotalHits, "routes": hitReceipt.Routes}
		}
	}
	return map[string]any{
		"kind": "native-n8n-resource-owner-scheduled-failure", "workspaceSandboxGateCredit": false,
		"workflowVariant": variant, "publicWorkflowId": "16364",
		"sanitizedSourceSha256": hex.EncodeToString(workflowDigest[:]), "resourceTaskScopeId": sandboxID,
		"runtimeId": lease.RuntimeID, "runtimeVersion": "2.48.0", "profileSha256": hex.EncodeToString(profileDigest[:]),
		"failureClass": failureClass, "diagnosticSha256": diagnosticSHA256, "routeHits": routeHits, "rawEngineOutputRetained": false,
	}
}

func public16364NoCodeExecutionPrefix(raw string) (string, error) {
	var document map[string]any
	if err := json.Unmarshal([]byte(raw), &document); err != nil || document == nil {
		return "", errors.New("public workflow is not a JSON object")
	}
	nodes, ok := document["nodes"].([]any)
	if !ok {
		return "", errors.New("public workflow has no node list")
	}
	keep := map[string]bool{"schedule-lite": true, "read-products-lite": true}
	filtered := make([]any, 0, len(keep))
	for _, rawNode := range nodes {
		node, ok := rawNode.(map[string]any)
		if !ok {
			return "", errors.New("public workflow has a non-object node")
		}
		id, _ := node["id"].(string)
		if keep[id] {
			filtered = append(filtered, node)
		}
	}
	if len(filtered) != len(keep) {
		return "", errors.New("public workflow diagnostic nodes are incomplete")
	}
	document["nodes"] = filtered
	document["connections"] = map[string]any{"Daily at 08:00": map[string]any{"main": []any{[]any{map[string]any{"node": "Read Products", "type": "main", "index": 0}}}}}
	encoded, err := json.Marshal(document)
	if err != nil {
		return "", errors.New("encode public workflow trigger-to-provider diagnostic copy")
	}
	return string(encoded), nil
}

func public16364ExecutionPrefix(raw string) (string, error) {
	var document map[string]any
	if err := json.Unmarshal([]byte(raw), &document); err != nil || document == nil {
		return "", errors.New("public workflow is not a JSON object")
	}
	keepIDs := map[string]bool{"schedule-lite": true, "config-lite": true, "read-products-lite": true}
	keepNames := map[string]bool{"Daily at 08:00": true, "⚙️ Config": true, "Read Products": true}
	nodes, ok := document["nodes"].([]any)
	if !ok {
		return "", errors.New("public workflow has no node list")
	}
	filtered := make([]any, 0, len(keepIDs))
	for _, rawNode := range nodes {
		node, ok := rawNode.(map[string]any)
		if !ok {
			return "", errors.New("public workflow has a non-object node")
		}
		id, _ := node["id"].(string)
		if keepIDs[id] {
			filtered = append(filtered, node)
		}
	}
	if len(filtered) != len(keepIDs) {
		return "", errors.New("public workflow prefix nodes are incomplete")
	}
	document["nodes"] = filtered
	connections, ok := document["connections"].(map[string]any)
	if !ok {
		return "", errors.New("public workflow has no connection map")
	}
	filteredConnections := map[string]any{}
	for source, rawConnection := range connections {
		if !keepNames[source] || source == "Read Products" {
			continue
		}
		connection, ok := rawConnection.(map[string]any)
		if !ok {
			return "", errors.New("public workflow prefix connection is invalid")
		}
		filteredConnections[source] = connection
	}
	document["connections"] = filteredConnections
	encoded, err := json.Marshal(document)
	if err != nil {
		return "", errors.New("encode public workflow execution prefix")
	}
	return string(encoded), nil
}
