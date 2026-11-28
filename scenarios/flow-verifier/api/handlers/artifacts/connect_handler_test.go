package artifacts_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	artifactsH "flow-verifier/handlers/artifacts"
	"flow-verifier/internal/artifacts"

	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/artifacts"
	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/artifacts/artifacts_v1connect"
)

type fakeGenerator struct{ err error }

func (f fakeGenerator) Generate(_ context.Context, _, _ string) error { return f.err }

func newClient(t *testing.T, gen artifacts.Generator) artifactsconnect.ArtifactsServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	svc := artifacts.NewService(gen)
	path, handler := artifactsconnect.NewArtifactsServiceHandler(artifactsH.NewConnectHandler(artifactsH.Deps{
		Service:   svc,
		Scenarios: nil,
		Logger:    logger,
	}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return artifactsconnect.NewArtifactsServiceClient(server.Client(), server.URL)
}

func TestGetArtifactStatusFlowNotFound(t *testing.T) {
	client := newClient(t, fakeGenerator{})
	_, err := client.GetArtifactStatus(context.Background(), connect.NewRequest(&artifactsv1.GetArtifactStatusRequest{
		Root:   t.TempDir(),
		FlowId: "ghost",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestGenerateArtifactsScenariosUnconfigured(t *testing.T) {
	client := newClient(t, fakeGenerator{err: errors.New("boom")})
	_, err := client.GenerateArtifacts(context.Background(), connect.NewRequest(&artifactsv1.GenerateArtifactsRequest{
		// No root, no scenario id → must fall through to "scenarios service not configured".
		FlowId: "x",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}
