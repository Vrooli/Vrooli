package retrieval

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"code-facts/internal/catalog"
)

func TestLexicalEvaluationMeetsCategoryRecallAndMRR(t *testing.T) {
	documents := []Document{
		evalDocument("new-service", "scenarios/code-facts/api/internal/facts/service.go", "NewService", "NewService constructs the Code Facts service and its providers."),
		evalDocument("search-method", "scenarios/code-facts/api/internal/facts/service.go", "Service.Search", "Search returns bounded implementation evidence."),
		evalDocument("search-path", "scenarios/code-facts/api/internal/facts/search_lexical.go", "search_lexical.go", "Lexical tokenization and ranking policy."),
		evalDocument("demotion", "scenarios/search-hub/api/internal/metrics/demotion.go", "ProviderDemotion", "Persists provider demotion state and recovery metrics."),
		evalDocument("reconciler", "packages/ai-go/search/reconciler.go", "Reconciler", "Incrementally reconciles vector search documents."),
		{ID: "contract-search", SourceFileID: "file:contract-search", SourceHash: "sha256:contract-search", Path: "packages/proto/schemas/code-facts/v1/facts/facts.proto", Language: "protobuf", Role: "contract", Scope: "scenario:code-facts", Authority: "authoritative", Kind: "contract", Title: "CodeFactsService.Search", ExactText: "CodeFactsService.Search", Body: "Code Facts Search RPC request response", ContractText: "rpc Search(SearchRequest) returns (SearchResponse)"},
	}
	_, index := openRetrievalIndex(t, documents)
	cases := []struct {
		category string
		query    string
		want     string
	}{
		{"exact_identifier", "NewService", "new-service"},
		{"exact_identifier", "Service.Search", "search-method"},
		{"path", "api/internal/facts/search_lexical.go", "search-path"},
		{"natural_language", "where provider demotion state is persisted", "demotion"},
		{"natural_language", "incrementally reconcile vector search documents", "reconciler"},
		{"contract", "CodeFactsService Search RPC request response family:contract", "contract-search"},
		{"typo", "incrementaly vector documents", "reconciler"},
		{"prefix", "ProviderDemo", "demotion"},
		{"acronym", "Code Facts RPC", "contract-search"},
	}
	categoryHits := map[string]int{}
	categoryTotal := map[string]int{}
	reciprocalRank := 0.0
	for _, testCase := range cases {
		results, err := index.SearchLexical(context.Background(), Query{Text: testCase.query, Limit: 5})
		if err != nil {
			t.Fatalf("%s: %v", testCase.category, err)
		}
		categoryTotal[testCase.category]++
		rank := 0
		for index, result := range results {
			if result.ID == testCase.want {
				rank = index + 1
				break
			}
		}
		t.Logf("category=%s query=%q expected=%s rank=%d results=%v", testCase.category, testCase.query, testCase.want, rank, candidateIDs(results))
		if rank > 0 {
			categoryHits[testCase.category]++
			if rank <= 3 {
				reciprocalRank += 1 / float64(rank)
			}
		}
	}
	for category, total := range categoryTotal {
		if recall := float64(categoryHits[category]) / float64(total); recall < 0.95 {
			t.Errorf("%s recall@5 %.2f below 0.95", category, recall)
		}
	}
	if mrr := reciprocalRank / float64(len(cases)); mrr < 0.85 {
		t.Fatalf("MRR@3 %.2f below 0.85", mrr)
	}
	negative, err := index.SearchLexical(context.Background(), Query{Text: "zxqv no-such-code-fact 94721", Limit: 5})
	if err != nil || len(negative) != 0 {
		t.Fatalf("negative query returned evidence: %+v err=%v", negative, err)
	}
}

func candidateIDs(candidates []Candidate) []string {
	ids := make([]string, len(candidates))
	for index, candidate := range candidates {
		ids[index] = candidate.ID
	}
	return ids
}

func TestSQLiteExactSearchPerformanceBudgets(t *testing.T) {
	if os.Getenv("CODE_FACTS_PERF_ASSERT") != "1" {
		t.Skip("set CODE_FACTS_PERF_ASSERT=1 for current and three-times corpus latency proof")
	}
	for _, scale := range []struct {
		name   string
		count  int
		budget time.Duration
	}{
		{name: "current", count: 32000, budget: 100 * time.Millisecond},
		{name: "three-times", count: 96000, budget: 200 * time.Millisecond},
	} {
		t.Run(scale.name, func(t *testing.T) {
			documents := make([]Document, 0, scale.count)
			for id := 0; id < scale.count; id++ {
				documents = append(documents, evalDocument(
					fmt.Sprintf("symbol-%06d", id),
					fmt.Sprintf("scenarios/fixture-%03d/api/internal/symbol_%06d.go", id%1000, id),
					fmt.Sprintf("Symbol%06d", id),
					fmt.Sprintf("Deterministic benchmark symbol %06d for exact retrieval.", id),
				))
			}
			_, index := openRetrievalIndex(t, documents)
			durations := make([]time.Duration, 101)
			for run := range durations {
				started := time.Now()
				results, err := index.SearchLexical(context.Background(), Query{Text: fmt.Sprintf("Symbol%06d", (run*7919)%scale.count), Limit: 5})
				durations[run] = time.Since(started)
				if err != nil || len(results) == 0 {
					t.Fatalf("benchmark query %d: results=%d err=%v", run, len(results), err)
				}
			}
			sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
			p50, p95, p99 := durations[50], durations[95], durations[99]
			allocs := testing.AllocsPerRun(10, func() {
				_, _ = index.SearchLexical(context.Background(), Query{Text: "Symbol000001", Limit: 5})
			})
			t.Logf("documents=%d p50=%s p95=%s p99=%s allocations/query=%.0f", scale.count, p50, p95, p99, allocs)
			if p95 > scale.budget {
				t.Fatalf("exact p95 %s exceeds %s budget", p95, scale.budget)
			}
		})
	}
}

func TestRetrievalProductionDoesNotOpenRepositorySourceFiles(t *testing.T) {
	entries, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), entry, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		importsOS := false
		for _, spec := range file.Imports {
			if spec.Path.Value == `"os"` {
				importsOS = true
			}
		}
		if importsOS {
			t.Errorf("%s imports os; serving retrieval must read only indexed stores", entry)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if ok && (selector.Sel.Name == "Walk" || selector.Sel.Name == "WalkDir") {
				t.Errorf("%s contains filesystem traversal %s", entry, selector.Sel.Name)
			}
			return true
		})
	}
}

func evalDocument(id, path, title, body string) Document {
	return Document{
		ID: id, SourceFileID: catalog.StableFileID(path), SourceHash: "sha256:" + path, Path: path,
		Language: "go", Role: "implementation", Scope: "project:", Authority: "authoritative",
		Kind: "symbol", Title: title, ExactText: title, Body: body,
	}
}
