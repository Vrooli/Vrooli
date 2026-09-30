package staleness_test

import (
	"context"
	"testing"

	stalenessH "development-toolchain-validator/handlers/staleness"
	stalenessdom "development-toolchain-validator/internal/staleness"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	stalenessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/staleness"
	stalenessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/staleness/staleness_v1connect"
)

type fakeService struct {
	Out []stalenessdom.Entry
	Err error
}

func (f *fakeService) ListStale(context.Context) ([]stalenessdom.Entry, error) {
	return f.Out, f.Err
}

var _ stalenessdom.Service = (*fakeService)(nil)

func newClient(t *testing.T, svc stalenessdom.Service) stalenessconnect.StalenessServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := stalenessconnect.NewStalenessServiceHandler(stalenessH.NewConnectHandler(stalenessH.Deps{
		Service: svc, Logger: logger,
	}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return stalenessconnect.NewStalenessServiceClient(server.Client(), server.URL)
}

func TestListStale_TranslatesKind(t *testing.T) {
	client := newClient(t, &fakeService{Out: []stalenessdom.Entry{
		{SkillID: "a", GoldenSlug: "g", Kind: stalenessdom.StaleKindBoth},
	}})
	resp, err := client.ListStale(context.Background(), connect.NewRequest(&stalenessv1.ListStaleRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Entries, 1)
	require.Equal(t, stalenessv1.StaleKind_STALE_KIND_BOTH, resp.Msg.Entries[0].Kind)
}

func TestListStale_EmptyOK(t *testing.T) {
	client := newClient(t, &fakeService{})
	resp, err := client.ListStale(context.Background(), connect.NewRequest(&stalenessv1.ListStaleRequest{}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.Entries)
}
