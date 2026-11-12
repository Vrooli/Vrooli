package aisearch

import (
	"reflect"
	"testing"

	pkg "github.com/vrooli/ai-go/search"
)

func TestGenerationCensusReportsActiveRetainedAndShadowWithoutInventingRecipe(t *testing.T) {
	service := &Service{
		eng:      &engine{spec: pkg.CollectionSpec{Name: DefaultCollection, Model: "nomic-embed-text:latest", Role: "embedding.default", DenseSize: 768, PolicySchemaVersion: "v1"}, lastActiveEpoch: 4},
		retained: &engine{spec: pkg.CollectionSpec{Name: "architecture-cartographer-domains-v1", Model: "old-model", DenseSize: 384}},
		shadows: map[string]*engine{
			"architecture-cartographer-domains-v3": {spec: pkg.CollectionSpec{Name: "architecture-cartographer-domains-v3", Model: "candidate-model", DenseSize: 1024}},
		},
		lifecycleEpoch: 4,
	}

	got := service.GenerationCensus()
	if len(got) != 3 {
		t.Fatalf("GenerationCensus returned %d entries, want active + retained + shadow: %+v", len(got), got)
	}
	byCollection := make(map[string]GenerationCensusEntry, len(got))
	for _, entry := range got {
		byCollection[entry.Collection] = entry
		if entry.Owner != "architecture-cartographer" || entry.StoreKind != "qdrant" || entry.GenerationID != entry.Collection || entry.PointerEpoch != 4 {
			t.Fatalf("census lost owner/store/generation identity: %+v", entry)
		}
		if entry.RecipeID != "" || entry.RecipeIdentityState != "unknown" {
			t.Fatalf("census fabricated a complete recipe identity: %+v", entry)
		}
		if entry.Durability != "in_process_only" || entry.RestartRecovery != "unsupported" || entry.PhysicalState != "unobserved" {
			t.Fatalf("census overstated restart or physical-store evidence: %+v", entry)
		}
	}
	if byCollection[DefaultCollection].LifecycleState != "active" || byCollection[DefaultCollection].Dimensions != 768 || byCollection[DefaultCollection].Role != "embedding.default" {
		t.Fatalf("active collection identity not reported from owner state: %+v", byCollection[DefaultCollection])
	}
	if byCollection["architecture-cartographer-domains-v1"].LifecycleState != "retained" {
		t.Fatalf("retained collection missing from census: %+v", byCollection)
	}
	if byCollection["architecture-cartographer-domains-v3"].LifecycleState != "shadow" {
		t.Fatalf("shadow collection missing from census: %+v", byCollection)
	}
	if !reflect.DeepEqual(got, service.GenerationCensus()) {
		t.Fatal("generation census order is not deterministic")
	}
}

func TestGenerationCensusOmitsDuplicateCollectionPointers(t *testing.T) {
	active := &engine{spec: pkg.CollectionSpec{Name: DefaultCollection}}
	service := &Service{eng: active, retained: active, shadows: map[string]*engine{"same": active}}
	got := service.GenerationCensus()
	if len(got) != 1 || got[0].LifecycleState != "active" {
		t.Fatalf("duplicate pointer was not represented once with strongest live state: %+v", got)
	}
}
