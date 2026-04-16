package structure

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	rules "scenario-auditor/rules"
)

var ruleDir = discoverRuleDir()

func newUIViolation(path, message string, severity string) rules.Violation {
	return rules.Violation{
		Severity:       severity,
		Message:        message,
		FilePath:       filepath.ToSlash(path),
		Recommendation: "Ensure the scenario UI follows the required structure and lifecycle patterns.",
	}
}

func resolveScenarioRoot(input string, fallback string) string {
	candidates := []string{strings.TrimSpace(input), strings.TrimSpace(fallback)}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if resolved := resolveCandidate(candidate); resolved != "" {
			return resolved
		}
	}
	return ""
}

func resolveCandidate(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return filepath.Clean(path)
		}
		return ""
	}

	tryPaths := []string{
		path,
		filepath.Join(ruleDir, path),
		filepath.Join(filepath.Dir(ruleDir), path),
		filepath.Join(filepath.Dir(filepath.Dir(ruleDir)), path),
		filepath.Join(ruleDir, "scenarios", path),
		filepath.Join(filepath.Dir(ruleDir), "scenarios", path),
		filepath.Join(filepath.Dir(filepath.Dir(ruleDir)), "scenarios", path),
	}

	for _, envVar := range []string{"VROOLI_ROOT", "APP_ROOT"} {
		if base := strings.TrimSpace(os.Getenv(envVar)); base != "" {
			tryPaths = append(tryPaths,
				filepath.Join(base, path),
				filepath.Join(base, "scenarios", path),
			)
		}
	}

	if wd, err := os.Getwd(); err == nil {
		tryPaths = append(tryPaths,
			filepath.Join(wd, path),
			filepath.Join(wd, "rules", "structure", path),
			filepath.Join(wd, "api", "rules", "structure", path),
			filepath.Join(wd, "scenarios", "scenario-auditor", "api", "rules", "structure", path),
			filepath.Join(wd, "scenarios", path),
		)
	}

	for _, candidate := range tryPaths {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			if abs, err := filepath.Abs(candidate); err == nil {
				return abs
			}
			return candidate
		}
	}

	return ""
}

func discoverRuleDir() string {
	if _, file, _, ok := runtime.Caller(0); ok {
		if strings.HasSuffix(file, "types.go") {
			return filepath.Dir(file)
		}
	}

	if wd, err := os.Getwd(); err == nil {
		if dir := searchRuleDirFrom(wd); dir != "" {
			return dir
		}
	}

	for _, envVar := range []string{"VROOLI_ROOT", "APP_ROOT"} {
		if base := strings.TrimSpace(os.Getenv(envVar)); base != "" {
			if dir := searchRuleDirFrom(base); dir != "" {
				return dir
			}
		}
	}

	return "."
}

func searchRuleDirFrom(start string) string {
	start = strings.TrimSpace(start)
	if start == "" {
		return ""
	}
	current := filepath.Clean(start)
	visited := map[string]struct{}{}
	for {
		if _, seen := visited[current]; seen {
			break
		}
		visited[current] = struct{}{}

		candidates := []string{
			current,
			filepath.Join(current, "rules", "structure"),
			filepath.Join(current, "api", "rules", "structure"),
			filepath.Join(current, "scenarios", "scenario-auditor", "api", "rules", "structure"),
		}

		for _, candidate := range candidates {
			file := filepath.Join(candidate, "ui_structure.go")
			if info, err := os.Stat(file); err == nil && !info.IsDir() {
				return filepath.Dir(file)
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return ""
}
