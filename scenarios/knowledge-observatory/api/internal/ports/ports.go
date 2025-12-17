package ports

import "context"

type VectorStore interface {
	EnsureCollection(ctx context.Context, collection string, vectorSize int) error
	UpsertPoint(ctx context.Context, collection string, id string, vector []float64, payload map[string]interface{}) error
	DeletePoint(ctx context.Context, collection string, id string) error
	Search(ctx context.Context, collection string, vector []float64, limit int, threshold float64) ([]VectorSearchResult, error)
	ListCollections(ctx context.Context) ([]string, error)
	CountPoints(ctx context.Context, collection string) (int, error)
}

type VectorSearchResult struct {
	ID      string
	Score   float64
	Payload map[string]interface{}
}

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

type MetadataStore interface {
	UpsertKnowledgeMetadata(ctx context.Context, vectorID, collectionName, contentHash, sourceScenario, sourceType string) error
	InsertIngestHistory(ctx context.Context, row IngestHistoryRow) error
	LookupCollectionForVectorID(ctx context.Context, vectorID string) (string, bool, error)
}

type IngestHistoryRow struct {
	RecordID      string
	Namespace     string
	Collection    string
	ContentHash   string
	Visibility    string
	Source        string
	SourceType    string
	Status        string
	ErrorMessage  string
	TookMS        int64
}

type JobStore interface {
	EnqueueDocumentIngest(ctx context.Context, req DocumentIngestJobRequest) (jobID string, err error)
	GetJob(ctx context.Context, jobID string) (JobStatus, bool, error)
	ClaimNextPendingJob(ctx context.Context) (DocumentIngestJob, bool, error)
	UpdateJobProgress(ctx context.Context, jobID string, completedChunks, totalChunks int) error
	CompleteJob(ctx context.Context, jobID string, status string, errorMessage string) error
}

type DocumentIngestJobRequest struct {
	Namespace    string
	Collection   string
	DocumentID   string
	Content      string
	Metadata     map[string]interface{}
	Visibility   string
	Source       string
	SourceType   string
	ChunkSize    int
	ChunkOverlap int
}

type DocumentIngestJob struct {
	JobID string
	Req   DocumentIngestJobRequest
}

type JobStatus struct {
	JobID           string
	Status          string
	ErrorMessage    string
	CreatedAt       *string
	StartedAt       *string
	FinishedAt      *string
	TotalChunks     int
	CompletedChunks int
}
