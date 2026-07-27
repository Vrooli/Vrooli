// Package docs owns the filesystem-safe documentation domain primitives.
package docs

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Entry struct {
	Name     string  `json:"name"`
	Path     string  `json:"path"`
	IsDir    bool    `json:"isDir"`
	Children []Entry `json:"children,omitempty"`
}

func IsWithinDirectory(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func ExtractTitle(content, path string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimPrefix(trimmed, "# ")
		}
	}
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func BuildTree(root, relativePath string) ([]Entry, error) {
	entries, err := os.ReadDir(filepath.Join(root, relativePath))
	if err != nil {
		return nil, err
	}
	result := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(relativePath, name)
		if entry.IsDir() {
			children, err := BuildTree(root, path)
			if err == nil && len(children) > 0 {
				result = append(result, Entry{Name: name, Path: path, IsDir: true, Children: children})
			}
		} else if strings.HasSuffix(strings.ToLower(name), ".md") {
			result = append(result, Entry{Name: name, Path: path})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result, nil
}
