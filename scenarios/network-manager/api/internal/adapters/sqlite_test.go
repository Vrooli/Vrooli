package adapters

import (
	"context"
	"testing"
	"time"

	"network-manager/internal/testutil/db"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

func TestSQLiteRepositorySaveAndReadLatestReport(t *testing.T) {
	// [REQ:NM-P0-006] Capability reports are stored in domain-owned SQLite tables.
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(d)
	firstAt := time.Date(2026, 6, 23, 17, 0, 0, 0, time.UTC)
	secondAt := firstAt.Add(time.Minute)

	require.NoError(t, repo.SaveReport(context.Background(), Report{
		ObservedAt: firstAt,
		Platform:   PlatformSummary{OS: "linux", Arch: "amd64", Profile: "host-diagnostics", Notes: []string{"first"}, ObservedAt: firstAt},
		Capabilities: []Capability{
			{Adapter: "host-linux", Action: "read_network_status", Supported: true, Reason: "first", ObservedAt: firstAt},
		},
	}))
	require.NoError(t, repo.SaveReport(context.Background(), Report{
		ObservedAt: secondAt,
		Platform:   PlatformSummary{OS: "darwin", Arch: "arm64", Profile: "host-diagnostics", Notes: []string{"second"}, ObservedAt: secondAt},
		Capabilities: []Capability{
			{Adapter: "host-darwin", Action: "read_network_status", Supported: true, Reason: "second", ObservedAt: secondAt},
			{Adapter: "manual-router", Action: "router_dns_enforcement", Supported: false, Reason: "manual", ObservedAt: secondAt},
		},
	}))

	caps, err := repo.LatestCapabilities(context.Background())
	require.NoError(t, err)
	require.Len(t, caps, 2)
	require.Equal(t, "host-darwin", caps[0].Adapter)
	require.Equal(t, secondAt, caps[0].ObservedAt)

	summary, err := repo.LatestPlatformSummary(context.Background())
	require.NoError(t, err)
	require.Equal(t, "darwin", summary.OS)
	require.Equal(t, []string{"second"}, summary.Notes)
}
