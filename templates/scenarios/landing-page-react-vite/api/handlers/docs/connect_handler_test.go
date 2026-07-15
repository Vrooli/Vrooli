package docs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	docsH "landing-page-react-vite-api/handlers/docs"
	internaldocs "landing-page-react-vite-api/internal/docs"
)

func newHandler(t *testing.T) (*docsH.Deps, string) {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "intro.md"), []byte("# Intro\n\nHello."), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "guides"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "guides", "setup.md"), []byte("Setup body without heading."), 0o644))
	// An empty dir and a non-markdown file must not appear in the tree.
	require.NoError(t, os.MkdirAll(filepath.Join(root, "empty"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "ignore.txt"), []byte("nope"), 0o644))
	return &docsH.Deps{Service: internaldocs.NewService(root)}, root
}

func TestGetDocsTree(t *testing.T) {
	deps, _ := newHandler(t)
	h := docsH.NewConnectHandler(*deps)
	resp, err := h.GetDocsTree(context.Background(), connect.NewRequest(&landingv1.GetDocsTreeRequest{}))
	require.NoError(t, err)
	// Directory (guides) sorts before file (intro.md); empty/ and ignore.txt excluded.
	require.Len(t, resp.Msg.Entries, 2)
	require.Equal(t, "guides", resp.Msg.Entries[0].Name)
	require.True(t, resp.Msg.Entries[0].IsDir)
	require.Len(t, resp.Msg.Entries[0].Children, 1)
	require.Equal(t, "intro.md", resp.Msg.Entries[1].Name)
}

func TestGetDocContentExtractsTitle(t *testing.T) {
	deps, _ := newHandler(t)
	h := docsH.NewConnectHandler(*deps)
	resp, err := h.GetDocContent(context.Background(), connect.NewRequest(&landingv1.GetDocContentRequest{Path: "intro.md"}))
	require.NoError(t, err)
	require.Equal(t, "Intro", resp.Msg.Title)
	require.Contains(t, resp.Msg.Content, "Hello.")

	// Falls back to filename when there is no H1.
	resp, err = h.GetDocContent(context.Background(), connect.NewRequest(&landingv1.GetDocContentRequest{Path: "guides/setup.md"}))
	require.NoError(t, err)
	require.Equal(t, "setup", resp.Msg.Title)
}

func TestGetDocContentRejectsTraversalAndNonMarkdown(t *testing.T) {
	deps, _ := newHandler(t)
	h := docsH.NewConnectHandler(*deps)
	_, err := h.GetDocContent(context.Background(), connect.NewRequest(&landingv1.GetDocContentRequest{Path: "../secret.md"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = h.GetDocContent(context.Background(), connect.NewRequest(&landingv1.GetDocContentRequest{Path: "ignore.txt"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = h.GetDocContent(context.Background(), connect.NewRequest(&landingv1.GetDocContentRequest{Path: "missing.md"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
