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

// This opt-in proof runs the public-derived #6338 graph on pinned n8n with
// task-private synthetic Sheets and Gmail HTTP endpoints. It does not use the
// Workspace Sandbox API and earns no Sandbox gate credit.
func TestLivePublic6338NativeOwnerParityRun(t *testing.T) {
	if os.Getenv("F7_LIVE_PUBLIC_6338_OWNER") != "1" {
		t.Skip("set F7_LIVE_PUBLIC_6338_OWNER=1 to run the isolated n8n owner integration")
	}
	outPath := strings.TrimSpace(os.Getenv("F7_NATIVE_6338_RECEIPT_OUT"))
	if outPath == "" {
		t.Fatal("F7_NATIVE_6338_RECEIPT_OUT must name the content-free receipt path")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	binary := strings.TrimSpace(os.Getenv("F7_VROOLI_BIN"))
	if binary == "" {
		binary = "/home/matthalloran8/.vrooli/bin/vrooli"
	}
	owner := N8NWorkflowRuntimeCLI{Binary: binary, Environment: os.Environ}
	sandboxID := uuid.NewString()
	publicJSON := readFixture(t, "public-workflows/6338.json")
	sourceHash := sha256.Sum256([]byte(publicJSON))
	workflow, _, _, err := normalizeWorkflow(publicJSON)
	if err != nil || len(workflow.Nodes) != 4 || workflow.TriggerKinds[0] != TriggerManual {
		t.Fatalf("unexpected sanitized public #6338 graph: nodes=%d err=%v", len(workflow.Nodes), err)
	}
	if got := hex.EncodeToString(sourceHash[:]); got != "6d960557d679630dd8ec561ac3ec552c0563990607678fb9027e83220cb08f7b" {
		t.Fatalf("public #6338 source digest changed: %s", got)
	}
	// These are the same two rows and frozen date used by the Node-RED peer run.
	rows := []map[string]string{
		{"Due Date": "2026-11-16", "Email": "reminder@synthetic.invalid", "Subject": "Synthetic due today", "Body": "Synthetic reminder body"},
		{"Due Date": "2026-11-17", "Email": "later@synthetic.invalid", "Subject": "Synthetic due tomorrow", "Body": "Synthetic later body"},
	}
	rowBody, _ := json.Marshal(rows)
	profile := struct {
		Name     string          `json:"name"`
		Version  string          `json:"version"`
		Fixtures []HTTPMockRoute `json:"fixtures"`
	}{Name: taskSandboxProfileName, Version: taskSandboxProfileVersion, Fixtures: []HTTPMockRoute{
		{Host: "sheets.synthetic.invalid", Method: "GET", Path: "/v1/reminders", Status: 200, ContentType: "application/json", Body: string(rowBody)},
		{Host: "mail.synthetic.invalid", Method: "POST", Path: "/v1/messages", Status: 200, ContentType: "application/json", Body: `{"accepted":true}`},
	}}
	configJSON, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	profileHash := sha256.Sum256(configJSON)

	lease, err := owner.Start(ctx, sandboxID)
	if err != nil {
		t.Fatalf("start pinned n8n runtime: %v", err)
	}
	var companionID string
	runtimeStopped, companionStopped := false, false
	var cleanupErr error
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cleanupCancel()
		if companionID != "" && !companionStopped {
			_, stopErr := owner.run(cleanupCtx, nil, "resource", "n8n", "runtime", "companion-stop-http-mock", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
			cleanupErr = errors.Join(cleanupErr, stopErr)
			companionStopped = stopErr == nil
		}
		if !runtimeStopped {
			cleanupErr = errors.Join(cleanupErr, owner.Stop(cleanupCtx, lease))
			runtimeStopped = cleanupErr == nil
		}
		statusJSON, statusErr := owner.run(cleanupCtx, nil, "resource", "n8n", "runtime", "status", "--sandbox-id", sandboxID, "--json")
		var status struct {
			Status string `json:"status"`
		}
		if statusErr == nil {
			statusErr = json.Unmarshal(statusJSON, &status)
		}
		if cleanupErr != nil || statusErr != nil || status.Status != "stopped" {
			t.Errorf("isolated cleanup/status incomplete: runtimeStopped=%t companionStopped=%t status=%s err=%v statusErr=%v", runtimeStopped, companionStopped, status.Status, cleanupErr, statusErr)
		}
	}()
	startJSON, err := owner.run(ctx, configJSON, "resource", "n8n", "runtime", "companion-start-http-mock", "--sandbox-id", sandboxID, "--config-stdin", "--json")
	if err != nil {
		t.Fatalf("start task-local synthetic Sheets/Gmail routes: %v", err)
	}
	var companion struct {
		CompanionRuntimeID string `json:"companionRuntimeId"`
		Status             string `json:"status"`
		NoEgressVerified   bool   `json:"noEgressVerified"`
		FixtureCount       int    `json:"fixtureCount"`
		ConfigSHA256       string `json:"configSha256"`
	}
	if err := json.Unmarshal(startJSON, &companion); err != nil || companion.Status != "attached" || !companion.NoEgressVerified || companion.FixtureCount != 2 || companion.ConfigSHA256 != hex.EncodeToString(profileHash[:]) {
		t.Fatalf("HTTP mock companion receipt invalid: err=%v", err)
	}
	companionID = companion.CompanionRuntimeID

	// Adapt only the task copy: the public Google Sheets and Gmail nodes become
	// exact no-egress HTTP calls. Preserve the public trigger and IF expression.
	var doc map[string]any
	if err := json.Unmarshal([]byte(publicJSON), &doc); err != nil {
		t.Fatal(err)
	}
	nodes, _ := doc["nodes"].([]any)
	for _, rawNode := range nodes {
		node, _ := rawNode.(map[string]any)
		if node == nil {
			continue
		}
		kind, _ := node["type"].(string)
		params, _ := node["parameters"].(map[string]any)
		switch kind {
		case "n8n-nodes-base.googleSheets":
			node["type"], node["typeVersion"] = "n8n-nodes-base.httpRequest", 4.2
			node["parameters"] = map[string]any{"authentication": "none", "method": "GET", "url": "http://sheets.synthetic.invalid:18080/v1/reminders", "options": map[string]any{}}
		case "n8n-nodes-base.gmail":
			node["type"], node["typeVersion"] = "n8n-nodes-base.httpRequest", 4.2
			_ = params
			node["parameters"] = map[string]any{"authentication": "none", "method": "POST", "url": "http://mail.synthetic.invalid:18080/v1/messages", "sendBody": true, "specifyBody": "json", "jsonBody": "={{ {to: $json.Email, subject: $json.Subject, body: $json.Body} }}", "options": map[string]any{}}
		}
	}
	taskJSON, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	execution, err := owner.ImportAndExecuteManual(ctx, lease, string(taskJSON))
	if err != nil {
		t.Fatalf("execute public #6338 task copy natively: %v", err)
	}
	if execution.Status != "completed" || execution.FailedRuns != 0 || execution.TriggerKind != string(TriggerManual) || execution.NodeCount < 4 {
		t.Fatalf("native execution did not complete expected public graph: %+v", execution)
	}
	hitsJSON, err := owner.run(ctx, nil, "resource", "n8n", "runtime", "companion-route-hits", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
	if err != nil {
		t.Fatalf("read content-free route hits: %v", err)
	}
	var hits struct {
		Status    string `json:"status"`
		TotalHits uint64 `json:"totalHits"`
		Routes    []struct {
			Host   string `json:"host"`
			Method string `json:"method"`
			Path   string `json:"path"`
			Count  uint64 `json:"count"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(hitsJSON, &hits); err != nil || hits.Status != "available" || hits.TotalHits != 2 || len(hits.Routes) != 2 {
		t.Fatalf("native route hits do not show one Sheets read and one Gmail sink: err=%v", err)
	}
	seen := map[string]uint64{}
	for _, route := range hits.Routes {
		seen[route.Host+"|"+route.Method+"|"+route.Path] = route.Count
	}
	if seen["sheets.synthetic.invalid|GET|/v1/reminders"] != 1 || seen["mail.synthetic.invalid|POST|/v1/messages"] != 1 {
		t.Fatalf("unexpected synthetic route hits: %s", hitsJSON)
	}
	if err := owner.Stop(ctx, lease); err != nil {
		t.Fatalf("stop n8n runtime: %v", err)
	}
	runtimeStopped = true
	_, err = owner.run(ctx, nil, "resource", "n8n", "runtime", "companion-stop-http-mock", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
	if err != nil {
		t.Fatalf("stop HTTP mock: %v", err)
	}
	companionStopped, companionID = true, ""
	receipt := map[string]any{"kind": "native-n8n-resource-owner-cross-engine-comparison", "workspaceSandboxGateCredit": false, "publicWorkflowId": "6338", "sourceSha256": hex.EncodeToString(sourceHash[:]), "taskCopySha256": fmt.Sprintf("%x", sha256.Sum256(taskJSON)), "inputRowsSha256": fmt.Sprintf("%x", sha256.Sum256(rowBody)), "frozenDate": "2026-11-16", "taskScopeId": sandboxID, "runtimeId": lease.RuntimeID, "runtimeVersion": "2.48.0", "profileSha256": hex.EncodeToString(profileHash[:]), "noEgressVerified": companion.NoEgressVerified, "execution": map[string]any{"status": execution.Status, "triggerKind": execution.TriggerKind, "nodeCount": execution.NodeCount, "runCount": execution.RunCount, "itemCount": execution.ItemCount, "failedRuns": execution.FailedRuns, "outputSha256": execution.OutputSHA, "outputBytes": execution.OutputBytes, "nodeOutputItemCounts": execution.NodeOutputItemCounts}, "routeHits": hits, "simulatedServices": []string{"Google Sheets rows", "Gmail sink"}, "externalEffects": false, "cleanup": "runtime stopped; companion stopped; owner status verified stopped"}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outPath, append(encoded, '\n'), 0o600); err != nil {
		t.Fatalf("write content-free receipt: %v", err)
	}
}
