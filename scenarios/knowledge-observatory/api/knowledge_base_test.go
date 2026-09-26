package main

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	pkg "github.com/vrooli/ai-go/search"
	kov1 "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1"
	koconnect "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1/knowledgeobservatoryv1connect"
	"knowledge-observatory/internal/knowledgebase"
	"knowledge-observatory/internal/services/docsearch"
)

// [REQ:KO-KB-001] [REQ:KO-KB-003]
func TestKnowledgeBaseConnectReadContract(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"scenarios", "docs"} {
		if err := os.MkdirAll(filepath.Join(root, p), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "rule.md"), []byte("# Rule\nexact fallback needle\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	search, err := docsearch.NewService(filepath.Join(root, "scenarios"))
	if err != nil {
		t.Fatal(err)
	}
	handler := &knowledgeBaseHandler{server: &Server{docSearchService: search}, sources: &knowledgebase.Service{RepoRoot: root}}
	_, route := koconnect.NewKnowledgeBaseServiceHandler(handler)
	server := httptest.NewServer(route)
	defer server.Close()
	client := koconnect.NewKnowledgeBaseServiceClient(server.Client(), server.URL)
	found, err := client.SearchDocuments(context.Background(), connect.NewRequest(&kov1.SearchDocumentsRequest{Query: "exact fallback needle", Scope: "path", Target: "docs", Mode: "text"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(found.Msg.Results) != 1 || found.Msg.Results[0].Path != "docs/rule.md" || found.Msg.Method != "text" {
		t.Fatalf("bad fallback: %v", found.Msg)
	}
	doc, err := client.InspectDocument(context.Background(), connect.NewRequest(&kov1.InspectDocumentRequest{Path: found.Msg.Results[0].Path}))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Msg.Sha256) != 64 {
		t.Fatal("missing revision")
	}
	_, err = client.InspectDocument(context.Background(), connect.NewRequest(&kov1.InspectDocumentRequest{Path: "docs/rule.md", ExpectedSha256: "wrong"}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("wrong conflict code: %v", err)
	}
	_, err = client.SearchDocuments(context.Background(), connect.NewRequest(&kov1.SearchDocumentsRequest{Query: "needle", Scope: "invented"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid scope accepted: %v", err)
	}
	_, err = client.SearchDocuments(context.Background(), connect.NewRequest(&kov1.SearchDocumentsRequest{Query: "needle", Mode: "hybrid"}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("unavailable hybrid mislabeled: %v", err)
	}
}

// [REQ:KO-KB-003] Authority updates take effect before index reconciliation.
func TestCurrentManifestOverridesStaleIndexedAuthority(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "manifest.json"), []byte(`{"sections":[{"documents":[{"path":"rule.md","knowledgeStatus":"superseded","supersededBy":["docs/current.md"]}]}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	service, err := docsearch.NewService(filepath.Join(root, "scenarios"))
	if err != nil {
		t.Fatal(err)
	}
	original := map[string]any{"knowledge_status": "accepted", "file_sha256": "indexed-revision"}
	hits := (&Server{docSearchService: service}).currentDocHits([]pkg.SearchResult{{RelativePath: "docs/rule.md", Path: "docs/rule.md", Payload: original}})
	if hits[0].Metadata["knowledge_status"] != "superseded" || hits[0].Metadata["file_sha256"] != "indexed-revision" {
		t.Fatalf("stale authority or invented source revision: %+v", hits)
	}
	if original["knowledge_status"] != "accepted" {
		t.Fatal("mutated shared search result")
	}
}
