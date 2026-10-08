package ports

import "context"

type VectorStore interface {
	EnsureCollection(ctx context.Context, collection string, vectorSize int) error
	UpsertPoint(ctx context.Context, collection string, id string, vector []float64, payload map[string]interface{}) error
	DeletePoint(ctx context.Context, collection string, id string) error
	Search(ctx context.Context, collection string, vector []float64, limit int, threshold float64, filter *VectorFilter) ([]VectorSearchResult, error)
	ListCollections(ctx context.Context) ([]string, error)
	CountPoints(ctx context.Context, collection string) (int, error)
	SamplePoints(ctx context.Context, collection string, limit int) ([]VectorPoint, error)
}

type VectorSearchResult struct {
	ID      string
	Score   float64
	Payload map[string]interface{}
}

type VectorPoint struct {
	ID      string
	Vector  []float64
	Payload map[string]interface{}
}

type VectorFilter struct {
	Namespaces       []string
	Visibility       []string
	Tags             []string
	IngestedAfterMS  *int64
	IngestedBeforeMS *int64
}

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

type MetadataStore interface {
	UpsertKnowledgeMetadata(ctx context.Context, vectorID, collectionName, contentHash, sourceScenario, sourceType string) error
	InsertIngestHistory(ctx context.Context, row IngestHistoryRow) error
	InsertSearchHistory(ctx context.Context, row SearchHistoryRow) error
	LookupCollectionForVectorID(ctx context.Context, vectorID string) (string, bool, error)
	UpsertExternalIDMapping(ctx context.Context, mapping ExternalIDMapping) error
	LookupExternalIDMapping(ctx context.Context, namespace, externalID, kind string) (ExternalIDMapping, bool, error)
	UpsertQualityMetrics(ctx context.Context, row QualityMetricsRow) error
	UpsertCollectionStats(ctx context.Context, row CollectionStatsRow) error
	UpsertRelationshipEdges(ctx context.Context, edges []RelationshipEdgeRow) error
}

type IngestHistoryRow struct {
	RecordID     string
	Namespace    string
	Collection   string
	ContentHash  string
	Visibility   string
	Source       string
	SourceType   string
	Status       string
	ErrorMessage string
	TookMS       int64
}

type SearchHistoryRow struct {
	Query          string
	Collection     string
	ResultCount    int
	AvgScore       *float64
	ResponseTimeMS int64
	UserSession    string
}

type ExternalIDMapping struct {
	Namespace   string
	ExternalID  string
	Kind        string // "record" | "document"
	RecordID    string
	DocumentID  string
	ContentHash string
}

type QualityMetricsRow struct {
	CollectionName string
	Coherence      *float64
	Freshness      *float64
	Redundancy     *float64
	Coverage       *float64
	TotalEntries   int
}

type CollectionStatsRow struct {
	CollectionName string
	TotalEntries   int
}

type RelationshipEdgeRow struct {
	SourceID         string
	TargetID         string
	RelationshipType string
	Weight           float64
}

type DocAccessLogger interface {
	LogAccess(ctx context.Context, row DocAccessRow) error
	QueryStats(ctx context.Context, filter DocAccessFilter) ([]DocAccessStat, error)
}

type DocAccessRow struct {
	ScenarioName string
	DocType      string
	Operation    string // "read", "write", "reset"
}

type DocAccessFilter struct {
	ScenarioName string
	DocType      string
}

type DocAccessStat struct {
	ScenarioName string
	DocType      string
	ReadCount    int
	WriteCount   int
	ResetCount   int
}
