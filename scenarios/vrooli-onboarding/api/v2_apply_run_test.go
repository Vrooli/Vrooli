package main

import "testing"

func TestSelectionDigestChangesWhenConsentPlanChanges(t *testing.T) {
	base := []applyItem{{ID: "safeguard:firewall", Kind: "safeguard", Name: "firewall"}}
	withResource := append(append([]applyItem(nil), base...), applyItem{ID: "resource:postgres", Kind: "resource", Name: "postgres"})
	if selectionDigest(base) == selectionDigest(withResource) {
		t.Fatal("selection digest ignored a plan item")
	}
}

func TestApplyRunPersistsPendingItems(t *testing.T) {
	t.Setenv("VROOLI_ROOT", t.TempDir())
	run := applyRun{ID: "apply-test", Status: "pending", Items: []applyItemResult{{applyItem: applyItem{ID: "resource:postgres", Kind: "resource", Name: "postgres"}, Outcome: "pending"}}}
	storeApplyRun(run)
	loaded, ok := applyRunSnapshot(run.ID)
	if !ok || loaded.Items[0].Outcome != "pending" {
		t.Fatalf("persisted run = %#v, found=%v", loaded, ok)
	}
}
