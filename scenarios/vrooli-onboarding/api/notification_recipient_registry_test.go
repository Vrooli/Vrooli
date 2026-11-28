package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/identity"
)

func TestNotificationRecipientUsesRegistryCASAndReceipt(t *testing.T) {
	previousPath, previousRoots := operatorStatePath, operatorStateRoots
	statePath := filepath.Join(t.TempDir(), "operator-state.json")
	operatorStatePath = func() (string, error) { return statePath, nil }
	operatorStateRoots = nil
	t.Cleanup(func() { operatorStatePath, operatorStateRoots = previousPath, previousRoots })

	ctx := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorHuman, Subject: "operator-notify", Verified: true,
		Scopes: []string{"vrooli-onboarding:read", "vrooli-onboarding:write"},
	})
	if err := patchNotificationRecipient(ctx, "  operator@example.test  "); err != nil {
		t.Fatalf("patch recipient: %v", err)
	}
	doc, err := operatorStateService().Load(context.Background())
	if err != nil {
		t.Fatalf("load owner state: %v", err)
	}
	if doc.Notifications == nil || doc.Notifications.Recipient != "operator@example.test" {
		t.Fatalf("notification recipient = %+v", doc.Notifications)
	}
	if len(doc.PermissionReceipts) != 1 {
		t.Fatalf("permission receipts = %+v, want one durable registry receipt", doc.PermissionReceipts)
	}
	for _, receipt := range doc.PermissionReceipts {
		if receipt.Owner != "vrooli-onboarding" || receipt.PermissionID != "onboarding/operator-state" ||
			receipt.ActorID != "operator-notify" || receipt.ExpectedRevision == "" || receipt.ResultRevision == "" ||
			receipt.RequestID == "" || receipt.RequestDigest == "" || receipt.Outcome == "" {
			t.Fatalf("recipient receipt does not bind source, actor and operation: %+v", receipt)
		}
	}
}
