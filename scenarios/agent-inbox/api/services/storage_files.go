// Package services provides business logic orchestration.
// This file implements file operations for the storage service.
package services

import (
	"agent-inbox/domain"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

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

// SaveBase64Image saves a base64 data URL image and returns attachment metadata.
// Used for storing AI-generated images from model responses.
func (s *LocalStorageService) SaveBase64Image(ctx context.Context, dataURL string, filenamePrefix string) (*domain.Attachment, error) {
	// Parse the data URL: data:image/png;base64,{base64data}
	contentType, data, err := parseBase64DataURL(dataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse data URL: %w", err)
	}

	// Generate storage path: YYYY/MM/DD/{uuid}.{ext}
	now := time.Now()
	dateDir := now.Format("2006/01/02")
	ext := extensionFromContentType(contentType)
	if ext == "" {
		ext = ".png" // Default to PNG for generated images
	}
	fileID := uuid.New().String()
	storagePath := filepath.Join(dateDir, fileID+ext)
	fullPath := filepath.Join(s.cfg.BasePath, storagePath)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write the image data
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return nil, fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	// Get image dimensions if applicable
	width, height := 0, 0
	if strings.HasPrefix(contentType, "image/") {
		width, height = getImageDimensions(fullPath)
	}

	// Generate a human-readable filename
	filename := fmt.Sprintf("%s_%s%s", filenamePrefix, fileID[:8], ext)

	return &domain.Attachment{
		ID:          fileID,
		FileName:    filename,
		ContentType: contentType,
		FileSize:    int64(len(data)),
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
