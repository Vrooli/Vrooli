package skill_catalog

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

	skillcatalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/skill_catalog"
	skillcatalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/skill_catalog/skill_catalog_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "development-toolchain-validator/cli/internal/testutil"
)

type fakeService struct {
	mu       sync.Mutex
	syncResp *skillcatalogv1.SyncResponse
	listResp *skillcatalogv1.ListSkillsResponse
	getResp  *skillcatalogv1.GetSkillResponse

	syncErr error
	listErr error
	getErr  error

	syncCalls int
	getIDs    []string
}

func (s *fakeService) Sync(context.Context, *connect.Request[skillcatalogv1.SyncRequest]) (*connect.Response[skillcatalogv1.SyncResponse], error) {
	s.mu.Lock()
	s.syncCalls++
	s.mu.Unlock()
	if s.syncErr != nil {
		return nil, s.syncErr
	}
	if s.syncResp == nil {
		s.syncResp = &skillcatalogv1.SyncResponse{}
	}
	return connect.NewResponse(s.syncResp), nil
}

func (s *fakeService) ListSkills(context.Context, *connect.Request[skillcatalogv1.ListSkillsRequest]) (*connect.Response[skillcatalogv1.ListSkillsResponse], error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.listResp == nil {
		s.listResp = &skillcatalogv1.ListSkillsResponse{}
	}
	return connect.NewResponse(s.listResp), nil
}

func (s *fakeService) GetSkill(_ context.Context, req *connect.Request[skillcatalogv1.GetSkillRequest]) (*connect.Response[skillcatalogv1.GetSkillResponse], error) {
	s.mu.Lock()
	s.getIDs = append(s.getIDs, req.Msg.Id)
	s.mu.Unlock()
	if s.getErr != nil {
		return nil, s.getErr
	}
	return connect.NewResponse(s.getResp), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := skillcatalogconnect.NewSkillCatalogServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func sampleProto(id string) *skillcatalogv1.Skill {
	ts := timestamppb.New(time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC))
	return &skillcatalogv1.Skill{
		Id: id, Version: "2026-05-01T00:00:00Z", ContentHash: "deadbeef0123456789", SyncedAt: ts,
	}
}

func TestSync_ReportsCounts(t *testing.T) {
	svc := &fakeService{syncResp: &skillcatalogv1.SyncResponse{
		Skills: []*skillcatalogv1.Skill{sampleProto("implementation-plan-authoring")},
		Added:  1, Updated: 0, Removed: 0,
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.sync(ctx))
	require.Equal(t, 1, svc.syncCalls)
	require.Contains(t, out.String(), "Synced skill catalog")
	require.Contains(t, out.String(), "added=1")
}

func TestSync_SurfacesUnavailable(t *testing.T) {
	svc := &fakeService{syncErr: connect.NewError(connect.CodeUnavailable, io.ErrUnexpectedEOF)}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	err := h.sync(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unavailable")
}

func TestList_RendersResults(t *testing.T) {
	svc := &fakeService{listResp: &skillcatalogv1.ListSkillsResponse{
		Skills: []*skillcatalogv1.Skill{sampleProto("alpha"), sampleProto("bravo")},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.list(ctx))
	require.Contains(t, out.String(), "Mirroring 2 skill(s).")
	require.Contains(t, out.String(), "alpha")
	require.Contains(t, out.String(), "bravo")
}

func TestGet_PassesID(t *testing.T) {
	svc := &fakeService{getResp: &skillcatalogv1.GetSkillResponse{Skill: sampleProto("implementation-plan-authoring")}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
	}, cliapptest.TestRunContextOptions{Positionals: map[string]string{"id": "implementation-plan-authoring"}})
	require.NoError(t, h.get(ctx))
	require.Equal(t, []string{"implementation-plan-authoring"}, svc.getIDs)
	require.Contains(t, out.String(), "Fetched skill implementation-plan-authoring.")
}

func TestGet_SurfacesNotFound(t *testing.T) {
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
