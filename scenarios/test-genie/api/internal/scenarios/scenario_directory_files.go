package scenarios

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileNode represents a file or directory within a scenario.
type FileNode struct {
	Name  string `json:"name"`
	Path  string `json:"path"` // scenario-relative (e.g., "api/handlers/auth.go")
	IsDir bool   `json:"isDir"`
}

// FileListResult bundles file results with metadata about filtered entries.
type FileListResult struct {
	Items       []FileNode `json:"items"`
	HiddenCount int        `json:"hiddenCount"`
}

// FileListOptions controls which portion of the tree is returned.
type FileListOptions struct {
	Path          string // scenario-relative path to list children (defaults to root)
	Search        string // optional substring filter; if set, performs a bounded walk
	Limit         int    // optional cap on returned entries (default 200, max 500)
	IncludeHidden bool   // when true, do not filter ignored names or extensions
}

const (
	defaultFileLimit = 200
	maxFileLimit     = 500
)

var allowedScenarioRoots = []string{"api", "ui"}
var ignoredNames = []string{
	"node_modules",
	".git",
	".husky",
	".turbo",
	"dist",
	"build",
	".next",
	".vite",
	"coverage",
	".cache",
	".pnpm-store",
	"docs",
	"doc",
	"storybook-static",
	"tmp",
	"temp",
	".idea",
	".vscode",
	".DS_Store",
	".venv",
	"venv",
	".pytest_cache",
	"logs",
	"artifacts",
	"bin",
	"obj",
	"vendor",
}

// ListFiles returns a shallow listing or search results for a scenario's api/ui directories.
func (s *ScenarioDirectoryService) ListFiles(ctx context.Context, scenario string, opts FileListOptions) ([]FileNode, error) {
	result, err := s.ListFilesWithMeta(ctx, scenario, opts)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// ListFilesWithMeta returns files plus metadata about filtered entries.
func (s *ScenarioDirectoryService) ListFilesWithMeta(ctx context.Context, scenario string, opts FileListOptions) (FileListResult, error) {
	if s == nil {
		return FileListResult{}, fmt.Errorf("scenario directory service unavailable")
	}
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return FileListResult{}, fmt.Errorf("scenario name is required")
	}
	if strings.TrimSpace(s.scenariosRoot) == "" {
		return FileListResult{}, fmt.Errorf("scenarios root is not configured")
	}

	root := filepath.Join(s.scenariosRoot, scenario)
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return FileListResult{}, fmt.Errorf("scenario directory not found")
		}
		return FileListResult{}, fmt.Errorf("failed to access scenario directory")
	}
	if !info.IsDir() {
		return FileListResult{}, fmt.Errorf("scenario path is not a directory")
	}

	limit := opts.Limit
	if limit <= 0 || limit > maxFileLimit {
		limit = defaultFileLimit
	}

	normalizedPath := strings.Trim(strings.TrimSpace(opts.Path), "/")
	search := strings.ToLower(strings.TrimSpace(opts.Search))

	if search != "" {
		return s.searchScenarioFiles(ctx, root, normalizedPath, search, limit, opts.IncludeHidden)
	}

	return s.listScenarioPath(ctx, root, normalizedPath, limit, opts.IncludeHidden)
}

func (s *ScenarioDirectoryService) listScenarioPath(ctx context.Context, root, relative string, limit int, includeHidden bool) (FileListResult, error) {
	// Root listing: show allowed roots that exist
	if relative == "" {
		nodes := make([]FileNode, 0, len(allowedScenarioRoots))
		for _, base := range allowedScenarioRoots {
			select {
			case <-ctx.Done():
				return FileListResult{}, ctx.Err()
			default:
			}
			dir := filepath.Join(root, base)
			info, err := os.Stat(dir)
			if err != nil || !info.IsDir() {
				continue
			}
			nodes = append(nodes, FileNode{
				Name:  base,
				Path:  base,
				IsDir: true,
			})
		}
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })
		return FileListResult{Items: nodes}, nil
	}

	// Ensure the requested path stays within api/ or ui/
	if !isAllowedScenarioPath(relative) {
		return FileListResult{}, fmt.Errorf("path must be under api/ or ui/")
	}

	target := filepath.Join(root, filepath.Clean(relative))
	rootClean := filepath.Clean(root)
	targetClean := filepath.Clean(target)
	if targetClean != rootClean && !strings.HasPrefix(targetClean, rootClean+string(os.PathSeparator)) {
		return FileListResult{}, fmt.Errorf("invalid path")
	}
	info, err := os.Stat(targetClean)
	if err != nil {
		if os.IsNotExist(err) {
			return FileListResult{}, fmt.Errorf("path not found")
		}
		return FileListResult{}, fmt.Errorf("failed to access path")
	}
	if !info.IsDir() {
		// If a file is requested directly, return its parent listing to support lazy UIs.
		targetClean = filepath.Dir(targetClean)
		relative = filepath.Dir(relative)
	}

	entries, err := os.ReadDir(targetClean)
	if err != nil {
		return FileListResult{}, fmt.Errorf("failed to read directory")
	}

	nodes := make([]FileNode, 0, len(entries))
	hidden := 0
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return FileListResult{}, ctx.Err()
		default:
		}
		name := entry.Name()
		if !includeHidden && strings.HasPrefix(name, ".") { // hide dotfiles
			hidden++
			continue
		}
		if !includeHidden && (isIgnoredName(name) || hasIgnoredExtension(name)) {
			hidden++
			continue
		}
		entryPath := filepath.Join(relative, name)
		if !isAllowedScenarioPath(entryPath) {
			continue
		}
		nodes = append(nodes, FileNode{
			Name:  name,
			Path:  filepath.ToSlash(entryPath),
			IsDir: entry.IsDir(),
		})
		if len(nodes) >= limit {
			break
		}
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir == nodes[j].IsDir {
			return nodes[i].Name < nodes[j].Name
		}
		return nodes[i].IsDir && !nodes[j].IsDir
	})

	return FileListResult{Items: nodes, HiddenCount: hidden}, nil
}

func (s *ScenarioDirectoryService) searchScenarioFiles(ctx context.Context, root, relative, term string, limit int, includeHidden bool) (FileListResult, error) {
	term = strings.ToLower(term)
	var searchRoots []string
	if relative != "" {
		if !isAllowedScenarioPath(relative) {
			return FileListResult{}, fmt.Errorf("path must be under api/ or ui/")
		}
		rootClean := filepath.Clean(root)
		target := filepath.Join(rootClean, filepath.Clean(relative))
		if target != rootClean && !strings.HasPrefix(target, rootClean+string(os.PathSeparator)) {
			return FileListResult{}, fmt.Errorf("invalid path")
		}
		searchRoots = []string{target}
	} else {
		for _, base := range allowedScenarioRoots {
			searchRoots = append(searchRoots, filepath.Join(root, base))
		}
	}

	nodes := make([]FileNode, 0, limit)
	hidden := 0

	for _, start := range searchRoots {
		select {
		case <-ctx.Done():
			return FileListResult{}, ctx.Err()
		default:
		}
		if len(nodes) >= limit {
			break
		}
		_ = filepath.WalkDir(start, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if len(nodes) >= limit {
				return fs.SkipAll
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			if rel == "." {
				return nil
			}
			if !isAllowedScenarioPath(rel) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !includeHidden && (isIgnoredName(d.Name()) || hasIgnoredExtension(d.Name())) {
				hidden++
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			name := strings.ToLower(d.Name())
			relLower := strings.ToLower(filepath.ToSlash(rel))
			if strings.Contains(name, term) || strings.Contains(relLower, term) {
				nodes = append(nodes, FileNode{
					Name:  d.Name(),
					Path:  filepath.ToSlash(rel),
					IsDir: d.IsDir(),
				})
				if len(nodes) >= limit {
					return fs.SkipAll
				}
			}
			return nil
		})
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir == nodes[j].IsDir {
			return nodes[i].Path < nodes[j].Path
		}
		return nodes[i].IsDir && !nodes[j].IsDir
	})

	return FileListResult{Items: nodes, HiddenCount: hidden}, nil
}

func isAllowedScenarioPath(path string) bool {
	clean := strings.Trim(strings.TrimSpace(filepath.ToSlash(path)), "/")
	if clean == "" {
		return true
	}
	for _, base := range allowedScenarioRoots {
		if clean == base || strings.HasPrefix(clean, base+"/") {
			return true
		}
	}
	return false
}

func isIgnoredName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, ig := range ignoredNames {
		if lower == strings.ToLower(ig) {
			return true
		}
	}
	return false
}

func hasIgnoredExtension(name string) bool {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".json"),
		strings.HasSuffix(lower, ".yaml"),
		strings.HasSuffix(lower, ".yml"),
		strings.HasSuffix(lower, ".d.ts"),
		strings.HasSuffix(lower, ".mod"),
		strings.HasSuffix(lower, ".sum"),
		strings.HasSuffix(lower, ".exe"),
		strings.HasSuffix(lower, ".dll"),
		strings.HasSuffix(lower, ".so"),
		strings.HasSuffix(lower, ".dylib"),
		strings.HasSuffix(lower, ".bin"),
		strings.HasSuffix(lower, ".dat"),
		strings.HasSuffix(lower, ".wasm"),
		strings.HasSuffix(lower, ".o"),
		strings.HasSuffix(lower, ".a"),
		strings.HasSuffix(lower, "tsconfig.json"),
		strings.HasSuffix(lower, "tsconfig.app.json"),
		strings.HasSuffix(lower, "tsconfig.base.json"),
		strings.HasSuffix(lower, "tsconfig.build.json"),
		strings.HasSuffix(lower, "tsconfig.test.json"),
		strings.HasSuffix(lower, "vite.config.ts"),
		strings.HasSuffix(lower, "vite.config.js"),
		strings.HasSuffix(lower, "vitest.config.ts"),
		strings.HasSuffix(lower, "vitest.config.js"):
		return true
	default:
		return false
	}
}
