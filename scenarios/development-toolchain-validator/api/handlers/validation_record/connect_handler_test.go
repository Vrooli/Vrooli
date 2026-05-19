package validation_record_test

import (
	"context"
	"testing"
	"time"

	vrH "development-toolchain-validator/handlers/validation_record"
	vr "development-toolchain-validator/internal/validation_record"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	vrv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record"
	vrconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record/validation_record_v1connect"
)

type fakeService struct {
	ListOut   vr.ListResult
	ListErr   error
	GetOut    vr.Record
	GetErr    error
	GetID     string
	ListSeen  vr.ListFilter
	ListPage  int
	ListToken string
}

func (f *fakeService) Append(_ context.Context, _ vr.AppendInput) (vr.Record, error) {
	panic("not used in handler tests")
}

func (f *fakeService) Get(_ context.Context, id string) (vr.Record, error) {
	f.GetID = id
	return f.GetOut, f.GetErr
}

func (f *fakeService) List(_ context.Context, fl vr.ListFilter, pageSize int, pageToken string) (vr.ListResult, error) {
	f.ListSeen = fl
	f.ListPage = pageSize
	f.ListToken = pageToken
	return f.ListOut, f.ListErr
}

var _ vr.Service = (*fakeService)(nil)

func newClient(t *testing.T, svc vr.Service) vrconnect.ValidationRecordServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := vrconnect.NewValidationRecordServiceHandler(vrH.NewConnectHandler(vrH.Deps{
		Service: svc, Logger: logger,
	}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return vrconnect.NewValidationRecordServiceClient(server.Client(), server.URL)
}

func TestList_PassesFiltersAndReturnsCursor(t *testing.T) {
	svc := &fakeService{ListOut: vr.ListResult{
		Records: []vr.Record{{ID: "r1", TupleKind: vr.TupleKindSkill, SubjectID: "s", GoldenSlug: "g", Verdict: vr.VerdictPass, EndedAt: time.Now()}},
		NextPageToken: "tok",
	}}
	client := newClient(t, svc)
	resp, err := client.ListRecords(context.Background(), connect.NewRequest(&vrv1.ListRecordsRequest{
		GoldenSlug: "g", SubjectId: "s", TupleKind: vrv1.TupleKind_TUPLE_KIND_SKILL,
		PageSize: 10, PageToken: "in",
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Records, 1)
	require.Equal(t, "tok", resp.Msg.NextPageToken)
	require.Equal(t, "g", svc.ListSeen.GoldenSlug)
	require.Equal(t, vr.TupleKindSkill, svc.ListSeen.TupleKind)
	require.Equal(t, 10, svc.ListPage)
	require.Equal(t, "in", svc.ListToken)
}

func TestGet_NotFound(t *testing.T) {
	client := newClient(t, &fakeService{GetErr: vr.ErrRecordNotFound{ID: "missing"}})
	_, err := client.GetRecord(context.Background(), connect.NewRequest(&vrv1.GetRecordRequest{Id: "missing"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestGet_Invalid(t *testing.T) {
	client := newClient(t, &fakeService{GetErr: vr.ErrInvalidRecord{Field: "id", Reason: "required"}})
	_, err := client.GetRecord(context.Background(), connect.NewRequest(&vrv1.GetRecordRequest{Id: ""}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
