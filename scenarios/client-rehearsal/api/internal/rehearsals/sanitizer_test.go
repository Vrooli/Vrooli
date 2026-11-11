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

func TestNormalizeManualSetAndSplitOutUsesBoundedStaticSelectors(t *testing.T) {
	base := `{"nodes":[{"name":"Manual","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"name":"Settings","type":"n8n-nodes-base.set","parameters":{"mode":"manual","includeOtherFields":false,"assignments":{"assignments":[{"name":"account","type":"string","value":"synthetic-account"}]} }},{"name":"Split","type":"n8n-nodes-base.splitOut","parameters":{"fieldToSplitOut":"payload.items","include":"noOtherFields"}}],"connections":{"Manual":{"main":[[{"node":"Settings"}]]},"Settings":{"main":[[{"node":"Split"}]]}}}`
	workflow, firstHash, _, err := normalizeWorkflow(base)
	if err != nil {
		t.Fatal(err)
	}
	if workflow.Nodes[1].SetValues["account"] != "synthetic-account" || workflow.Nodes[1].SetIncludeInput || workflow.Nodes[2].SplitField != "payload.items" {
		t.Fatalf("unexpected normalized manual Set or Split Out nodes: %#v", workflow.Nodes)
	}
	changedValue := strings.Replace(base, "synthetic-account", "another-account", 1)
	_, changedHash, _, err := normalizeWorkflow(changedValue)
	if err != nil {
		t.Fatal(err)
	}
	if firstHash == changedHash {
		t.Fatal("manual Set value changes must change the sanitized workflow fingerprint")
	}
	changedSplit := strings.Replace(base, "payload.items", "payload.entries", 1)
	_, splitHash, _, err := normalizeWorkflow(changedSplit)
	if err != nil {
		t.Fatal(err)
	}
	if firstHash == splitHash {
		t.Fatal("Split Out path changes must change the sanitized workflow fingerprint")
	}
	projection := strings.Replace(base, `"value":"synthetic-account"`, `"value":"={{ $json.account }}"`, 1)
	projected, projectionHash, _, err := normalizeWorkflow(projection)
	if err != nil {
		t.Fatalf("simple JSON-path Set projection should be supported: %v", err)
	}
	if projectionHash == firstHash || projected.Nodes[1].SetProjections["account"] != "account" {
		t.Fatalf("expected projection path to be normalized and fingerprinted: %#v", projected.Nodes[1])
	}
	for _, unsafe := range []string{
		strings.Replace(base, `"value":"synthetic-account"`, `"value":"={{ $json.account + process.env.SECRET }}"`, 1),
	} {
		if _, _, _, err := normalizeWorkflow(unsafe); err == nil {
			t.Fatal("expected executable Set expression or indexed Split Out path to fail closed")
		}
	}
	indexed := strings.Replace(base, `"fieldToSplitOut":"payload.items"`, `"fieldToSplitOut":"payload.items[0]"`, 1)
	parsed, _, _, err := normalizeWorkflow(indexed)
	if err != nil || parsed.Nodes[2].SplitField != "payload.items[0]" {
		t.Fatalf("bounded static array path should be supported: %#v %v", parsed, err)
	}
	for _, oversized := range []string{"payload.items[101]", "payload.items[9999]", "payload.items[one]"} {
		if validJSONPath(oversized) {
			t.Fatalf("unsafe array path %q accepted", oversized)
		}
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

func TestClassifyPostgresStatementRetainsOnlyOneSafeQueryVerb(t *testing.T) {
	for _, test := range []struct {
		query string
		want  string
	}{
		{"SELECT * FROM jobs", "select"},
		{"INSERT INTO jobs (title) VALUES ('synthetic')", "insert"},
		{"UPDATE jobs SET status = 'closed'", "update"},
		{"DELETE FROM jobs WHERE id = 'synthetic'", "delete"},
	} {
		got, ok := classifyPostgresStatement(test.query)
		if !ok || got != test.want {
			t.Errorf("classifyPostgresStatement(%q) = %q, %v; want %q, true", test.query, got, ok, test.want)
		}
	}
	for _, query := range []string{
		"DROP TABLE jobs",
		"SELECT * FROM jobs; DELETE FROM jobs",
		"={{ $json.query }}",
		strings.Repeat("x", 16*1024+1),
	} {
		if got, ok := classifyPostgresStatement(query); ok {
			t.Errorf("classifyPostgresStatement(%q) = %q, true; want rejection", query, got)
		}
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
