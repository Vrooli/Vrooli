package aisearch

import "testing"

func TestComposePayloadHashStable(t *testing.T) {
	payload := map[string]any{
		"scenario": "cli-health",
		"title":    "Search engine",
		bodyKey:    "chunk body text",
	}
	h1 := composePayloadHash("embed text", payload)
	h2 := composePayloadHash("embed text", payload)
	if h1 != h2 {
		t.Fatalf("hash not stable: %s != %s", h1, h2)
	}
	if h1 == "" {
		t.Fatal("hash is empty")
	}
}

func TestComposePayloadHashIgnoresManagedHashes(t *testing.T) {
	base := map[string]any{"title": "x", bodyKey: "y"}
	withHashes := map[string]any{
		"title":        "x",
		bodyKey:        "y",
		payloadHashKey: "sha256:deadbeef",
		sourceHashKey:  "sha256:cafef00d",
	}
	if composePayloadHash("t", base) != composePayloadHash("t", withHashes) {
		t.Fatal("payload_hash/source_hash must be excluded from the chunk hash")
	}
}

func TestComposePayloadHashChangesWithContent(t *testing.T) {
	p := map[string]any{bodyKey: "a"}
	if composePayloadHash("text", p) == composePayloadHash("different text", p) {
		t.Fatal("hash must change when embedding text changes")
	}
	if composePayloadHash("text", map[string]any{bodyKey: "a"}) == composePayloadHash("text", map[string]any{bodyKey: "b"}) {
		t.Fatal("hash must change when payload changes")
	}
}

func TestPointIDForDeterministicAndDistinct(t *testing.T) {
	// Single-chunk source keeps the un-suffixed natural key (cli-health compat).
	single := PointIDFor("cli-health:", "vrooli scenario start", 0, 1)
	if single != PointIDFor("cli-health:", "vrooli scenario start", 0, 1) {
		t.Fatal("PointIDFor must be deterministic")
	}
	// Multi-chunk source suffixes the index → distinct, stable IDs.
	c0 := PointIDFor("knowledge-observatory:", "docs/README.md", 0, 3)
	c1 := PointIDFor("knowledge-observatory:", "docs/README.md", 1, 3)
	if c0 == c1 {
		t.Fatal("distinct chunk indices must yield distinct point IDs")
	}
	// Different consumer prefixes never collide on the same natural key.
	if PointIDFor("a:", "same", 0, 1) == PointIDFor("b:", "same", 0, 1) {
		t.Fatal("different ID prefixes must not collide")
	}
	// A single chunk and chunk-0-of-1 share an ID (the no-suffix rule).
	if single == "" || len(single) != 36 {
		t.Fatalf("expected a 36-char uuid, got %q", single)
	}
}
