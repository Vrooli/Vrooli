package aisearch

import "strings"

type EmbeddingMetadata struct {
	Role                string `json:"role"`
	Model               string `json:"model"`
	Dimensions          int    `json:"dimensions"`
	PolicySchemaVersion string `json:"policy_schema_version,omitempty"`
}

type EmbeddingStoreKind string

const (
	EmbeddingStoreQdrant   EmbeddingStoreKind = "qdrant"
	EmbeddingStorePgvector EmbeddingStoreKind = "pgvector"
)

type EmbeddingStore struct {
	ID              string             `json:"id"`
	Kind            EmbeddingStoreKind `json:"kind"`
	Collection      string             `json:"collection,omitempty"`
	Table           string             `json:"table,omitempty"`
	EntityIDColumn  string             `json:"entity_id_column,omitempty"`
	EmbeddingColumn string             `json:"embedding_column,omitempty"`
	MetadataTable   string             `json:"metadata_table,omitempty"`
}

type PgvectorMetadataPattern struct {
	Table           string   `json:"table"`
	EntityIDColumn  string   `json:"entity_id_column"`
	EmbeddingColumn string   `json:"embedding_column"`
	MetadataTable   string   `json:"metadata_table"`
	RequiredFields  []string `json:"required_fields"`
}

type RetargetCompatibility string

const (
	RetargetNoop              RetargetCompatibility = "compatible_noop"
	RetargetReembed           RetargetCompatibility = "compatible_reembed"
	RetargetIncompatibleShape RetargetCompatibility = "incompatible_shape"
)

type RetargetPlan struct {
	Role           string                `json:"role"`
	Old            EmbeddingMetadata     `json:"old"`
	New            EmbeddingMetadata     `json:"new"`
	Compatibility  RetargetCompatibility `json:"compatibility"`
	RequiredAction string                `json:"required_action"`
	ApplySafety    string                `json:"apply_safety_status"`
	AffectedStores []string              `json:"affected_stores,omitempty"`
	StoreDetails   []EmbeddingStore      `json:"affected_store_details,omitempty"`
}

func PlanEmbeddingRetarget(oldMeta, newMeta EmbeddingMetadata, stores []string) RetargetPlan {
	role := strings.TrimSpace(newMeta.Role)
	if role == "" {
		role = strings.TrimSpace(oldMeta.Role)
	}
	plan := RetargetPlan{
		Role:           role,
		Old:            normalizeEmbeddingMetadata(oldMeta),
		New:            normalizeEmbeddingMetadata(newMeta),
		AffectedStores: append([]string{}, stores...),
		ApplySafety:    "dry-run only; no destructive apply is implemented",
	}
	switch {
	case plan.Old.Model == plan.New.Model &&
		plan.Old.Dimensions == plan.New.Dimensions &&
		plan.Old.Role == plan.New.Role &&
		plan.Old.PolicySchemaVersion == plan.New.PolicySchemaVersion:
		plan.Compatibility = RetargetNoop
		plan.RequiredAction = "none"
	case plan.Old.Dimensions == plan.New.Dimensions:
		plan.Compatibility = RetargetReembed
		plan.RequiredAction = "reembed affected stores before serving mixed vector spaces"
	default:
		plan.Compatibility = RetargetIncompatibleShape
		plan.RequiredAction = "create shadow storage with the new dimensions, reembed, validate, then cut over"
	}
	return plan
}

func PlanEmbeddingRetargetForStores(oldMeta, newMeta EmbeddingMetadata, stores []EmbeddingStore) RetargetPlan {
	normalizedStores := normalizeEmbeddingStores(stores)
	ids := make([]string, 0, len(normalizedStores))
	for _, store := range normalizedStores {
		if store.ID != "" {
			ids = append(ids, store.ID)
		}
	}
	plan := PlanEmbeddingRetarget(oldMeta, newMeta, ids)
	plan.StoreDetails = normalizedStores
	return plan
}

func NewQdrantEmbeddingStore(collection string) EmbeddingStore {
	collection = strings.TrimSpace(collection)
	return EmbeddingStore{
		ID:         "qdrant:" + collection,
		Kind:       EmbeddingStoreQdrant,
		Collection: collection,
	}
}

func NewPgvectorEmbeddingStore(table, entityIDColumn, embeddingColumn string) EmbeddingStore {
	table = strings.TrimSpace(table)
	entityIDColumn = strings.TrimSpace(entityIDColumn)
	embeddingColumn = strings.TrimSpace(embeddingColumn)
	return EmbeddingStore{
		ID:              "postgres:" + table + "." + embeddingColumn,
		Kind:            EmbeddingStorePgvector,
		Table:           table,
		EntityIDColumn:  entityIDColumn,
		EmbeddingColumn: embeddingColumn,
		MetadataTable:   DefaultPgvectorMetadataTable,
	}
}

const DefaultPgvectorMetadataTable = "embedding_metadata"

var PgvectorRequiredMetadataFields = []string{
	"embedding_role",
	"embedding_model",
	"embedding_dimensions",
	"embedding_policy_schema_version",
	"source_content_hash",
	"generated_at",
}

func PgvectorMetadataForColumn(table, entityIDColumn, embeddingColumn string) PgvectorMetadataPattern {
	return PgvectorMetadataPattern{
		Table:           strings.TrimSpace(table),
		EntityIDColumn:  strings.TrimSpace(entityIDColumn),
		EmbeddingColumn: strings.TrimSpace(embeddingColumn),
		MetadataTable:   DefaultPgvectorMetadataTable,
		RequiredFields:  append([]string{}, PgvectorRequiredMetadataFields...),
	}
}

func normalizeEmbeddingMetadata(meta EmbeddingMetadata) EmbeddingMetadata {
	meta.Role = strings.TrimSpace(meta.Role)
	meta.Model = strings.TrimSpace(meta.Model)
	meta.PolicySchemaVersion = strings.TrimSpace(meta.PolicySchemaVersion)
	return meta
}

func normalizeEmbeddingStores(stores []EmbeddingStore) []EmbeddingStore {
	out := make([]EmbeddingStore, 0, len(stores))
	for _, store := range stores {
		store.ID = strings.TrimSpace(store.ID)
		store.Collection = strings.TrimSpace(store.Collection)
		store.Table = strings.TrimSpace(store.Table)
		store.EntityIDColumn = strings.TrimSpace(store.EntityIDColumn)
		store.EmbeddingColumn = strings.TrimSpace(store.EmbeddingColumn)
		store.MetadataTable = strings.TrimSpace(store.MetadataTable)
		if store.ID == "" {
			switch store.Kind {
			case EmbeddingStoreQdrant:
				store.ID = "qdrant:" + store.Collection
			case EmbeddingStorePgvector:
				store.ID = "postgres:" + store.Table + "." + store.EmbeddingColumn
			}
		}
		if store.Kind == EmbeddingStorePgvector && store.MetadataTable == "" {
			store.MetadataTable = DefaultPgvectorMetadataTable
		}
		out = append(out, store)
	}
	return out
}
