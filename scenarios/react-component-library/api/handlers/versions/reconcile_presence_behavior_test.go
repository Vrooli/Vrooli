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

func TestReconcilePresencePlansColdAndRetiredVersionTransitions(t *testing.T) {
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
	require.NoError(t, os.WriteFile(manifest, []byte(`{"deprecatedVersions":[]}`), 0o600))
	_, err := db.ExecContext(ctx, `INSERT INTO components (id, library_id, slug, source_path, latest_version, manifest_path, indexed_at, updated_at)
		VALUES ('cmp-1', 'lib-1', 'button', 'components/button', '3.0.0', 'components/button/component.json', '2025-01-01', '2025-01-01')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO component_versions (id, component_id, library_id, version, status, source_path, content_sha256, indexed_at, created_at, released_at, presence)
		VALUES ('v1', 'cmp-1', 'lib-1', '1.0.0', 'deprecated', 'components/button/versions/1.0.0/button.tsx', 'sha1', '2025-01-01', '2025-01-01', '2025-01-02', 'materialized'),
		('v2', 'cmp-1', 'lib-1', '2.0.0', 'retired', 'components/button/versions/2.0.0/button.tsx', 'sha2', '2025-01-01', '2025-01-01', '2025-01-02', 'materialized'),
		('v3', 'cmp-1', 'lib-1', '2.1.0', 'retired', 'components/button/versions/2.1.0/button.tsx', 'sha3', '2025-01-01', '2025-01-01', '2025-01-02', 'evicted'),
		('v4', 'cmp-1', 'lib-1', '3.0.0', 'released', 'components/button/versions/3.0.0/button.tsx', 'sha4', '2025-01-01', '2025-01-01', '2025-01-02', 'evicted')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO version_ledger (library_id, version, lifecycle_state) VALUES
		('lib-1', '1.0.0', 'deprecated'), ('lib-1', '2.0.0', 'retired'),
		('lib-1', '2.1.0', 'retired'), ('lib-1', '3.0.0', 'released')`)
	require.NoError(t, err)

	asset := components.Component{ID: "cmp-1", LibraryID: "lib-1"}
	service := &materializeComponents{
		assets: []components.Component{asset},
		versions: map[string][]components.ComponentVersion{"cmp-1": {
			{Version: "1.0.0", Status: components.VersionStatusDeprecated, Presence: "materialized"},
			{Version: "2.0.0", Status: components.ComponentVersionStatus("retired"), Presence: "materialized"},
			{Version: "2.1.0", Status: components.ComponentVersionStatus("retired"), Presence: "evicted"},
			{Version: "3.0.0", Status: components.VersionStatusReleased, Presence: "evicted"},
		}},
	}
	materializer := &materializeRecorder{}
	h := NewConnectHandler(Deps{Components: service, Ledger: versionledger.NewRepository(db, root), Materializer: materializer})
	response, err := h.ReconcilePresence(ctx, connect.NewRequest(&versionsv1.ReconcilePresenceRequest{}))
	require.NoError(t, err)
	require.False(t, response.Msg.GetApplied())
	require.Equal(t, []string{"1.0.0", "2.0.0"}, presenceVersions(response.Msg.GetEvict()))
	require.Equal(t, []string{"3.0.0"}, presenceVersions(response.Msg.GetMaterialize()))
	require.Equal(t, []string{"2.1.0"}, presenceVersions(response.Msg.GetUnchanged()))
	require.Empty(t, materializer.calls, "previewing reconciliation must not mutate source presence")

	filtered, err := h.ReconcilePresence(ctx, connect.NewRequest(&versionsv1.ReconcilePresenceRequest{ComponentId: "different-component"}))
	require.NoError(t, err)
	require.Empty(t, filtered.Msg.GetEvict())
	require.Empty(t, filtered.Msg.GetMaterialize())
	require.Empty(t, filtered.Msg.GetUnchanged())

	unavailable := NewConnectHandler(Deps{
		Components:   &materializeComponents{errAt: "list"},
		Ledger:       versionledger.NewRepository(db, root),
		Materializer: &materializeRecorder{},
	})
	_, err = unavailable.ReconcilePresence(ctx, connect.NewRequest(&versionsv1.ReconcilePresenceRequest{}))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))

	versionReadFailure := NewConnectHandler(Deps{
		Components:   &materializeComponents{assets: []components.Component{asset}, errAt: "list-versions"},
		Ledger:       versionledger.NewRepository(db, root),
		Materializer: &materializeRecorder{},
	})
	_, err = versionReadFailure.ReconcilePresence(ctx, connect.NewRequest(&versionsv1.ReconcilePresenceRequest{}))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func presenceVersions(candidates []*versionsv1.RetireCandidate) []string {
	versions := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		versions = append(versions, candidate.GetVersion())
	}
	return versions
}
