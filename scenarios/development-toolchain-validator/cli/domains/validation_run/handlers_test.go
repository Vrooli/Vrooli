package validation_run

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	vrv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record"
	vrunv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_run"
	vrunconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_run/validation_run_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "development-toolchain-validator/cli/internal/testutil"
)

type fakeService struct {
	mu sync.Mutex

	startResp *vrunv1.StartResponse
	getResp   *vrunv1.GetResponse
	listResp  *vrunv1.ListActiveResponse

	startErr error
	getErr   error
	listErr  error

	startReq *vrunv1.StartRequest
}

func (f *fakeService) Start(_ context.Context, req *connect.Request[vrunv1.StartRequest]) (*connect.Response[vrunv1.StartResponse], error) {
	f.mu.Lock()
	f.startReq = req.Msg
	f.mu.Unlock()
	if f.startErr != nil {
		return nil, f.startErr
	}
	return connect.NewResponse(f.startResp), nil
}

func (f *fakeService) Get(_ context.Context, _ *connect.Request[vrunv1.GetRequest]) (*connect.Response[vrunv1.GetResponse], error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return connect.NewResponse(f.getResp), nil
}

func (f *fakeService) ListActive(_ context.Context, _ *connect.Request[vrunv1.ListActiveRequest]) (*connect.Response[vrunv1.ListActiveResponse], error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResp == nil {
		f.listResp = &vrunv1.ListActiveResponse{}
	}
	return connect.NewResponse(f.listResp), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := vrunconnect.NewValidationRunServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func sampleRun() *vrunv1.ValidationRun {
	return &vrunv1.ValidationRun{
		Id: "r1", TupleKind: vrv1.TupleKind_TUPLE_KIND_SKILL,
		SubjectId: "plan-skill-discovery", GoldenSlug: "reference-react-vite",
		Status:    vrunv1.Status_STATUS_QUEUED,
		CreatedAt: timestamppb.New(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)),
	}
}

func TestStart_SkillFlag(t *testing.T) {
	svc := &fakeService{startResp: &vrunv1.StartResponse{Run: sampleRun()}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{
			{Name: "skill"},
			{Name: "tool"},
			{Name: "golden", Required: true},
			{Name: "force", Bool: true},
			{Name: "wait", Bool: true},
			{Name: "wait-timeout", Default: "300"},
		},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{
		"skill":  "plan-skill-discovery",
		"golden": "reference-react-vite",
	}})
	require.NoError(t, h.start(ctx))
	require.Equal(t, vrv1.TupleKind_TUPLE_KIND_SKILL, svc.startReq.TupleKind)
	require.Equal(t, "plan-skill-discovery", svc.startReq.SubjectId)
	require.Contains(t, out.String(), "Queued run r1.")
}

func TestStart_RequiresExactlyOneSubject(t *testing.T) {
	svc := &fakeService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{
			{Name: "skill"},
			{Name: "tool"},
			{Name: "golden", Required: true},
			{Name: "force", Bool: true},
			{Name: "wait", Bool: true},
			{Name: "wait-timeout", Default: "300"},
		},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{
		"skill":  "a",
		"tool":   "b",
		"golden": "g",
	}})
	require.Error(t, h.start(ctx))
}

func TestListActive_RendersResults(t *testing.T) {
	svc := &fakeService{listResp: &vrunv1.ListActiveResponse{Runs: []*vrunv1.ValidationRun{sampleRun()}}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.listActive(ctx))
	require.Contains(t, out.String(), "Found 1 active run(s).")
}
