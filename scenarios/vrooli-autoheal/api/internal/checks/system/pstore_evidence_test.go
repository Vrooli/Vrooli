package system

import (
	"context"
	"errors"
	"io/fs"
	"runtime"
	"testing"
	"time"
	"vrooli-autoheal/internal/checks"
)

type mockDirEntry struct {
	name  string
	isDir bool
}

func (m mockDirEntry) Name() string               { return m.name }
func (m mockDirEntry) IsDir() bool                { return m.isDir }
func (m mockDirEntry) Type() fs.FileMode          { return 0 }
func (m mockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

type mockPstoreReader struct {
	entries []fs.DirEntry
	err     error
}

func (m mockPstoreReader) ReadDir(string) ([]fs.DirEntry, error) {
	return m.entries, m.err
}

type pathPstoreReader struct {
	entries map[string][]fs.DirEntry
	errs    map[string]error
}

func (m pathPstoreReader) ReadDir(path string) ([]fs.DirEntry, error) {
	if err, ok := m.errs[path]; ok {
		return nil, err
	}
	return m.entries[path], nil
}

func runPstore(entries []fs.DirEntry, err error) checks.Result {
	c := NewPstoreEvidenceCheck(WithPstoreReader(mockPstoreReader{entries: entries, err: err}))
	return c.Run(context.Background())
}

func TestPstoreOK_Empty(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only check")
	}
	r := runPstore(nil, nil)
	if r.Status != checks.StatusOK {
		t.Errorf("Status = %s, want OK", r.Status)
	}
}

func TestPstoreOK_NotConfigured(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only check")
	}
	r := runPstore(nil, fs.ErrNotExist)
	if r.Status != checks.StatusOK {
		t.Errorf("Status = %s, want OK on missing pstore", r.Status)
	}
	if r.Details["pstoreConfigured"] != false {
		t.Errorf("pstoreConfigured should be false: %v", r.Details)
	}
}

func TestPstoreWarning_EACCES(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only check")
	}
	r := runPstore(nil, fs.ErrPermission)
	if r.Status != checks.StatusWarning {
		t.Errorf("Status = %s, want WARNING on EACCES", r.Status)
	}
	if r.Details["coverageGap"] != true {
		t.Errorf("coverageGap = %v, want true", r.Details["coverageGap"])
	}
}

func TestPstoreUsesExportWhenDirectUnreadable(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only check")
	}
	c := NewPstoreEvidenceCheck(
		WithPstorePath("/direct/pstore"),
		WithPstoreExportPath("/export/pstore"),
		WithPstoreReader(pathPstoreReader{
			entries: map[string][]fs.DirEntry{"/export/pstore": {mockDirEntry{name: "manifest.json"}}},
			errs:    map[string]error{"/direct/pstore": fs.ErrPermission},
		}),
	)
	r := c.Run(context.Background())
	if r.Status != checks.StatusOK {
		t.Fatalf("Status = %s, want OK with readable export", r.Status)
	}
	if r.Details["sourceKind"] != "export" {
		t.Fatalf("sourceKind = %v, want export", r.Details["sourceKind"])
	}
}

func TestPstoreCritical_OnDmesg(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only check")
	}
	r := runPstore([]fs.DirEntry{
		mockDirEntry{name: "dmesg-efi_pstore-1234"},
		mockDirEntry{name: "dmesg-efi_pstore-1235"},
	}, nil)
	if r.Status != checks.StatusCritical {
		t.Errorf("Status = %s, want CRITICAL on dmesg-* entries", r.Status)
	}
	if got := r.Details["dmesgCount"]; got != 2 {
		t.Errorf("dmesgCount = %v, want 2", got)
	}
}

func TestPstoreCritical_OnConsole(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only check")
	}
	r := runPstore([]fs.DirEntry{
		mockDirEntry{name: "console-ramoops-0"},
	}, nil)
	if r.Status != checks.StatusCritical {
		t.Errorf("Status = %s, want CRITICAL", r.Status)
	}
}

func TestPstoreWarning_OnPmsgOnly(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only check")
	}
	r := runPstore([]fs.DirEntry{
		mockDirEntry{name: "pmsg-ramoops-0"},
	}, nil)
	if r.Status != checks.StatusWarning {
		t.Errorf("Status = %s, want WARNING for pmsg-only", r.Status)
	}
}

func TestPstoreWarning_OnGenericError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only check")
	}
	r := runPstore(nil, errors.New("some weird I/O error"))
	if r.Status != checks.StatusWarning {
		t.Errorf("Status = %s, want WARNING on generic error", r.Status)
	}
}

func TestPstoreMetadata(t *testing.T) {
	c := NewPstoreEvidenceCheck()
	if c.ID() != "system-pstore-evidence" {
		t.Errorf("ID = %s", c.ID())
	}
	if c.Category() != checks.CategorySystem {
		t.Errorf("Category = %s", c.Category())
	}
	if c.IntervalSeconds() != 300 {
		t.Errorf("Interval = %d", c.IntervalSeconds())
	}
}

// Sanity: timestamp gets set on healthy result.
func TestPstoreSetsTimestamp(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only check")
	}
	r := runPstore(nil, nil)
	if r.Timestamp.IsZero() || time.Since(r.Timestamp) > time.Minute {
		t.Errorf("Timestamp not set: %v", r.Timestamp)
	}
}
