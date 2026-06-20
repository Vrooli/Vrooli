package exposure

import (
	"context"
	"io"
	"testing"
	"time"

	internalconfig "tunnel-manager/internal/config"
	localdb "tunnel-manager/internal/database"
	internalroutes "tunnel-manager/internal/routes"
	"tunnel-manager/internal/testutil/db"
	"tunnel-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

func TestIngressAdapter_ReconcileUsesConfiguredRemoteIngress(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalconfig.Schema),
		apidb.SchemaProviderFunc(internalroutes.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))

	cfgRepo := internalconfig.NewSQLiteRepository(d)
	_, err := cfgRepo.Upsert(ctx, internalconfig.TunnelConfig{Mode: internalconfig.ModeRemote})
	require.NoError(t, err)
	routesSvc := internalroutes.NewService(internalroutes.NewSQLiteRepository(d, clk))
	_, err = routesSvc.Create(ctx, internalroutes.CreateInput{
		Subdomain: "agent-manager",
		Scenario:  "agent-manager",
		LocalPort: 21100,
	})
	require.NoError(t, err)

	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true,"result":{"config":{"ingress":[{"service":"http_status:404"}]}}}`))
	doer.AddResponse(200, []byte(`{"success":true,"result":{}}`))
	cfgSvc := internalconfig.NewProductionService(d, clk, internalconfig.ProductionOptions{
		Doer:    doer,
		HomeDir: t.TempDir(),
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

	err = (ingressAdapter{cfg: cfgSvc}).Reconcile(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(2), doer.Calls.Load(), "adapter must use the configured Cloudflare ingress client")
	require.Equal(t, "GET", doer.Requests[0].Method)
	require.Equal(t, "PUT", doer.Requests[1].Method)
	body, err := io.ReadAll(doer.Requests[1].Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "agent-manager.itsagitime.com")
	require.Contains(t, string(body), "http://localhost:21100")
}
