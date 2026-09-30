package dispatch_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	databasetest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"

	localdb "vrooli-bridge/internal/database"
	"vrooli-bridge/internal/dispatch"
	"vrooli-bridge/internal/registry"
)

// TestLiveNodeAdmission preserves the migrated owner grants and successful verb
// history observed in the live registry on 2026-08-18. The live database is
// never opened by this test: records are seeded through the same routed
// repository seam used by production, keeping the evidence readable and safe
// to run in CI. The Plan Manager log retains the pre-migration scope arrays.
func TestLiveNodeAdmission(t *testing.T) {
	t.Parallel()
	manifest, _, err := dispatch.BuildManifest()
	require.NoError(t, err)

	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db,
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(registry.Schema),
	))
	repo := registry.NewSQLiteRepository(db, scheduletest.New(time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)))

	type observedNode struct {
		node  registry.Node
		verbs []string
	}
	observed := []observedNode{
		{
			node: registry.Node{
				ID:     "25c7e426-c76c-421a-8351-aaf964589802",
				Name:   "minimouse",
				Kind:   registry.KindAgent,
				OS:     "darwin",
				Arch:   "amd64",
				Scopes: []string{"vrooli-bridge:read", "vrooli-bridge:write", "vrooli:read", "vrooli:write"},
			},
			verbs: []string{
				"capability ledger",
				"host inventory",
				"resource start",
				"resource status",
				"scenario status",
				"setup",
				"setup status",
			},
		},
		{
			node: registry.Node{
				ID:     "697b6224-6283-4a31-90e2-73724e424c05",
				Name:   "swarminator",
				Kind:   registry.KindAgent,
				OS:     "linux",
				Arch:   "amd64",
				Scopes: []string{"vrooli-bridge:read", "vrooli-bridge:write", "vrooli:read", "vrooli:write", "device-control:read", "device-control:write"},
			},
			verbs: []string{
				"device-control flow",
				"scenario status",
				"scenario test",
			},
		},
	}

	for _, item := range observed {
		_, createErr := repo.Create(ctx, item.node)
		require.NoError(t, createErr)
	}

	nodes, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, nodes, len(observed))

	verbsByNode := make(map[string][]string, len(observed))
	for _, item := range observed {
		verbsByNode[item.node.ID] = item.verbs
	}
	// These successful rows invoked historical parent/retired help surfaces,
	// not a current governed leaf command. Restoring them would bypass the
	// manifest rather than restore authority, so they remain typed refusals.
	retired := map[string]bool{"device-control flow": true, "setup status": true}
	for _, node := range nodes {
		for _, verb := range verbsByNode[node.ID] {
			err := dispatch.Admit(dispatch.Job{NodeID: node.ID, Verb: verb}, dispatch.TargetNode{
				ID: node.ID, Kind: node.Kind, OS: node.OS, Arch: node.Arch, Scopes: node.Scopes,
			}, manifest)
			if retired[verb] {
				var notInManifest dispatch.ErrVerbNotInManifest
				if !errors.As(err, &notInManifest) {
					t.Errorf("node %s (%s) historical retired verb %q must remain outside the current manifest, got: %v", node.Name, node.ID, verb, err)
				}
				continue
			}
			if err != nil {
				t.Errorf("node %s (%s) must retain admission for historically successful verb %q: %v", node.Name, node.ID, verb, err)
			}
		}
	}
}
