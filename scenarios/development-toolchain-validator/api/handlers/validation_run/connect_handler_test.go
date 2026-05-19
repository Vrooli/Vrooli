package validation_run_test

import (
	"context"
	"testing"
	"time"

	vrunH "development-toolchain-validator/handlers/validation_run"
	vr "development-toolchain-validator/internal/validation_record"
	vrun "development-toolchain-validator/internal/validation_run"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	vrunv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_run"
	vrunconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_run/validation_run_v1connect"
	vrv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record"
)

type fakeService struct {
	StartOut      vrun.Run
	StartErr      error
	GetOut        vrun.Run
	GetErr        error
	ListOut       []vrun.Run
	ListErr       error
	StartSeenKind vr.TupleKind
}

func (f *fakeService) Start(_ context.Context, in vrun.StartInput) (vrun.Run, error) {
	f.StartSeenKind = in.TupleKind
	return f.StartOut, f.StartErr
}

func (f *fakeService) Get(context.Context, string) (vrun.Run, error) {
	return f.GetOut, f.GetErr
}

func (f *fakeService) ListActive(context.Context) ([]vrun.Run, error) {
	return f.ListOut, f.ListErr
}

var _ vrun.Service = (*fakeService)(nil)

func newClient(t *testing.T, svc vrun.Service) vrunconnect.ValidationRunServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := vrunconnect.NewValidationRunServiceHandler(vrunH.NewConnectHandler(vrunH.Deps{
		Service: svc, Logger: logger,
	}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return vrunconnect.NewValidationRunServiceClient(server.Client(), server.URL)
}

func TestStart_ReturnsQueued(t *testing.T) {
	svc := &fakeService{StartOut: vrun.Run{
		ID: "r1", TupleKind: vr.TupleKindSkill, SubjectID: "s", GoldenSlug: "g",
		Status: vrun.StatusQueued, CreatedAt: time.Now(),
	}}
	client := newClient(t, svc)
	resp, err := client.Start(context.Background(), connect.NewRequest(&vrunv1.StartRequest{
		TupleKind: vrv1.TupleKind_TUPLE_KIND_SKILL, SubjectId: "s", GoldenSlug: "g",
	}))
	require.NoError(t, err)
	require.Equal(t, "r1", resp.Msg.Run.Id)
	require.Equal(t, vrunv1.Status_STATUS_QUEUED, resp.Msg.Run.Status)
	require.Equal(t, vr.TupleKindSkill, svc.StartSeenKind)
}

func TestStart_InvalidMapsToInvalidArgument(t *testing.T) {
	client := newClient(t, &fakeService{StartErr: vrun.ErrInvalidRun{Field: "subject_id", Reason: "required"}})
	_, err := client.Start(context.Background(), connect.NewRequest(&vrunv1.StartRequest{
		TupleKind: vrv1.TupleKind_TUPLE_KIND_SKILL, GoldenSlug: "g",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestStart_UnavailableMapsToUnavailable(t *testing.T) {
	client := newClient(t, &fakeService{StartErr: vrun.ErrDependencyUnavailable{Dependency: "agent-manager", Reason: "not running"}})
	_, err := client.Start(context.Background(), connect.NewRequest(&vrunv1.StartRequest{
		TupleKind: vrv1.TupleKind_TUPLE_KIND_SKILL, SubjectId: "s", GoldenSlug: "g",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestGet_NotFound(t *testing.T) {
	client := newClient(t, &fakeService{GetErr: vrun.ErrRunNotFound{ID: "ghost"}})
	_, err := client.Get(context.Background(), connect.NewRequest(&vrunv1.GetRequest{Id: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestListActive_PassesThrough(t *testing.T) {
	client := newClient(t, &fakeService{ListOut: []vrun.Run{
		{ID: "r1", TupleKind: vr.TupleKindSkill, SubjectID: "s", GoldenSlug: "g", Status: vrun.StatusRunning, CreatedAt: time.Now()},
	}})
	resp, err := client.ListActive(context.Background(), connect.NewRequest(&vrunv1.ListActiveRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Runs, 1)
	require.Equal(t, vrunv1.Status_STATUS_RUNNING, resp.Msg.Runs[0].Status)
}
