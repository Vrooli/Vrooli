package aisearch

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	pkg "github.com/vrooli/ai-go/search"
	"knowledge-observatory/internal/services/docsearch"
)

// [REQ:KO-KB-003]
func TestFallbackIdentityStableAcrossScopes(t *testing.T) {
	root := t.TempDir()
	scenarios := filepath.Join(root, "scenarios")
	if err := os.MkdirAll(filepath.Join(scenarios, "alpha", "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(scenarios, "alpha", "docs", "rule.md")
	if err := os.WriteFile(p, []byte("# Rule\nportable identity needle"), 0o600); err != nil {
		t.Fatal(err)
	}
	svc, err := docsearch.NewService(scenarios)
	if err != nil {
		t.Fatal(err)
	}
	for _, scope := range []pkg.Scope{{Kind: pkg.ScopeGlobal}, {Kind: pkg.ScopeScenario, Value: "alpha"}, {Kind: pkg.ScopePath, Value: "scenarios/alpha/docs"}} {
		hits, err := NewDocsearchFallback(svc)(context.Background(), pkg.SearchQuery{Query: "portable identity needle", Scope: scope, Limit: 5})
		if err != nil {
			t.Fatal(err)
		}
		if len(hits) != 1 {
			t.Fatalf("scope %v: %+v", scope, hits)
		}
		if hits[0].RelativePath != "scenarios/alpha/docs/rule.md" || hits[0].Payload[MetaRelativePath] != hits[0].RelativePath {
			t.Fatalf("unstable identity: %+v", hits[0])
		}
		if hits[0].Payload["knowledge_status"] != "unknown" {
			t.Fatal("fallback invented authority")
		}
	}
}
