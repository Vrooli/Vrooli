package continuity

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestPlanReconciliationIsStableWithLegacyAliasCollisions(t *testing.T) {
	existing := map[string]CatalogRecord{
		"session-b": {SessionID: "session-b", Aliases: []Alias{{Kind: "agent_session", Value: "native"}}, SourceFingerprint: "sha256:b"},
		"session-a": {SessionID: "session-a", Aliases: []Alias{{Kind: "agent_session", Value: "native"}}, SourceFingerprint: "sha256:a"},
	}
	observations := []Evidence{{SessionID: "new", AgentSessionID: "native", CreatedAt: time.Unix(1, 0), LastActivityAt: time.Unix(1, 0)}}
	var expected string
	for i := 0; i < 20; i++ {
		items, err := PlanReconciliation(observations, existing)
		if err != nil {
			t.Fatal(err)
		}
		got := ManifestHash("generation", items)
		if i == 0 {
			expected = got
		} else if got != expected {
			t.Fatalf("manifest changed across identical plans: first=%s current=%s", expected, got)
		}
	}
}

type catalogTestStore struct {
	records []CatalogRecord
	aliases int
}

func (s *catalogTestStore) Upsert(_ context.Context, record CatalogRecord) error {
	s.records = append(s.records, record)
	return nil
}

func (s *catalogTestStore) AddAliases(_ context.Context, record CatalogRecord) error {
	s.aliases += len(record.Aliases)
	return nil
}

func TestSQLCatalogObserveIncludesOrphanEvidence(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY, backend TEXT, agent_type TEXT, agent_session_id TEXT, cwd TEXT, display_label TEXT, last_rollout_path TEXT, created_at TEXT, last_activity_at TEXT)`,
		`CREATE TABLE conversation_sessions(session_id TEXT PRIMARY KEY)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT)`,
		`CREATE TABLE agent_transcript_checkpoints(source TEXT, source_key TEXT, web_console_session_id TEXT, updated_at TEXT)`,
		`CREATE TABLE continuity_publication_queue(session_id TEXT PRIMARY KEY, lifecycle_state TEXT, source_fingerprint TEXT, attempts INTEGER, last_error TEXT, published_fingerprint TEXT, updated_at TEXT)`,
		`CREATE TABLE workspace_panes(session_id TEXT PRIMARY KEY)`,
		`CREATE TABLE continuity_reconciliation_manifests(hash TEXT PRIMARY KEY, generation TEXT NOT NULL, items_json TEXT NOT NULL, previous_json TEXT NOT NULL DEFAULT '{}', created_at TEXT NOT NULL)`,
		`INSERT INTO sessions VALUES ('known','tmux','codex','agent-1','/tmp','Known','','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`,
		`INSERT INTO conversation_events VALUES ('event-1','orphan')`,
		`INSERT INTO agent_transcript_checkpoints VALUES ('codex','01a06a6b-88da-7422-b391-bb59c5f5e5e0','orphan','2026-09-03T23:17:07Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	evidence, err := NewSQLCatalogStore(db).Observe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence) != 2 || evidence[1].SessionID != "orphan" || !evidence[1].HasConversation {
		t.Fatalf("evidence=%+v", evidence)
	}
	if evidence[1].AgentSessionID != "01a06a6b-88da-7422-b391-bb59c5f5e5e0" {
		t.Fatalf("evidence lost native thread alias: %+v", evidence[1])
	}
	record, err := BuildCatalogRecord(evidence[1])
	if err != nil {
		t.Fatal(err)
	}
	if record.LifecycleState != StateRecoverable {
		t.Fatalf("record=%+v", record)
	}
	manifest := ReconciliationManifest{Hash: "sha256:test", Generation: "sha256:g", Items: []ReconcileItem{{Action: ActionCreate, Record: record}}}
	store := NewSQLCatalogStore(db)
	if err := store.SaveManifest(context.Background(), manifest); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadManifest(context.Background(), manifest.Hash)
	if err != nil || loaded.Generation != manifest.Generation || len(loaded.Items) != 1 {
		t.Fatalf("loaded manifest=%+v err=%v", loaded, err)
	}
}

func TestIncidentFixtureReconcilesWithoutLosingEvents(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY, backend TEXT, agent_type TEXT, agent_session_id TEXT, cwd TEXT, display_label TEXT, last_rollout_path TEXT, created_at TEXT, last_activity_at TEXT, archived_at TEXT, status TEXT)`,
		`CREATE TABLE conversation_sessions(session_id TEXT PRIMARY KEY)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT, sequence INTEGER, role TEXT, created_at TEXT, text TEXT)`,
		`CREATE VIRTUAL TABLE conversation_events_fts USING fts5(text, content='conversation_events', content_rowid='rowid')`,
		`CREATE TABLE agent_transcript_checkpoints(source TEXT, source_key TEXT, web_console_session_id TEXT, updated_at TEXT)`,
		`CREATE TABLE workspace_panes(session_id TEXT PRIMARY KEY)`,
		`CREATE TABLE conversation_catalog(session_id TEXT PRIMARY KEY, lifecycle_state TEXT, lifecycle_version INTEGER, backend TEXT, agent_type TEXT, agent_session_id TEXT, agent_home_ref TEXT, rollout_ref TEXT, original_title TEXT, current_title TEXT, topic_summary TEXT, cwd TEXT, created_at TEXT, last_activity_at TEXT, source_fingerprint TEXT)`,
		`CREATE TABLE conversation_aliases(session_id TEXT, alias_kind TEXT, alias_value TEXT, observed_at TEXT, PRIMARY KEY(alias_kind, alias_value))`,
		`CREATE TABLE continuity_publication_queue(session_id TEXT PRIMARY KEY, lifecycle_state TEXT, source_fingerprint TEXT, attempts INTEGER, last_error TEXT, published_fingerprint TEXT, updated_at TEXT)`,
		`CREATE TABLE continuity_reconciliation_manifests(hash TEXT PRIMARY KEY, generation TEXT, items_json TEXT, previous_json TEXT, created_at TEXT)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	const paneID = "a7e71c3c-e422-4c89-916a-03f92906fb89"
	const threadID = "01a06a6b-88da-7422-b391-bb59c5f5e5e0"
	const rolloutPath = "/state/sessions/codex/s-1/sessions/2026/09/03/rollout-2026-09-03T23-17-07-" + threadID + ".jsonl"
	if _, err := db.Exec(`INSERT INTO conversation_sessions VALUES (?)`, paneID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO agent_transcript_checkpoints VALUES ('codex', ?, ?, '2026-09-03T23:17:07Z')`, rolloutPath, paneID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspace_panes VALUES (?)`, paneID); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 88; i++ {
		text := "retained incident discussion"
		if i == 88 {
			text = "Plan Manager Agent Manager Git Control Tower continuity"
		}
		if _, err := db.Exec(`INSERT INTO conversation_events VALUES (?, ?, ?, 'assistant', ?, ?)`, fmt.Sprintf("incident-event-%02d", i), paneID, i, "2026-09-03T23:17:07Z", text); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO conversation_events_fts(rowid,text) SELECT rowid,text FROM conversation_events`); err != nil {
		t.Fatal(err)
	}

	auditor := NewIntegrityAuditor(db)
	before, err := auditor.Audit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if before.ConversationEvents != 88 || before.OrphanConversations != 1 || before.OrphanCheckpoints != 1 || before.OrphanWorkspacePanes != 1 {
		t.Fatalf("unexpected pre-reconciliation inventory: %+v", before)
	}

	store := NewSQLCatalogStore(db)
	evidence, err := store.Observe(context.Background())
	if err != nil || len(evidence) != 1 || evidence[0].SessionID != paneID || evidence[0].AgentSessionID != threadID {
		t.Fatalf("evidence=%+v err=%v", evidence, err)
	}
	record, err := BuildCatalogRecord(evidence[0])
	if err != nil {
		t.Fatal(err)
	}
	items, err := PlanReconciliation(evidence, map[string]CatalogRecord{})
	if err != nil || len(items) != 1 || items[0].Action != ActionCreate {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if items[0].Record.SourceFingerprint != record.SourceFingerprint {
		t.Fatalf("reconciliation changed deterministic record: item=%+v record=%+v", items[0].Record, record)
	}
	if _, err := ApplyReconciliation(context.Background(), store, items); err != nil {
		t.Fatal(err)
	}

	after, err := auditor.Audit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if after.ConversationEvents != before.ConversationEvents || after.EventContentHash != before.EventContentHash {
		t.Fatalf("reconciliation changed event conservation: before=%+v after=%+v", before, after)
	}
	if after.UncatalogedConversations != 0 || after.OrphanConversations != 1 {
		t.Fatalf("catalog did not repair the actionable orphan signal: %+v", after)
	}
	for _, query := range []string{paneID, threadID} {
		matches, _, total, distinct, err := SearchLocal(context.Background(), db, query, "recoverable", "", "", time.Time{}, 100)
		if err != nil || total != 88 || distinct != 1 || len(matches) != 88 || matches[0].SessionID != paneID {
			t.Fatalf("query=%q matches=%d total=%d distinct=%d err=%v", query, len(matches), total, distinct, err)
		}
	}
	matches, _, total, distinct, err := SearchLocal(context.Background(), db, "Plan Manager", "recoverable", "", "", time.Time{}, 100)
	if err != nil || total != 1 || distinct != 1 || len(matches) != 1 || matches[0].SessionID != paneID {
		t.Fatalf("text search matches=%d total=%d distinct=%d err=%v", len(matches), total, distinct, err)
	}
}

func TestNormalizeRolloutCheckpointSeparatesPrivatePathAndThreadIdentity(t *testing.T) {
	evidence := Evidence{AgentSessionID: "/state/sessions/codex/s-1/sessions/2026/09/03/rollout-2026-09-03T23-17-07-01a06a6b-88da-7422-b391-bb59c5f5e5e0.jsonl"}
	normalizeRolloutCheckpoint(&evidence)
	if evidence.AgentSessionID != "01a06a6b-88da-7422-b391-bb59c5f5e5e0" {
		t.Fatalf("thread identity=%q", evidence.AgentSessionID)
	}
	if evidence.RolloutRef == "" || evidence.RolloutRef == evidence.AgentSessionID {
		t.Fatalf("rollout reference=%q", evidence.RolloutRef)
	}
}

func TestApplyReconciliationBatchAdvancesWithoutRepeatingItems(t *testing.T) {
	items, err := PlanReconciliation([]Evidence{{SessionID: "one"}, {SessionID: "two"}, {SessionID: "three"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	store := &catalogTestStore{}
	if n, err := ApplyReconciliationBatch(context.Background(), store, items, 0, 2); err != nil || n != 2 {
		t.Fatalf("first batch mutations=%d err=%v", n, err)
	}
	if n, err := ApplyReconciliationBatch(context.Background(), store, items, 2, 2); err != nil || n != 1 {
		t.Fatalf("second batch mutations=%d err=%v", n, err)
	}
	if len(store.records) != 3 || store.records[0].SessionID != "one" || store.records[2].SessionID != "two" {
		t.Fatalf("batch records=%+v", store.records)
	}
}

func TestRollbackManifestRestoresCatalogPreimage(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE conversation_catalog(session_id TEXT PRIMARY KEY, lifecycle_state TEXT, lifecycle_version INTEGER, backend TEXT, agent_type TEXT, agent_session_id TEXT, agent_home_ref TEXT, rollout_ref TEXT, original_title TEXT, current_title TEXT, topic_summary TEXT, cwd TEXT, created_at TEXT, last_activity_at TEXT, source_fingerprint TEXT)`,
		`CREATE TABLE conversation_aliases(session_id TEXT, alias_kind TEXT, alias_value TEXT, observed_at TEXT, PRIMARY KEY(alias_kind, alias_value))`,
		`CREATE TABLE continuity_reconciliation_manifests(hash TEXT PRIMARY KEY, generation TEXT, items_json TEXT, previous_json TEXT, created_at TEXT)`,
		`CREATE TABLE continuity_publication_queue(session_id TEXT PRIMARY KEY, lifecycle_state TEXT, source_fingerprint TEXT, attempts INTEGER, last_error TEXT, published_fingerprint TEXT, updated_at TEXT)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	store := NewSQLCatalogStore(db)
	prior := CatalogRecord{SessionID: "s-1", LifecycleState: StateArchived, CurrentTitle: "before", SourceFingerprint: "sha256:before", Aliases: []Alias{{Kind: "agent_session", Value: "native-1"}}}
	if err := store.Upsert(context.Background(), prior); err != nil {
		t.Fatal(err)
	}
	if err := store.AddAliases(context.Background(), prior); err != nil {
		t.Fatal(err)
	}
	updated := prior
	updated.CurrentTitle = "after"
	updated.SourceFingerprint = "sha256:after"
	if err := store.Upsert(context.Background(), updated); err != nil {
		t.Fatal(err)
	}
	manifest := ReconciliationManifest{Hash: "sha256:rollback", Generation: "g", Items: []ReconcileItem{{Action: ActionUpdate, Record: updated}}, Previous: map[string]CatalogRecord{"s-1": prior}}
	if err := store.SaveManifest(context.Background(), manifest); err != nil {
		t.Fatal(err)
	}
	if err := store.RollbackManifest(context.Background(), manifest.Hash); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Existing(context.Background())
	if err != nil || loaded["s-1"].CurrentTitle != "before" || len(loaded["s-1"].Aliases) != 1 {
		t.Fatalf("restored=%+v err=%v", loaded["s-1"], err)
	}
}

func TestSQLCatalogLifecycleUpdateKeepsSearchStateInSync(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE conversation_catalog (
		session_id TEXT PRIMARY KEY, lifecycle_state TEXT NOT NULL,
		lifecycle_version INTEGER NOT NULL, backend TEXT, agent_type TEXT,
		agent_session_id TEXT, agent_home_ref TEXT, rollout_ref TEXT,
		original_title TEXT, current_title TEXT, topic_summary TEXT, cwd TEXT,
		created_at TEXT, last_activity_at TEXT, source_fingerprint TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE continuity_publication_queue (
		session_id TEXT PRIMARY KEY, lifecycle_state TEXT, source_fingerprint TEXT,
		attempts INTEGER, last_error TEXT, published_fingerprint TEXT, updated_at TEXT)`); err != nil {
		t.Fatal(err)
	}
	store := NewSQLCatalogStore(db)
	if err := store.Upsert(context.Background(), CatalogRecord{
		SessionID: "session-1", LifecycleState: StateLive,
		SourceFingerprint: "sha256:one", CreatedAt: time.Unix(1, 0),
		LastActivityAt: time.Unix(1, 0),
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateLifecycleState(context.Background(), "session-1", StateArchived); err != nil {
		t.Fatal(err)
	}
	var state string
	var version int
	if err := db.QueryRow(`SELECT lifecycle_state, lifecycle_version FROM conversation_catalog WHERE session_id = 'session-1'`).Scan(&state, &version); err != nil {
		t.Fatal(err)
	}
	if state != string(StateArchived) || version != 2 {
		t.Fatalf("catalog lifecycle=%q version=%d, want archived/2", state, version)
	}
	var queuedState, queuedFingerprint, publishedFingerprint string
	if err := db.QueryRow(`SELECT lifecycle_state, source_fingerprint, published_fingerprint FROM continuity_publication_queue WHERE session_id = 'session-1'`).Scan(&queuedState, &queuedFingerprint, &publishedFingerprint); err != nil {
		t.Fatal(err)
	}
	if queuedState != string(StateArchived) || queuedFingerprint != "sha256:one" || publishedFingerprint != "" {
		t.Fatalf("publication queue=%q %q %q, want archived/sha256:one/unpublished", queuedState, queuedFingerprint, publishedFingerprint)
	}
}
