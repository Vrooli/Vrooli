package rehearsals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func digestString(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

// F7_LIVE_VIRTUALAPI=1 opts into one real local Workspace Sandbox + n8n owner
// journey. It is excluded from ordinary package runs because it allocates and
// cleans up a real task scope and task-owned containers.
func TestLiveTaskVirtualAPIHTTPSOwnerJourney(t *testing.T) {
	if os.Getenv("F7_LIVE_VIRTUALAPI") != "1" {
		t.Skip("set F7_LIVE_VIRTUALAPI=1 to run the one-shot real local owner journey")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	sandbox, err := NewWorkspaceSandboxProfileClient("http://127.0.0.1:15120", &http.Client{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	taskID := "f7-live-vapi-" + now.Format("20060102t150405z")
	repo := newMemoryRepository()
	runtime := N8NWorkflowRuntimeCLI{Binary: "/home/matthalloran8/.vrooli/bin/vrooli", Environment: os.Environ}
	svc := NewServiceWithNativeN8N(repo, SyntheticEngine{}, time.Now, sandbox, runtime)
	workflow := `{"name":"F7 task TLS profile check","nodes":[{"id":"start","name":"Start","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"id":"call","name":"Create synthetic page","type":"n8n-nodes-base.httpRequest","parameters":{"method":"POST","url":"https://api.notion.com/v1/pages","sendBody":true,"specifyBody":"json","jsonBody":"={\"title\":\"F7 synthetic page\"}","authentication":"none"}}],"connections":{"Start":{"main":[[{"node":"Create synthetic page"}]]}}}`
	apiDescription := `{"kind":"openapi","document":{"openapi":"3.0.3","servers":[{"url":"https://api.notion.com/v1"}],"paths":{"/pages":{"post":{"x-n8n-node-type":"n8n-nodes-base.httpRequest","x-n8n-resource":"request","x-n8n-operation":"request","requestBody":{"content":{"application/json":{"schema":{"type":"object"}}}},"responses":{"200":{"content":{"application/json":{"example":{"id":"source-page-id","object":"page","status":"created"}}}}}}}}}}`
	started := time.Now()
	result, err := svc.CreateAndRunTaskScoped(ctx, CreateInput{Title: "F7 real Workspace Sandbox TLS journey", WorkflowJSON: workflow, SimulatorSourceJSON: apiDescription}, WorkspaceSandboxTaskRequest{TaskID: taskID, Name: "f7-vapi-live"})
	if err != nil {
		t.Fatalf("real Workspace Sandbox / native n8n owner journey failed: %v", err)
	}
	if result.Rehearsal.State != StateNeedsReview || result.Lifecycle.RouteHitsStatus != "available" || result.Lifecycle.RouteHits == nil || result.Lifecycle.RouteHits.TotalHits < 1 || result.Lifecycle.N8NExecution == nil || result.Lifecycle.N8NExecution.Status != "completed" || result.Lifecycle.TeardownStatus != "succeeded" {
		t.Fatalf("real owner journey lacks execution, route, handoff or cleanup evidence: %#v", result.Lifecycle)
	}
	receipt := struct {
		StartedAt     time.Time                         `json:"startedAt"`
		ElapsedMillis int64                             `json:"elapsedMillis"`
		TaskLogicalID string                            `json:"taskLogicalId"`
		SandboxID     string                            `json:"sandboxId"`
		SourceSHA256  string                            `json:"sourceSha256"`
		State         State                             `json:"state"`
		Lifecycle     WorkspaceSandboxLifecycleEvidence `json:"lifecycle"`
		HandoffSHA256 string                            `json:"handoffSha256"`
	}{StartedAt: started.UTC(), ElapsedMillis: time.Since(started).Milliseconds(), TaskLogicalID: taskID, SandboxID: result.Lifecycle.SandboxID, SourceSHA256: result.Rehearsal.SourceSHA256, State: result.Rehearsal.State, Lifecycle: result.Lifecycle, HandoffSHA256: digestString(result.Rehearsal.HandoffMarkdown)}
	path := os.Getenv("F7_LIVE_RECEIPT_PATH")
	if path != "" {
		encoded, marshalErr := json.MarshalIndent(receipt, "", "  ")
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("real local owner receipt: sandbox=%s routes=%d hits=%d execution=%s cleanup=%s elapsed_ms=%d", result.Lifecycle.SandboxID, result.Lifecycle.OperationCount, result.Lifecycle.RouteHits.TotalHits, result.Lifecycle.N8NExecution.Status, result.Lifecycle.TeardownStatus, receipt.ElapsedMillis)
}
