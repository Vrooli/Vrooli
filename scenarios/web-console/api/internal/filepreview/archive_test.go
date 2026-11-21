package filepreview

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func archiveFixture(t *testing.T, format string, names []string, bodies [][]byte) []byte {
	t.Helper()
	var b bytes.Buffer
	if format == "zip" {
		w := zip.NewWriter(&b)
		for i, n := range names {
			h := &zip.FileHeader{Name: n, Method: zip.Deflate}
			if n == "link" {
				h.SetMode(os.ModeSymlink | 0777)
			}
			f, err := w.CreateHeader(h)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = f.Write(bodies[i]); err != nil {
				t.Fatal(err)
			}
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	var output = &b
	var zipped bytes.Buffer
	var gz *gzip.Writer
	if format == "tgz" {
		gz = gzip.NewWriter(&zipped)
	}
	w := tar.NewWriter(output)
	if gz != nil {
		w = tar.NewWriter(gz)
	}
	for i, n := range names {
		h := &tar.Header{Name: n, Mode: 0600, Size: int64(len(bodies[i]))}
		if n == "link" {
			h.Typeflag = tar.TypeSymlink
			h.Linkname = "/etc/passwd"
			h.Size = 0
		}
		if err := w.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag != tar.TypeSymlink {
			if _, err := w.Write(bodies[i]); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if gz != nil {
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
		return zipped.Bytes()
	}
	return b.Bytes()
}
func issuedArchive(t *testing.T, name string, data []byte) (*Store, *Entry) {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(p)
	s := NewStore(time.Hour)
	id, _, err := s.Issue("session", &Target{ResolvedPath: p, Basename: name, Kind: KindArchive, SourceInfo: info, SizeBytes: info.Size(), ModTimeUnixNano: info.ModTime().UnixNano()})
	if err != nil {
		t.Fatal(err)
	}
	e, _ := s.Lookup("session", id)
	return s, e
}
func TestArchivePortableBrowsingAndDuplicateIdentity(t *testing.T) {
	for _, format := range []string{"zip", "tar", "tgz"} {
		t.Run(format, func(t *testing.T) {
			data := archiveFixture(t, format, []string{"docs/readme.md", "duplicate.txt", "duplicate.txt", "../escape", "C:/evil", "link"}, [][]byte{[]byte("# hello"), []byte("first"), []byte("second"), []byte("blocked"), nil, []byte("target")})
			s, root := issuedArchive(t, "sample."+format, data)
			list := s.ListArchive(context.Background(), "session", root.ID)
			if list.Status != "ready" || len(list.Entries) != 6 {
				t.Fatalf("listing: %+v", list)
			}
			for _, i := range []int{3, 4} {
				if list.Entries[i].Status != "unsafe" {
					t.Fatal(list.Entries[i])
				}
			}
			if list.Entries[5].Status != "link" {
				t.Fatal(list.Entries[5])
			}
			first, status := s.OpenArchiveEntry(context.Background(), "session", root.ID, "1")
			if status != "ready" || string(first.Data) != "first" {
				t.Fatal(status, first)
			}
			second, status := s.OpenArchiveEntry(context.Background(), "session", root.ID, "2")
			if status != "ready" || string(second.Data) != "second" {
				t.Fatal(status, second)
			}
			md, status := s.OpenArchiveEntry(context.Background(), "session", root.ID, "0")
			if status != "ready" || md.Kind != KindCode || md.RootID != root.ID || !md.ExpiresAt.Equal(root.ExpiresAt) {
				t.Fatal(status, md)
			}
			if _, status := s.OpenArchiveEntry(context.Background(), "session", root.ID, "3"); status != "blocked" {
				t.Fatal(status)
			}
			if got := s.ListArchive(context.Background(), "other", root.ID); got.Status != "expired" {
				t.Fatal(got)
			}
			if err := os.WriteFile(root.ResolvedPath, []byte("changed"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := s.Fresh(first); ArchiveFailureCode(err) != "stale" {
				t.Fatal(err)
			}
		})
	}
}
func TestArchiveRejectsBombsAndCorruption(t *testing.T) {
	t.Run("actual gzip expansion", func(t *testing.T) {
		data := archiveFixture(t, "tgz", []string{"large"}, [][]byte{bytes.Repeat([]byte{0}, 2<<20)})
		s, e := issuedArchive(t, "bomb.tgz", data)
		e.Budget.remaining = 1 << 20
		if got := s.ListArchive(context.Background(), "session", e.ID); got.Status != "limit" {
			t.Fatal(got)
		}
	})
	t.Run("declared ZIP member limit", func(t *testing.T) {
		data := archiveFixture(t, "zip", []string{"big.txt"}, [][]byte{[]byte("short")})
		pos := bytes.Index(data, []byte{'P', 'K', 1, 2})
		binary.LittleEndian.PutUint32(data[pos+24:], uint32(ArchiveMemberLimit+1))
		s, e := issuedArchive(t, "big.zip", data)
		list := s.ListArchive(context.Background(), "session", e.ID)
		if list.Status != "ready" || list.Entries[0].Status != "too_large" {
			t.Fatal(list)
		}
		if _, status := s.OpenArchiveEntry(context.Background(), "session", e.ID, "0"); status != "blocked" {
			t.Fatal(status)
		}
	})
	t.Run("forged ZIP count", func(t *testing.T) {
		data := archiveFixture(t, "zip", []string{"ok"}, [][]byte{nil})
		binary.LittleEndian.PutUint16(data[len(data)-12:], 4000)
		s, e := issuedArchive(t, "count.zip", data)
		if got := s.ListArchive(context.Background(), "session", e.ID); got.Status != "unsupported" {
			t.Fatal(got)
		}
	})
	t.Run("gzip checksum", func(t *testing.T) {
		data := archiveFixture(t, "tgz", []string{"ok"}, [][]byte{[]byte("yes")})
		data[len(data)-8] ^= 1
		s, e := issuedArchive(t, "crc.tgz", data)
		if got := s.ListArchive(context.Background(), "session", e.ID); got.Status != "corrupt" {
			t.Fatal(got)
		}
	})
	t.Run("ZIP checksum", func(t *testing.T) {
		data := archiveFixture(t, "zip", []string{"ok.txt"}, [][]byte{[]byte("yes")})
		pos := bytes.Index(data, []byte{'P', 'K', 1, 2})
		data[pos+16] ^= 1
		s, e := issuedArchive(t, "crc.zip", data)
		if _, status := s.OpenArchiveEntry(context.Background(), "session", e.ID, "0"); status != "corrupt" {
			t.Fatal(status)
		}
	})
	t.Run("cancel and concurrency", func(t *testing.T) {
		s, e := issuedArchive(t, "empty.zip", archiveFixture(t, "zip", nil, nil))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if got := s.ListArchive(ctx, "session", e.ID); got.Status != "timeout" {
			t.Fatal(got)
		}
		s.archiveJobs <- struct{}{}
		s.archiveJobs <- struct{}{}
		if got := s.ListArchive(context.Background(), "session", e.ID); got.Status != "busy" {
			t.Fatal(got)
		}
	})
}
func TestArchiveNestedDepthAndSharedBudget(t *testing.T) {
	inner := archiveFixture(t, "zip", []string{"deep.txt"}, [][]byte{[]byte("hello")})
	middle := archiveFixture(t, "zip", []string{"inner.zip"}, [][]byte{inner})
	outer := archiveFixture(t, "zip", []string{"middle.zip"}, [][]byte{middle})
	s, root := issuedArchive(t, "outer.zip", outer)
	m, status := s.OpenArchiveEntry(context.Background(), "session", root.ID, "0")
	if status != "ready" {
		t.Fatal(status)
	}
	i, status := s.OpenArchiveEntry(context.Background(), "session", m.ID, "0")
	if status != "ready" || i.Budget != root.Budget {
		t.Fatal(status)
	}
	leaf, status := s.OpenArchiveEntry(context.Background(), "session", i.ID, "0")
	if status != "ready" || string(leaf.Data) != "hello" {
		t.Fatal(status)
	}
	root.Budget.remaining = 0
	if _, status = s.OpenArchiveEntry(context.Background(), "session", i.ID, "0"); status != "limit" {
		t.Fatal(status)
	}
	s.now = func() time.Time { return root.ExpiresAt.Add(time.Second) }
	if _, err := s.Lookup("session", leaf.ID); err == nil {
		t.Fatal("expired member served")
	}
	s.Sweep()
	if s.virtualBytes != 0 {
		t.Fatal(s.virtualBytes)
	}
}
func TestArchivePortableNames(t *testing.T) {
	for _, name := range []string{"../x", "/root", "C:/x", `a\b`, "a/../b", "CON.txt", "a.", "x\u202ey", "a//b", strings.Repeat("x/", 33) + "x"} {
		if safeArchiveName(name) {
			t.Errorf("unsafe accepted: %q", name)
		}
	}
	for _, name := range []string{"docs/readme.txt", "unicode/你好.txt", "folder/"} {
		if !safeArchiveName(name) {
			t.Errorf("safe rejected: %q", name)
		}
	}
}

func TestArchiveMetadataPreflightAndVirtualQuota(t *testing.T) {
	t.Run("central directory entry ceiling", func(t *testing.T) {
		names := make([]string, archiveEntryLimit+1)
		bodies := make([][]byte, len(names))
		for i := range names {
			names[i] = "empty"
		}
		s, e := issuedArchive(t, "many.zip", archiveFixture(t, "zip", names, bodies))
		if got := s.ListArchive(context.Background(), "session", e.ID); got.Status != "limit" {
			t.Fatal(got.Status)
		}
	})
	t.Run("ambiguous end record in ZIP comment", func(t *testing.T) {
		data := archiveFixture(t, "zip", []string{"safe"}, [][]byte{nil})
		end := len(data) - 22
		comment := make([]byte, 24)
		copy(comment, []byte{'P', 'K', 5, 6})
		binary.LittleEndian.PutUint16(data[end+20:], uint16(len(comment)))
		data = append(data, comment...)
		s, e := issuedArchive(t, "ambiguous.zip", data)
		if got := s.ListArchive(context.Background(), "session", e.ID); got.Status != "corrupt" {
			t.Fatal(got.Status)
		}
	})
	t.Run("virtual handle quota", func(t *testing.T) {
		s, e := issuedArchive(t, "small.zip", archiveFixture(t, "zip", []string{"a.txt"}, [][]byte{[]byte("a")}))
		var first string
		for i := 0; i < virtualEntryLimit+1; i++ {
			opened, status := s.OpenArchiveEntry(context.Background(), "session", e.ID, "0")
			if status != "ready" {
				t.Fatal(status)
			}
			if i == 0 {
				first = opened.ID
			}
		}
		if _, err := s.Lookup("session", first); err == nil {
			t.Fatal("oldest virtual handle retained")
		}
		if _, err := s.Lookup("session", e.ID); err != nil {
			t.Fatal("original archive evicted")
		}
		if s.virtualBytes != virtualEntryLimit {
			t.Fatal(s.virtualBytes)
		}
	})
	t.Run("nested archive depth ceiling", func(t *testing.T) {
		data := archiveFixture(t, "zip", []string{"leaf.txt"}, [][]byte{nil})
		for i := 0; i < 3; i++ {
			data = archiveFixture(t, "zip", []string{"child.zip"}, [][]byte{data})
		}
		s, e := issuedArchive(t, "root.zip", data)
		for i := 0; i < 2; i++ {
			next, status := s.OpenArchiveEntry(context.Background(), "session", e.ID, "0")
			if status != "ready" {
				t.Fatal(status)
			}
			e = next
		}
		if _, status := s.OpenArchiveEntry(context.Background(), "session", e.ID, "0"); status != "depth" {
			t.Fatal(status)
		}
	})
}

func TestArchiveMalformedNamesAndConventionalTarRoot(t *testing.T) {
	t.Run("invalid UTF-8 stays inert and serializable", func(t *testing.T) {
		s, e := issuedArchive(t, "names.zip", archiveFixture(t, "zip", []string{string([]byte{0xff}) + ".txt"}, [][]byte{nil}))
		list := s.ListArchive(context.Background(), "session", e.ID)
		if list.Status != "ready" || list.Entries[0].Status != "unsafe" || list.Entries[0].Name != "�.txt" {
			t.Fatal(list)
		}
	})
	t.Run("conventional dot prefix", func(t *testing.T) {
		var b bytes.Buffer
		w := tar.NewWriter(&b)
		if err := w.WriteHeader(&tar.Header{Name: "./", Typeflag: tar.TypeDir, Mode: 0755}); err != nil {
			t.Fatal(err)
		}
		if err := w.WriteHeader(&tar.Header{Name: "./docs/hello.txt", Size: 5, Mode: 0600}); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte("hello"))
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		s, e := issuedArchive(t, "dot.tar", b.Bytes())
		list := s.ListArchive(context.Background(), "session", e.ID)
		if list.Status != "ready" || len(list.Entries) != 1 || list.Entries[0].Name != "docs/hello.txt" || list.Entries[0].ID != "1" {
			t.Fatal(list)
		}
		opened, status := s.OpenArchiveEntry(context.Background(), "session", e.ID, "1")
		if status != "ready" || string(opened.Data) != "hello" {
			t.Fatal(status)
		}
	})
	t.Run("replacement with preserved size and timestamp", func(t *testing.T) {
		s, e := issuedArchive(t, "swap.zip", archiveFixture(t, "zip", []string{"a.txt"}, [][]byte{nil}))
		info, _ := os.Stat(e.ResolvedPath)
		e.SourceInfo = info
		replacement := e.ResolvedPath + ".new"
		data, _ := os.ReadFile(e.ResolvedPath)
		if err := os.WriteFile(replacement, data, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(replacement, info.ModTime(), info.ModTime()); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(e.ResolvedPath); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(replacement, e.ResolvedPath); err != nil {
			t.Fatal(err)
		}
		if got := s.ListArchive(context.Background(), "session", e.ID); got.Status != "stale" {
			t.Fatal(got.Status)
		}
	})
}

func TestArchiveActiveBlobRetentionAndConcurrentOpen(t *testing.T) {
	s, root := issuedArchive(t, "pin.zip", archiveFixture(t, "zip", []string{"a.txt"}, [][]byte{[]byte("hello")}))
	member, status := s.OpenArchiveEntry(context.Background(), "session", root.ID, "0")
	if status != "ready" {
		t.Fatal(status)
	}
	pinned, release, err := s.AcquireBlob("session", member.ID)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < virtualEntryLimit; i++ {
		if _, status := s.OpenArchiveEntry(context.Background(), "session", root.ID, "0"); status != "ready" {
			t.Fatal(status)
		}
	}
	if _, err := s.Lookup("session", pinned.ID); err != nil {
		t.Fatal("active response evicted", err)
	}
	s.now = func() time.Time { return root.ExpiresAt.Add(time.Second) }
	s.Sweep()
	if _, err := s.Lookup("session", pinned.ID); err == nil {
		t.Fatal("expired handle usable")
	}
	if s.virtualBytes != 5 {
		t.Fatal("active bytes escaped quota", s.virtualBytes)
	}
	release()
	release()
	if s.virtualBytes != 0 {
		t.Fatal("active bytes not released", s.virtualBytes)
	}
}

func TestArchiveMIMEFallbackIsBinary(t *testing.T) {
	if got := mimeForKind(KindArchive, ".unregistered-archive-test"); got != "application/octet-stream" {
		t.Fatalf("archive binary fallback = %q", got)
	}
}
