package facts

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"code-facts/internal/catalog"
	internalfacts "code-facts/internal/facts"
	"code-facts/internal/indexcontrol"
	"connectrpc.com/connect"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	_ "modernc.org/sqlite"
)

func TestConnectSearchExposesStableImportProvenanceAndRejectsBroadScope(t *testing.T) {
	index := newImportEndpointFixture(t, indexPolicy, map[string]string{
		"scenarios/demo/api/server.go":  "package api\nimport (\n\t\"context\"\n)\nfunc Serve(context.Context) {}\n",
		"scenarios/other/api/server.go": "package api\nimport \"context\"\n",
	})
	handler := NewConnectHandler(Deps{Service: internalfacts.NewService(internalfacts.WithSearchBackend(index))})
	request := &factsv1.SearchRequest{
		Query: "context", Limit: 5, Families: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
		Target: &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "demo"},
	}
	response, err := handler.Search(context.Background(), connect.NewRequest(request))
	if err != nil || response == nil || len(response.Msg.GetResults()) != 1 {
		t.Fatalf("connect import search response=%+v err=%v", response, err)
	}
	hit := response.Msg.GetResults()[0]
	if hit.GetPath() != "scenarios/demo/api/server.go" || hit.GetFactKind() != "imports" || hit.GetSourceHash() == "" || hit.GetGeneration() == "" || hit.GetScope() != "scenario:demo" || hit.GetStartLine() != 3 || hit.GetEndLine() != 3 {
		t.Fatalf("connect import provenance = %+v", hit)
	}
	repeated, err := handler.Search(context.Background(), connect.NewRequest(request))
	if err != nil || repeated == nil || len(repeated.Msg.GetResults()) != 1 || repeated.Msg.GetResults()[0].GetId() != hit.GetId() || repeated.Msg.GetResults()[0].GetSourceHash() != hit.GetSourceHash() || repeated.Msg.GetResults()[0].GetGeneration() != hit.GetGeneration() {
		t.Fatalf("repeated import search changed stable provenance: first=%+v repeated=%+v err=%v", hit, repeated, err)
	}
	for _, unsafe := range []*factsv1.SearchRequest{
		{Query: "context", Families: request.GetFamilies(), Target: request.GetTarget(), Scope: "scenario:other"},
		{Query: "context target:scenarios/other", Families: request.GetFamilies(), Target: request.GetTarget()},
		{Query: "context scope:scenario:other", Families: request.GetFamilies(), Target: request.GetTarget()},
	} {
		if _, err := handler.Search(context.Background(), connect.NewRequest(unsafe)); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("broad import scope error = %v, want invalid argument", err)
		}
	}
}

func TestConnectSearchReportsMalformedTruncatedAndCappedImportEvidence(t *testing.T) {
	truncated := "package api\nimport (\n\"context\"\n"
	var capped string
	capped = "package api\nimport (\n"
	for index := 0; index <= maxImportEdgesPerFile; index++ {
		capped += fmt.Sprintf("\"example.test/pkg%d\"\n", index)
	}
	capped += ")\n"
	oversized := "package api\nimport \"context\"\n" + strings.Repeat("// filler line\n", maxIndexedFile/14)
	index := newImportEndpointFixture(t, indexPolicy, map[string]string{
		"scenarios/demo/api/truncated.go": truncated,
		"scenarios/demo/api/capped.go":    capped,
		"scenarios/demo/api/oversized.go": oversized,
	})
	handler := NewConnectHandler(Deps{Service: internalfacts.NewService(internalfacts.WithSearchBackend(index))})
	response, err := handler.Search(context.Background(), connect.NewRequest(&factsv1.SearchRequest{
		Query: "incomplete", Limit: 10, Families: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
		Target: &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "demo"},
	}))
	if err != nil || response == nil {
		t.Fatalf("incomplete import search response=%+v err=%v", response, err)
	}
	seen := map[string]bool{}
	for _, hit := range response.Msg.GetResults() {
		if hit.GetFactKind() != "imports_partial" || hit.GetEvidenceStatus() != factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN || hit.GetSourceHash() == "" || hit.GetGeneration() == "" {
			t.Fatalf("incomplete import hit lacks UNKNOWN provenance: %+v", hit)
		}
		seen[hit.GetPath()] = true
	}
	for _, path := range []string{"scenarios/demo/api/truncated.go", "scenarios/demo/api/capped.go", "scenarios/demo/api/oversized.go"} {
		if !seen[path] {
			t.Errorf("incomplete import result missing %q; got paths %v", path, seen)
		}
	}
	if !containsString(response.Msg.GetDegradedStages(), "import_index_incomplete") {
		t.Fatalf("incomplete import degradation missing: %v", response.Msg.GetDegradedStages())
	}
}

func TestConnectSearchReturnsUnavailableForPreImportGeneration(t *testing.T) {
	index := newImportEndpointFixture(t, "code-facts-corpus-v1", map[string]string{
		"scenarios/demo/api/server.go": "package api\nimport \"context\"\n",
	})
	handler := NewConnectHandler(Deps{Service: internalfacts.NewService(internalfacts.WithSearchBackend(index))})
	_, err := handler.Search(context.Background(), connect.NewRequest(&factsv1.SearchRequest{
		Query: "context", Families: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
		Target: &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "demo"},
	}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("pre-import active generation error = %v, want unavailable", err)
	}
}

func newImportEndpointFixture(t *testing.T, policy string, files map[string]string) *ProductionIndex {
	t.Helper()
	repoRoot := t.TempDir()
	for path, body := range files {
		writeProductionFixture(t, repoRoot, path, body)
	}
	runGit(t, repoRoot, "init", "--quiet")
	runGit(t, repoRoot, "add", ".")
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "imports.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	index, err := newProductionIndex(db, repoRoot, NewAdmission())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	job := indexcontrol.Job{ID: "imports-fixture", Kind: "reindex", State: "running", Generation: "g-imports", CreatedAt: now, UpdatedAt: now}
	if err := index.catalog.BeginGeneration(context.Background(), catalog.Generation{ID: job.Generation, Policy: policy, DescriptorDigest: "fixture", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := index.jobs.Create(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if err := index.buildGeneration(context.Background(), &job); err != nil {
		t.Fatal(err)
	}
	job.State, job.UpdatedAt = "succeeded", time.Now().UTC()
	if err := index.jobs.Update(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if err := index.Promote(context.Background(), job.Generation); err != nil {
		t.Fatal(err)
	}
	return index
}
