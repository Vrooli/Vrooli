package rehearsals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestWaitNodeNormalizesToBoundedVirtualResume(t *testing.T) {
	polling := syntheticTriggerInput(TriggerPolling)
	if polling["Name"] != "Synthetic Lead" || polling["Email"] != "lead@example.invalid" || polling["Status"] != "New" || polling["Tag"] != "unprocessed" {
		t.Fatalf("polling trigger must receive the generic synthetic row, got %#v", polling)
	}
	for _, test := range []struct {
		amount any
		unit   string
		want   uint32
	}{
		{amount: float64(1), unit: "seconds", want: 1},
		{amount: float64(2), unit: "minutes", want: 120},
		{amount: float64(3), unit: "hours", want: 10800},
		{amount: float64(1), unit: "days", want: 86400},
	} {
		got, err := normalizeWait(map[string]any{"amount": test.amount, "unit": test.unit})
		if err != nil || got != test.want {
			t.Fatalf("normalizeWait(%v,%q) = %d, %v; want %d", test.amount, test.unit, got, err, test.want)
		}
	}
	for _, params := range []map[string]any{
		{"amount": float64(0), "unit": "seconds"},
		{"amount": float64(1.5), "unit": "seconds"},
		{"amount": float64(1), "unit": "weeks"},
		{"amount": float64(31), "unit": "days"},
		{"amount": "={{$json.delay}}", "unit": "seconds"},
		{"amount": float64(1), "unit": "seconds", "resume": map[string]any{"mode": "webhook"}},
	} {
		if _, err := normalizeWait(params); err == nil {
			t.Errorf("unsafe or unbounded Wait parameters must fail closed: %#v", params)
		}
	}
	seconds, path, unit, err := normalizeWaitNode(map[string]any{"amount": "={{ $json.seconds }}"})
	if err != nil || seconds != 0 || path != "seconds" || unit != "hours" {
		t.Fatalf("bounded current-item Wait expression must preserve the pinned default unit: seconds=%d path=%q unit=%q err=%v", seconds, path, unit, err)
	}
	for _, amount := range []string{"={{ $node[\"Earlier\"].json.seconds }}", "={{ $json.seconds + 1 }}", "={{ $json.seconds.toString() }}"} {
		if _, _, _, err := normalizeWaitNode(map[string]any{"amount": amount, "unit": "seconds"}); err == nil {
			t.Errorf("non-current-item or computed Wait expression must fail closed: %s", amount)
		}
	}
}

func TestPublic8995ImportFailsClosedOnUnmodeledOutputPorts(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/8995.json")
	digest := sha256.Sum256([]byte(workflowJSON))
	fixtureHash := hex.EncodeToString(digest[:])
	if fixtureHash != "ed98c6a7769266d79a7bfceb0491f3f91e3efc488922a6ce309c9e5b53694247" {
		t.Fatalf("sanitized 8995 fixture changed without provenance update: %s", fixtureHash)
	}
	if strings.Contains(strings.ToLower(workflowJSON), "credentials") || strings.Contains(workflowJSON, "jack-bluecrabai") || strings.Contains(workflowJSON, "@gmail.com") || strings.Contains(workflowJSON, "@company.com") || strings.Contains(workflowJSON, "BlueCrabAi") {
		t.Fatal("fixture retained credential markers, attribution metadata, client-specific prompt text, or a non-synthetic address")
	}
	var document map[string]any
	if err := json.Unmarshal([]byte(workflowJSON), &document); err != nil {
		t.Fatal(err)
	}
	nodes, _ := document["nodes"].([]any)
	if len(nodes) != 6 {
		t.Fatalf("sanitized public graph should contain six executable nodes, got %d", len(nodes))
	}
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err == nil || !strings.Contains(err.Error(), "outside the modeled output set") || !strings.Contains(err.Error(), "@n8n/n8n-nodes-langchain.openAi") {
		t.Fatalf("the source OpenAI node's output ports 1 and 2 must fail closed until their semantics are modeled, got workflow=%#v err=%v", workflow, err)
	}
}

func TestPublic8995NodeAdaptersRunOnlyThroughSyntheticProfiles(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/8995.json")
	var document struct {
		Nodes []map[string]any `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(workflowJSON), &document); err != nil {
		t.Fatal(err)
	}
	nodesByType := make(map[string]WorkflowNode, len(document.Nodes))
	for _, rawNode := range document.Nodes {
		id, _ := rawNode["id"].(string)
		name, _ := rawNode["name"].(string)
		typeName, _ := rawNode["type"].(string)
		params, _ := rawNode["parameters"].(map[string]any)
		node, err := normalizeWorkflowNode(id, name, typeName, params)
		if err != nil {
			t.Fatalf("normalize public node type %q for adapter proof: %v", typeName, err)
		}
		nodesByType[typeName] = node
	}
	for typeName, want := range map[string]string{
		"@n8n/n8n-nodes-langchain.openAi": "@n8n/n8n-nodes-langchain.openAi:chat.completion",
		"n8n-nodes-base.gmail":            "n8n-nodes-base.gmail:message.send",
		"n8n-nodes-base.googleSheets":     "n8n-nodes-base.googleSheets:sheet.update",
		"n8n-nodes-base.slack":            "n8n-nodes-base.slack:message.post",
	} {
		node, ok := nodesByType[typeName]
		if !ok || node.Request == nil || node.Request.Operation != want {
			t.Errorf("public node type %q should bind to its generic profile operation %q: %#v", typeName, want, node)
		}
	}
	if _, err := normalizeWorkflowNode("unsafe-openai", "Synthetic model", "@n8n/n8n-nodes-langchain.openAi", map[string]any{"resource": "assistant", "operation": "create"}); err == nil {
		t.Fatal("LangChain OpenAI adapter must reject operations outside synthetic chat completion")
	}
	if got := nodesByType["n8n-nodes-base.wait"].WaitSeconds; got != 1 {
		t.Errorf("public Wait node should use one virtual second, got %d", got)
	}

	profile := readFixture(t, "public-workflow-8995-profile.json")
	simulator, err := NewDescriptionSimulator(profile, false)
	if err != nil {
		t.Fatalf("load synthetic model, Gmail, Sheets, and Slack profiles: %v", err)
	}
	workflow := Workflow{
		ID: "generic-node-adapter-proof",
		Nodes: []WorkflowNode{
			{ID: "trigger", Name: "Synthetic trigger", Type: "n8n-nodes-base.manualTrigger", Trigger: TriggerManual},
			nodesByType["n8n-nodes-base.wait"],
			nodesByType["@n8n/n8n-nodes-langchain.openAi"],
			nodesByType["n8n-nodes-base.gmail"],
			nodesByType["n8n-nodes-base.googleSheets"],
			nodesByType["n8n-nodes-base.slack"],
		},
		EntryNodeIDs: []string{"trigger"},
	}
	for index, id := range []string{"trigger", "wait", "model", "mail", "sheet", "chat"} {
		workflow.Nodes[index].ID = id
	}
	workflow.Connections = []Connection{{From: "trigger", To: "wait"}, {From: "wait", To: "model"}, {From: "model", To: "mail"}, {From: "model", To: "sheet"}, {From: "model", To: "chat"}}
	run, err := (SyntheticEngine{}).ExecuteDetailed(context.Background(), workflow, "trigger", simulator, false, time.Date(2026, 11, 10, 13, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("run node adapters through synthetic profile: %v", err)
	}
	for _, operation := range []string{
		"Wait / virtual resume",
		"@n8n/n8n-nodes-langchain.openAi:chat.completion",
		"n8n-nodes-base.gmail:message.send",
		"n8n-nodes-base.googleSheets:sheet.update",
		"n8n-nodes-base.slack:message.post",
	} {
		if !traceHasOperation(run.Trace, operation) {
			t.Errorf("expected synthetic operation %q in %#v", operation, run.Trace)
		}
	}
	for _, event := range run.Trace {
		if strings.Contains(event.RequestSummary, "synthetic-recipient") || strings.Contains(event.ResponseSummary, "synthetic-recipient") || strings.Contains(event.ResponseSummary, "synthetic-leads") {
			t.Errorf("trace leaked synthetic destination fields: %#v", event)
		}
	}
}

func TestPublic7032OpenAIResourceLocatorIsBoundedAndNormalized(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/7032.json")
	var document struct {
		Nodes []map[string]any `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(workflowJSON), &document); err != nil {
		t.Fatal(err)
	}
	var openAINode map[string]any
	for _, node := range document.Nodes {
		if node["type"] == "@n8n/n8n-nodes-langchain.openAi" {
			openAINode = node
			break
		}
	}
	if openAINode == nil {
		t.Fatal("public 7032 fixture must retain its OpenAI node")
	}
	params, _ := openAINode["parameters"].(map[string]any)
	normalized, err := normalizeWorkflowNode("model", "Model", "@n8n/n8n-nodes-langchain.openAi", params)
	if err != nil {
		t.Fatalf("normalize public resource-locator model selector: %v", err)
	}
	if normalized.Request == nil || normalized.Request.Operation != "@n8n/n8n-nodes-langchain.openAi:chat.completion" {
		t.Fatalf("unexpected normalized adapter: %#v", normalized.Request)
	}

	base := map[string]any{"messages": map[string]any{"values": []any{map[string]any{"content": "static prompt"}}}}
	for name, selector := range map[string]any{
		"plain model ID":   "gpt-4.1",
		"n8n list locator": map[string]any{"__rl": true, "mode": "list", "value": "gpt-4.1", "cachedResultName": "GPT-4.1"},
		"n8n ID locator":   map[string]any{"__rl": true, "mode": "id", "value": "model_123"},
	} {
		params := map[string]any{"modelId": selector, "messages": base["messages"]}
		if _, err := normalizeWorkflowNode("model", "Model", "@n8n/n8n-nodes-langchain.openAi", params); err != nil {
			t.Errorf("accept %s: %v", name, err)
		}
	}
	for name, selector := range map[string]any{
		"unmarked locator": map[string]any{"mode": "list", "value": "gpt-4.1"},
		"expression value": map[string]any{"__rl": true, "mode": "list", "value": "={{$json.model}}"},
		"invalid mode":     map[string]any{"__rl": true, "mode": "expression", "value": "gpt-4.1"},
		"non-string ID":    map[string]any{"__rl": true, "mode": "list", "value": 42},
	} {
		params := map[string]any{"modelId": selector, "messages": base["messages"]}
		if _, err := normalizeWorkflowNode("model", "Model", "@n8n/n8n-nodes-langchain.openAi", params); err == nil {
			t.Errorf("reject %s", name)
		}
	}
}

func TestPublic7032WorkflowNormalizesWithoutChangingFixture(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/7032.json")
	workflow, sourceHash, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize original public 7032 workflow: %v", err)
	}
	rawDigest := sha256.Sum256([]byte(workflowJSON))
	if hex.EncodeToString(rawDigest[:]) != "53447d4861db07cf400a5bb6fcfaba0b6e5c24f0f3479640fcd5641be5cfca45" || sourceHash == "" || len(workflow.Nodes) != 12 || len(workflow.Connections) != 12 {
		t.Fatalf("unexpected normalized public graph shape: source=%q nodes=%d connections=%d", sourceHash, len(workflow.Nodes), len(workflow.Connections))
	}
	if len(workflow.EntryNodeIDs) != 2 {
		t.Errorf("both manual and schedule triggers should remain represented; got %v", workflow.EntryNodeIDs)
	}
}
