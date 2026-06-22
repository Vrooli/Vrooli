package validation

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SQLFile is a discovered schema/SQL file with its repo-relative path and the
// domain that owns it (inferred from api/internal/<domain>/ layout; "" for the
// system home or files not under a domain directory).
type SQLFile struct {
	// AbsPath is the absolute path on disk.
	AbsPath string
	// RelPath is the path relative to the scenario directory (forward-slashed),
	// suitable for use as a finding Location.
	RelPath string
	// Domain is the owning domain inferred from api/internal/<domain>/, or "".
	Domain string
	// IsSystem reports whether this is the api/internal/database/system.sql home.
	IsSystem bool
}

// GoFile is a discovered Go source file (non-test, non-exempt).
type GoFile struct {
	AbsPath string
	RelPath string
	Domain  string
}

// exemptDirSegments are path segments that exempt a file from storage analysis:
// test fixtures, versioned migrations, one-shot scripts, generated code, and the
// api-core package itself (analyzers never flag the seams they enforce). A file
// is exempt if any of these appears as a path segment.
var exemptDirSegments = map[string]struct{}{
	"migrations":   {},
	"scripts":      {},
	"testdata":     {},
	"node_modules": {},
	"vendor":       {},
	"gen":          {},
	"dist":         {},
}

// isExemptPath reports whether a repo-relative path is exempt from storage
// analysis (test files, migrations/, scripts/, generated/vendored code).
func isExemptPath(rel string) bool {
	rel = filepath.ToSlash(rel)
	if strings.HasSuffix(rel, "_test.go") {
		return true
	}
	for _, seg := range strings.Split(rel, "/") {
		if _, ok := exemptDirSegments[seg]; ok {
			return true
		}
	}
	return false
}

// CollectSQLFiles walks the scenario's api/ surface and returns every embedded
// schema .sql file (per-domain schema.sql + the system home), excluding exempt
// paths (migrations/, scripts/, testdata/). Returns nil when there is no api/
// directory. Deterministic order.
func CollectSQLFiles(ac AnalyzerContext) []SQLFile {
	if ac.APIDir == "" {
		return nil
	}
	var out []SQLFile
	_ = filepath.WalkDir(ac.APIDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".sql") {
			return nil
		}
		rel := relTo(ac.ScenarioDir, path)
		if isExemptPath(rel) {
			return nil
		}
		out = append(out, SQLFile{
			AbsPath:  path,
			RelPath:  rel,
			Domain:   domainOf(ac.APIDir, path),
			IsSystem: strings.HasSuffix(filepath.ToSlash(path), "/internal/database/system.sql"),
		})
		return nil
	})
	sortByRel(out, func(f SQLFile) string { return f.RelPath })
	return out
}

// CollectGoFiles walks the scenario's api/ surface and returns every non-test,
// non-exempt Go source file with its inferred domain. Returns nil when there is
// no api/ directory. Deterministic order.
func CollectGoFiles(ac AnalyzerContext) []GoFile {
	if ac.APIDir == "" {
		return nil
	}
	var out []GoFile
	_ = filepath.WalkDir(ac.APIDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel := relTo(ac.ScenarioDir, path)
		if isExemptPath(rel) {
			return nil
		}
		out = append(out, GoFile{AbsPath: path, RelPath: rel, Domain: domainOf(ac.APIDir, path)})
		return nil
	})
	sortByRel(out, func(f GoFile) string { return f.RelPath })
	return out
}

// ReadFile reads a discovered file, returning "" on error so analyzers can stay
// branch-light. Errors here are non-fatal (a file that vanished mid-scan).
func ReadFile(absPath string) string {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return ""
	}
	return string(data)
}

// domainOf infers the owning domain from an api/internal/<domain>/... path.
// Returns "" when the file is not under api/internal/<domain> or the segment is
// cross-cutting infrastructure (e.g. database, server).
func domainOf(apiDir, path string) string {
	rel, err := filepath.Rel(filepath.Join(apiDir, "internal"), path)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "..") {
		return ""
	}
	seg := strings.SplitN(rel, "/", 2)
	if len(seg) == 0 {
		return ""
	}
	domain := seg[0]
	if _, infra := infraDirs[domain]; infra {
		return ""
	}
	return domain
}

func relTo(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

// sortByRel sorts a slice in place by a string key. Generic over the element
// type so both SQLFile and GoFile slices use it.
func sortByRel[T any](s []T, key func(T) string) {
	// insertion sort keeps the dependency surface tiny and the slices are small
	// (a handful of schema files per scenario).
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && key(s[j]) < key(s[j-1]); j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
