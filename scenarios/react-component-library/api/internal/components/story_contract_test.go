package components

import "testing"

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
