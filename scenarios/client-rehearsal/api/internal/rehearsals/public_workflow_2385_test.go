package rehearsals

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"
)

func public2385SyntheticProfile(t *testing.T, fileStatus int) string {
	t.Helper()
	saved := map[string]any{"id": "wf-same", "name": "Synthetic Same", "updatedAt": "2026-01-01T00:00:00Z", "global": map[string]any{"version": 1}, "nodes": []any{map[string]any{"id": "n1", "parameters": map[string]any{"status": "open"}}}, "connections": map[string]any{}, "settings": map[string]any{}}
	currentSame := map[string]any{"id": "wf-same", "name": "Synthetic Same", "updatedAt": "2026-11-11T00:00:00Z", "global": map[string]any{"version": 2}, "nodes": []any{map[string]any{"id": "n1", "parameters": map[string]any{"status": "open"}}}, "connections": map[string]any{}, "settings": map[string]any{}}
	currentDiff := map[string]any{"id": "wf-diff", "name": "Synthetic Diff", "updatedAt": "2026-11-11T00:00:00Z", "global": map[string]any{"version": 2}, "nodes": []any{map[string]any{"id": "n1", "parameters": map[string]any{"status": "closed"}}}, "connections": map[string]any{}, "settings": map[string]any{}}
	_ = currentDiff
	binaryJSON, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	workflowExample := []any{currentSame, currentDiff}
	fileResponse := map[string]any{"binary": map[string]any{"file-from-gitlab": map[string]any{"encoding": "base64-json", "data": base64.StdEncoding.EncodeToString(binaryJSON)}}}
	service := func(id, host string, paths map[string]any) map[string]any {
		return map[string]any{"id": id, "document": map[string]any{"openapi": "3.0.3", "info": map[string]any{"title": "Synthetic " + id, "version": "1"}, "servers": []any{map[string]any{"url": "https://" + host + ".synthetic.invalid"}}, "paths": paths}}
	}
	operation := func(method, nodeType, resource, action string, status int, example any) map[string]any {
		return map[string]any{method: map[string]any{"x-n8n-node-type": nodeType, "x-n8n-resource": resource, "x-n8n-operation": action, "responses": map[string]any{jsonNumber(status): map[string]any{"content": map[string]any{"application/json": map[string]any{"example": example}}}}}}
	}
	n8nPaths := map[string]any{"/api/v1/workflows": operation("get", "n8n-nodes-base.n8n", "workflow", "getAll", 200, workflowExample)}
	gitlabPaths := map[string]any{
		"/api/v4/projects/synthetic/files":        operation("get", "n8n-nodes-base.gitlab", "file", "get", fileStatus, fileResponse),
		"/api/v4/projects/synthetic/files/create": operation("post", "n8n-nodes-base.gitlab", "file", "create", 201, map[string]any{"status": "created"}),
		"/api/v4/projects/synthetic/files/edit":   operation("put", "n8n-nodes-base.gitlab", "file", "edit", 200, map[string]any{"status": "updated"}),
	}
	profile := map[string]any{"kind": "openapi-catalog", "services": []any{service("n8n", "api", n8nPaths), service("gitlab", "gitlab", gitlabPaths)}}
	encoded, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}
func jsonNumber(value int) string { return strconv.Itoa(value) }

func TestPublic2385RunsGenericWorkflowListingLoopFileComparisonAndGitLabHandoff(t *testing.T) {
	fixture := readFixture(t, "public-workflows/2385.json")
	workflow, _, _, err := normalizeWorkflow(fixture)
	if err != nil {
		t.Fatalf("normalize public GitLab archive workflow: %v", err)
	}
	profile := public2385SyntheticProfile(t, 200)
	var manualID, loopID string
	for _, node := range workflow.Nodes {
		if node.Trigger == TriggerManual {
			manualID = node.ID
		}
		if node.LoopBatchSize > 0 {
			loopID = node.ID
		}
	}
	if manualID == "" || loopID == "" {
		t.Fatalf("public graph lacks manual trigger or batch loop: %s %s", manualID, loopID)
	}
	run, err := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC) }).CreateAndRun(context.Background(), CreateInput{Title: "Sanitized public GitLab archive rehearsal", WorkflowJSON: fixture, TriggerNodeID: manualID, SimulatorProfile: "public2385", SimulatorSourceJSON: profile})
	if err != nil {
		t.Fatalf("run original public graph through synthetic service profiles: %v", err)
	}
	if run.State != StateNeedsReview || !strings.Contains(run.HandoffMarkdown, "Human review is required") {
		t.Fatalf("expected saved synthetic handoff for operator review, got state=%s markdown=%s", run.State, run.HandoffMarkdown)
	}
	for _, operation := range []string{"n8n-nodes-base.n8n:workflow.getAll", "n8n-nodes-base.gitlab:file.get", "n8n-nodes-base.gitlab:file.edit"} {
		if !traceHasOperation(run.Trace, operation) {
			t.Fatalf("workflow did not exercise %s: %#v", operation, run.Trace)
		}
	}
	loopCount := 0
	for _, event := range run.Trace {
		if strings.Contains(event.Operation, "/Loop Over Items") {
			loopCount++
		}
	}
	if loopCount != 2 {
		t.Fatalf("expected two bounded workflow batches, got %d", loopCount)
	}
	if !traceHasNode(run.Trace, "Status same") || !traceHasNode(run.Trace, "Status diff") {
		t.Fatalf("public same and changed workflow branches were not both exercised: %v", traceNodeNames(run.Trace))
	}
	if strings.Contains(run.HandoffMarkdown, "automation@example.invalid") {
		t.Fatal("synthetic fixture author email leaked into handoff")
	}
}

func TestPublic2385GenericFallbackErrorOutputReachesErrorStatus(t *testing.T) {
	fixture := readFixture(t, "public-workflows/2385.json")
	workflow, _, _, err := normalizeWorkflow(fixture)
	if err != nil {
		t.Fatal(err)
	}
	profile := public2385SyntheticProfile(t, 404)
	simulator, err := NewDescriptionSimulator(profile, false)
	if err != nil {
		t.Fatal(err)
	}
	var manual string
	for _, node := range workflow.Nodes {
		if node.Trigger == TriggerManual {
			manual = node.ID
		}
	}
	if manual == "" {
		t.Fatal("manual trigger missing")
	}
	result, err := executeDetailedWithInput(context.Background(), workflow, manual, simulator, false, time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC), nil)

	if err != nil {
		t.Fatalf("public error-output branch should complete the synthetic run: %v", err)
	}
	if !traceHasNode(result.Trace, "Error output to normal output") || !traceHasNode(result.Trace, "Status error") {
		t.Fatalf("generic fallback did not reach public status error branch: nodes=%v", traceNodeNames(result.Trace))
	}
}

func traceNodeNames(trace []TraceEvent) []string {
	out := make([]string, 0, len(trace))
	for _, event := range trace {
		out = append(out, event.RequestSummary)
	}
	return out
}
