package config_test

import (
	"context"
	"io"
	"testing"
	"time"

	"tunnel-manager/internal/config"
	localdb "tunnel-manager/internal/database"
	internalexposure "tunnel-manager/internal/exposure"
	internalroutes "tunnel-manager/internal/routes"
	"tunnel-manager/internal/testutil/mocks"
	internaltunnel "tunnel-manager/internal/tunnel"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

func TestNewProductionService_RemoteWiresCloudflareIngress(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internaltunnel.Schema),
		apidb.SchemaProviderFunc(internalexposure.Schema),
		apidb.SchemaProviderFunc(internalroutes.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))

	cfgRepo := config.NewSQLiteRepository(d)
	_, err := cfgRepo.Upsert(ctx, config.TunnelConfig{Mode: config.ModeRemote})
	require.NoError(t, err)
	routesSvc := internalroutes.NewService(internalroutes.NewSQLiteRepository(d, clk))
	_, err = routesSvc.Create(ctx, internalroutes.CreateInput{
		Subdomain: "web-console",
		Scenario:  "web-console",
		LocalPort: 21240,
	})
	require.NoError(t, err)

	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true,"result":{"config":{"ingress":[{"service":"http_status:404"}]}}}`)) // 1: read ingress
	doer.AddResponse(200, []byte(`{"success":true,"result":{}}`))                                                     // 2: push ingress
	// DNS automation now runs on the same remote reconcile path: resolve the
	// apex zone, look up an existing record, then create the proxied CNAME.
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"id":"zone1"}]}`)) // 3: zone lookup
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`))               // 4: find record (none)
	doer.AddResponse(200, []byte(`{"success":true,"result":{"id":"rec1"}}`))    // 5: create CNAME

	svc := config.NewProductionService(d, clk, config.ProductionOptions{
		Doer:    doer,
		HomeDir: t.TempDir(),
		Routes:  routesSvc,
		CredentialAuthority: &fakeAuthority{values: map[string]string{
			"cloudflare-account-id": "acct",
			"cloudflare-tunnel-id":  "tun",
			"cloudflare-api-token":  "tok",
		}},
	})

	res, err := svc.Sync(ctx, false, false)
	require.NoError(t, err)
	require.Equal(t, config.ModeRemote, res.Mode)
	require.Equal(t, []string{"web-console.itsagitime.com"}, res.Added)
	require.Equal(t, int64(5), doer.Calls.Load(), "remote sync reads+pushes ingress, then resolves zone, looks up, and creates the CNAME")
	require.Equal(t, "GET", doer.Requests[0].Method)
	require.Equal(t, "PUT", doer.Requests[1].Method)
	require.Contains(t, doer.Requests[1].Header.Get("Authorization"), "tok")
	body, err := io.ReadAll(doer.Requests[1].Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "web-console.itsagitime.com")
	require.Contains(t, string(body), "http://localhost:21240")

	// The CNAME create targets the tunnel and is proxied.
	require.Equal(t, "POST", doer.Requests[4].Method)
	dnsBody, err := io.ReadAll(doer.Requests[4].Body)
	require.NoError(t, err)
	require.Contains(t, string(dnsBody), "web-console.itsagitime.com")
	require.Contains(t, string(dnsBody), "tun.cfargotunnel.com")
	require.Contains(t, string(dnsBody), "\"proxied\":true")
}
