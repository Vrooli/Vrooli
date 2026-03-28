// Package fileops provides domain-agnostic file management utilities shared
// across backlog items, initiatives, and any other folder-based storage.
//
// This package centralizes recursive directory scanning, path validation,
// protected path guards, file copying, and MIME type detection so that
// multiple domain handlers can reuse the same logic without duplication.
package fileops

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileNode represents a file or directory in a folder tree.
type FileNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Type     string     `json:"type"` // "file" or "directory"
	Size     int64      `json:"size,omitempty,string"`
	Children []FileNode `json:"children,omitempty"`
}

// BuildFileTree recursively reads a directory and returns a sorted tree of
// FileNode entries (directories first, then alphabetical by name).
func BuildFileTree(baseDir, relativePath string) ([]FileNode, error) {
	dirPath := filepath.Join(baseDir, relativePath)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	files := make([]FileNode, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(relativePath, name)
		node := FileNode{
			Name: name,
			Path: path,
		}

		if entry.IsDir() {
			node.Type = "directory"
			children, childErr := BuildFileTree(baseDir, path)
			if childErr == nil {
				node.Children = children
			}
		} else {
			node.Type = "file"
			if info, infoErr := entry.Info(); infoErr == nil {
				node.Size = info.Size()
			}
		}

		files = append(files, node)
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].Type != files[j].Type {
			return files[i].Type == "directory"
		}
		return files[i].Name < files[j].Name
	})

	if files == nil {
		files = []FileNode{}
	}
	return files, nil
}

// NormalizeRelativePath validates and normalizes a relative file path,
// rejecting absolute paths, traversal attempts, and empty values.
func NormalizeRelativePath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("path is required")
	}
	cleaned := filepath.Clean(filepath.FromSlash(trimmed))
	if cleaned == "." {
		return "", errors.New("path must reference a file or directory")
	}
	if filepath.IsAbs(cleaned) {
		return "", errors.New("path must be relative")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", errors.New("path traversal is not allowed")
	}
	return filepath.ToSlash(cleaned), nil
}

// IsProtectedPath returns true if the path targets the given protected file name
// (case-insensitive base name comparison).
func IsProtectedPath(path, protectedFileName string) bool {
	return strings.EqualFold(filepath.Base(path), protectedFileName)
}

// CopyPath copies a file or directory tree from src to dst.
func CopyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode()); err != nil {
			return err
		}
		return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if path == src {
				return nil
			}
			rel, relErr := filepath.Rel(src, path)
			if relErr != nil {
				return relErr
			}
			target := filepath.Join(dst, rel)
			entryInfo, infoErr := d.Info()
			if infoErr != nil {
				return infoErr
			}
			if d.IsDir() {
				return os.MkdirAll(target, entryInfo.Mode())
			}
			return CopyFile(path, target, entryInfo.Mode())
		})
	}
	return CopyFile(src, dst, info.Mode())
}

// CopyFile copies a single file preserving the given mode.
func CopyFile(src, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(mode)
}

// BuildFileNodeFromPath constructs a FileNode from filesystem info. For
// directories, treeBuilder is called to populate children recursively.
func BuildFileNodeFromPath(absolutePath, relativePath string, info os.FileInfo, treeBuilder func(string, string) ([]FileNode, error)) (FileNode, error) {
	normalizedPath := filepath.ToSlash(relativePath)
	if normalizedPath == "." {
		normalizedPath = ""
	}
	node := FileNode{
		Name: filepath.Base(absolutePath),
		Path: normalizedPath,
	}
	if info.IsDir() {
		node.Type = "directory"
		children, err := treeBuilder(absolutePath, "")
		if err != nil {
			return FileNode{}, err
		}
		node.Children = children
		return node, nil
	}
	node.Type = "file"
	node.Size = info.Size()
	return node, nil
}

// GetContentType maps file extensions to MIME content types.
func GetContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".md", ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".js", ".jsx", ".ts", ".tsx":
		return "text/javascript"
	case ".go":
		return "text/x-go"
	case ".py":
		return "text/x-python"
	case ".rs":
		return "text/x-rust"
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".yaml", ".yml":
		return "text/yaml"
	case ".xml":
		return "application/xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	default:
		return "text/plain"
	}
}
