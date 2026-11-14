package aisearch

import "testing"

// TestPlanEmbeddingRetargetClassifiesIncompatibleShape [REQ:REQ-P1-018]
func TestPlanEmbeddingRetargetClassifiesIncompatibleShape(t *testing.T) {
	plan := PlanEmbeddingRetarget(
		EmbeddingMetadata{Role: "embedding.default", Model: "old-model", Dimensions: fixtureEmbeddingDimensions, PolicySchemaVersion: "old"},
		EmbeddingMetadata{Role: "embedding.default", Model: "new-model", Dimensions: 2345, PolicySchemaVersion: "new"},
		[]string{"qdrant:docs", "postgres:items.embedding"},
	)
	if plan.Compatibility != RetargetIncompatibleShape {
		t.Fatalf("compatibility = %q", plan.Compatibility)
	}
	if plan.RequiredAction == "" || plan.ApplySafety == "" {
		t.Fatalf("plan must carry action and safety status: %+v", plan)
	}
}

func TestPlanEmbeddingRetargetClassifiesSameDimensionModelChange(t *testing.T) {
	plan := PlanEmbeddingRetarget(
		EmbeddingMetadata{Role: "embedding.default", Model: "old-model", Dimensions: fixtureEmbeddingDimensions},
		EmbeddingMetadata{Role: "embedding.default", Model: "new-model", Dimensions: fixtureEmbeddingDimensions},
		nil,
	)
	if plan.Compatibility != RetargetReembed {
		t.Fatalf("compatibility = %q", plan.Compatibility)
	}
}

func TestPlanEmbeddingRetargetClassifiesUnchangedMetadata(t *testing.T) {
	meta := EmbeddingMetadata{Role: "embedding.default", Model: "same-model", Dimensions: fixtureEmbeddingDimensions, PolicySchemaVersion: "same"}
	plan := PlanEmbeddingRetarget(meta, meta, nil)
	if plan.Compatibility != RetargetNoop {
		t.Fatalf("compatibility = %q", plan.Compatibility)
	}
	if plan.RequiredAction != "none" {
		t.Fatalf("required_action = %q", plan.RequiredAction)
	}
}

func TestPlanEmbeddingRetargetForStoresIncludesPgvectorDetails(t *testing.T) {
	plan := PlanEmbeddingRetargetForStores(
		EmbeddingMetadata{Role: "embedding.default", Model: "old-model", Dimensions: fixtureEmbeddingDimensions},
		EmbeddingMetadata{Role: "embedding.default", Model: "new-model", Dimensions: 2345},
		[]EmbeddingStore{
			NewQdrantEmbeddingStore("docs"),
			NewPgvectorEmbeddingStore("items", "id", "embedding"),
		},
	)
	if len(plan.AffectedStores) != 2 {
		t.Fatalf("AffectedStores = %#v", plan.AffectedStores)
	}
	if len(plan.StoreDetails) != 2 {
		t.Fatalf("StoreDetails = %#v", plan.StoreDetails)
	}
	pg := plan.StoreDetails[1]
	if pg.Kind != EmbeddingStorePgvector || pg.ID != "postgres:items.embedding" || pg.MetadataTable != DefaultPgvectorMetadataTable {
		t.Fatalf("pgvector store detail = %#v", pg)
	}
	if plan.Compatibility != RetargetIncompatibleShape {
		t.Fatalf("compatibility = %q", plan.Compatibility)
	}
}

func TestPgvectorMetadataForColumnDeclaresRequiredRetargetFields(t *testing.T) {
	pattern := PgvectorMetadataForColumn("items", "id", "embedding")
	want := []string{
		"embedding_role",
		"embedding_model",
		"embedding_dimensions",
		"embedding_policy_schema_version",
		"source_content_hash",
		"generated_at",
	}
	if len(pattern.RequiredFields) != len(want) {
		t.Fatalf("RequiredFields = %#v", pattern.RequiredFields)
	}
	for i := range want {
		if pattern.RequiredFields[i] != want[i] {
			t.Fatalf("RequiredFields[%d] = %q, want %q", i, pattern.RequiredFields[i], want[i])
		}
	}
	if pattern.MetadataTable != DefaultPgvectorMetadataTable {
		t.Fatalf("MetadataTable = %q", pattern.MetadataTable)
	}
}
