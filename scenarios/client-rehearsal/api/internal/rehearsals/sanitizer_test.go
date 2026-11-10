package rehearsals

import (
	"os"
	"strings"
	"testing"
)

// [REQ:CR-P0-001]
func TestNormalizeWorkflowImportsN8NGraphAndDropsCredentialMaterial(t *testing.T) {
	export := `{"id":"synthetic-flow","name":"synthetic intake","nodes":[{"id":"trigger-1","name":"Manual Trigger","type":"n8n-nodes-base.manualTrigger","position":[0,0],"parameters":{}},{"id":"request-1","name":"Create contact","type":"n8n-nodes-base.httpRequest","parameters":{"method":"POST","url":"https://crm.synthetic.invalid/v1/contacts?api_key=made-up-secret","jsonBody":"{\"name\":\"Avery Example\"}","sendHeaders":true,"headerParameters":{"parameters":[{"name":"Authorization","value":"Bearer synthetic-token"}]}},"credentials":{"httpHeaderAuth":{"id":"secret-id","name":"synthetic credential"}}}],"connections":{"Manual Trigger":{"main":[[{"node":"Create contact"}]]}}}`
	workflow, fingerprint, replacements, err := normalizeWorkflow(export)
	if err != nil {
		t.Fatal(err)
	}
	if len(workflow.Nodes) != 2 || workflow.Nodes[1].Request == nil || workflow.Nodes[1].Request.Method != "POST" || workflow.Nodes[1].Request.URL != "/v1/contacts" {
		t.Fatalf("unexpected normalized graph: %#v", workflow.Nodes)
	}
	if len(workflow.Connections) != 1 || replacements == 0 {
		t.Fatalf("expected one graph edge and replaced credentials, got edges=%d replacements=%d", len(workflow.Connections), replacements)
	}
	if strings.Contains(fingerprint, "secret") || strings.Contains(fingerprint, "Avery") {
		t.Fatalf("fingerprint must not retain imported content: %q", fingerprint)
	}
}

func TestWorkflowFingerprintIncludesNativeOperationAndSetProjection(t *testing.T) {
	base := `{"nodes":[{"name":"Manual","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"name":"Set","type":"n8n-nodes-base.set","parameters":{"mode":"raw","jsonOutput":"={{ $json.body }}"}},{"name":"Native","type":"n8n-nodes-base.supabase","parameters":{"resource":"row","operation":"get"}}],"connections":{"Manual":{"main":[[{"node":"Set"}]]},"Set":{"main":[[{"node":"Native"}]]}}}`
	first, firstHash, _, err := normalizeWorkflow(base)
	if err != nil {
		t.Fatal(err)
	}
	if firstHash == "" || first.Nodes[1].OutputPath != "body" {
		t.Fatalf("expected a fingerprinted safe Set projection, workflow=%#v hash=%q", first.Nodes[1], firstHash)
	}
	changedOperation := strings.Replace(base, `"operation":"get"`, `"operation":"delete"`, 1)
	_, operationHash, _, err := normalizeWorkflow(changedOperation)
	if err != nil {
		t.Fatal(err)
	}
	if firstHash == operationHash {
		t.Fatal("native operation changes must change the sanitized workflow fingerprint")
	}
	changedProjection := strings.Replace(base, `json.body`, `json.query`, 1)
	_, projectionHash, _, err := normalizeWorkflow(changedProjection)
	if err != nil {
		t.Fatal(err)
	}
	if firstHash == projectionHash {
		t.Fatal("safe Set projection changes must change the sanitized workflow fingerprint")
	}
}

func TestNormalizeWorkflowClassifiesTriggerFamilies(t *testing.T) {
	cases := []struct {
		typeName string
		want     TriggerKind
	}{
		{"n8n-nodes-base.manualTrigger", TriggerManual},
		{"n8n-nodes-base.webhook", TriggerWebhook},
		{"n8n-nodes-base.scheduleTrigger", TriggerScheduled},
		{"n8n-nodes-base.gmailTrigger", TriggerPolling},
		{"n8n-nodes-base.chatTrigger", TriggerChat},
	}
	for _, test := range cases {
		t.Run(string(test.want), func(t *testing.T) {
			raw := `{"id":"trigger-test","nodes":[{"id":"trigger","name":"Trigger","type":"` + test.typeName + `","parameters":{}}],"connections":{}}`
			workflow, _, _, err := normalizeWorkflow(raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(workflow.TriggerKinds) != 1 || workflow.TriggerKinds[0] != test.want || len(workflow.EntryNodeIDs) != 1 {
				t.Fatalf("got triggers=%v entries=%v; want %s", workflow.TriggerKinds, workflow.EntryNodeIDs, test.want)
			}
		})
	}
}

func TestNormalizeWorkflowRejectsUnknownCodeAndCycles(t *testing.T) {
	exports := []string{
		`{"nodes":[{"name":"Trigger","type":"n8n-nodes-base.webhook","parameters":{}},{"name":"Code","type":"n8n-nodes-base.code","parameters":{}}],"connections":{"Trigger":{"main":[[{"node":"Code"}]]}}}`,
		`{"nodes":[{"name":"Trigger","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"name":"No-op","type":"n8n-nodes-base.noOp","parameters":{}}],"connections":{"Trigger":{"main":[[{"node":"No-op"}]]},"No-op":{"main":[[{"node":"No-op"}]]}}}`,
	}
	for _, export := range exports {
		if _, _, _, err := normalizeWorkflow(export); err == nil {
			t.Fatal("expected unsupported executable node or graph cycle to be rejected")
		}
	}
}

func TestNormalizeWorkflowRejectsExportAboveBound(t *testing.T) {
	tooLarge := strings.Repeat(" ", maxWorkflowBytes+1)
	if _, _, _, err := normalizeWorkflow(tooLarge); err == nil {
		t.Fatal("expected the bounded importer to reject an oversized export")
	}
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
