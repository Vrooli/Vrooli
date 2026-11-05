package forensics

import (
	"errors"
	"io/fs"
	"testing"
	"time"
)

func TestPstoreNotPresent(t *testing.T) {
	fsys := FileSystem{
		ReadDirFn: func(string) ([]fs.DirEntry, error) {
			return nil, fs.ErrNotExist
		},
	}
	s := NewService(nil, nil, fsys, fixedNow)
	env := s.Pstore()
	if env.Available {
		t.Fatal("expected not available")
	}
	if env.Reason == "" {
		t.Fatal("expected reason")
	}
}

func TestPstorePermissionDenied(t *testing.T) {
	fsys := FileSystem{
		ReadDirFn: func(string) ([]fs.DirEntry, error) { return nil, fs.ErrPermission },
	}
	s := NewService(nil, nil, fsys, fixedNow)
	env := s.Pstore()
	if env.Available {
		t.Fatal("expected not available")
	}
}

func TestPstoreEmpty(t *testing.T) {
	fsys := FileSystem{
		ReadDirFn: func(string) ([]fs.DirEntry, error) { return nil, nil },
	}
	s := NewService(nil, nil, fsys, fixedNow)
	env := s.Pstore()
	if !env.Available {
		t.Fatalf("expected available, got reason=%q", env.Reason)
	}
	r := env.Data.(PstoreReport)
	if len(r.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(r.Entries))
	}
}

func TestPstoreEntries(t *testing.T) {
	fsys := FileSystem{
		ReadDirFn: func(string) ([]fs.DirEntry, error) {
			return []fs.DirEntry{
				stubDirEntry{name: "dmesg-erst-1"},
				stubDirEntry{name: "console-ramoops-0"},
				stubDirEntry{name: "pmsg-ramoops-0"},
				stubDirEntry{name: "weird-thing"},
			}, nil
		},
		StatFn: func(string) (fs.FileInfo, error) {
			return stubFileInfo{size: 4096, mod: time.Date(2026, 5, 6, 1, 2, 3, 0, time.UTC)}, nil
		},
	}
	s := NewService(nil, nil, fsys, fixedNow)
	env := s.Pstore()
	if !env.Available {
		t.Fatal("expected available")
	}
	r := env.Data.(PstoreReport)
	if len(r.Entries) != 4 {
		t.Fatalf("got %d entries, want 4", len(r.Entries))
	}
	want := map[string]string{
		"dmesg-erst-1":      "dmesg",
		"console-ramoops-0": "console",
		"pmsg-ramoops-0":    "pmsg",
		"weird-thing":       "unknown",
	}
	for _, e := range r.Entries {
		if want[e.Name] != e.Kind {
			t.Errorf("kind for %s = %s, want %s", e.Name, e.Kind, want[e.Name])
		}
	}
}

func TestPstoreOtherError(t *testing.T) {
	fsys := FileSystem{
		ReadDirFn: func(string) ([]fs.DirEntry, error) { return nil, errors.New("kaboom") },
	}
	s := NewService(nil, nil, fsys, fixedNow)
	env := s.Pstore()
	if env.Available {
		t.Fatal("expected not available")
	}
}
