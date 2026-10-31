package main

import (
	"context"
	"testing"

	"github.com/vrooli/api-core/database"
	"web-console/internal/continuity"
)

func TestReconcileDryRunPersistsExactManifestForApply(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO conversation_sessions (session_id, created_at, updated_at) VALUES ('manifest-orphan', '2026-09-06T00:00:00Z', '2026-09-06T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO conversation_events (id, session_id, source, role, text, created_at, sequence, delivery_state, tts_state, consumption_state) VALUES ('manifest-event', 'manifest-orphan', 'codex_tailer', 'assistant', 'durable evidence', '2026-09-06T00:00:00Z', 1, 'pending', 'idle', 'unseen')`); err != nil {
		t.Fatal(err)
	}

	srv := &Server{db: database.NewFromPrimary(db)}
	_, observations, _, _, manifestHash, _, complete, err := srv.Reconcile(ctx, false, "", "", "", 0, 0)
	if err != nil || observations == 0 || manifestHash == "" || !complete {
		t.Fatalf("dry-run observations=%d manifest=%q complete=%v err=%v", observations, manifestHash, complete, err)
	}

	items, _, mutations, receipt, appliedManifest, next, appliedComplete, err := srv.Reconcile(ctx, true, "", "manifest-reconcile", manifestHash, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 || mutations == 0 || receipt.Status != "succeeded" || appliedManifest != manifestHash || next != len(items) || !appliedComplete {
		t.Fatalf("apply items=%d mutations=%d receipt=%+v manifest=%q next=%d complete=%v", len(items), mutations, receipt, appliedManifest, next, appliedComplete)
	}

	existing, err := continuity.NewSQLCatalogStore(db).Existing(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := existing["manifest-orphan"]; !ok {
		t.Fatal("manifest apply did not materialize the reviewed orphan record")
	}

	_, _, replayMutations, replayReceipt, replayManifest, replayNext, replayComplete, err := srv.Reconcile(ctx, true, "", "manifest-reconcile-replay", manifestHash, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	if replayMutations != 0 || replayReceipt.Status != "succeeded" || replayManifest != manifestHash || replayNext != len(items) || !replayComplete {
		t.Fatalf("replay mutations=%d receipt=%+v manifest=%q next=%d complete=%v", replayMutations, replayReceipt, replayManifest, replayNext, replayComplete)
	}
	var nextOffset, totalItems int
	var progressStatus string
	if err := db.QueryRowContext(ctx, `SELECT next_offset, total_items, status FROM continuity_reconciliation_progress WHERE operation_id = 'manifest-reconcile'`).Scan(&nextOffset, &totalItems, &progressStatus); err != nil {
		t.Fatal(err)
	}
	if nextOffset != len(items) || totalItems != len(items) || progressStatus != "succeeded" {
		t.Fatalf("progress=(%d/%d,%s), want completed %d-item operation", nextOffset, totalItems, progressStatus, len(items))
	}
	var itemReceipts, completedItems int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(status = 'succeeded'), 0) FROM continuity_reconciliation_item_receipts WHERE operation_id = 'manifest-reconcile'`).Scan(&itemReceipts, &completedItems); err != nil {
		t.Fatal(err)
	}
	if itemReceipts != len(items) || completedItems != len(items) {
		t.Fatalf("item receipts=(%d,%d), want %d completed receipts", itemReceipts, completedItems, len(items))
	}
}
