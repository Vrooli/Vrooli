package config_test

import (
	"context"
	"io"
	"testing"
	"time"

	"tunnel-manager/internal/config"
	localdb "tunnel-manager/internal/database"
	internalroutes "tunnel-manager/internal/routes"
	"tunnel-manager/internal/testutil/db"
	"tunnel-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

func TestNewProductionService_RemoteWiresCloudflareIngress(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(config.Schema),
		apidb.SchemaProviderFunc(internalroutes.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))

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
	doer.AddResponse(200, []byte(`{"success":true,"result":{"config":{"ingress":[{"service":"http_status:404"}]}}}`))
	doer.AddResponse(200, []byte(`{"success":true,"result":{}}`))

	svc := config.NewProductionService(d, clk, config.ProductionOptions{
		Doer:    doer,
		HomeDir: t.TempDir(),
		Routes:  routesSvc,
		EnvLookup: func(key string) string {
			switch key {
			case "CLOUDFLARE_ACCOUNT_ID":
				return "acct"
			case "CLOUDFLARE_TUNNEL_ID":
				return "tun"
			case "CLOUDFLARE_API_TOKEN":
				return "tok"
			default:
				return ""
			}
		},
	})

	res, err := svc.Sync(ctx, false)
	require.NoError(t, err)
	require.Equal(t, config.ModeRemote, res.Mode)
	require.Equal(t, []string{"web-console.itsagitime.com"}, res.Added)
	require.Equal(t, int64(2), doer.Calls.Load(), "remote sync reads and pushes Cloudflare ingress")
	require.Equal(t, "GET", doer.Requests[0].Method)
	require.Equal(t, "PUT", doer.Requests[1].Method)
	require.Contains(t, doer.Requests[1].Header.Get("Authorization"), "tok")
	body, err := io.ReadAll(doer.Requests[1].Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "web-console.itsagitime.com")
	require.Contains(t, string(body), "http://localhost:21240")
}
