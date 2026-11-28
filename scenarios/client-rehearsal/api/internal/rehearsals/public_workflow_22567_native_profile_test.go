package rehearsals

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// This binds the public workflow's generic HTTP Request node to one exact
// synthetic OpenAPI operation. The task path then uses the Sandbox's existing
// HTTP-mock declaration contract because .invalid hosts are not VirtualAPI
// hosts; this is profile/source coverage, not native-run acceptance.
func TestPublic22567ExportsExactSandboxHTTPMockRoute(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/22567.json")
	profileJSON := readFixture(t, "public-workflow-22567-api-profile.json")
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize sanitized public workflow: %v", err)
	}
	simulator, err := NewDescriptionSimulator(profileJSON, false)
	if err != nil {
		t.Fatal(err)
	}
	virtualProfile, err := simulator.ExportVirtualAPIProfile()
	if err != nil || !workflowUsesExactVirtualAPIRoutes(workflow, virtualProfile) {
		t.Fatalf("expected exact synthetic API routes in description profile: profile=%+v err=%v", virtualProfile, err)
	}
	if virtualAPIProfileUsesExternalHosts(virtualProfile) {
		t.Fatal("reserved synthetic.invalid host must use the Sandbox HTTP-mock contract")
	}
	profile, err := simulator.ExportHTTPMockProfile()
	if err != nil {
		t.Fatalf("export public profile as Sandbox HTTP mock: %v", err)
	}
	if len(profile.Operations) != 1 {
		t.Fatalf("expected exactly one public workflow route, got %d", len(profile.Operations))
	}
	op := profile.Operations[0]
	if op.Selector.NodeType != "n8n-nodes-base.httpRequest" || op.Selector.Resource != "request" || op.Selector.Operation != "request" {
		t.Fatalf("unexpected generic HTTP Request selector: %+v", op.Selector)
	}
	if op.Route.Host != "api.synthetic.invalid" || op.Route.Method != http.MethodGet || op.Route.Path != "/v1/invoices-and-payments" || op.Route.Status != http.StatusOK {
		t.Fatalf("unexpected exact public route: %+v", op.Route)
	}
	var response struct {
		Assignment struct {
			Invoices []any `json:"invoices"`
			Payments []any `json:"payments"`
		} `json:"assignment"`
	}
	if err := json.Unmarshal([]byte(op.Route.Body), &response); err != nil || len(response.Assignment.Invoices) != 2 || len(response.Assignment.Payments) != 1 {
		t.Fatalf("public route lacks bounded invoice/payment fixtures: body=%s err=%v", op.Route.Body, err)
	}

	// Ensure the unchanged public graph is routed to the address returned by
	// Workspace Sandbox's typed HTTP-mock profile start response.
	address := HTTPMockOperationAddress{Selector: op.Selector, Method: op.Route.Method, URL: "http://api.synthetic.invalid:18080/v1/invoices-and-payments"}
	encoded, err := rewriteWorkflowHTTPRequestsForSandbox(workflowJSON, []HTTPMockOperationAddress{address})
	if err != nil {
		t.Fatalf("rewrite public workflow to task-local route: %v", err)
	}
	var rewritten map[string]any
	if err := json.Unmarshal([]byte(encoded), &rewritten); err != nil {
		t.Fatal(err)
	}
	nodes := rewritten["nodes"].([]any)
	found := false
	for _, rawNode := range nodes {
		node := rawNode.(map[string]any)
		if node["type"] != "n8n-nodes-base.httpRequest" {
			continue
		}
		params := node["parameters"].(map[string]any)
		if params["url"] == address.URL {
			found = true
		}
	}
	if !found || strings.Contains(encoded, "https://api.synthetic.invalid") {
		t.Fatalf("public graph was not narrowed to its Sandbox route: %s", encoded)
	}

}
