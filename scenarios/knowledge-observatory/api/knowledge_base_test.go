package main

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	pkg "github.com/vrooli/ai-go/search"
	kov1 "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1"
	koconnect "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1/knowledgeobservatoryv1connect"
	"knowledge-observatory/internal/dbtest"
	"knowledge-observatory/internal/knowledgebase"
	"knowledge-observatory/internal/modules"
	"knowledge-observatory/internal/services/docsearch"
)

// [REQ:KO-KB-001] [REQ:KO-KB-003]
func TestKnowledgeBaseConnectReadContract(t *testing.T) {
	for _, key := range []string{
		"DATABASE_URL", "POSTGRES_URL", "POSTGRES_USER", "POSTGRES_HOST",
		"POSTGRES_PORT", "POSTGRES_PASSWORD", "POSTGRES_DB",
	} {
		t.Setenv(key, "")
	}
	root := t.TempDir()
	for _, p := range []string{"scenarios", "docs", filepath.Join("docs", "reference")} {
		if err := os.MkdirAll(filepath.Join(root, p), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Use current maintained sources as fixtures so the API proves the same
	// storage and route behavior claims documented for operators are retrievable.
	for _, source := range []struct {
		from string
		to   string
	}{
		{from: filepath.Join("..", "docs", "concepts", "DATA.md"), to: filepath.Join(root, "docs", "DATA.md")},
		{from: filepath.Join("..", "docs", "concepts", "ARCHITECTURE.md"), to: filepath.Join(root, "docs", "ARCHITECTURE.md")},
		{from: filepath.Join("..", "docs", "reference", "api-endpoints.md"), to: filepath.Join(root, "docs", "reference", "api-endpoints.md")},
	} {
		fixtureContent, err := os.ReadFile(source.from)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(source.to, fixtureContent, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	search, err := docsearch.NewService(filepath.Join(root, "scenarios"))
	if err != nil {
		t.Fatal(err)
	}
	// The source-search API must still resolve an exact case when vector
	// services are absent; its fallback uses the same fixture and live SQLite
	// storage contract as the scenario runtime.
	db := dbtest.New(t, modules.AllSchemas()...)
	serverState := &Server{db: db, stores: modules.NewSQLiteStores(db), docSearchService: search}
	handler := &knowledgeBaseHandler{server: serverState, sources: &knowledgebase.Service{RepoRoot: root}}
	_, route := koconnect.NewKnowledgeBaseServiceHandler(handler)
	server := httptest.NewServer(route)
	defer server.Close()
	client := koconnect.NewKnowledgeBaseServiceClient(server.Client(), server.URL)
	found, err := client.SearchDocuments(context.Background(), connect.NewRequest(&kov1.SearchDocumentsRequest{Query: "scenario-scoped SQLite database", Scope: "path", Target: "docs", Mode: "auto"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(found.Msg.Results) != 1 || found.Msg.Results[0].Path != "docs/DATA.md" || found.Msg.Method != "text" {
		t.Fatalf("bad fallback: %v", found.Msg)
	}
	if !strings.Contains(found.Msg.Results[0].Snippet, "scenario-scoped SQLite database") {
		t.Fatalf("retrieval did not return the fixture fact: %q", found.Msg.Results[0].Snippet)
	}
	viewer, err := client.SearchDocuments(context.Background(), connect.NewRequest(&kov1.SearchDocumentsRequest{Query: "legacy", Scope: "path", Target: "docs", Mode: "auto"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(viewer.Msg.Results) == 0 || viewer.Msg.Results[0].Path != "docs/ARCHITECTURE.md" || viewer.Msg.Method != "text" {
		t.Fatalf("bad Viewer architecture fallback: %v", viewer.Msg)
	}
	if !strings.Contains(viewer.Msg.Results[0].Snippet, "legacy `#/viewer` URL redirects to Explorer") {
		t.Fatalf("retrieval did not return the maintained route contract: %q", viewer.Msg.Results[0].Snippet)
	}
	apiContract, err := client.SearchDocuments(context.Background(), connect.NewRequest(&kov1.SearchDocumentsRequest{Query: "manifest-supported append log", Scope: "path", Target: "docs", Mode: "auto"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(apiContract.Msg.Results) == 0 || apiContract.Msg.Results[0].Path != "docs/reference/api-endpoints.md" || apiContract.Msg.Method != "text" {
		t.Fatalf("bad append API contract fallback: %v", apiContract.Msg)
	}
	if !strings.Contains(apiContract.Msg.Results[0].Snippet, "manifest-supported append log") {
		t.Fatalf("retrieval did not return the append contract: %q", apiContract.Msg.Results[0].Snippet)
	}
	doc, err := client.InspectDocument(context.Background(), connect.NewRequest(&kov1.InspectDocumentRequest{Path: found.Msg.Results[0].Path}))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Msg.Sha256) != 64 {
		t.Fatal("missing revision")
	}
	_, err = client.InspectDocument(context.Background(), connect.NewRequest(&kov1.InspectDocumentRequest{Path: "docs/DATA.md", ExpectedSha256: "wrong"}))
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
