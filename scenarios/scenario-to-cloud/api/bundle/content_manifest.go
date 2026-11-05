package bundle

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ContentEntry is one file the bundle carries and its content digest.
type ContentEntry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// ContentManifest lists every regular file a bundle spec would carry, in tar
// order, with its sha256. Generated extra files are included under their
// bundle path so the manifest covers exactly what the archive covers.
// Symlinks are recorded by the sha256 of their link target text.
func ContentManifest(repoRoot string, spec MiniBundleSpec) ([]ContentEntry, error) {
	paths, err := collectIncludedPaths(repoRoot, spec)
	if err != nil {
		return nil, err
	}
	entries := make([]ContentEntry, 0, len(paths)+len(spec.ExtraFiles))
	for _, rel := range paths {
		abs := filepath.Join(repoRoot, rel)
		info, err := os.Lstat(abs)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			continue
		}
		var sum string
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(abs)
			if err != nil {
				return nil, err
			}
			digest := sha256.Sum256([]byte(target))
			sum = hex.EncodeToString(digest[:])
		} else {
			sum, err = fileSHA256(abs)
			if err != nil {
				return nil, err
			}
		}
		entries = append(entries, ContentEntry{Path: toTarPath(rel), SHA256: sum})
	}
	extraKeys := make([]string, 0, len(spec.ExtraFiles))
	for rel := range spec.ExtraFiles {
		extraKeys = append(extraKeys, rel)
	}
	sort.Strings(extraKeys)
	for _, rel := range extraKeys {
		digest := sha256.Sum256(spec.ExtraFiles[rel])
		entries = append(entries, ContentEntry{Path: toTarPath(rel), SHA256: hex.EncodeToString(digest[:])})
	}
	return entries, nil
}

// ContentManifestDigest is sha256 over the sorted "path\tsha256\n" lines of a
// content manifest.
func ContentManifestDigest(entries []ContentEntry) string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, entry.Path+"\t"+entry.SHA256+"\n")
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "")))
	return hex.EncodeToString(sum[:])
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
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
