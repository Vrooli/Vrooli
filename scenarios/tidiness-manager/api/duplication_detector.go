package main

import (
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/factgraph"
	debt "tidiness-manager/internal/discovery"
)

// DuplicateResult adapts the shared body-identity detector to scan responses.
type DuplicateResult struct {
	TotalDuplicates int              `json:"total_duplicates"`
	DuplicateBlocks []DuplicateBlock `json:"duplicate_blocks"`
	TotalLines      int              `json:"total_lines"`
	Skipped         bool             `json:"skipped"`
	SkipReason      string           `json:"skip_reason,omitempty"`
	Tool            string           `json:"tool,omitempty"`
}

// DuplicateMinimumLines is the physical-line floor used by both scan responses
// and ranked refactor recommendations.
const DuplicateMinimumLines = 5

type DuplicateBlock struct {
	Files []DuplicateLocation `json:"files"`
	Lines int                 `json:"lines"`
}

type DuplicateLocation struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// DuplicationDetector projects Code Facts evidence into the legacy scan shape;
// source parsing and duplicate grouping have one owner in Code Facts and the
// discovery registry.
type DuplicationDetector struct {
	scenarioPath string
}

func NewDuplicationDetector(scenarioPath string) *DuplicationDetector {
	return &DuplicationDetector{scenarioPath: scenarioPath}
}

// DetectDuplicationForLanguagesFromIndex projects all language groups from one
// graph traversal, allowing multi-language scans to reuse shared symbols.
func (dd *DuplicationDetector) DetectDuplicationForLanguagesFromIndex(filesByLanguage map[string][]string, index *factgraph.Index) map[string]*DuplicateResult {
	groupsByLanguage, warnings, evidence := debt.DuplicateFunctionBodiesByLanguageWithEvidence(index, DuplicateMinimumLines)
	results := make(map[string]*DuplicateResult, len(filesByLanguage))
	for language, files := range filesByLanguage {
		if len(files) == 0 {
			results[language] = &DuplicateResult{Skipped: true, SkipReason: "no files to analyze"}
			continue
		}
		if _, err := resolveScenarioFiles(dd.scenarioPath, files); err != nil {
			results[language] = &DuplicateResult{Skipped: true, SkipReason: err.Error()}
			continue
		}

		allowed := make(map[string]bool, len(files))
		for _, file := range files {
			allowed[filepath.ToSlash(filepath.Clean(file))] = true
		}
		hasEvidence := false
		for path := range allowed {
			if evidence[path][language] {
				hasEvidence = true
				break
			}
		}
		if !hasEvidence {
			results[language] = &DuplicateResult{Skipped: true, SkipReason: "Code Facts returned no " + language + " source evidence for the requested files", Tool: "code-facts"}
			continue
		}
		groups := groupsByLanguage[language]
		blocks := make([]DuplicateBlock, 0, len(groups))
		totalLines := 0
		for _, group := range groups {
			block := DuplicateBlock{Lines: group.Lines}
			for _, location := range group.Locations {
				if allowed[filepath.ToSlash(filepath.Clean(location.Path))] {
					block.Files = append(block.Files, DuplicateLocation{Path: location.Path, StartLine: location.StartLine, EndLine: location.EndLine})
				}
			}
			if len(block.Files) > 1 {
				blocks = append(blocks, block)
				totalLines += block.Lines
			}
		}

		results[language] = &DuplicateResult{
			TotalDuplicates: len(blocks),
			DuplicateBlocks: blocks,
			TotalLines:      totalLines,
			Skipped:         len(warnings) > 0,
			SkipReason:      strings.Join(warnings, "; "),
			Tool:            "code-facts",
		}
	}
	return results
}
