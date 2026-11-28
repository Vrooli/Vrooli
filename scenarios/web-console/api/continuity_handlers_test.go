package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/identity"
	"web-console/internal/continuity"
)

// uncatalogedRepairContext carries an agent holding the delegated grant for
// the missing-catalog repair, and nothing wider.
func uncatalogedRepairContext() context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorAgent, Subject: "fixture-agent", Verified: true,
		Source: identity.SourceAgentProvenance, Scopes: []string{"web-console:write"},
		Grant: &identity.OperationGrant{
			ID: "fixture-repair", Approver: "fixture-operator", ExpiresAt: time.Now().Add(time.Hour),
			Operations: []string{"continuity.reconcile"}, Targets: []string{"web-console:continuity/uncataloged"},
		},
	})
}

func TestReconcileDryRunIsReadOnlyAndBindsBoundedApply(t *testing.T) {
	db := setupTestDB(t)
	ctx := uncatalogedRepairContext()
	if _, err := db.ExecContext(ctx, `INSERT INTO conversation_sessions (session_id, created_at, updated_at) VALUES ('manifest-orphan', '2026-09-06T00:00:00Z', '2026-09-06T00:00:00Z'),
			('second-orphan', '2026-09-06T00:00:00Z', '2026-09-06T00:00:00Z'),
			('live-pane', '2026-09-06T00:00:00Z', '2026-09-06T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO conversation_events (id, session_id, source, role, text, created_at, sequence, delivery_state, tts_state, consumption_state) VALUES ('manifest-event', 'manifest-orphan', 'codex_tailer', 'assistant', 'durable evidence', '2026-09-06T00:00:00Z', 1, 'pending', 'idle', 'unseen')`); err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO sessions (id, backend) VALUES ('live-pane', 'persistent')`); err != nil {
		t.Fatal(err)
	}
	srv := &Server{db: database.NewFromPrimary(db)}
	before, err := continuity.NewIntegrityAuditor(db).Audit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// The first preview page carries no generation: it observes one and binds
	// the manifest to it, and later pages and the apply present both.
	preview, err := srv.reconcileCatalog(ctx, false, "", "", "", 0, 1, true)
	if err != nil || len(preview.Items) != 1 || preview.Observations != 2 || preview.ManifestHash == "" || preview.NextOffset != 1 || preview.Complete || preview.Generation != before.Generation {
		t.Fatalf("dry-run %+v err=%v; want one item bound to generation %s", preview, err, before.Generation)
	}
	generation, manifestHash := preview.Generation, preview.ManifestHash
	last, err := srv.reconcileCatalog(ctx, false, generation, "", manifestHash, preview.NextOffset, 1, true)
	if err != nil || len(last.Items) != 1 || !last.Complete || preview.Items[0].Record.SessionID == last.Items[0].Record.SessionID {
		t.Fatalf("second preview page: %+v err=%v", last, err)
	}
	if _, err := srv.reconcileCatalog(ctx, true, generation, "manifest-reconcile", manifestHash, 0, 1, false); err == nil {
		t.Fatal("narrow repair grant was accepted for the whole catalog")
	}

	if _, err := srv.reconcileCatalog(context.Background(), true, generation, "manifest-reconcile", manifestHash, 0, 1, true); err == nil {
		t.Fatal("apply accepted a caller without verified operation authority")
	}
	var manifests int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM continuity_reconciliation_manifests").Scan(&manifests); err != nil || manifests != 0 {
		t.Fatalf("preview wrote %d manifests, err=%v", manifests, err)
	}
	if _, err := srv.reconcileCatalog(ctx, true, generation, "manifest-reconcile", "different", 0, 1, true); err == nil {
		t.Fatal("apply accepted an unreviewed manifest")
	}
	if _, err := srv.reconcileCatalog(ctx, true, generation, "manifest-reconcile", manifestHash, 1, 1, true); err == nil {
		t.Fatal("apply accepted an initial offset past unreviewed work")
	}
	first, err := srv.reconcileCatalog(ctx, true, generation, "manifest-reconcile", manifestHash, 0, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.Mutations != 1 || first.Receipt.Status != "pending" || first.Receipt.ActorID != "fixture-agent" || first.ManifestHash != manifestHash || first.NextOffset != 1 || first.Complete {
		t.Fatalf("first apply %+v", first)
	}
	items := first.Items
	if _, err := db.ExecContext(ctx, `INSERT INTO sessions (id, backend) VALUES ('new-live-pane', 'persistent')`); err != nil {
		t.Fatal(err)
	}
	second, err := srv.reconcileCatalog(ctx, true, generation, "manifest-reconcile", manifestHash, first.NextOffset, 1, true)
	if err != nil || len(second.Items) != 1 || second.Mutations != 1 || second.Receipt.Status != "succeeded" || second.NextOffset != 2 || !second.Complete {
		t.Fatalf("second apply %+v err=%v", second, err)
	}
	items = append(items, second.Items...)

	existing, err := continuity.NewSQLCatalogStore(db).Existing(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := existing["manifest-orphan"]; !ok {
		t.Fatal("manifest apply did not materialize the reviewed orphan record")
	}
	if _, ok := existing["live-pane"]; ok {
		t.Fatal("narrow manifest touched an unrelated live session")
	}

	replay, err := srv.reconcileCatalog(ctx, true, generation, "manifest-reconcile", manifestHash, 0, 500, true)
	if err != nil {
		t.Fatal(err)
	}
	if replay.Mutations != 0 || replay.Receipt.Status != "succeeded" || replay.ManifestHash != manifestHash || replay.NextOffset != len(items) || !replay.Complete {
		t.Fatalf("replay %+v", replay)
	}
	after, err := continuity.NewIntegrityAuditor(db).Audit(ctx)
	if err != nil || after.UncatalogedConversations != 0 || before.EventContentHash != after.EventContentHash {
		t.Fatalf("catalog repair failed conservation: before=%+v after=%+v err=%v", before, after, err)
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

// Live capture moves the inventory while counts stay level: the native
// archive indexer replaces a checkpoint whose key is a content hash. A
// generation taken before such a change must be refused, and a preview that
// observes its own generation must yield a manifest an apply can complete.
func TestReconcilePreviewBindsTheGenerationItObserves(t *testing.T) {
	db := setupTestDB(t)
	ctx := uncatalogedRepairContext()
	for _, statement := range []string{
		`INSERT INTO conversation_sessions (session_id, created_at, updated_at) VALUES ('lost-orphan', '2026-09-06T00:00:00Z', '2026-09-06T00:00:00Z')`,
		`INSERT INTO agent_transcript_checkpoints (source, source_key, web_console_session_id, cursor, updated_at) VALUES ('native-archive-index', 'sha-old', 'wc-archive-one', '{}', '2026-09-06T00:00:00Z')`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	srv := &Server{db: database.NewFromPrimary(db)}
	stale, err := continuity.NewIntegrityAuditor(db).Audit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`DELETE FROM agent_transcript_checkpoints WHERE source_key = 'sha-old'`,
		`INSERT INTO agent_transcript_checkpoints (source, source_key, web_console_session_id, cursor, updated_at) VALUES ('native-archive-index', 'sha-new', 'wc-archive-one', '{}', '2026-09-06T00:01:00Z')`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	moved, err := continuity.NewIntegrityAuditor(db).Audit(ctx)
	if err != nil || moved.Checkpoints != stale.Checkpoints || moved.Generation == stale.Generation {
		t.Fatalf("fixture must move the inventory at a level count: before %+v after %+v err=%v", stale, moved, err)
	}
	if _, err := srv.reconcileCatalog(ctx, false, stale.Generation, "", "", 0, 0, true); err == nil || !strings.Contains(err.Error(), "integrity generation changed") {
		t.Fatalf("a preview accepted a generation the inventory has left: %v", err)
	}
	preview, err := srv.reconcileCatalog(ctx, false, "", "", "", 0, 0, true)
	if err != nil || preview.Generation != moved.Generation || len(preview.Items) != 1 || !preview.Complete {
		t.Fatalf("preview %+v err=%v; want one item bound to %s", preview, err, moved.Generation)
	}
	applied, err := srv.reconcileCatalog(ctx, true, preview.Generation, "observed-generation-repair", preview.ManifestHash, 0, 0, true)
	if err != nil || applied.Mutations != 1 || !applied.Complete || applied.Receipt.Status != "succeeded" {
		t.Fatalf("apply with the preview's generation: %+v err=%v", applied, err)
	}
}
