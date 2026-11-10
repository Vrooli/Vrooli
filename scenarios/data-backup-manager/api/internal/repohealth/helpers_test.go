package repohealth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"data-backup-manager/internal/engine"
)

// The real incident this package exists for: a USB NTFS repository where some
// leftover temporary files cannot be stat-ed (EIO) while kopia lists everything.
const (
	tempRel1 = "s/f43/64b7506cef44e4fbb3fc66ba7f2d9-sf40366ce03934d79144.f.tmp.8c9d63ca28444838"
	tempRel2 = "s/4e5/7bdb040b1caeb47f4c5ba9c2867c5-sc3c89b33ffab9119144.f.tmp.dde5f267db32227a"
	blobRel  = "q/63/90189f3d3250c965c737fd82cc103e-s42b636fa5d06735f144.f"
	readable = "p/bbb/7098cafebeccc21a63be88c33846b-s9e453c5eb992a889141.f"
	root     = "/repo"
)

var fixedNow = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

type fakeInfo struct {
	name string
	size int64
	mode fs.FileMode
}

func (f fakeInfo) Name() string       { return f.name }
func (f fakeInfo) Size() int64        { return f.size }
func (f fakeInfo) Mode() fs.FileMode  { return f.mode }
func (f fakeInfo) ModTime() time.Time { return fixedNow }
func (f fakeInfo) IsDir() bool        { return f.mode.IsDir() }
func (f fakeInfo) Sys() any           { return nil }

type fakeNode struct {
	dir  bool
	size int64
	err  error
	mode fs.FileMode // overrides regular/dir when set
}

// fakeFS is an in-memory repository tree that can fail like a damaged drive.
type fakeFS struct {
	mu    sync.Mutex
	nodes map[string]fakeNode
	lists int
}

func newFakeFS(rels ...string) *fakeFS {
	f := &fakeFS{nodes: map[string]fakeNode{root: {dir: true}}}
	for _, rel := range rels {
		f.addFile(rel, 100)
	}
	return f
}

func (f *fakeFS) ensureDirs(rel string) {
	for dir := path.Dir(rel); dir != "." && dir != "/"; dir = path.Dir(dir) {
		full := path.Join(root, dir)
		if _, ok := f.nodes[full]; !ok {
			f.nodes[full] = fakeNode{dir: true}
		}
	}
}

func (f *fakeFS) addFile(rel string, size int64) {
	f.ensureDirs(rel)
	f.nodes[path.Join(root, rel)] = fakeNode{size: size}
}

func (f *fakeFS) addUnreadable(rel string, err error) {
	f.ensureDirs(rel)
	f.nodes[path.Join(root, rel)] = fakeNode{err: err}
}

func (f *fakeFS) set(rel string, n fakeNode) { f.nodes[path.Join(root, rel)] = n }

func (f *fakeFS) Names(dir string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lists++
	if n, ok := f.nodes[dir]; !ok || !n.dir {
		return nil, &fs.PathError{Op: "open", Path: dir, Err: fs.ErrNotExist}
	} else if n.err != nil {
		return nil, &fs.PathError{Op: "readdir", Path: dir, Err: n.err}
	}
	var names []string
	for p := range f.nodes {
		if path.Dir(p) == dir && p != dir {
			names = append(names, path.Base(p))
		}
	}
	sort.Strings(names)
	return names, nil
}

func (f *fakeFS) Lstat(p string) (fs.FileInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, ok := f.nodes[p]
	if !ok {
		return nil, &fs.PathError{Op: "lstat", Path: p, Err: fs.ErrNotExist}
	}
	if n.err != nil && !n.dir {
		return nil, &fs.PathError{Op: "lstat", Path: p, Err: n.err}
	}
	mode := fs.FileMode(0o644)
	if n.dir {
		mode = fs.ModeDir | 0o755
	}
	if n.mode != 0 {
		mode = n.mode
	}
	return fakeInfo{name: path.Base(p), size: n.size, mode: mode}, nil
}

var errEIO = syscall.EIO

func kopiaEIOError(rel string) error {
	return fmt.Errorf("kopia blob stats --raw: error listing blobs: error processing directory shards: error reading directory: lstat %s: input/output error", path.Join(root, rel))
}

// fakeEngine is the kopia seam: only the calls repohealth makes are real.
type fakeEngine struct {
	engine.KopiaEngine
	statusErr error
	listErr   error
	statsErr  error
	snaps     []engine.Snapshot
	stats     engine.RepoStats
	statsCall int
}

func (e *fakeEngine) RepoStatus(context.Context, string) (engine.RepoStatus, error) {
	return engine.RepoStatus{Connected: e.statusErr == nil}, e.statusErr
}

func (e *fakeEngine) SnapshotList(context.Context, string, string) ([]engine.Snapshot, error) {
	return e.snaps, e.listErr
}

func (e *fakeEngine) RepoStats(context.Context, string) (engine.RepoStats, error) {
	e.statsCall++
	return e.stats, e.statsErr
}

type memExceptions struct {
	mu   sync.Mutex
	list []Exception
}

func (m *memExceptions) List(_ context.Context, _ string) ([]Exception, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Exception(nil), m.list...), nil
}

func (m *memExceptions) Add(_ context.Context, e Exception) error {
	if err := e.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.list = append(m.list, e)
	return nil
}

func (m *memExceptions) Revoke(context.Context, string, string) error { return errors.New("unused") }

func approved(rel string) Exception {
	return Exception{
		ID: ExceptionID("dest-1", rel), DestinationID: "dest-1", RelPath: rel,
		Class: ClassLeftoverTemp, BlobKind: BlobSessionMarker, Problem: ProblemIO,
		ApprovedBy: "operator", ApprovedBySource: "notification-hub", ApprovedAt: fixedNow.Add(-time.Hour),
	}
}

func joinedErr(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	return strings.ToLower(err.Error())
}
