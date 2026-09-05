package aisearch

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLive_EmbedUpsertSearch exercises the real Ollama embedder + real Qdrant
// vector store end-to-end: it embeds a handful of dependency records into a
// throwaway collection, runs a semantic query, and asserts the most relevant
// record ranks first. Skipped unless SECURITY_HEALTH_LIVE_AISEARCH=1 so CI and
// the normal suite never depend on live resources.
//
// Run with:
//
//	SECURITY_HEALTH_LIVE_AISEARCH=1 go test ./internal/dependencies/aisearch/ -run TestLive -v
func TestLive_EmbedUpsertSearch(t *testing.T) {
	if os.Getenv("SECURITY_HEALTH_LIVE_AISEARCH") != "1" {
		t.Skip("set SECURITY_HEALTH_LIVE_AISEARCH=1 to run the live Ollama+Qdrant proof")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	collection := "security-health-deps-livetest"
	emb := NewEmbedder(DefaultEmbedRole)
	if !emb.Available(ctx) {
		t.Fatal("ollama embedder unavailable — start the ollama resource")
	}
	cfg, err := ResolveConfigEmbedding(ctx, Config{EmbedRole: DefaultEmbedRole})
	if err != nil {
		t.Fatalf("resolve embedding config: %v", err)
	}
	vs := NewVectorStoreForPolicy(DefaultQdrantURL, "", collection, cfg.EmbeddingPolicy)
	if !vs.Available(ctx) {
		t.Fatal("qdrant unavailable — start the qdrant resource")
	}
	if err := vs.EnsureCollection(ctx); err != nil {
		t.Fatalf("ensure collection: %v", err)
	}
	ix := NewIndexer(emb, vs)

	// Package-keyed (ecosystem|name|version) — the index no longer carries the
	// scenario dimension; a package is embedded once regardless of how many
	// scenarios use it.
	items := []Item{
		{Key: "go|golang.org/x/net|v0.17.0", Text: embedTextHelper("golang.org/x/net", "v0.17.0", "Go module", []string{"GO-2023-2102"}, "high")},
		{Key: "npm|esbuild|0.21.5", Text: embedTextHelper("esbuild", "0.21.5", "npm", []string{"GHSA-67mh-4wv8-2f99"}, "moderate")},
		{Key: "npm|left-pad|1.3.0", Text: embedTextHelper("left-pad", "1.3.0", "npm", nil, "")},
	}
	up, _, err := ix.Sync(ctx, items)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	t.Logf("synced %d packages into %s", up, collection)

	// Re-syncing the identical set must upsert 0 (skip-unchanged via
	// payload_hash still holds with the new package keys).
	if up2, _, err := ix.Sync(ctx, items); err != nil {
		t.Fatalf("re-sync: %v", err)
	} else if up2 != 0 {
		t.Errorf("re-sync upserted %d, want 0 (skip-unchanged broken)", up2)
	}

	// Clean up the throwaway points regardless of outcome.
	defer func() {
		ids := make([]string, 0, len(items))
		for _, it := range items {
			ids = append(ids, pointID(it.Key))
		}
		_ = vs.BatchDelete(context.Background(), ids)
	}()

	// Semantic query phrased nothing like the literal package names — the
	// vulnerable Go networking library should still rank first.
	hits, err := ix.Query(ctx, "which dependency has a high-severity vulnerability in a Go networking library", 3)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected at least one semantic hit")
	}
	t.Logf("top hit: %s (score %.4f)", hits[0].Key, hits[0].Score)
	if hits[0].Key != "go|golang.org/x/net|v0.17.0" {
		t.Errorf("semantic ranking miss: top hit %q, want the x/net package", hits[0].Key)
	}
}

func embedTextHelper(name, version, eco string, vulnIDs []string, sev string) string {
	r := struct {
		name, version, eco, sev string
		vulns                   []string
	}{name, version, eco, sev, vulnIDs}
	// Mirror service.packageEmbeddingText's shape without importing the parent
	// package (package-keyed: no scenario / source file).
	out := r.name + " version " + r.version + ", a " + r.eco + " package."
	if len(r.vulns) > 0 {
		out += " Known vulnerabilities:"
		for i, v := range r.vulns {
			if i > 0 {
				out += ","
			}
			out += " " + v
		}
		out += " (maximum severity " + r.sev + ")."
	} else {
		out += " No known vulnerabilities."
	}
	return out
}
