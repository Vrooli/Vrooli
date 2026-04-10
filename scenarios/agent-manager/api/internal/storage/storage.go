// Package storage provides file storage for uploaded attachments.
package storage

import (
	"context"
	"mime/multipart"
	"time"
)

// Service defines the interface for file storage operations.
type Service interface {
	// Upload stores a file and returns its metadata.
	Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*AttachmentMeta, error)
	// Get returns metadata for an attachment by ID.
	Get(ctx context.Context, id string) (*AttachmentMeta, error)
	// GetMultiple returns metadata for multiple attachments by IDs.
	GetMultiple(ctx context.Context, ids []string) ([]*AttachmentMeta, error)
	// GetFilePath returns the absolute filesystem path for a stored file.
	GetFilePath(storagePath string) string
	// GetServingURL returns the URL path for HTTP serving.
	GetServingURL(storagePath string) string
	// Delete removes a file from storage.
	Delete(ctx context.Context, id string) error
	// IsAllowedType checks if a content type is allowed for upload.
	IsAllowedType(contentType string) bool
	// MaxFileSize returns the maximum allowed file size in bytes.
	MaxFileSize() int64
}

// AttachmentMeta contains metadata about a stored file.
type AttachmentMeta struct {
	ID          string
	FileName    string
	ContentType string
	FileSize    int64
	StoragePath string // Relative path within storage root
	CreatedAt   time.Time
}
