package linkedartifact

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const MaxBytes int64 = 64 << 20

var ErrUnavailable = errors.New("linked artifact source is unavailable")
var ErrNotFound = errors.New("linked artifact is not available")
var ErrTooLarge = errors.New("linked artifact exceeds the preview limit")

// Resolver exposes regular files only from explicitly configured roots. The
// caller must authorize the owner and ask before asking this resolver to open a
// path from decision text.
type Resolver struct {
	roots []string
}

func New(roots ...string) *Resolver {
	clean := make([]string, 0, len(roots))
	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			continue
		}
		abs, err := filepath.Abs(filepath.Clean(root))
		if err == nil {
			resolved, resolveErr := filepath.EvalSymlinks(abs)
			if resolveErr != nil {
				continue
			}
			info, statErr := os.Stat(resolved)
			if statErr != nil || !info.IsDir() {
				continue
			}
			clean = append(clean, filepath.Clean(resolved))
		}
	}
	return &Resolver{roots: clean}
}

func (r *Resolver) Open(reference string) (*os.File, fs.FileInfo, error) {
	if r == nil || len(r.roots) == 0 {
		return nil, nil, ErrUnavailable
	}
	path := strings.TrimSpace(reference)
	path = strings.TrimPrefix(path, "file://")
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, nil, ErrNotFound
		}
		path = filepath.Join(home, filepath.FromSlash(strings.TrimPrefix(path, "~/")))
	}
	if path == "" || strings.IndexByte(path, 0) >= 0 {
		return nil, nil, ErrNotFound
	}
	candidates := []string{}
	if filepath.IsAbs(path) {
		candidates = append(candidates, filepath.Clean(path))
	} else {
		for _, root := range r.roots {
			candidates = append(candidates, filepath.Join(root, filepath.FromSlash(path)))
		}
	}
	for _, candidate := range candidates {
		resolved, err := filepath.EvalSymlinks(candidate)
		if err != nil || !r.contains(resolved) {
			continue
		}
		file, err := os.Open(resolved)
		if err != nil {
			continue
		}
		info, err := file.Stat()
		if err != nil || !info.Mode().IsRegular() {
			_ = file.Close()
			continue
		}
		// Recheck after open so a changed symlink cannot redirect the serving path.
		after, err := filepath.EvalSymlinks(candidate)
		if err != nil || filepath.Clean(after) != filepath.Clean(resolved) || !r.contains(after) {
			_ = file.Close()
			continue
		}
		if info.Size() > MaxBytes {
			_ = file.Close()
			return nil, nil, ErrTooLarge
		}
		return file, info, nil
	}
	return nil, nil, ErrNotFound
}

func (r *Resolver) contains(path string) bool {
	path = filepath.Clean(path)
	for _, root := range r.roots {
		rel, err := filepath.Rel(root, path)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
			return true
		}
	}
	return false
}
