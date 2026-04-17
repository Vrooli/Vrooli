// Package persistence provides data access for session profile management.
package persistence

import (
	"errors"
	"io/fs"
	"path/filepath"
	"sync"
	"time"
)

// MockFileSystem implements FileSystem for testing.
// It stores files in memory and is safe for concurrent use.
type MockFileSystem struct {
	mu    sync.RWMutex
	files map[string][]byte
	dirs  map[string]bool

	// Error injection for testing error paths
	ReadFileErr  error
	WriteFileErr error
	RemoveErr    error
	RenameErr    error
	ReadDirErr   error
	MkdirAllErr  error
	StatErr      error
}

// NewMockFileSystem creates a new mock file system for testing.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

// ReadFile reads the contents of a file from memory.
func (m *MockFileSystem) ReadFile(name string) ([]byte, error) {
	if m.ReadFileErr != nil {
		return nil, m.ReadFileErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	// Return a copy to prevent mutation
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// WriteFile writes data to a file in memory.
func (m *MockFileSystem) WriteFile(name string, data []byte, perm fs.FileMode) error {
	if m.WriteFileErr != nil {
		return m.WriteFileErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// Store a copy to prevent mutation
	stored := make([]byte, len(data))
	copy(stored, data)
	m.files[name] = stored
	return nil
}

// Remove removes a file from memory.
func (m *MockFileSystem) Remove(name string) error {
	if m.RemoveErr != nil {
		return m.RemoveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.files[name]; !ok {
		return fs.ErrNotExist
	}
	delete(m.files, name)
	return nil
}

// Rename moves a file in memory.
func (m *MockFileSystem) Rename(oldpath, newpath string) error {
	if m.RenameErr != nil {
		return m.RenameErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.files[oldpath]
	if !ok {
		return fs.ErrNotExist
	}
	m.files[newpath] = data
	delete(m.files, oldpath)
	return nil
}

// ReadDir returns entries for a directory from memory.
func (m *MockFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	if m.ReadDirErr != nil {
		return nil, m.ReadDirErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect files in this directory
	prefix := name
	if prefix != "" && prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}

	var entries []fs.DirEntry
	seen := make(map[string]bool)

	for path := range m.files {
		if len(path) > len(prefix) && path[:len(prefix)] == prefix {
			// Get the relative path after the prefix
			rel := path[len(prefix):]
			// Get just the first component (file name or subdir)
			if idx := indexOf(rel, '/'); idx >= 0 {
				// This is a subdirectory
				dirName := rel[:idx]
				if !seen[dirName] {
					seen[dirName] = true
					entries = append(entries, &mockDirEntry{name: dirName, isDir: true})
				}
			} else {
				// This is a file
				entries = append(entries, &mockDirEntry{name: rel, isDir: false})
			}
		}
	}

	return entries, nil
}

// MkdirAll marks a directory as existing.
func (m *MockFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	if m.MkdirAllErr != nil {
		return m.MkdirAllErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dirs[path] = true
	return nil
}

// Stat returns file info for a path.
func (m *MockFileSystem) Stat(name string) (fs.FileInfo, error) {
	if m.StatErr != nil {
		return nil, m.StatErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if data, ok := m.files[name]; ok {
		return &mockFileInfo{name: filepath.Base(name), size: int64(len(data)), isDir: false}, nil
	}
	if m.dirs[name] {
		return &mockFileInfo{name: filepath.Base(name), size: 0, isDir: true}, nil
	}
	return nil, fs.ErrNotExist
}

// SetFile sets a file's contents directly (test helper).
func (m *MockFileSystem) SetFile(name string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stored := make([]byte, len(data))
	copy(stored, data)
	m.files[name] = stored
}

// GetFile returns a file's contents (test helper).
func (m *MockFileSystem) GetFile(name string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.files[name]
	if !ok {
		return nil, false
	}
	result := make([]byte, len(data))
	copy(result, data)
	return result, true
}

// FileExists checks if a file exists (test helper).
func (m *MockFileSystem) FileExists(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.files[name]
	return ok
}

// ListFiles returns all file paths (test helper).
func (m *MockFileSystem) ListFiles() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]string, 0, len(m.files))
	for path := range m.files {
		result = append(result, path)
	}
	return result
}

// Clear removes all files (test helper).
func (m *MockFileSystem) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files = make(map[string][]byte)
	m.dirs = make(map[string]bool)
}

// Reset clears all data and errors (test helper).
func (m *MockFileSystem) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files = make(map[string][]byte)
	m.dirs = make(map[string]bool)
	m.ReadFileErr = nil
	m.WriteFileErr = nil
	m.RemoveErr = nil
	m.RenameErr = nil
	m.ReadDirErr = nil
	m.MkdirAllErr = nil
	m.StatErr = nil
}

// indexOf returns the index of the first occurrence of substr in s, or -1 if not found.
func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// mockDirEntry implements fs.DirEntry for testing.
type mockDirEntry struct {
	name  string
	isDir bool
}

func (e *mockDirEntry) Name() string               { return e.name }
func (e *mockDirEntry) IsDir() bool                { return e.isDir }
func (e *mockDirEntry) Type() fs.FileMode          { return 0 }
func (e *mockDirEntry) Info() (fs.FileInfo, error) { return nil, errors.New("not implemented") }

// mockFileInfo implements fs.FileInfo for testing.
type mockFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (i *mockFileInfo) Name() string       { return i.name }
func (i *mockFileInfo) Size() int64        { return i.size }
func (i *mockFileInfo) Mode() fs.FileMode  { return 0o644 }
func (i *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (i *mockFileInfo) IsDir() bool        { return i.isDir }
func (i *mockFileInfo) Sys() interface{}   { return nil }

// Ensure MockFileSystem implements FileSystem at compile time.
var _ FileSystem = (*MockFileSystem)(nil)
