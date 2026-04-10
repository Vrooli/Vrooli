// Package services provides business logic orchestration.
// This file contains helper functions and the mock implementation for storage.
package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"agent-inbox/domain"

	"github.com/google/uuid"
)

// Helper functions

// detectContentType infers content type from filename extension.
func detectContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

// extensionFromContentType returns the file extension for a content type.
func extensionFromContentType(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}

// getImageDimensions returns width and height of an image file.
// Returns (0, 0) if dimensions cannot be determined.
func getImageDimensions(path string) (width, height int) {
	// For now, return 0,0. In a full implementation, we'd use
	// image.DecodeConfig from the standard library or a third-party lib.
	// This is deferred to avoid adding dependencies until needed.
	return 0, 0
}

// parseBase64DataURL parses a data URL and returns content type and decoded data.
// Expected format: data:image/png;base64,{base64data}
func parseBase64DataURL(dataURL string) (contentType string, data []byte, err error) {
	// Check for data URL prefix
	if !strings.HasPrefix(dataURL, "data:") {
		return "", nil, fmt.Errorf("invalid data URL: missing 'data:' prefix")
	}

	// Find the comma that separates metadata from data
	commaIdx := strings.Index(dataURL, ",")
	if commaIdx == -1 {
		return "", nil, fmt.Errorf("invalid data URL: missing comma separator")
	}

	// Parse metadata (e.g., "data:image/png;base64")
	metadata := dataURL[5:commaIdx] // Skip "data:"
	base64Data := dataURL[commaIdx+1:]

	// Extract content type and verify it's base64
	parts := strings.Split(metadata, ";")
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("invalid data URL: no content type")
	}
	contentType = parts[0]

	// Verify base64 encoding
	isBase64 := false
	for _, part := range parts[1:] {
		if part == "base64" {
			isBase64 = true
			break
		}
	}
	if !isBase64 {
		return "", nil, fmt.Errorf("invalid data URL: not base64 encoded")
	}

	// Decode the base64 data
	data, err = base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	return contentType, data, nil
}

// Mock storage implementation

// Upload stores a file in memory.
func (m *MockStorageService) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*domain.Attachment, error) {
	if header.Size > m.maxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum %d bytes", header.Size, m.maxFileSize)
	}

	contentType := header.Header.Get("Content-Type")
	if !m.IsAllowedType(contentType) {
		return nil, fmt.Errorf("content type %q is not allowed", contentType)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	fileID := uuid.New().String()
	storagePath := "mock/" + fileID + filepath.Ext(header.Filename)

	att := &domain.Attachment{
		ID:          fileID,
		FileName:    header.Filename,
		ContentType: contentType,
		FileSize:    header.Size,
		StoragePath: storagePath,
		CreatedAt:   time.Now(),
	}

	m.files[storagePath] = att
	m.fileData[storagePath] = data

	return att, nil
}

// SaveBase64Image saves a base64 data URL image in memory.
func (m *MockStorageService) SaveBase64Image(ctx context.Context, dataURL string, filenamePrefix string) (*domain.Attachment, error) {
	contentType, data, err := parseBase64DataURL(dataURL)
	if err != nil {
		return nil, err
	}

	fileID := uuid.New().String()
	ext := extensionFromContentType(contentType)
	if ext == "" {
		ext = ".png"
	}
	storagePath := "mock/" + fileID + ext
	filename := fmt.Sprintf("%s_%s%s", filenamePrefix, fileID[:8], ext)

	att := &domain.Attachment{
		ID:          fileID,
		FileName:    filename,
		ContentType: contentType,
		FileSize:    int64(len(data)),
		StoragePath: storagePath,
		CreatedAt:   time.Now(),
	}

	m.files[storagePath] = att
	m.fileData[storagePath] = data

	return att, nil
}

// GetPath returns a mock path.
func (m *MockStorageService) GetPath(storagePath string) string {
	return "/mock/" + storagePath
}

// GetFileURL returns a mock URL.
func (m *MockStorageService) GetFileURL(storagePath string) string {
	return "/api/v1/uploads/" + storagePath
}

// Delete removes a file from the mock storage.
func (m *MockStorageService) Delete(ctx context.Context, storagePath string) error {
	delete(m.files, storagePath)
	delete(m.fileData, storagePath)
	return nil
}

// ReadAsBase64DataURI returns the file as a base64 data URI.
func (m *MockStorageService) ReadAsBase64DataURI(ctx context.Context, storagePath string) (string, error) {
	data, ok := m.fileData[storagePath]
	if !ok {
		return "", fmt.Errorf("file not found: %s", storagePath)
	}

	att := m.files[storagePath]
	encoded := base64.StdEncoding.EncodeToString(data)

	return fmt.Sprintf("data:%s;base64,%s", att.ContentType, encoded), nil
}

// IsAllowedType checks if a content type is allowed.
func (m *MockStorageService) IsAllowedType(contentType string) bool {
	for _, allowed := range m.allowedTypes {
		if contentType == allowed {
			return true
		}
	}
	return false
}

// GetMaxFileSize returns the maximum file size.
func (m *MockStorageService) GetMaxFileSize() int64 {
	return m.maxFileSize
}

// SetFile adds a file directly to the mock (for test setup).
func (m *MockStorageService) SetFile(storagePath string, att *domain.Attachment, data []byte) {
	m.files[storagePath] = att
	m.fileData[storagePath] = data
}
