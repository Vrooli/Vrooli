package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DocEntry represents a documentation file or directory
type DocEntry struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	IsDir    bool        `json:"isDir"`
	Children []DocEntry  `json:"children,omitempty"`
}

// DocContent represents the content of a documentation file
type DocContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Title   string `json:"title"`
}

// getDocsRoot returns the absolute path to the docs directory
func getDocsRoot() string {
	// Check for explicit override first
	if override := strings.TrimSpace(os.Getenv("DOCS_ROOT")); override != "" {
		if abs, err := filepath.Abs(override); err == nil {
			return abs
		}
		return override
	}

	// Check for scenario root
	if scenarioRoot := strings.TrimSpace(os.Getenv("SCENARIO_ROOT")); scenarioRoot != "" {
		return filepath.Join(scenarioRoot, "docs")
	}

	// Fallback: API runs from api/ subdirectory, so go up one level
	// Always resolve to absolute path for consistent behavior
	if abs, err := filepath.Abs(filepath.Join("..", "docs")); err == nil {
		return abs
	}
	return filepath.Join("..", "docs")
}

// handleDocsTree returns the hierarchical structure of docs files
func handleDocsTree() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		docsRoot := getDocsRoot()

		// Resolve to absolute path for logging
		absPath, _ := filepath.Abs(docsRoot)
		logStructured("docs_tree_request", map[string]interface{}{
			"docs_root":     docsRoot,
			"absolute_path": absPath,
		})

		// Check if docs directory exists
		if _, err := os.Stat(docsRoot); os.IsNotExist(err) {
			logStructured("docs_directory_not_found", map[string]interface{}{
				"path":  docsRoot,
				"error": err.Error(),
			})
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]DocEntry{})
			return
		}

		entries, err := buildDocsTree(docsRoot, "")
		if err != nil {
			logStructuredError("docs_tree_build_failed", map[string]interface{}{
				"path":  docsRoot,
				"error": err.Error(),
			})
			http.Error(w, "Failed to read docs directory", http.StatusInternalServerError)
			return
		}

		logStructured("docs_tree_success", map[string]interface{}{
			"path":        docsRoot,
			"entry_count": len(entries),
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	}
}

// handleDocsContent returns the content of a specific doc file
func handleDocsContent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		docPath := r.URL.Query().Get("path")
		if docPath == "" {
			http.Error(w, "Missing path parameter", http.StatusBadRequest)
			return
		}

		// Sanitize path to prevent directory traversal
		cleanPath := filepath.Clean(docPath)
		if strings.Contains(cleanPath, "..") {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		// Only allow .md files
		if !strings.HasSuffix(strings.ToLower(cleanPath), ".md") {
			http.Error(w, "Only markdown files are allowed", http.StatusBadRequest)
			return
		}

		docsRoot := getDocsRoot()
		fullPath := filepath.Join(docsRoot, cleanPath)

		// Resolve to absolute for security comparison
		absFullPath, err := filepath.Abs(fullPath)
		if err != nil {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}
		absDocsRoot, _ := filepath.Abs(docsRoot)

		// Verify the file is within docs directory
		if !strings.HasPrefix(absFullPath, absDocsRoot) {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		content, err := os.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}

		// Extract title from first H1 heading or filename
		title := extractTitle(string(content), cleanPath)

		doc := DocContent{
			Path:    cleanPath,
			Content: string(content),
			Title:   title,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(doc)
	}
}

// buildDocsTree recursively builds the docs file tree
func buildDocsTree(root, relativePath string) ([]DocEntry, error) {
	currentPath := filepath.Join(root, relativePath)
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return nil, err
	}

	var result []DocEntry

	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files and non-markdown files (except directories)
		if strings.HasPrefix(name, ".") {
			continue
		}

		entryRelPath := filepath.Join(relativePath, name)

		if entry.IsDir() {
			children, err := buildDocsTree(root, entryRelPath)
			if err != nil {
				continue
			}
			// Only include directories that have markdown files
			if len(children) > 0 {
				result = append(result, DocEntry{
					Name:     name,
					Path:     entryRelPath,
					IsDir:    true,
					Children: children,
				})
			}
		} else if strings.HasSuffix(strings.ToLower(name), ".md") {
			result = append(result, DocEntry{
				Name:  name,
				Path:  entryRelPath,
				IsDir: false,
			})
		}
	}

	// Sort: directories first, then files, alphabetically
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result, nil
}

// extractTitle extracts the title from markdown content or filename
func extractTitle(content, path string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimPrefix(trimmed, "# ")
		}
	}

	// Fallback to filename without extension
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
