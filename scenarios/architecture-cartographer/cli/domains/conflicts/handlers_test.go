package conflicts

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	conflictsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts"
	conflictsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts/conflicts_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "architecture-cartographer/cli/internal/testutil"
)

// fakeService implements ConflictsServiceHandler so the CLI tests exercise
// the real Connect-RPC client transport against an httptest server. Only
// the methods a given test drives are overridden; the rest inherit
// Unimplemented from the embedded base. The conflicts service is
// detection-only — there are no lifecycle RPCs to fake.
type fakeService struct {
	conflictsconnect.UnimplementedConflictsServiceHandler

	mu sync.Mutex

	detectResp   *conflictsv1.DetectConflictsResponse
	listResp     *conflictsv1.ListConflictsResponse
	listReqs     []*conflictsv1.ListConflictsRequest
	getResp      *conflictsv1.GetConflictResponse
	getReqs      []*conflictsv1.GetConflictRequest
	validateResp *conflictsv1.ValidateConflictsResponse
	detectorResp *conflictsv1.ListDetectorsResponse
	resolverResp *conflictsv1.ListResolversResponse
	err          error
}

func (s *fakeService) DetectConflicts(_ context.Context, _ *connect.Request[conflictsv1.DetectConflictsRequest]) (*connect.Response[conflictsv1.DetectConflictsResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.detectResp), nil
}

func (s *fakeService) ListConflicts(_ context.Context, req *connect.Request[conflictsv1.ListConflictsRequest]) (*connect.Response[conflictsv1.ListConflictsResponse], error) {
	s.mu.Lock()
	s.listReqs = append(s.listReqs, req.Msg)
	s.mu.Unlock()
	return connect.NewResponse(s.listResp), nil
}

func (s *fakeService) GetConflict(_ context.Context, req *connect.Request[conflictsv1.GetConflictRequest]) (*connect.Response[conflictsv1.GetConflictResponse], error) {
	s.mu.Lock()
	s.getReqs = append(s.getReqs, req.Msg)
	s.mu.Unlock()
	return connect.NewResponse(s.getResp), nil
}

func (s *fakeService) ValidateConflicts(_ context.Context, _ *connect.Request[conflictsv1.ValidateConflictsRequest]) (*connect.Response[conflictsv1.ValidateConflictsResponse], error) {
	return connect.NewResponse(s.validateResp), nil
}

func (s *fakeService) ListDetectors(_ context.Context, _ *connect.Request[conflictsv1.ListDetectorsRequest]) (*connect.Response[conflictsv1.ListDetectorsResponse], error) {
	return connect.NewResponse(s.detectorResp), nil
}

func (s *fakeService) ListResolvers(_ context.Context, _ *connect.Request[conflictsv1.ListResolversRequest]) (*connect.Response[conflictsv1.ListResolversResponse], error) {
	return connect.NewResponse(s.resolverResp), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := conflictsconnect.NewConflictsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func sampleConflict() *sharedv1.Conflict {
	return &sharedv1.Conflict{
		Id:        "c-1",
		Scenario:  "demo",
		Type:      "cycle",
		Severity:  sharedv1.Severity_SEVERITY_ERROR,
		Locations: []string{"api/internal/a", "api/internal/b"},
	}
}

func TestDetect_RendersConflictList(t *testing.T) {
	svc := &fakeService{detectResp: &conflictsv1.DetectConflictsResponse{
		Conflicts: []*sharedv1.Conflict{sampleConflict()},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
		Flags:       []cliapp.Flag{{Name: "snapshot-id"}, {Name: "idempotency-key"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo"},
	})

	require.NoError(t, h.detect(ctx))
	body := out.String()
	require.Contains(t, body, "Detected 1 conflict(s)")
	require.Contains(t, body, "c-1")
	require.Contains(t, body, "error")
}

func TestList_PassesTypeAndPageSize(t *testing.T) {
	svc := &fakeService{listResp: &conflictsv1.ListConflictsResponse{}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, listSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo"},
		Flags:       map[string]string{"type": "cycle,mislocated_file", "page-size": "5"},
	})

	require.NoError(t, h.list(ctx))
	require.Len(t, svc.listReqs, 1)
	require.Equal(t, []string{"cycle", "mislocated_file"}, svc.listReqs[0].GetTypes())
	require.Equal(t, int32(5), svc.listReqs[0].GetPageSize())
}

func TestValidate_RendersCleanGate(t *testing.T) {
	svc := &fakeService{validateResp: &conflictsv1.ValidateConflictsResponse{Clean: true}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo"},
	})

	require.NoError(t, h.validate(ctx))
	require.Contains(t, out.String(), "cartographer-clean")
}

func TestDetectors_ListsRegistry(t *testing.T) {
	svc := &fakeService{detectorResp: &conflictsv1.ListDetectorsResponse{
		Detectors: []*conflictsv1.DetectorDescriptor{
			{Name: "cycle", Stability: "stable", EmitsTypes: []string{"cycle"}, Description: "import cycles"},
		},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	require.NoError(t, h.detectors(ctx))
	require.Contains(t, out.String(), "cycle")
	require.Contains(t, out.String(), "import cycles")
}

func TestDetect_SurfacesConnectErrors(t *testing.T) {
	svc := &fakeService{err: connect.NewError(connect.CodeInvalidArgument, io.ErrUnexpectedEOF)}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
		Flags:       []cliapp.Flag{{Name: "snapshot-id"}, {Name: "idempotency-key"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo"},
	})

	err := h.detect(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid_argument")
}

// TestShow_NormalizesShortStableID asserts the CLI prefixes a bare
// 16-hex short form with "csid:" before calling the API. Agents
// reflexively copy the hex digits without the prefix; this closes
// the "stable-ID UX trap" (Plan Problem 4).
func TestShow_NormalizesShortStableID(t *testing.T) {
	cases := []struct {
		name string
		arg  string
		want string
	}{
		{"short hex", "16ee6eb253627c0e", "csid:16ee6eb253627c0e"},
		{"already prefixed", "csid:16ee6eb253627c0e", "csid:16ee6eb253627c0e"},
		{"upper-case hex", "16EE6EB253627C0E", "csid:16EE6EB253627C0E"},
		{"non-hex passes through", "abc-123-uuid", "abc-123-uuid"},
		{"wrong length passes through", "1234abcd", "1234abcd"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeService{getResp: &conflictsv1.GetConflictResponse{Conflict: sampleConflict()}}
			core := clitest.NewTestApp(t, connectAPI(t, svc))
			h := newHandlers(core)
			ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
				Positionals: []cliapp.Positional{{Name: "id", Required: true}},
			}, cliapptest.TestRunContextOptions{
				Positionals: map[string]string{"id": tc.arg},
			})

			require.NoError(t, h.show(ctx))
			require.Len(t, svc.getReqs, 1)
			require.Equal(t, tc.want, svc.getReqs[0].GetId())
		})
	}
}

// TestNormalizeConflictID_Unit gives the function a direct unit test
// so failure points are localized when the integration assertion fires.
func TestNormalizeConflictID_Unit(t *testing.T) {
	cases := map[string]string{
		"16ee6eb253627c0e":      "csid:16ee6eb253627c0e",
		"csid:16ee6eb253627c0e": "csid:16ee6eb253627c0e",
		"  16ee6eb253627c0e  ":  "csid:16ee6eb253627c0e",
		"not-hex-at-all":        "not-hex-at-all",
		"abcd":                  "abcd",
		"":                      "",
	}
	for in, want := range cases {
		if got := normalizeConflictID(in); got != want {
			t.Errorf("normalizeConflictID(%q)=%q want %q", in, got, want)
		}
	}
}

func listSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
		Flags: []cliapp.Flag{
			{Name: "type"}, {Name: "page-size"}, {Name: "page-token"},
		},
	}
}
