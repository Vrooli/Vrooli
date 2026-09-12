package apply_test

import (
	"context"
	"testing"

	"brand-manager/handlers/apply"
	internalapply "brand-manager/internal/apply"
	mocks "brand-manager/internal/apply/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/apply"
	applyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/apply/apply_v1connect"
)

// newClient wires the real internal apply service over in-memory fakes behind the
// generated Connect handler, exercising handler + adapter + service together.
func newClient(t *testing.T, brands *mocks.FakeBrandStore, assets *mocks.FakeAssetStore, recorder *mocks.FakeAssignmentRecorder, ws *mocks.FakeWorkspace) applyconnect.ApplyServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	svc := internalapply.NewService(brands, assets, recorder, ws, logger)
	path, handler := applyconnect.NewApplyServiceHandler(apply.NewConnectHandler(apply.Deps{Service: svc, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return applyconnect.NewApplyServiceClient(server.Client(), server.URL)
}

func seeded(t *testing.T) (*mocks.FakeBrandStore, *mocks.FakeWorkspace) {
	t.Helper()
	brands := &mocks.FakeBrandStore{}
	brands.Seed(internalapply.BrandView{
		ID:          "b1",
		Version:     2,
		DisplayName: "Acme",
		Colors:      internalapply.Colors{Primary: "#112233"},
	})
	ws := &mocks.FakeWorkspace{}
	ws.SeedScenario("web-console")
	return brands, ws
}

func TestConnect_PreviewDoesNotWrite(t *testing.T) {
	brands, ws := seeded(t)
	recorder := &mocks.FakeAssignmentRecorder{}
	client := newClient(t, brands, &mocks.FakeAssetStore{}, recorder, ws)

	resp, err := client.PreviewApply(context.Background(), connect.NewRequest(&applyv1.PreviewApplyRequest{
		BrandId:      "b1",
		ScenarioName: "web-console",
		Elements:     []string{"colors"},
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.DryRun)
	require.Len(t, resp.Msg.Applied, 1)
	require.Equal(t, "colors", resp.Msg.Applied[0].Element)
	require.Zero(t, ws.WriteCount())
	require.Empty(t, recorder.Recorded())
}

func TestConnect_ApplyWritesAndRecords(t *testing.T) {
	brands, ws := seeded(t)
	recorder := &mocks.FakeAssignmentRecorder{}
	client := newClient(t, brands, &mocks.FakeAssetStore{}, recorder, ws)

	resp, err := client.ApplyBrand(context.Background(), connect.NewRequest(&applyv1.ApplyBrandRequest{
		BrandId:      "b1",
		ScenarioName: "web-console",
		Elements:     []string{"colors"},
	}))
	require.NoError(t, err)
	require.False(t, resp.Msg.DryRun)
	require.Equal(t, int32(2), resp.Msg.BrandVersion)
	require.Equal(t, 1, ws.WriteCount())
	require.Len(t, recorder.Recorded(), 1)
}

func TestConnect_ApplyUnknownBrandIsNotFound(t *testing.T) {
	ws := &mocks.FakeWorkspace{}
	ws.SeedScenario("web-console")
	client := newClient(t, &mocks.FakeBrandStore{}, &mocks.FakeAssetStore{}, &mocks.FakeAssignmentRecorder{}, ws)

	_, err := client.ApplyBrand(context.Background(), connect.NewRequest(&applyv1.ApplyBrandRequest{
		BrandId:      "ghost",
		ScenarioName: "web-console",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestConnect_ApplyMissingScenarioIsNotFound(t *testing.T) {
	brands, _ := seeded(t)
	client := newClient(t, brands, &mocks.FakeAssetStore{}, &mocks.FakeAssignmentRecorder{}, &mocks.FakeWorkspace{})

	_, err := client.ApplyBrand(context.Background(), connect.NewRequest(&applyv1.ApplyBrandRequest{
		BrandId:      "b1",
		ScenarioName: "nope",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestConnect_PreviewMissingArgsIsInvalidArgument(t *testing.T) {
	brands, ws := seeded(t)
	client := newClient(t, brands, &mocks.FakeAssetStore{}, &mocks.FakeAssignmentRecorder{}, ws)

	_, err := client.PreviewApply(context.Background(), connect.NewRequest(&applyv1.PreviewApplyRequest{
		ScenarioName: "web-console",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
