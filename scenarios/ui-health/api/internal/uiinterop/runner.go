package uiinterop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// RunAll runs all rules matching the scenario's tech stack and returns results.
func RunAll(scenarioRoot, scenarioName string) []RuleResult {
	stack := EnrichTechStack(scenarioRoot)
	rules := ForTechStack(stack)
	sources := WalkUISourceSet(scenarioRoot, "ui")

	ctx := CheckContext{
		ScenarioRoot: scenarioRoot,
		TechStack:    stack,
		ScenarioName: scenarioName,
		Sources:      sources.Production,
		TestSources:  sources.Tests,
	}

	results := make([]RuleResult, 0, len(rules))
	for _, r := range rules {
		result := r.Check(ctx)
		result.RuleID = r.Def.ID
		applyRuleDefSeverity(&result, r.Def)
		results = append(results, result)
	}
	return results
}

func applyRuleDefSeverity(result *RuleResult, def RuleDef) {
	if result == nil || def.Severity == "" {
		return
	}
	for i := range result.Violations {
		result.Violations[i].Severity = def.Severity
	}
}

// WalkUISource walks the given subtree under scenarioRoot (e.g. "ui" or
// "ui/src"), yielding every production source file with a scannable extension.
// Test files, snapshot/fixture dirs, build output and dependency dirs are
// skipped. It returns nil when the subtree does not exist, so callers can treat
// "no UI" as "skip".
func WalkUISource(scenarioRoot, subdir string) []SourceFile {
	return WalkUISourceSet(scenarioRoot, subdir).Production
}

// WalkUITestSource walks a UI subtree and returns scannable test source files.
func WalkUITestSource(scenarioRoot, subdir string) []SourceFile {
	return WalkUISourceSet(scenarioRoot, subdir).Tests
}

// WalkUISourceSet walks a UI subtree once and splits scannable files into
// production and test buckets.
func WalkUISourceSet(scenarioRoot, subdir string) SourceSet {
	base := filepath.Join(scenarioRoot, filepath.FromSlash(subdir))
	info, err := os.Stat(base)
	if err != nil || !info.IsDir() {
		return SourceSet{}
	}

	var set SourceSet
	_ = filepath.WalkDir(base, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if _, skip := skipDirectories[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if _, ok := scanExtensions[filepath.Ext(d.Name())]; !ok {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		rel, relErr := filepath.Rel(scenarioRoot, path)
		if relErr != nil {
			rel = path
		}
		file := SourceFile{
			RelPath: filepath.ToSlash(rel),
			AbsPath: path,
			Content: string(data),
		}
		if IsTestFile(d.Name()) {
			set.Tests = append(set.Tests, file)
		} else {
			set.Production = append(set.Production, file)
		}
		return nil
	})
	return set
}

// IsTestFile returns true for test files (*.test.ts, *.spec.tsx, etc.) and
// test-harness setup files (test-setup.ts, *.setup.ts, setup-tests.ts) that
// should be excluded from production-code scans.
func IsTestFile(name string) bool {
	lower := strings.ToLower(name)
	ext := filepath.Ext(lower)
	base := strings.TrimSuffix(lower, ext)
	if strings.HasSuffix(base, ".test") || strings.HasSuffix(base, ".spec") ||
		strings.HasSuffix(base, "_test") || strings.HasSuffix(base, "_spec") {
		return true
	}
	if strings.HasSuffix(base, ".setup") || strings.HasSuffix(base, "-setup") {
		return true
	}
	switch base {
	case "test-setup", "setup-tests", "setuptests", "test-utils", "testutils":
		return true
	}
	return false
}

// skipDirectories are directories to skip during file scanning.
var skipDirectories = map[string]struct{}{
	".git":          {},
	".hg":           {},
	".svn":          {},
	".cache":        {},
	".next":         {},
	".nuxt":         {},
	"dist":          {},
	"build":         {},
	"node_modules":  {},
	"vendor":        {},
	".venv":         {},
	".idea":         {},
	".vscode":       {},
	"coverage":      {},
	"tmp":           {},
	"__tests__":     {},
	"__mocks__":     {},
	"__fixtures__":  {},
	"__snapshots__": {},
}

// scanExtensions are file extensions to scan in source directories.
var scanExtensions = map[string]struct{}{
	".cjs":    {},
	".css":    {},
	".go":     {},
	".htm":    {},
	".html":   {},
	".js":     {},
	".jsx":    {},
	".less":   {},
	".mjs":    {},
	".sass":   {},
	".scss":   {},
	".svelte": {},
	".ts":     {},
	".tsx":    {},
	".vue":    {},
}

// EnrichTechStack reads ui/package.json and derives synthetic tech stack signals.
func EnrichTechStack(scenarioRoot string) []string {
	stack := []string{}

	pkgPath := filepath.Join(scenarioRoot, "ui", "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return stack
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return stack
	}

	allDeps := map[string]bool{}
	for k := range pkg.Dependencies {
		allDeps[k] = true
	}
	for k := range pkg.DevDependencies {
		allDeps[k] = true
	}

	if allDeps["@vrooli/iframe-bridge"] {
		stack = append(stack, "iframe-bridge")
	}
	if allDeps["@vrooli/api-base"] {
		stack = append(stack, "api-base")
	}
	if allDeps["react"] || allDeps["react-dom"] {
		stack = append(stack, "React")
	}
	if allDeps["vite"] {
		stack = append(stack, "Vite")
	}

	return stack
}
