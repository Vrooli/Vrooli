package embedding

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	aisearch "github.com/vrooli/ai-go/search"
)

func TestEmbeddingInventoryCallPublishesCompleteOrUnknownQdrantBytes(t *testing.T) {
	t.Setenv("QDRANT_URL", "http://qdrant.fixture")
	tests := []struct {
		name      string
		entries   []aisearch.EmbeddingInventoryEntry
		wantBytes int64
		wantKnown bool
	}{
		{
			name: "complete recipe and generation inventory",
			entries: []aisearch.EmbeddingInventoryEntry{
				{Store: aisearch.NewQdrantEmbeddingStore("docs--generation--g1"), Metadata: aisearch.EmbeddingMetadata{RecipeID: "sha256:recipe", RecipeCanonical: `{"model_id":"embed"}`}, GenerationIDStatus: aisearch.GenerationIdentityObserved, PhysicalGenerationID: "g1", PointCount: 42, QdrantDiskBytes: 1024, QdrantDiskBytesKnown: true},
				{Store: aisearch.NewQdrantEmbeddingStore("legacy"), GenerationIDStatus: aisearch.GenerationIdentityUnknown, PointCount: 7, QdrantDiskBytes: 2048, QdrantDiskBytesKnown: true},
			},
			wantBytes: 3072, wantKnown: true,
		},
		{
			name: "one unknown collection suppresses partial total",
			entries: []aisearch.EmbeddingInventoryEntry{
				{Store: aisearch.NewQdrantEmbeddingStore("docs--generation--g1"), GenerationIDStatus: aisearch.GenerationIdentityObserved, PhysicalGenerationID: "g1", PointCount: 42, QdrantDiskBytes: 1024, QdrantDiskBytesKnown: true},
				{Store: aisearch.NewQdrantEmbeddingStore("legacy"), GenerationIDStatus: aisearch.GenerationIdentityUnknown, PointCount: 7},
			},
			wantKnown: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handlers{inventory: func(ctx context.Context, _, _ string) ([]aisearch.EmbeddingInventoryEntry, error) {
				if _, ok := ctx.Deadline(); !ok {
					t.Error("inventory call must have a bounded context")
				}
				return tt.entries, nil
			}}
			got, err := h.inventoryCall(nil)
			if err != nil {
				t.Fatal(err)
			}
			if got.Summary == nil || got.Summary.QdrantDiskBytesKnown != tt.wantKnown || got.Summary.QdrantDiskBytes != tt.wantBytes {
				t.Fatalf("summary=%+v, want bytes=%d known=%v", got.Summary, tt.wantBytes, tt.wantKnown)
			}
			if got.StartedAt == nil || got.EndedAt == nil || got.EndedAt.Before(*got.StartedAt) {
				t.Fatalf("inventory observation window is invalid: start=%v end=%v", got.StartedAt, got.EndedAt)
			}
			if tt.wantKnown && !strings.Contains(got.Message, "Qdrant-reported bytes=3072") || !tt.wantKnown && !strings.Contains(got.Message, "Qdrant-reported bytes=unknown") {
				t.Fatalf("human summary is not explicit about byte completeness: %q", got.Message)
			}
			blob, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			var decoded map[string]any
			if err := json.Unmarshal(blob, &decoded); err != nil {
				t.Fatal(err)
			}
			rows := decoded["inventory"].([]any)
			row := rows[0].(map[string]any)
			if tt.name == "complete recipe and generation inventory" {
				metadata := row["metadata"].(map[string]any)
				if metadata["recipe_id"] != "sha256:recipe" || metadata["recipe_canonical"] != `{"model_id":"embed"}` || row["physical_generation_id"] != "g1" || row["generation_identity_status"] != aisearch.GenerationIdentityObserved || row["point_count"] != float64(42) || row["qdrant_disk_bytes"] != float64(1024) || row["qdrant_disk_bytes_known"] != true {
					t.Fatalf("inventory row omitted canonical identity or storage facts: %s", blob)
				}
			} else {
				unknownRow := rows[1].(map[string]any)
				if unknownRow["generation_identity_status"] != aisearch.GenerationIdentityUnknown || unknownRow["physical_generation_id"] != "" || unknownRow["qdrant_disk_bytes_known"] != false || decoded["inventory_summary"].(map[string]any)["qdrant_disk_bytes_known"] != false || decoded["inventory_summary"].(map[string]any)["qdrant_disk_bytes"] != float64(0) {
					t.Fatalf("unknown storage must remain explicit and suppress total: %s", blob)
				}
			}
		})
	}
}

func TestEmbeddingInventoryCallReadsQdrantRecipeGenerationPointsAndBytesEndToEnd(t *testing.T) {
	const collection = "search-hub_docs--generation--arm-7"
	recipe, err := aisearch.NewEmbeddingRecipe(aisearch.EmbeddingRecipeInput{
		SchemaVersion: "embedding-recipe-v1", ModelID: "local/e5-small", ModelRevision: "revision-7",
		ArtifactSHA256: strings.Repeat("a", 64), TokenizerRevision: "tokenizer-7", Runtime: "onnx",
		RuntimeBuild: "runtime-1", Precision: "fp32", QueryInstruction: "query: ", DocumentInstruction: "passage: ",
		Pooling: "mean", Normalization: "l2", Dimensions: 384, OutputDType: "float32", Truncation: "512",
		ChunkerVersion: "chunk-v4", ComposerVersion: "search-hub-v7",
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"__aisearch_meta__": true, "model": "local/e5-small", "role": "embedding.default",
		"dense_size": 384, "dense_distance": "Cosine", "policy_schema_version": "policy-v7",
		"embedding_recipe_id": recipe.ID(), "embedding_recipe_canonical": string(recipe.Canonical()),
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/points/count") {
			if r.Method != http.MethodPost {
				t.Fatalf("count method=%s; want exact count read POST", r.Method)
			}
			body, readErr := io.ReadAll(r.Body)
			if readErr != nil || strings.TrimSpace(string(body)) != `{"exact":true}` {
				t.Fatalf("count body=%q err=%v", body, readErr)
			}
			_, _ = w.Write([]byte(`{"result":{"count":11}}`))
			return
		}
		if r.Method != http.MethodGet {
			t.Fatalf("inventory used non-read method %s %s", r.Method, r.URL.Path)
		}
		switch {
		case r.URL.Path == "/aliases":
			_, _ = fmt.Fprintf(w, `{"result":{"aliases":[{"alias_name":"search-hub_docs","collection_name":%q}]}}`, collection)
		case r.URL.Path == "/collections":
			_, _ = fmt.Fprintf(w, `{"result":{"collections":[{"name":%q}]}}`, collection)
		case r.URL.Path == "/collections/"+collection:
			_, _ = w.Write([]byte(`{"result":{"disk_data_size":4096,"config":{"params":{"vectors":{"size":384,"distance":"Cosine"}}}}}`))
		case strings.HasPrefix(r.URL.Path, "/collections/"+collection+"/points/"):
			_, _ = fmt.Fprintf(w, `{"result":{"payload":%s}}`, payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("QDRANT_URL", server.URL)
	h := &handlers{}
	got, err := h.inventoryCall(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Inventory) != 1 {
		t.Fatalf("inventory returned %d rows", len(got.Inventory))
	}
	entry := got.Inventory[0]
	if entry.PhysicalGenerationID != "arm-7" || entry.GenerationIDStatus != aisearch.GenerationIdentityObserved || entry.RecipeIdentityStatus != aisearch.RecipeIdentityVerified || entry.Metadata.RecipeID != recipe.ID() || entry.Metadata.RecipeCanonical != string(recipe.Canonical()) {
		t.Fatalf("canonical recipe/generation not emitted by owner CLI: %+v", entry)
	}
	if entry.PointCount != 10 || !entry.QdrantDiskBytesKnown || entry.QdrantDiskBytes != 4096 {
		t.Fatalf("Qdrant point/byte observations not emitted: %+v", entry)
	}
	if got.Summary == nil || !got.Summary.QdrantDiskBytesKnown || got.Summary.QdrantDiskBytes != 4096 {
		t.Fatalf("complete physical aggregate not emitted: %+v", got.Summary)
	}
	if got.StartedAt == nil || got.EndedAt == nil || got.EndedAt.Before(*got.StartedAt) {
		t.Fatalf("inventory did not retain its bounded observation window: start=%v end=%v", got.StartedAt, got.EndedAt)
	}
}
