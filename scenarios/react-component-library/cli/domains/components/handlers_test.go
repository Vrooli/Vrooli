package components

import (
	"context"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "react-component-library/cli/internal/testutil"
)

// componentsService is a hand-written ComponentsServiceHandler used as a fake
// API behind the Connect mux. Mirrors the notes-domain test stub shape.
type componentsService struct {
	mu              sync.Mutex
	listResp        *componentsv1.ListComponentsResponse
	getResp         *componentsv1.GetComponentResponse
	byLibIDResp     *componentsv1.GetComponentByLibraryIdResponse
	indexResp       *componentsv1.IndexComponentsResponse
	contentGetResp  *componentsv1.GetComponentContentResponse
	contentSetResp  *componentsv1.UpdateComponentContentResponse
	listErr         error
	getErr          error
	byLibIDErr      error
	indexErr        error
	contentGetErr   error
	contentSetErr   error
	listReqs        []*componentsv1.ListComponentsRequest
	getReqs         []string
	byLibIDReqs     []string
	contentSetReqs  []*componentsv1.UpdateComponentContentRequest
}

func (s *componentsService) ListComponents(_ context.Context, req *connect.Request[componentsv1.ListComponentsRequest]) (*connect.Response[componentsv1.ListComponentsResponse], error) {
	s.mu.Lock()
	s.listReqs = append(s.listReqs, req.Msg)
	s.mu.Unlock()
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.listResp == nil {
		s.listResp = &componentsv1.ListComponentsResponse{}
	}
	return connect.NewResponse(s.listResp), nil
}

func (s *componentsService) GetComponent(_ context.Context, req *connect.Request[componentsv1.GetComponentRequest]) (*connect.Response[componentsv1.GetComponentResponse], error) {
	s.mu.Lock()
	s.getReqs = append(s.getReqs, req.Msg.Id)
	s.mu.Unlock()
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getResp == nil {
		s.getResp = &componentsv1.GetComponentResponse{}
	}
	return connect.NewResponse(s.getResp), nil
}

func (s *componentsService) GetComponentByLibraryId(_ context.Context, req *connect.Request[componentsv1.GetComponentByLibraryIdRequest]) (*connect.Response[componentsv1.GetComponentByLibraryIdResponse], error) {
	s.mu.Lock()
	s.byLibIDReqs = append(s.byLibIDReqs, req.Msg.LibraryId)
	s.mu.Unlock()
	if s.byLibIDErr != nil {
		return nil, s.byLibIDErr
	}
	if s.byLibIDResp == nil {
		s.byLibIDResp = &componentsv1.GetComponentByLibraryIdResponse{}
	}
	return connect.NewResponse(s.byLibIDResp), nil
}

func (s *componentsService) IndexComponents(_ context.Context, _ *connect.Request[componentsv1.IndexComponentsRequest]) (*connect.Response[componentsv1.IndexComponentsResponse], error) {
	if s.indexErr != nil {
		return nil, s.indexErr
	}
	if s.indexResp == nil {
		s.indexResp = &componentsv1.IndexComponentsResponse{}
	}
	return connect.NewResponse(s.indexResp), nil
}

func (s *componentsService) GetComponentContent(_ context.Context, _ *connect.Request[componentsv1.GetComponentContentRequest]) (*connect.Response[componentsv1.GetComponentContentResponse], error) {
	if s.contentGetErr != nil {
		return nil, s.contentGetErr
	}
	if s.contentGetResp == nil {
		s.contentGetResp = &componentsv1.GetComponentContentResponse{}
	}
	return connect.NewResponse(s.contentGetResp), nil
}

func (s *componentsService) UpdateComponentContent(_ context.Context, req *connect.Request[componentsv1.UpdateComponentContentRequest]) (*connect.Response[componentsv1.UpdateComponentContentResponse], error) {
	s.mu.Lock()
	s.contentSetReqs = append(s.contentSetReqs, req.Msg)
	s.mu.Unlock()
	if s.contentSetErr != nil {
		return nil, s.contentSetErr
	}
	if s.contentSetResp == nil {
		s.contentSetResp = &componentsv1.UpdateComponentContentResponse{}
	}
	return connect.NewResponse(s.contentSetResp), nil
}

func connectAPI(t *testing.T, svc *componentsService) http.Handler {
	t.Helper()
	path, handler := componentsconnect.NewComponentsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func sampleComponent() *componentsv1.Component {
	ts := timestamppb.New(time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC))
	return &componentsv1.Component{
		Id:          "abc",
		LibraryId:   "lib:Button",
		DisplayName: "Button",
		Description: "CTA",
		SourcePath:  "components/Button.tsx",
		Version:     "1.0.0",
		Tags:        []string{"form"},
		IndexedAt:   ts,
		UpdatedAt:   ts,
	}
}

func TestComponentsIndex_HumanReport(t *testing.T) {
	svc := &componentsService{indexResp: &componentsv1.IndexComponentsResponse{
		Scanned: 3, Indexed: 2, Skipped: 1, Deleted: 0,
		LibraryIds: []string{"lib:Button", "lib:Card"},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	require.NoError(t, h.index(ctx))
	body := out.String()
	require.Contains(t, body, "Scanned 3 file(s); indexed 2, skipped 1, deleted 0.")
	require.Contains(t, body, "lib:Button")
	require.Contains(t, body, "lib:Card")
}

func TestComponentsList_ForwardsFiltersAndRenders(t *testing.T) {
	svc := &componentsService{listResp: &componentsv1.ListComponentsResponse{
		Components: []*componentsv1.Component{sampleComponent()},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "match"}, {Name: "tag"}, {Name: "tags"}, {Name: "category"}, {Name: "limit"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"match": "btn", "tag": "form", "limit": "50"},
	})

	require.NoError(t, h.list(ctx))
	require.Len(t, svc.listReqs, 1)
	require.Equal(t, "btn", svc.listReqs[0].Match)
	require.Equal(t, "form", svc.listReqs[0].Tag)
	require.Equal(t, int32(50), svc.listReqs[0].Limit)
	require.Empty(t, svc.listReqs[0].Tags)
	require.Empty(t, svc.listReqs[0].Category)
	require.Contains(t, out.String(), "Found 1 component(s).")
	require.Contains(t, out.String(), "lib:Button")
	require.Contains(t, out.String(), "v1.0.0")
}

func TestComponentsList_ForwardsMultiTagAndCategory(t *testing.T) {
	svc := &componentsService{listResp: &componentsv1.ListComponentsResponse{
		Components: []*componentsv1.Component{sampleComponent()},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "match"}, {Name: "tag"}, {Name: "tags"}, {Name: "category"}, {Name: "limit"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"tags": " form , , layout ", "category": "controls"},
	})

	require.NoError(t, h.list(ctx))
	require.Len(t, svc.listReqs, 1)
	require.Equal(t, []string{"form", "layout"}, svc.listReqs[0].Tags,
		"comma-separated --tags is parsed and trimmed; blanks dropped")
	require.Equal(t, "controls", svc.listReqs[0].Category)
}

func TestComponentsList_RejectsBadLimit(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &componentsService{}))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "match"}, {Name: "tag"}, {Name: "tags"}, {Name: "category"}, {Name: "limit"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"limit": "abc"},
	})
	err := h.list(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--limit must be an integer")
}

// TestComponentsList_JSONIsProtoWireShape pins the contract that --json output
// is the proto-typed ListComponentsResponse wire shape.
func TestComponentsList_JSONIsProtoWireShape(t *testing.T) {
	svc := &componentsService{listResp: &componentsv1.ListComponentsResponse{
		Components: []*componentsv1.Component{sampleComponent()},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "match"}, {Name: "tag"}, {Name: "tags"}, {Name: "category"}, {Name: "limit"}},
	}, cliapptest.TestRunContextOptions{JSON: true})

	require.NoError(t, h.list(ctx))

	body := out.String()
	require.NotContains(t, body, "summary",
		"--json output must be proto wire shape, not the human ListReport wrapper")
	require.NotContains(t, body, "retrieval_hints")

	var got componentsv1.ListComponentsResponse
	require.NoError(t, protojson.Unmarshal(out.Bytes(), &got))
	require.Len(t, got.Components, 1)
	require.Equal(t, "lib:Button", got.Components[0].LibraryId)
}

func TestComponentsGetByLibraryID_Fetches(t *testing.T) {
	svc := &componentsService{byLibIDResp: &componentsv1.GetComponentByLibraryIdResponse{
		Component: sampleComponent(),
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "library-id"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"library-id": "lib:Button"},
	})
	require.NoError(t, h.getByLibraryID(ctx))
	require.Equal(t, []string{"lib:Button"}, svc.byLibIDReqs)
	require.Contains(t, out.String(), "Fetched component lib:Button.")
}

func TestComponentsContentGet_PrintsBody(t *testing.T) {
	svc := &componentsService{contentGetResp: &componentsv1.GetComponentContentResponse{
		Content:    "// hello",
		SourcePath: "Button.tsx",
		Sha256:     "abc123",
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "abc"},
	})

	require.NoError(t, h.contentGet(ctx))
	body := out.String()
	require.Contains(t, body, "// hello")
	require.Contains(t, body, "sha256=abc123")
}

func TestComponentsContentSet_FromFile(t *testing.T) {
	svc := &componentsService{contentSetResp: &componentsv1.UpdateComponentContentResponse{
		Sha256:     "deadbeef",
		SourcePath: "Button.tsx",
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)

	tmpFile := t.TempDir() + "/new.tsx"
	require.NoError(t, writeFile(tmpFile, "// rewritten\n"))

	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}, {Name: "file", Required: true}},
		Flags:       []cliapp.Flag{{Name: "expected-sha256"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "abc", "file": tmpFile},
		Flags:       map[string]string{"expected-sha256": "stale"},
	})
	require.NoError(t, h.contentSet(ctx))
	require.Len(t, svc.contentSetReqs, 1)
	require.Equal(t, "abc", svc.contentSetReqs[0].Id)
	require.Equal(t, "// rewritten\n", svc.contentSetReqs[0].Content)
	require.Equal(t, "stale", svc.contentSetReqs[0].ExpectedSha256)
	require.Contains(t, out.String(), "sha256=deadbeef")
}

func writeFile(path, body string) error {
	return os.WriteFile(path, []byte(body), 0o600)
}

func TestComponentsGet_ReportsNotFound(t *testing.T) {
	svc := &componentsService{getErr: connect.NewError(connect.CodeNotFound, io.ErrUnexpectedEOF)}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "ghost"},
	})

	err := h.get(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not_found")
}
