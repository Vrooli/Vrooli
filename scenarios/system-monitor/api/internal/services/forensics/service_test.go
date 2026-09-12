package forensics

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"
)

// stubExec implements CommandExecutor for tests.
type stubExec struct {
	out []byte
	err error
}

func (s *stubExec) CombinedOutput(_ context.Context, _ string, _ ...string) ([]byte, error) {
	return s.out, s.err
}

// stubDirEntry implements fs.DirEntry for tests.
type stubDirEntry struct {
	name string
	dir  bool
}

func (s stubDirEntry) Name() string               { return s.name }
func (s stubDirEntry) IsDir() bool                { return s.dir }
func (s stubDirEntry) Type() fs.FileMode          { return 0 }
func (s stubDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

// stubFileInfo implements fs.FileInfo for tests.
type stubFileInfo struct {
	size int64
	mod  time.Time
}

func (s stubFileInfo) Name() string       { return "" }
func (s stubFileInfo) Size() int64        { return s.size }
func (s stubFileInfo) Mode() fs.FileMode  { return 0 }
func (s stubFileInfo) ModTime() time.Time { return s.mod }
func (s stubFileInfo) IsDir() bool        { return false }
func (s stubFileInfo) Sys() any           { return nil }

func fixedNow() time.Time {
	return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
}

func TestMemoCacheHit(t *testing.T) {
	calls := 0
	exec := &stubExec{err: errors.New("executable file not found in $PATH")}
	s := NewService(nil, exec, FileSystem{}, fixedNow)
	// Wrap exec to count calls.
	s.exec = countingExec{inner: exec, count: &calls}

	_ = s.MCE(context.Background())
	_ = s.MCE(context.Background())
	if calls != 1 {
		t.Errorf("expected 1 exec call due to memoization, got %d", calls)
	}
}

type countingExec struct {
	inner CommandExecutor
	count *int
}

func (c countingExec) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	*c.count++
	return c.inner.CombinedOutput(ctx, name, args...)
}

func TestMemoCacheExpires(t *testing.T) {
	calls := 0
	now := fixedNow()
	exec := &stubExec{err: errors.New("not installed")}
	s := NewService(nil, countingExec{inner: exec, count: &calls}, FileSystem{}, func() time.Time { return now })

	_ = s.MCE(context.Background())
	now = now.Add(memoTTL + time.Second)
	_ = s.MCE(context.Background())
	if calls != 2 {
		t.Errorf("expected 2 exec calls (cache expired), got %d", calls)
	}
}
