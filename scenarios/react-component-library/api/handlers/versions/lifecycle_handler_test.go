package versions

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"
	"react-component-library/internal/adoptions"
	"react-component-library/internal/components"
	localdb "react-component-library/internal/database"
	"react-component-library/internal/versionledger"
)

func TestLifecycleMethodsRequireConfiguredLedger(t *testing.T) {
	h := NewConnectHandler(Deps{})
	ctx := context.Background()
	checks := []struct {
		name string
		call func() error
	}{
		{"list retire candidates", func() error {
			_, err := h.ListRetireCandidates(ctx, connect.NewRequest(&versionsv1.ListRetireCandidatesRequest{}))
			return err
		}},
		{"plan cleanup", func() error {
			_, err := h.PlanCleanup(ctx, connect.NewRequest(&versionsv1.PlanCleanupRequest{}))
			return err
		}},
		{"apply cleanup", func() error {
			_, err := h.CleanupVersions(ctx, connect.NewRequest(&versionsv1.CleanupVersionsRequest{}))
			return err
		}},
		{"cleanup draft", func() error {
			_, err := h.CleanupDraft(ctx, connect.NewRequest(&versionsv1.CleanupDraftRequest{}))
			return err
		}},
		{"deprecate", func() error {
			_, err := h.DeprecateVersion(ctx, connect.NewRequest(&versionsv1.VersionLifecycleRequest{}))
			return err
		}},
		{"archive", func() error {
			_, err := h.ArchiveVersion(ctx, connect.NewRequest(&versionsv1.VersionLifecycleRequest{}))
			return err
		}},
		{"retire", func() error {
			_, err := h.RetireVersion(ctx, connect.NewRequest(&versionsv1.VersionLifecycleRequest{}))
			return err
		}},
		{"list ledger", func() error {
			_, err := h.ListVersionLedger(ctx, connect.NewRequest(&versionsv1.ListVersionLedgerRequest{}))
			return err
		}},
		{"reconcile presence", func() error {
			_, err := h.ReconcilePresence(ctx, connect.NewRequest(&versionsv1.ReconcilePresenceRequest{}))
			return err
		}},
		{"doctor", func() error { _, err := h.Doctor(ctx, connect.NewRequest(&versionsv1.DoctorRequest{})); return err }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if got := connect.CodeOf(check.call()); got != connect.CodeFailedPrecondition {
				t.Fatalf("code = %s, want failed precondition", got)
			}
		})
	}
}

func TestCleanupLifecycleHandlersMapPlanAndApplyResults(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db,
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(components.Schema),
		database.SchemaProviderFunc(adoptions.Schema),
		database.SchemaProviderFunc(versionledger.Schema),
	))
	root := t.TempDir()
	manifest := filepath.Join(root, "components/button/component.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(manifest), 0o755))
	require.NoError(t, os.WriteFile(manifest, []byte(`{"deprecatedVersions":[]}`), 0o644))
	_, err := db.ExecContext(ctx, `INSERT INTO components (id, library_id, slug, source_path, latest_version, manifest_path, indexed_at, updated_at)
		VALUES ('cmp-1', 'lib-1', 'button', 'components/button', '2.0.0', 'components/button/component.json', '2025-01-01', '2025-01-01')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO component_versions (id, component_id, library_id, version, status, source_path, content_sha256, indexed_at, created_at, released_at)
		VALUES ('v1', 'cmp-1', 'lib-1', '1.0.0', 'deprecated', 'components/button/versions/1.0.0/button.tsx', 'sha1', '2025-01-01', '2025-01-01', '2025-01-02'),
		('v2', 'cmp-1', 'lib-1', '2.0.0', 'released', 'components/button/versions/2.0.0/button.tsx', 'sha2', '2026-01-01', '2026-01-01', '2026-01-02')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO version_ledger (library_id, version, lifecycle_state) VALUES ('lib-1', '1.0.0', 'deprecated'), ('lib-1', '2.0.0', 'released')`)
	require.NoError(t, err)

	h := NewConnectHandler(Deps{Ledger: versionledger.NewRepository(db, root)})
	candidates, err := h.ListRetireCandidates(ctx, connect.NewRequest(&versionsv1.ListRetireCandidatesRequest{ComponentId: "cmp-1"}))
	require.NoError(t, err)
	require.Len(t, candidates.Msg.GetCandidates(), 1)
	require.Equal(t, "1.0.0", candidates.Msg.GetCandidates()[0].GetVersion())

	plan, err := h.PlanCleanup(ctx, connect.NewRequest(&versionsv1.PlanCleanupRequest{Scope: &versionsv1.CleanupScope{ComponentId: "lib-1"}}))
	require.NoError(t, err)
	require.NotEmpty(t, plan.Msg.GetPlanHash())
	require.Len(t, plan.Msg.GetItems(), 2)
	require.Equal(t, "1.0.0", plan.Msg.GetItems()[0].GetVersion().GetVersion())
	require.True(t, plan.Msg.GetItems()[0].GetEligible())
	require.Equal(t, "latest version", plan.Msg.GetItems()[1].GetReason())

	preview, err := h.CleanupVersions(ctx, connect.NewRequest(&versionsv1.CleanupVersionsRequest{Scope: &versionsv1.CleanupScope{ComponentId: "lib-1"}, PlanHash: plan.Msg.GetPlanHash()}))
	require.NoError(t, err)
	require.False(t, preview.Msg.GetApplied())
	require.Zero(t, preview.Msg.GetRetiredCount())

	_, err = h.CleanupVersions(ctx, connect.NewRequest(&versionsv1.CleanupVersionsRequest{Scope: &versionsv1.CleanupScope{ComponentId: "lib-1"}, PlanHash: "stale-plan", Confirm: true}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestCleanupHandlersMapLedgerFailuresToContractCodes(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db,
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(components.Schema),
		database.SchemaProviderFunc(adoptions.Schema),
		database.SchemaProviderFunc(versionledger.Schema),
	))
	h := NewConnectHandler(Deps{Ledger: versionledger.NewRepository(db, t.TempDir())})
	_, err := h.ListRetireCandidates(ctx, connect.NewRequest(&versionsv1.ListRetireCandidatesRequest{}))
	require.NoError(t, err)
	_, err = h.PlanCleanup(ctx, connect.NewRequest(&versionsv1.PlanCleanupRequest{}))
	require.NoError(t, err)
	_, err = h.CleanupVersions(ctx, connect.NewRequest(&versionsv1.CleanupVersionsRequest{Scope: &versionsv1.CleanupScope{ComponentId: "cmp", LibraryId: "lib"}, Confirm: true}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	_, err = h.CleanupDraft(ctx, connect.NewRequest(&versionsv1.CleanupDraftRequest{ComponentId: "missing", Confirm: true}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}
