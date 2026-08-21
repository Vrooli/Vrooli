package facts

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"code-facts/internal/catalog"
	"code-facts/internal/indexcontrol"
	"code-facts/internal/retrieval"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	_ "modernc.org/sqlite"
)

func TestShadowGenerationReviewedCorpus(t *testing.T) {
	dbPath, generation := os.Getenv("CODE_FACTS_LIVE_DB"), os.Getenv("CODE_FACTS_LIVE_GENERATION")
	if dbPath == "" || generation == "" {
		t.Skip("set CODE_FACTS_LIVE_DB and CODE_FACTS_LIVE_GENERATION for shadow comparison")
	}
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "search.json"))
	if err != nil {
		t.Fatal(err)
	}
	var descriptor struct {
		Providers []struct {
			ProviderID string `json:"provider_id"`
			Tests      struct {
				Cases []struct {
					ID, Query, Scope string
					ExpectIDs        []string `json:"expect_ids"`
					TopK             int      `json:"expect_within_top_k"`
					Negative         bool     `json:"expect_no_strong_hit"`
				} `json:"cases"`
			} `json:"tests"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(payload, &descriptor); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	index := retrieval.NewSQLiteIndex(db)
	for _, provider := range descriptor.Providers {
		kind, roles := "source", []string{"implementation"}
		if provider.ProviderID == "code-facts.contracts" {
			kind, roles = "contract", []string{"contract"}
		}
		for _, testCase := range provider.Tests.Cases {
			target, scope := indexedTarget(nil, testCase.Scope)
			if kind == "contract" && strings.HasPrefix(scope, "scenario:") {
				target, scope = "packages/proto/schemas/"+strings.TrimPrefix(scope, "scenario:"), ""
			}
			limit := testCase.TopK
			if limit <= 0 {
				limit = 5
			}
			results, err := index.SearchLexical(context.Background(), retrieval.Query{
				Text: testCase.Query, Target: target, Scope: scope, Roles: roles,
				Families: []string{kind}, Generation: generation, Limit: limit,
			})
			if err != nil {
				t.Errorf("%s/%s: %v", provider.ProviderID, testCase.ID, err)
				continue
			}
			if testCase.Negative {
				if len(results) > 0 && results[0].Score > 0.3 {
					t.Errorf("%s/%s negative top=%s score=%.3f", provider.ProviderID, testCase.ID, results[0].ID, results[0].Score)
				}
				continue
			}
			matched := false
			for _, result := range results {
				for _, expected := range testCase.ExpectIDs {
					matched = matched || result.ID == expected
				}
			}
			if !matched {
				ids := make([]string, len(results))
				for i := range results {
					ids[i] = results[i].ID
				}
				t.Errorf("%s/%s expected=%v results=%v", provider.ProviderID, testCase.ID, testCase.ExpectIDs, ids)
			}
		}
	}
}

func TestProductionIndexSearchReadsActiveSQLiteGenerationAfterSourceRemoval(t *testing.T) {
	repoRoot := t.TempDir()
	writeProductionFixture(t, repoRoot, "packages/demo/demotion.go", "package demo\n\nfunc ComputeProviderDemotion() int { return 7 }\n")
	writeProductionFixture(t, repoRoot, "packages/proto/gen/descriptor/image.binpb", "descriptor-image")
	runGit(t, repoRoot, "init", "--quiet")
	runGit(t, repoRoot, "add", "packages/demo/demotion.go", "packages/proto/gen/descriptor/image.binpb")

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "facts.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
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
	job := indexcontrol.Job{ID: "test-build", Kind: "reindex", State: "running", Generation: "g1", CreatedAt: now, UpdatedAt: now}
	if err := index.catalog.BeginGeneration(context.Background(), catalog.Generation{ID: job.Generation, Policy: indexPolicy, DescriptorDigest: "fixture", CreatedAt: now}); err != nil {
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
	if err := os.Remove(filepath.Join(repoRoot, "packages", "demo", "demotion.go")); err != nil {
		t.Fatal(err)
	}

	response, err := index.Search(context.Background(), &factsv1.SearchRequest{Query: "ComputeProviderDemotion", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.GetResults()) == 0 {
		t.Fatalf("indexed response = %#v, want persisted source evidence", response)
	}
	hit := response.GetResults()[0]
	if hit.GetPath() != "packages/demo/demotion.go" || hit.GetSourceHash() == "" || hit.GetGeneration() != "g1" {
		t.Fatalf("indexed provenance = %#v", hit)
	}
	if hit.GetAnalyzer() != "code-facts.sqlite-fts" || hit.GetRetrievalRegime() != "exact" {
		t.Fatalf("retrieval evidence = %#v", hit)
	}
}

func writeProductionFixture(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
