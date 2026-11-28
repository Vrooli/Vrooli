package rehearsals

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func TestPublic16364ScheduledNewsletterBindsExactNativeSheetsAndGmailRoutes(t *testing.T) {
	raw := readFixture(t, "public-workflows/16364.json")
	workflow, _, _, err := normalizeWorkflow(raw)
	if err != nil {
		t.Fatalf("normalize sanitized public #16364: %v", err)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-16364-native-profile.json"), false)
	if err != nil {
		t.Fatalf("load bounded Gmail/Sheets native API profile: %v", err)
	}
	profile, err := simulator.ExportVirtualAPIProfile()
	if err != nil || len(profile.Hosts) != 2 || len(profile.Routes) != 3 {
		t.Fatalf("unexpected exact schedule API profile: hosts=%v routes=%d err=%v", profile.Hosts, len(profile.Routes), err)
	}
	if !slices.Contains(profile.Hosts, "sheets.googleapis.com") || !slices.Contains(profile.Hosts, "gmail.googleapis.com") {
		t.Fatalf("profile hosts are not the exact provider endpoints: %v", profile.Hosts)
	}
	if !workflowUsesExactVirtualAPIRoutes(workflow, profile) {
		t.Fatal("public scheduled workflow nodes are not bound to exact API-description routes")
	}
	runtimeJSON, err := rewriteWorkflowHTTPRequestsForVirtualAPI(raw, workflow, profile)
	if err != nil {
		t.Fatalf("adapt provider nodes to task-local API aliases: %v", err)
	}
	var runtimeDocument struct {
		Nodes []struct {
			ID          string         `json:"id"`
			Type        string         `json:"type"`
			Parameters  map[string]any `json:"parameters"`
			Credentials map[string]any `json:"credentials"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(runtimeJSON), &runtimeDocument); err != nil {
		t.Fatalf("decode task-runtime workflow copy: %v", err)
	}
	wantTaskRoutes := map[string]string{
		"read-products-lite":   "GET https://sheets.googleapis.com/v4/spreadsheets/synthetic-sheet-document/values/products",
		"read-recipients-lite": "GET https://sheets.googleapis.com/v4/spreadsheets/synthetic-sheet-document/values/recipients",
		"send-lite":            "POST https://gmail.googleapis.com/gmail/v1/users/me/messages/send",
	}
	for _, node := range runtimeDocument.Nodes {
		want, ok := wantTaskRoutes[node.ID]
		if !ok {
			continue
		}
		if node.Type != "n8n-nodes-base.httpRequest" || node.Parameters["method"] != strings.Fields(want)[0] || node.Parameters["url"] != strings.Join(strings.Fields(want)[1:], " ") || node.Parameters["authentication"] != "none" || node.Credentials != nil {
			t.Errorf("provider node %q was not adapted to its no-auth exact companion route: type=%s params=%v", node.ID, node.Type, node.Parameters)
		}
		delete(wantTaskRoutes, node.ID)
	}
	if len(wantTaskRoutes) != 0 {
		t.Errorf("task runtime adaptation omitted provider nodes: %v", wantTaskRoutes)
	}
	if !nativeScheduledWorkflowSupported(workflow) {
		t.Fatal("public schedule workflow with bounded Code subset is outside the reusable native runner")
	}
	oneShot, err := rewriteScheduledWorkflowForOneShotDispatch(raw, workflow)
	if err != nil {
		t.Fatalf("prepare deterministic one-shot runtime copy: %v", err)
	}
	if !strings.Contains(raw, `n8n-nodes-base.scheduleTrigger`) || !strings.Contains(oneShot, `n8n-nodes-base.manualTrigger`) {
		t.Fatal("one-shot dispatch must adapt only the runtime copy and retain the public schedule source")
	}
	triggerCount := 0
	for _, node := range workflow.Nodes {
		if node.Trigger == TriggerScheduled {
			triggerCount++
		}
	}
	if triggerCount != 1 {
		t.Fatalf("public schedule trigger count = %d, want exactly one", triggerCount)
	}
	want := map[string]bool{
		"googleapis.com|GET|/v4/spreadsheets/synthetic-sheet-document/values/products":   false,
		"googleapis.com|GET|/v4/spreadsheets/synthetic-sheet-document/values/recipients": false,
		"googleapis.com|POST|/gmail/v1/users/me/messages/send":                           false,
	}
	for _, route := range profile.Routes {
		if route.Status != http.StatusOK || route.ContentType != "application/json" || len(route.Bindings) != 1 {
			t.Fatalf("route is not a single bounded JSON operation: %#v", route)
		}
		key := route.Method + "|" + route.Path
		if route.Host == "gmail.googleapis.com" {
			key = "googleapis.com|" + key
		} else if route.Host == "sheets.googleapis.com" {
			key = "googleapis.com|" + key
		} else {
			t.Fatalf("unexpected native provider host %q", route.Host)
		}
		if _, ok := want[key]; !ok {
			t.Fatalf("unexpected route %q", key)
		}
		want[key] = true
		if strings.Contains(route.Body, "@example.com") || strings.Contains(route.Body, "@gmail.com") {
			t.Fatalf("synthetic route contains a non-synthetic address: %s", key)
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("profile is missing exact native operation %q", route)
		}
	}
}

func TestVirtualAPIAdapterDropsProviderCredentialsOnlyFromTaskCopy(t *testing.T) {
	raw := readFixture(t, "public-workflows/16364.json")
	workflow, _, _, err := normalizeWorkflow(raw)
	if err != nil {
		t.Fatalf("normalize sanitized public #16364: %v", err)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-16364-native-profile.json"), false)
	if err != nil {
		t.Fatalf("load bounded Gmail/Sheets native API profile: %v", err)
	}
	profile, err := simulator.ExportVirtualAPIProfile()
	if err != nil {
		t.Fatalf("export task API profile: %v", err)
	}
	var source map[string]any
	if err := json.Unmarshal([]byte(raw), &source); err != nil {
		t.Fatal(err)
	}
	nodes := source["nodes"].([]any)
	for _, value := range nodes {
		node := value.(map[string]any)
		if node["id"] == "read-products-lite" {
			node["credentials"] = map[string]any{"googleSheetsOAuth2Api": map[string]any{"id": "source-only-ref", "name": "source credential"}}
		}
	}
	withReference, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	taskCopy, err := rewriteWorkflowHTTPRequestsForVirtualAPI(string(withReference), workflow, profile)
	if err != nil {
		t.Fatalf("rewrite provider node: %v", err)
	}
	if !strings.Contains(string(withReference), `"source-only-ref"`) {
		t.Fatal("source workflow copy lost its credential reference")
	}
	if strings.Contains(taskCopy, `"source-only-ref"`) || strings.Contains(taskCopy, `"credentials"`) {
		t.Fatal("no-auth task companion copy retained a provider credential reference")
	}
}
