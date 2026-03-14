package storage

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"sync"
)

// MockService is an in-memory mock that implements Service for testing.
type MockService struct {
	mu    sync.RWMutex
	files map[string]*AttachmentMeta

	maxFileSize  int64
	allowedTypes map[string]string

	// Call tracking for assertions.
	UploadCalls      int
	GetCalls         int
	GetMultipleCalls int
	GetFilePathCalls int
	GetServingCalls  int
	DeleteCalls      int

	// UploadErr, if non-nil, is returned by Upload.
	UploadErr error
}

// NewMockService creates a MockService with default settings.
func NewMockService() *MockService {
	return &MockService{
		files:        make(map[string]*AttachmentMeta),
		maxFileSize:  defaultMaxFileSize,
		allowedTypes: defaultAllowedTypes,
	}
}

// SetFile adds or replaces an attachment in the mock store (for test setup).
func (m *MockService) SetFile(id string, meta *AttachmentMeta) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[id] = meta
}

// Upload implements Service.
func (m *MockService) Upload(_ context.Context, _ multipart.File, header *multipart.FileHeader) (*AttachmentMeta, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UploadCalls++
	if m.UploadErr != nil {
		return nil, m.UploadErr
	}
	meta := &AttachmentMeta{
		ID:          fmt.Sprintf("mock-%d", m.UploadCalls),
		FileName:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		FileSize:    header.Size,
		StoragePath: fmt.Sprintf("mock/%s", header.Filename),
	}
	m.files[meta.ID] = meta
	return meta, nil
}

// Get implements Service.
func (m *MockService) Get(_ context.Context, id string) (*AttachmentMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetCalls++
	f, ok := m.files[id]
	if !ok {
		return nil, fmt.Errorf("attachment %q not found", id)
	}
	return f, nil
}

// GetMultiple implements Service.
func (m *MockService) GetMultiple(_ context.Context, ids []string) ([]*AttachmentMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetMultipleCalls++
	var result []*AttachmentMeta
	for _, id := range ids {
		if f, ok := m.files[id]; ok {
			result = append(result, f)
		}
	}
	return result, nil
}

// GetFilePath implements Service.
func (m *MockService) GetFilePath(storagePath string) string {
	m.mu.Lock()
	m.GetFilePathCalls++
	m.mu.Unlock()
	return filepath.Join("/mock-storage", storagePath)
}

// GetServingURL implements Service.
func (m *MockService) GetServingURL(storagePath string) string {
	m.mu.Lock()
	m.GetServingCalls++
	m.mu.Unlock()
	return "/api/v1/uploads/" + storagePath
}

// Delete implements Service.
func (m *MockService) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeleteCalls++
	if _, ok := m.files[id]; !ok {
		return fmt.Errorf("attachment %q not found", id)
	}
	delete(m.files, id)
	return nil
}

// IsAllowedType implements Service.
func (m *MockService) IsAllowedType(contentType string) bool {
	_, ok := m.allowedTypes[contentType]
	return ok
}

// MaxFileSize implements Service.
func (m *MockService) MaxFileSize() int64 {
	return m.maxFileSize
}
