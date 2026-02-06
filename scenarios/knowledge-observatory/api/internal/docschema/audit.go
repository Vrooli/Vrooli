package docschema

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// UndocumentedFile represents a code file with exported symbols but no DOC: references.
type UndocumentedFile struct {
	Path            string `json:"path"`
	ExportedSymbols int    `json:"exported_symbols"`
}

// BrokenRef represents a [CODE: ...] reference in documentation that points to a missing file.
type BrokenRef struct {
	DocPath string `json:"doc_path"`
	Line    int    `json:"line"`
	Target  string `json:"target"`
}

// DuplicateTitle represents a heading title that appears in multiple documentation files.
type DuplicateTitle struct {
	Title string   `json:"title"`
	Files []string `json:"files"`
}

// AuditResult contains the full audit of a scenario's documentation.
type AuditResult struct {
	ScenarioName        string             `json:"scenario_name"`
	HealthScore         float64            `json:"health_score"`
	TotalDocs           int                `json:"total_docs"`
	Infrastructure      *ValidationResult  `json:"infrastructure"`
	CodeWithoutDocRefs  []UndocumentedFile `json:"code_without_doc_refs"`
	BrokenCodeRefs      []BrokenRef        `json:"broken_code_refs"`
	OrphanedDocs        []string           `json:"orphaned_docs"`
	DuplicateTitles     []DuplicateTitle   `json:"duplicate_titles"`
	UndocumentedTargets []string           `json:"undocumented_targets"`
}

var (
	// Matches exported symbols in Go files.
	goExportedFunc = regexp.MustCompile(`^func\s+(?:\([^)]+\)\s+)?([A-Z]\w*)`)
	goExportedDecl = regexp.MustCompile(`^(?:type|var|const)\s+([A-Z]\w*)`)

	// Matches exported symbols in TypeScript/JavaScript files.
	tsExportedSymbol = regexp.MustCompile(`^export\s+(?:default\s+)?(?:function|class|const|let|var|interface|type|enum)\s+`)

	// Matches DOC: comments in code.
	docRefPattern = regexp.MustCompile(`(?i)//\s*DOC:`)

	// Matches heading lines in markdown.
	headingPattern = regexp.MustCompile(`^#{1,6}\s+(.+)$`)

	// Matches OT-P identifiers in PRD files.
	otPattern = regexp.MustCompile(`OT-P\d+-\d+`)

	// Code file extensions to scan.
	codeExtensions = map[string]bool{
		".go":  true,
		".ts":  true,
		".tsx": true,
		".js":  true,
		".jsx": true,
	}

	// Directories to skip during code scanning.
	skipDirs = map[string]bool{
		"node_modules": true,
		".git":         true,
		"dist":         true,
		"build":        true,
		"vendor":       true,
	}
)

// AuditScenarioDocumentation runs a comprehensive audit of a scenario's documentation.
func AuditScenarioDocumentation(scenarioPath string) (*AuditResult, error) {
	scenarioPath = strings.TrimSpace(scenarioPath)
	if scenarioPath == "" {
		return nil, ErrEmptyPath
	}

	result := &AuditResult{
		ScenarioName: filepath.Base(scenarioPath),
	}

	// Step 1: Infrastructure check (reuses existing validation).
	validation, err := ValidateScenarioDocumentation(scenarioPath)
	if err != nil {
		return nil, err
	}
	result.Infrastructure = validation
	result.HealthScore = validation.HealthScore

	// Count docs.
	result.TotalDocs = countDocFiles(scenarioPath)

	// Step 2: Find code without DOC: references.
	result.CodeWithoutDocRefs = findCodeWithoutDocRefs(scenarioPath)

	// Step 3: Find broken [CODE: ...] references in docs.
	result.BrokenCodeRefs = findBrokenCodeRefs(scenarioPath)

	// Step 4: Find orphaned docs not in manifest.
	result.OrphanedDocs = findOrphanedDocs(scenarioPath)

	// Step 5: Find duplicate titles.
	result.DuplicateTitles = findDuplicateTitles(scenarioPath)

	// Step 6: Find undocumented PRD targets.
	result.UndocumentedTargets = findUndocumentedTargets(scenarioPath)

	return result, nil
}

// findCodeWithoutDocRefs walks code directories and finds files with exported symbols
// that have no DOC: references.
func findCodeWithoutDocRefs(scenarioPath string) []UndocumentedFile {
	var undocumented []UndocumentedFile

	codeDirs := []string{
		filepath.Join(scenarioPath, "api"),
		filepath.Join(scenarioPath, "ui", "src"),
	}

	for _, dir := range codeDirs {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}

			ext := strings.ToLower(filepath.Ext(d.Name()))
			if !codeExtensions[ext] {
				return nil
			}
			// Skip test files.
			name := d.Name()
			if strings.HasSuffix(name, "_test.go") ||
				strings.Contains(name, ".test.") ||
				strings.Contains(name, ".spec.") {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			content := string(data)

			// Count exported symbols.
			exported := countExportedSymbols(content, ext)
			if exported == 0 {
				return nil
			}

			// Check for DOC: references.
			if docRefPattern.MatchString(content) {
				return nil
			}

			rel, err := filepath.Rel(scenarioPath, path)
			if err != nil {
				return nil
			}
			undocumented = append(undocumented, UndocumentedFile{
				Path:            filepath.ToSlash(rel),
				ExportedSymbols: exported,
			})
			return nil
		})
	}

	sort.Slice(undocumented, func(i, j int) bool {
		return undocumented[i].Path < undocumented[j].Path
	})
	return undocumented
}

func countExportedSymbols(content, ext string) int {
	lines := strings.Split(content, "\n")
	count := 0
	for _, line := range lines {
		switch ext {
		case ".go":
			if goExportedFunc.MatchString(line) || goExportedDecl.MatchString(line) {
				count++
			}
		case ".ts", ".tsx", ".js", ".jsx":
			if tsExportedSymbol.MatchString(strings.TrimSpace(line)) {
				count++
			}
		}
	}
	return count
}

// findBrokenCodeRefs finds [CODE: ...] references in docs that point to missing files.
func findBrokenCodeRefs(scenarioPath string) []BrokenRef {
	var broken []BrokenRef
	docsDir := filepath.Join(scenarioPath, "docs")

	info, err := os.Stat(docsDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	_ = filepath.WalkDir(docsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(d.Name())) != ".md" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		refs := ParseMarkdownReferences(string(data))
		docRel, _ := filepath.Rel(scenarioPath, path)

		for _, ref := range refs {
			if ref.Kind != ReferenceKindCode {
				continue
			}
			// Extract the file path from the target (strip #fragment and :line).
			target := ref.Target
			if idx := strings.IndexAny(target, "#:"); idx >= 0 {
				target = target[:idx]
			}
			target = strings.TrimSpace(target)
			if target == "" {
				continue
			}
			// Check if the file exists relative to scenario root.
			fullPath := filepath.Join(scenarioPath, target)
			if _, err := os.Stat(fullPath); err != nil {
				broken = append(broken, BrokenRef{
					DocPath: filepath.ToSlash(docRel),
					Line:    ref.Line,
					Target:  ref.Target,
				})
			}
		}
		return nil
	})

	sort.Slice(broken, func(i, j int) bool {
		if broken[i].DocPath != broken[j].DocPath {
			return broken[i].DocPath < broken[j].DocPath
		}
		return broken[i].Line < broken[j].Line
	})
	return broken
}

// findOrphanedDocs finds docs not registered in manifest.json.
func findOrphanedDocs(scenarioPath string) []string {
	manifestPath := filepath.Join(scenarioPath, "docs", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil // No manifest means we can't check.
	}

	// Extract all paths registered in the manifest.
	registeredPaths := extractManifestPaths(data)
	if registeredPaths == nil {
		return nil
	}

	var orphaned []string
	docsDir := filepath.Join(scenarioPath, "docs")

	_ = filepath.WalkDir(docsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(d.Name())) != ".md" {
			return nil
		}

		rel, err := filepath.Rel(docsDir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if !registeredPaths[rel] {
			orphaned = append(orphaned, "docs/"+rel)
		}
		return nil
	})

	sort.Strings(orphaned)
	return orphaned
}

// extractManifestPaths parses manifest.json and returns all registered doc paths.
func extractManifestPaths(data []byte) map[string]bool {
	var manifest struct {
		Sections []struct {
			Documents []struct {
				Path string `json:"path"`
			} `json:"documents"`
		} `json:"sections"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil
	}
	paths := make(map[string]bool)
	for _, section := range manifest.Sections {
		for _, doc := range section.Documents {
			if p := strings.TrimSpace(doc.Path); p != "" {
				paths[p] = true
			}
		}
	}
	return paths
}

// findDuplicateTitles finds heading titles that appear in multiple doc files.
func findDuplicateTitles(scenarioPath string) []DuplicateTitle {
	docsDir := filepath.Join(scenarioPath, "docs")
	info, err := os.Stat(docsDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	// Map title -> list of files.
	titleFiles := make(map[string][]string)

	_ = filepath.WalkDir(docsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(d.Name())) != ".md" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		rel, _ := filepath.Rel(scenarioPath, path)
		relSlash := filepath.ToSlash(rel)

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			matches := headingPattern.FindStringSubmatch(line)
			if matches == nil {
				continue
			}
			title := strings.TrimSpace(matches[1])
			if title == "" || title == "Last Updated" {
				continue
			}
			titleFiles[title] = append(titleFiles[title], relSlash)
		}
		return nil
	})

	var duplicates []DuplicateTitle
	for title, files := range titleFiles {
		// Deduplicate file entries.
		unique := uniqueStrings(files)
		if len(unique) < 2 {
			continue
		}
		sort.Strings(unique)
		duplicates = append(duplicates, DuplicateTitle{
			Title: title,
			Files: unique,
		})
	}
	sort.Slice(duplicates, func(i, j int) bool {
		return duplicates[i].Title < duplicates[j].Title
	})
	return duplicates
}

func uniqueStrings(s []string) []string {
	seen := make(map[string]bool, len(s))
	out := make([]string, 0, len(s))
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// findUndocumentedTargets parses PRD.md for OT-P* identifiers and checks if they
// appear anywhere in the docs/ directory.
func findUndocumentedTargets(scenarioPath string) []string {
	prdPath := filepath.Join(scenarioPath, "PRD.md")
	data, err := os.ReadFile(prdPath)
	if err != nil {
		return nil // No PRD means nothing to check.
	}

	// Find all OT-P identifiers in PRD.
	matches := otPattern.FindAllString(string(data), -1)
	if len(matches) == 0 {
		return nil
	}

	// Deduplicate.
	seen := make(map[string]bool)
	var targets []string
	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			targets = append(targets, m)
		}
	}
	sort.Strings(targets)

	// Read all docs content into a single string for fast lookup.
	var docsContent strings.Builder
	docsDir := filepath.Join(scenarioPath, "docs")
	if info, err := os.Stat(docsDir); err == nil && info.IsDir() {
		_ = filepath.WalkDir(docsDir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if ext != ".md" && ext != ".json" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			docsContent.Write(data)
			docsContent.WriteByte('\n')
			return nil
		})
	}

	docsText := docsContent.String()
	var undocumented []string
	for _, target := range targets {
		if !strings.Contains(docsText, target) {
			undocumented = append(undocumented, target)
		}
	}
	return undocumented
}

// countDocFiles counts documentation files in a scenario.
func countDocFiles(scenarioPath string) int {
	count := 0
	for _, rootFile := range []string{"README.md", "PRD.md"} {
		if fileExists(filepath.Join(scenarioPath, rootFile)) {
			count++
		}
	}
	docsDir := filepath.Join(scenarioPath, "docs")
	if info, err := os.Stat(docsDir); err == nil && info.IsDir() {
		_ = filepath.WalkDir(docsDir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if isDocFile(d.Name()) {
				count++
			}
			return nil
		})
	}
	return count
}

// ErrEmptyPath is returned when the scenario path is empty.
var ErrEmptyPath = errEmptyPath{}

type errEmptyPath struct{}

func (errEmptyPath) Error() string { return "scenario path is required" }
