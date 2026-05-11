package lint

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var codeFileExtensions = map[string]struct{}{
	".c": {}, ".cc": {}, ".cpp": {}, ".cs": {}, ".go": {}, ".java": {}, ".js": {}, ".jsx": {},
	".kt": {}, ".mjs": {}, ".mts": {}, ".php": {}, ".py": {}, ".rb": {}, ".rs": {}, ".sh": {},
	".sql": {}, ".swift": {}, ".ts": {}, ".tsx": {},
}

var projectIndicatorFiles = map[string]struct{}{
	"go.mod":           {},
	"package.json":     {},
	"pyproject.toml":   {},
	"requirements.txt": {},
	"setup.py":         {},
	"pytest.ini":       {},
	"mypy.ini":         {},
	"ruff.toml":        {},
	".golangci.yml":    {},
	".golangci.yaml":   {},
}

func discoverComponents(scenarioDir string, settings *Settings) ([]Component, error) {
	components := []Component{discoverRootComponent(scenarioDir)}

	entries, err := os.ReadDir(scenarioDir)
	if err != nil {
		return nil, err
	}

	ignored := ignoredNames(settings)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if ignored[name] {
			continue
		}
		component := Component{
			Name:         name,
			RelativePath: name,
			AbsolutePath: filepath.Join(scenarioDir, name),
		}
		if hasDirectProjectIndicator(component.AbsolutePath) {
			component.CodeBearing, component.CodeEvidence = detectCodeBearing(component.AbsolutePath, false)
		} else {
			component.CodeBearing, component.CodeEvidence = detectCodeBearing(component.AbsolutePath, true)
			nested, err := discoverNestedProjectComponents(scenarioDir, component.RelativePath, ignored)
			if err != nil {
				return nil, err
			}
			components = append(components, nested...)
		}
		components = append(components, component)
	}

	return components, nil
}

func discoverNestedProjectComponents(scenarioDir, parentRel string, ignored map[string]bool) ([]Component, error) {
	var components []Component
	parentAbs := filepath.Join(scenarioDir, parentRel)
	err := filepath.WalkDir(parentAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != parentAbs {
			if ignored[name] || shouldSkipDiscoveryDir(name) {
				return filepath.SkipDir
			}
		}
		if path == parentAbs || !hasDirectProjectIndicator(path) {
			return nil
		}
		rel, err := filepath.Rel(scenarioDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		component := Component{
			Name:         rel,
			RelativePath: rel,
			AbsolutePath: path,
		}
		component.CodeBearing, component.CodeEvidence = detectCodeBearing(component.AbsolutePath, false)
		components = append(components, component)
		return filepath.SkipDir
	})
	return components, err
}

func discoverRootComponent(scenarioDir string) Component {
	component := Component{
		Name:         ".",
		RelativePath: ".",
		AbsolutePath: scenarioDir,
		IsRoot:       true,
	}
	component.CodeBearing, component.CodeEvidence = detectCodeBearing(scenarioDir, true)
	return component
}

func ignoredNames(settings *Settings) map[string]bool {
	names := make(map[string]bool, len(settings.Ignore))
	for _, name := range settings.Ignore {
		names[name] = true
	}
	return names
}

func detectCodeBearing(dir string, topLevelOnly bool) (bool, []string) {
	evidence := make([]string, 0, 4)
	if topLevelOnly {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return false, nil
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if _, ok := projectIndicatorFiles[name]; ok {
				evidence = append(evidence, name)
				continue
			}
			if _, ok := codeFileExtensions[strings.ToLower(filepath.Ext(name))]; ok {
				evidence = append(evidence, name)
			}
		}
		return len(evidence) > 0, slices.Compact(evidence)
	}

	seen := map[string]bool{}
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "dist", "build", "coverage", ".venv", "venv", "__pycache__", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if _, ok := projectIndicatorFiles[name]; ok {
			if !seen[name] {
				evidence = append(evidence, name)
				seen[name] = true
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if _, ok := codeFileExtensions[ext]; ok {
			label := "*" + ext
			if !seen[label] {
				evidence = append(evidence, label)
				seen[label] = true
			}
		}
		return nil
	})

	return len(evidence) > 0, evidence
}

func hasDirectProjectIndicator(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if _, ok := projectIndicatorFiles[entry.Name()]; ok {
			return true
		}
	}
	return false
}

func shouldSkipDiscoveryDir(name string) bool {
	switch name {
	case ".git", "node_modules", "dist", "build", "coverage", ".venv", "venv", "__pycache__", "vendor":
		return true
	default:
		return false
	}
}
