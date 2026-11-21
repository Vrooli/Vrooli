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

// This opt-in run exercises a public-derived webhook workflow against the
// isolated n8n resource owner and the task-private exact-route VirtualAPI. It
// does not use Workspace Sandbox and earns no Workspace Sandbox gate credit.
func TestLivePublic21013NativeOwnerWebhookRun(t *testing.T) {
	if os.Getenv("F7_LIVE_PUBLIC_21013_OWNER") != "1" {
		t.Skip("set F7_LIVE_PUBLIC_21013_OWNER=1 to run the isolated n8n webhook owner integration")
	}
	outPath := strings.TrimSpace(os.Getenv("F7_NATIVE_21013_RECEIPT_OUT"))
	if outPath == "" {
		t.Fatal("F7_NATIVE_21013_RECEIPT_OUT must name the content-free owner receipt path")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	binary := strings.TrimSpace(os.Getenv("F7_VROOLI_BIN"))
	if binary == "" {
		binary = "/home/matthalloran8/.vrooli/bin/vrooli"
	}
	owner := N8NWorkflowRuntimeCLI{Binary: binary, Environment: os.Environ}
	sandboxID := uuid.NewString()
	workflowJSON := readFixture(t, "public-workflows/21013.json")
	profileJSON := readFixture(t, "public-workflow-21013-native-profile.json")
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize sanitized public #21013 graph: %v", err)
	}
	simulator, err := NewDescriptionSimulator(profileJSON, false)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := simulator.ExportVirtualAPIProfile()
	if err != nil || len(profile.Hosts) != 2 || len(profile.Routes) != 2 {
		t.Fatalf("expected exactly two bounded Google service routes: hosts=%v routes=%d err=%v", profile.Hosts, len(profile.Routes), err)
	}
	if !workflowUsesExactVirtualAPIRoutes(workflow, profile) {
		t.Fatal("public #21013 Gmail and Sheets nodes are not bound to the exact API-description routes")
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
		t.Fatalf("start exact task-private Gmail/Sheets API companion for task %s: %v", sandboxID, err)
	}
	var companion struct {
		SandboxID          string `json:"sandboxId"`
		CompanionRuntimeID string `json:"companionRuntimeId"`
		Status             string `json:"status"`
		RouteCount         int    `json:"routeCount"`
		ProfileFingerprint string `json:"profileFingerprintSha256"`
		NoEgressVerified   bool   `json:"noEgressVerified"`
	}
	decodeCompanionErr := json.Unmarshal(companionStart, &companion)
	lease := N8NTaskRuntimeLease{SandboxID: sandboxID, RuntimeID: "n8n:" + sandboxID}
	companionID := companion.CompanionRuntimeID
	var cleanupErr error
	runtimeStopped, companionStopped := false, false
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cleanupCancel()
		if companionID != "" && !companionStopped {
			_, stopErr := owner.run(cleanupCtx, nil, "resource", "n8n", "runtime", "virtual-api-stop", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
			cleanupErr = errors.Join(cleanupErr, stopErr)
		}
		if !runtimeStopped {
			cleanupErr = errors.Join(cleanupErr, owner.Stop(cleanupCtx, lease))
		}
		if cleanupErr != nil {
			t.Errorf("isolated VirtualAPI runtime cleanup failed: %v", cleanupErr)
		}
	}()
	if decodeCompanionErr != nil || companion.SandboxID != sandboxID || companion.Status != "attached" || companion.RouteCount != 2 || companion.ProfileFingerprint != hex.EncodeToString(profileDigest[:]) || !companion.NoEgressVerified {
		t.Fatalf("task VirtualAPI did not attest exact no-egress contract: receipt=%s err=%v", companionStart, decodeCompanionErr)
	}

	event := []byte(`{"body":{"name":"Synthetic Contact","email":"lead@example.invalid","message":"Request a demo","source":"public form"}}`)
	seeded := strings.Replace(workflowJSON, `"leftValue": "={{ $json.valid }}"`, `"leftValue": "={{ false }}"`, 1)
	if seeded == workflowJSON {
		t.Fatal("could not seed a deterministic public validation defect")
	}
	failed, faultErr := owner.ImportAndDispatchWebhook(ctx, lease, seeded, event)
	if faultErr == nil || failed.HTTPStatus != 400 || failed.OperationID == "" {
		t.Fatalf("seeded validation defect did not produce a content-free HTTP 400 handoff for task %s: evidence=%+v err=%v", sandboxID, failed, faultErr)
	}
	corrected, err := owner.ImportAndDispatchWebhook(ctx, lease, workflowJSON, event)
	if err != nil || corrected.Status != "completed" || corrected.TriggerKind != string(TriggerWebhook) || corrected.HTTPStatus != 200 || corrected.ResponseBytes < 1 || len(corrected.ResponseSHA256) != 64 {
		t.Fatalf("corrected public #21013 webhook did not complete: evidence=%+v err=%v", corrected, err)
	}

	hitsJSON, err := owner.run(ctx, nil, "resource", "n8n", "runtime", "virtual-api-hits", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
	if err != nil {
		t.Fatalf("read exact content-free Gmail/Sheets route hits: %v", err)
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
		t.Fatalf("corrected native graph did not make exactly two API calls: receipt=%s err=%v", hitsJSON, err)
	}
	seen := map[string]bool{}
	for _, route := range profile.Routes {
		found := false
		for _, hit := range hits.Routes {
			if hit.Host == route.Host && hit.Method == route.Method && hit.Path == route.Path && hit.Count == 1 {
				found = true
				seen[route.Host+route.Path] = true
			}
		}
		if !found {
			t.Errorf("exact synthetic API route was not hit once: %s %s%s", route.Method, route.Host, route.Path)
		}
	}
	if len(seen) != 2 {
		t.Fatalf("native Gmail/Sheets operation coverage=%d, want both exact routes", len(seen))
	}

	if _, err := owner.run(ctx, nil, "resource", "n8n", "runtime", "virtual-api-stop", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json"); err != nil {
		t.Fatalf("stop task VirtualAPI companion: %v", err)
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
		"kind": "native-n8n-resource-owner-webhook-run", "workspaceSandboxGateCredit": false,
		"publicWorkflowId": "21013", "sanitizedSourceSha256": hex.EncodeToString(workflowDigest[:]),
		"resourceTaskScopeId": sandboxID, "runtimeId": lease.RuntimeID,
		"runtimeImage": "n8nio/n8n@sha256:416c29f75a0f9ac65b4339761a372af48f4a7e948de9df0d837b080d2b9d47a8", "runtimeVersion": "2.48.0",
		"profileSha256": hex.EncodeToString(profileDigest[:]), "noEgressVerified": companion.NoEgressVerified,
		"seededFailure":      map[string]any{"operationId": failed.OperationID, "status": "completed-with-http-error", "httpStatus": failed.HTTPStatus},
		"correctedExecution": map[string]any{"operationId": corrected.OperationID, "status": corrected.Status, "triggerKind": corrected.TriggerKind, "httpStatus": corrected.HTTPStatus, "responseSha256": corrected.ResponseSHA256, "responseBytes": corrected.ResponseBytes},
		"routeHits":          map[string]any{"status": hits.Status, "totalHits": hits.TotalHits, "routes": hits.Routes},
		"cleanup":            "VirtualAPI companion stopped; n8n runtime stopped; owner status confirmed stopped",
	}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outPath, append(encoded, '\n'), 0o600); err != nil {
		t.Fatalf("write content-free native owner receipt: %v", err)
	}
}
