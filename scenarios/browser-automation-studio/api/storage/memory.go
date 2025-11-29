package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

// MemoryStorage is an in-memory implementation of StorageInterface for testing.
// It stores screenshots in memory and provides a simple way to verify storage operations.
type MemoryStorage struct {
	mu         sync.RWMutex
	objects    map[string]memoryObject
	bucketName string
}

type memoryObject struct {
	data        []byte
	contentType string
	storedAt    time.Time
}

// NewMemoryStorage creates a new in-memory storage instance for testing.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		objects:    make(map[string]memoryObject),
		bucketName: "test-bucket",
	}
}

// GetScreenshot retrieves a screenshot from memory storage.
func (m *MemoryStorage) GetScreenshot(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	obj, exists := m.objects[objectName]
	if !exists {
		return nil, nil, fmt.Errorf("object not found: %s", objectName)
	}

	info := &minio.ObjectInfo{
		Key:          objectName,
		Size:         int64(len(obj.data)),
		ContentType:  obj.contentType,
		LastModified: obj.storedAt,
	}

	return io.NopCloser(bytes.NewReader(obj.data)), info, nil
}

// StoreScreenshot stores a screenshot in memory.
func (m *MemoryStorage) StoreScreenshot(ctx context.Context, executionID uuid.UUID, stepName string, data []byte, contentType string) (*ScreenshotInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	objectName := fmt.Sprintf("%s/%s.png", executionID.String(), stepName)

	m.objects[objectName] = memoryObject{
		data:        append([]byte{}, data...), // Copy data
		contentType: contentType,
		storedAt:    time.Now(),
	}

	return &ScreenshotInfo{
		URL:          fmt.Sprintf("memory://%s/%s", m.bucketName, objectName),
		ThumbnailURL: fmt.Sprintf("memory://%s/thumb/%s", m.bucketName, objectName),
		SizeBytes:    int64(len(data)),
		Width:        1920, // Default test values
		Height:       1080,
	}, nil
}

// StoreScreenshotFromFile reads a file and stores it in memory.
func (m *MemoryStorage) StoreScreenshotFromFile(ctx context.Context, executionID uuid.UUID, stepName string, filePath string) (*ScreenshotInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	contentType := "image/png"
	if ext := filepath.Ext(filePath); ext != "" {
		switch strings.ToLower(ext) {
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".gif":
			contentType = "image/gif"
		}
	}

	return m.StoreScreenshot(ctx, executionID, stepName, data, contentType)
}

// DeleteScreenshot removes a screenshot from memory.
func (m *MemoryStorage) DeleteScreenshot(ctx context.Context, objectName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.objects, objectName)
	return nil
}

// ListExecutionScreenshots lists all screenshots for an execution.
func (m *MemoryStorage) ListExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	prefix := executionID.String() + "/"
	var result []string

	for key := range m.objects {
		if strings.HasPrefix(key, prefix) {
			result = append(result, key)
		}
	}

	return result, nil
}

// HealthCheck always returns nil (healthy) for memory storage.
func (m *MemoryStorage) HealthCheck(ctx context.Context) error {
	return nil
}

// GetBucketName returns the configured bucket name.
func (m *MemoryStorage) GetBucketName() string {
	return m.bucketName
}

// --- Test helper methods ---

// GetStoredData returns the raw data for a stored object (for test assertions).
func (m *MemoryStorage) GetStoredData(objectName string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	obj, exists := m.objects[objectName]
	if !exists {
		return nil, false
	}
	return obj.data, true
}

// ObjectCount returns the number of stored objects (for test assertions).
func (m *MemoryStorage) ObjectCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.objects)
}

// Clear removes all stored objects (for test cleanup).
func (m *MemoryStorage) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects = make(map[string]memoryObject)
}

// Compile-time interface enforcement
var _ StorageInterface = (*MemoryStorage)(nil)
