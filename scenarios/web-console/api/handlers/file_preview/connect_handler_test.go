package file_preview

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	filepreviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/file_preview"
)

type fakePreviewService struct{ err error }

func (f fakePreviewService) Resolve(context.Context, ResolveInput) (ResolveResult, error) {
	return ResolveResult{PreviewID: "p1", InputPath: "src/a.go:4", ResolvedPath: "/tmp/a.go", Basename: "a.go", Line: 4, HasLine: true, ResolutionBasis: "cwd", Kind: "code", MIMEType: "text/plain", SizeBytes: 8, CanPreview: true, CanDownload: true, SupportsRange: true, TextContentAvailable: true, BlobURL: "/blob", Warnings: []string{"warning"}}, f.err
}
func (f fakePreviewService) GetTextContent(context.Context, string, string) (TextResult, error) {
	return TextResult{ResolvedPath: "/tmp/a.go", Kind: "markdown", MIMEType: "text/markdown", Content: "# hi", Truncated: true, Line: 2, HasLine: true}, f.err
}
func (f fakePreviewService) ListDirectory(context.Context, ListInput) (ListResult, error) {
	return ListResult{ResolvedPath: "/tmp", ParentPath: "/", Entries: []ListEntry{{Name: "a.go", EntryType: "file", Kind: "code", CanPreview: true}}, TotalEntries: 2, Truncated: true, NextPageToken: "next", EffectiveSort: "name", Warnings: []string{"warn"}}, f.err
}

func TestConnectHandlerPreviewOperations(t *testing.T) {
	h := NewConnectHandler(Deps{Service: fakePreviewService{}})
	ctx := context.Background()
	res, err := h.Resolve(ctx, connect.NewRequest(&filepreviewv1.ResolveRequest{SessionId: "s1", Path: "src/a.go:4", SourceContext: filepreviewv1.SourceContext_SOURCE_CONTEXT_INLINE_CODE}))
	if err != nil || res.Msg.PreviewId != "p1" || res.Msg.PreviewKind != filepreviewv1.PreviewKind_PREVIEW_KIND_CODE || res.Msg.Line != 4 {
		t.Fatalf("resolve: %#v %v", res, err)
	}
	text, err := h.GetTextContent(ctx, connect.NewRequest(&filepreviewv1.GetTextContentRequest{SessionId: "s1", PreviewId: "p1"}))
	if err != nil || text.Msg.Content != "# hi" || text.Msg.PreviewKind != filepreviewv1.PreviewKind_PREVIEW_KIND_MARKDOWN {
		t.Fatalf("text: %#v %v", text, err)
	}
	list, err := h.ListDirectory(ctx, connect.NewRequest(&filepreviewv1.ListDirectoryRequest{SessionId: "s1", PreviewId: "p1", Sort: filepreviewv1.DirectorySort_DIRECTORY_SORT_NAME, ShowHidden: true, PageSize: 20, PageToken: "page"}))
	if err != nil || list.Msg.TotalEntries != 2 || len(list.Msg.Entries) != 1 || list.Msg.EffectiveSort != filepreviewv1.DirectorySort_DIRECTORY_SORT_NAME {
		t.Fatalf("list: %#v %v", list, err)
	}
}

func TestConnectHandlerPreviewValidationAndErrorMapping(t *testing.T) {
	h := NewConnectHandler(Deps{Service: fakePreviewService{}})
	ctx := context.Background()
	for _, call := range []func() error{
		func() error {
			_, e := h.Resolve(ctx, connect.NewRequest(&filepreviewv1.ResolveRequest{Path: "x"}))
			return e
		},
		func() error {
			_, e := h.Resolve(ctx, connect.NewRequest(&filepreviewv1.ResolveRequest{SessionId: "s"}))
			return e
		},
		func() error {
			_, e := h.GetTextContent(ctx, connect.NewRequest(&filepreviewv1.GetTextContentRequest{SessionId: "s"}))
			return e
		},
		func() error {
			_, e := h.ListDirectory(ctx, connect.NewRequest(&filepreviewv1.ListDirectoryRequest{SessionId: "s"}))
			return e
		},
	} {
		if err := call(); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("got %v", err)
		}
	}
	for _, tc := range []struct {
		in   error
		want connect.Code
	}{{ErrSessionNotFound, connect.CodeNotFound}, {ErrNotFound, connect.CodeNotFound}, {ErrInvalidArgument, connect.CodeInvalidArgument}, {ErrPermissionDenied, connect.CodePermissionDenied}, {ErrPreviewUnavailable, connect.CodeFailedPrecondition}, {ErrStale, connect.CodeAborted}, {errors.New("x"), connect.CodeInternal}} {
		var ce *connect.Error
		if err := h.classify(tc.in, "test"); !errors.As(err, &ce) || ce.Code() != tc.want {
			t.Errorf("%v: got %v want %v", tc.in, err, tc.want)
		}
	}
	for _, kind := range []string{"markdown", "code", "text", "svg", "image", "pdf", "audio", "video", "csv", "diff", "directory", "unknown"} {
		_ = kindToProto(kind)
	}
	for _, typ := range []string{"file", "directory", "symlink", "other", "unknown"} {
		_ = entryTypeToProto(typ)
	}
	for _, sort := range []string{"dirs_first_name", "name", "size_desc", "mtime_desc", "unknown"} {
		_ = sortToProto(sort)
	}
	for _, sc := range []filepreviewv1.SourceContext{filepreviewv1.SourceContext_SOURCE_CONTEXT_MESSAGE_LINK, filepreviewv1.SourceContext_SOURCE_CONTEXT_INLINE_CODE, filepreviewv1.SourceContext_SOURCE_CONTEXT_CLI, filepreviewv1.SourceContext_SOURCE_CONTEXT_UNSPECIFIED} {
		_ = sourceContextToString(sc)
	}
}
