package retrieval

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"code-facts/internal/catalog"
	_ "modernc.org/sqlite"
)

type testClock struct{ now time.Time }

func (clock testClock) Now() time.Time { return clock.now }

func openRetrievalIndex(t *testing.T, documents []Document) (*sql.DB, *SQLiteIndex) {
	t.Helper()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:retrieval-%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repository := catalog.NewSQLiteRepository(db, testClock{now: time.Unix(100, 0)})
	if err := repository.BeginGeneration(context.Background(), catalog.Generation{ID: "g1", Policy: "test"}); err != nil {
		t.Fatal(err)
	}
	files := make([]catalog.SourceFile, 0, len(documents))
	seen := map[string]struct{}{}
	for _, document := range documents {
		if _, exists := seen[document.SourceFileID]; exists {
			continue
		}
		seen[document.SourceFileID] = struct{}{}
		files = append(files, catalog.SourceFile{
			ID: document.SourceFileID, Path: document.Path, Language: document.Language,
			Role: catalog.Role(document.Role), Scope: document.Scope, Authority: document.Authority,
			Owner: "test", Hash: document.SourceHash, Size: 100, ModTime: time.Unix(50, 0), Searchable: true,
		})
	}
	if err := repository.UpsertFiles(context.Background(), "g1", files); err != nil {
		t.Fatal(err)
	}
	index := NewSQLiteIndex(db)
	if err := index.Upsert(context.Background(), "g1", documents); err != nil {
		t.Fatal(err)
	}
	if err := repository.CompleteGeneration(context.Background(), "g1", "source-digest", "descriptor-digest"); err != nil {
		t.Fatal(err)
	}
	if err := repository.Activate(context.Background(), "g1"); err != nil {
		t.Fatal(err)
	}
	return db, index
}

func TestSQLiteIndexExactBM25FiltersAndExplanations(t *testing.T) {
	documents := []Document{
		{ID: "symbol:http-server", SourceFileID: "file:server", SourceHash: "sha256:server", Path: "scenarios/demo/api/server.go", Language: "go", Role: "implementation", Scope: "scenario:demo", Authority: "authoritative", Kind: "symbol", Title: "HTTPServer", ExactText: "demo.HTTPServer", Body: "Starts the secure web transport and request router.", Aliases: []string{"http server"}, StartLine: 10, EndLine: 20},
		{ID: "contract:search", SourceFileID: "file:proto", SourceHash: "sha256:proto", Path: "packages/proto/schemas/demo.proto", Language: "protobuf", Role: "contract", Scope: "package:proto", Authority: "authoritative", Kind: "contract", Title: "Search", ExactText: "demo.SearchService.Search", Body: "Searches indexed implementation evidence.", ContractText: "rpc Search(SearchRequest) returns (SearchResponse)", StartLine: 30, EndLine: 31},
		{ID: "test:http-server", SourceFileID: "file:test", SourceHash: "sha256:test", Path: "scenarios/demo/api/server_test.go", Language: "go", Role: "test", Scope: "scenario:demo", Authority: "supporting", Kind: "symbol", Title: "TestHTTPServer", ExactText: "demo.TestHTTPServer", Body: "Tests the web transport.", StartLine: 8, EndLine: 12},
	}
	_, index := openRetrievalIndex(t, documents)

	exact, err := index.SearchLexical(context.Background(), Query{Text: "demo.HTTPServer", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(exact) == 0 || exact[0].ID != "symbol:http-server" || exact[0].Regime != RegimeExact || exact[0].Generation != "g1" || exact[0].SourceHash == "" {
		t.Fatalf("exact result lost identity or freshness metadata: %+v", exact)
	}
	if exact[0].Explanation == "" || exact[0].ScoreFactors["exact"] == 0 || exact[0].Evidence != "current_source_hash" || exact[0].Proof != "source_hash_verified" {
		t.Fatalf("ranking and proof semantics are not separated: %+v", exact[0])
	}

	natural, err := index.SearchLexical(context.Background(), Query{Text: "secure request router role:implementation lang:go scope:scenario:demo", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(natural) != 1 || natural[0].ID != "symbol:http-server" || natural[0].Regime != RegimeNatural {
		t.Fatalf("scoped BM25 query mismatch: %+v", natural)
	}
	contract, err := index.SearchLexical(context.Background(), Query{Text: "indexed evidence family:contract", Limit: 5})
	if err != nil || len(contract) != 1 || contract[0].ID != "contract:search" || contract[0].Regime != RegimeContract {
		t.Fatalf("contract query mismatch: %+v err=%v", contract, err)
	}
}

func TestSQLiteIndexFreshnessFenceAndDeleteTrigger(t *testing.T) {
	document := Document{ID: "symbol:gone", SourceFileID: "file:gone", SourceHash: "sha256:old", Path: "gone.go", Language: "go", Role: "implementation", Scope: "repo", Authority: "authoritative", Kind: "symbol", Title: "GoneSymbol", ExactText: "GoneSymbol", Body: "unique searchable payload", StartLine: 1, EndLine: 2}
	db, index := openRetrievalIndex(t, []Document{document})
	if _, err := db.Exec(`UPDATE code_facts_source_files SET content_hash='sha256:new' WHERE id='file:gone'`); err != nil {
		t.Fatal(err)
	}
	results, err := index.SearchLexical(context.Background(), Query{Text: "unique searchable", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("stale source hash crossed freshness fence: %+v", results)
	}
	if _, err := db.Exec(`UPDATE code_facts_source_files SET content_hash='sha256:old' WHERE id='file:gone'`); err != nil {
		t.Fatal(err)
	}
	if err := index.Delete(context.Background(), "g1", []string{"symbol:gone"}); err != nil {
		t.Fatal(err)
	}
	results, err = index.SearchLexical(context.Background(), Query{Text: "unique searchable", Limit: 5})
	if err != nil || len(results) != 0 {
		t.Fatalf("deleted FTS record remained searchable: %+v err=%v", results, err)
	}
}

func TestNormalizeQueryAndIdentifierTokenization(t *testing.T) {
	query, err := NormalizeQuery(Query{Text: "HTTPServer path:scenarios/demo role:implementation language:go family:symbol limit:7"})
	if err != nil {
		t.Fatal(err)
	}
	if query.Text != "HTTPServer" || query.Target != "scenarios/demo" || query.Limit != 7 || strings.Join(query.Roles, ",") != "implementation" || strings.Join(query.Languages, ",") != "go" || strings.Join(query.Families, ",") != "symbol" {
		t.Fatalf("query parsed incorrectly: %+v", query)
	}
	if got := strings.Join(splitIdentifier("HTTPServer_Search-v2.tsx"), " "); got != "http server search v2 tsx" {
		t.Fatalf("mixed identifier tokenization mismatch: %q", got)
	}
	if _, err := NormalizeQuery(Query{Text: "thing limit:1000"}); err == nil {
		t.Fatal("unbounded explicit limit must fail")
	}
}

func TestSQLiteIndexUsesFTSVirtualTablePlan(t *testing.T) {
	_, index := openRetrievalIndex(t, []Document{{ID: "one", SourceFileID: "file:one", SourceHash: "sha256:one", Path: "one.go", Language: "go", Role: "implementation", Scope: "repo", Authority: "authoritative", Kind: "symbol", Title: "One", Body: "needle text"}})
	rows, err := index.db.Query(`EXPLAIN QUERY PLAN SELECT rowid FROM code_facts_search_fts WHERE code_facts_search_fts MATCH 'needle'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		plan.WriteString(detail)
	}
	if !strings.Contains(strings.ToUpper(plan.String()), "VIRTUAL TABLE") {
		t.Fatalf("query plan did not use FTS virtual table: %s", plan.String())
	}
}
