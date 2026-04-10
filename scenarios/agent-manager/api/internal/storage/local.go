package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
)

const (
	defaultBasePath    = "/tmp/agent-manager-uploads"
	defaultMaxFileSize = 20 * 1024 * 1024 // 20MB
	maxFileNameLen     = 255
	sniffSize          = 512
)

// defaultAllowedTypes is the set of MIME types permitted by default.
var defaultAllowedTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// Option configures a LocalService.
type Option func(*LocalService)

// WithMaxFileSize sets the maximum upload size in bytes.
func WithMaxFileSize(n int64) Option {
	return func(s *LocalService) { s.maxFileSize = n }
}

// WithAllowedTypes replaces the default allowed MIME type map.
// Keys are MIME types, values are file extensions (including leading dot).
func WithAllowedTypes(types map[string]string) Option {
	return func(s *LocalService) { s.allowedTypes = types }
}

// LocalService stores uploaded files on the local filesystem.
type LocalService struct {
	basePath     string
	maxFileSize  int64
	allowedTypes map[string]string // mime-type → extension

	mu    sync.RWMutex
	index map[string]*AttachmentMeta
}

// NewLocalService creates a new LocalService rooted at basePath.
// If basePath is empty the default path is used.
func NewLocalService(basePath string, opts ...Option) *LocalService {
	if basePath == "" {
		basePath = defaultBasePath
	}
	s := &LocalService{
		basePath:     basePath,
		maxFileSize:  defaultMaxFileSize,
		allowedTypes: defaultAllowedTypes,
		index:        make(map[string]*AttachmentMeta),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Upload stores a file and returns its metadata.
func (s *LocalService) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*AttachmentMeta, error) {
	// Validate size from header.
	if header.Size > s.maxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum %d bytes", header.Size, s.maxFileSize)
	}

	// Read the first 512 bytes for content-type detection.
	buf := make([]byte, sniffSize)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("reading file header: %w", err)
	}
	buf = buf[:n]

	detectedType := http.DetectContentType(buf)
	if !s.IsAllowedType(detectedType) {
		return nil, fmt.Errorf("content type %q is not allowed", detectedType)
	}

	// Seek back to the beginning so we can write the full file.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seeking file: %w", err)
	}

	ext := s.allowedTypes[detectedType]
	id := uuid.New().String()
	now := time.Now().UTC()

	// Build date-organized relative path.
	relDir := filepath.Join(
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		fmt.Sprintf("%02d", now.Day()),
	)
	relPath := filepath.Join(relDir, id+ext)
	absDir := filepath.Join(s.basePath, relDir)

	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating storage directory: %w", err)
	}

	absPath := filepath.Join(s.basePath, relPath)
	out, err := os.Create(absPath)
	if err != nil {
		return nil, fmt.Errorf("creating file: %w", err)
	}
	defer out.Close()

	written, err := io.Copy(out, file)
	if err != nil {
		os.Remove(absPath)
		return nil, fmt.Errorf("writing file: %w", err)
	}

	meta := &AttachmentMeta{
		ID:          id,
		FileName:    sanitizeFileName(header.Filename),
		ContentType: detectedType,
		FileSize:    written + int64(n), // sniffed bytes were seeked back, but total is header.Size
		StoragePath: relPath,
		CreatedAt:   now,
	}
	// Use the actual bytes written (io.Copy wrote the whole file after seek).
	meta.FileSize = written

	s.mu.Lock()
	s.index[id] = meta
	s.mu.Unlock()

	return meta, nil
}

// Get returns metadata for an attachment by ID.
func (s *LocalService) Get(_ context.Context, id string) (*AttachmentMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.index[id]
	if !ok {
		return nil, fmt.Errorf("attachment %q not found", id)
	}
	return m, nil
}

// GetMultiple returns metadata for multiple attachments by IDs.
// Unknown IDs are silently skipped.
func (s *LocalService) GetMultiple(_ context.Context, ids []string) ([]*AttachmentMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*AttachmentMeta
	for _, id := range ids {
		if m, ok := s.index[id]; ok {
			result = append(result, m)
		}
	}
	return result, nil
}

// GetFilePath returns the absolute filesystem path for a stored file.
func (s *LocalService) GetFilePath(storagePath string) string {
	return filepath.Join(s.basePath, storagePath)
}

// GetServingURL returns the URL path for HTTP serving.
func (s *LocalService) GetServingURL(storagePath string) string {
	return "/api/v1/uploads/" + storagePath
}

// Delete removes a file from storage and its index entry.
func (s *LocalService) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	m, ok := s.index[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("attachment %q not found", id)
	}
	delete(s.index, id)
	s.mu.Unlock()

	absPath := filepath.Join(s.basePath, m.StoragePath)
	if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing file: %w", err)
	}
	return nil
}

// IsAllowedType checks if a content type is allowed for upload.
func (s *LocalService) IsAllowedType(contentType string) bool {
	_, ok := s.allowedTypes[contentType]
	return ok
}

// MaxFileSize returns the maximum allowed file size in bytes.
func (s *LocalService) MaxFileSize() int64 {
	return s.maxFileSize
}

// sanitizeFileName cleans a user-provided filename to prevent path traversal
// and other issues.
func sanitizeFileName(name string) string {
	// Strip any directory components (handle both Unix and Windows separators).
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)

	// Replace path separators and null bytes.
	name = strings.Map(func(r rune) rune {
		switch {
		case r == '/' || r == '\\' || r == '\x00':
			return '_'
		case !unicode.IsPrint(r):
			return '_'
		}
		return r
	}, name)

	// Truncate to max length.
	if len(name) > maxFileNameLen {
		ext := filepath.Ext(name)
		base := name[:maxFileNameLen-len(ext)]
		name = base + ext
	}

	if name == "" || name == "." || name == ".." {
		name = "unnamed"
	}

	return name
}
