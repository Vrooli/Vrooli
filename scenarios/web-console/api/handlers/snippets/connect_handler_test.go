package snippets

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	snippetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/snippets"

	snippetdomain "web-console/internal/snippets"
)

func newHandler(t *testing.T) *connectHandler {
	t.Helper()
	return NewConnectHandler(Deps{
		Service: snippetdomain.NewMemStore(),
		Now:     func() time.Time { return time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC) },
	})
}

func TestSnippetRPCsRoundTrip(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()
	created, err := h.UpsertSnippet(ctx, connect.NewRequest(&snippetsv1.UpsertSnippetRequest{
		Name: "Check {{scenario}}", Body: "Inspect {{scenario}} and {{evidence}}.",
		Color: "#22d3ee", Pinned: true, HasPinned: true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	id := created.Msg.GetSnippet().GetId()
	if id == "" || !created.Msg.GetSnippet().GetPinned() {
		t.Fatalf("created snippet = %#v", created.Msg.GetSnippet())
	}

	touched, err := h.TouchSnippet(ctx, connect.NewRequest(&snippetsv1.TouchSnippetRequest{Id: id}))
	if err != nil {
		t.Fatal(err)
	}
	if touched.Msg.GetSnippet().GetUseCount() != 1 || touched.Msg.GetSnippet().GetLastUsedAt() == "" {
		t.Fatalf("touched snippet = %#v", touched.Msg.GetSnippet())
	}

	listed, err := h.ListSnippets(ctx, connect.NewRequest(&snippetsv1.ListSnippetsRequest{}))
	if err != nil || len(listed.Msg.GetSnippets()) != 1 {
		t.Fatalf("list = %#v, %v", listed, err)
	}

	deleted, err := h.DeleteSnippet(ctx, connect.NewRequest(&snippetsv1.DeleteSnippetRequest{Id: id}))
	if err != nil || !deleted.Msg.GetDeleted() {
		t.Fatalf("delete = %#v, %v", deleted, err)
	}
	deleted, err = h.DeleteSnippet(ctx, connect.NewRequest(&snippetsv1.DeleteSnippetRequest{Id: id}))
	if err != nil || deleted.Msg.GetDeleted() {
		t.Fatalf("second delete = %#v, %v", deleted, err)
	}
}

func TestSnippetRPCErrorMappings(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()
	_, err := h.UpsertSnippet(ctx, connect.NewRequest(&snippetsv1.UpsertSnippetRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("blank name code = %v, want invalid_argument", connect.CodeOf(err))
	}
	_, err = h.TouchSnippet(ctx, connect.NewRequest(&snippetsv1.TouchSnippetRequest{Id: "missing"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing touch code = %v, want not_found", connect.CodeOf(err))
	}
}
