package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func TestRecoveryActivationRejectsAnOldCapabilityAfterRestoredRecordSurvives(t *testing.T) {
	manager := newPasswordManager(nil)
	token := "old-use-token"
	digest := sha256.Sum256([]byte(token))
	old := brokerSession{
		ID: "restored-session", TokenDigest: hex.EncodeToString(digest[:]), Workspace: "workspace-a", Actor: "agent-a",
		GrantID: "restored-grant", ItemID: "item-a", Origin: "https://example.test", Status: "active",
		RecoveryEpoch: 1, Expires: time.Now().UTC().Add(time.Hour),
	}
	manager.brokers[old.ID] = old

	newEpoch, err := manager.activateRecovery(context.Background(), "workspace-a", "owner-a", 1)
	if err != nil {
		t.Fatal(err)
	}
	if newEpoch != 2 {
		t.Fatalf("new recovery epoch = %d, want 2", newEpoch)
	}

	// Simulate a restored metadata row that was not rewritten by activation.
	// The epoch check remains authoritative even if its status was copied as
	// active from the old database.
	old.Status = "active"
	manager.brokers[old.ID] = old
	if _, err := manager.brokerSessionFor(context.Background(), "workspace-a", "agent-a", token); err == nil {
		t.Fatal("capability from the previous recovery epoch was accepted")
	}
}

func TestRecoveryActivationQuarantinesRestoredGrants(t *testing.T) {
	manager := newPasswordManager(nil)
	manager.grants["grant-a"] = grantRecord{ID: "grant-a", WorkspaceID: "workspace-a", Status: "active"}
	if _, err := manager.activateRecovery(context.Background(), "workspace-a", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if got := manager.grants["grant-a"].Status; got != "recovery_review" {
		t.Fatalf("restored grant status = %q, want recovery_review", got)
	}
}

func TestRecoveryActivationAllowsNewAuthorizedUseAfterDenyingOldSession(t *testing.T) {
	manager, item, oldGrant := credentialUseFixture(t)
	executor := &recordingBrowserExecutor{}
	manager.SetTrustedBrowserExecutor(executor)
	oldSession, err := manager.createBrowserSession(context.Background(), "workspace-a", "agent-a", browserUseInput{
		GrantID: oldGrant.ID, ItemID: item.ID, Origin: "https://login.example.test", Account: item.Username,
		DocumentID: "old-document", Allowed: []string{"login"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.activateRecovery(context.Background(), "workspace-a", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.browserAction(context.Background(), "workspace-a", "agent-a", oldSession.ID, browserActionInput{
		Action: "fill", Origin: oldSession.Origin, DocumentID: oldSession.DocumentID,
	}); err == nil {
		t.Fatal("old browser capability was accepted after recovery activation")
	}

	newGrant, err := manager.createGrant(context.Background(), "workspace-a", "owner-a", createGrantInput{
		VaultID: oldGrant.VaultID, ItemID: item.ID, PrincipalType: "agent", PrincipalID: "agent-a",
		SelectorMode: "current_snapshot", Members: []string{"agent-a"}, Operations: []string{"inject"},
		Target: "https://login.example.test", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	newSession, err := manager.createBrowserSession(context.Background(), "workspace-a", "agent-a", browserUseInput{
		GrantID: newGrant.ID, ItemID: item.ID, Origin: "https://login.example.test", Account: item.Username,
		DocumentID: "new-document", Allowed: []string{"login"},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := manager.browserAction(context.Background(), "workspace-a", "agent-a", newSession.ID, browserActionInput{
		Action: "fill", Origin: newSession.Origin, DocumentID: newSession.DocumentID,
	})
	if err != nil || result.Status != "active" || result.OperationClass != "trusted_fill" || len(executor.filled) == 0 {
		t.Fatalf("new authorized browser use = %+v, err=%v", result, err)
	}
}
