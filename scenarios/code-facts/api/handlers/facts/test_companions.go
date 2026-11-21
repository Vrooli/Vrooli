package facts

import (
	"path/filepath"
	"strings"
)

// splitTestSourcePaths classifies the active FILE_METRICS catalog paths once,
// so every consumer uses the same test-file convention.
func splitTestSourcePaths(paths []string) (map[string]bool, map[string]bool) {
	tests, sources := make(map[string]bool), make(map[string]bool)
	for _, path := range paths {
		if isTestSourcePath(path) {
			tests[path] = true
		} else {
			sources[path] = true
		}
	}
	return tests, sources
}

func isTestSourcePath(path string) bool {
	base := filepath.Base(path)
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return strings.HasSuffix(base, "_test.go")
	case ".ts", ".tsx", ".js", ".jsx":
		return strings.Contains(filepath.ToSlash(path), "__tests__") ||
			strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".test.tsx") ||
			strings.HasSuffix(base, ".spec.ts") || strings.HasSuffix(base, ".spec.tsx") ||
			strings.HasSuffix(base, ".test.js") || strings.HasSuffix(base, ".test.jsx") ||
			strings.HasSuffix(base, ".spec.js") || strings.HasSuffix(base, ".spec.jsx")
	case ".py":
		return strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py")
	case ".rs":
		return strings.Contains(filepath.ToSlash(path), "/tests/")
	default:
		return false
	}
}

// hasTestCompanion preserves Tidiness's filename heuristic, including Go's
// longest-source-stem rule for descriptive test filenames. It does not claim
// behavioral coverage.
func hasTestCompanion(sourcePath string, testFiles, sourceFiles map[string]bool) bool {
	dir, base := filepath.Dir(sourcePath), filepath.Base(sourcePath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	switch strings.ToLower(ext) {
	case ".go":
		if testFiles[filepath.Join(dir, stem+"_test.go")] {
			return true
		}
		for candidate := range testFiles {
			if filepath.Dir(candidate) != dir {
				continue
			}
			candidateBase := filepath.Base(candidate)
			if !strings.HasPrefix(candidateBase, stem+"_") || !strings.HasSuffix(candidateBase, "_test.go") {
				continue
			}
			candidateStem := strings.TrimSuffix(candidateBase, "_test.go")
			mostSpecific := ""
			if sourceFiles[filepath.Join(dir, candidateStem+".go")] {
				mostSpecific = filepath.Join(dir, candidateStem+".go")
			}
			for i, r := range candidateStem {
				if r != '_' {
					continue
				}
				possible := filepath.Join(dir, candidateStem[:i]+".go")
				if sourceFiles[possible] && len(candidateStem[:i]) > len(strings.TrimSuffix(filepath.Base(mostSpecific), ".go")) {
					mostSpecific = possible
				}
			}
			if mostSpecific == "" || mostSpecific == sourcePath {
				return true
			}
		}
	case ".ts", ".tsx", ".js", ".jsx":
		for _, candidate := range []string{
			filepath.Join(dir, stem+".test"+ext), filepath.Join(dir, stem+".spec"+ext),
			filepath.Join(dir, "__tests__", stem+".test"+ext), filepath.Join(dir, "__tests__", stem+".spec"+ext),
			filepath.Join(dir, "__tests__", base),
		} {
			if testFiles[candidate] {
				return true
			}
		}
	case ".py":
		for _, candidate := range []string{filepath.Join(dir, "test_"+base), filepath.Join(dir, stem+"_test.py")} {
			if testFiles[candidate] {
				return true
			}
		}
	case ".rs":
		testsDir := filepath.Join(dir, "tests")
		for candidate := range testFiles {
			if strings.HasPrefix(candidate, testsDir+string(filepath.Separator)) {
				return true
			}
		}
	}
	return false
}
