package machines_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"vrooli-bridge/internal/machines"
	"vrooli-bridge/internal/testutil/db"
	"vrooli-bridge/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
	apiDB "github.com/vrooli/api-core/database"
)

type nodeReaderStub struct{ node machines.NodeSnapshot }

func (s nodeReaderStub) GetNode(context.Context, string) (machines.NodeSnapshot, error) {
	return s.node, nil
}

type presenceReaderStub struct{ presence machines.PresenceSnapshot }

func (s presenceReaderStub) GetPresence(context.Context, string) (machines.PresenceSnapshot, error) {
	return s.presence, nil
}

func newDB(t *testing.T) (*sql.DB, *mocks.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	clk := mocks.NewFakeClock(time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))
	require.NoError(t, apiDB.EnsureSchemas(context.Background(), d, apiDB.SchemaProviderFunc(machines.Schema)))
	return d, clk
}

// [REQ:BRG-MEC-001] A Machine has stable identity before contact, maintains
// ordered locators, and permits only one current paired Node lineage entry.
func TestMachineCreateAndNodeLineage(t *testing.T) {
	d, clk := newDB(t)
	repo := machines.NewSQLiteRepository(d, clk)
	created, err := repo.Create(context.Background(), machines.CreateInput{Locators: []machines.Locator{{Kind: "hostname", Value: "Mac-Mini.Local."}, {Kind: "ip", Value: "192.0.2.4"}}})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, "managed-connection", created.DesiredProfileID)
	require.Equal(t, "v1", created.DesiredProfileVersion)
	require.Len(t, created.Locators, 2)

	first, err := repo.LinkNode(context.Background(), created.ID, "node-old", "corr-1")
	require.NoError(t, err)
	require.Len(t, first.Lineage, 1)
	require.True(t, first.Lineage[0].Current)

	second, err := repo.LinkNode(context.Background(), created.ID, "node-new", "corr-2")
	require.NoError(t, err)
	require.Len(t, second.Lineage, 2)
	var old, current machines.NodeLineage
	for _, entry := range second.Lineage {
		if entry.NodeID == "node-old" {
			old = entry
		}
		if entry.NodeID == "node-new" {
			current = entry
		}
	}
	require.False(t, old.Current)
	require.True(t, current.Current)
	require.Equal(t, "corr-2", current.CorrelationID)
}

func TestArchiveUsesOptimisticVersion(t *testing.T) {
	d, clk := newDB(t)
	repo := machines.NewSQLiteRepository(d, clk)
	m, err := repo.Create(context.Background(), machines.CreateInput{Locators: []machines.Locator{{Kind: "ssh", Value: "operator@host"}}})
	require.NoError(t, err)
	_, err = repo.Archive(context.Background(), m.ID, m.Version+1)
	require.ErrorAs(t, err, &machines.ErrConflict{})
	archived, err := repo.Archive(context.Background(), m.ID, m.Version)
	require.NoError(t, err)
	require.Equal(t, machines.LifecycleArchived, archived.Lifecycle)
	require.Equal(t, m.Version+1, archived.Version)
}

func TestRemovePreservesHistoryAndCleanupTombstoneIsExplicit(t *testing.T) {
	d, clk := newDB(t)
	repo := machines.NewSQLiteRepository(d, clk)
	machine, err := repo.Create(context.Background(), machines.CreateInput{Locators: []machines.Locator{{Kind: "hostname", Value: "mac.local"}}})
	require.NoError(t, err)
	removed, err := repo.Remove(context.Background(), machine.ID, machine.Version)
	require.NoError(t, err)
	require.Equal(t, machines.LifecycleRemoved, removed.Lifecycle)
	require.NotEmpty(t, removed.ID, "removal preserves durable Machine history")
	tombstone, err := repo.CreateCleanupTombstone(context.Background(), machines.CleanupTombstone{MachineID: machine.ID, Action: "remove_ssh_access", Detail: "host unreachable"})
	require.NoError(t, err)
	require.Equal(t, machines.CleanupPending, tombstone.Status)
	acknowledged, err := repo.UpdateCleanupTombstone(context.Background(), tombstone.ID, machines.CleanupAbandoned, "operator accepted unreachable host")
	require.NoError(t, err)
	require.False(t, acknowledged.AcknowledgedAt.IsZero())
	history, err := repo.ListCleanupTombstones(context.Background(), machine.ID)
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, acknowledged.ID, history[0].ID)
	require.Equal(t, machines.CleanupAbandoned, history[0].Status)
}

func TestCleanupTombstonePendingEffectIsIdempotent(t *testing.T) {
	d, clk := newDB(t)
	repo := machines.NewSQLiteRepository(d, clk)
	ctx := context.Background()
	machine, err := repo.Create(ctx, machines.CreateInput{Locators: []machines.Locator{{Kind: "hostname", Value: "cleanup.example"}}})
	require.NoError(t, err)
	first, err := repo.CreateCleanupTombstone(ctx, machines.CleanupTombstone{MachineID: machine.ID, Action: "remove_ssh_access", Detail: "first request"})
	require.NoError(t, err)
	second, err := repo.CreateCleanupTombstone(ctx, machines.CleanupTombstone{MachineID: machine.ID, Action: "remove_ssh_access", Detail: "replayed request"})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, "first request", second.Detail)
	_, err = repo.UpdateCleanupTombstone(ctx, first.ID, machines.CleanupConfirmed, "removed")
	require.NoError(t, err)
	next, err := repo.CreateCleanupTombstone(ctx, machines.CleanupTombstone{MachineID: machine.ID, Action: "remove_ssh_access"})
	require.NoError(t, err)
	require.NotEqual(t, first.ID, next.ID, "a later distinct cleanup effect remains historical")
}

func TestMachineAuditIsAppendOnlyAndExcludesSensitiveMaterial(t *testing.T) {
	d, clk := newDB(t)
	repo := machines.NewSQLiteRepository(d, clk).(interface {
		AppendAudit(context.Context, machines.AuditEvent) error
	})
	require.NoError(t, repo.AppendAudit(context.Background(), machines.AuditEvent{MachineID: "machine-1", Action: "archive", Actor: "owner-1", Detail: "history preserved"}))
	var action, actor, detail string
	require.NoError(t, d.QueryRow("SELECT action,actor,detail FROM machine_audit_events WHERE machine_id='machine-1'").Scan(&action, &actor, &detail))
	require.Equal(t, "archive", action)
	require.Equal(t, "owner-1", actor)
	require.NotContains(t, detail, "PRIVATE")
}

func TestTrustStoresOnlyOpaqueReferenceAndTypedFingerprints(t *testing.T) {
	d, clk := newDB(t)
	repo := machines.NewSQLiteRepository(d, clk)
	machine, err := repo.Create(context.Background(), machines.CreateInput{Locators: []machines.Locator{{Kind: "ssh", Value: "operator@mac"}}})
	require.NoError(t, err)
	stored, err := repo.UpsertTrust(context.Background(), machines.TrustRecord{MachineID: machine.ID, ClientKeyRef: "ssh-key://machine/abc", ClientKeyFingerprint: "SHA256:client", HostKeyFingerprint: "SHA256:host", HostKeyState: machines.HostKeyVerified})
	require.NoError(t, err)
	require.Equal(t, "ssh-key://machine/abc", stored.ClientKeyRef)
	loaded, err := repo.GetTrust(context.Background(), machine.ID)
	require.NoError(t, err)
	require.Equal(t, machines.HostKeyVerified, loaded.HostKeyState)
	require.NotContains(t, loaded.ClientKeyRef, "PRIVATE", "private material has no database field")
}

// [REQ:BRG-MEC-003] Historic Nodes and one-shot onboarding records are never
// guessed into Machines: ambiguity remains a durable, typed review item.
func TestBackfillLegacyPreservesAmbiguousRecordsForReview(t *testing.T) {
	d, _ := newDB(t)
	ctx := context.Background()
	_, err := d.ExecContext(ctx, "CREATE TABLE nodes (id TEXT PRIMARY KEY)")
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, "CREATE TABLE onboarding_ops (id TEXT PRIMARY KEY)")
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, "INSERT INTO nodes (id) VALUES ('legacy-node')")
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, "INSERT INTO onboarding_ops (id) VALUES ('legacy-op')")
	require.NoError(t, err)

	require.NoError(t, machines.BackfillLegacy(ctx, d))
	require.NoError(t, machines.BackfillLegacy(ctx, d), "backfill is idempotent")

	var reviews, machineCount int
	require.NoError(t, d.QueryRowContext(ctx, "SELECT COUNT(*) FROM machine_migration_reviews WHERE status='needs_review'").Scan(&reviews))
	require.NoError(t, d.QueryRowContext(ctx, "SELECT COUNT(*) FROM machines").Scan(&machineCount))
	require.Equal(t, 2, reviews)
	require.Zero(t, machineCount)

	repo := machines.NewSQLiteRepository(d, mocks.NewFakeClock(time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)))
	items, err := repo.ListMigrationReviews(ctx)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "ambiguous", items[0].Confidence)
	acknowledged, err := repo.AcknowledgeMigrationReview(ctx, items[0].ID)
	require.NoError(t, err)
	require.Equal(t, "acknowledged", acknowledged.Status)
	require.False(t, acknowledged.ReviewedAt.IsZero())
}

func TestComposeUsesLiveOwnersWithoutPersistingTheirFacts(t *testing.T) {
	machine := machines.Machine{ID: "machine-1", Lineage: []machines.NodeLineage{{NodeID: "node-1", Current: true}}}
	projection, err := machines.Compose(context.Background(), machine,
		nodeReaderStub{node: machines.NodeSnapshot{ID: "node-1", Name: "paired-mac", Capabilities: []string{"darwin"}}},
		presenceReaderStub{presence: machines.PresenceSnapshot{Connected: true}})
	require.NoError(t, err)
	require.True(t, projection.HasNode)
	require.Equal(t, "paired-mac", projection.Node.Name)
	require.True(t, projection.Presence.Connected)
	require.Empty(t, machine.TrustRef, "live Registry/Presence facts are not copied into Machine")
}

func TestMigrateAddsReviewConfidenceWithoutDroppingEvidence(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()
	_, err := d.ExecContext(ctx, `CREATE TABLE machine_migration_reviews (
        id TEXT PRIMARY KEY, legacy_source TEXT NOT NULL, legacy_id TEXT NOT NULL,
        status TEXT NOT NULL, reason TEXT NOT NULL, created_at TEXT NOT NULL,
        reviewed_at TEXT NOT NULL DEFAULT '')`)
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, "INSERT INTO machine_migration_reviews (id,legacy_source,legacy_id,status,reason,created_at) VALUES ('review-1','nodes','node-1','needs_review','missing correlation','2026-07-17T12:00:00Z')")
	require.NoError(t, err)
	require.NoError(t, machines.Migrate(ctx, d))
	require.NoError(t, machines.Migrate(ctx, d), "schema evolution is idempotent")
	var confidence string
	require.NoError(t, d.QueryRowContext(ctx, "SELECT confidence FROM machine_migration_reviews WHERE id='review-1'").Scan(&confidence))
	require.Equal(t, "ambiguous", confidence)
}
