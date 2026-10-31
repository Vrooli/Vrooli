package metadata

import (
	"context"
	"time"
)

// Entry is cached metadata about one vector stored in Qdrant.
type Entry struct {
	ID             string
	VectorID       string
	CollectionName string
	ContentHash    string
	SourceScenario string
	SourceType     string
	QualityScore   *float64
	AccessCount    int
	LastAccessed   *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ExternalIDMapping ties a caller's own identifier to a record or document, so
// a repeated ingest is recognised instead of duplicated.
type ExternalIDMapping struct {
	ID          string
	Namespace   string
	ExternalID  string
	Kind        string // "record" | "document"
	RecordID    string
	DocumentID  string
	ContentHash string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Repository is the metadata domain's storage surface.
type Repository interface {
	UpsertEntry(ctx context.Context, e Entry) error
	GetEntry(ctx context.Context, vectorID string) (Entry, bool, error)
	LookupCollectionForVectorID(ctx context.Context, vectorID string) (string, bool, error)
	CountByCollection(ctx context.Context, collection string) (int, error)
	DeleteByCollection(ctx context.Context, collection string) (int64, error)

	UpsertExternalIDMapping(ctx context.Context, m ExternalIDMapping) error
	LookupExternalIDMapping(ctx context.Context, namespace, externalID, kind string) (ExternalIDMapping, bool, error)
}
