package binaryfetch

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// tarWith builds an in-memory tar containing the given headers/bodies.
func tarWith(t *testing.T, entries []struct {
	hdr  tar.Header
	body string
},
) *tar.Reader {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		hdr := e.hdr
		hdr.Size = int64(len(e.body))
		if hdr.Typeflag == tar.TypeXGlobalHeader {
			// Go's writer accepts a global header only in this exact shape;
			// real tarballs carry it with a "pax_global_header" name, which is
			// what the extractor sees on read.
			hdr = tar.Header{Typeflag: tar.TypeXGlobalHeader, Name: e.hdr.Name, PAXRecords: map[string]string{"comment": "fixture"}}
		}
		if err := tw.WriteHeader(&hdr); err != nil {
			t.Fatalf("write header %q: %v", hdr.Name, err)
		}
		if e.body != "" {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatalf("write body %q: %v", hdr.Name, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return tar.NewReader(bytes.NewReader(buf.Bytes()))
}

// Regression: hardening the extractor to reject unknown tar entry types also
// rejected `pax_global_header`, which carries no file at all and appears in
// every git-produced tarball. It made a legitimate upstream archive
// un-extractable and blocked the resource from installing.
func TestExtractAllTarReaderSkipsPaxGlobalHeader(t *testing.T) {
	dest := t.TempDir()
	tr := tarWith(t, []struct {
		hdr  tar.Header
		body string
	}{
		{tar.Header{Name: "pax_global_header", Typeflag: tar.TypeXGlobalHeader, Mode: 0o644}, ""},
		{tar.Header{Name: "app/", Typeflag: tar.TypeDir, Mode: 0o755}, ""},
		{tar.Header{Name: "app/main.py", Typeflag: tar.TypeReg, Mode: 0o644}, "print('hi')\n"},
	})
	if err := extractAllTarReader(tr, dest); err != nil {
		t.Fatalf("extractAllTarReader() error = %v, want nil", err)
	}
	body, err := os.ReadFile(filepath.Join(dest, "app", "main.py"))
	if err != nil {
		t.Fatalf("expected the real entries to extract: %v", err)
	}
	if string(body) != "print('hi')\n" {
		t.Fatalf("extracted body = %q", body)
	}
	// The metadata entry must not become a file.
	if _, err := os.Stat(filepath.Join(dest, "pax_global_header")); !os.IsNotExist(err) {
		t.Fatalf("pax_global_header was materialised as a file (err=%v)", err)
	}
}

// The strictness itself must survive: an entry that DOES claim a file we cannot
// place still has to fail loudly rather than be dropped.
func TestExtractAllTarReaderStillRejectsUnplaceableEntries(t *testing.T) {
	dest := t.TempDir()
	tr := tarWith(t, []struct {
		hdr  tar.Header
		body string
	}{
		{tar.Header{Name: "dev/null", Typeflag: tar.TypeChar, Mode: 0o666}, ""},
	})
	if err := extractAllTarReader(tr, dest); err == nil {
		t.Fatal("extractAllTarReader() = nil, want an error for a character device entry")
	}
}

// Hardlinks are the entry type OCI layers use routinely and the one whose
// silent drop produced incomplete trees that still reported success.
func TestExtractAllTarReaderMaterialisesHardlinks(t *testing.T) {
	dest := t.TempDir()
	tr := tarWith(t, []struct {
		hdr  tar.Header
		body string
	}{
		{tar.Header{Name: "bin/tool", Typeflag: tar.TypeReg, Mode: 0o755}, "#!/bin/sh\n"},
		{tar.Header{Name: "bin/tool-alias", Typeflag: tar.TypeLink, Linkname: "bin/tool", Mode: 0o755}, ""},
	})
	if err := extractAllTarReader(tr, dest); err != nil {
		t.Fatalf("extractAllTarReader() error = %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dest, "bin", "tool-alias"))
	if err != nil || string(body) != "#!/bin/sh\n" {
		t.Fatalf("hardlink not materialised: body=%q err=%v", body, err)
	}
}
