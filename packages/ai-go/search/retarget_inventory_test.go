package aisearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestInventoryQdrantCollectionsReadsSentinelAndNeverWrites [REQ:REQ-P1-018]
func TestInventoryQdrantCollectionsReadsSentinelAndNeverWrites(t *testing.T) {
	const collection = "architecture-cartographer-domain-map"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && !strings.HasSuffix(r.URL.Path, "/points/count") {
			t.Fatalf("inventory used %s %s; inventory must be read-only", r.Method, r.URL.Path)
		}
		switch {
		case r.URL.Path == "/collections":
			_, _ = w.Write([]byte(`{"result":{"collections":[{"name":"` + collection + `"}]}}`))
		case r.URL.Path == "/collections/"+collection:
			_, _ = w.Write([]byte(`{"result":{"config":{"params":{"vectors":{"size":768,"distance":"Cosine"}}}}}`))
		case strings.HasSuffix(r.URL.Path, "/points/"+PointIDFor(metaIDPrefix, collection, 0, 1)):
			_, _ = w.Write([]byte(`{"result":{"payload":{"__aisearch_meta__":true,"model":"nomic-embed-text","role":"embedding.default","dense_size":768,"policy_schema_version":"v2","dense_distance":"Cosine"}}}`))
		case strings.HasSuffix(r.URL.Path, "/points/count"):
			_, _ = w.Write([]byte(`{"result":{"count":43}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	entries, err := InventoryQdrantCollections(context.Background(), server.URL, "")
	if err != nil {
		t.Fatalf("InventoryQdrantCollections: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("inventory length = %d, want 1", len(entries))
	}
	entry := entries[0]
	if entry.Metadata.Model != "nomic-embed-text" || entry.Metadata.Dimensions != 768 || entry.Metadata.Role != "embedding.default" {
		t.Fatalf("unexpected metadata: %+v", entry.Metadata)
	}
	if entry.PointCount != 42 || entry.Distance != "Cosine" || !entry.SentinelPresent {
		t.Fatalf("unexpected inventory entry: %+v", entry)
	}
}

func TestInventoryQdrantCollectionsRejectsMalformedMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/collections" {
			_, _ = w.Write([]byte(`{"result":{"collections":[{"name":"broken"}]}}`))
			return
		}
		if r.URL.Path == "/collections/broken" {
			_, _ = w.Write([]byte(`{"result":{"config":{"params":{"vectors":{"size":768}}}}}`))
			return
		}
		if strings.Contains(r.URL.Path, "/points/") {
			_, _ = w.Write([]byte(`{"result":{"payload":null}}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/points/count") {
			_, _ = w.Write([]byte(`{"result":{"count":0}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	entries, err := InventoryQdrantCollections(context.Background(), server.URL, "")
	if err != nil {
		t.Fatalf("inventory should retain an explicit sentinel absence result: %v", err)
	}
	blob, _ := json.Marshal(entries)
	if !strings.Contains(string(blob), `"sentinel_present":false`) {
		t.Fatalf("inventory did not report missing sentinel: %s", blob)
	}
}
