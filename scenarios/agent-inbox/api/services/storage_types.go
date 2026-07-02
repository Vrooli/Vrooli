// Package services provides business logic orchestration.
// This file defines types and interfaces for the storage service.
package services

import (
	"context"
	"mime/multipart"

	"agent-inbox/config"
	"agent-inbox/domain"
)

// StorageService defines the interface for file storage operations.
// This abstraction enables testing with mock storage and future
// migration to cloud storage (S3, GCS) without changing callers.
type StorageService interface {
	// Upload stores a file and returns its metadata.
	// The file is stored with a generated UUID name in a date-organized directory.
	Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*domain.Attachment, error)

	// SaveBase64Image saves a base64 data URL image and returns attachment metadata.
	// Used for storing AI-generated images from model responses.
	// The dataURL should be in format: data:image/png;base64,{base64data}
	SaveBase64Image(ctx context.Context, dataURL string, filenamePrefix string) (*domain.Attachment, error)

	// GetPath returns the full filesystem path for a stored file.
	GetPath(storagePath string) string

	// GetFileURL returns the URL for accessing a stored file.
	GetFileURL(storagePath string) string

	// Delete removes a file from storage.
	Delete(ctx context.Context, storagePath string) error

	// ReadAsBase64DataURI reads a file and returns it as a base64 data URI.
	// This is used for sending files to OpenRouter's multimodal API.
	ReadAsBase64DataURI(ctx context.Context, storagePath string) (string, error)

	// IsAllowedType checks if a content type is allowed for upload.
	IsAllowedType(contentType string) bool

	// GetMaxFileSize returns the maximum allowed file size in bytes.
	GetMaxFileSize() int64
}

// LocalStorageService implements StorageService using the local filesystem.
// Files are stored in a date-organized directory structure:
// basePath/YYYY/MM/DD/{uuid}.{ext}
type LocalStorageService struct {
	cfg *config.StorageConfig
}

// NewLocalStorageService creates a new local storage service.
func NewLocalStorageService(cfg *config.StorageConfig) *LocalStorageService {
	return &LocalStorageService{cfg: cfg}
}

// MockStorageService provides an in-memory implementation for testing.
type MockStorageService struct {
	files        map[string]*domain.Attachment
	fileData     map[string][]byte
	maxFileSize  int64
	allowedTypes []string
}

// NewMockStorageService creates a new mock storage service for testing.
func NewMockStorageService() *MockStorageService {
	return &MockStorageService{
		files:        make(map[string]*domain.Attachment),
		fileData:     make(map[string][]byte),
		maxFileSize:  20 * 1024 * 1024,
		allowedTypes: []string{"image/jpeg", "image/png", "image/gif", "image/webp", "application/pdf"},
	}
}
