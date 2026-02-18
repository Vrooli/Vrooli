package services

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ConfigStore abstracts configuration file I/O for testability.
type ConfigStore interface {
	ReadConfig(name string) ([]byte, error)
	WriteConfig(name string, data []byte) error
}

// FileConfigStore reads and writes configuration files on disk.
type FileConfigStore struct {
	basePath string
}

// ReadConfig reads a configuration file from the base path.
func (f *FileConfigStore) ReadConfig(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(f.basePath, name))
}

// WriteConfig writes a configuration file to the base path, creating
// directories as needed.
func (f *FileConfigStore) WriteConfig(name string, data []byte) error {
	fullPath := filepath.Join(f.basePath, name)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	return os.WriteFile(fullPath, data, 0o644)
}

// MemoryConfigStore is an in-memory ConfigStore for tests.
type MemoryConfigStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewMemoryConfigStore creates a MemoryConfigStore.
func NewMemoryConfigStore() *MemoryConfigStore {
	return &MemoryConfigStore{data: make(map[string][]byte)}
}

// ReadConfig returns the stored bytes for name, or an error if not found.
func (m *MemoryConfigStore) ReadConfig(name string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.data[name]
	if !ok {
		return nil, fmt.Errorf("config not found: %s", name)
	}
	// Return a copy to avoid mutation.
	out := make([]byte, len(b))
	copy(out, b)
	return out, nil
}

// WriteConfig stores data under name.
func (m *MemoryConfigStore) WriteConfig(name string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	stored := make([]byte, len(data))
	copy(stored, data)
	m.data[name] = stored
	return nil
}
