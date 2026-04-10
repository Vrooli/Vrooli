package main

import (
	"os"
	"time"
)

// FakeWriteCall records a WriteFile call for assertions.
type FakeWriteCall struct {
	Path string
	Data string
	Perm os.FileMode
}

// FakeFileIO implements FileIO for tests without touching the real filesystem.
type FakeFileIO struct {
	Files    map[string]string
	StatErr  map[string]error
	ReadErr  error
	WriteErr error
	Writes   []FakeWriteCall
}

// NewFakeFileIO creates a FakeFileIO with empty state.
func NewFakeFileIO() *FakeFileIO {
	return &FakeFileIO{
		Files:   make(map[string]string),
		StatErr: make(map[string]error),
	}
}

// WithFile adds a file with the given content.
func (f *FakeFileIO) WithFile(path, content string) *FakeFileIO {
	f.Files[path] = content
	return f
}

// WithStatErr sets a custom error for Stat on a specific path.
func (f *FakeFileIO) WithStatErr(path string, err error) *FakeFileIO {
	f.StatErr[path] = err
	return f
}

// WithReadErr sets a global read error.
func (f *FakeFileIO) WithReadErr(err error) *FakeFileIO {
	f.ReadErr = err
	return f
}

// WithWriteErr sets a global write error.
func (f *FakeFileIO) WithWriteErr(err error) *FakeFileIO {
	f.WriteErr = err
	return f
}

func (f *FakeFileIO) ReadFile(path string) ([]byte, error) {
	if f.ReadErr != nil {
		return nil, f.ReadErr
	}
	content, ok := f.Files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return []byte(content), nil
}

func (f *FakeFileIO) WriteFile(path string, data []byte, perm os.FileMode) error {
	if f.WriteErr != nil {
		return f.WriteErr
	}
	f.Writes = append(f.Writes, FakeWriteCall{Path: path, Data: string(data), Perm: perm})
	f.Files[path] = string(data)
	return nil
}

func (f *FakeFileIO) Stat(path string) (os.FileInfo, error) {
	if err, ok := f.StatErr[path]; ok {
		return nil, err
	}
	content, ok := f.Files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return &FakeFileInfo{name: path, size: int64(len(content))}, nil
}

func (f *FakeFileIO) MkdirAll(_ string, _ os.FileMode) error {
	if f.WriteErr != nil {
		return f.WriteErr
	}
	return nil
}

// FakeFileInfo implements os.FileInfo for use by FakeFileIO.Stat.
type FakeFileInfo struct {
	name string
	size int64
}

func (fi *FakeFileInfo) Name() string       { return fi.name }
func (fi *FakeFileInfo) Size() int64        { return fi.size }
func (fi *FakeFileInfo) Mode() os.FileMode  { return 0o644 }
func (fi *FakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fi *FakeFileInfo) IsDir() bool        { return false }
func (fi *FakeFileInfo) Sys() interface{}   { return nil }
