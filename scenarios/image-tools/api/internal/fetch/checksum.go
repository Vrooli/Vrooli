package fetch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// HashFile returns the sha256 (hex) and byte size of the file at p.
func HashFile(p string) (sum string, size int64, err error) {
	f, err := os.Open(p) //nolint:gosec // path is engine-constructed under Root
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// TreeManifestHash computes a deterministic integrity hash over every regular
// file under dir: sha256 of the sorted "relpath\x00filesha256\n" lines. It is the
// multi-file analogue of a single artifact's sha256 — stable across re-fetches of
// the same immutable revision, sensitive to any changed/added/removed file. The
// HuggingFace cache bookkeeping dir (.cache) is excluded. Returns the hash and
// the total bytes of the hashed files.
func TreeManifestHash(dir string) (string, int64, error) {
	type entry struct {
		rel string
		sha string
	}
	var entries []entry
	var total int64
	walkErr := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".cache" {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		sum, size, hErr := HashFile(p)
		if hErr != nil {
			return hErr
		}
		rel, relErr := filepath.Rel(dir, p)
		if relErr != nil {
			return relErr
		}
		entries = append(entries, entry{rel: filepath.ToSlash(rel), sha: sum})
		total += size
		return nil
	})
	if walkErr != nil {
		return "", 0, walkErr
	}
	if len(entries) == 0 {
		return "", 0, fmt.Errorf("repo snapshot is empty")
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].rel < entries[j].rel })
	h := sha256.New()
	for _, e := range entries {
		_, _ = io.WriteString(h, e.rel)
		_, _ = h.Write([]byte{0})
		_, _ = io.WriteString(h, e.sha)
		_, _ = h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil)), total, nil
}
