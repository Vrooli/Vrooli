package cleanup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseChaosRate(t *testing.T) {
	tests := []struct {
		raw  string
		want int64
	}{
		{"20GiB/h", 20 * (1 << 30)},
		{"1.5gb/h", 1_500_000_000},
		{"512MiB/h", 512 * (1 << 20)},
	}
	for _, test := range tests {
		got, err := parseChaosRate(test.raw)
		if err != nil || got != test.want {
			t.Errorf("parseChaosRate(%q) = %d, %v; want %d", test.raw, got, err, test.want)
		}
	}
	for _, raw := range []string{"20GiB", "0GiB/h", "101GiB/h"} {
		if _, err := parseChaosRate(raw); err == nil {
			t.Errorf("parseChaosRate(%q) accepted invalid rate", raw)
		}
	}
}

func TestGovernedChaosRootRequiresExistingDirectoryUnderGoWork(t *testing.T) {
	base := t.TempDir()
	t.Setenv("VROOLI_HOME", base)
	root := filepath.Join(base, "tmp", "go-work", "chaos")
	if _, err := governedChaosRoot(root); err == nil {
		t.Fatal("missing governed root was accepted")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if got, err := governedChaosRoot(root); err != nil || got != root {
		t.Fatalf("governedChaosRoot(%q) = %q, %v", root, got, err)
	}
	if _, err := governedChaosRoot(filepath.Join(base, "other")); err == nil {
		t.Fatal("root outside governed tmp was accepted")
	}
	if _, err := governedChaosRoot(filepath.Join(base, "tmp", "other")); err == nil {
		t.Fatal("root outside governed go-work was accepted")
	}
}

func TestWriteChaosChunkAllocatesARealFileExtent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chaos.bin")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer file.Close()

	const size = 4096
	n, err := writeChaosChunk(file, make([]byte, size), 0)
	if err != nil || n != size {
		t.Fatalf("writeChaosChunk() = %d, %v; want %d bytes", n, err, size)
	}
	info, err := file.Stat()
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != size {
		t.Fatalf("allocated file size = %d, want %d", info.Size(), size)
	}
}
