package main

import (
	"context"
	"testing"

	"github.com/vrooli/api-core/database"
)

func TestConversationCapabilitiesExposeSafeOperationEffects(t *testing.T) {
	capability := conversationCapabilities("codex_app_server")
	if capability.Provider != "codex" || len(capability.Options) != 5 {
		t.Fatalf("capability = %+v", capability)
	}
	for _, option := range capability.Options {
		if option.ID == "restore_conversation" && (!option.Default || !option.ChangesConversation || option.ChangesWorkspace || !option.RequiresConfirmation) {
			t.Fatalf("conversation option effects = %+v", option)
		}
		if option.ID == "restore_code" && !option.ChangesWorkspace {
			t.Fatalf("code option must declare workspace effect = %+v", option)
		}
	}
}

func TestLegacyCapabilityDoesNotAdvertiseNativeOperations(t *testing.T) {
	capability := legacyCapability("codex")
	if capability.Available || capability.LaunchMode != "terminal_pty" || capability.ControlMode != "transcript_only" {
		t.Fatalf("legacy capability = %+v", capability)
	}
	for _, option := range capability.Options {
		if option.Supported || option.UnavailableReason == "" {
			t.Fatalf("legacy option = %+v", option)
		}
	}
}

func TestReconcileOrphanedControlReceiptsMarksRestartUncertain(t *testing.T) {
	db := setupTestDB(t)
	if _, err := db.Exec(`INSERT INTO conversation_control_receipts(operation_id, session_id, event_id, state, reason_code, detail, created_at, updated_at) VALUES ('op-restart', 'session-1', 'event-1', 'executing', 'supported', 'running', '2026-09-14T00:00:00Z', '2026-09-14T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	srv := &Server{db: database.NewFromPrimary(db)}
	if err := srv.reconcileOrphanedControlReceipts(context.Background()); err != nil {
		t.Fatal(err)
	}
	var state, reason, detail string
	if err := db.QueryRow(`SELECT state, reason_code, detail FROM conversation_control_receipts WHERE operation_id = 'op-restart'`).Scan(&state, &reason, &detail); err != nil {
		t.Fatal(err)
	}
	if state != "failed-uncertain" || reason != "failed_uncertain" || detail == "running" {
		t.Fatalf("receipt = state=%q reason=%q detail=%q", state, reason, detail)
	}
}
