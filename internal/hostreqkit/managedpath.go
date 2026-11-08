package hostreqkit

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateManagedFilePath qualifies a privileged managed-file destination.
// Missing entries may be created by the existing directory/content owner;
// existing ancestors must be actual directories and the final entry a regular
// file. Directory aliases are refused even if their target is readable.
// The optional lstat seam permits a caller's coherent filesystem adapter;
// nil uses the real filesystem. This is bounded endpoint qualification, not
// an atomic transaction against arbitrary concurrent privileged writers.
func ValidateManagedFilePath(path string, lstat func(string) (os.FileInfo, error)) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == string(filepath.Separator) || len(path) > 4096 || strings.ContainsRune(path, 0) {
		return fmt.Errorf("managed file destination must be a bounded canonical absolute file path")
	}
	if lstat == nil {
		lstat = os.Lstat
	}
	ancestors := []string{path}
	for p := filepath.Dir(path); ; p = filepath.Dir(p) {
		ancestors = append(ancestors, p)
		if len(ancestors) > 256 {
			return fmt.Errorf("managed file ancestry exceeds its bound")
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	missing := false
	for i := len(ancestors) - 1; i >= 0; i-- {
		p := ancestors[i]
		info, err := lstat(p)
		if errors.Is(err, os.ErrNotExist) {
			missing = true
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect managed file path %s: %w", p, err)
		}
		if info == nil || missing {
			return fmt.Errorf("managed file ancestry changed or is unknown at %s", p)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("managed file path is a symlink at %s; preserved", p)
		}
		if i > 0 && !info.IsDir() {
			return fmt.Errorf("managed file parent is not a directory at %s; preserved", p)
		}
		if i == 0 && !info.Mode().IsRegular() {
			return fmt.Errorf("managed destination is not a regular file at %s; preserved", p)
		}
	}
	return nil
}
