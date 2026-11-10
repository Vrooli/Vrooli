package facts

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestDescribeGraphsSharesProviderCacheAndPreservesContract(t *testing.T) {
	for _, storage := range []string{"memory", "sqlite"} {
		t.Run(storage, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "go.mod"), "module example.com/graph\n\ngo 1.25\n")
			writeFile(t, filepath.Join(root, "main.go"), "package main\n")
			provider := &countingProvider{fakeProvider: fakeProvider{language: "go", analyzer: "go-code-graph", result: &factsv1.LanguageGraph{
				GraphHash: "graph", ExtractionProfile: "structural", ExtractionMs: 9,
				OmittedInformation: []*factsv1.GraphInformationOmission{{Capability: "resolved_usage_facts", Reason: "structural profile"}},
				Graph:              &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{{Id: "file:main.go", Path: "main.go", Kind: commonv1.NodeKind_NODE_KIND_FILE}}},
			}}}
			var cache CacheRepository = NewMemoryCacheRepository()
			if storage == "sqlite" {
				cache = NewSQLiteCacheRepository(openCacheTestDB(t), 0)
			}
			svc := NewService(WithBroker(NewBroker(provider)), WithCacheRepository(cache))
			req := &factsv1.DescribeCodeFactsRequest{Target: &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PATH, Path: root}, Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS}, UseCache: true}
			facts, err := svc.Describe(context.Background(), req)
			if err != nil || len(facts.GetGraphs()) != 0 {
				t.Fatalf("ordinary facts: %v %v", facts, err)
			}
			req.IncludeGraphs = true
			first, err := svc.Describe(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if len(first.GetGraphs()) != 1 {
				t.Fatalf("graph missing after fact report was cached: %v", first)
			}
			g := first.Graphs[0]
			if g.GetParseUnit().GetRootPath() != root || g.GetExtractionProfile() != "structural" || len(g.GetOmittedInformation()) != 1 || g.GetExtractionMs() != 9 || !g.GetCache().GetHit() {
				t.Fatalf("graph contract missing: %v", g)
			}
			g.Graph.Nodes[0].Path = "caller-mutation"
			second, err := svc.Describe(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if provider.calls != 1 || second.Graphs[0].Graph.Nodes[0].Path != "main.go" {
				t.Fatalf("cache sharing/isolation failed: calls=%d graph=%v", provider.calls, second.Graphs)
			}
			status, err := svc.CacheStatus(context.Background(), &factsv1.GetCacheStatusRequest{Target: req.Target})
			if err != nil || status.GetEntries() != 2 {
				t.Fatalf("graph report duplicated graph cache: %v %v", status, err)
			}
		})
	}
}

func TestDescribeGraphsRetriesUnavailableProviderWithoutSourceChange(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/graph\n\ngo 1.25\n")
	provider := &countingProvider{fakeProvider: fakeProvider{language: "go", analyzer: "go-code-graph", err: ProviderUnavailableError{Analyzer: "go-code-graph", Err: errors.New("restarting")}}}
	svc := NewService(WithBroker(NewBroker(provider)))
	req := &factsv1.DescribeCodeFactsRequest{Target: &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PATH, Path: root}, IncludeGraphs: true, UseCache: true}
	partial, err := svc.Describe(context.Background(), req)
	if err != nil || len(partial.GetWarnings()) == 0 {
		t.Fatalf("missing partial evidence: %v %v", partial, err)
	}
	provider.err = nil
	recovered, err := svc.Describe(context.Background(), req)
	if err != nil || len(recovered.GetGraphs()) != 1 || provider.calls != 2 {
		t.Fatalf("partial report hid provider recovery: %v %v calls=%d", recovered, err, provider.calls)
	}
	req.PageSize = 1
	if _, err := svc.Describe(context.Background(), req); err == nil {
		t.Fatal("graph payload accepted fact-only pagination")
	}
}
