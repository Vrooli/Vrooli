package aisearch

import (
	"context"
	"encoding/json"
	"fmt"
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
		case r.URL.Path == "/aliases":
			_, _ = w.Write([]byte(`{"result":{"aliases":[{"alias_name":"architecture-cartographer-domains","collection_name":"` + collection + `"}]}}`))
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
	if entry.PointCount != 42 || entry.Distance != "Cosine" || !entry.SentinelPresent || entry.SentinelStatus != SentinelComplete || !entry.AliasTarget || len(entry.QdrantAliases) != 1 || entry.QdrantAliases[0] != "architecture-cartographer-domains" {
		t.Fatalf("unexpected inventory entry: %+v", entry)
	}
}

func TestInventoryQdrantCollectionsReportsVerifiedAndUnknownRecipeIdentity(t *testing.T) {
	const physicalGeneration = "owner_docs--generation--arm-1"
	recipe := fixtureEmbeddingRecipe(t, "local/e5-small", 384)
	verifiedPayload, err := json.Marshal(map[string]any{
		metaMarkerKey: true, metaModelKey: "local/e5-small", metaRoleKey: "embedding.default",
		metaDenseSizeKey: 384, metaDenseDistanceKey: "Cosine", metaPolicySchemaKey: "policy-v2",
		metaRecipeIDKey: recipe.ID(), metaRecipeCanonicalKey: string(recipe.Canonical()),
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/aliases":
			_, _ = w.Write([]byte(`{"result":{"aliases":[{"alias_name":"owner_docs","collection_name":"` + physicalGeneration + `"}]}}`))
		case r.URL.Path == "/collections":
			_, _ = w.Write([]byte(`{"result":{"collections":[{"name":"` + physicalGeneration + `"},{"name":"legacy"}]}}`))
		case r.URL.Path == "/collections/"+physicalGeneration:
			_, _ = w.Write([]byte(`{"result":{"config":{"params":{"vectors":{"size":384,"distance":"Cosine"}}}}}`))
		case r.URL.Path == "/collections/legacy":
			_, _ = w.Write([]byte(`{"result":{"config":{"params":{"vectors":{"size":768,"distance":"Cosine"}}}}}`))
		case strings.HasSuffix(r.URL.Path, "/points/count"):
			_, _ = w.Write([]byte(`{"result":{"count":4}}`))
		case strings.Contains(r.URL.Path, "/collections/"+physicalGeneration+"/points/"):
			_, _ = fmt.Fprintf(w, `{"result":{"payload":%s}}`, verifiedPayload)
		case strings.Contains(r.URL.Path, "/collections/legacy/points/"):
			_, _ = w.Write([]byte(`{"result":{"payload":{"__aisearch_meta__":true,"model":"nomic-embed-text","role":"embedding.default","dense_size":768,"policy_schema_version":"v1","dense_distance":"Cosine"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	entries, err := InventoryQdrantCollections(context.Background(), server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	byCollection := map[string]EmbeddingInventoryEntry{}
	for _, entry := range entries {
		byCollection[entry.Store.Collection] = entry
	}
	got := byCollection[physicalGeneration]
	if got.RecipeIdentityStatus != RecipeIdentityVerified || got.Metadata.RecipeID != recipe.ID() || got.Metadata.RecipeCanonical != string(recipe.Canonical()) {
		t.Fatalf("canonical recipe should verify in inventory: %+v", got)
	}
	if !got.AliasTarget || len(got.QdrantAliases) != 1 || got.QdrantAliases[0] != "owner_docs" {
		t.Fatalf("Qdrant alias target relation was omitted: %+v", got)
	}
	if got.PhysicalGenerationID != "arm-1" {
		t.Fatalf("physical generation identity=%q, want arm-1", got.PhysicalGenerationID)
	}
	if legacy := byCollection["legacy"]; legacy.RecipeIdentityStatus != RecipeIdentityUnknown || legacy.Metadata.RecipeID != "" {
		t.Fatalf("legacy sentinel should retain an explicit unknown identity: %+v", legacy)
	}
}

func TestInventoryQdrantCollectionsRejectsMalformedMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/aliases" {
			_, _ = w.Write([]byte(`{"result":{"aliases":[]}}`))
			return
		}
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
	if entries[0].PointCount != 0 || entries[0].SentinelStatus != SentinelAbsent {
		t.Fatalf("missing sentinel should not be subtracted from corpus count: %+v", entries[0])
	}
}

func TestInventoryQdrantCollectionsReportsIncompleteSentinelAndKeepsCorpusCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/aliases":
			_, _ = w.Write([]byte(`{"result":{"aliases":[{"alias_name":"owner_partial","collection_name":"partial"}]}}`))
		case r.URL.Path == "/collections":
			_, _ = w.Write([]byte(`{"result":{"collections":[{"name":"partial"}]}}`))
		case r.URL.Path == "/collections/partial":
			_, _ = w.Write([]byte(`{"result":{"config":{"params":{"vectors":{"size":768,"distance":"Cosine"}}}}}`))
		case strings.Contains(r.URL.Path, "/points/") && !strings.HasSuffix(r.URL.Path, "/points/count"):
			_, _ = w.Write([]byte(`{"result":{"payload":{"__aisearch_meta__":true,"model":"nomic-embed-text"}}}`))
		case strings.HasSuffix(r.URL.Path, "/points/count"):
			_, _ = w.Write([]byte(`{"result":{"count":12}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	entries, err := InventoryQdrantCollections(context.Background(), server.URL, "")
	if err != nil {
		t.Fatalf("InventoryQdrantCollections: %v", err)
	}
	if len(entries) != 1 || entries[0].SentinelStatus != SentinelIncomplete || !entries[0].SentinelPresent || entries[0].PointCount != 11 {
		t.Fatalf("incomplete sentinel inventory = %+v, want incomplete sentinel and 11 corpus points", entries)
	}
}

func TestPhysicalGenerationIDUsesOnlyExplicitQdrantMarker(t *testing.T) {
	if got := physicalGenerationID("agent-manager_conversation--generation--arm-42"); got != "arm-42" {
		t.Fatalf("physical generation id=%q, want arm-42", got)
	}
	if got := physicalGenerationID("owner_generation_arm-42"); got != "" {
		t.Fatalf("unmarked collection inferred generation id %q", got)
	}
}
