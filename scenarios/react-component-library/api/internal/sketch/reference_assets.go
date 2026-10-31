package sketch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxReferenceAssetBytes int64 = 20 << 20

// SaveReferenceAsset stores the user-owned source bytes beside the experience
// that consumes them. The content hash is the stable revision identity; the
// original filename is retained only as display metadata.
func SaveReferenceAsset(root, scenario, page, filename, mime string, src io.Reader) (id, storedName string, size int64, err error) {
	if !safeSegment.MatchString(scenario) || !safeSegment.MatchString(page) {
		return "", "", 0, fmt.Errorf("invalid reference asset target")
	}
	if !strings.HasPrefix(strings.ToLower(mime), "image/") {
		return "", "", 0, fmt.Errorf("reference asset must be an image")
	}
	limited := io.LimitReader(src, maxReferenceAssetBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", "", 0, err
	}
	if int64(len(data)) > maxReferenceAssetBytes {
		return "", "", 0, fmt.Errorf("reference asset exceeds %d MiB", maxReferenceAssetBytes>>20)
	}
	sum := sha256.Sum256(data)
	id = hex.EncodeToString(sum[:])
	ext := filepath.Ext(filename)
	if len(ext) > 8 || strings.ContainsAny(ext, `/\\`) {
		ext = ""
	}
	storedName = id + ext
	dir := filepath.Join(root, "scenarios", scenario, "experience", "reference-assets", page)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", 0, err
	}
	path := filepath.Join(dir, storedName)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", "", 0, err
	}
	return id, storedName, int64(len(data)), nil
}

func ReferenceAssetPath(root, scenario, page, storedName string) (string, error) {
	if !safeSegment.MatchString(scenario) || !safeSegment.MatchString(page) || !safeSegment.MatchString(strings.TrimSuffix(storedName, filepath.Ext(storedName))) {
		return "", fmt.Errorf("invalid reference asset path")
	}
	if filepath.Base(storedName) != storedName {
		return "", fmt.Errorf("invalid reference asset name")
	}
	return filepath.Join(root, "scenarios", scenario, "experience", "reference-assets", page, storedName), nil
}
