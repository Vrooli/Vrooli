package userconfig

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeConfigIO struct {
	statErr        error
	readErrs       map[string]error
	writeErrs      map[string]error
	renameErr      error
	homeDir        string
	homeDirErr     error
	files          map[string][]byte
	readCalls      int
	removeCalls    int
	lastRemoved    string
	lastWritePath  string
	lastRenameFrom string
	lastRenameTo   string
}

func newFakeConfigIO() *fakeConfigIO {
	return &fakeConfigIO{
		files:     make(map[string][]byte),
		readErrs:  make(map[string]error),
		writeErrs: make(map[string]error),
	}
}

func (f *fakeConfigIO) Stat(name string) (os.FileInfo, error) {
	if f.statErr != nil {
		return nil, f.statErr
	}
	if _, ok := f.files[name]; ok {
		return fakeFileInfo{}, nil
	}
	return nil, fs.ErrNotExist
}

func (f *fakeConfigIO) ReadFile(name string) ([]byte, error) {
	f.readCalls++
	if err, ok := f.readErrs[name]; ok {
		return nil, err
	}
	data, ok := f.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return data, nil
}

func (f *fakeConfigIO) MkdirAll(path string, perm os.FileMode) error { return nil }

func (f *fakeConfigIO) WriteFile(name string, data []byte, perm os.FileMode) error {
	f.lastWritePath = name
	if err, ok := f.writeErrs[name]; ok {
		return err
	}
	f.files[name] = append([]byte(nil), data...)
	return nil
}

func (f *fakeConfigIO) Rename(oldpath, newpath string) error {
	f.lastRenameFrom = oldpath
	f.lastRenameTo = newpath
	if f.renameErr != nil {
		return f.renameErr
	}
	if data, ok := f.files[oldpath]; ok {
		f.files[newpath] = data
		delete(f.files, oldpath)
	}
	return nil
}

func (f *fakeConfigIO) Remove(name string) error {
	f.removeCalls++
	f.lastRemoved = name
	delete(f.files, name)
	return nil
}

func (f *fakeConfigIO) UserHomeDir() (string, error) {
	if f.homeDirErr != nil {
		return "", f.homeDirErr
	}
	return f.homeDir, nil
}

type fakeFileInfo struct{}

func (fakeFileInfo) Name() string       { return "fake" }
func (fakeFileInfo) Size() int64        { return 1 }
func (fakeFileInfo) Mode() os.FileMode  { return 0o644 }
func (fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fakeFileInfo) IsDir() bool        { return false }
func (fakeFileInfo) Sys() any           { return nil }

func TestManagerLoadSeamMissingConfigUsesDefaults(t *testing.T) {
	io := newFakeConfigIO()
	io.statErr = fs.ErrNotExist

	m := newManagerWithIO("config.json", "schema.json", io)
	if err := m.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if io.readCalls != 0 {
		t.Fatalf("expected no read calls for missing config, got %d", io.readCalls)
	}
	if got := m.Get().Version; got != "1.0" {
		t.Fatalf("expected defaults to remain loaded, got version %q", got)
	}
}

func TestManagerSaveSeamRenameFailureCleansTemp(t *testing.T) {
	io := newFakeConfigIO()
	io.renameErr = errors.New("rename denied")
	configPath := filepath.Join("tmp", "config.json")

	m := newManagerWithIO(configPath, "schema.json", io)
	err := m.Save()
	if err == nil {
		t.Fatal("expected Save to fail when rename fails")
	}

	expectedTemp := configPath + ".tmp"
	if io.lastWritePath != expectedTemp {
		t.Fatalf("temp write path = %q, want %q", io.lastWritePath, expectedTemp)
	}
	if io.lastRemoved != expectedTemp {
		t.Fatalf("cleanup path = %q, want %q", io.lastRemoved, expectedTemp)
	}
	if io.removeCalls != 1 {
		t.Fatalf("remove calls = %d, want 1", io.removeCalls)
	}
}

func TestDefaultConfigPathSeamFallbackOnHomeDirError(t *testing.T) {
	io := newFakeConfigIO()
	io.homeDirErr = errors.New("home unavailable")

	got := defaultConfigPathWithIO(io)
	want := filepath.Join(".", ".vrooli-autoheal", "config.json")
	if got != want {
		t.Fatalf("default config path = %q, want %q", got, want)
	}
}
