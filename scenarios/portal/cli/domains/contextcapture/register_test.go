package contextcapture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	wire "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/contextcapture"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/contextcapture/contextcapture_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fixtureServer struct {
	rpc.UnimplementedContextCaptureServiceHandler
	doc    *wire.Document
	pixels []byte
	calls  atomic.Int32
}

func (f *fixtureServer) Import(_ context.Context, r *connect.Request[wire.ImportRequest]) (*connect.Response[wire.Document], error) {
	f.calls.Add(1)
	return connect.NewResponse(f.doc), nil
}

func (f *fixtureServer) Read(_ context.Context, r *connect.Request[wire.ReferenceRequest]) (*connect.Response[wire.ReadResponse], error) {
	f.calls.Add(1)
	return connect.NewResponse(&wire.ReadResponse{Document: f.doc, Png: f.pixels}), nil
}

func (f *fixtureServer) Delete(_ context.Context, r *connect.Request[wire.ReferenceRequest]) (*connect.Response[wire.DeleteResponse], error) {
	f.calls.Add(1)
	return connect.NewResponse(&wire.DeleteResponse{}), nil
}

func TestContextCommandsUsePrivateFilesAndNeverPrintPixels(t *testing.T) {
	dir := t.TempDir()
	token := filepath.Join(dir, "token")
	request := filepath.Join(dir, "request.json")
	output := filepath.Join(dir, "image.png")
	require.NoError(t, os.WriteFile(token, []byte("explicit-token\n"), 0o600))
	require.NoError(t, os.WriteFile(request, []byte(`{"retentionSeconds":60}`), 0o600))
	pixels := []byte("bounded fixture bytes")
	digest := sha256.Sum256(pixels)
	fixture := &fixtureServer{doc: &wire.Document{Id: "artifact-id", OriginalSha256: hex.EncodeToString(digest[:]), ExpiresAt: timestamppb.New(time.Now().Add(time.Minute))}, pixels: pixels}
	_, handler := rpc.NewContextCaptureServiceHandler(fixture)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer explicit-token", r.Header.Get("Authorization"))
		handler.ServeHTTP(w, r)
	}))
	defer server.Close()
	client := rpc.NewContextCaptureServiceClient(server.Client(), server.URL)
	ctx := context.Background()
	imported, err := call(ctx, client, "Import", request, "", "", token)
	require.NoError(t, err)
	require.IsType(t, &wire.Document{}, imported)
	read, err := call(ctx, client, "Read", "", "artifact-id", output, token)
	require.NoError(t, err)
	require.IsType(t, &wire.Document{}, read)
	saved, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, pixels, saved)
	info, err := os.Stat(output)
	require.NoError(t, err)
	require.EqualValues(t, 0o600, info.Mode().Perm())
	_, err = call(ctx, client, "Read", "", "artifact-id", output, token)
	require.Error(t, err)
	saved, err = os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, pixels, saved)
	_, err = call(ctx, client, "Delete", "", "artifact-id", "", token)
	require.NoError(t, err)
	status, err := call(ctx, client, "ReconcileImport", "", "request-id", "", token)
	require.NoError(t, err)
	require.Equal(t, "request-id", status.(*wire.ReconcileImportResponse).RequestId)
	require.Equal(t, fixture.doc.Id, status.(*wire.ReconcileImportResponse).Document.Id)
	_, err = call(ctx, client, "CancelImport", "", "request-id", "", token)
	require.NoError(t, err)

	renderedPath := filepath.Join(dir, "rendered.png")
	rendered, err := call(ctx, client, "Render", "", "artifact-id", renderedPath, token)
	require.NoError(t, err)
	require.Empty(t, rendered.(*wire.RenderResponse).Png)
	renderedBytes, err := os.ReadFile(renderedPath)
	require.NoError(t, err)
	require.Equal(t, []byte("rendered selected crop"), renderedBytes)
	require.Equal(t, fixture.doc.OriginalSha256, rendered.(*wire.RenderResponse).Document.OriginalSha256)
	calls := fixture.calls.Load()
	require.NoError(t, os.Chmod(token, 0o644))
	_, err = call(ctx, client, "Import", request, "", "", token)
	require.Error(t, err)
	require.Equal(t, calls, fixture.calls.Load())
	require.NoError(t, os.Chmod(token, 0o600))
	link := filepath.Join(dir, "token-link")
	require.NoError(t, os.Symlink(token, link))
	_, err = call(ctx, client, "Delete", "", "artifact-id", "", link)
	require.Error(t, err)
	require.Equal(t, calls, fixture.calls.Load())
}

func (f *fixtureServer) ReconcileImport(_ context.Context, r *connect.Request[wire.ReconcileImportRequest]) (*connect.Response[wire.ReconcileImportResponse], error) {
	f.calls.Add(1)
	return connect.NewResponse(&wire.ReconcileImportResponse{RequestId: r.Msg.RequestId, State: wire.ImportState_IMPORT_STATE_READY, Document: f.doc}), nil
}

func (f *fixtureServer) CancelImport(_ context.Context, r *connect.Request[wire.ReconcileImportRequest]) (*connect.Response[wire.DeleteResponse], error) {
	f.calls.Add(1)
	return connect.NewResponse(&wire.DeleteResponse{}), nil
}

func (f *fixtureServer) Render(_ context.Context, r *connect.Request[wire.ReferenceRequest]) (*connect.Response[wire.RenderResponse], error) {
	f.calls.Add(1)
	pixels := []byte("rendered selected crop")
	digest := sha256.Sum256(pixels)
	return connect.NewResponse(&wire.RenderResponse{Document: f.doc, Png: pixels, RenderedSha256: hex.EncodeToString(digest[:])}), nil
}
