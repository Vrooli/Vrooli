package components

import "testing"

func TestParseStoryContractSchemaVersionTwoParsesHarnessAndDescription(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 2,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"controlled","name":"Controlled","description":"Shows the selected value.","harness":"ControlledWithReadout","args":{}}]
}`))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if got := contract.Stories[0]; got.Harness != "ControlledWithReadout" || got.Description != "Shows the selected value." {
		t.Fatalf("story fields = %#v", got)
	}
}

func TestParseStoryContractRejectsInvalidHarnessExport(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 2,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"controlled","name":"Controlled","harness":"not-a-valid-export","args":{}}]
}`))
	if len(diagnostics) != 1 || diagnostics[0].Rule != "javascript_identifier" {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
}

func TestParseStoryContractStillRejectsUnknownSchemaVersionTwoField(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 2,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"controlled","name":"Controlled","caption":"misspelled","args":{}}]
}`))
	if len(diagnostics) != 1 || diagnostics[0].Rule != "valid_json" {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
}

func TestParseStoryContractValidatesOneAssetLevelSchema(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 1,
  "kind": "component",
  "args": {"fields": [
    {"path":"tone","kind":"enum","required":true,"options":["success","warning"]},
    {"path":"count","kind":"number","default":1,"minimum":0}
  ]},
  "environment": {"fixtures":[{"key":"voiceInput","adapter":"voice-input","options":["idle"]}]},
  "stories":[{"id":"success","name":"Success","args":{"tone":"success"},"environment":{"voiceInput":"idle"},"expect":[{"kind":"text","value":"Ready"}]}]
}`))
	if contract == nil || len(diagnostics) != 0 {
		t.Fatalf("contract=%#v diagnostics=%v", contract, diagnostics)
	}
}

func TestParseStoryContractRejectsUnsafeAndLegacyStyleInput(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 1,
  "kind": "component",
  "args": {"fields": [{"path":"__proto__.tone","kind":"select"}]},
  "environment": {"fixtures": []},
  "stories":[{"id":"bad","name":"Bad","args":{"__proto__":"pollution"},"controls":{}}]
}`))
	if len(diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}
}

func TestParseStoryContractRejectsUndeclaredFixtureAdapter(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 1,
  "kind": "hook",
  "args": {"fields": []},
  "environment": {"fixtures": [{"key":"state","adapter":"arbitrary-import","options":["idle"]}]},
  "stories": [{"id":"idle","name":"Idle","args":{},"environment":{"state":"idle"}}]
}`))
	if len(diagnostics) == 0 {
		t.Fatal("expected unsupported adapter diagnostic")
	}
}

func TestParseStoryContractRequiresSafeComponentInteractionLocator(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 1,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"open","name":"Open","args":{},"interactions":[{"kind":"click"}]}]
}`))
	if len(diagnostics) == 0 {
		t.Fatal("expected interaction target diagnostic")
	}
}

func TestStoryCoverageGapsNamesEveryUnstoriedEnumValue(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 1,
  "kind": "component",
  "args": {"fields": [{"path":"state","kind":"enum","options":["idle","recording","error"]}]},
  "environment": {"fixtures": []},
  "stories": [
    {"id":"idle","name":"Idle","args":{"state":"idle"}},
    {"id":"recording","name":"Recording","args":{"state":"recording"}}
  ]
}`))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	gaps := StoryCoverageGaps(contract)
	if len(gaps) != 1 || gaps[0].Path != "state" || gaps[0].Value != `"error"` {
		t.Fatalf("coverage gaps = %#v", gaps)
	}
}
