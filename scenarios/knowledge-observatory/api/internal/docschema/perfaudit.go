package docschema

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// perfAuditComponentTableSeparator matches the markdown separator row of the
// per-component aggregation table: at least 5 pipe-delimited dash cells. We
// allow trailing extra columns (>=5) so authors can extend the table without
// invalidating the doc.
var perfAuditComponentTableSeparator = regexp.MustCompile(`(?m)^\s*\|\s*-{3,}\s*(?:\|\s*-{3,}\s*){4,}\|?\s*$`)

// perfAuditComponentTableHeading matches a heading that contains "Per-component"
// (case-insensitive).
var perfAuditComponentTableHeading = regexp.MustCompile(`(?im)^\s*#{1,6}\s+.*per-?component.*$`)

var perfAuditFilenamePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-[a-z0-9][a-z0-9.-]*\.md$`)

// findPerfAuditDocs walks <scenarioPath>/docs/perf and returns the relative
// paths of files matching the perf audit filename pattern. Returns nil if the
// directory doesn't exist.
func findPerfAuditDocs(scenarioPath string) []string {
	dir := filepath.Join(scenarioPath, "docs", "perf")
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil
	}
	var found []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path == dir {
				return nil
			}
			// docs/perf/ is intentionally flat — don't descend into subdirs.
			return filepath.SkipDir
		}
		if !perfAuditFilenamePattern.MatchString(d.Name()) {
			return nil
		}
		rel, err := filepath.Rel(scenarioPath, path)
		if err != nil {
			return nil
		}
		found = append(found, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(found)
	return found
}

// auditPerfDocs validates every recognized perf-audit doc against the
// frontmatter schema and the per-component-table body shape. Skips files in
// docs/perf/ whose names don't match the canonical date-slug pattern (those
// are surfaced as ExtraDocs by the infrastructure pass — except the conventional
// README.md, which is allowed and not validated here).
func auditPerfDocs(scenarioPath string) []FrontmatterIssue {
	docs := findPerfAuditDocs(scenarioPath)
	if len(docs) == 0 {
		return nil
	}
	var issues []FrontmatterIssue
	for _, rel := range docs {
		full := filepath.Join(scenarioPath, rel)
		data, err := os.ReadFile(full)
		if err != nil {
			issues = append(issues, FrontmatterIssue{
				DocPath:  rel,
				Code:     "perf-audit:read-error",
				Message:  "could not read file: " + err.Error(),
				Severity: "error",
			})
			continue
		}
		content := string(data)

		fmIssues := ValidateFrontmatter(content, PerfAuditFrontmatterSchema)
		for i := range fmIssues {
			fmIssues[i].DocPath = rel
		}
		issues = append(issues, fmIssues...)

		// Cross-check that the filename's leading date matches the `date`
		// frontmatter scalar when both are present.
		if fm, _ := extractFrontmatter(content); fm.present {
			if got, ok := fm.scalars["date"]; ok && got != "" {
				base := filepath.Base(rel)
				if len(base) >= 10 && base[:10] != got {
					issues = append(issues, FrontmatterIssue{
						DocPath:  rel,
						Code:     "perf-audit:date-mismatch",
						Field:    "date",
						Message:  "filename's leading date does not match frontmatter `date`",
						Severity: "warning",
					})
				}
			}
		}

		// Body-shape check: a per-component aggregation table must exist.
		body := stripFrontmatterBody(content)
		if !perfAuditComponentTableHeading.MatchString(body) {
			issues = append(issues, FrontmatterIssue{
				DocPath:  rel,
				Code:     "perf-audit:missing-component-table-heading",
				Message:  "no `## Per-component` heading found",
				Severity: "warning",
			})
		}
		if !perfAuditComponentTableSeparator.MatchString(body) {
			issues = append(issues, FrontmatterIssue{
				DocPath:  rel,
				Code:     "perf-audit:missing-component-table",
				Message:  "no per-component aggregation table (5+ column markdown table) found",
				Severity: "warning",
			})
		}
	}

	// Stable order: group by doc path, then by code.
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].DocPath != issues[j].DocPath {
			return issues[i].DocPath < issues[j].DocPath
		}
		return issues[i].Code < issues[j].Code
	})
	return issues
}

// stripFrontmatterBody returns the markdown body, with the leading frontmatter
// block removed if present. Light wrapper over extractFrontmatter that
// discards the parsed frontmatter and keeps only the body.
func stripFrontmatterBody(content string) string {
	_, body := extractFrontmatter(content)
	return body
}

// Sanity guard — ensures we don't accidentally treat the README.md inside
// docs/perf/ as a perf-audit doc.
func init() {
	if perfAuditFilenamePattern.MatchString("README.md") {
		// This would silently break: README.md should NOT match the pattern.
		// Fail fast in tests/dev if the regex is loosened in error.
		panic("perfAuditFilenamePattern incorrectly matches README.md")
	}
	_ = strings.TrimSpace // keep import alive when init becomes empty across edits
}
