package validation_record

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	vrv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record"
	vrconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record/validation_record_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "development-toolchain-validator/cli/internal/testutil"
)

type fakeService struct {
	mu sync.Mutex

	listResp *vrv1.ListRecordsResponse
	getResp  *vrv1.GetRecordResponse

	listErr error
	getErr  error

	listReq *vrv1.ListRecordsRequest
	getReq  *vrv1.GetRecordRequest
}

func (f *fakeService) ListRecords(_ context.Context, req *connect.Request[vrv1.ListRecordsRequest]) (*connect.Response[vrv1.ListRecordsResponse], error) {
	f.mu.Lock()
	f.listReq = req.Msg
	f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResp == nil {
		f.listResp = &vrv1.ListRecordsResponse{}
	}
	return connect.NewResponse(f.listResp), nil
}

func (f *fakeService) GetRecord(_ context.Context, req *connect.Request[vrv1.GetRecordRequest]) (*connect.Response[vrv1.GetRecordResponse], error) {
	f.mu.Lock()
	f.getReq = req.Msg
	f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	return connect.NewResponse(f.getResp), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := vrconnect.NewValidationRecordServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func sample(id string) *vrv1.ValidationRecord {
	return &vrv1.ValidationRecord{
		Id:         id,
		TupleKind:  vrv1.TupleKind_TUPLE_KIND_SKILL,
		SubjectId:  "implementation-plan-authoring",
		GoldenSlug: "reference-react-vite",
		EndedAt:    timestamppb.New(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)),
		Verdict:    vrv1.Verdict_VERDICT_PASS,
		DurationMs: 1234,
	}
}

func TestList_RendersResults(t *testing.T) {
	svc := &fakeService{listResp: &vrv1.ListRecordsResponse{
		Records:       []*vrv1.ValidationRecord{sample("r1"), sample("r2")},
		NextPageToken: "tok",
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{
			{Name: "golden"},
			{Name: "subject"},
			{Name: "kind"},
			{Name: "page-size", Default: "50"},
			{Name: "page-token"},
		},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{
		"golden":    "reference-react-vite",
		"kind":      "skill",
		"page-size": "5",
	}})
	require.NoError(t, h.list(ctx))
	require.Equal(t, "reference-react-vite", svc.listReq.GoldenSlug)
	require.Equal(t, vrv1.TupleKind_TUPLE_KIND_SKILL, svc.listReq.TupleKind)
	require.Equal(t, int32(5), svc.listReq.PageSize)
	require.Contains(t, out.String(), "Found 2 record(s).")
	require.Contains(t, out.String(), "tok")
}

func TestGet_PassesID(t *testing.T) {
	svc := &fakeService{getResp: &vrv1.GetRecordResponse{Record: sample("r1")}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
	}, cliapptest.TestRunContextOptions{Positionals: map[string]string{"id": "r1"}})
	require.NoError(t, h.get(ctx))
	require.Equal(t, "r1", svc.getReq.Id)
	require.Contains(t, out.String(), "Fetched record r1.")
}

func TestGet_NotFoundSurfaced(t *testing.T) {
	svc := &fakeService{getErr: connect.NewError(connect.CodeNotFound, io.ErrUnexpectedEOF)}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
	}, cliapptest.TestRunContextOptions{Positionals: map[string]string{"id": "ghost"}})
	err := h.get(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not_found")
}
