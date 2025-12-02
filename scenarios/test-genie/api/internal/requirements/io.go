package requirements

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Reader abstracts file reading operations for testing.
type Reader interface {
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Stat(path string) (fs.FileInfo, error)
	Exists(path string) bool
}

// Writer abstracts file writing operations for testing.
type Writer interface {
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Remove(path string) error
}

// FileSystem combines Reader and Writer interfaces.
type FileSystem interface {
	Reader
	Writer
}

// osReader implements Reader using the os package.
type osReader struct{}

// NewOSReader creates a Reader that uses the real file system.
func NewOSReader() Reader {
	return &osReader{}
}

func (r *osReader) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (r *osReader) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

func (r *osReader) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

func (r *osReader) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// osWriter implements Writer using the os package.
type osWriter struct{}

// NewOSWriter creates a Writer that uses the real file system.
func NewOSWriter() Writer {
	return &osWriter{}
}

func (w *osWriter) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (w *osWriter) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (w *osWriter) Remove(path string) error {
	return os.Remove(path)
}

// osFileSystem implements FileSystem using the os package.
type osFileSystem struct {
	osReader
	osWriter
}

// NewOSFileSystem creates a FileSystem that uses the real file system.
func NewOSFileSystem() FileSystem {
	return &osFileSystem{}
}

// Walker abstracts directory walking for testing.
type Walker interface {
	Walk(root string, fn filepath.WalkFunc) error
	WalkDir(root string, fn fs.WalkDirFunc) error
}

// osWalker implements Walker using filepath.
type osWalker struct{}

// NewOSWalker creates a Walker that uses the real file system.
func NewOSWalker() Walker {
	return &osWalker{}
}

func (w *osWalker) Walk(root string, fn filepath.WalkFunc) error {
	return filepath.Walk(root, fn)
}

func (w *osWalker) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

// memReader implements Reader with in-memory data for testing.
type memReader struct {
	files map[string][]byte
	dirs  map[string][]fs.DirEntry
}

// NewMemReader creates an in-memory Reader for testing.
func NewMemReader() *memReader {
	return &memReader{
		files: make(map[string][]byte),
		dirs:  make(map[string][]fs.DirEntry),
	}
}

// AddFile adds a file to the in-memory filesystem.
func (r *memReader) AddFile(path string, content []byte) {
	r.files[path] = content
}

// AddDir adds a directory listing to the in-memory filesystem.
func (r *memReader) AddDir(path string, entries []fs.DirEntry) {
	r.dirs[path] = entries
}

func (r *memReader) ReadFile(path string) ([]byte, error) {
	if content, ok := r.files[path]; ok {
		return content, nil
	}
	return nil, os.ErrNotExist
}

func (r *memReader) ReadDir(path string) ([]fs.DirEntry, error) {
	if entries, ok := r.dirs[path]; ok {
		return entries, nil
	}
	return nil, os.ErrNotExist
}

func (r *memReader) Stat(path string) (fs.FileInfo, error) {
	if _, ok := r.files[path]; ok {
		return &memFileInfo{name: filepath.Base(path), isDir: false}, nil
	}
	if _, ok := r.dirs[path]; ok {
		return &memFileInfo{name: filepath.Base(path), isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

func (r *memReader) Exists(path string) bool {
	_, err := r.Stat(path)
	return err == nil
}

// memFileInfo implements fs.FileInfo for testing.
type memFileInfo struct {
	name  string
	isDir bool
	size  int64
}

func (f *memFileInfo) Name() string      { return f.name }
func (f *memFileInfo) Size() int64       { return f.size }
func (f *memFileInfo) Mode() fs.FileMode { return 0644 }
func (f *memFileInfo) ModTime() time.Time { return time.Time{} }
func (f *memFileInfo) IsDir() bool       { return f.isDir }
func (f *memFileInfo) Sys() any          { return nil }

// memWriter implements Writer with in-memory storage for testing.
type memWriter struct {
	files   map[string][]byte
	dirs    map[string]bool
	removed map[string]bool
}

// NewMemWriter creates an in-memory Writer for testing.
func NewMemWriter() *memWriter {
	return &memWriter{
		files:   make(map[string][]byte),
		dirs:    make(map[string]bool),
		removed: make(map[string]bool),
	}
}

func (w *memWriter) WriteFile(path string, data []byte, perm fs.FileMode) error {
	w.files[path] = data
	return nil
}

func (w *memWriter) MkdirAll(path string, perm fs.FileMode) error {
	w.dirs[path] = true
	return nil
}

func (w *memWriter) Remove(path string) error {
	delete(w.files, path)
	w.removed[path] = true
	return nil
}

// GetWrittenFile retrieves a file written during testing.
func (w *memWriter) GetWrittenFile(path string) ([]byte, bool) {
	data, ok := w.files[path]
	return data, ok
}

// WasRemoved checks if a path was removed during testing.
func (w *memWriter) WasRemoved(path string) bool {
	return w.removed[path]
}

// WrittenFiles returns all written file paths.
func (w *memWriter) WrittenFiles() []string {
	paths := make([]string, 0, len(w.files))
	for path := range w.files {
		paths = append(paths, path)
	}
	return paths
}
