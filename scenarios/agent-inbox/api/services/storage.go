// Package services provides business logic orchestration.
// This file implements file storage for attachments.
package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-inbox/config"
	"agent-inbox/domain"

	"github.com/google/uuid"
)

// StorageService defines the interface for file storage operations.
// This abstraction enables testing with mock storage and future
// migration to cloud storage (S3, GCS) without changing callers.
type StorageService interface {
	// Upload stores a file and returns its metadata.
	// The file is stored with a generated UUID name in a date-organized directory.
	Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*domain.Attachment, error)

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

// Upload stores a file and returns its metadata.
func (s *LocalStorageService) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*domain.Attachment, error) {
	// Validate file size
	if header.Size > s.cfg.MaxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum %d bytes", header.Size, s.cfg.MaxFileSize)
	}

	// Validate content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = detectContentType(header.Filename)
	}
	if !s.IsAllowedType(contentType) {
		return nil, fmt.Errorf("content type %q is not allowed", contentType)
	}

	// Generate storage path: YYYY/MM/DD/{uuid}.{ext}
	now := time.Now()
	dateDir := now.Format("2006/01/02")
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = extensionFromContentType(contentType)
	}
	fileID := uuid.New().String()
	storagePath := filepath.Join(dateDir, fileID+ext)
	fullPath := filepath.Join(s.cfg.BasePath, storagePath)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %s: %w", fullPath, err)
	}
	defer dst.Close()

	// Copy content
	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(fullPath) // Clean up on failure
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Get image dimensions if applicable
	width, height := 0, 0
	if strings.HasPrefix(contentType, "image/") {
		width, height = getImageDimensions(fullPath)
	}

	return &domain.Attachment{
		ID:          fileID,
		FileName:    header.Filename,
		ContentType: contentType,
		FileSize:    header.Size,
		StoragePath: storagePath,
		Width:       width,
		Height:      height,
		CreatedAt:   now,
	}, nil
}

// GetPath returns the full filesystem path for a stored file.
func (s *LocalStorageService) GetPath(storagePath string) string {
	return filepath.Join(s.cfg.BasePath, storagePath)
}

// GetFileURL returns the URL for accessing a stored file.
func (s *LocalStorageService) GetFileURL(storagePath string) string {
	return s.cfg.BaseURL + "/" + storagePath
}

// Delete removes a file from storage.
func (s *LocalStorageService) Delete(ctx context.Context, storagePath string) error {
	fullPath := s.GetPath(storagePath)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file %s: %w", fullPath, err)
	}
	return nil
}

// ReadAsBase64DataURI reads a file and returns it as a base64 data URI.
// Format: data:{contentType};base64,{base64data}
func (s *LocalStorageService) ReadAsBase64DataURI(ctx context.Context, storagePath string) (string, error) {
	fullPath := s.GetPath(storagePath)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}

	contentType := detectContentType(storagePath)
	encoded := base64.StdEncoding.EncodeToString(data)

	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded), nil
}

// IsAllowedType checks if a content type is allowed for upload.
func (s *LocalStorageService) IsAllowedType(contentType string) bool {
	for _, allowed := range s.cfg.AllowedImageTypes {
		if contentType == allowed {
			return true
		}
	}
	for _, allowed := range s.cfg.AllowedDocumentTypes {
		if contentType == allowed {
			return true
		}
	}
	return false
}

// GetMaxFileSize returns the maximum allowed file size in bytes.
func (s *LocalStorageService) GetMaxFileSize() int64 {
	return s.cfg.MaxFileSize
}

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
