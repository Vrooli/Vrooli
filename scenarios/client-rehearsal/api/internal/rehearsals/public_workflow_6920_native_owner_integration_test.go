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

// This opt-in proof compares the public-derived #6920 notification semantics
// on native n8n against its isolated Node-RED peer. The native Error Trigger
// context is represented by an explicit synthetic event value; no mail is sent.
func TestLivePublic6920NativeOwnerParityRun(t *testing.T) {
	if os.Getenv("F7_LIVE_PUBLIC_6920_OWNER") != "1" {
		t.Skip("set F7_LIVE_PUBLIC_6920_OWNER=1 to run the isolated n8n owner integration")
	}
	outPath := strings.TrimSpace(os.Getenv("F7_NATIVE_6920_RECEIPT_OUT"))
	if outPath == "" {
		t.Fatal("F7_NATIVE_6920_RECEIPT_OUT must name the content-free receipt path")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	binary := strings.TrimSpace(os.Getenv("F7_VROOLI_BIN"))
	if binary == "" {
		binary = "/home/matthalloran8/.vrooli/bin/vrooli"
	}
	owner := N8NWorkflowRuntimeCLI{Binary: binary, Environment: os.Environ}
	sandboxID := uuid.NewString()
	publicJSON := readFixture(t, "public-workflows/6920.json")
	sourceHash := sha256.Sum256([]byte(publicJSON))
	if hex.EncodeToString(sourceHash[:]) != "5e095fad1bba7bb2868fda9a03c429cdd0aee97d7b347cdb94d3683b2dfb9e53" {
		t.Fatalf("public #6920 source digest changed: %x", sourceHash)
	}
	config := struct {
		Name     string          `json:"name"`
		Version  string          `json:"version"`
		Fixtures []HTTPMockRoute `json:"fixtures"`
	}{Name: taskSandboxProfileName, Version: taskSandboxProfileVersion, Fixtures: []HTTPMockRoute{{Host: "mail.synthetic.invalid", Method: "POST", Path: "/v1/notify", Status: 200, ContentType: "application/json", Body: `{"accepted":true}`}}}
	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	configHash := sha256.Sum256(configJSON)
	lease, err := owner.Start(ctx, sandboxID)
	if err != nil {
		t.Fatalf("start pinned n8n runtime: %v", err)
	}
	var companionID string
	runtimeStopped, companionStopped := false, false
	var cleanupErr error
	var taskHash [32]byte
	var failureEvidence map[string]any
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cleanupCancel()
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
		if cleanupErr != nil || statusErr != nil || status.Status != "stopped" {
			t.Errorf("isolated cleanup/status incomplete: runtimeStopped=%t companionStopped=%t status=%s err=%v statusErr=%v", runtimeStopped, companionStopped, status.Status, cleanupErr, statusErr)
		}
		if failureEvidence != nil {
			receipt := map[string]any{"kind": "native-n8n-resource-owner-cross-engine-comparison", "outcome": "failed", "workspaceSandboxGateCredit": false, "publicWorkflowId": "6920", "sourceSha256": hex.EncodeToString(sourceHash[:]), "taskCopySha256": hex.EncodeToString(taskHash[:]), "taskScopeId": sandboxID, "runtimeId": lease.RuntimeID, "runtimeVersion": "2.48.0", "failure": failureEvidence, "routeHitStatus": "unavailable", "executionAttempted": false, "externalEffects": false, "cleanup": map[string]any{"runtimeStopped": runtimeStopped, "companionStopped": companionStopped, "status": status.Status, "statusVerified": statusErr == nil && status.Status == "stopped", "cleanupError": cleanupErr != nil}}
			encoded, encodeErr := json.MarshalIndent(receipt, "", "  ")
			if encodeErr == nil {
				encodeErr = os.WriteFile(outPath, append(encoded, '\n'), 0o600)
			}
			if encodeErr != nil {
				t.Errorf("write content-free failure receipt: %v", encodeErr)
			}
		}
	}()
	startJSON, err := owner.run(ctx, configJSON, "resource", "n8n", "runtime", "companion-start-http-mock", "--sandbox-id", sandboxID, "--config-stdin", "--json")
	if err != nil {
		t.Fatalf("start synthetic notification sink: %v", err)
	}
	var companion struct {
		CompanionRuntimeID string `json:"companionRuntimeId"`
		Status             string `json:"status"`
		NoEgressVerified   bool   `json:"noEgressVerified"`
		FixtureCount       int    `json:"fixtureCount"`
		ConfigSHA256       string `json:"configSha256"`
	}
	if err := json.Unmarshal(startJSON, &companion); err != nil || companion.Status != "attached" || !companion.NoEgressVerified || companion.FixtureCount != 1 || companion.ConfigSHA256 != hex.EncodeToString(configHash[:]) {
		t.Fatalf("mock companion receipt invalid: err=%v", err)
	}
	companionID = companion.CompanionRuntimeID

	// The public Error Trigger is replaced only in the task copy by the owner
	// manual trigger. A fixed synthetic event string preserves its output value.
	var doc map[string]any
	if err := json.Unmarshal([]byte(publicJSON), &doc); err != nil {
		t.Fatal(err)
	}
	// The sanitized public export omits settings; the successful #6338 native
	// import carries an explicit execution order. Add that documented n8n field
	// to the task copy while retaining the public graph and its connections.
	doc["settings"] = map[string]any{"executionOrder": "v1"}
	nodes, _ := doc["nodes"].([]any)
	for _, raw := range nodes {
		node, _ := raw.(map[string]any)
		if node == nil {
			continue
		}
		switch node["type"] {
		case "n8n-nodes-base.errorTrigger":
			node["type"], node["typeVersion"], node["parameters"] = "n8n-nodes-base.manualTrigger", 1, map[string]any{}
			node["position"] = []int{240, 200}
		case "n8n-nodes-base.gmail":
			node["type"], node["typeVersion"] = "n8n-nodes-base.httpRequest", 4.2
			node["position"] = []int{480, 200}
			node["parameters"] = map[string]any{"authentication": "none", "method": "POST", "url": "http://mail.synthetic.invalid:18080/v1/notify", "sendBody": true, "specifyBody": "json", "jsonBody": "={{ {to: 'workflow-owner@example.invalid', subject: 'Synthetic workflow failure', message: 'Synthetic failure notification for synthetic failure fixture'} }}", "options": map[string]any{}}
		}
	}
	taskJSON, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	taskHash = sha256.Sum256(taskJSON)
	execution, err := owner.ImportAndExecuteManual(ctx, lease, string(taskJSON))
	if err != nil {
		text := err.Error()
		phase := "native-import-or-execution"
		if strings.Contains(text, "n8n workflow import failed") {
			phase = "native-import"
		}
		safe := map[string]any{"phase": phase}
		if marker := "failureClass="; strings.Contains(text, marker) {
			class := strings.SplitN(strings.SplitN(text, marker, 2)[1], " ", 2)[0]
			if class == "cli-stderr-diagnostic" {
				safe["failureClass"] = class
			}
		}
		if marker := "diagnosticSha256="; strings.Contains(text, marker) {
			digest := strings.SplitN(strings.SplitN(text, marker, 2)[1], " ", 2)[0]
			if len(digest) == 64 {
				if _, decodeErr := hex.DecodeString(digest); decodeErr == nil {
					safe["diagnosticSha256"] = digest
				}
			}
		}
		failureEvidence = safe
		t.Fatalf("execute public #6920 task copy on n8n: %v", err)
	}
	if execution.Status != "completed" || execution.FailedRuns != 0 || execution.TriggerKind != string(TriggerManual) || execution.NodeCount < 2 {
		t.Fatalf("native notification execution incomplete: %+v", execution)
	}
	hitsJSON, err := owner.run(ctx, nil, "resource", "n8n", "runtime", "companion-route-hits", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
	if err != nil {
		t.Fatalf("read synthetic notification route hits: %v", err)
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
	if err := json.Unmarshal(hitsJSON, &hits); err != nil || hits.Status != "available" || hits.TotalHits != 1 || len(hits.Routes) != 1 || hits.Routes[0].Host != "mail.synthetic.invalid" || hits.Routes[0].Method != "POST" || hits.Routes[0].Path != "/v1/notify" || hits.Routes[0].Count != 1 {
		t.Fatalf("expected one synthetic notification sink request: err=%v", err)
	}
	if err := owner.Stop(ctx, lease); err != nil {
		t.Fatalf("stop n8n runtime: %v", err)
	}
	runtimeStopped = true
	_, err = owner.run(ctx, nil, "resource", "n8n", "runtime", "companion-stop-http-mock", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
	if err != nil {
		t.Fatalf("stop HTTP mock: %v", err)
	}
	companionStopped = true
	companionID = ""
	receipt := map[string]any{"kind": "native-n8n-resource-owner-cross-engine-comparison", "workspaceSandboxGateCredit": false, "publicWorkflowId": "6920", "sourceSha256": hex.EncodeToString(sourceHash[:]), "taskCopySha256": hex.EncodeToString(taskHash[:]), "frozenSyntheticErrorMessage": "synthetic failure fixture", "taskScopeId": sandboxID, "runtimeId": lease.RuntimeID, "runtimeVersion": "2.48.0", "profileSha256": hex.EncodeToString(configHash[:]), "noEgressVerified": companion.NoEgressVerified, "execution": map[string]any{"status": execution.Status, "triggerKind": execution.TriggerKind, "nodeCount": execution.NodeCount, "runCount": execution.RunCount, "itemCount": execution.ItemCount, "failedRuns": execution.FailedRuns, "outputSha256": execution.OutputSHA, "outputBytes": execution.OutputBytes}, "routeHits": hits, "externalEffects": false, "incompatibilities": []string{"Native Error Trigger context was replaced by an explicit synthetic error event/manual trigger in this owner run.", "Gmail delivery/auth replaced by isolated content-free HTTP sink."}, "cleanup": "runtime stopped; companion stopped; owner status verified stopped"}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outPath, append(encoded, '\n'), 0o600); err != nil {
		t.Fatalf("write content-free receipt: %v", err)
	}
}
