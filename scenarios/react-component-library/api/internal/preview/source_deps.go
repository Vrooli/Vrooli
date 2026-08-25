package preview

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"react-component-library/internal/components"
	"react-component-library/internal/deps"
)

var sourceImportRE = regexp.MustCompile(`(?s)(?:import|export)\s+(?:[^"';]*?\s+from\s+)?["']([^"']+)["']`)

// scanImportedSourceDeclarations follows relative source imports from the
// preview entry and collects the dependency headers carried by those files.
// The registry graph remains the authoritative closure for cross-asset
// validation; this source walk keeps the browser runtime complete when an
// authored relative import predates a registry re-index.
func (s *service) scanImportedSourceDeclarations(versionFiles []components.ComponentVersionFile, sourcePath string) ([]deps.DeclarationFields, error) {
	if strings.TrimSpace(s.repoRoot) == "" || strings.TrimSpace(sourcePath) == "" {
		return nil, nil
	}
	libraryRoot, err := filepath.Abs(filepath.Join(s.repoRoot, "scenarios", "react-component-library", "library"))
	if err != nil {
		return nil, fmt.Errorf("resolve preview source root: %w", err)
	}
	entry, err := filepath.Abs(filepath.Join(libraryRoot, filepath.FromSlash(filepath.Clean(sourcePath))))
	if err != nil || !isWithinRoot(entry, libraryRoot) {
		return nil, fmt.Errorf("preview source path %q escapes the library root", sourcePath)
	}

	virtual := make(map[string]string, len(versionFiles))
	for _, file := range versionFiles {
		path := filepath.Join(filepath.Dir(entry), filepath.FromSlash(file.Path))
		absolute, absErr := filepath.Abs(path)
		if absErr == nil && isWithinRoot(absolute, libraryRoot) {
			virtual[absolute] = file.Content
		}
	}

	seen := map[string]struct{}{}
	var out []deps.DeclarationFields
	var visit func(string, string) error
	visit = func(path, source string) error {
		absolute, absErr := filepath.Abs(path)
		if absErr != nil || !isWithinRoot(absolute, libraryRoot) {
			return nil
		}
		if _, exists := seen[absolute]; exists {
			return nil
		}
		seen[absolute] = struct{}{}

		fields, parseErr := deps.ParseSourceDeclarations(source)
		if parseErr != nil {
			return fmt.Errorf("parse preview dependencies in %s: %w", filepath.ToSlash(absolute), parseErr)
		}
		out = append(out, fields...)
		for _, match := range sourceImportRE.FindAllStringSubmatch(source, -1) {
			importPath := match[1]
			if !strings.HasPrefix(importPath, ".") {
				continue
			}
			resolved, ok := resolveSourceImport(filepath.Dir(absolute), importPath, virtual)
			if !ok {
				continue
			}
			child, readErr := readSourceImport(resolved, virtual)
			if readErr != nil {
				return readErr
			}
			if err := visit(resolved, child); err != nil {
				return err
			}
		}
		return nil
	}

	root, readErr := readSourceImport(entry, virtual)
	if readErr != nil {
		return nil, readErr
	}
	if err := visit(entry, root); err != nil {
		return nil, err
	}
	return out, nil
}

func resolveSourceImport(dir, importPath string, virtual map[string]string) (string, bool) {
	base := filepath.Clean(filepath.Join(dir, filepath.FromSlash(importPath)))
	candidates := []string{base, base + ".ts", base + ".tsx", base + ".js", base + ".jsx"}
	for _, candidate := range candidates {
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if _, ok := virtual[absolute]; ok {
			return absolute, true
		}
		if info, err := os.Stat(absolute); err == nil && !info.IsDir() {
			return absolute, true
		}
	}
	return "", false
}

func readSourceImport(path string, virtual map[string]string) (string, error) {
	if source, ok := virtual[path]; ok {
		return source, nil
	}
	source, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read preview source import %s: %w", filepath.ToSlash(path), err)
	}
	return string(source), nil
}

func isWithinRoot(path, root string) bool {
	return path == root || strings.HasPrefix(path, root+string(os.PathSeparator))
}
