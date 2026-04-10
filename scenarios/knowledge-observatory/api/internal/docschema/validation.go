package docschema

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// MisplacedDoc represents a documentation file in the wrong location.
type MisplacedDoc struct {
	ActualPath   string  `json:"actual_path"`
	ExpectedPath string  `json:"expected_path"`
	DocType      DocType `json:"doc_type"`
	Severity     string  `json:"severity"` // "error", "warning", "info"
}

// ValidationResult contains the results of validating a scenario's docs.
type ValidationResult struct {
	ScenarioName  string         `json:"scenario_name"`
	MisplacedDocs []MisplacedDoc `json:"misplaced_docs"`
	MissingDocs   []DocType      `json:"missing_docs"`
	ExtraDocs     []string       `json:"extra_docs"` // docs not matching any known pattern
	TemporaryDocs []string       `json:"temporary_docs"`
	HealthScore   float64        `json:"health_score"` // 0-1 score based on compliance
}

// ValidateScenarioDocumentation checks if docs are in the right places.
func ValidateScenarioDocumentation(scenarioPath string) (*ValidationResult, error) {
	scenarioPath = strings.TrimSpace(scenarioPath)
	if scenarioPath == "" {
		return nil, errors.New("scenarioPath is required")
	}
	info, err := os.Stat(scenarioPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("scenarioPath must be a directory")
	}

	structure := StandardScenarioStructure
	expected := expectedPaths(structure)
	present := map[DocType]bool{}

	for dt, rel := range expected {
		if rel == "" {
			continue
		}
		if fileExists(filepath.Join(scenarioPath, rel)) {
			present[dt] = true
		}
	}

	knownNames := knownDocTypeNames()
	found := discoverDocTypePaths(scenarioPath, knownNames)
	result := &ValidationResult{ScenarioName: filepath.Base(scenarioPath)}

	for dt, relPaths := range found {
		expectedRel := filepath.ToSlash(expected[dt])
		for _, rel := range relPaths {
			if expectedRel != "" && rel != expectedRel {
				result.MisplacedDocs = append(result.MisplacedDocs, MisplacedDoc{
					ActualPath:   rel,
					ExpectedPath: expectedRel,
					DocType:      dt,
					Severity:     docSeverity(dt, structure),
				})
			}
			if expectedRel != "" && rel == expectedRel {
				present[dt] = true
			}
		}
	}

	for _, dt := range allDocTypes(structure) {
		if expected[dt] == "" {
			continue
		}
		if !present[dt] {
			result.MissingDocs = append(result.MissingDocs, dt)
		}
	}

	result.ExtraDocs = findExtraDocs(scenarioPath, knownNames)
	result.TemporaryDocs = findTemporaryDocs(scenarioPath)
	result.HealthScore = computeHealthScore(structure, result.MissingDocs, result.MisplacedDocs, result.TemporaryDocs)

	sort.Slice(result.MisplacedDocs, func(i, j int) bool {
		if result.MisplacedDocs[i].DocType != result.MisplacedDocs[j].DocType {
			return result.MisplacedDocs[i].DocType < result.MisplacedDocs[j].DocType
		}
		return result.MisplacedDocs[i].ActualPath < result.MisplacedDocs[j].ActualPath
	})
	sort.Slice(result.MissingDocs, func(i, j int) bool {
		return result.MissingDocs[i] < result.MissingDocs[j]
	})
	sort.Strings(result.ExtraDocs)
	sort.Strings(result.TemporaryDocs)

	return result, nil
}

func expectedPaths(structure ScenarioDocStructure) map[DocType]string {
	paths := make(map[DocType]string)
	for _, dt := range allDocTypes(structure) {
		paths[dt] = dt.ExpectedPath()
	}
	for key, value := range structure.CustomPaths {
		dt := DocType(key)
		if value != "" {
			paths[dt] = filepath.ToSlash(value)
		}
	}
	return paths
}

func allDocTypes(structure ScenarioDocStructure) []DocType {
	seen := map[DocType]bool{}
	var all []DocType
	for _, dt := range structure.Required {
		if !seen[dt] {
			seen[dt] = true
			all = append(all, dt)
		}
	}
	for _, dt := range structure.Optional {
		if !seen[dt] {
			seen[dt] = true
			all = append(all, dt)
		}
	}
	return all
}

func docSeverity(dt DocType, structure ScenarioDocStructure) string {
	if isRequiredDocType(dt, structure) {
		return "error"
	}
	return "warning"
}

func isRequiredDocType(dt DocType, structure ScenarioDocStructure) bool {
	for _, required := range structure.Required {
		if required == dt {
			return true
		}
	}
	return false
}

func knownDocTypeNames() map[string]DocType {
	return map[string]DocType{
		"PROBLEMS.md":                      DocTypeProblems,
		"PROGRESS.md":                      DocTypeProgress,
		"SEAMS.md":                         DocTypeSeams,
		"INVARIANTS.md":                    DocTypeInvariants,
		"ASSUMPTIONS.md":                   DocTypeAssumptions,
		"ERROR-SEMANTICS.md":               DocTypeErrorSemantics,
		"SECURITY-POSTURE.md":              DocTypeSecurityPosture,
		"TEMPORAL-FLOWS.md":                DocTypeTemporalFlows,
		"COHERENCE-NOTES.md":               DocTypeCoherenceNotes,
		"EXPERIENCE-AUDIT.md":              DocTypeExperienceAudit,
		"QUICKSTART.md":                    DocTypeQuickstart,
		"ARCHITECTURE.md":                  DocTypeArchitecture,
		"GLOSSARY.md":                      DocTypeGlossary,
		"PRD.md":                           DocTypePRD,
		"README.md":                        DocTypeReadme,
		"manifest.json":                    DocTypeManifest,
		"COHERENCE_NOTES.md":               DocTypeCoherenceNotes,
		"EXPERIENCE_AUDIT.md":              DocTypeExperienceAudit,
		"EXPERIENCE_ARCHITECTURE_AUDIT.md": DocTypeExperienceAudit,
	}
}

func discoverDocTypePaths(scenarioPath string, knownNames map[string]DocType) map[DocType][]string {
	found := make(map[DocType][]string)

	rootEntries, err := os.ReadDir(scenarioPath)
	if err == nil {
		for _, entry := range rootEntries {
			if entry.IsDir() {
				continue
			}
			if dt, ok := knownNames[entry.Name()]; ok {
				found[dt] = appendUnique(found[dt], entry.Name())
			}
		}
	}

	docsDir := filepath.Join(scenarioPath, "docs")
	if info, err := os.Stat(docsDir); err == nil && info.IsDir() {
		_ = filepath.WalkDir(docsDir, func(filePath string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if dt, ok := knownNames[d.Name()]; ok {
				rel, err := filepath.Rel(scenarioPath, filePath)
				if err == nil {
					found[dt] = appendUnique(found[dt], filepath.ToSlash(rel))
				}
			}
			return nil
		})
	}

	return found
}

func findExtraDocs(scenarioPath string, knownNames map[string]DocType) []string {
	var extras []string

	rootEntries, err := os.ReadDir(scenarioPath)
	if err == nil {
		for _, entry := range rootEntries {
			if entry.IsDir() {
				continue
			}
			if !isDocFile(entry.Name()) {
				continue
			}
			rel := entry.Name()
			if !isRecognizedDocPath(rel, knownNames) {
				extras = append(extras, rel)
			}
		}
	}

	docsDir := filepath.Join(scenarioPath, "docs")
	if info, err := os.Stat(docsDir); err == nil && info.IsDir() {
		_ = filepath.WalkDir(docsDir, func(filePath string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if !isDocFile(d.Name()) {
				return nil
			}
			rel, err := filepath.Rel(scenarioPath, filePath)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if !isRecognizedDocPath(rel, knownNames) {
				extras = append(extras, rel)
			}
			return nil
		})
	}

	return extras
}

func isRecognizedDocPath(rel string, knownNames map[string]DocType) bool {
	rel = filepath.ToSlash(rel)
	if rel == "" {
		return false
	}
	if _, ok := knownNames[path.Base(rel)]; ok {
		return true
	}
	if rel == "PRD.md" {
		return true
	}
	if rel == "docs/QUICKSTART.md" {
		return true
	}
	if rel == "docs/manifest.json" {
		return true
	}
	for _, prefix := range []string{
		"docs/concepts/",
		"docs/guides/",
		"docs/reference/",
		"docs/internal/",
		"docs/plans/",
	} {
		if strings.HasPrefix(rel, prefix) {
			return true
		}
	}
	return false
}

func isDocFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".md" || ext == ".json"
}

var temporaryDocBaseNames = map[string]struct{}{
	"implementation_plan.md": {},
	"implementation-plan.md": {},
	"temp.md":                {},
	"tmp.md":                 {},
	"scratch.md":             {},
	"wip.md":                 {},
	"todo.md":                {},
}

func findTemporaryDocs(scenarioPath string) []string {
	temporary := make([]string, 0, 4)
	skipDirs := map[string]struct{}{
		".git":         {},
		".vrooli":      {},
		"node_modules": {},
		"vendor":       {},
		"dist":         {},
		"build":        {},
		"coverage":     {},
		".next":        {},
	}

	_ = filepath.WalkDir(scenarioPath, func(filePath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := strings.ToLower(strings.TrimSpace(d.Name()))
			if strings.HasPrefix(name, ".") && name != ".well-known" {
				return filepath.SkipDir
			}
			if _, ok := skipDirs[name]; ok {
				return filepath.SkipDir
			}
			return nil
		}
		if !isDocFile(d.Name()) {
			return nil
		}
		if !isTemporaryDocName(d.Name()) {
			return nil
		}
		rel, err := filepath.Rel(scenarioPath, filePath)
		if err != nil {
			return nil
		}
		temporary = append(temporary, filepath.ToSlash(rel))
		return nil
	})

	return temporary
}

func isTemporaryDocName(name string) bool {
	candidate := strings.ToLower(strings.TrimSpace(name))
	if candidate == "" {
		return false
	}
	if _, ok := temporaryDocBaseNames[candidate]; ok {
		return true
	}
	trimmed := strings.TrimSuffix(candidate, filepath.Ext(candidate))
	for _, marker := range []string{
		"implementation_plan",
		"implementation-plan",
		"temporary",
		"scratch",
		"-tmp",
		"_tmp",
		"-temp",
		"_temp",
		"wip",
	} {
		if strings.Contains(trimmed, marker) {
			return true
		}
	}
	return false
}

func appendUnique(existing []string, value string) []string {
	for _, item := range existing {
		if item == value {
			return existing
		}
	}
	return append(existing, value)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func computeHealthScore(structure ScenarioDocStructure, missing []DocType, misplaced []MisplacedDoc, temporary []string) float64 {
	requiredCount := len(structure.Required)
	missingRequired := 0
	for _, dt := range missing {
		if isRequiredDocType(dt, structure) {
			missingRequired++
		}
	}

	score := 1.0
	if requiredCount > 0 {
		score = float64(requiredCount-missingRequired) / float64(requiredCount)
	}
	score -= 0.05 * float64(len(misplaced))
	score -= 0.01 * float64(len(temporary))
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score
}
