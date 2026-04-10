package docsearch

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Service struct {
	scenariosRoot string
	repoRoot      string
	Semantic      SemanticSearcher
}

type docFile struct {
	Path     string
	Scenario string
	RelBase  string
}

// NewService initializes the documentation search service.
func NewService(scenariosRoot string) (*Service, error) {
	scenariosRoot = strings.TrimSpace(scenariosRoot)
	if scenariosRoot == "" {
		return nil, ErrScenarioRootEmpty
	}
	info, err := os.Stat(scenariosRoot)
	if err != nil || !info.IsDir() {
		return nil, ErrScenarioRootEmpty
	}
	repoRoot := filepath.Dir(scenariosRoot)
	if repoRoot == scenariosRoot {
		repoRoot = ""
	}
	return &Service{scenariosRoot: scenariosRoot, repoRoot: repoRoot}, nil
}

func (s *Service) scenarioPath(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "", ErrScenarioRequired
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", ErrScenarioRequired
	}
	path := filepath.Join(s.scenariosRoot, name)
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", ErrScenarioRequired
	}
	return path, nil
}

func (s *Service) resolveBasePath(base string) (string, error) {
	if strings.TrimSpace(base) == "" {
		return "", ErrBasePathRequired
	}
	base = filepath.Clean(base)
	var abs string
	if filepath.IsAbs(base) {
		abs = base
	} else if s.repoRoot != "" {
		abs = filepath.Join(s.repoRoot, base)
	} else {
		abs = base
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", ErrBasePathRequired
	}
	if !info.IsDir() {
		return "", ErrBasePathRequired
	}
	if s.repoRoot != "" {
		rel, err := filepath.Rel(s.repoRoot, abs)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", ErrBasePathInvalid
		}
	}
	return abs, nil
}

func (s *Service) repoRelative(path string) string {
	if s.repoRoot == "" {
		return filepath.ToSlash(filepath.Clean(path))
	}
	rel, err := filepath.Rel(s.repoRoot, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(filepath.Clean(path))
	}
	return filepath.ToSlash(rel)
}

func (s *Service) relPath(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(target))
	}
	return filepath.ToSlash(rel)
}

func isDocFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".json" || ext == ".txt"
}

func (s *Service) collectDocFiles(ctx context.Context, scope, scenario, basePath string) ([]docFile, error) {
	switch scope {
	case ScopeScenario:
		return s.collectScenarioDocs(ctx, scenario)
	case ScopePath:
		root, err := s.resolveBasePath(basePath)
		if err != nil {
			return nil, err
		}
		return s.collectPathDocs(ctx, root)
	case ScopeGlobal:
		return s.collectGlobalDocs(ctx)
	default:
		return nil, ErrScopeInvalid
	}
}

func (s *Service) collectScenarioDocs(ctx context.Context, scenario string) ([]docFile, error) {
	scenarioPath, err := s.scenarioPath(scenario)
	if err != nil {
		return nil, err
	}
	var out []docFile
	rootDocs := []string{"README.md", "PRD.md"}
	for _, name := range rootDocs {
		candidate := filepath.Join(scenarioPath, name)
		if exists(candidate) {
			out = append(out, docFile{Path: candidate, Scenario: scenario, RelBase: scenarioPath})
		}
	}
	docsRoot := filepath.Join(scenarioPath, "docs")
	files, err := walkDocsDir(ctx, docsRoot, scenario, scenarioPath)
	if err != nil {
		return nil, err
	}
	return append(out, files...), nil
}

func (s *Service) collectGlobalDocs(ctx context.Context) ([]docFile, error) {
	var out []docFile
	if s.repoRoot != "" {
		rootDocs := []string{"README.md", "PRD.md"}
		for _, name := range rootDocs {
			candidate := filepath.Join(s.repoRoot, name)
			if exists(candidate) {
				out = append(out, docFile{Path: candidate, RelBase: s.repoRoot})
			}
		}
		docsRoot := filepath.Join(s.repoRoot, "docs")
		files, err := walkDocsDir(ctx, docsRoot, "", s.repoRoot)
		if err != nil {
			return nil, err
		}
		out = append(out, files...)
	}

	scenarios, err := s.listScenarioNames()
	if err != nil {
		return nil, err
	}
	for _, scenario := range scenarios {
		files, err := s.collectScenarioDocs(ctx, scenario)
		if err != nil {
			continue
		}
		out = append(out, files...)
	}
	return out, nil
}

func (s *Service) collectPathDocs(ctx context.Context, root string) ([]docFile, error) {
	return walkDocsDir(ctx, root, "", root)
}

func (s *Service) listScenarioNames() ([]string, error) {
	entries, err := os.ReadDir(s.scenariosRoot)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func walkDocsDir(ctx context.Context, root, scenario, relBase string) ([]docFile, error) {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, nil
	}
	var out []docFile
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isDocFile(path) {
			return nil
		}
		out = append(out, docFile{Path: path, Scenario: scenario, RelBase: relBase})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func shouldSkipDir(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "node_modules", "dist", "build", "coverage", "logs", "data", "tmp", "vendor":
		return true
	default:
		return false
	}
}

func exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
