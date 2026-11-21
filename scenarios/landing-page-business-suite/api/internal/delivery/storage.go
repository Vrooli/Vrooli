package delivery

import (
	"context"
	"io"
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

// ObjectReader is an optional capability for providers that can return the
// object bytes. It is separate from Storage so existing provider adapters keep
// their shallow connectivity contract while deep verification can require the
// stronger capability explicitly.
type ObjectReader interface {
	ReadObject(ctx context.Context, bucket, key string) (io.ReadCloser, int64, string, error)
}

// OperationVerifier is an optional provider capability for readiness checks.
// A provider that implements it must use a bounded, cleanup-owned key and
// verify the actual operations required by distribution, not just bucket
// discovery.
type OperationVerifier interface {
	VerifyOperations(ctx context.Context, bucket, prefix string) error
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
