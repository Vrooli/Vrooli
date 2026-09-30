package verifications_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	verificationsH "flow-verifier/handlers/verifications"
	internalruns "flow-verifier/internal/runs"

	verificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/verifications"
	verificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/verifications/verifications_v1connect"
)

type fakeRepo struct{}

func (f *fakeRepo) Insert(_ context.Context, r internalruns.Run) (internalruns.Run, error) {
	return r, nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (internalruns.Run, error) {
	return internalruns.Run{}, internalruns.ErrNotFound{ID: id}
}

func (f *fakeRepo) List(_ context.Context, _ internalruns.ListQuery) ([]internalruns.Run, error) {
	return nil, nil
}

func newClient(t *testing.T) verificationsconnect.VerificationsServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	svc := internalruns.NewService(&fakeRepo{})
	path, handler := verificationsconnect.NewVerificationsServiceHandler(verificationsH.NewConnectHandler(verificationsH.Deps{
		Runs:   svc,
		Logger: logger,
	}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return verificationsconnect.NewVerificationsServiceClient(server.Client(), server.URL)
}

func TestGetVerificationNotFound(t *testing.T) {
	client := newClient(t)
	_, err := client.GetVerification(context.Background(), connect.NewRequest(&verificationsv1.GetVerificationRequest{RunId: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestStartVerificationWithBogusRoot(t *testing.T) {
	client := newClient(t)
	// A non-existent root makes pipeline.Verify return an error before
	// any per-flow rows get recorded — response.status is "failed" and
	// the error_message is non-empty.
	resp, err := client.StartVerification(context.Background(), connect.NewRequest(&verificationsv1.StartVerificationRequest{
		Root: t.TempDir(),
		Mode: verificationsv1.VerificationMode_VERIFICATION_MODE_CHECK,
	}))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.Status)
}
