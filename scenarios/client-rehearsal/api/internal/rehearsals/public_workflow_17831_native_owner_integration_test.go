package rehearsals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// This opt-in owner proof exercises the public webhook project-intake graph
// against task-private Sheets and Gmail fixtures. The manual demonstration
// branch and provider integrations are adapted only in the task copy.
func TestLivePublic17831NativeWebhookOwnerRepairRun(t *testing.T) {
	if os.Getenv("F7_LIVE_PUBLIC_17831_OWNER") != "1" {
		t.Skip("set F7_LIVE_PUBLIC_17831_OWNER=1 to run the isolated n8n owner integration")
	}
	outPath := strings.TrimSpace(os.Getenv("F7_NATIVE_17831_RECEIPT_OUT"))
	if outPath == "" {
		t.Fatal("F7_NATIVE_17831_RECEIPT_OUT must name the content-free receipt path")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	binary := strings.TrimSpace(os.Getenv("F7_VROOLI_BIN"))
	if binary == "" {
		binary = "/home/matthalloran8/.vrooli/bin/vrooli"
	}
	owner := N8NWorkflowRuntimeCLI{Binary: binary, Environment: os.Environ}
	sandboxID := uuid.NewString()
	publicJSON := readFixture(t, "public-workflows/17831.json")
	profileJSON := readFixture(t, "public-workflow-17831-profile.json")
	sourceHash := sha256.Sum256([]byte(publicJSON))
	if hex.EncodeToString(sourceHash[:]) != "8184b1c054300842c0dd144eaa137a237b9c017badb537ebc249614bb1eff56a" {
		t.Fatalf("sanitized public #17831 digest changed: %x", sourceHash)
	}
	workflow, _, _, err := normalizeWorkflow(publicJSON)
	if err != nil || len(workflow.Nodes) != 9 || len(workflow.EntryNodeIDs) != 2 || !oneOfFold(string(workflow.TriggerKinds[0]), string(TriggerWebhook), string(TriggerManual)) {
		t.Fatalf("public #17831 trigger contract changed: nodes=%d entries=%d triggers=%v err=%v", len(workflow.Nodes), len(workflow.EntryNodeIDs), workflow.TriggerKinds, err)
	}
	simulator, err := NewDescriptionSimulator(profileJSON, false)
	if err != nil {
		t.Fatalf("load public #17831 API descriptions: %v", err)
	}
	profile, err := simulator.ExportHTTPMockProfile()
	if err != nil || len(profile.Operations) != 2 {
		t.Fatalf("derive exact Sheets/Gmail mock profile: operations=%d err=%v", len(profile.Operations), err)
	}
	fixtures := make([]HTTPMockRoute, 0, len(profile.Operations)+1)
	addresses := make([]HTTPMockOperationAddress, 0, len(profile.Operations)+1)
	var sheetsURL, mailURL string
	var sheetsRoute, mailRoute HTTPMockRoute
	for _, operation := range profile.Operations {
		route := operation.Route
		fixtures = append(fixtures, route)
		addressURL, addressErr := syntheticHTTPRouteURL(route, "http", "18080")
		if addressErr != nil {
			t.Fatalf("compile exact #17831 companion route: %v", addressErr)
		}
		addresses = append(addresses, HTTPMockOperationAddress{Selector: operation.Selector, Method: route.Method, URL: addressURL})
		switch {
		case operation.Selector.NodeType == "n8n-nodes-base.googleSheets" && operation.Selector.Operation == "append":
			sheetsURL, addressErr = syntheticHTTPRouteURL(route, "https", "")
			sheetsRoute = route
		case operation.Selector.NodeType == "n8n-nodes-base.gmail" && operation.Selector.Operation == "send":
			mailURL, addressErr = syntheticHTTPRouteURL(route, "https", "")
			mailRoute = route
		}
		if addressErr != nil {
			t.Fatalf("compile #17831 task workflow URL: %v", addressErr)
		}
	}
	if sheetsURL == "" || mailURL == "" {
		t.Fatalf("synthetic profile does not bind both public destinations: sheets=%t mail=%t", sheetsURL != "", mailURL != "")
	}
	faultRoute := mailRoute
	faultRoute.Method = http.MethodGet
	faultRoute.Status = http.StatusServiceUnavailable
	faultRoute.Body = `{"error":"synthetic seeded method defect"}`
	fixtures = append(fixtures, faultRoute)
	faultURL, err := syntheticHTTPRouteURL(faultRoute, "http", "18080")
	if err != nil {
		t.Fatalf("compile seeded #17831 route: %v", err)
	}
	addresses = append(addresses, HTTPMockOperationAddress{Method: faultRoute.Method, URL: faultURL})
	configJSON, err := json.Marshal(struct {
		Name     string          `json:"name"`
		Version  string          `json:"version"`
		Fixtures []HTTPMockRoute `json:"fixtures"`
	}{Name: taskSandboxProfileName, Version: taskSandboxProfileVersion, Fixtures: fixtures})
	if err != nil {
		t.Fatal(err)
	}
	profileHash := sha256.Sum256(configJSON)
	buildTaskCopy := func(mailMethod string) (string, error) {
		var doc map[string]any
		if err := json.Unmarshal([]byte(publicJSON), &doc); err != nil {
			return "", err
		}
		kept := make([]any, 0)
		for _, raw := range doc["nodes"].([]any) {
			node := raw.(map[string]any)
			if node["name"] == "Test manuel" || node["name"] == "Données de démonstration" {
				continue
			}
			switch node["type"] {
			case "n8n-nodes-base.googleSheets":
				node["type"], node["typeVersion"] = "n8n-nodes-base.httpRequest", 4.2
				node["parameters"] = map[string]any{"authentication": "none", "method": http.MethodPost, "url": sheetsURL, "sendBody": true, "specifyBody": "json", "jsonBody": "={{ $json }}", "options": map[string]any{}}
			case "n8n-nodes-base.gmail":
				node["type"], node["typeVersion"] = "n8n-nodes-base.httpRequest", 4.2
				node["parameters"] = map[string]any{"authentication": "none", "method": mailMethod, "url": mailURL, "sendBody": true, "specifyBody": "json", "jsonBody": `={{ {to: $('Normaliser et valider').item.json.email, project: $('Normaliser et valider').item.json.project_name, project_id: $('Normaliser et valider').item.json.project_id} }}`, "options": map[string]any{}}
			}
			kept = append(kept, node)
		}
		doc["nodes"] = kept
		connections := doc["connections"].(map[string]any)
		delete(connections, "Test manuel")
		delete(connections, "Données de démonstration")
		encoded, err := json.Marshal(doc)
		if err != nil {
			return "", err
		}
		return rewriteWorkflowHTTPRequestsForSandbox(string(encoded), addresses)
	}
	seededWorkflow, err := buildTaskCopy(http.MethodGet)
	if err != nil {
		t.Fatalf("build seeded public #17831 webhook copy: %v", err)
	}
	correctedWorkflow, err := buildTaskCopy(http.MethodPost)
	if err != nil {
		t.Fatalf("build repaired public #17831 webhook copy: %v", err)
	}
	eventJSON, err := json.Marshal(map[string]any{"body": map[string]any{
		"client_name": "Synthetic Client", "email": "client@synthetic.invalid", "project_name": "Synthetic Workflow Project",
		"project_type": "Automation", "phone": "+10000000000", "deadline": "2026-11-30", "message": "Synthetic project intake.",
	}, "source": "synthetic_webhook"})
	if err != nil {
		t.Fatal(err)
	}
	eventHash := sha256.Sum256(eventJSON)
	lease, err := owner.Start(ctx, sandboxID)
	if err != nil {
		t.Fatalf("start pinned n8n runtime: %v", err)
	}
	var companionID string
	runtimeStopped, companionStopped := false, false
	var cleanupErr error
	phase := "companion-start"
	receipt := map[string]any{
		"kind": "native-n8n-resource-owner-webhook-defect-repair-attempt", "workspaceSandboxGateCredit": false,
		"publicWorkflowId": "17831", "sourceSha256": hex.EncodeToString(sourceHash[:]),
		"taskScopeId": sandboxID, "runtimeId": lease.RuntimeID, "runtimeVersion": "2.48.0",
		"profileSha256": hex.EncodeToString(profileHash[:]), "fixtureCount": len(fixtures),
		"eventSha256": hex.EncodeToString(eventHash[:]), "eventBytes": len(eventJSON), "externalEffects": false,
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cleanupCancel()
		if companionID != "" {
			hitsJSON, hitsErr := owner.run(cleanupCtx, nil, "resource", "n8n", "runtime", "companion-route-hits", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
			var hits HTTPMockRouteHitsReceipt
			if hitsErr == nil && json.Unmarshal(hitsJSON, &hits) == nil && hits.Status == "available" {
				receipt["routeHitsAtTermination"] = hits
			} else {
				receipt["routeHitsReadStatus"] = "unavailable"
			}
		}
		if companionID != "" && !companionStopped {
			_, stopErr := owner.run(cleanupCtx, nil, "resource", "n8n", "runtime", "companion-stop-http-mock", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
			cleanupErr = errors.Join(cleanupErr, stopErr)
			companionStopped = stopErr == nil
		}
		if !runtimeStopped {
			stopErr := owner.Stop(cleanupCtx, lease)
			cleanupErr = errors.Join(cleanupErr, stopErr)
			runtimeStopped = stopErr == nil
		}
		statusJSON, statusErr := owner.run(cleanupCtx, nil, "resource", "n8n", "runtime", "status", "--sandbox-id", sandboxID, "--json")
		var status struct {
			Status string `json:"status"`
		}
		if statusErr == nil {
			statusErr = json.Unmarshal(statusJSON, &status)
		}
		receipt["phaseAtTermination"] = phase
		receipt["cleanup"] = map[string]any{"runtimeStopCommandSucceeded": runtimeStopped, "companionStopCommandSucceeded": companionStopped, "ownerStatus": status.Status, "ownerStatusReadSucceeded": statusErr == nil}
		receipt["cleanupVerified"] = cleanupErr == nil && statusErr == nil && status.Status == "stopped"
		encoded, encodeErr := json.MarshalIndent(receipt, "", "  ")
		if encodeErr == nil {
			writeErr := os.WriteFile(outPath, append(encoded, '\n'), 0o600)
			if writeErr == nil {
				writeErr = os.Chmod(outPath, 0o600)
			}
			if writeErr != nil {
				t.Errorf("write content-free public #17831 receipt: %v", writeErr)
			}
		} else {
			t.Errorf("encode content-free public #17831 receipt: %v", encodeErr)
		}
		if cleanupErr != nil || statusErr != nil || status.Status != "stopped" {
			t.Errorf("isolated owner cleanup/status incomplete: runtimeStopped=%t companionStopped=%t status=%s", runtimeStopped, companionStopped, status.Status)
		}
	}()
	startJSON, err := owner.run(ctx, configJSON, "resource", "n8n", "runtime", "companion-start-http-mock", "--sandbox-id", sandboxID, "--config-stdin", "--json")
	if err != nil {
		t.Fatalf("start task-local Sheets/Gmail mock: %v", err)
	}
	var companion struct {
		CompanionRuntimeID string `json:"companionRuntimeId"`
		Status             string `json:"status"`
		NoEgressVerified   bool   `json:"noEgressVerified"`
		FixtureCount       int    `json:"fixtureCount"`
		ConfigSHA256       string `json:"configSha256"`
	}
	if err := json.Unmarshal(startJSON, &companion); err != nil || companion.Status != "attached" || !companion.NoEgressVerified || companion.FixtureCount != len(fixtures) || companion.ConfigSHA256 != hex.EncodeToString(profileHash[:]) {
		t.Fatalf("task mock receipt invalid: err=%v", err)
	}
	companionID = companion.CompanionRuntimeID
	receipt["noEgressVerified"] = companion.NoEgressVerified
	receipt["companionRuntimeId"] = companion.CompanionRuntimeID
	phase = "seeded-webhook-execution"
	seededHash := sha256.Sum256([]byte(seededWorkflow))
	seededStarted := time.Now()
	seeded, seededErr := owner.ImportAndDispatchWebhook(ctx, lease, seededWorkflow, eventJSON)
	seededElapsed := time.Since(seededStarted).Milliseconds()
	if seededErr == nil {
		t.Fatalf("seeded public #17831 Gmail method defect unexpectedly succeeded: %+v", seeded)
	}
	receipt["seededFailure"] = map[string]any{"taskCopySha256": hex.EncodeToString(seededHash[:]), "elapsedMs": seededElapsed, "triggerKind": seeded.TriggerKind, "status": seeded.Status, "httpStatus": seeded.HTTPStatus, "publicationState": seeded.PublicationState, "published": seeded.Published, "cleanupRequired": seeded.CleanupRequired, "responseSha256": seeded.ResponseSHA256, "responseBytes": seeded.ResponseBytes, "diagnostic": contentFreeN8NFailureEvidence(seededErr)}
	seedHitsJSON, err := owner.run(ctx, nil, "resource", "n8n", "runtime", "companion-route-hits", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
	if err != nil {
		t.Fatalf("read seeded failure route hits: %v", err)
	}
	var seedHits HTTPMockRouteHitsReceipt
	if err := json.Unmarshal(seedHitsJSON, &seedHits); err != nil || seedHits.Status != "available" {
		t.Fatalf("seeded failure route evidence is invalid: err=%v", err)
	}
	receipt["seededFailure"].(map[string]any)["routeHits"] = seedHits
	seededFaultRouteHits := exactHTTPMockRouteHitCount(seedHits, faultRoute)
	receipt["seededFailure"].(map[string]any)["exactFaultRouteHits"] = seededFaultRouteHits
	if !seededNativeFaultRouteSatisfied(seedHits, faultRoute) {
		t.Fatalf("seeded webhook error did not reach the exact Gmail fault route; routeHits=%d", seededFaultRouteHits)
	}
	phase = "repaired-webhook-execution"
	correctedHash := sha256.Sum256([]byte(correctedWorkflow))
	correctedStarted := time.Now()
	execution, correctedErr := owner.ImportAndDispatchWebhook(ctx, lease, correctedWorkflow, eventJSON)
	correctedElapsed := time.Since(correctedStarted).Milliseconds()
	if correctedErr != nil {
		receipt["repairedExecutionFailure"] = map[string]any{"taskCopySha256": hex.EncodeToString(correctedHash[:]), "elapsedMs": correctedElapsed, "status": execution.Status, "httpStatus": execution.HTTPStatus, "publicationState": execution.PublicationState, "published": execution.Published, "cleanupRequired": execution.CleanupRequired, "responseSha256": execution.ResponseSHA256, "responseBytes": execution.ResponseBytes, "diagnostic": contentFreeN8NFailureEvidence(correctedErr)}
		t.Fatalf("execute repaired public #17831 webhook graph on pinned n8n: %v", correctedErr)
	}
	if execution.Status != "completed" || execution.TriggerKind != string(TriggerWebhook) || execution.HTTPStatus < 200 || execution.HTTPStatus >= 300 || len(execution.ResponseSHA256) != 64 {
		t.Fatalf("native repaired webhook execution is incomplete: %+v", execution)
	}
	finalHitsJSON, err := owner.run(ctx, nil, "resource", "n8n", "runtime", "companion-route-hits", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
	if err != nil {
		t.Fatalf("read repaired route hits: %v", err)
	}
	var finalHits HTTPMockRouteHitsReceipt
	if err := json.Unmarshal(finalHitsJSON, &finalHits); err != nil || finalHits.Status != "available" || finalHits.TotalHits < 4 {
		t.Fatalf("repaired public workflow did not reach both synthetic service routes: err=%v", err)
	}
	if exactHTTPMockRouteHitCount(finalHits, faultRoute) != 1 || exactHTTPMockRouteHitCount(finalHits, sheetsRoute) != 2 || exactHTTPMockRouteHitCount(finalHits, mailRoute) != 1 {
		t.Fatalf("repaired webhook route counts differ from one seeded Gmail fault plus two Sheets and one Gmail repaired calls: %+v", finalHits.Routes)
	}
	receipt["repairedExecution"] = map[string]any{"taskCopySha256": hex.EncodeToString(correctedHash[:]), "status": execution.Status, "elapsedMs": correctedElapsed, "triggerKind": execution.TriggerKind, "httpStatus": execution.HTTPStatus, "eventSha256": execution.InputSHA256, "eventBytes": execution.InputBytes, "responseSha256": execution.ResponseSHA256, "responseBytes": execution.ResponseBytes}
	receipt["finalRouteHits"] = finalHits
	receipt["simulatedServices"] = []string{"Google Sheets project registry", "Gmail acknowledgement sink"}
	if err := owner.Stop(ctx, lease); err != nil {
		t.Fatalf("stop isolated n8n runtime: %v", err)
	}
	runtimeStopped = true
	_, err = owner.run(ctx, nil, "resource", "n8n", "runtime", "companion-stop-http-mock", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
	if err != nil {
		t.Fatalf("stop task HTTP mock: %v", err)
	}
	companionStopped, companionID = true, ""
	phase = "completed"
}
