package rehearsals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// This opt-in integration uses only the supported per-task n8n resource owner
// and its fixed HTTP-mock companion. It deliberately does not contact the
// shared Workspace Sandbox API. Its receipt proves a native owner execution,
// not a Workspace Sandbox acceptance gate.
func TestLivePublic22567NativeOwnerRun(t *testing.T) {
	if os.Getenv("F7_LIVE_PUBLIC_22567_OWNER") != "1" {
		t.Skip("set F7_LIVE_PUBLIC_22567_OWNER=1 to run the isolated n8n owner integration")
	}
	outPath := strings.TrimSpace(os.Getenv("F7_NATIVE_RECEIPT_OUT"))
	if outPath == "" {
		t.Fatal("F7_NATIVE_RECEIPT_OUT must name the content-free owner receipt path")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	binary := strings.TrimSpace(os.Getenv("F7_VROOLI_BIN"))
	if binary == "" {
		binary = "/home/matthalloran8/.vrooli/bin/vrooli"
	}
	owner := N8NWorkflowRuntimeCLI{Binary: binary, Environment: os.Environ}
	sandboxID := uuid.NewString()
	workflowJSON := readFixture(t, "public-workflows/22567.json")
	profileJSON := readFixture(t, "public-workflow-22567-api-profile.json")
	sourceDigest := sha256.Sum256([]byte(workflowJSON))
	_, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("sanitize public #22567: %v", err)
	}
	simulator, err := NewDescriptionSimulator(profileJSON, false)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := simulator.ExportHTTPMockProfile()
	if err != nil || len(profile.Operations) != 1 {
		t.Fatalf("derive exact public API fixture: operations=%d err=%v", len(profile.Operations), err)
	}
	op := profile.Operations[0]
	routeURL := "http://" + op.Route.Host + ":18080" + op.Route.Path
	operationAddresses := []HTTPMockOperationAddress{{Selector: op.Selector, Method: op.Route.Method, URL: routeURL}}
	faultRoute := op.Route
	faultRoute.Method = "POST"
	faultRoute.Status = 503
	faultRoute.Body = `{"error":"synthetic seeded HTTP method fault"}`
	faultAddress := HTTPMockOperationAddress{Selector: op.Selector, Method: faultRoute.Method, URL: routeURL}
	operationAddresses = append(operationAddresses, faultAddress)
	runtimeWorkflow, err := rewriteWorkflowHTTPRequestsForSandbox(workflowJSON, operationAddresses)
	if err != nil {
		t.Fatalf("rewrite only the public request to the isolated task route: %v", err)
	}

	fixtureConfig := struct {
		Name     string          `json:"name"`
		Version  string          `json:"version"`
		Fixtures []HTTPMockRoute `json:"fixtures"`
	}{Name: taskSandboxProfileName, Version: taskSandboxProfileVersion, Fixtures: []HTTPMockRoute{op.Route, faultRoute}}
	configJSON, err := json.Marshal(fixtureConfig)
	if err != nil {
		t.Fatal(err)
	}
	profileDigest := sha256.Sum256(configJSON)

	lease, err := owner.Start(ctx, sandboxID)
	if err != nil {
		t.Fatalf("start isolated pinned n8n runtime: %v", err)
	}
	var companionID string
	var cleanupErr error
	runtimeStopped := false
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cleanupCancel()
		if !runtimeStopped {
			if stopErr := owner.Stop(cleanupCtx, lease); stopErr != nil {
				cleanupErr = errors.Join(cleanupErr, fmt.Errorf("stop isolated n8n runtime: %w", stopErr))
			}
		}
		if companionID != "" {
			_, stopErr := owner.run(cleanupCtx, nil, "resource", "n8n", "runtime", "companion-stop-http-mock", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
			if stopErr != nil {
				cleanupErr = errors.Join(cleanupErr, fmt.Errorf("stop isolated HTTP-mock companion: %w", stopErr))
			}
		}
		if cleanupErr != nil {
			t.Errorf("owner cleanup did not complete: %v", cleanupErr)
		}
	}()

	startJSON, err := owner.run(ctx, configJSON, "resource", "n8n", "runtime", "companion-start-http-mock", "--sandbox-id", sandboxID, "--config-stdin", "--json")
	if err != nil {
		t.Fatalf("start exact task-local HTTP mock: %v", err)
	}
	var companion struct {
		SandboxID          string `json:"sandboxId"`
		CompanionRuntimeID string `json:"companionRuntimeId"`
		Status             string `json:"status"`
		NoEgressVerified   bool   `json:"noEgressVerified"`
		FixtureCount       int    `json:"fixtureCount"`
		ConfigSHA256       string `json:"configSha256"`
	}
	decodeErr := json.Unmarshal(startJSON, &companion)
	companionID = companion.CompanionRuntimeID
	if decodeErr != nil || companion.SandboxID != sandboxID || companion.Status != "attached" || !companion.NoEgressVerified || companion.FixtureCount != 2 || companion.ConfigSHA256 != hex.EncodeToString(profileDigest[:]) {
		t.Fatalf("owner returned incomplete or unsafe HTTP-mock start receipt: receipt=%s err=%v", startJSON, decodeErr)
	}

	seededWorkflow := readFixture(t, "repairs/22567-native-seeded.json")
	seededDigest := sha256.Sum256([]byte(seededWorkflow))
	if hex.EncodeToString(seededDigest[:]) != "67e15257c261b6ef4a6d5e007ea27bf60feb84561489549c4f2ac6e8ea6211e2" {
		t.Fatalf("native seeded HTTP method defect changed without updating its receipt contract: %x", seededDigest)
	}
	seededRuntimeWorkflow, err := rewriteWorkflowHTTPRequestsForSandbox(seededWorkflow, operationAddresses)
	if err != nil {
		t.Fatalf("rewrite seeded public defect to the exact isolated task route: %v", err)
	}
	seededFailure, seededErr := owner.ImportAndExecuteManual(ctx, lease, seededRuntimeWorkflow)
	if seededErr == nil || !strings.Contains(seededErr.Error(), "failureClass=") || !strings.Contains(seededErr.Error(), "diagnosticSha256=") || len(seededFailure.OperationID) == 0 {
		t.Fatalf("seeded native HTTP method defect did not fail with typed content-free evidence for task %s: evidence=%+v err=%v", sandboxID, seededFailure, seededErr)
	}
	failureClass := strings.SplitN(strings.SplitN(seededErr.Error(), "failureClass=", 2)[1], " ", 2)[0]
	failureDigest := strings.SplitN(strings.SplitN(seededErr.Error(), "diagnosticSha256=", 2)[1], ")", 2)[0]
	digestBytes, digestErr := hex.DecodeString(failureDigest)
	if digestErr != nil || len(digestBytes) != sha256.Size {
		t.Fatalf("seeded native failure did not preserve a SHA-256 diagnostic fingerprint: class=%s", failureClass)
	}
	execution, err := owner.ImportAndExecuteManual(ctx, lease, runtimeWorkflow)
	if err != nil {
		t.Fatalf("import and execute unchanged public #22567 graph on pinned n8n: %v", err)
	}
	if execution.Status != "completed" || execution.TriggerKind != string(TriggerManual) || execution.NodeCount < 5 || execution.ItemCount < 1 || len(execution.OutputSHA) != 64 {
		t.Fatalf("native execution receipt is incomplete: %+v", execution)
	}
	hitsJSON, err := owner.run(ctx, nil, "resource", "n8n", "runtime", "companion-route-hits", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
	if err != nil {
		t.Fatalf("read content-free exact route hits: %v", err)
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
	if err := json.Unmarshal(hitsJSON, &hits); err != nil || hits.SandboxID != sandboxID || hits.Status != "available" || hits.TotalHits != 2 || len(hits.Routes) != 2 {
		t.Fatalf("native route receipt does not prove one fault and one repaired request: receipt=%s err=%v", hitsJSON, err)
	}
	hitCounts := map[string]uint64{}
	for _, hit := range hits.Routes {
		if hit.Host != op.Route.Host || hit.Path != op.Route.Path {
			t.Errorf("route hit escaped the exact task profile: %#v", hit)
		}
		hitCounts[hit.Method] = hit.Count
	}
	if hitCounts[op.Route.Method] != 1 || hitCounts[faultRoute.Method] != 1 {
		t.Fatalf("GET repair and POST fault route hits = %v", hitCounts)
	}
	if err := owner.Stop(ctx, lease); err != nil {
		t.Fatalf("stop isolated n8n runtime: %v", err)
	}
	runtimeStopped = true
	stopJSON, err := owner.run(ctx, nil, "resource", "n8n", "runtime", "companion-stop-http-mock", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
	if err != nil || !strings.Contains(string(stopJSON), `"status":"stopped"`) {
		t.Fatalf("stop isolated HTTP-mock companion: receipt=%s err=%v", stopJSON, err)
	}
	companionID = ""
	statusJSON, err := owner.run(ctx, nil, "resource", "n8n", "runtime", "status", "--sandbox-id", sandboxID, "--json")
	var status struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(statusJSON, &status); err != nil || status.Status != "stopped" {
		t.Fatalf("owner did not confirm isolated runtime teardown: receipt=%s err=%v", statusJSON, err)
	}

	receipt := map[string]any{
		"kind":                       "native-n8n-resource-owner-run",
		"workspaceSandboxGateCredit": false,
		"publicWorkflowId":           "22567",
		"sanitizedSourceSha256":      hex.EncodeToString(sourceDigest[:]),
		"resourceTaskScopeId":        sandboxID,
		"runtimeId":                  lease.RuntimeID,
		"runtimeImage":               "n8nio/n8n@sha256:416c29f75a0f9ac65b4339761a372af48f4a7e948de9df0d837b080d2b9d47a8",
		"runtimeVersion":             "2.48.0",
		"profileSha256":              hex.EncodeToString(profileDigest[:]),
		"seededFailure": map[string]any{
			"operationId": seededFailure.OperationID, "triggerKind": string(TriggerManual),
			"failureClass": failureClass, "diagnosticSha256": failureDigest,
			"injectedRoute": map[string]any{"host": faultRoute.Host, "method": faultRoute.Method, "path": faultRoute.Path, "status": faultRoute.Status},
		},
		"execution": map[string]any{
			"operationId": execution.OperationID, "triggerKind": execution.TriggerKind,
			"status": execution.Status, "outputSha256": execution.OutputSHA,
			"outputBytes": execution.OutputBytes, "nodeCount": execution.NodeCount,
			"runCount": execution.RunCount, "itemCount": execution.ItemCount,
			"failedRuns": execution.FailedRuns,
		},
		"routeHits":        map[string]any{"status": hits.Status, "totalHits": hits.TotalHits, "routes": hits.Routes},
		"noEgressVerified": companion.NoEgressVerified,
		"cleanup":          "runtime stopped; companion stopped; owner status confirmed stopped",
	}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outPath, append(encoded, '\n'), 0o600); err != nil {
		t.Fatalf("write content-free native owner receipt: %v", err)
	}
}
