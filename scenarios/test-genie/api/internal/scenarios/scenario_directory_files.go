package scenarios

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileNode represents a file or directory within a scenario.
type FileNode struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"` // scenario-relative (e.g., "api/handlers/auth.go")
	IsDir       bool     `json:"isDir"`
	CoveragePct *float64 `json:"coveragePct,omitempty"`
}

// FileListResult bundles file results with metadata about filtered entries.
type FileListResult struct {
	Items       []FileNode `json:"items"`
	HiddenCount int        `json:"hiddenCount"`
}

// FileListOptions controls which portion of the tree is returned.
type FileListOptions struct {
	Path            string // scenario-relative path to list children (defaults to root)
	Search          string // optional substring filter; if set, performs a bounded walk
	Limit           int    // optional cap on returned entries (default 200, max 500)
	IncludeHidden   bool   // when true, do not filter ignored names or extensions
	IncludeCoverage bool   // when true, attach coverage percentages when available
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

	coverage := map[string]float64{}
	if opts.IncludeCoverage {
		coverage = loadCoverageMap(root)
	}

	if search != "" {
		return s.searchScenarioFiles(ctx, root, normalizedPath, search, limit, opts.IncludeHidden, coverage)
	}

	return s.listScenarioPath(ctx, root, normalizedPath, limit, opts.IncludeHidden, coverage)
}

func (s *ScenarioDirectoryService) listScenarioPath(ctx context.Context, root, relative string, limit int, includeHidden bool, coverage map[string]float64) (FileListResult, error) {
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
			node := FileNode{
				Name:  base,
				Path:  base,
				IsDir: true,
			}
			attachCoverage(&node, coverage)
			nodes = append(nodes, node)
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
		node := FileNode{
			Name:  name,
			Path:  filepath.ToSlash(entryPath),
			IsDir: entry.IsDir(),
		}
		attachCoverage(&node, coverage)
		nodes = append(nodes, node)
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

func (s *ScenarioDirectoryService) searchScenarioFiles(ctx context.Context, root, relative, term string, limit int, includeHidden bool, coverage map[string]float64) (FileListResult, error) {
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
				node := FileNode{
					Name:  d.Name(),
					Path:  filepath.ToSlash(rel),
					IsDir: d.IsDir(),
				}
				attachCoverage(&node, coverage)
				nodes = append(nodes, node)
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

// loadCoverageMap collects coverage percentages from common summary files (NYC/Vitest-style)
// and returns a map keyed by scenario-relative paths using forward slashes.
func loadCoverageMap(scenarioDir string) map[string]float64 {
	coverage := make(map[string]float64)
	summaries := []string{
		filepath.Join(scenarioDir, "coverage", "coverage-summary.json"),
		filepath.Join(scenarioDir, "ui", "coverage", "coverage-summary.json"),
	}
	for _, path := range summaries {
		mergeCoverageSummary(coverage, scenarioDir, path)
	}
	addDirCoverage(coverage)
	return coverage
}

type coverageSummaryEntry struct {
	Lines struct {
		Pct float64 `json:"pct"`
	} `json:"lines"`
	Statements struct {
		Pct float64 `json:"pct"`
	} `json:"statements"`
	Branches struct {
		Pct float64 `json:"pct"`
	} `json:"branches"`
	Functions struct {
		Pct float64 `json:"pct"`
	} `json:"functions"`
}

func mergeCoverageSummary(dest map[string]float64, scenarioDir, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}

	for key, entry := range raw {
		if strings.EqualFold(strings.TrimSpace(key), "total") {
			continue
		}
		var parsed coverageSummaryEntry
		if err := json.Unmarshal(entry, &parsed); err != nil {
			continue
		}
		pct := firstPositive(parsed.Lines.Pct, parsed.Statements.Pct, parsed.Branches.Pct, parsed.Functions.Pct)
		if pct <= 0 {
			continue
		}
		normalized := normalizeCoveragePath(key, scenarioDir)
		if normalized == "" {
			continue
		}
		dest[normalized] = pct
	}
}

func addDirCoverage(files map[string]float64) {
	if len(files) == 0 {
		return
	}
	type accum struct {
		sum   float64
		count int
	}
	dirs := make(map[string]accum)
	for path, pct := range files {
		if pct <= 0 {
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(path))
		for dir != "" && dir != "." {
			entry := dirs[dir]
			entry.sum += pct
			entry.count++
			dirs[dir] = entry
			parent := filepath.ToSlash(filepath.Dir(dir))
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	for dir, agg := range dirs {
		if agg.count == 0 {
			continue
		}
		if _, exists := files[dir]; exists {
			// Do not overwrite explicit coverage entries if present.
			continue
		}
		files[dir] = agg.sum / float64(agg.count)
	}
}

func normalizeCoveragePath(path, scenarioDir string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return ""
	}

	// If absolute, attempt to make it scenario-relative.
	if filepath.IsAbs(path) {
		if rel, err := filepath.Rel(filepath.Clean(scenarioDir), filepath.Clean(path)); err == nil {
			if !strings.HasPrefix(rel, "..") {
				path = rel
			}
		}
	}

	path = strings.TrimPrefix(path, "./")
	path = strings.TrimLeft(path, "/")
	return filepath.ToSlash(path)
}

func firstPositive(vals ...float64) float64 {
	for _, v := range vals {
		if v > 0 {
			return v
		}
	}
	return 0
}

func attachCoverage(node *FileNode, coverage map[string]float64) {
	if node == nil || coverage == nil {
		return
	}
	if pct, ok := coverage[node.Path]; ok && pct > 0 {
		node.CoveragePct = &pct
	}
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
