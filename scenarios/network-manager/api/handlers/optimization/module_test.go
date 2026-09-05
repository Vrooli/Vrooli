package optimization

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	domainadapters "network-manager/internal/adapters"
	domainoptimization "network-manager/internal/optimization"
	domainresolver "network-manager/internal/resolver"
	domainsnapshot "network-manager/internal/snapshot"

	db "github.com/vrooli/api-core/databasetest"
)

func TestModuleExposesEndpoints(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(domainadapters.Schema),
		apidb.SchemaProviderFunc(domainoptimization.Schema),
		apidb.SchemaProviderFunc(domainresolver.Schema),
		apidb.SchemaProviderFunc(domainsnapshot.Schema),
	))
	m := Module(d)
	if m.Name != "optimization" {
		t.Fatalf("module name = %q, want optimization", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected optimization endpoints")
	}
}

func TestModuleServiceUsesResolverAwareCapabilities(t *testing.T) {
	// [REQ:NM-P0-005] Optimization candidates include AdGuard-backed safe apply only when the governed resolver backend exists.
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(domainadapters.Schema),
		apidb.SchemaProviderFunc(domainoptimization.Schema),
		apidb.SchemaProviderFunc(domainresolver.Schema),
		apidb.SchemaProviderFunc(domainsnapshot.Schema),
	))
	_, err := domainresolver.NewSQLiteRepository(d).SaveBackend(context.Background(), domainresolver.BackendConfig{
		Backend:       domainresolver.AdGuardHomeBackend,
		BaseURL:       "http://adguard.local",
		CredentialRef: "vrooli/adguard-home",
	})
	require.NoError(t, err)
	_, err = domainsnapshot.NewSQLiteRepository(d).Create(context.Background(), domainsnapshot.Snapshot{
		ID:        "baseline-1",
		Status:    "baseline",
		Profile:   "home",
		Summary:   "baseline ready",
		CreatedAt: time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	run, err := newService(d).CreateRun(context.Background(), "", false)
	require.NoError(t, err)
	var found bool
	for _, candidate := range run.Candidates {
		if candidate.ApprovalRequired && candidate.RollbackSupported && strings.Contains(candidate.ID, "adguard-home-dns-filtering-stability") {
			found = true
			break
		}
	}
	require.True(t, found, "expected AdGuard DNS filtering optimization candidate")
}
