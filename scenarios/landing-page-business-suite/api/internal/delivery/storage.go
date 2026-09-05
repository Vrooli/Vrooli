package delivery

import (
	"context"
	"time"
)

// Storage is the provider-neutral binary object boundary for release delivery.
//
// seam: Storage
type Storage interface {
	TestConnection(ctx context.Context, bucket string) error
	PresignGet(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
	PresignPut(ctx context.Context, bucket, key string, ttl time.Duration, contentType string) (string, map[string]string, error)
	HeadObject(ctx context.Context, bucket, key string) (etag string, size int64, contentType string, err error)
}

// StorageProvider builds a request-safe storage implementation from persisted
// delivery settings.
type StorageProvider interface {
	ProviderKey() string
	New(ctx context.Context, settings StorageSettings) (Storage, error)
}

// StorageSettings is the domain configuration required to address an artifact
// store. Secrets are intentionally kept server-side and are never serialized
// directly to UI clients.
type StorageSettings struct {
	ID                  int64
	BundleKey           string
	Provider            string
	Bucket              string
	Region              string
	Endpoint            string
	ForcePathStyle      bool
	DefaultPrefix       string
	SignedURLTTLSeconds int
	PublicBaseURL       string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
