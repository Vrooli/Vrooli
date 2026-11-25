package facts

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"code-facts/internal/catalog"
	"code-facts/internal/indexcontrol"
	"code-facts/internal/retrieval"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	_ "modernc.org/sqlite"
)

func TestDescribeFileMetricsUsesActiveCatalogAndFencesStaleFiles(t *testing.T) {
	repoRoot := t.TempDir()
	filePath := "scenarios/demo/api/main.go"
	absPath := filepath.Join(repoRoot, filepath.FromSlash(filePath))
	writeProductionFixture(t, repoRoot, filePath, "package main\n// TODO FIXME HACK\nfunc main() {}\n")
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "metrics.db"))
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
	if err := index.catalog.BeginGeneration(context.Background(), catalog.Generation{ID: "metrics-g1", Policy: indexPolicy, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(content)
	file := catalog.SourceFile{ID: catalog.StableFileID(filePath), Path: filePath, Language: "go", Role: catalog.RoleImplementation, Scope: "scenario:demo", Authority: "authoritative", Owner: "scenario:demo", Hash: fmt.Sprintf("sha256:%x", digest), Size: int64(len(content)), ModTime: now, Searchable: true}
	if err := index.catalog.UpsertFiles(context.Background(), "metrics-g1", []catalog.SourceFile{file}); err != nil {
		t.Fatal(err)
	}
	if err := index.catalog.CompleteGeneration(context.Background(), "metrics-g1", "sha256:test", "fixture"); err != nil {
		t.Fatal(err)
	}
	if err := index.catalog.Activate(context.Background(), "metrics-g1"); err != nil {
		t.Fatal(err)
	}
	target := &factsv1.TargetContext{RootPath: filepath.Join(repoRoot, "scenarios", "demo"), ScenarioAware: true, Scenario: "demo"}
	facts, _, warnings, err := index.DescribeFileMetrics(context.Background(), target)
	if err != nil || len(warnings) != 0 || len(facts) != 1 {
		t.Fatalf("DescribeFileMetrics facts=%+v warnings=%+v err=%v", facts, warnings, err)
	}
	attrs := facts[0].GetAttributes()
	if attrs["file_path"] != "api/main.go" || attrs["line_count"] != "3" || attrs["code_lines"] != "2" ||
		attrs["comment_lines"] != "1" || attrs["todo_count"] != "1" || attrs["fixme_count"] != "1" || attrs["hack_count"] != "1" {
		t.Fatalf("metric attributes = %#v", facts[0].GetAttributes())
	}
	if attrs["test_classification_version"] != "1" || attrs["is_test_file"] != "false" || attrs["has_test_file"] != "false" {
		t.Fatalf("test classification attributes = %#v", attrs)
	}
	writeProductionFixture(t, repoRoot, filePath, "package main\nfunc main() { println(1) }\n")
	facts, _, warnings, err = index.DescribeFileMetrics(context.Background(), target)
	if err != nil || len(facts) != 1 || len(warnings) != 1 || warnings[0].GetCode() != "file_metrics_stale_catalog" {
		t.Fatalf("stale catalog facts=%+v warnings=%+v err=%v", facts, warnings, err)
	}
	if got := facts[0].GetAttributes()["line_count"]; got != "2" {
		t.Fatalf("stale catalog line_count = %q, want the current file's 2 lines", got)
	}
	writeProductionFixture(t, repoRoot, filePath, string([]byte{0xff, '\n'}))
	facts, _, warnings, err = index.DescribeFileMetrics(context.Background(), target)
	if err != nil || len(facts) != 0 || len(warnings) != 1 || warnings[0].GetCode() != "file_metrics_unavailable" {
		t.Fatalf("unsupported binary source facts=%+v warnings=%+v err=%v", facts, warnings, err)
	}
	// A catalog row can outlive its file between reconciliations. A deleted
	// file contributes no metrics and must not degrade the whole projection.
	if err := os.Remove(absPath); err != nil {
		t.Fatal(err)
	}
	facts, _, warnings, err = index.DescribeFileMetrics(context.Background(), target)
	if err != nil || len(facts) != 0 || len(warnings) != 0 {
		t.Fatalf("deleted catalog row facts=%+v warnings=%+v err=%v", facts, warnings, err)
	}
}

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

func TestProductionIndexOrdinaryFileRefreshIsAtomicAndWithinFreshnessBudget(t *testing.T) {
	repoRoot := t.TempDir()
	const sourcePath = "packages/demo/freshness.go"
	writeProductionFixture(t, repoRoot, sourcePath, "package demo\nfunc OldFreshnessSymbol() {}\n")
	writeProductionFixture(t, repoRoot, "packages/proto/gen/descriptor/image.binpb", "descriptor-image")
	runGit(t, repoRoot, "init", "--quiet")
	runGit(t, repoRoot, "add", sourcePath, "packages/proto/gen/descriptor/image.binpb")
	runGit(t, repoRoot, "-c", "user.name=Code Facts Test", "-c", "user.email=code-facts@test.invalid", "commit", "--quiet", "-m", "initial")
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "freshness.db"))
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
	job := indexcontrol.Job{ID: "freshness-build", Kind: "reindex", State: "running", Generation: "g-fresh", CreatedAt: now, UpdatedAt: now}
	if err := index.catalog.BeginGeneration(context.Background(), catalog.Generation{ID: job.Generation, Policy: indexPolicy, DescriptorDigest: "fixture", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := index.jobs.Create(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if err := index.buildGeneration(context.Background(), &job); err != nil {
		t.Fatal(err)
	}
	job.State = "succeeded"
	if err := index.jobs.Update(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if err := index.Promote(context.Background(), job.Generation); err != nil {
		t.Fatal(err)
	}

	primed, err := index.Search(context.Background(), &factsv1.SearchRequest{Query: "OldFreshnessSymbol", Limit: 5})
	if err != nil || !searchResponseContainsPath(primed, sourcePath) {
		t.Fatalf("failed to prime old search result: response=%+v err=%v", primed, err)
	}
	watchCtx, cancelWatch := context.WithCancel(context.Background())
	defer cancelWatch()
	go index.watch(watchCtx)
	started := time.Now()
	writeProductionFixture(t, repoRoot, sourcePath, "package demo\nfunc NewFreshnessSymbol() {}\n")
	var response *factsv1.SearchResponse
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		response, err = index.Search(context.Background(), &factsv1.SearchRequest{Query: "NewFreshnessSymbol", Limit: 5})
		if err == nil && searchResponseContainsPath(response, sourcePath) && searchResponseContainsTitle(response, "NewFreshnessSymbol") {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if elapsed := time.Since(started); elapsed >= 15*time.Second {
		t.Fatalf("ordinary watcher refresh took %s, want <15s", elapsed)
	}
	if err != nil || !searchResponseContainsPath(response, sourcePath) || !searchResponseContainsTitle(response, "NewFreshnessSymbol") {
		t.Fatalf("new source not searchable after atomic refresh: response=%+v err=%v", response, err)
	}
	old, err := index.Search(context.Background(), &factsv1.SearchRequest{Query: "OldFreshnessSymbol", Limit: 5})
	if err != nil || searchResponseContainsPath(old, sourcePath) && searchResponseContainsTitle(old, "OldFreshnessSymbol") {
		t.Fatalf("stale source survived atomic refresh: response=%+v err=%v", old, err)
	}
	cancelWatch()

	// Simulate a watcher event lost across a clean commit. The manifest audit
	// must rediscover the hash drift even though git status is empty.
	runGit(t, repoRoot, "add", sourcePath)
	runGit(t, repoRoot, "-c", "user.name=Code Facts Test", "-c", "user.email=code-facts@test.invalid", "commit", "--quiet", "-m", "second")
	writeProductionFixture(t, repoRoot, sourcePath, "package demo\nfunc AuditRepairSymbol() {}\n")
	runGit(t, repoRoot, "add", sourcePath)
	runGit(t, repoRoot, "-c", "user.name=Code Facts Test", "-c", "user.email=code-facts@test.invalid", "commit", "--quiet", "-m", "missed-event")
	if dirty, err := index.dirtyPaths(context.Background()); err != nil || len(dirty) != 0 {
		t.Fatalf("missed-event fixture is not clean: paths=%v err=%v", dirty, err)
	}
	drift, err := index.manifestDrift(context.Background())
	if err != nil || !slices.Contains(drift, sourcePath) {
		t.Fatalf("manifest audit missed committed drift: paths=%v err=%v", drift, err)
	}
	index.refreshBatch(context.Background(), drift)
	repaired, err := index.Search(context.Background(), &factsv1.SearchRequest{Query: "AuditRepairSymbol", Limit: 5})
	if err != nil || !searchResponseContainsPath(repaired, sourcePath) {
		t.Fatalf("manifest audit did not repair missed event: response=%+v err=%v", repaired, err)
	}
	if err := os.Remove(filepath.Join(repoRoot, filepath.FromSlash(sourcePath))); err != nil {
		t.Fatal(err)
	}
	if err := index.refreshPath(context.Background(), job.Generation, sourcePath); err != nil {
		t.Fatal(err)
	}
	deleted, err := index.Search(context.Background(), &factsv1.SearchRequest{Query: "NewFreshnessSymbol", Limit: 5})
	if err != nil || searchResponseContainsPath(deleted, sourcePath) {
		t.Fatalf("deleted source survived atomic refresh: response=%+v err=%v", deleted, err)
	}
}

func TestProductionIndexManifestAuditSkipsFileReadsAtReconciledRevision(t *testing.T) {
	repoRoot := t.TempDir()
	writeProductionFixture(t, repoRoot, "packages/demo/audit.go", "package demo\nfunc AuditFixture() {}\n")
	writeProductionFixture(t, repoRoot, "packages/proto/gen/descriptor/image.binpb", "descriptor-image")
	runGit(t, repoRoot, "init", "--quiet")
	runGit(t, repoRoot, "add", ".")
	runGit(t, repoRoot, "-c", "user.name=Code Facts Test", "-c", "user.email=code-facts@test.invalid", "commit", "--quiet", "-m", "initial")

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "audit.db"))
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
	job := indexcontrol.Job{ID: "audit-build", Kind: "reindex", State: "running", Generation: "g-audit", CreatedAt: now, UpdatedAt: now}
	if err := index.catalog.BeginGeneration(context.Background(), catalog.Generation{ID: job.Generation, Policy: indexPolicy, DescriptorDigest: "fixture", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := index.jobs.Create(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if err := index.buildGeneration(context.Background(), &job); err != nil {
		t.Fatal(err)
	}
	job.State = "succeeded"
	if err := index.jobs.Update(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if err := index.Promote(context.Background(), job.Generation); err != nil {
		t.Fatal(err)
	}
	revision := strings.TrimSpace(gitOutput(t, repoRoot, "rev-parse", "HEAD"))
	if err := index.catalog.SetGenerationRevision(context.Background(), job.Generation, revision); err != nil {
		t.Fatal(err)
	}
	inspector := &countingFileInspector{delegate: catalog.OSFileInspector{}}
	index.inspector = inspector

	drift, err := index.manifestDrift(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(drift) != 0 {
		t.Fatalf("unchanged revision drift = %v", drift)
	}
	if got := inspector.calls.Load(); got != 0 {
		t.Fatalf("unchanged revision inspected %d files, want 0", got)
	}

	writeProductionFixture(t, repoRoot, "packages/demo/audit.go", "package demo\nfunc RevisedAuditFixture() {}\n")
	runGit(t, repoRoot, "add", "packages/demo/audit.go")
	runGit(t, repoRoot, "-c", "user.name=Code Facts Test", "-c", "user.email=code-facts@test.invalid", "commit", "--quiet", "-m", "revise")
	inspector.calls.Store(0)
	drift, err = index.manifestDrift(context.Background())
	if err != nil || !slices.Contains(drift, "packages/demo/audit.go") {
		t.Fatalf("revision delta drift = %v, err=%v", drift, err)
	}
	if got := inspector.calls.Load(); got != 0 {
		t.Fatalf("revision delta inspected %d files before refresh, want 0", got)
	}
	if err := index.auditManifest(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := inspector.calls.Load(); got != 2 {
		t.Fatalf("revision delta filtered and refreshed with %d file inspections, want 2", got)
	}
	inspector.calls.Store(0)
	if drift, err := index.manifestDrift(context.Background()); err != nil || len(drift) != 0 {
		t.Fatalf("completed revision audit drift = %v, err=%v", drift, err)
	}
	if got := inspector.calls.Load(); got != 0 {
		t.Fatalf("completed revision audit inspected %d files, want 0", got)
	}
}

type countingFileInspector struct {
	delegate catalog.FileInspector
	calls    atomic.Int64
}

func (i *countingFileInspector) Inspect(ctx context.Context, path string) (catalog.FileSnapshot, error) {
	i.calls.Add(1)
	return i.delegate.Inspect(ctx, path)
}

func searchResponseContainsPath(response *factsv1.SearchResponse, path string) bool {
	for _, hit := range response.GetResults() {
		if hit.GetPath() == path {
			return true
		}
	}
	return false
}

func searchResponseContainsTitle(response *factsv1.SearchResponse, title string) bool {
	for _, hit := range response.GetResults() {
		if strings.Contains(hit.GetTitle(), title) {
			return true
		}
	}
	return false
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

func gitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(output)
}

func TestProductionIndexSearchReturnsScenarioScopedGoImportEdges(t *testing.T) {
	repoRoot := t.TempDir()
	const sourcePath = "scenarios/demo/api/server.go"
	writeProductionFixture(t, repoRoot, sourcePath, "package api\nimport (\n\t\"errors\"\n\t\"context\"\n)\nfunc Serve(context.Context) error { return errors.New(\"x\") }\n")
	writeProductionFixture(t, repoRoot, "scenarios/other/api/server.go", "package api\nimport \"context\"\n")
	writeProductionFixture(t, repoRoot, "scenarios/demo/ui/src/App.tsx", "import React from 'react';\nexport { Button } from '../Button';\n")
	writeProductionFixture(t, repoRoot, "packages/proto/gen/descriptor/image.binpb", "descriptor-image")
	runGit(t, repoRoot, "init", "--quiet")
	runGit(t, repoRoot, "add", ".")

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "imports.db"))
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
	job := indexcontrol.Job{ID: "imports-build", Kind: "reindex", State: "running", Generation: "g-imports", CreatedAt: now, UpdatedAt: now}
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

	response, err := index.Search(context.Background(), &factsv1.SearchRequest{
		Query: "context", Limit: 5, Families: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
		Languages: []string{"go"}, Scope: "scenario:demo",
		Target: &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "demo"},
	})
	if err != nil || len(response.GetResults()) != 1 {
		t.Fatalf("scoped import response=%+v err=%v", response, err)
	}
	hit := response.GetResults()[0]
	if hit.GetPath() != sourcePath || hit.GetFactKind() != "imports" || hit.GetStartLine() != 4 || hit.GetEndLine() != 4 || hit.GetSourceHash() == "" || hit.GetGeneration() != job.Generation || hit.GetScope() != "scenario:demo" {
		t.Fatalf("import edge provenance = %+v", hit)
	}

	tsResponse, err := index.Search(context.Background(), &factsv1.SearchRequest{
		Query: "react", Limit: 5, Families: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
		Languages: []string{"typescript"}, Scope: "scenario:demo",
		Target: &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "demo"},
	})
	if err != nil || len(tsResponse.GetResults()) != 1 {
		t.Fatalf("scoped TypeScript import response=%+v err=%v", tsResponse, err)
	}
	tsHit := tsResponse.GetResults()[0]
	if tsHit.GetPath() != "scenarios/demo/ui/src/App.tsx" || tsHit.GetFactKind() != "imports" || tsHit.GetStartLine() != 1 || tsHit.GetSourceHash() == "" || tsHit.GetGeneration() != job.Generation || tsHit.GetScope() != "scenario:demo" {
		t.Fatalf("TypeScript import edge provenance = %+v", tsHit)
	}

	for _, request := range []*factsv1.SearchRequest{
		{Query: "context", Families: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS}, Target: &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PROJECT}},
		{Query: "context", Families: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS}, Scope: "scenario:other", Target: &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "demo"}},
	} {
		if _, err := index.Search(context.Background(), request); err == nil {
			t.Fatalf("unsupported import search scope was accepted: %+v", request)
		}
	}
}
