package binaryfetch

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type tarEntry struct {
	hdr  tar.Header
	body string
}

// writeTarGz materialises a tar.gz fixture on disk for the bounded walkers.
func writeTarGz(t *testing.T, entries []tarEntry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := e.hdr
		if hdr.Typeflag == 0 {
			hdr.Typeflag = tar.TypeReg
		}
		if hdr.Typeflag == tar.TypeReg {
			hdr.Size = int64(len(e.body))
		}
		if hdr.Mode == 0 {
			hdr.Mode = 0o644
		}
		if err := tw.WriteHeader(&hdr); err != nil {
			t.Fatalf("write header %q: %v", hdr.Name, err)
		}
		if hdr.Typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func violationOf(t *testing.T, err error) ArchiveViolation {
	t.Helper()
	var archiveErr *ArchiveError
	if !errors.As(err, &archiveErr) {
		t.Fatalf("error %v is not an ArchiveError", err)
	}
	return archiveErr.Violation
}

// [REQ:STC-P0-010] A malicious archive path never produces a write outside the
// owned staging root, and the dry pass refuses it before any write.
func TestBoundedWalkRefusesTraversalBeforeWriting(t *testing.T) {
	for name, entries := range map[string][]tarEntry{
		"dot-dot":       {{hdr: tar.Header{Name: "ok.txt"}, body: "a"}, {hdr: tar.Header{Name: "../escape.txt"}, body: "b"}},
		"absolute":      {{hdr: tar.Header{Name: "/etc/passwd"}, body: "b"}},
		"hardlink-out":  {{hdr: tar.Header{Name: "ok.txt"}, body: "a"}, {hdr: tar.Header{Name: "alias", Typeflag: tar.TypeLink, Linkname: "../../outside"}}},
		"nested-dotdot": {{hdr: tar.Header{Name: "a/b/../../../x"}, body: "b"}},
	} {
		t.Run(name, func(t *testing.T) {
			archive := writeTarGz(t, entries)
			if _, err := InspectArchiveBounded(archive, "tar.gz", ExtractOptions{}); violationOf(t, err) != ViolationTraversal {
				t.Fatalf("inspect violation = %v, want traversal", err)
			}
			dest := filepath.Join(t.TempDir(), "stage")
			_, err := ExtractArchiveBounded(archive, "tar.gz", dest, ExtractOptions{})
			if violationOf(t, err) != ViolationTraversal {
				t.Fatalf("extract violation = %v, want traversal", err)
			}
			if _, statErr := os.Stat(filepath.Join(filepath.Dir(dest), "escape.txt")); statErr == nil {
				t.Fatal("traversal entry was written outside the stage root")
			}
		})
	}
}

// [REQ:STC-P0-010] An escaping symlink is refused by the dry pass and by
// extraction through the same rule.
func TestBoundedWalkRefusesEscapingSymlink(t *testing.T) {
	for name, linkname := range map[string]string{
		"relative-out": "../../outside",
		"absolute":     "/etc",
	} {
		t.Run(name, func(t *testing.T) {
			archive := writeTarGz(t, []tarEntry{
				{hdr: tar.Header{Name: "dir/", Typeflag: tar.TypeDir, Mode: 0o755}},
				{hdr: tar.Header{Name: "dir/link", Typeflag: tar.TypeSymlink, Linkname: linkname, Mode: 0o777}},
			})
			if _, err := InspectArchiveBounded(archive, "tar.gz", ExtractOptions{}); violationOf(t, err) != ViolationSymlinkEscape {
				t.Fatalf("inspect violation = %v, want symlink_escape", err)
			}
			if _, err := ExtractArchiveBounded(archive, "tar.gz", filepath.Join(t.TempDir(), "stage"), ExtractOptions{}); violationOf(t, err) != ViolationSymlinkEscape {
				t.Fatalf("extract violation = %v, want symlink_escape", err)
			}
		})
	}
}

func TestBoundedWalkAcceptsInternalSymlink(t *testing.T) {
	archive := writeTarGz(t, []tarEntry{
		{hdr: tar.Header{Name: "bin/tool"}, body: "#!/bin/sh\n"},
		{hdr: tar.Header{Name: "bin/alias", Typeflag: tar.TypeSymlink, Linkname: "tool", Mode: 0o777}},
	})
	summary, err := InspectArchiveBounded(archive, "tar.gz", ExtractOptions{})
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if summary.Files != 1 || summary.Symlinks != 1 || summary.Entries != 2 {
		t.Fatalf("summary = %+v", summary)
	}
	dest := filepath.Join(t.TempDir(), "stage")
	if _, err := ExtractArchiveBounded(archive, "tar.gz", dest, ExtractOptions{}); err != nil {
		t.Fatalf("extract: %v", err)
	}
	if target, err := os.Readlink(filepath.Join(dest, "bin", "alias")); err != nil || target != "tool" {
		t.Fatalf("symlink = %q err=%v", target, err)
	}
}

// [REQ:STC-P0-010] Entry-count bombs are refused at the declared budget.
func TestBoundedWalkRefusesEntryCountBomb(t *testing.T) {
	entries := make([]tarEntry, 0, 50)
	for i := 0; i < 50; i++ {
		entries = append(entries, tarEntry{hdr: tar.Header{Name: "f" + strings.Repeat("x", i%7) + ".txt"}, body: "1"})
	}
	archive := writeTarGz(t, entries)
	if _, err := InspectArchiveBounded(archive, "tar.gz", ExtractOptions{MaxEntries: 10}); violationOf(t, err) != ViolationEntryLimit {
		t.Fatalf("inspect violation = %v, want entry_limit", err)
	}
	dest := filepath.Join(t.TempDir(), "stage")
	if _, err := ExtractArchiveBounded(archive, "tar.gz", dest, ExtractOptions{MaxEntries: 10}); violationOf(t, err) != ViolationEntryLimit {
		t.Fatalf("extract violation = %v, want entry_limit", err)
	}
	written, _ := os.ReadDir(dest)
	if len(written) > 10 {
		t.Fatalf("extraction wrote %d entries past the budget", len(written))
	}
}

// [REQ:STC-P0-010] Expanded-size and per-entry bombs are refused before the
// offending entry is written.
func TestBoundedWalkRefusesSizeBombs(t *testing.T) {
	big := strings.Repeat("A", 4096)
	archive := writeTarGz(t, []tarEntry{
		{hdr: tar.Header{Name: "a.bin"}, body: big},
		{hdr: tar.Header{Name: "b.bin"}, body: big},
		{hdr: tar.Header{Name: "c.bin"}, body: big},
	})
	t.Run("expanded", func(t *testing.T) {
		if _, err := InspectArchiveBounded(archive, "tar.gz", ExtractOptions{MaxExpandedBytes: 10000}); violationOf(t, err) != ViolationTooLarge {
			t.Fatalf("inspect violation = %v, want too_large", err)
		}
		dest := filepath.Join(t.TempDir(), "stage")
		if _, err := ExtractArchiveBounded(archive, "tar.gz", dest, ExtractOptions{MaxExpandedBytes: 10000}); violationOf(t, err) != ViolationTooLarge {
			t.Fatalf("extract violation = %v, want too_large", err)
		}
		if _, err := os.Stat(filepath.Join(dest, "c.bin")); err == nil {
			t.Fatal("entry past the expanded budget was written")
		}
	})
	t.Run("per-entry", func(t *testing.T) {
		if _, err := InspectArchiveBounded(archive, "tar.gz", ExtractOptions{MaxEntryBytes: 100}); violationOf(t, err) != ViolationTooLarge {
			t.Fatalf("inspect violation = %v, want too_large", err)
		}
		dest := filepath.Join(t.TempDir(), "stage")
		if _, err := ExtractArchiveBounded(archive, "tar.gz", dest, ExtractOptions{MaxEntryBytes: 100}); violationOf(t, err) != ViolationTooLarge {
			t.Fatalf("extract violation = %v, want too_large", err)
		}
		if _, err := os.Stat(filepath.Join(dest, "a.bin")); err == nil {
			t.Fatal("oversized entry was written")
		}
	})
}

func TestBoundedWalkRefusesDeviceEntriesAndSpentDeadline(t *testing.T) {
	archive := writeTarGz(t, []tarEntry{{hdr: tar.Header{Name: "dev/null", Typeflag: tar.TypeChar, Mode: 0o666}}})
	if _, err := InspectArchiveBounded(archive, "tar.gz", ExtractOptions{}); violationOf(t, err) != ViolationUnsupportedEntry {
		t.Fatalf("inspect violation = %v, want unsupported_entry", err)
	}
	fine := writeTarGz(t, []tarEntry{{hdr: tar.Header{Name: "a.txt"}, body: "a"}})
	if _, err := InspectArchiveBounded(fine, "tar.gz", ExtractOptions{Deadline: time.Now().Add(-time.Second)}); violationOf(t, err) != ViolationDeadline {
		t.Fatalf("inspect violation = %v, want deadline", err)
	}
}

func TestInspectArchiveBoundedWritesNothing(t *testing.T) {
	archive := writeTarGz(t, []tarEntry{
		{hdr: tar.Header{Name: "dir/", Typeflag: tar.TypeDir, Mode: 0o755}},
		{hdr: tar.Header{Name: "dir/a.txt"}, body: "hello"},
	})
	summary, err := InspectArchiveBounded(archive, "tar.gz", ExtractOptions{MaxEntries: 10, MaxExpandedBytes: 1 << 20})
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if summary.Entries != 2 || summary.ExpandedBytes != 5 || summary.Directories != 1 || summary.Files != 1 {
		t.Fatalf("summary = %+v", summary)
	}
	if _, err := os.Stat(inspectRoot); err == nil {
		t.Fatal("inspect pass created its placeholder root")
	}
}
