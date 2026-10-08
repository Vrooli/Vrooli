package docs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
)

func TestConnectHandlerReadsTreeAndContent(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "guides/intro.md", "# Intro\nBody")
	handler := NewConnectHandler(ConnectDependencies{DocsRoot: func() string { return root }})

	tree, err := handler.GetDocsTree(context.Background(), connect.NewRequest(&lpbsv1.GetDocsTreeRequest{}))
	if err != nil || len(tree.Msg.GetEntries()) != 1 || tree.Msg.GetEntries()[0].GetName() != "guides" {
		t.Fatalf("tree = %#v, err = %v", tree, err)
	}
	content, err := handler.GetDocContent(context.Background(), connect.NewRequest(&lpbsv1.GetDocContentRequest{Path: "guides/intro.md"}))
	if err != nil || content.Msg.GetTitle() != "Intro" || content.Msg.GetContent() != "# Intro\nBody" {
		t.Fatalf("content = %#v, err = %v", content, err)
	}
}

func TestConnectHandlerRejectsUnsafeDocumentPath(t *testing.T) {
	handler := NewConnectHandler(ConnectDependencies{DocsRoot: t.TempDir})
	_, err := handler.GetDocContent(context.Background(), connect.NewRequest(&lpbsv1.GetDocContentRequest{Path: "../secret.md"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid argument", connect.CodeOf(err))
	}
}

func writeDoc(t *testing.T, root, path, body string) {
	t.Helper()
	full := root + "/" + path
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
