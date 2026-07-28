package verify

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "flow-verifier/cli/internal/testutil"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/runs"
	verificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/verifications"
	verificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/verifications/verifications_v1connect"
)

type fakeService struct {
	startResp *verificationsv1.StartVerificationResponse
	getResp   *verificationsv1.GetVerificationResponse
	inputs    []*verificationsv1.StartVerificationRequest
}

func (f *fakeService) StartVerification(_ context.Context, req *connect.Request[verificationsv1.StartVerificationRequest]) (*connect.Response[verificationsv1.StartVerificationResponse], error) {
	f.inputs = append(f.inputs, req.Msg)
	if f.startResp == nil {
		f.startResp = &verificationsv1.StartVerificationResponse{Status: "passed"}
	}
	return connect.NewResponse(f.startResp), nil
}

func (f *fakeService) GetVerification(context.Context, *connect.Request[verificationsv1.GetVerificationRequest]) (*connect.Response[verificationsv1.GetVerificationResponse], error) {
	if f.getResp == nil {
		f.getResp = &verificationsv1.GetVerificationResponse{Run: &runsv1.Run{Id: "r1", FlowId: "f1"}}
	}
	return connect.NewResponse(f.getResp), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := verificationsconnect.NewVerificationsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func TestVerifyRun_PassesGenerateMode(t *testing.T) {
	svc := &fakeService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "root"}, {Name: "flow"}}}
	ctx, _ := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{Flags: map[string]string{"root": "."}})

	require.NoError(t, h.run(ctx))
	require.Len(t, svc.inputs, 1)
	require.Equal(t, verificationsv1.VerificationMode_VERIFICATION_MODE_GENERATE, svc.inputs[0].Mode)
	wantRoot, err := filepath.Abs(".")
	require.NoError(t, err)
	require.Equal(t, wantRoot, svc.inputs[0].Root)
}

func TestVerifyCheck_PassesCheckMode(t *testing.T) {
	svc := &fakeService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "root"}, {Name: "flow"}}}
	ctx, _ := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{Flags: map[string]string{"root": "."}})

	require.NoError(t, h.check(ctx))
	require.Equal(t, verificationsv1.VerificationMode_VERIFICATION_MODE_CHECK, svc.inputs[0].Mode)
	wantRoot, err := filepath.Abs(".")
	require.NoError(t, err)
	require.Equal(t, wantRoot, svc.inputs[0].Root)
}

func TestVerifyShow_RendersRun(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &fakeService{}))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "run-id", Required: true}}}
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"run-id": "r1"}})

	require.NoError(t, h.show(ctx))
	require.Contains(t, out.String(), "r1")
}
