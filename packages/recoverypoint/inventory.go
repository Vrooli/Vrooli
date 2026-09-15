package recoverypoint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileEntry is one regular file inside an object-store binding.
type FileEntry struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// ListFiles walks dir and returns every regular file (symlinks are not
// followed) sorted by relative slash path.
func ListFiles(dir string) ([]FileEntry, error) {
	var entries []FileEntry
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		sum, err := FileSHA256(path)
		if err != nil {
			return err
		}
		entries = append(entries, FileEntry{Path: filepath.ToSlash(rel), Bytes: info.Size(), SHA256: sum})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

// InventoryOf derives the deterministic inventory of a file list: the count
// of files and sha256 over "<path>\n<sha256>\n" per file in sorted order,
// matching the fixture-oracle/v1 seed rule.
func InventoryOf(entries []FileEntry) Inventory {
	var lines strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&lines, "%s\n%s\n", e.Path, e.SHA256)
	}
	sum := sha256.Sum256([]byte(lines.String()))
	return Inventory{Count: int64(len(entries)), Checksum: hex.EncodeToString(sum[:]), Comparable: true}
}

// DirInventory lists and digests every regular file beneath dir.
func DirInventory(dir string) (Inventory, []FileEntry, error) {
	entries, err := ListFiles(dir)
	if err != nil {
		return Inventory{}, nil, err
	}
	return InventoryOf(entries), entries, nil
}

// FileSHA256 returns the lowercase hex sha256 of a file.
func FileSHA256(path string) (string, error) {
	f, err := os.Open(path) //nolint:gosec // caller-owned path
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// BytesSHA256 returns the lowercase hex sha256 of data.
func BytesSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// DirIsClean reports whether dir does not exist or holds no entries. A
// non-empty directory is operator data the restore must never overwrite.
func DirIsClean(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	return len(entries) == 0, nil
}
