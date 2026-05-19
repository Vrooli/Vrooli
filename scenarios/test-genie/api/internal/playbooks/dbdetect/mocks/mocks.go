package mocks

import (
	"errors"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	"test-genie/internal/playbooks/dbdetect"
)

// FakeFilesystem is an in-memory dbdetect.Filesystem for tests.
type FakeFilesystem struct {
	Files map[string][]byte
	// ReadErr, if non-nil, is returned from ReadFile for the matching path.
	ReadErr map[string]error
	// WalkErr, if non-nil, is returned from Walk regardless of root.
	WalkErr error
}

var _ dbdetect.Filesystem = (*FakeFilesystem)(nil)

func (f *FakeFilesystem) ReadFile(p string) ([]byte, error) {
	if f.ReadErr != nil {
		if err, ok := f.ReadErr[p]; ok {
			return nil, err
		}
	}
	if data, ok := f.Files[p]; ok {
		return data, nil
	}
	return nil, errors.New("not found: " + p)
}

func (f *FakeFilesystem) Walk(root string, fn fs.WalkDirFunc) error {
	if f.WalkErr != nil {
		return f.WalkErr
	}
	paths := make([]string, 0, len(f.Files))
	for p := range f.Files {
		if p == root || strings.HasPrefix(p, root+"/") {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	dirs := map[string]bool{}
	for _, p := range paths {
		for d := path.Dir(p); d != "." && d != "/" && d != root && !dirs[d]; d = path.Dir(d) {
			dirs[d] = true
		}
	}
	if !dirs[root] {
		dirs[root] = true
	}
	emitted := map[string]bool{}
	var entries []string
	for d := range dirs {
		entries = append(entries, d)
	}
	entries = append(entries, paths...)
	sort.Strings(entries)
	for _, p := range entries {
		if emitted[p] {
			continue
		}
		emitted[p] = true
		isDir := dirs[p]
		if err := fn(p, fakeDirEntry{name: path.Base(p), isDir: isDir}, nil); err != nil {
			if errors.Is(err, fs.SkipDir) {
				continue
			}
			return err
		}
	}
	return nil
}

type fakeDirEntry struct {
	name  string
	isDir bool
}

func (e fakeDirEntry) Name() string      { return e.name }
func (e fakeDirEntry) IsDir() bool       { return e.isDir }
func (e fakeDirEntry) Type() fs.FileMode { return 0 }
func (e fakeDirEntry) Info() (fs.FileInfo, error) {
	return fakeFileInfo{name: e.name, isDir: e.isDir}, nil
}

type fakeFileInfo struct {
	name  string
	isDir bool
}

func (i fakeFileInfo) Name() string       { return i.name }
func (i fakeFileInfo) Size() int64        { return 0 }
func (i fakeFileInfo) Mode() fs.FileMode  { return 0 }
func (i fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (i fakeFileInfo) IsDir() bool        { return i.isDir }
func (i fakeFileInfo) Sys() any           { return nil }

// FakeManifest is an in-memory dbdetect.Manifest for tests.
type FakeManifest struct {
	ResourcesList []dbdetect.ManifestResource
	SQLiteVars    []string
}

var _ dbdetect.Manifest = (*FakeManifest)(nil)

func (m *FakeManifest) Resources() []dbdetect.ManifestResource {
	return m.ResourcesList
}

func (m *FakeManifest) SQLitePathEnvVars() []string {
	return m.SQLiteVars
}
