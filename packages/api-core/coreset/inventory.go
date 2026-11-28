package coreset

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
)

// FileInventoryEntry is a hashed source file bound to a selected dependency
// closure. The graph digest and this source inventory digest remain distinct.
type FileInventoryEntry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Mode   uint32 `json:"mode"`
	Size   int64  `json:"size_bytes"`
}

func NormalizeFileMode(mode uint32) uint32 {
	mode &= 0o777
	if mode == 0 {
		return 0o640
	}
	return mode
}

// FileInventoryDigest computes a stable digest over exact path/content/mode/size
// membership. It validates ordering-independent uniqueness and safe paths.
func FileInventoryDigest(entries []FileInventoryEntry) (string, error) {
	canonical := append([]FileInventoryEntry(nil), entries...)
	for i := range canonical {
		canonical[i].Path = strings.TrimSpace(canonical[i].Path)
		canonical[i].SHA256 = strings.ToLower(strings.TrimSpace(canonical[i].SHA256))
		canonical[i].Mode = NormalizeFileMode(canonical[i].Mode)
	}
	sort.Slice(canonical, func(i, j int) bool { return canonical[i].Path < canonical[j].Path })
	for i, e := range canonical {
		if err := ValidateInventoryPath(e.Path); err != nil {
			return "", err
		}
		if e.Size < 0 {
			return "", fmt.Errorf("negative file size for %s", e.Path)
		}
		decoded, err := hex.DecodeString(e.SHA256)
		if err != nil || len(decoded) != sha256.Size {
			return "", fmt.Errorf("invalid sha256 for %s", e.Path)
		}
		if i > 0 && canonical[i-1].Path == e.Path {
			return "", fmt.Errorf("duplicate inventory path %q", e.Path)
		}
	}
	b, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func ValidateInventoryPath(value string) error {
	if value == "" || path.IsAbs(value) || path.Clean(value) != value || value == "." || value == ".." || strings.HasPrefix(value, "../") || strings.Contains(value, "\\") || strings.ContainsRune(value, 0) {
		return fmt.Errorf("invalid closure path %q", value)
	}
	return nil
}

// VerifyFileInventory validates that delivered membership equals the selected
// manifest and that both hash to the expected inventory digest.
func VerifyFileInventory(expected, actual []FileInventoryEntry, digest string) error {
	wd, err := FileInventoryDigest(expected)
	if err != nil {
		return err
	}
	if wd != digest {
		return errors.New("selected file inventory digest mismatch")
	}
	got, err := FileInventoryDigest(actual)
	if err != nil {
		return err
	}
	if got != digest {
		return errors.New("delivered file inventory digest mismatch")
	}
	return nil
}
