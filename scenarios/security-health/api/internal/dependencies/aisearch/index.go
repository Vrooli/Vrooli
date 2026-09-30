package aisearch

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strings"
)

// payloadHashKey is the field added to every vector-store payload so Sync can
// decide if a point needs re-embedding without comparing the full text. It is
// excluded from the hash input itself.
const payloadHashKey = "payload_hash"

// corpusKeyField is the payload field carrying the caller's corpus key (a
// package identity in the dependencies domain), so a vector hit maps straight
// back to the records it covers. The package is key-agnostic; this is just the
// payload field name under which Item.Key round-trips.
const corpusKeyField = "corpus_key"

// qdrantNamespace seeds the deterministic UUIDv5 point IDs. A fixed, arbitrary
// 16-byte value scoped to this scenario so re-runs upsert (not duplicate).
var qdrantNamespace = [16]byte{
	0x73, 0x65, 0x63, 0x68, 0x6c, 0x74, 0x68, 0x64,
	0x65, 0x70, 0x73, 0x71, 0x64, 0x72, 0x6e, 0x74,
}

// Item is one record to index: a stable corpus Key plus the natural-language
// Text whose embedding ranks it. The caller (dependencies) composes Text from a
// DependencyRecord, keeping this package free of any dependency-domain types.
type Item struct {
	Key  string
	Text string
}

// KeyScore is one ranked hit: the corpus Key and its cosine similarity.
type KeyScore struct {
	Key   string
	Score float64
}

// Indexer is the semantic overlay: it embeds Items into a Qdrant collection and
// answers vector queries. Nil-safe construction via NewIndexer.
type Indexer struct {
	embedder    Embedder
	vectorStore VectorStore
	threshold   float64
}

// NewIndexer wires an embedder + vector store into an Indexer.
func NewIndexer(embedder Embedder, vs VectorStore) *Indexer {
	return &Indexer{embedder: embedder, vectorStore: vs}
}

// NewFromConfig builds the production Indexer (CLI ollama embedder + Qdrant
// store) from environment config. Returns nil when disabled, so callers can
// treat a nil Indexer as "TEXT-only".
func NewFromConfig(cfg Config) *Indexer {
	if cfg.Disabled {
		return nil
	}
	return NewIndexer(
		NewEmbedder(cfg.EmbedRole),
		NewVectorStoreForPolicy(cfg.QdrantURL, cfg.QdrantKey, cfg.Collection, cfg.EmbeddingPolicy),
	)
}

// EnsureCollection creates the Qdrant collection if absent (idempotent).
func (ix *Indexer) EnsureCollection(ctx context.Context) error {
	if ix == nil || ix.vectorStore == nil {
		return nil
	}
	return ix.vectorStore.EnsureCollection(ctx)
}

// Available reports whether the embedder (ollama) and vector store (qdrant) are
// each reachable. A nil Indexer reports both false.
func (ix *Indexer) Available(ctx context.Context) (ollama, qdrant bool) {
	if ix == nil {
		return false, false
	}
	if ix.embedder != nil {
		ollama = ix.embedder.Available(ctx)
	}
	if ix.vectorStore != nil {
		qdrant = ix.vectorStore.Available(ctx)
	}
	return ollama, qdrant
}

// CountPoints reports how many points are currently in the collection — the
// coverage numerator for the readiness gate. A nil Indexer (or nil store)
// reports 0 with no error, so a TEXT-only deployment reads as zero coverage.
func (ix *Indexer) CountPoints(ctx context.Context) (int, error) {
	if ix == nil || ix.vectorStore == nil {
		return 0, nil
	}
	return ix.vectorStore.CountPoints(ctx)
}

// Sync makes the Qdrant collection match items exactly: it embeds + upserts new
// or changed items (skipping those whose payload_hash is unchanged) and deletes
// points whose key is no longer present. Returns the upsert/delete counts.
//
// Best-effort by contract: callers run it after the authoritative SQLite Apply
// and log (never fail) on error, so a down embedder/qdrant degrades search to
// TEXT rather than breaking reconcile.
func (ix *Indexer) Sync(ctx context.Context, items []Item) (upserted, deleted int, err error) {
	if ix == nil || ix.vectorStore == nil || ix.embedder == nil {
		return 0, 0, nil
	}
	if err := ix.vectorStore.EnsureCollection(ctx); err != nil {
		return 0, 0, fmt.Errorf("ensure collection: %w", err)
	}

	existing, err := ix.vectorStore.ScrollIDs(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("scroll existing points: %w", err)
	}

	wantIDs := make(map[string]struct{}, len(items))
	for _, it := range items {
		key := strings.TrimSpace(it.Key)
		if key == "" {
			continue
		}
		id := pointID(key)
		wantIDs[id] = struct{}{}
		hash := payloadHash(it.Text)
		if prior, ok := existing[id]; ok && prior.PayloadHash == hash {
			continue // unchanged — skip the embed call
		}
		vec, err := ix.embedder.Embed(ctx, it.Text)
		if err != nil {
			return upserted, deleted, fmt.Errorf("embed %q: %w", key, err)
		}
		payload := map[string]interface{}{
			corpusKeyField: key,
			payloadHashKey: hash,
		}
		if err := ix.vectorStore.Upsert(ctx, id, vec, payload); err != nil {
			return upserted, deleted, fmt.Errorf("upsert %q: %w", key, err)
		}
		upserted++
	}

	var stale []string
	for id := range existing {
		if _, ok := wantIDs[id]; !ok {
			stale = append(stale, id)
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale) // deterministic batching
		if err := ix.vectorStore.BatchDelete(ctx, stale); err != nil {
			return upserted, deleted, fmt.Errorf("delete stale points: %w", err)
		}
		deleted = len(stale)
	}
	return upserted, deleted, nil
}

// Query embeds the text and returns the corpus keys ranked by vector
// similarity. A blank query or nil Indexer returns no hits (the caller then
// uses structured/TEXT search).
func (ix *Indexer) Query(ctx context.Context, query string, limit int) ([]KeyScore, error) {
	if ix == nil || ix.embedder == nil || ix.vectorStore == nil {
		return nil, nil
	}
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	vec, err := ix.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	results, err := ix.vectorStore.Search(ctx, vec, limit, ix.threshold)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	out := make([]KeyScore, 0, len(results))
	for _, r := range results {
		key, _ := r.Payload[corpusKeyField].(string)
		if strings.TrimSpace(key) == "" {
			log.Printf("[security-health/aisearch] vector hit %s missing %s payload — skipping", r.ID, corpusKeyField)
			continue
		}
		out = append(out, KeyScore{Key: key, Score: r.Score})
	}
	return out, nil
}

// pointID is the deterministic Qdrant point ID for a corpus key (UUIDv5 so a
// re-index upserts the same point rather than duplicating it).
func pointID(key string) string {
	return uuidV5(qdrantNamespace, "security-health:"+key)
}

// payloadHash is a stable short digest of the embedding text, used to skip
// re-embedding unchanged records.
func payloadHash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return "sha256:" + hex.EncodeToString(sum[:8])
}

func uuidV5(namespace [16]byte, name string) string {
	hash := sha1.New()
	_, _ = hash.Write(namespace[:])
	_, _ = hash.Write([]byte(name))
	sum := hash.Sum(nil)

	var uuid [16]byte
	copy(uuid[:], sum[:16])

	uuid[6] = (uuid[6] & 0x0f) | 0x50
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	hexStr := hex.EncodeToString(uuid[:])
	return hexStr[0:8] + "-" + hexStr[8:12] + "-" + hexStr[12:16] + "-" + hexStr[16:20] + "-" + hexStr[20:32]
}
