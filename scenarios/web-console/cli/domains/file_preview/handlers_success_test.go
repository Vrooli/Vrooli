package file_preview

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	filepreviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/file_preview"
	filepreviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/file_preview/file_preview_v1connect"
)

type filePreviewTestClient struct {
	filepreviewconnect.FilePreviewServiceClient
}

func (filePreviewTestClient) Resolve(context.Context, *connect.Request[filepreviewv1.ResolveRequest]) (*connect.Response[filepreviewv1.ResolveResponse], error) {
	return connect.NewResponse(&filepreviewv1.ResolveResponse{}), nil
}
func (filePreviewTestClient) GetTextContent(context.Context, *connect.Request[filepreviewv1.GetTextContentRequest]) (*connect.Response[filepreviewv1.GetTextContentResponse], error) {
	return connect.NewResponse(&filepreviewv1.GetTextContentResponse{}), nil
}
func (filePreviewTestClient) ListDirectory(context.Context, *connect.Request[filepreviewv1.ListDirectoryRequest]) (*connect.Response[filepreviewv1.ListDirectoryResponse], error) {
	return connect.NewResponse(&filepreviewv1.ListDirectoryResponse{}), nil
}

func TestHandlersRenderSuccessfulResponses(t *testing.T) {
	h := &handlers{client: filePreviewTestClient{}}
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "session"}, {Name: "path"}, {Name: "preview-id"}, {Name: "sort"}, {Name: "page-size"}, {Name: "page-token"}, {Name: "show-hidden", Bool: true}}}
	ctx := func(flags map[string]string) cliapp.RunContext {
		return cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: schema, Flags: flags, JSON: true})
	}
	if err := h.resolve(ctx(map[string]string{"session": "s1", "path": "/tmp"})); err != nil {
		t.Fatal(err)
	}
	if err := h.text(ctx(map[string]string{"session": "s1", "preview-id": "p1"})); err != nil {
		t.Fatal(err)
	}
	if err := h.list(ctx(map[string]string{"session": "s1", "preview-id": "p1", "sort": "name", "page-size": "10"})); err != nil {
		t.Fatal(err)
	}
}
