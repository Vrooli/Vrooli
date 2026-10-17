package templateengine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

// templateFileFingerprint records a single emitted file's contribution to
// the content hash. We aggregate over a path-sorted slice so the result is
// independent of filesystem walk order.
type templateFileFingerprint struct {
	rel  string
	size int64
	hash string
}

// computeTemplateHashes returns a manifest-only hash (raw template.json bytes)
// and a content hash over the file set that would actually be emitted into a
// generated scenario destination. The content hash is computed pre-substitution
// so it is a property of the template itself, not of any particular generated
// scenario's values.
//
// Both return values are lowercase hex sha256 digests. Either may be empty if
// computation fails for that hash; callers should treat hash failure as
// non-fatal (drift just becomes unmeasurable for that path).
func computeTemplateHashes(info templatecontracts.TemplateInfo) (manifestSha string, contentSha string, err error) {
	if info.Path == "" {
		return "", "", fmt.Errorf("template path is empty")
	}
	manifestSha, manifestErr := hashTemplateManifestFile(info.Path)
	contentSha, contentErr := hashTemplateContent(info.Path, info.Manifest)
	switch {
	case manifestErr != nil && contentErr != nil:
		return manifestSha, contentSha, fmt.Errorf("hash template: manifest=%v content=%v", manifestErr, contentErr)
	case manifestErr != nil:
		return manifestSha, contentSha, fmt.Errorf("hash template manifest: %w", manifestErr)
	case contentErr != nil:
		return manifestSha, contentSha, fmt.Errorf("hash template content: %w", contentErr)
	}
	return manifestSha, contentSha, nil
}

func hashTemplateManifestFile(templateDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(templateDir, "template.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func hashTemplateContent(templateDir string, manifest templatecontracts.TemplateManifest) (string, error) {
	var entries []templateFileFingerprint
	err := walkTemplateEmissions(templateDir, manifest, func(relPath, absPath string, entry fs.DirEntry) error {
		if entry.IsDir() {
			return nil
		}
		fi, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		entries = append(entries, templateFileFingerprint{
			rel:  filepath.ToSlash(relPath),
			size: fi.Size(),
			hash: hex.EncodeToString(sum[:]),
		})
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].rel < entries[j].rel })
	h := sha256.New()
	for _, fp := range entries {
		fmt.Fprintf(h, "%s|%d|%s\n", fp.rel, fp.size, fp.hash)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// templateFilesByPath returns a path-keyed map of per-file fingerprints. This
// is what `scenario template drift --verbose` consumes to point at individual
// files that differ between the current template and a generated scenario.
func templateFilesByPath(templateDir string, manifest templatecontracts.TemplateManifest) (map[string]templateFileFingerprint, error) {
	out := make(map[string]templateFileFingerprint)
	err := walkTemplateEmissions(templateDir, manifest, func(relPath, absPath string, entry fs.DirEntry) error {
		if entry.IsDir() {
			return nil
		}
		fi, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		rel := filepath.ToSlash(relPath)
		out[rel] = templateFileFingerprint{
			rel:  rel,
			size: fi.Size(),
			hash: hex.EncodeToString(sum[:]),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
