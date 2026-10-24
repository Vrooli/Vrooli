package analysis

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"code-facts/internal/catalog"
	_ "modernc.org/sqlite"
)

type graphClock struct{}

func (graphClock) Now() time.Time { return time.Unix(100, 0) }

func TestSQLiteProjectionStoreReturnsOnlyCurrentAnalyzerEvidence(t *testing.T) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:graph-%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repository := catalog.NewSQLiteRepository(db, graphClock{})
	if err := repository.BeginGeneration(context.Background(), catalog.Generation{ID: "g1", Policy: "test"}); err != nil {
		t.Fatal(err)
	}
	file := catalog.SourceFile{ID: "file:service", Path: "service.go", Language: "go", Role: catalog.RoleImplementation, Scope: "repo", Authority: "authoritative", Owner: "test", Hash: "sha256:current", Size: 10, Searchable: true}
	if err := repository.UpsertFiles(context.Background(), "g1", []catalog.SourceFile{file}); err != nil {
		t.Fatal(err)
	}
	store := NewSQLiteProjectionStore(db)
	facts := []Fact{{ID: "edge:caller", Family: "callers", Kind: "call", Subject: "Service.Search", Predicate: "called_by", Object: "Handler.Search", Path: "service.go", SourceHash: "sha256:current", Proof: "analyzer_confirmed", Analyzer: "go-code-graph", Version: "v2"}}
	if err := store.Replace(context.Background(), "g1", file.ID, facts); err != nil {
		t.Fatal(err)
	}
	if err := repository.CompleteGeneration(context.Background(), "g1", "source", "descriptor"); err != nil {
		t.Fatal(err)
	}
	if err := repository.Activate(context.Background(), "g1"); err != nil {
		t.Fatal(err)
	}
	got, err := store.Expand(context.Background(), "g1", "Service.Search", []string{"callers"}, 5)
	if err != nil || len(got) != 1 || got[0].Analyzer != "go-code-graph" || got[0].Version != "v2" || got[0].Proof != "analyzer_confirmed" {
		t.Fatalf("graph evidence mismatch: %+v err=%v", got, err)
	}
	if _, err := db.Exec(`UPDATE code_facts_source_files SET content_hash='sha256:changed' WHERE id='file:service'`); err != nil {
		t.Fatal(err)
	}
	got, err = store.Expand(context.Background(), "g1", "Service.Search", []string{"callers"}, 5)
	if err != nil || len(got) != 0 {
		t.Fatalf("stale graph evidence crossed freshness fence: %+v err=%v", got, err)
	}
}
