package sessions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/identity"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"

	_ "modernc.org/sqlite"
	"web-console/internal/backend"
	"web-console/internal/continuity"
	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/ptyfake"
	"web-console/internal/sessionstore"
	"web-console/internal/workspace"
	"web-console/session"
)

type adapterConversationStub struct{}

func (adapterConversationStub) DeleteSession(context.Context, string) error       { return nil }
func (adapterConversationStub) CopySession(context.Context, string, string) error { return nil }
func (adapterConversationStub) HasConversationAfter(context.Context, string, time.Time) bool {
	return true
}
func (adapterConversationStub) CountSessionEvents(context.Context, string) int64  { return 2 }
func (adapterConversationStub) SessionStorageBytes(context.Context, string) int64 { return 128 }

type adapterConversationErrorStub struct{ adapterConversationStub }

func (adapterConversationErrorStub) DeleteSession(context.Context, string) error {
	return context.DeadlineExceeded
}

type emptyConversationStub struct{ adapterConversationStub }

func (emptyConversationStub) CountSessionEvents(context.Context, string) int64  { return 0 }
func (emptyConversationStub) SessionStorageBytes(context.Context, string) int64 { return 0 }

type failingCreateMetadataStore struct {
	sessionstore.Store
	failAgentInfo  bool
	failProvenance bool
}

func (s failingCreateMetadataStore) UpdateAgentInfo(ctx context.Context, id string, info sessionstore.AgentInfo) error {
	if s.failAgentInfo {
		return errors.New("agent metadata unavailable")
	}
	return s.Store.UpdateAgentInfo(ctx, id, info)
}

func (s failingCreateMetadataStore) SetProvenance(ctx context.Context, id string, origin sessionstore.Origin, owner, label string) error {
	if s.failProvenance {
		return errors.New("provenance metadata unavailable")
	}
	return s.Store.SetProvenance(ctx, id, origin, owner, label)
}

func TestAdapterCreateFailsClosedWhenEnrichmentPersistenceFails(t *testing.T) {
	for _, tc := range []struct {
		name           string
		failAgentInfo  bool
		failProvenance bool
		input          CreateInput
	}{
		{name: "agent metadata", failAgentInfo: true, input: CreateInput{AgentType: "codex"}},
		{name: "provenance", failProvenance: true, input: CreateInput{Origin: "ui", Owner: "operator"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			base := sessionstore.NewInMemory()
			store := failingCreateMetadataStore{Store: base, failAgentInfo: tc.failAgentInfo, failProvenance: tc.failProvenance}
			mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
			mgr.SetStore(store)
			a := &Adapter{Manager: mgr, Store: store}

			if _, err := a.Create(ctx, tc.input); err == nil {
				t.Fatal("create succeeded after enrichment persistence failed")
			}
			if got := len(mgr.List()); got != 0 {
				t.Fatalf("runtime sessions after failed create = %d, want 0", got)
			}
			if rows, err := base.List(ctx); err != nil {
				t.Fatal(err)
			} else if len(rows) != 0 {
				t.Fatalf("metadata rows after failed create = %+v, want none", rows)
			}
		})
	}
}

type adapterCatalogStub struct {
	records []continuity.CatalogRecord
	err     error
}

func (s adapterCatalogStub) EnqueueTombstone(context.Context, string) error { return nil }
func (s adapterCatalogStub) List(context.Context, string, int) ([]continuity.CatalogRecord, bool, error) {
	return s.records, false, s.err
}

func (s adapterCatalogStub) ListAll(ctx context.Context, state string) ([]continuity.CatalogRecord, error) {
	records, _, err := s.List(ctx, state, 0)
	return records, err
}

func TestAdapterLiveArchiveAndPolicyLifecycle(t *testing.T) {
	ctx := context.Background()
	mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
	store := sessionstore.NewInMemory()
	a := &Adapter{
		Manager: mgr, Store: store, Events: events.NewLogger(20), Metrics: metrics.New(),
		Conversations: adapterConversationStub{}, Workspace: workspace.NewMemStore(),
		AgentHistoryPresent: func(sessionstore.Metadata) bool { return true },
		RetentionPolicy:     func() ArchiveRetentionPolicy { return ArchiveRetentionPolicy{MaxBytes: 64} },
		AgentHistorySize:    func(sessionstore.Metadata) (int64, error) { return 32, nil },
		PruneAgentHistory:   func(sessionstore.Metadata) (int64, error) { return 32, nil },
	}

	created, err := a.Create(ctx, CreateInput{Shell: "/bin/sh", Cols: 80, Rows: 24, Backend: "standard", Origin: "ui", Owner: "test", DisplayLabel: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Delete(ctx, created.ID) }()
	if created.ID == "" || a.Metrics.SessionsCreated.Load() != 1 {
		t.Fatalf("created = %+v", created)
	}
	if err := store.Save(ctx, sessionstore.Metadata{ID: created.ID, Backend: backend.Standard, Shell: "/bin/sh", Created: time.Now().UTC(), Origin: sessionstore.OriginUI, Owner: "test"}); err != nil {
		t.Fatal(err)
	}
	if listed, err := a.List(ctx); err != nil || len(listed) != 1 || listed[0].Owner != "test" {
		t.Fatalf("listed = %+v, err=%v", listed, err)
	}
	if got, err := a.Get(ctx, created.ID); err != nil || got.ID != created.ID {
		t.Fatalf("get = %+v, err=%v", got, err)
	}
	if view, err := a.GetPolicy(ctx, created.ID); err != nil || view.SessionID != created.ID {
		t.Fatalf("policy view = %+v, err=%v", view, err)
	}
	if view, err := a.UpdatePolicy(ctx, created.ID, Policy{Mode: "preset", Duration: "1h"}); err != nil || !view.HasExpiry {
		t.Fatalf("updated policy = %+v, err=%v", view, err)
	}
	if _, err := a.UpdatePolicy(ctx, created.ID, Policy{Mode: "invalid"}); err == nil {
		t.Fatal("invalid policy was accepted")
	}

	archivedID := "archived"
	if err := store.Save(ctx, sessionstore.Metadata{ID: archivedID, Backend: backend.Persistent, Status: sessionstore.StatusDismissed, AgentType: sessionstore.AgentCodex, Created: time.Now().UTC(), ArchivedAt: time.Now().UTC(), LastRolloutPath: "/tmp/history"}); err != nil {
		t.Fatal(err)
	}
	if rows, err := a.ListArchived(ctx); err != nil || len(rows) != 1 || rows[0].ID != archivedID {
		t.Fatalf("archived = %+v, err=%v", rows, err)
	}
	if snap, err := a.GetArchiveRetention(ctx); err != nil || snap.Stats.EntryCount != 1 {
		t.Fatalf("retention = %+v, err=%v", snap, err)
	}
	if dry, err := a.PruneArchive(ctx, false); err != nil || len(dry.Actions) == 0 || !dry.DryRun {
		t.Fatalf("prune dry run = %+v, err=%v", dry, err)
	}
	if err := a.Archive(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
}

func TestAdapterPruneUsesDistinctChildReceipts(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE session_lifecycle_receipts (
		operation_id TEXT PRIMARY KEY, session_id TEXT NOT NULL, actor_kind TEXT NOT NULL,
		actor_id TEXT NOT NULL, command TEXT NOT NULL, from_state TEXT NOT NULL,
		to_state TEXT NOT NULL, reason_code TEXT NOT NULL, status TEXT NOT NULL,
		error_code TEXT NOT NULL, created_at TEXT NOT NULL, completed_at TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithOperationID(context.Background(), "prune-operation")
	store := sessionstore.NewInMemory()
	archivedAt := time.Now().UTC().Add(-2 * time.Hour)
	for _, id := range []string{"prune-a", "prune-b"} {
		if err := store.Save(ctx, sessionstore.Metadata{ID: id, Status: sessionstore.StatusDismissed, ArchivedAt: archivedAt}); err != nil {
			t.Fatal(err)
		}
	}
	a := &Adapter{
		Store: store, Conversations: emptyConversationStub{}, LifecycleLedger: continuity.NewSQLLedger(db),
		Events: events.NewLogger(10), Metrics: metrics.New(),
		RetentionPolicy: func() ArchiveRetentionPolicy { return ArchiveRetentionPolicy{MessageLessAge: time.Hour} },
		Now:             func() time.Time { return time.Now().UTC() },
	}
	result, err := a.PruneArchive(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Actions) != 2 {
		t.Fatalf("prune actions = %+v", result.Actions)
	}
	ledger := continuity.NewSQLLedger(db)
	for _, id := range []string{"prune-a", "prune-b"} {
		if _, err := store.Get(ctx, id); err == nil {
			t.Fatalf("pruned session %q still exists", id)
		}
		receipt, err := ledger.Get(ctx, "prune-operation:delete:"+id)
		if err != nil {
			t.Fatal(err)
		}
		if receipt.Status != "succeeded" || receipt.Command != "delete" {
			t.Fatalf("child receipt %q = %+v", id, receipt)
		}
	}
}

func TestAdapterAgentHistoryPruneWritesRetentionReceipt(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE session_lifecycle_receipts (
		operation_id TEXT PRIMARY KEY, session_id TEXT NOT NULL, actor_kind TEXT NOT NULL,
		actor_id TEXT NOT NULL, command TEXT NOT NULL, from_state TEXT NOT NULL,
		to_state TEXT NOT NULL, reason_code TEXT NOT NULL, status TEXT NOT NULL,
		error_code TEXT NOT NULL, created_at TEXT NOT NULL, completed_at TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithOperationID(context.Background(), "home-retention-operation")
	store := sessionstore.NewInMemory()
	if err := store.Save(ctx, sessionstore.Metadata{
		ID: "home-retention", Status: sessionstore.StatusDismissed,
		AgentType: sessionstore.AgentCodex, ArchivedAt: time.Now().UTC().Add(-2 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{
		Store: store, LifecycleLedger: continuity.NewSQLLedger(db), Metrics: metrics.New(),
		RetentionPolicy:   func() ArchiveRetentionPolicy { return ArchiveRetentionPolicy{AgentHomeAge: time.Hour} },
		AgentHistorySize:  func(sessionstore.Metadata) (int64, error) { return 12, nil },
		PruneAgentHistory: func(sessionstore.Metadata) (int64, error) { return 12, nil },
	}
	result, err := a.PruneArchive(ctx, true)
	if err != nil || len(result.Actions) != 1 || !result.Actions[0].Applied {
		t.Fatalf("home prune = %+v, err=%v", result, err)
	}
	receipt, err := continuity.NewSQLLedger(db).Get(ctx, "home-retention-operation:agent-home:home-retention")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Command != "retention_agent_home" || receipt.Status != "succeeded" {
		t.Fatalf("retention receipt = %+v", receipt)
	}
}

func TestAdapterArchiveWritesReplaySafeReceipt(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE session_lifecycle_receipts (
		operation_id TEXT PRIMARY KEY, session_id TEXT NOT NULL, actor_kind TEXT NOT NULL,
		actor_id TEXT NOT NULL, command TEXT NOT NULL, from_state TEXT NOT NULL,
		to_state TEXT NOT NULL, reason_code TEXT NOT NULL, status TEXT NOT NULL,
		error_code TEXT NOT NULL, created_at TEXT NOT NULL, completed_at TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithOperationID(context.Background(), "archive-receipt-1")
	mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
	store := sessionstore.NewInMemory()
	a := &Adapter{Manager: mgr, Store: store, Metrics: metrics.New(), Events: events.NewLogger(5), LifecycleLedger: continuity.NewSQLLedger(db)}
	created, err := a.Create(context.Background(), CreateInput{Shell: "/bin/sh", Cols: 80, Rows: 24, Backend: "standard"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), sessionstore.Metadata{ID: created.ID, Backend: backend.Standard, Shell: "/bin/sh", Created: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if err := a.Archive(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if err := a.Archive(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	receipt, err := continuity.NewSQLLedger(db).Get(context.Background(), "archive-receipt-1")
	if err != nil || receipt.Status != "succeeded" {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
}

func TestAdapterFailedLifecycleReplayDoesNotReportSuccess(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE session_lifecycle_receipts (
		operation_id TEXT PRIMARY KEY, session_id TEXT NOT NULL, actor_kind TEXT NOT NULL,
		actor_id TEXT NOT NULL, command TEXT NOT NULL, from_state TEXT NOT NULL,
		to_state TEXT NOT NULL, reason_code TEXT NOT NULL, status TEXT NOT NULL,
		error_code TEXT NOT NULL, created_at TEXT NOT NULL, completed_at TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	created := time.Now().UTC()
	if _, err := db.Exec(`INSERT INTO session_lifecycle_receipts
		(operation_id, session_id, actor_kind, actor_id, command, from_state, to_state,
		reason_code, status, error_code, created_at, completed_at)
		VALUES (?, ?, 'operator', '', 'archive', 'live', 'archived', 'archive', 'failed', 'lifecycle_operation_failed', ?, ?)`,
		"archive-failed-replay", "session-1", created.Format(time.RFC3339Nano), created.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{LifecycleLedger: continuity.NewSQLLedger(db), Metrics: metrics.New(), Events: events.NewLogger(5)}
	// The retry runs (a recorded failure must not brick the command) but it
	// still cannot succeed: this adapter has neither a store nor a manager, so
	// the second attempt fails on its own merits and records that.
	if err := a.Archive(WithOperationID(context.Background(), "archive-failed-replay"), "session-1"); err == nil {
		t.Fatal("archive reported success for a session that exists nowhere")
	}
	receipt, err := continuity.NewSQLLedger(db).Get(context.Background(), "archive-failed-replay")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "failed" {
		t.Fatalf("receipt status = %q, want failed", receipt.Status)
	}
}

// A recorded failure describes one attempt, not a verdict on every future one.
// The web UI derives the archive operation id from the session id alone, so
// replaying the failure made the operator's Close button permanently dead for
// that session.
func TestAdapterFailedArchiveReceiptIsRetryable(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE session_lifecycle_receipts (
		operation_id TEXT PRIMARY KEY, session_id TEXT NOT NULL, actor_kind TEXT NOT NULL,
		actor_id TEXT NOT NULL, command TEXT NOT NULL, from_state TEXT NOT NULL,
		to_state TEXT NOT NULL, reason_code TEXT NOT NULL, status TEXT NOT NULL,
		error_code TEXT NOT NULL, created_at TEXT NOT NULL, completed_at TEXT)`); err != nil {
		t.Fatal(err)
	}
	id := "retryable-archive"
	failedAt := time.Now().UTC()
	if _, err := db.Exec(`INSERT INTO session_lifecycle_receipts
		(operation_id, session_id, actor_kind, actor_id, command, from_state, to_state,
		reason_code, status, error_code, created_at, completed_at)
		VALUES (?, ?, 'operator', '', 'archive', 'live', 'archived', 'archive', 'failed', 'lifecycle_operation_failed', ?, ?)`,
		"archive:"+id, id, failedAt.Format(time.RFC3339Nano), failedAt.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	store := sessionstore.NewInMemory()
	if err := store.Save(context.Background(), sessionstore.Metadata{
		ID: id, Backend: backend.Persistent, Shell: "/bin/sh", Created: failedAt.Add(-time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{Store: store, Metrics: metrics.New(), Events: events.NewLogger(5), LifecycleLedger: continuity.NewSQLLedger(db)}
	if err := a.Archive(WithOperationID(context.Background(), "archive:"+id), id); err != nil {
		t.Fatalf("archive after a recorded failure: %v", err)
	}
	meta, err := store.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if meta.ArchivedAt.IsZero() {
		t.Fatalf("archive marker missing after retry: %+v", meta)
	}
	receipt, err := continuity.NewSQLLedger(db).Get(context.Background(), "archive:"+id)
	if err != nil || receipt.Status != "succeeded" {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
}

// A live session can outlive its metadata row (a routed test lease writes its
// rows to a database that is later discarded while the PTY keeps running).
// The sidebar still lists it, so Close has to work on it.
func TestAdapterArchiveClosesLiveSessionWithoutMetadataRow(t *testing.T) {
	ctx := context.Background()
	store := sessionstore.NewInMemory()
	mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
	a := &Adapter{Manager: mgr, Store: store, Metrics: metrics.New(), Events: events.NewLogger(5)}
	created, err := a.Create(ctx, CreateInput{Shell: "/bin/sh", Cols: 80, Rows: 24, Backend: "standard"})
	if err != nil {
		t.Fatal(err)
	}
	// Lose the row the way a discarded routed database does: the process keeps
	// running, and List keeps showing it.
	if err := store.Delete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if err := a.Archive(WithOperationID(ctx, "archive:"+created.ID), created.ID); err != nil {
		t.Fatalf("archive live session with no metadata row: %v", err)
	}
	meta, err := store.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("archive left no durable row: %v", err)
	}
	if meta.ArchivedAt.IsZero() {
		t.Fatalf("archive marker missing: %+v", meta)
	}
	if _, live := mgr.Get(created.ID); live {
		t.Fatal("archived session is still running")
	}
}

// Nothing to close is still an error: a session that exists in neither the
// manager nor the store must not report a successful archive.
func TestAdapterArchiveUnknownSessionStillNotFound(t *testing.T) {
	a := &Adapter{
		Manager: session.NewManagerWithFactory(ptyfake.NewFactory()),
		Store:   sessionstore.NewInMemory(),
		Metrics: metrics.New(),
		Events:  events.NewLogger(5),
	}
	err := a.Archive(WithOperationID(context.Background(), "archive:ghost"), "ghost")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("archive of an unknown session = %v, want ErrNotFound", err)
	}
}

func TestAdapterArchivePreservesDurableStateWhenManagerUnavailable(t *testing.T) {
	store := sessionstore.NewInMemory()
	id := "durable-without-manager"
	created := time.Now().UTC().Add(-time.Minute)
	if err := store.Save(context.Background(), sessionstore.Metadata{
		ID: id, Backend: backend.Persistent, Shell: "/bin/sh", Created: created,
	}); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{Store: store, Metrics: metrics.New(), Events: events.NewLogger(5)}
	if err := a.Archive(WithOperationID(context.Background(), "archive-without-manager"), id); err != nil {
		t.Fatal(err)
	}
	meta, err := store.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if meta.ArchivedAt.IsZero() {
		t.Fatalf("archive marker missing: %+v", meta)
	}
}

func TestAdapterRecoverRefusalWritesReceipt(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE session_lifecycle_receipts (
		operation_id TEXT PRIMARY KEY, session_id TEXT NOT NULL, actor_kind TEXT NOT NULL,
		actor_id TEXT NOT NULL, command TEXT NOT NULL, from_state TEXT NOT NULL,
		to_state TEXT NOT NULL, reason_code TEXT NOT NULL, status TEXT NOT NULL,
		error_code TEXT NOT NULL, created_at TEXT NOT NULL, completed_at TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithOperationID(context.Background(), "recover-refusal-1")
	store := sessionstore.NewInMemory()
	oldID := "plain-shell"
	if err := store.Save(ctx, sessionstore.Metadata{
		ID: oldID, Backend: backend.Persistent, Shell: "/bin/bash", Status: sessionstore.StatusDismissed,
		ArchivedAt: time.Now().UTC(), Created: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{
		Store: store, Conversations: adapterConversationStub{}, LifecycleLedger: continuity.NewSQLLedger(db),
	}
	if _, err := a.Recover(ctx, RecoverInput{ID: oldID, IdempotencyKey: "recover-refusal-1"}); !errors.Is(err, ErrFailedPrecondition) {
		t.Fatalf("recover refusal = %v", err)
	}
	receipt, err := continuity.NewSQLLedger(db).Get(ctx, "recover-refusal-1")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "failed" || receipt.FromState != continuity.StateArchived || receipt.ToState != continuity.StateLive {
		t.Fatalf("refusal receipt = %+v", receipt)
	}
}

func TestAdapterRecoverArchivedWritesArchivedSourceReceipt(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE session_lifecycle_receipts (
		operation_id TEXT PRIMARY KEY, session_id TEXT NOT NULL, actor_kind TEXT NOT NULL,
		actor_id TEXT NOT NULL, command TEXT NOT NULL, from_state TEXT NOT NULL,
		to_state TEXT NOT NULL, reason_code TEXT NOT NULL, status TEXT NOT NULL,
		error_code TEXT NOT NULL, created_at TEXT NOT NULL, completed_at TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithOperationID(context.Background(), "recover-archived-1")
	store := sessionstore.NewInMemory()
	mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
	mgr.SetStore(store)
	a := &Adapter{
		Manager: mgr, Store: store,
		Conversations: adapterConversationStub{}, LifecycleLedger: continuity.NewSQLLedger(db),
		Events: events.NewLogger(5), Metrics: metrics.New(),
		AgentHistoryPresent: func(sessionstore.Metadata) bool { return true },
	}
	created, err := a.Create(ctx, CreateInput{Shell: "/bin/sh", Cols: 80, Rows: 24, Backend: "persistent", AgentType: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	oldID := created.ID
	if err := store.UpdateAgentInfo(ctx, oldID, sessionstore.AgentInfo{AgentType: sessionstore.AgentCodex, AgentSessionID: "codex-session-1"}); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkArchived(ctx, oldID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	result, err := a.Recover(ctx, RecoverInput{ID: oldID, IdempotencyKey: "recover-archived-1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.NewSessionID == "" {
		t.Fatalf("recovery result = %+v", result)
	}
	receipt, err := continuity.NewSQLLedger(db).Get(ctx, "recover-archived-1")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "succeeded" || receipt.FromState != continuity.StateArchived || receipt.ToState != continuity.StateLive {
		t.Fatalf("archived recovery receipt = %+v", receipt)
	}
	if receipt.ActorID != result.NewSessionID {
		t.Fatalf("recovered session missing from receipt = %+v", receipt)
	}
	// A fresh adapter has no in-memory idempotency cache and the source row was
	// dismissed by recovery. The durable receipt must still replay the minted
	// replacement instead of attempting a second recovery.
	fresh := &Adapter{Store: store, LifecycleLedger: continuity.NewSQLLedger(db)}
	replayed, err := fresh.Recover(ctx, RecoverInput{ID: oldID, IdempotencyKey: "recover-archived-1"})
	if err != nil {
		t.Fatalf("durable recovery replay: %v", err)
	}
	if replayed.NewSessionID != result.NewSessionID || replayed.OldSessionID != oldID {
		t.Fatalf("durable recovery replay = %+v, want replacement %q", replayed, result.NewSessionID)
	}
}

func TestAdapterLifecycleRefusalsWriteReceipts(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE session_lifecycle_receipts (
		operation_id TEXT PRIMARY KEY, session_id TEXT NOT NULL, actor_kind TEXT NOT NULL,
		actor_id TEXT NOT NULL, command TEXT NOT NULL, from_state TEXT NOT NULL,
		to_state TEXT NOT NULL, reason_code TEXT NOT NULL, status TEXT NOT NULL,
		error_code TEXT NOT NULL, created_at TEXT NOT NULL, completed_at TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	store := sessionstore.NewInMemory()
	if err := store.Save(context.Background(), sessionstore.Metadata{ID: "live-refusal", Status: sessionstore.StatusLive}); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{Store: store, LifecycleLedger: continuity.NewSQLLedger(db), Events: events.NewLogger(5), Metrics: metrics.New()}
	if err := a.Unarchive(WithOperationID(context.Background(), "unarchive-refusal-1"), "live-refusal"); !errors.Is(err, ErrFailedPrecondition) {
		t.Fatalf("unarchive refusal = %v", err)
	}
	if err := a.DismissRecoverable(WithOperationID(context.Background(), "dismiss-refusal-1"), "live-refusal"); !errors.Is(err, ErrFailedPrecondition) {
		t.Fatalf("dismiss refusal = %v", err)
	}
	ledger := continuity.NewSQLLedger(db)
	for _, operationID := range []string{"unarchive-refusal-1", "dismiss-refusal-1"} {
		receipt, err := ledger.Get(context.Background(), operationID)
		if err != nil {
			t.Fatal(err)
		}
		if receipt.Status != "failed" {
			t.Fatalf("refusal receipt %q = %+v", operationID, receipt)
		}
	}
	if err := a.Archive(WithOperationID(context.Background(), "archive-missing-1"), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("archive missing = %v", err)
	}
	if _, err := a.Recover(WithOperationID(context.Background(), "recover-missing-1"), RecoverInput{ID: "missing"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("recover missing = %v", err)
	}
	for _, operationID := range []string{"archive-missing-1", "recover-missing-1"} {
		receipt, err := ledger.Get(context.Background(), operationID)
		if err != nil {
			t.Fatal(err)
		}
		if receipt.Status != "failed" {
			t.Fatalf("missing-row receipt %q = %+v", operationID, receipt)
		}
	}
	archivedID := "archived-delete"
	if err := store.Save(context.Background(), sessionstore.Metadata{
		ID: archivedID, Status: sessionstore.StatusDismissed, ArchivedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.Delete(WithOperationID(context.Background(), "delete-archived-1"), archivedID); err != nil {
		t.Fatalf("archived delete = %v", err)
	}
	receipt, err := ledger.Get(context.Background(), "delete-archived-1")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "succeeded" || receipt.FromState != continuity.StateArchived || receipt.ToState != continuity.StateDeleted {
		t.Fatalf("archived delete receipt = %+v", receipt)
	}
}

func TestAdapterSuccessfulLifecycleReplaySurvivesStateMutation(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE session_lifecycle_receipts (
		operation_id TEXT PRIMARY KEY, session_id TEXT NOT NULL, actor_kind TEXT NOT NULL,
		actor_id TEXT NOT NULL, command TEXT NOT NULL, from_state TEXT NOT NULL,
		to_state TEXT NOT NULL, reason_code TEXT NOT NULL, status TEXT NOT NULL,
		error_code TEXT NOT NULL, created_at TEXT NOT NULL, completed_at TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	store := sessionstore.NewInMemory()
	archivedID := "replay-unarchive"
	if err := store.Save(context.Background(), sessionstore.Metadata{ID: archivedID, Status: sessionstore.StatusDismissed, ArchivedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	dismissID := "replay-dismiss"
	if err := store.Save(context.Background(), sessionstore.Metadata{ID: dismissID, Status: sessionstore.StatusAwaitingRecovery}); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{Store: store, LifecycleLedger: continuity.NewSQLLedger(db), Events: events.NewLogger(5), Metrics: metrics.New()}
	if err := a.Unarchive(WithOperationID(context.Background(), "unarchive-replay"), archivedID); err != nil {
		t.Fatal(err)
	}
	if err := a.Unarchive(WithOperationID(context.Background(), "unarchive-replay"), archivedID); err != nil {
		t.Fatalf("unarchive replay: %v", err)
	}
	if err := a.DismissRecoverable(WithOperationID(context.Background(), "dismiss-replay"), dismissID); err != nil {
		t.Fatal(err)
	}
	if err := a.DismissRecoverable(WithOperationID(context.Background(), "dismiss-replay"), dismissID); err != nil {
		t.Fatalf("dismiss replay: %v", err)
	}
}

func TestAdapterDeleteSurfacesConversationCleanupFailure(t *testing.T) {
	ctx := context.Background()
	mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
	store := sessionstore.NewInMemory()
	a := &Adapter{
		Manager: mgr, Store: store, Metrics: metrics.New(), Events: events.NewLogger(5),
		Conversations: adapterConversationErrorStub{}, Workspace: workspace.NewMemStore(),
	}
	created, err := a.Create(ctx, CreateInput{Shell: "/bin/sh", Cols: 80, Rows: 24, Backend: "standard"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, sessionstore.Metadata{ID: created.ID, Backend: backend.Standard, Shell: "/bin/sh", Created: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if err := a.Delete(ctx, created.ID); err == nil {
		t.Fatal("conversation cleanup failure was suppressed")
	}
	if _, err := store.Get(ctx, created.ID); err != nil {
		t.Fatalf("metadata disappeared after partial delete failure: %v", err)
	}
}

func TestAdapterListArchivedIncludesCatalogOnlyEvidence(t *testing.T) {
	ctx := context.Background()
	created := time.Now().UTC().Add(-time.Minute)
	a := &Adapter{
		Store:         sessionstore.NewInMemory(),
		Conversations: adapterConversationStub{},
		ContinuityCatalog: adapterCatalogStub{records: []continuity.CatalogRecord{{
			SessionID: "catalog-only", LifecycleState: continuity.StateRecoverable,
			AgentType: "codex", CurrentTitle: "Recovered catalog evidence",
			CreatedAt: created, LastActivityAt: created,
		}}},
	}
	rows, err := a.ListArchived(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != "catalog-only" {
		t.Fatalf("catalog-only archive rows = %+v", rows)
	}
	if rows[0].RestoreState != RestoreStateReadOnly || rows[0].MessageCount != 2 {
		t.Fatalf("catalog-only restore projection = %+v", rows[0])
	}
}

func TestAdapterListArchivedSurfacesCatalogFailure(t *testing.T) {
	a := &Adapter{
		Store:             sessionstore.NewInMemory(),
		ContinuityCatalog: adapterCatalogStub{err: context.DeadlineExceeded},
	}
	if _, err := a.ListArchived(context.Background()); err == nil {
		t.Fatal("continuity catalog failure was suppressed")
	} else if !errors.Is(err, ErrInternal) {
		t.Fatalf("catalog failure = %v, want ErrInternal", err)
	}
}

func TestAdapterRecoverableListingAndDismissal(t *testing.T) {
	ctx := context.Background()
	store := sessionstore.NewInMemory()
	if err := store.Save(ctx, sessionstore.Metadata{ID: "recoverable", Status: sessionstore.StatusAwaitingRecovery, Detached: true, AgentType: sessionstore.AgentCodex, Created: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{Store: store, Manager: emptySessionManager{}, Workspace: workspace.NewMemStore()}
	rows, err := a.ListRecoverable(ctx)
	if err != nil || len(rows) != 1 || !rows[0].Recoverable {
		t.Fatalf("recoverable = %+v, err=%v", rows, err)
	}
	if err := a.DismissRecoverable(ctx, "recoverable"); err != nil {
		t.Fatal(err)
	}
	if err := a.DismissRecoverable(ctx, "recoverable"); err == nil {
		t.Fatal("dismissed session was dismissed twice")
	}
}

func TestAdapterCreateUsesCanonicalOwnerAndDropsInvalidUIOwner(t *testing.T) {
	mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
	store := sessionstore.NewInMemory()
	a := &Adapter{Manager: mgr, Store: store, Events: events.NewLogger(20), Metrics: metrics.New()}
	p := identity.Principal{Kind: identity.ActorHuman, Subject: "verified-owner", Verified: true, Source: identity.SourceCloudflareAccess}
	ctx := identity.WithPrincipal(context.Background(), p)
	created, err := a.Create(ctx, CreateInput{Shell: "/bin/sh", Backend: "standard", Origin: "ui", Owner: "forged-owner"})
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Delete(context.Background(), created.ID)
	if created.Owner != p.Subject {
		t.Fatalf("owner = %q", created.Owner)
	}
	invalid := identity.WithStatus(context.Background(), identity.Status{FailureClass: identity.FailureInvalid})
	unowned, err := a.Create(invalid, CreateInput{Shell: "/bin/sh", Backend: "standard", Origin: "ui", Owner: "forged-owner"})
	if err != nil || unowned.Owner != "" {
		t.Fatalf("invalid proof should create only an unattributed UI session: session=%+v, err=%v", unowned, err)
	}
	defer mgr.Delete(context.Background(), unowned.ID)
	if _, err := a.Create(invalid, CreateInput{Shell: "/bin/sh", Backend: "standard", Origin: "programmatic"}); !errors.Is(err, ErrFailedPrecondition) {
		t.Fatalf("invalid programmatic create = %v, want failed precondition", err)
	}
}

func TestAdapterCreateUIContinuesWhenOwnerVerifierUnavailable(t *testing.T) {
	mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
	store := sessionstore.NewInMemory()
	a := &Adapter{Manager: mgr, Store: store, Events: events.NewLogger(20), Metrics: metrics.New()}
	ctx := identity.WithStatus(context.Background(), identity.Status{FailureClass: identity.FailureUnavailable})
	created, err := a.Create(ctx, CreateInput{Shell: "/bin/sh", Backend: "standard", Origin: "ui", Owner: "forged-owner"})
	if err != nil {
		t.Fatalf("UI session creation should continue without owner verification: %v", err)
	}
	defer mgr.Delete(context.Background(), created.ID)
	if created.Owner != "" {
		t.Fatalf("unverified owner was retained: %q", created.Owner)
	}
	if row, err := store.Get(context.Background(), created.ID); err != nil || row.Owner != "" {
		t.Fatalf("durable unverified owner = %q, err=%v", row.Owner, err)
	}
	if _, err := a.Create(ctx, CreateInput{Shell: "/bin/sh", Backend: "standard", Origin: "programmatic"}); !errors.Is(err, ErrFailedPrecondition) {
		t.Fatalf("programmatic create with unavailable verification = %v, want failed precondition", err)
	}
}

func TestAdapterReservedCreateUsesDurableIdentityAndLiveRuntime(t *testing.T) {
	ctx := t.Context()
	store := sessionstore.NewInMemory()
	mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
	mgr.SetStore(store)
	a := &Adapter{Manager: mgr, Store: store, Events: events.NewLogger(20), Metrics: metrics.New()}
	id := uuid.NewString()
	in := CreateInput{RequestedSessionID: id, Shell: "/bin/sh", Backend: "standard", Owner: "agent-manager", Origin: "programmatic", LaunchCommand: "bounded", DisplayLabel: "same run"}
	first, err := a.Create(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Delete(context.Background(), id)
	row, err := store.Get(ctx, id)
	if err != nil || row.CreateFingerprint == "" || row.Owner != "agent-manager" || first.ID != id {
		t.Fatalf("durable=%+v first=%+v err=%v", row, first, err)
	}
	// A second adapter uses the same production manager/store; no second spawn.
	other := &Adapter{Manager: mgr, Store: store}
	retry, err := other.Create(ctx, in)
	if err != nil || retry.ID != id || len(mgr.List()) != 1 {
		t.Fatalf("retry=%+v err=%v runtimes=%d", retry, err, len(mgr.List()))
	}
	changed := in
	changed.LaunchCommand = "changed"
	if _, err := other.Create(ctx, changed); err == nil {
		t.Fatal("changed request adopted identity")
	}
	if len(mgr.List()) != 1 {
		t.Fatal("collision launched another runtime")
	}
	// A restarted owner with metadata alone must not report a live launch.
	restarted := session.NewManagerWithFactory(ptyfake.NewFactory())
	restarted.SetStore(store)
	afterRestart := &Adapter{Manager: restarted, Store: store}
	if _, err := afterRestart.Create(ctx, in); err == nil {
		t.Fatal("metadata without recovered runtime was adopted")
	}
	if len(restarted.List()) != 0 {
		t.Fatal("restart recreated ambiguous identity")
	}
	if err := store.MarkDismissed(ctx, id, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := other.Create(ctx, in); err == nil {
		t.Fatal("archived identity was adopted")
	}
	if _, err := other.Get(ctx, id); err == nil {
		t.Fatal("archived metadata and surviving runtime exposed recovery proof")
	}
}

type refusedReservedSave struct{ sessionstore.Store }

func (s refusedReservedSave) Save(context.Context, sessionstore.Metadata) error {
	return errors.New("reserved owner save refused")
}

func TestAdapterReservedPersistenceFailureAndCancellation(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelled), func(t *testing.T) {
			base := sessionstore.NewInMemory()
			var store sessionstore.Store = refusedReservedSave{base}
			if cancelled {
				store = base
			}
			mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
			mgr.SetStore(store)
			a := &Adapter{Manager: mgr, Store: store, Events: events.NewLogger(20), Metrics: metrics.New()}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if cancelled {
				cancel()
			}
			id := uuid.NewString()
			_, err := a.Create(ctx, CreateInput{RequestedSessionID: id, Shell: "/bin/sh", Backend: "standard", Owner: "agent-manager", Origin: "programmatic"})
			if err == nil || len(mgr.List()) != 0 {
				t.Fatalf("create err=%v runtime=%d", err, len(mgr.List()))
			}
			rows, err := base.List(t.Context())
			if err != nil || len(rows) != 0 {
				t.Fatalf("rows=%+v err=%v", rows, err)
			}
		})
	}
}

func TestStableCreateCapabilityPinsCanonicalVerifiedOwner(t *testing.T) {
	store := sessionstore.NewInMemory()
	mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
	mgr.SetStore(store)
	a := &Adapter{Manager: mgr, Store: store, Events: events.NewLogger(20), Metrics: metrics.New()}
	h := NewConnectHandler(Deps{Service: a})
	principal := identity.Principal{Kind: identity.ActorService, Subject: "lawful-service-owner", Verified: true, Source: identity.SourceCloudflareAccess}
	ctx := identity.WithPrincipal(t.Context(), principal)
	caps, err := h.List(ctx, connect.NewRequest(&sessionsv1.ListRequest{}))
	if err != nil || !caps.Msg.SupportsStableCreateIdentity || caps.Msg.StableCreateOwner != principal.OwnerSubject() {
		t.Fatalf("capabilities=%+v err=%v", caps, err)
	}
	id := uuid.NewString()
	req := connect.NewRequest(&sessionsv1.CreateRequest{RequestedSessionId: id, Shell: "/bin/sh", Backend: "standard", Owner: "forged", Origin: sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC})
	created, err := h.Create(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Delete(context.Background(), id)
	got, err := h.Get(ctx, connect.NewRequest(&sessionsv1.GetRequest{Id: id}))
	if err != nil || got.Msg.Session.Owner != caps.Msg.StableCreateOwner || created.Msg.Session.Owner != caps.Msg.StableCreateOwner || got.Msg.Session.CreateFingerprint == "" || got.Msg.Session.CreateFingerprint != created.Msg.Session.CreateFingerprint {
		t.Fatalf("created=%+v got=%+v err=%v", created, got, err)
	}
	anonymous, err := h.List(t.Context(), connect.NewRequest(&sessionsv1.ListRequest{}))
	if err != nil || anonymous.Msg.StableCreateOwner != "" {
		t.Fatalf("anonymous capability=%+v err=%v", anonymous, err)
	}
}
