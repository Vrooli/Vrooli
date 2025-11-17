package validator

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var (
	defaultSelectorOnce sync.Once
	defaultSelectors    map[string]struct{}
	selectorCache       sync.Map // map[string]map[string]struct{}
)

var additionalSelectorAllowlist = []string{
	"execution-progress",
	"execution-status-running",
	"execution-status-completed",
	"execution-status-failed",
	"execution-status-cancelled",
	"ai-example-prompt-0",
	"breadcrumb-project",
	"execution-card",
	"execution-screenshots",
	"form-error",
	"node-palette-category-navigation-toggle",
	"node-palette-navigate-card",
	"node-palette-navigate-favorite-button",
	"node-property-url-input",
	"project-tab-executions",
	"replay-player",
	"replay-screenshot",
	"replay-timeline",
	"toolbar-lock-button",
	"url-mode-button",
}

var dataTestIDPattern = regexp.MustCompile(`(?i)data-testid\s*=\s*(?:"([^"]+)"|'([^']+)')`)

func getSelectorRegistry(root string) map[string]struct{} {
	key := strings.TrimSpace(root)
	if key == "" {
		defaultSelectorOnce.Do(func() {
			selectorRoot := strings.TrimSpace(os.Getenv("BAS_SELECTOR_ROOT"))
			if selectorRoot == "" {
				selectorRoot = detectDefaultSelectorRoot()
			}
			defaultSelectors = loadSelectorsFrom(selectorRoot)
		})
		return defaultSelectors
	}
	if abs, err := filepath.Abs(key); err == nil {
		key = abs
	}
	if cached, ok := selectorCache.Load(key); ok {
		if registry, okCast := cached.(map[string]struct{}); okCast {
			return registry
		}
	}
	registry := loadSelectorsFrom(key)
	selectorCache.Store(key, registry)
	return registry
}

func loadSelectorsFrom(root string) map[string]struct{} {
	selectors := map[string]struct{}{}
	if strings.TrimSpace(root) != "" {
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if strings.HasPrefix(name, ".") || name == "dist" || name == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			switch filepath.Ext(path) {
			case ".ts", ".tsx", ".js", ".jsx":
				content, readErr := os.ReadFile(path)
				if readErr != nil {
					return nil
				}
				matches := dataTestIDPattern.FindAllSubmatch(content, -1)
				for _, match := range matches {
					value := string(match[1])
					if value == "" && len(match) > 2 {
						value = string(match[2])
					}
					value = strings.TrimSpace(value)
					if value == "" {
						continue
					}
					selectors[value] = struct{}{}
				}
			}
			return nil
		})
	}
	for _, extra := range additionalSelectorAllowlist {
		if strings.TrimSpace(extra) == "" {
			continue
		}
		selectors[extra] = struct{}{}
	}
	return selectors
}

func detectDefaultSelectorRoot() string {
	if cwd, err := os.Getwd(); err == nil {
		if root := discoverSelectorRootFrom(cwd); root != "" {
			return root
		}
	}
	if exe, err := os.Executable(); err == nil {
		if root := discoverSelectorRootFrom(filepath.Dir(exe)); root != "" {
			return root
		}
	}
	// final fallback mirrors legacy behaviour (useful when running go test from api dir)
	candidate := filepath.Join("..", "ui", "src")
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}
	return ""
}

func discoverSelectorRootFrom(start string) string {
	if start == "" {
		return ""
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		abs = start
	}
	visited := map[string]struct{}{}
	for dir := abs; ; dir = filepath.Dir(dir) {
		if _, seen := visited[dir]; seen {
			break
		}
		visited[dir] = struct{}{}
		if filepath.Base(dir) == "browser-automation-studio" {
			candidate := filepath.Join(dir, "ui", "src")
			if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
				return candidate
			}
		}
		scenarioCandidate := filepath.Join(dir, "scenarios", "browser-automation-studio", "ui", "src")
		if info, statErr := os.Stat(scenarioCandidate); statErr == nil && info.IsDir() {
			return scenarioCandidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return ""
}

func lintSelectorValue(selector, pointer, nodeID, nodeType string, registry map[string]struct{}) []Issue {
	if len(registry) == 0 {
		return nil
	}
	matches := dataTestIDPattern.FindAllStringSubmatch(selector, -1)
	if len(matches) == 0 {
		return nil
	}
	var issues []Issue
	for _, match := range matches {
		value := match[1]
		if value == "" && len(match) > 2 {
			value = match[2]
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := registry[value]; !ok {
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Code:     "WF_SELECTOR_UNKNOWN_TESTID",
				Message:  "Selector references data-testid '" + value + "' which was not found in the UI source tree",
				NodeID:   nodeID,
				NodeType: nodeType,
				Pointer:  pointer,
				Hint:     "Ensure the selector matches an existing data-testid attribute, or update the workflow",
			})
		}
	}
	return issues
}
