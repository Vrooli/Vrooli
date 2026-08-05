// Package census provides host-independent filesystem and device fixtures for
// census tests. It intentionally has no helpers that create or remove host
// paths.
package census

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing/fstest"

	internal "storage-manager/internal/census"
)

type Tree struct {
	FS       fstest.MapFS
	HostRoot string
}

func NewTree(hostRoot string, files map[string][]byte) Tree {
	entries := make(fstest.MapFS, len(files))
	for path, data := range files {
		entries[filepath.ToSlash(path)] = &fstest.MapFile{Data: data}
	}
	return Tree{FS: entries, HostRoot: hostRoot}
}

func (t Tree) WalkDir(root string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(t.FS, t.virtual(root), func(path string, entry fs.DirEntry, err error) error {
		if path == "root" {
			path = t.HostRoot
		} else {
			path = filepath.Join(t.HostRoot, strings.TrimPrefix(path, "root/"))
		}
		return fn(path, entry, err)
	})
}

func (t Tree) Stat(path string) (os.FileInfo, error)  { return fs.Stat(t.FS, t.virtual(path)) }
func (t Tree) Lstat(path string) (os.FileInfo, error) { return fs.Stat(t.FS, t.virtual(path)) }

func (t Tree) virtual(path string) string {
	if path == t.HostRoot {
		return "root"
	}
	rel, err := filepath.Rel(t.HostRoot, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return filepath.ToSlash(filepath.Join("root", rel))
}

var _ internal.FileSystem = Tree{}

type DeviceProbe struct {
	TotalBytes     int64
	AvailableBytes int64
	Privilege      string
	Err            error
}

func (p DeviceProbe) Probe(string) (internal.DeviceInfo, error) {
	return internal.DeviceInfo{TotalBytes: p.TotalBytes, AvailableBytes: p.AvailableBytes, Privilege: p.Privilege}, p.Err
}

var _ internal.DeviceProbe = DeviceProbe{}
