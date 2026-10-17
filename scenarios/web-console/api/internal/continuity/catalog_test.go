package continuity

import "testing"

func TestCatalogRecordUsesDeterministicFallbackAndFingerprint(t *testing.T) {
	e := Evidence{SessionID: "a7e71c3c-e422-4c89-916a-03f92906fb89", AgentType: "codex", AgentSessionID: "01a06a6b-88da-7422-b391-bb59c5f5e5e0", HasConversation: true}
	a, err := BuildCatalogRecord(e)
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildCatalogRecord(e)
	if err != nil {
		t.Fatal(err)
	}
	if a.CurrentTitle != "Web Console conversation a7e71c3c" || a.SourceFingerprint != b.SourceFingerprint {
		t.Fatalf("record=%+v repeat=%+v", a, b)
	}
	if a.LifecycleState != StateRecoverable {
		t.Fatalf("state=%q", a.LifecycleState)
	}
}

func TestPlanReconciliationIsIdempotentAndQuarantinesAliasCollision(t *testing.T) {
	first := Evidence{SessionID: "one", AgentSessionID: "agent-one", HasNativeHistory: true}
	second := Evidence{SessionID: "two", AgentSessionID: "agent-one", HasNativeHistory: true}
	items, err := PlanReconciliation([]Evidence{second, first}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Action != ActionCreate || items[1].Action != ActionQuarantine {
		t.Fatalf("items=%+v", items)
	}
	existing := map[string]CatalogRecord{items[0].Record.SessionID: items[0].Record}
	repeat, err := PlanReconciliation([]Evidence{first}, existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(repeat) != 0 {
		t.Fatalf("idempotent repeat mutated: %+v", repeat)
	}
}

func TestManifestHashBindsGenerationAndDryRunItems(t *testing.T) {
	items, err := PlanReconciliation([]Evidence{{SessionID: "one", HasConversation: true}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := ManifestHash("generation-a", items)
	if a == "" || a == ManifestHash("generation-b", items) || a == ManifestHash("generation-a", nil) {
		t.Fatalf("manifest hash is not bound to reviewed evidence: %q", a)
	}
}
