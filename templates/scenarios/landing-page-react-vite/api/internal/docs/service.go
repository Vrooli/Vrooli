// Package docs is a filesystem-backed markdown docs browser for the admin
// portal: it derives a tree of markdown files (and the directories that contain
// them) from the scenario's docs directory and reads document bodies by
// root-relative path with traversal guarding. The Connect handler in
// handlers/docs is a thin adapter over this Service.
package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry is a node in the docs tree: a markdown file, or a directory that
// contains markdown descendants.
type Entry struct {
	Name     string
	Path     string
	IsDir    bool
	Children []Entry
}

// Content is a single document's body plus its derived title.
type Content struct {
	Path    string
	Content string
	Title   string
}

// Service reads the docs tree and document bodies from a root directory.
type Service struct {
	root string
}

// NewService constructs the docs Service. When root is empty it is resolved
// from DOCS_ROOT, then SCENARIO_ROOT/docs, then ../docs.
func NewService(root string) *Service {
	if strings.TrimSpace(root) == "" {
		root = resolveRoot()
	}
	return &Service{root: root}
}

func resolveRoot() string {
	if override := strings.TrimSpace(os.Getenv("DOCS_ROOT")); override != "" {
		if abs, err := filepath.Abs(override); err == nil {
			return abs
		}
		return override
	}
	if scenarioRoot := strings.TrimSpace(os.Getenv("SCENARIO_ROOT")); scenarioRoot != "" {
		return filepath.Join(scenarioRoot, "docs")
	}
	if abs, err := filepath.Abs(filepath.Join("..", "docs")); err == nil {
		return abs
	}
	return filepath.Join("..", "docs")
}

// Tree returns the docs tree, or an empty slice when the root is absent.
func (s *Service) Tree() ([]Entry, error) {
	if _, err := os.Stat(s.root); os.IsNotExist(err) {
		return []Entry{}, nil
	}
	return buildTree(s.root, "")
}

// Read returns a document's content by root-relative path, guarding against
// traversal and non-markdown paths.
func (s *Service) Read(docPath string) (*Content, error) {
	if strings.TrimSpace(docPath) == "" {
		return nil, &InvalidPathError{Reason: "path is required"}
	}
	cleanPath := filepath.Clean(docPath)
	if strings.Contains(cleanPath, "..") {
		return nil, &InvalidPathError{Reason: "invalid path"}
	}
	if !strings.HasSuffix(strings.ToLower(cleanPath), ".md") {
		return nil, &InvalidPathError{Reason: "only markdown files are allowed"}
	}

	fullPath := filepath.Join(s.root, cleanPath)
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, &InvalidPathError{Reason: "invalid path"}
	}
	absRoot, _ := filepath.Abs(s.root)
	if !strings.HasPrefix(absFullPath, absRoot) {
		return nil, &InvalidPathError{Reason: "invalid path"}
	}

	body, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &NotFoundError{Path: cleanPath}
		}
		return nil, fmt.Errorf("read file: %w", err)
	}
	return &Content{Path: cleanPath, Content: string(body), Title: extractTitle(string(body), cleanPath)}, nil
}

// InvalidPathError is a bad-request-level failure for docs reads.
type InvalidPathError struct{ Reason string }

func (e *InvalidPathError) Error() string { return e.Reason }

// NotFoundError indicates a requested document does not exist.
type NotFoundError struct{ Path string }

func (e *NotFoundError) Error() string { return fmt.Sprintf("document %q not found", e.Path) }

func buildTree(root, relativePath string) ([]Entry, error) {
	entries, err := os.ReadDir(filepath.Join(root, relativePath))
	if err != nil {
		return nil, err
	}

	var result []Entry
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		entryRelPath := filepath.Join(relativePath, name)
		if entry.IsDir() {
			children, err := buildTree(root, entryRelPath)
			if err != nil {
				continue
			}
			if len(children) > 0 {
				result = append(result, Entry{Name: name, Path: entryRelPath, IsDir: true, Children: children})
			}
		} else if strings.HasSuffix(strings.ToLower(name), ".md") {
			result = append(result, Entry{Name: name, Path: entryRelPath, IsDir: false})
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

func extractTitle(content, path string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimPrefix(trimmed, "# ")
		}
	}
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
