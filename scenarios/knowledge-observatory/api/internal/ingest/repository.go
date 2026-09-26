package ingest

import (
	"context"
	"time"
)

// HistoryEntry is one audited ingest attempt.
type HistoryEntry struct {
	ID             string
	RecordID       string
	Namespace      string
	CollectionName string
	ContentHash    string
	Visibility     string // "private" | "shared" | "global"
	Source         string
	SourceType     string
	Status         string // "success" | "failure"
	ErrorMessage   string
	TookMS         int64
	CreatedAt      time.Time
}

// Job is an asynchronous ingest request and its progress.
type Job struct {
	ID              string
	RequestJSON     string
	Status          string // "pending" | "running" | "success" | "failure"
	ErrorMessage    string
	CreatedAt       time.Time
	StartedAt       *time.Time
	FinishedAt      *time.Time
	TotalChunks     int
	CompletedChunks int
}

// Provenance summarises what a collection's ingest history says about it.
type Provenance struct {
	IngestAttempts     int
	DistinctNamespaces int
	LastIngestAt       *time.Time
}

// Health is the success/failure profile of a collection's ingest attempts.
type Health struct {
	TotalAttempts       int
	SuccessCount        int
	FailureCount        int
	FailureCountLast24H int
	LastFailureAt       *time.Time
}

// Repository is the ingest domain's storage surface.
type Repository interface {
	InsertHistory(ctx context.Context, h HistoryEntry) (string, error)
	GetHistory(ctx context.Context, id string) (HistoryEntry, bool, error)
	ProvenanceForCollection(ctx context.Context, collection string) (Provenance, error)
	HealthForCollection(ctx context.Context, collection string) (Health, error)
	DeleteHistoryByCollection(ctx context.Context, collection string) (int64, error)

	UpsertJob(ctx context.Context, j Job) (string, error)
	GetJob(ctx context.Context, id string) (Job, bool, error)
}
