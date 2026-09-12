package dochealth

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var defaultSkipDirs = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	"dist":         {},
	"build":        {},
	".turbo":       {},
	".next":        {},
	".pnpm-store":  {},
	"coverage":     {},
	".cache":       {},
	"tmp":          {},
	"logs":         {},
	"vendor":       {},
	"__pycache__":  {},
	".venv":        {},
	"venv":         {},
	"target":       {},
}

func collectMarkdownFiles(scenarioDir string, cfg effective) ([]string, error) {
	var files []string
	err := filepath.WalkDir(scenarioDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if shouldExcludePath(scenarioDir, p, d.IsDir(), cfg) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if _, skip := defaultSkipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".md") || strings.HasSuffix(d.Name(), ".mdx") {
			files = append(files, p)
		}
		return nil
	})
	return files, err
}

func shouldExcludePath(root, targetPath string, isDir bool, cfg effective) bool {
	rel, err := filepath.Rel(root, targetPath)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == "" {
		return false
	}

	for _, excluded := range cfg.scanExcludeDirs {
		excluded = filepath.ToSlash(strings.TrimSpace(excluded))
		excluded = strings.Trim(excluded, "/")
		if excluded == "" {
			continue
		}
		if strings.Contains(excluded, "/") {
			if rel == excluded || strings.HasPrefix(rel, excluded+"/") {
				return true
			}
			continue
		}
		for _, seg := range strings.Split(rel, "/") {
			if seg == excluded {
				return true
			}
		}
	}

	for _, glob := range cfg.scanExcludeGlobs {
		glob = filepath.ToSlash(strings.TrimSpace(glob))
		if glob == "" {
			continue
		}
		if doublestarMatch(glob, rel) {
			return true
		}
		if isDir && doublestarMatch(glob, rel+"/") {
			return true
		}
	}

	return false
}

func doublestarMatch(glob, value string) bool {
	quoted := regexp.QuoteMeta(glob)
	quoted = strings.ReplaceAll(quoted, `\*\*`, "<<<DOUBLESTAR>>>")
	quoted = strings.ReplaceAll(quoted, `\*`, `[^/]*`)
	quoted = strings.ReplaceAll(quoted, `\?`, `[^/]`)
	quoted = strings.ReplaceAll(quoted, "<<<DOUBLESTAR>>>", ".*")
	re, err := regexp.Compile("^" + quoted + "$")
	if err != nil {
		ok, _ := path.Match(glob, value)
		return ok
	}
	return re.MatchString(value)
}

func matchPattern(pattern, value string) bool {
	if strings.Contains(pattern, "*") {
		ok, _ := filepath.Match(pattern, value)
		return ok
	}
	return strings.HasPrefix(value, pattern)
}

func allowedPrefix(p string, allow []string) bool {
	if len(allow) == 0 {
		return false
	}
	for _, prefix := range allow {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}
