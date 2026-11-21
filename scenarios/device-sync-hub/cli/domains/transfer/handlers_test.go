package transfer

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	transferv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer"
	transferconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer/transfer_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

const testDeviceToken = "hub-token-xyz"

// transferService is a hand-rolled fake that records inputs and the device
// token header observed on each call.
type transferService struct {
	mu sync.Mutex

	createResp *transferv1.CreateTextItemResponse
	listResp   *transferv1.ListItemsResponse
	getResp    *transferv1.GetItemResponse
	deleteResp *transferv1.DeleteItemResponse

	listErr error

	createInputs []*transferv1.CreateTextItemRequest
	listInputs   []*transferv1.ListItemsRequest
	deleteInputs []string
	tokens       []string
}

func (s *transferService) recordToken(req interface{ Header() http.Header }) {
	s.mu.Lock()
	s.tokens = append(s.tokens, req.Header().Get("X-Device-Token"))
	s.mu.Unlock()
}

func (s *transferService) CreateTextItem(_ context.Context, req *connect.Request[transferv1.CreateTextItemRequest]) (*connect.Response[transferv1.CreateTextItemResponse], error) {
	s.recordToken(req)
	s.mu.Lock()
	s.createInputs = append(s.createInputs, req.Msg)
	s.mu.Unlock()
	if s.createResp == nil {
		s.createResp = &transferv1.CreateTextItemResponse{}
	}
	return connect.NewResponse(s.createResp), nil
}

func (s *transferService) ListItems(_ context.Context, req *connect.Request[transferv1.ListItemsRequest]) (*connect.Response[transferv1.ListItemsResponse], error) {
	s.recordToken(req)
	s.mu.Lock()
	s.listInputs = append(s.listInputs, req.Msg)
	s.mu.Unlock()
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.listResp == nil {
		s.listResp = &transferv1.ListItemsResponse{}
	}
	return connect.NewResponse(s.listResp), nil
}

func (s *transferService) GetItem(_ context.Context, req *connect.Request[transferv1.GetItemRequest]) (*connect.Response[transferv1.GetItemResponse], error) {
	s.recordToken(req)
	if s.getResp == nil {
		s.getResp = &transferv1.GetItemResponse{}
	}
	return connect.NewResponse(s.getResp), nil
}

func (s *transferService) DeleteItem(_ context.Context, req *connect.Request[transferv1.DeleteItemRequest]) (*connect.Response[transferv1.DeleteItemResponse], error) {
	s.recordToken(req)
	s.mu.Lock()
	s.deleteInputs = append(s.deleteInputs, req.Msg.Id)
	s.mu.Unlock()
	if s.deleteResp == nil {
		s.deleteResp = &transferv1.DeleteItemResponse{Id: req.Msg.Id}
	}
	return connect.NewResponse(s.deleteResp), nil
}

func connectAPI(t *testing.T, svc *transferService) http.Handler {
	t.Helper()
	path, handler := transferconnect.NewTransferServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func item(id, name string, kind transferv1.ItemKind) *transferv1.Item {
	return &transferv1.Item{
		Id:             id,
		OwnerId:        "owner-1",
		OriginDeviceId: "dev-1",
		Kind:           kind,
		Name:           name,
		Mime:           "text/plain",
		SizeBytes:      5,
		Retention:      transferv1.Retention_RETENTION_HELD,
		CreatedAt:      timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
	}
}

// tokenFlags returns an ArgSchema/options pair pre-loaded with the device-token
// flag set, so handler tests authenticate without hitting the env fallback.
func tokenOpts(extra map[string]string) cliapptest.TestRunContextOptions {
	flags := map[string]string{"device-token": testDeviceToken}
	for k, v := range extra {
		flags[k] = v
	}
	return cliapptest.TestRunContextOptions{Flags: flags}
}

func TestSendText_SendsTokenAndFields(t *testing.T) {
	svc := &transferService{createResp: &transferv1.CreateTextItemResponse{Item: item("new", "snippet", transferv1.ItemKind_ITEM_KIND_TEXT)}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "text", Required: true}},
		Flags:       []cliapp.Flag{{Name: "device-token"}, {Name: "name"}, {Name: "retention"}, {Name: "target"}},
	}, func() cliapptest.TestRunContextOptions {
		o := tokenOpts(map[string]string{"name": "snippet", "retention": "pinned", "target": "dev-2"})
		o.Positionals = map[string]string{"text": "hello world"}
		return o
	}())

	require.NoError(t, h.sendText(ctx))
	require.Len(t, svc.createInputs, 1)
	require.Equal(t, "hello world", svc.createInputs[0].Text)
	require.Equal(t, "snippet", svc.createInputs[0].Name)
	require.Equal(t, transferv1.Retention_RETENTION_PINNED, svc.createInputs[0].Retention)
	require.Equal(t, "dev-2", svc.createInputs[0].TargetDeviceId)
	require.Equal(t, []string{testDeviceToken}, svc.tokens)
	require.Contains(t, out.String(), "Sent text item new")
}

func TestSendText_RejectsUnknownRetention(t *testing.T) {
	svc := &transferService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "text", Required: true}},
		Flags:       []cliapp.Flag{{Name: "device-token"}, {Name: "name"}, {Name: "retention"}, {Name: "target"}},
	}, func() cliapptest.TestRunContextOptions {
		o := tokenOpts(map[string]string{"retention": "forever"})
		o.Positionals = map[string]string{"text": "hi"}
		return o
	}())

	err := h.sendText(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown retention")
	require.Empty(t, svc.createInputs, "must not call the server on an invalid retention")
}

func TestTransfer_RequiresDeviceToken(t *testing.T) {
	t.Setenv(deviceTokenEnvVar, "")
	svc := &transferService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "device-token"}, {Name: "query"}, {Name: "kind"}},
	}, cliapptest.TestRunContextOptions{})

	err := h.list(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no device token")
	require.Empty(t, svc.listInputs, "must not call the server without a token")
}

func TestList_FiltersAndJSONWireShape(t *testing.T) {
	svc := &transferService{listResp: &transferv1.ListItemsResponse{Items: []*transferv1.Item{
		item("a", "one.txt", transferv1.ItemKind_ITEM_KIND_FILE),
	}}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	opts := tokenOpts(map[string]string{"query": "one", "kind": "file"})
	opts.JSON = true
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "device-token"}, {Name: "query"}, {Name: "kind"}},
	}, opts)

	require.NoError(t, h.list(ctx))
	require.Len(t, svc.listInputs, 1)
	require.Equal(t, "one", svc.listInputs[0].Query)
	require.Equal(t, transferv1.ItemKind_ITEM_KIND_FILE, svc.listInputs[0].Kind)

	body := out.String()
	require.NotContains(t, body, "summary", "--json must be proto wire shape")
	var got transferv1.ListItemsResponse
	require.NoError(t, protojson.Unmarshal(out.Bytes(), &got))
	require.Len(t, got.Items, 1)
	require.Equal(t, "a", got.Items[0].Id)
}

func TestDelete_CallsServer(t *testing.T) {
	svc := &transferService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
		Flags:       []cliapp.Flag{{Name: "device-token"}},
	}, func() cliapptest.TestRunContextOptions {
		o := tokenOpts(nil)
		o.Positionals = map[string]string{"id": "doomed"}
		return o
	}())

	require.NoError(t, h.delete(ctx))
	require.Equal(t, []string{"doomed"}, svc.deleteInputs)
	require.Contains(t, out.String(), "Deleted item doomed")
}

// TestUpload_StreamsMultipartWithToken drives the REST upload byte edge against
// a raw HTTP fake, asserting the X-Device-Token header, the file body, and the
// retention form field all arrive, and the proto-typed Item renders.
func TestUpload_StreamsMultipartWithToken(t *testing.T) {
	var gotToken, gotRetention, gotBody, gotPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("X-Device-Token")
		require.NoError(t, r.ParseMultipartForm(32<<20))
		gotRetention = r.FormValue("retention")
		file, header, err := r.FormFile("file")
		require.NoError(t, err)
		defer file.Close()
		require.Equal(t, "photo.png", header.Filename)
		body, err := io.ReadAll(file)
		require.NoError(t, err)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(cliapptest.MustMarshalProto(t, &transferv1.UploadItemResponse{
			Item: item("uploaded", "photo.png", transferv1.ItemKind_ITEM_KIND_FILE),
		}))
	})
	core := clitest.NewTestApp(t, handler)
	h := newHandlers(core)
	tmp := filepath.Join(t.TempDir(), "photo.png")
	require.NoError(t, os.WriteFile(tmp, []byte("PNGBYTES"), 0o600))
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "file"}, {Name: "device-token"}, {Name: "name"}, {Name: "retention"}, {Name: "target"}},
	}, tokenOpts(map[string]string{"file": tmp, "retention": "pinned"}))

	require.NoError(t, h.upload(ctx))
	require.True(t, hasSuffix(gotPath, "/api/v1/transfer/items"), gotPath)
	require.Equal(t, testDeviceToken, gotToken)
	require.Equal(t, "pinned", gotRetention)
	require.Equal(t, "PNGBYTES", gotBody)
	require.Contains(t, out.String(), "Uploaded file as item uploaded")
}

func TestUpload_RejectsUnknownRetentionBeforeRequest(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	core := clitest.NewTestApp(t, handler)
	h := newHandlers(core)
	tmp := filepath.Join(t.TempDir(), "f.bin")
	require.NoError(t, os.WriteFile(tmp, []byte("x"), 0o600))
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "file"}, {Name: "device-token"}, {Name: "name"}, {Name: "retention"}, {Name: "target"}},
	}, tokenOpts(map[string]string{"file": tmp, "retention": "eternal"}))

	err := h.upload(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown retention")
	require.False(t, called, "must not POST on an invalid retention")
}

// TestDownload_StreamsToOriginalFilename drives the REST download byte edge,
// asserting the X-Device-Token header rides along and the Content-Disposition
// filename is preserved when --out is a directory.
func TestDownload_StreamsToOriginalFilename(t *testing.T) {
	var gotToken, gotPath, gotQuery string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Device-Token")
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Disposition", `attachment; filename="report.pdf"`)
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("PDFDATA"))
	})
	core := clitest.NewTestApp(t, handler)
	h := newHandlers(core)
	dir := t.TempDir()
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
		Flags:       []cliapp.Flag{{Name: "out"}, {Name: "device-token"}, {Name: "thumb", Bool: true}},
	}, func() cliapptest.TestRunContextOptions {
		o := tokenOpts(map[string]string{"out": dir})
		o.Positionals = map[string]string{"id": "item-1"}
		return o
	}())

	require.NoError(t, h.download(ctx))
	require.Equal(t, testDeviceToken, gotToken)
	require.True(t, hasSuffix(gotPath, "/api/v1/transfer/items/item-1/content"), gotPath)
	require.Empty(t, gotQuery, "no thumb flag => no ?thumb=1")

	written, err := os.ReadFile(filepath.Join(dir, "report.pdf"))
	require.NoError(t, err)
	require.Equal(t, "PDFDATA", string(written))
	require.Contains(t, out.String(), "Downloaded item item-1")
}

func TestDownload_ThumbFlagAddsQuery(t *testing.T) {
	var gotQuery string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Disposition", `attachment; filename="thumb.jpg"`)
		_, _ = w.Write([]byte("JPG"))
	})
	core := clitest.NewTestApp(t, handler)
	h := newHandlers(core)
	dir := t.TempDir()
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
		Flags:       []cliapp.Flag{{Name: "out"}, {Name: "device-token"}, {Name: "thumb", Bool: true}},
	}, func() cliapptest.TestRunContextOptions {
		o := tokenOpts(map[string]string{"out": dir, "thumb": "true"})
		o.BoolFlags = map[string]bool{"thumb": true}
		o.Positionals = map[string]string{"id": "item-1"}
		return o
	}())

	require.NoError(t, h.download(ctx))
	require.Equal(t, "thumb=1", gotQuery)
}

func registerForTest(t *testing.T, core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	t.Helper()
	manifest := readTransferManifest(t)
	group, err := Register(core, manifest)
	require.NoError(t, err)
	return group
}

func TestRegister_Wiring(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &transferService{}))
	group := registerForTest(t, core)

	require.Equal(t, "transfer", group.Name)
	require.True(t, group.NeedsAPI)
	names := make([]string, 0, len(group.Subcommands))
	for _, sc := range group.Subcommands {
		names = append(names, sc.Name)
		require.NotNil(t, sc.RunCtx, "subcommand %s should use RunCtx", sc.Name)
	}
	require.ElementsMatch(t, []string{"send-text", "list", "get", "delete", "upload", "download"}, names)
}

// hasSuffix is a tiny local helper to keep the assertions readable without
// importing strings just for one call site per test.
func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
