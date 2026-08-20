package exposure

import (
	"context"
	"io"
	"testing"
	"time"

	internalconfig "tunnel-manager/internal/config"
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

type testCredentialStore struct{}

func (testCredentialStore) Status(context.Context) (internalconfig.CredentialStatus, error) {
	return internalconfig.CredentialStatus{Ready: true}, nil
}

func (testCredentialStore) Resolve(context.Context) (internalconfig.CFConfig, error) {
	return internalconfig.CFConfig{AccountID: "acct", TunnelID: "tun", APIToken: "tok"}, nil
}

func (testCredentialStore) Save(context.Context, internalconfig.CredentialUpdate) (internalconfig.CredentialStatus, error) {
	return internalconfig.CredentialStatus{Ready: true}, nil
}

func (testCredentialStore) Delete(context.Context, []string) (internalconfig.CredentialStatus, error) {
	return internalconfig.CredentialStatus{Ready: true}, nil
}

func TestIngressAdapter_ReconcileUsesConfiguredRemoteIngress(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internaltunnel.Schema),
		apidb.SchemaProviderFunc(internalexposure.Schema),
		apidb.SchemaProviderFunc(internalroutes.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))

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
	doer.AddResponse(200, []byte(`{"success":true,"result":{"config":{"ingress":[{"service":"http_status:404"}]}}}`)) // read ingress
	doer.AddResponse(200, []byte(`{"success":true,"result":{}}`))                                                     // push ingress
	// Reconcile (Sync prune=true) now also ensures the proxied CNAME on the
	// remote path: resolve zone, look up, create.
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"id":"zone1"}]}`)) // zone lookup
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`))               // find record (none)
	doer.AddResponse(200, []byte(`{"success":true,"result":{"id":"rec1"}}`))    // create CNAME
	cfgSvc := internalconfig.NewProductionService(d, clk, internalconfig.ProductionOptions{
		Doer:            doer,
		HomeDir:         t.TempDir(),
		Routes:          routesSvc,
		CredentialStore: testCredentialStore{},
	})

	err = (ingressAdapter{cfg: cfgSvc}).Reconcile(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(5), doer.Calls.Load(), "adapter reconcile uses the configured Cloudflare ingress client, then ensures DNS")
	require.Equal(t, "GET", doer.Requests[0].Method)
	require.Equal(t, "PUT", doer.Requests[1].Method)
	body, err := io.ReadAll(doer.Requests[1].Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "agent-manager.itsagitime.com")
	require.Contains(t, string(body), "http://localhost:21100")
	require.Equal(t, "POST", doer.Requests[4].Method, "ensures the proxied CNAME after the ingress push")
}
