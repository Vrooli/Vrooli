package adoptions

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"
	adoptionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions/adoptions_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "react-component-library/cli/internal/testutil"
)

type adoptionsService struct {
	mu          sync.Mutex
	listResp    *adoptionsv1.ListAdoptionsResponse
	createResp  *adoptionsv1.CreateAdoptionResponse
	deleteResp  *adoptionsv1.DeleteAdoptionResponse
	refreshResp *adoptionsv1.RefreshAdoptionsResponse
	listReqs    []*adoptionsv1.ListAdoptionsRequest
	createReqs  []*adoptionsv1.CreateAdoptionRequest
	refreshReqs []*adoptionsv1.RefreshAdoptionsRequest
}

func (s *adoptionsService) ListAdoptions(_ context.Context, req *connect.Request[adoptionsv1.ListAdoptionsRequest]) (*connect.Response[adoptionsv1.ListAdoptionsResponse], error) {
	s.mu.Lock()
	s.listReqs = append(s.listReqs, req.Msg)
	s.mu.Unlock()
	if s.listResp == nil {
		s.listResp = &adoptionsv1.ListAdoptionsResponse{}
	}
	return connect.NewResponse(s.listResp), nil
}

func (s *adoptionsService) CreateAdoption(_ context.Context, req *connect.Request[adoptionsv1.CreateAdoptionRequest]) (*connect.Response[adoptionsv1.CreateAdoptionResponse], error) {
	s.mu.Lock()
	s.createReqs = append(s.createReqs, req.Msg)
	s.mu.Unlock()
	if s.createResp == nil {
		s.createResp = &adoptionsv1.CreateAdoptionResponse{Adoption: sampleAdoption()}
	}
	return connect.NewResponse(s.createResp), nil
}

func (s *adoptionsService) DeleteAdoption(_ context.Context, _ *connect.Request[adoptionsv1.DeleteAdoptionRequest]) (*connect.Response[adoptionsv1.DeleteAdoptionResponse], error) {
	if s.deleteResp == nil {
		s.deleteResp = &adoptionsv1.DeleteAdoptionResponse{}
	}
	return connect.NewResponse(s.deleteResp), nil
}

func (s *adoptionsService) RefreshAdoptions(_ context.Context, req *connect.Request[adoptionsv1.RefreshAdoptionsRequest]) (*connect.Response[adoptionsv1.RefreshAdoptionsResponse], error) {
	s.mu.Lock()
	s.refreshReqs = append(s.refreshReqs, req.Msg)
	s.mu.Unlock()
	if s.refreshResp == nil {
		s.refreshResp = &adoptionsv1.RefreshAdoptionsResponse{}
	}
	return connect.NewResponse(s.refreshResp), nil
}

func connectAPI(t *testing.T, svc *adoptionsService) http.Handler {
	t.Helper()
	path, handler := adoptionsconnect.NewAdoptionsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func sampleAdoption() *adoptionsv1.Adoption {
	ts := timestamppb.New(time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC))
	return &adoptionsv1.Adoption{
		Id:             "ad-1",
		ComponentId:    "cmp-btn",
		LibraryId:      "rcl:Button",
		Scenario:       "swarm-manager",
		AdoptedPath:    "ui/Button.tsx",
		AdoptedVersion: "1.0.0",
		Status:         adoptionsv1.AdoptionStatus_ADOPTION_STATUS_BEHIND,
		StatusDetail:   "library at 1.1.0",
		CreatedAt:      ts,
		RefreshedAt:    ts,
	}
}

func TestAdoptionsList_ForwardsFiltersAndRenders(t *testing.T) {
	svc := &adoptionsService{listResp: &adoptionsv1.ListAdoptionsResponse{
		Adoptions: []*adoptionsv1.Adoption{sampleAdoption()},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "component-id"}, {Name: "scenario"}, {Name: "limit"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"scenario": "swarm-manager", "limit": "50"},
	})
	require.NoError(t, h.list(ctx))
	require.Len(t, svc.listReqs, 1)
	require.Equal(t, "swarm-manager", svc.listReqs[0].Scenario)
	require.Equal(t, int32(50), svc.listReqs[0].Limit)
	require.Contains(t, out.String(), "Found 1 adoption(s).")
	require.Contains(t, out.String(), "behind")
}

func TestAdoptionsList_RejectsBadLimit(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &adoptionsService{}))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "component-id"}, {Name: "scenario"}, {Name: "limit"}},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{"limit": "x"}})
	require.Error(t, h.list(ctx))
}

func TestAdoptionsCreate_ForwardsPositionalsAndVersion(t *testing.T) {
	svc := &adoptionsService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "component-id"}, {Name: "scenario"}, {Name: "adopted-path"}},
		Flags:       []cliapp.Flag{{Name: "adopted-version"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{
			"component-id": "cmp-btn",
			"scenario":     "swarm-manager",
			"adopted-path": "ui/Button.tsx",
		},
		Flags: map[string]string{"adopted-version": "1.0.0"},
	})
	require.NoError(t, h.create(ctx))
	require.Len(t, svc.createReqs, 1)
	require.Equal(t, "cmp-btn", svc.createReqs[0].ComponentId)
	require.Equal(t, "ui/Button.tsx", svc.createReqs[0].AdoptedPath)
	require.Equal(t, "1.0.0", svc.createReqs[0].AdoptedVersion)
}

func TestAdoptionsRefresh_SummaryLineIncludesCounts(t *testing.T) {
	svc := &adoptionsService{refreshResp: &adoptionsv1.RefreshAdoptionsResponse{
		Adoptions: []*adoptionsv1.Adoption{sampleAdoption()},
		Current:   2, Behind: 1, Modified: 0, Unknown: 1,
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "component-id"}},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{"component-id": "cmp-btn"}})
	require.NoError(t, h.refresh(ctx))
	require.Len(t, svc.refreshReqs, 1)
	require.Equal(t, "cmp-btn", svc.refreshReqs[0].ComponentId)
	body := out.String()
	require.Contains(t, body, "current=2")
	require.Contains(t, body, "behind=1")
	require.Contains(t, body, "unknown=1")
}
