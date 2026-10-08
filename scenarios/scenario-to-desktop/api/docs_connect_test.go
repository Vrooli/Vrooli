package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

func TestDocumentationConnectServiceServesManifestAndContent(t *testing.T) {
	docsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(docsDir, "manifest.json"), []byte(`{"version":"1","title":"Desktop Docs","description":"Helpful docs","defaultDocument":"guide.md","navigation":{"primary":["guide.md"]},"sections":[{"id":"guide","title":"Guide","icon":"book","description":"Start here","documents":[{"path":"guide.md","title":"Guide","description":"The guide"}]}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte("# Guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SCENARIO_TO_DESKTOP_DOCS_DIR", docsDir)
	server := NewServer(0)
	t.Cleanup(func() { shutdownServer(t, server) })
	service := documentationConnectService{server: server}

	manifest, err := service.GetDocumentationManifest(context.Background(), connect.NewRequest(&domainv1.DocumentationManifestRequest{}))
	if err != nil || manifest.Msg.GetTitle() != "Desktop Docs" || manifest.Msg.GetDescription() != "Helpful docs" || len(manifest.Msg.GetSections()) != 1 || manifest.Msg.GetSections()[0].GetDocuments()[0].GetPath() != "guide.md" {
		t.Fatalf("GetDocumentationManifest() = %#v, %v", manifest, err)
	}
	content, err := service.GetDocumentationContent(context.Background(), connect.NewRequest(&domainv1.DocumentationContentRequest{Path: "guide.md"}))
	if err != nil || content.Msg.GetContent() != "# Guide\n" {
		t.Fatalf("GetDocumentationContent() = %#v, %v", content, err)
	}
}

func TestDocumentationConnectServiceRejectsInvalidAndUnavailableContent(t *testing.T) {
	docsDir := t.TempDir()
	t.Setenv("SCENARIO_TO_DESKTOP_DOCS_DIR", docsDir)
	server := NewServer(0)
	t.Cleanup(func() { shutdownServer(t, server) })
	service := documentationConnectService{server: server}

	if _, err := service.GetDocumentationManifest(context.Background(), connect.NewRequest(&domainv1.DocumentationManifestRequest{})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing manifest code = %v", connect.CodeOf(err))
	}
	if _, err := service.GetDocumentationContent(context.Background(), connect.NewRequest(&domainv1.DocumentationContentRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("empty path code = %v", connect.CodeOf(err))
	}
	if _, err := service.GetDocumentationContent(context.Background(), connect.NewRequest(&domainv1.DocumentationContentRequest{Path: "/etc/passwd"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("absolute path code = %v", connect.CodeOf(err))
	}
	if _, err := service.GetDocumentationContent(context.Background(), connect.NewRequest(&domainv1.DocumentationContentRequest{Path: "missing.md"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing content code = %v", connect.CodeOf(err))
	}
	if err := os.WriteFile(filepath.Join(docsDir, "manifest.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetDocumentationManifest(context.Background(), connect.NewRequest(&domainv1.DocumentationManifestRequest{})); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("invalid manifest code = %v", connect.CodeOf(err))
	}
}
