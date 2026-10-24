// Package blobstore defines storage primitives for opaque binary payloads.
package blobstore

import (
	"context"
	"io"
)

// BlobStore stores bytes under caller-supplied keys with MIME metadata.
type BlobStore interface {
	Put(ctx context.Context, key string, r io.Reader, mime string) error
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
	Delete(ctx context.Context, key string) error
}
